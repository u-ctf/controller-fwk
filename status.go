package ctrlfwk

import (
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GenericSetReadyCondition is a function type that sets the Ready condition on a controller resource.
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
		// Use reflection to set the Ready condition
		objValue := reflect.ValueOf(obj)
		if objValue.Kind() == reflect.Ptr {
			objValue = objValue.Elem()
		}

		statusField := objValue.FieldByName("Status")
		if !statusField.IsValid() {
			return false, nil // No status field found
		}

		conditionsField := statusField.FieldByName("Conditions")
		if !conditionsField.IsValid() || conditionsField.Kind() != reflect.Slice {
			return false, nil // No conditions field found
		}

		conditions := conditionsField.Interface().([]metav1.Condition)

		readyCondition := metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "Reconciled",
			Message:            "The resource is ready",
			LastTransitionTime: metav1.Now(),
			ObservedGeneration: obj.GetGeneration(),
		}

		changed := meta.SetStatusCondition(&conditions, readyCondition)
		if !changed {
			return false, nil
		}

		conditionsField.Set(reflect.ValueOf(conditions))

		return changed, nil
	}
}

// PatchCustomResourceStatus patches the status subresource of the custom resource stored in the context.
// This function assumes that the context contains a ReconcilerContextData with the CustomResource field populated.
// The step "FindControllerResource" does exactly that, populating the context.
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
