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
](
	reconciler Reconciler[ControllerResourceType],
	finalizerName string,
) Step[ControllerResourceType] {
	return Step[ControllerResourceType]{
		Name: fmt.Sprintf(StepAddFinalizer, finalizerName),
		Step: func(ctx Context[ControllerResourceType], logger logr.Logger, req ctrl.Request) StepResult {
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
