package ctrlfwk

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type FinalizingFunc func(ctx context.Context, logger logr.Logger, req ctrl.Request) (done bool, err error)

func NewExecuteFinalizerStep[
	ControllerResourceType ControllerCustomResource,
](
	reconciler Reconciler[ControllerResourceType],
	finalizerName string,
	finalizerFunc FinalizingFunc,
) Step {
	return Step{
		Name: fmt.Sprintf(StepExecuteFinalizer, finalizerName),
		Step: func(ctx context.Context, logger logr.Logger, req ctrl.Request) StepResult {
			cr := reconciler.GetCustomResource()

			if !IsFinalizing(cr) {
				return ResultSuccess()
			}

			done, err := finalizerFunc(ctx, logger, req)
			if err != nil {
				return ResultInError(err)
			}

			if done {
				// Remove finalizer from CR
				changed := controllerutil.RemoveFinalizer(cr, finalizerName)
				if changed {
					err := reconciler.Patch(context.TODO(), cr, client.MergeFrom(reconciler.GetCleanCustomResource()))
					if err != nil {
						return ResultInError(err)
					}
				}
			}

			return ResultSuccess()
		},
	}
}
