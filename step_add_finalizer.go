package ctrlfwk

import (
	"context"
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
) Step {
	return Step{
		Name: fmt.Sprintf(StepAddFinalizer, finalizerName),
		Step: func(ctx context.Context, logger logr.Logger, req ctrl.Request) StepResult {
			cr := reconciler.GetCustomResource()

			if IsFinalizing(cr) {
				return ResultSuccess()
			}

			changed := controllerutil.AddFinalizer(cr, finalizerName)
			if changed {
				err := reconciler.Patch(context.TODO(), cr, client.MergeFrom(reconciler.GetCleanCustomResource()))
				if err != nil {
					return ResultInError(err)
				}
			}

			return ResultSuccess()
		},
	}
}
