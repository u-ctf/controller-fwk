package ctrlfwk

import (
	"fmt"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func NewAddFinalizerStep[
	ControllerResourceType ControllerCustomResource,
	ContextType Context[ControllerResourceType],
](
	_ ContextType,
	reconciler Reconciler[ControllerResourceType],
	finalizerName string,
) Step[ControllerResourceType, ContextType] {
	return Step[ControllerResourceType, ContextType]{
		Name: fmt.Sprintf(StepAddFinalizer, finalizerName),
		Step: func(ctx ContextType, logger logr.Logger, req ctrl.Request) StepResult {
			cr := ctx.GetCustomResource()

			if IsFinalizing(cr) {
				return ResultSuccess()
			}

			changed := controllerutil.AddFinalizer(cr, finalizerName)
			if changed {
				err := reconciler.Patch(ctx, cr, client.MergeFrom(ctx.GetCleanCustomResource()))
				if err != nil {
					return ResultInError(err)
				}
			}

			return ResultSuccess()
		},
	}
}
