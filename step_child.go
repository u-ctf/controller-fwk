package library

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rxwycdh/rxhash"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	HashAnnotation = "multi.ch/last-applied-hash"
)

type ChildGenerator[ChildType client.Object] func(ctx context.Context, req ctrl.Request) (child ChildType, skip bool, err error)

func GenericGetter[
	ControllerResourceType ControllerResource,
	ChildType client.Object,
](ctx context.Context, reconciler Reconciler[ControllerResourceType], desired ChildType) (actual client.Object, err error) {
	actual = NewInstanceOf(desired)
	err = reconciler.Get(ctx, client.ObjectKey{
		Name:      desired.GetName(),
		Namespace: desired.GetNamespace(),
	}, actual)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get child resource")
	}

	return actual, nil
}

func NewReconcileChildStep[
	ControllerResourceType ControllerResource,
](
	reconciler Reconciler[ControllerResourceType],
	child GenericChildResource,
) Step {
	return Step{
		Name: fmt.Sprintf(StepReconcileChild, child.Kind()),
		Step: func(ctx context.Context, req ctrl.Request) StepResult {
			controller := reconciler.GetCustomResource()
			controllerStatus := controller.GetStatus()

			desired, result := getDesiredObject(reconciler, child)(ctx, req)
			if result.ShouldReturn() {
				return result.FromSubStep()
			}

			childRef, err := NewObjectReference(reconciler, desired, metav1.ConditionUnknown, controller.GetGeneration())
			if err != nil {
				return ResultInError(errors.Wrap(err, "failed to create child resource ref"))
			}

			actual, err := GenericGetter(ctx, reconciler, desired)
			if client.IgnoreNotFound(err) != nil {
				return ResultInError(err)
			}

			requiresCreation := actual == nil

			result = handleFinalization(reconciler, childRef, actual, requiresCreation)(ctx, req)
			if result.ShouldReturn() {
				return result.FromSubStep()
			}

			// Setup watch if not already set
			result = SetupWatch(reconciler, desired, false)(ctx, req)
			if result.ShouldReturn() {
				return result.FromSubStep()
			}

			resource, result := handleCreateOrUpdate(reconciler, desired, actual, requiresCreation)(ctx, req)
			if result.ShouldReturn() {
				childRef.Reason = result.err.Error()
				childRef.Status = metav1.ConditionFalse
				changed := controllerStatus.ChildResources.Set(childRef)
				if changed {
					err := reconciler.Status().Update(ctx, controller)
					if err != nil {
						return ResultInError(errors.Wrap(err, "failed to update status"))
					}
				}

				return result.FromSubStep()
			}

			childRef.UID = string(resource.GetUID())
			child.Set(resource)

			result = waitForChildReady(reconciler, child, childRef, resource)(ctx, req)
			if result.ShouldReturn() {
				return result
			}

			return ResultSuccess()
		},
	}
}

func waitForChildReady[
	ControllerResourceType ControllerResource,
](
	reconciler Reconciler[ControllerResourceType],
	child GenericChildResource,
	childRef *ObjectReference,
	resource client.Object,
) func(ctx context.Context, req ctrl.Request) StepResult {
	return func(ctx context.Context, req ctrl.Request) StepResult {
		controller := reconciler.GetCustomResource()
		controllerStatus := controller.GetStatus()

		// Wait for actual to be ready
		childStatus := child.Status(resource)
		readyCondition := meta.FindStatusCondition(childStatus.Conditions, ConditionTypeReady)

		if readyCondition != nil {
			childRef.Status = readyCondition.Status
			if readyCondition.Status == metav1.ConditionTrue {
				childRef.Reason = ""
				childRef.Message = ""
			} else {
				childRef.Reason = readyCondition.Reason
				childRef.Message = readyCondition.Message
			}
		} else {
			childRef.Status = metav1.ConditionUnknown
			childRef.Reason = ReasonUnknown
			childRef.Message = "the child resource is not ready"
		}

		changed := controllerStatus.ChildResources.Set(childRef)
		if changed {
			err := reconciler.Status().Update(ctx, controller)
			if err != nil {
				return ResultInError(errors.Wrap(err, "failed to update status"))
			}
		}

		if readyCondition == nil || readyCondition.Status != metav1.ConditionTrue {
			return ResultEarlyReturn()
		}

		return ResultSuccess()
	}
}

func handleCreateOrUpdate[
	ControllerResourceType ControllerResource,
](
	reconciler Reconciler[ControllerResourceType],
	desired client.Object,
	actual client.Object,
	requiresCreation bool,
) func(ctx context.Context, req ctrl.Request) (client.Object, StepResult) {
	return func(ctx context.Context, req ctrl.Request) (client.Object, StepResult) {
		if requiresCreation {
			// Create the actual object with the desired object
			err := reconciler.Create(ctx, desired)
			if err != nil {
				return nil, ResultInError(fmt.Errorf("failed to create child resource: %w", err))
			}

			return desired, ResultSuccess()
		}

		desiredHash := GetAnnotation(desired, HashAnnotation)
		actualHash := GetAnnotation(actual, HashAnnotation)

		if actualHash != desiredHash {
			// Update the actual object with the desired object
			desired.SetResourceVersion(actual.GetResourceVersion())

			err := reconciler.Update(ctx, desired)
			if err != nil {
				return desired, ResultInError(fmt.Errorf("failed to update child resource: %w", err))
			}
		}

		return actual, ResultSuccess()
	}
}

func handleFinalization[
	ControllerResourceType ControllerResource,
](
	reconciler Reconciler[ControllerResourceType],
	childRef *ObjectReference,
	actual client.Object,
	requiresCreation bool,
) func(ctx context.Context, req ctrl.Request) StepResult {
	controller := reconciler.GetCustomResource()
	status := controller.GetStatus()

	return func(ctx context.Context, req ctrl.Request) StepResult {
		if isFinalizing(reconciler) {
			if requiresCreation {
				return ResultEarlyReturn()
			}
			if err := reconciler.Delete(ctx, actual); err != nil {
				return ResultInError(errors.Wrap(err, "failed to delete child resource"))
			}
			changed := status.ChildResources.Remove(childRef)
			if changed {
				err := reconciler.Status().Update(ctx, controller)
				if err != nil {
					return ResultInError(errors.Wrap(err, "failed to update status"))
				}
			}
		}

		return ResultSuccess()
	}
}

func getDesiredObject[
	ControllerResourceType ControllerResource,
](
	reconciler Reconciler[ControllerResourceType],
	child GenericChildResource,
) func(ctx context.Context, req ctrl.Request) (client.Object, StepResult) {
	controller := reconciler.GetCustomResource()
	status := controller.GetStatus()

	return func(ctx context.Context, req ctrl.Request) (client.Object, StepResult) {
		desired, skip, err := child.Generator(ctx, req)
		if skip {
			if desired != nil {
				childRef, err := EmptyObjectReference(reconciler, desired)
				if err != nil {
					return nil, ResultInError(errors.Wrap(err, "failed to create child resource ref"))
				}

				changed := status.ChildResources.Remove(childRef)
				if changed {
					err = reconciler.Status().Update(ctx, controller)
					if err != nil {
						return nil, ResultInError(errors.Wrap(err, "failed to update status"))
					}
				}
			}
			return nil, ResultEarlyReturn()
		}
		if err != nil {
			return nil, ResultInError(errors.Wrap(err, "failed to generate child resource"))
		}

		err = ctrl.SetControllerReference(controller, desired, reconciler.Scheme())
		if err != nil {
			return nil, ResultInError(errors.Wrap(err, "failed to set controller reference"))
		}

		// Set the hash annotation
		hash, err := rxhash.HashStruct(desired)
		if err != nil {
			return nil, ResultInError(errors.Wrap(err, "failed to hash child resource"))
		}

		SetAnnotation(desired, HashAnnotation, hash)

		return desired, ResultSuccess()
	}
}
