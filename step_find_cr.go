package ctrlfwk

import (
	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewFindControllerCustomResourceStep[
	ControllerResourceType ControllerCustomResource,
	ContextType Context[ControllerResourceType],
](
	_ ContextType,
	reconciler Reconciler[ControllerResourceType],
) Step[ControllerResourceType, ContextType] {
	return Step[ControllerResourceType, ContextType]{
		Name: StepFindControllerCustomResource,
		Step: func(ctx ContextType, logger logr.Logger, req ctrl.Request) StepResult {
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

			// Check labels for pause
			labels := cr.GetLabels()
			if labels != nil {
				if _, ok := labels[LabelReconciliationPaused]; ok {
					logger.Info("Reconciliation is paused for this resource, skipping further steps")
					return ResultEarlyReturn()
				}
			}

			// Set the controller resource in the reconciler
			ctx.SetCustomResource(cr)

			return ResultSuccess()
		},
	}
}
