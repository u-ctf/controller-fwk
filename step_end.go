package library

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	defaultEndReadyCondition = metav1.Condition{
		Type:    ConditionTypeReady,
		Reason:  ReasonReconciled,
		Status:  metav1.ConditionTrue,
		Message: "the resource reached the end of reconciliation",
	}
)

func NewEndStep[
	ControllerResourceType ControllerResource,
](
	reconciler Reconciler[ControllerResourceType],
) Step {
	return Step{
		Name: StepEndReconciliation,
		Step: func(ctx context.Context, req ctrl.Request) StepResult {
			// Get the controller resource
			controllerResource := reconciler.GetCustomResource()
			controllerResourceStatus := controllerResource.GetStatus()

			// Set the ready condition if it doesn't exist
			readyCondition := &defaultEndReadyCondition
			readyCondition.ObservedGeneration = controllerResource.GetGeneration()
			changed := meta.SetStatusCondition(&controllerResourceStatus.Conditions, *readyCondition)
			if changed {
				err := reconciler.Status().Update(ctx, controllerResource)
				if err != nil {
					return ResultInError(errors.Wrap(err, "failed to update controller resource status"))
				}
			}

			// If it's finalizing, remove the finalizer
			if isFinalizing(reconciler) {
				changed = controllerutil.RemoveFinalizer(controllerResource, reconciler.GetFinalizer())
				if changed {
					err := reconciler.Update(ctx, controllerResource)
					if err != nil {
						return ResultInError(errors.Wrap(err, "failed to update controller resource"))
					}
				}
			}

			return ResultSuccess()
		},
	}
}
