package library

import (
	"context"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	defaultReadyCondition = metav1.Condition{
		Type:    ConditionTypeReady,
		Reason:  ReasonReconciling,
		Message: "the resource is being reconciled for the first time",
		Status:  metav1.ConditionFalse,
	}
)

func NewFindControllerResourceStep[
	ControllerResourceType ControllerResource,
](
	reconciler Reconciler[ControllerResourceType],
) Step {
	return Step{
		Name: StepFindControllerResource,
		Step: func(ctx context.Context, req ctrl.Request) StepResult {
			hub := sentry.GetHubFromContext(ctx)

			tx := sentry.StartTransaction(ctx, "step.find_controller_resource")
			defer tx.Finish()

			// Add breadcrumb: beginning of the step
			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "controller.step",
				Message:  "Finding controller resource",
				Level:    sentry.LevelInfo,
				Data: map[string]any{
					"step": StepFindControllerResource,
				},
			}, nil)

			// Get the controller resource
			controllerResource := reconciler.GetCustomResource()

			// Get the controller resource from the client
			err := reconciler.Get(ctx, req.NamespacedName, controllerResource)
			if err != nil {
				if client.IgnoreNotFound(err) != nil {
					// If the resource is not found, return early
					return ResultInError(errors.Wrap(err, "failed to get controller resource"))
				}

				return ResultEarlyReturn()
			}

			// Set the finalizer if not already set
			changed := controllerutil.AddFinalizer(controllerResource, reconciler.GetFinalizer())
			if changed {
				err = reconciler.Update(ctx, controllerResource)
				if err != nil {
					return ResultInError(errors.Wrap(err, "failed to update controller resource"))
				}
			}

			// Set the controller resource in the reconciler
			reconciler.SetCustomResource(controllerResource)

			controllerResourceStatus := controllerResource.GetStatus()

			// Set the ready condition if it doesn't exist
			readyCondition, defaulted := controllerResourceStatus.FindOrDefaultCondition(defaultReadyCondition)
			if defaulted {
				readyCondition.ObservedGeneration = controllerResource.GetGeneration()
				meta.SetStatusCondition(&controllerResourceStatus.Conditions, *readyCondition)

				err = reconciler.Status().Update(ctx, controllerResource)
				if err != nil {
					return ResultInError(errors.Wrap(err, "failed to update controller resource status"))
				}
			} else if readyCondition.ObservedGeneration != controllerResource.GetGeneration() {
				// If the observed generation is not equal to the current generation, update it
				readyCondition.ObservedGeneration = controllerResource.GetGeneration()
				readyCondition.Status = metav1.ConditionFalse
				readyCondition.Reason = ReasonReconciling
				readyCondition.Message = "the resource is being reconciled"
				changed = meta.SetStatusCondition(&controllerResourceStatus.Conditions, *readyCondition)
				if changed {
					err = reconciler.Status().Update(ctx, controllerResource)
					if err != nil {
						return ResultInError(errors.Wrap(err, "failed to update controller resource status"))
					}
				}

			}

			// If it's finalizing, change the ready condition to false and set the reason
			if isFinalizing(reconciler) {
				readyCondition.Status = metav1.ConditionFalse
				readyCondition.Reason = ReasonFinalizing
				readyCondition.Message = "the resource is being finalized"
				readyCondition.ObservedGeneration = controllerResource.GetGeneration()

				changed = meta.SetStatusCondition(&controllerResourceStatus.Conditions, *readyCondition)
				if changed {
					err = reconciler.Status().Update(ctx, controllerResource)
					if err != nil {
						return ResultInError(errors.Wrap(err, "failed to update controller resource status"))
					}
				}
			}

			return ResultSuccess()
		},
	}
}
