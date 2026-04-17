package ctrlfwk

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func setReadyCondition[ControllerResourceType client.Object](
	obj ControllerResourceType,
	status metav1.ConditionStatus,
	reason string,
	message string,
) (bool, error) {
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	statusField := objValue.FieldByName("Status")
	if !statusField.IsValid() {
		return false, fmt.Errorf("status field not found on controller resource")
	}

	conditionsField := statusField.FieldByName("Conditions")
	if !conditionsField.IsValid() || conditionsField.Kind() != reflect.Slice {
		return false, fmt.Errorf("conditions field not found or is not a slice on status")
	}

	conditions := conditionsField.Interface().([]metav1.Condition)
	readyCondition := metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: obj.GetGeneration(),
	}

	changed := meta.SetStatusCondition(&conditions, readyCondition)
	if !changed {
		return false, nil
	}

	conditionsField.Set(reflect.ValueOf(conditions))
	return true, nil
}

// SetReadyCondition is a function type that sets the Ready condition on a controller resource.
// It uses reflection and assumes that the controller resource has a standard status field with conditions.
// Your api MUST have a field like so:
//
//	type MyCustomResourceStatus struct {
//	    Conditions []metav1.Condition `json:"conditions,omitempty"`
//	    ...
//	}
//
// If your status field or conditions field is named differently, this function will not work correctly.
func SetReadyCondition[ControllerResourceType client.Object](_ Reconciler[ControllerResourceType]) func(obj ControllerResourceType) (bool, error) {
	return func(obj ControllerResourceType) (bool, error) {
		return setReadyCondition(obj, metav1.ConditionTrue, "Reconciled", "The resource is ready")
	}
}

// SetReadyConditionFromResult derives the Ready condition from the last executed step result.
//
// Successful reconciliations set Ready to True. Errors, requeues, and early returns set Ready
// to False with a message describing the last executed step.
func SetReadyConditionFromResult[ControllerResourceType client.Object](_ Reconciler[ControllerResourceType]) func(obj ControllerResourceType, lastStepName string, lastStepResult StepResult) (bool, error) {
	return func(obj ControllerResourceType, lastStepName string, lastStepResult StepResult) (bool, error) {
		status := metav1.ConditionTrue
		reason := "Reconciled"
		message := "The resource is ready"

		switch {
		case lastStepResult.Error() != nil:
			status = metav1.ConditionFalse
			reason = "ReconciliationFailed"
			if lastStepName == "" {
				message = fmt.Sprintf("Reconciliation failed: %v", lastStepResult.Error())
			} else {
				message = fmt.Sprintf("Step %q failed: %v", lastStepName, lastStepResult.Error())
			}
		case lastStepResult.RequeueAfter() > 0:
			status = metav1.ConditionFalse
			reason = "Reconciling"
			if lastStepName == "" {
				message = fmt.Sprintf("Reconciliation will continue after %s", lastStepResult.RequeueAfter())
			} else {
				message = fmt.Sprintf("Step %q requested a requeue after %s", lastStepName, lastStepResult.RequeueAfter())
			}
		case lastStepResult.IsEarlyReturn():
			status = metav1.ConditionFalse
			reason = "Reconciling"
			if lastStepName == "" {
				message = "Reconciliation is still in progress"
			} else {
				message = fmt.Sprintf("Step %q requested an early return", lastStepName)
			}
		}

		return setReadyCondition(obj, status, reason, message)
	}
}

// PatchCustomResourceStatus patches the status subresource of the custom resource stored in the context.
// This function assumes that the context contains a ReconcilerContextData with the CustomResource field populated.
// The step "FindControllerResource" does exactly that, populating the context.
//
// It also sets the updated custom resource back into the context after patching.
func PatchCustomResourceStatus[CustomResourceType client.Object](ctx Context[CustomResourceType], reconciler Reconciler[CustomResourceType]) error {
	// Get the custom resource from the context
	cleanObject := ctx.GetCleanCustomResource()
	modifiableObject := ctx.GetCustomResource()

	// Patch the status subresource
	err := reconciler.Status().Patch(ctx, modifiableObject, client.MergeFrom(cleanObject))
	if err != nil {
		return err
	}

	ctx.SetCustomResource(modifiableObject)

	return nil
}
