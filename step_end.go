package ctrlfwk

import (
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewEndStep creates a regular terminal step that runs only if all previous steps succeeded.
//
// For status updates that must run on success, requeue, and error paths alike, prefer
// WithFinalStep together with NewReadyConditionFinalStep.
func NewEndStep[
	ControllerResourceType ControllerCustomResource,
	ContextType Context[ControllerResourceType],
](
	_ ContextType,
	reconciler Reconciler[ControllerResourceType],
	setReadyCondF func(ControllerResourceType) (bool, error),
) Step[ControllerResourceType, ContextType] {
	return Step[ControllerResourceType, ContextType]{
		Name: StepEndReconciliation,
		Step: func(ctx ContextType, logger logr.Logger, req ctrl.Request) StepResult {
			cr := ctx.GetCustomResource()

			// Set Ready condition
			if setReadyCondF != nil {
				changed, err := setReadyCondF(cr)
				if err != nil {
					return ResultInError(errors.Wrap(err, "failed to set ready condition"))
				}

				if changed {
					if err = PatchCustomResourceStatus(ctx, reconciler); err != nil {
						return ResultInError(errors.Wrap(err, "failed to update controller resource"))
					}
				}
			}

			return ResultSuccess()
		},
	}
}

// NewReadyConditionFinalStep creates a final step that patches the controller resource Ready
// condition from the last executed step result.
//
// This final step is intended to be registered with StepperBuilder.WithFinalStep so that the
// Ready condition is updated even when reconciliation exits early or returns an error.
//
// If the controller resource was not loaded successfully, the final step becomes a no-op.
func NewReadyConditionFinalStep[
	ControllerResourceType ControllerCustomResource,
	ContextType Context[ControllerResourceType],
](
	_ ContextType,
	reconciler Reconciler[ControllerResourceType],
	setReadyCondF func(ControllerResourceType, string, StepResult) (bool, error),
) FinalStep[ControllerResourceType, ContextType] {
	return NewFinalStep(
		StepEndReconciliation,
		func(ctx ContextType, logger logr.Logger, req ctrl.Request, lastStepName string, lastStepResult StepResult) error {
			if setReadyCondF == nil {
				return nil
			}

			// If we failed to load the controller resource, there is nothing safe to patch.
			if lastStepName == StepFindControllerCustomResource && !lastStepResult.IsSuccess() {
				return nil
			}

			cr := ctx.GetCustomResource()
			if cr.GetName() == "" {
				return nil
			}

			changed, err := setReadyCondF(cr, lastStepName, lastStepResult)
			if err != nil {
				return errors.Wrap(err, "failed to set ready condition")
			}

			if !changed {
				return nil
			}

			if err = PatchCustomResourceStatus(ctx, reconciler); err != nil {
				return errors.Wrap(err, "failed to update controller resource")
			}

			return nil
		},
	)
}
