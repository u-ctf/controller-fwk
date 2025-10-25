package ctrlfwk

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

func NewEndStep[
	ControllerResourceType ControllerCustomResource,
](
	reconciler Reconciler[ControllerResourceType],
	setReadyCondF func(ControllerResourceType) (bool, error),
) Step {
	return Step{
		Name: StepEndReconciliation,
		Step: func(ctx context.Context, logger logr.Logger, req ctrl.Request) StepResult {
			// Get the controller resource
			controllerResource := reconciler.GetCustomResource()

			// Set Ready condition
			if setReadyCondF != nil {
				changed, err := setReadyCondF(controllerResource)
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
