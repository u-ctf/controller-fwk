package ctrlfwk

import (
	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewFindControllerCustomResourceStep[
	ControllerResourceType ControllerCustomResource,
](
	reconciler Reconciler[ControllerResourceType],
) Step[ControllerResourceType] {
	return Step[ControllerResourceType]{
		Name: StepFindControllerCustomResource,
		Step: func(ctx Context[ControllerResourceType], logger logr.Logger, req ctrl.Request) StepResult {
			cr := ctx.GetCustomResource()

			// Get the controller resource from the client
			err := reconciler.Get(ctx, req.NamespacedName, cr)
			if err != nil {
				if client.IgnoreNotFound(err) != nil {
					// If the resource is not found, return early
					return ResultInError(errors.Wrap(err, "failed to get controller resource"))
				}

				return ResultEarlyReturn()
			}

			// Set the controller resource in the reconciler
			ctx.SetCustomResource(cr)

			return ResultSuccess()
		},
	}
}
