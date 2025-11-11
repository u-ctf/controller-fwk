package ctrlfwk

import (
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

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
