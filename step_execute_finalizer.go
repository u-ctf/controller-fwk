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

func NilFinalizerFunc(ctx context.Context, logger logr.Logger, req ctrl.Request) (done bool, err error) {
	return true, nil
}

func NewExecuteFinalizerStep[
	ControllerResourceType ControllerCustomResource,
](
	reconciler Reconciler[ControllerResourceType],
	finalizerName string,
	finalizerFunc FinalizingFunc,
) Step[ControllerResourceType] {
	return Step[ControllerResourceType]{
		Name: fmt.Sprintf(StepExecuteFinalizer, finalizerName),
		Step: func(ctx Context[ControllerResourceType], logger logr.Logger, req ctrl.Request) StepResult {
			cr := ctx.GetCustomResource()

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
					err := reconciler.Patch(ctx, cr, client.MergeFrom(ctx.GetCleanCustomResource()))
					if err != nil {
						return ResultInError(err)
					}
				}
			}

			return ResultSuccess()
		},
	}
}
