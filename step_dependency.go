package ctrlfwk

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewResolveDependencyStep[
	ControllerResourceType ControllerCustomResource,
](
	reconciler Reconciler[ControllerResourceType],
	dependency GenericDependency,
) Step {
	return Step{
		Name: fmt.Sprintf(StepResolveDependency, dependency.Kind()),
		Step: func(ctx context.Context, logger logr.Logger, req ctrl.Request) StepResult {
			var dep client.Object

			funcResult := func() StepResult {
				if err := dependency.BeforeReconcile(ctx); err != nil {
					return ResultInError(errors.Wrap(err, "failed to run BeforeReconcile hook"))
				}

				controller := reconciler.GetCustomResource()

				depKey := dependency.Key()
				dep = dependency.New()

				err := reconciler.Get(ctx, depKey, dep)
				if err != nil {
					if client.IgnoreNotFound(err) != nil {
						return ResultInError(errors.Wrap(err, "failed to get dependency resource"))
					}

					if isFinalizing(reconciler) {
						return ResultSuccess()
					}

					return ResultRequeueIn(30 * time.Second)
				}
				cleanDep := dep.DeepCopyObject().(client.Object)

				dependency.Set(dep)

				if isFinalizing(reconciler) {
					changed, err := RemoveManagedBy(dep, controller, reconciler.Scheme())
					if err != nil {
						return ResultInError(err)
					}
					if changed {
						if err := reconciler.Patch(ctx, dep, client.MergeFrom(cleanDep)); err != nil {
							return ResultInError(err)
						}
					}

					return ResultSuccess()
				}

				// Setup watch if we can
				reconcilerWithWatcher, ok := reconciler.(ReconcilerWithWatcher[ControllerResourceType])
				if ok {
					result := SetupWatch(reconcilerWithWatcher, dep, true)(ctx, req)
					if result.ShouldReturn() {
						return result.FromSubStep()
					}
				}

				changed, err := AddManagedBy(dep, controller, reconciler.Scheme())
				if err != nil {
					return ResultInError(err)
				}
				if changed {
					if err := reconciler.Patch(ctx, dep, client.MergeFrom(cleanDep)); err != nil {
						return ResultInError(err)
					}
				}

				if dependency.ShouldWaitForReady() && !dependency.IsReady() {
					return ResultRequeueIn(30 * time.Second)
				}

				return ResultSuccess()
			}()

			if err := dependency.AfterReconcile(ctx, dep); err != nil {
				return ResultInError(errors.Wrap(err, "failed to run AfterReconcile hook"))
			}

			return funcResult
		},
	}
}
