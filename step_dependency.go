package ctrlfwk

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DependencyStepResult struct {
	earlyReturn  bool
	err          error
	requeueAfter time.Duration
	wasFound     bool
}

var _ StepResult = DependencyStepResult{}

func (result DependencyStepResult) WasFound() bool {
	return result.wasFound
}

// Error returns the underlying step error, if any.
func (result DependencyStepResult) Error() error {
	return result.err
}

// RequeueAfter returns the requested requeue duration, if any.
func (result DependencyStepResult) RequeueAfter() time.Duration {
	return result.requeueAfter
}

// IsEarlyReturn reports whether the step requested an early return without error.
func (result DependencyStepResult) IsEarlyReturn() bool {
	return result.earlyReturn
}

// IsSuccess reports whether the step completed without error, requeue, or early return.
func (result DependencyStepResult) IsSuccess() bool {
	return !result.ShouldReturn()
}

// ShouldReturn reports whether reconciliation should stop after this step.
func (result DependencyStepResult) ShouldReturn() bool {
	return result.err != nil || result.requeueAfter > 0 || result.earlyReturn
}

// FromSubStep clears early return semantics when bubbling a nested step result upward.
func (result DependencyStepResult) FromSubStep() StepResult {
	result.earlyReturn = false
	return result
}

// Normal converts the step result into a controller-runtime reconcile result.
func (result DependencyStepResult) Normal() (ctrl.Result, error) {
	if result.err != nil {
		return ctrl.Result{}, result.err
	}
	if result.requeueAfter > 0 {
		return ctrl.Result{RequeueAfter: result.requeueAfter}, nil
	}
	return ctrl.Result{}, nil
}

func NewResolveDependencyStep[
	ControllerResourceType ControllerCustomResource,
	ContextType Context[ControllerResourceType],
](
	_ ContextType,
	reconciler Reconciler[ControllerResourceType],
	dependency GenericDependency[ControllerResourceType, ContextType],
) Step[ControllerResourceType, ContextType] {
	return Step[ControllerResourceType, ContextType]{
		Name: fmt.Sprintf(StepResolveDependency, dependency.Kind()),
		Step: func(ctx ContextType, logger logr.Logger, req ctrl.Request) StepResult {
			var dep client.Object

			funcResult := func() StepResult {
				if err := dependency.BeforeReconcile(ctx); err != nil {
					return DependencyStepResult{
						err: errors.Wrap(err, "failed to run BeforeReconcile hook"),
					}
				}

				cr := ctx.GetCustomResource()

				depKey := dependency.Key()
				dep = dependency.New()

				err := reconciler.Get(ctx, depKey, dep)
				if err != nil {
					if client.IgnoreNotFound(err) != nil {
						return DependencyStepResult{
							err: errors.Wrap(err, "failed to get dependency resource"),
						}
					}

					if IsFinalizing(cr) || dependency.IsOptional() {
						return DependencyStepResult{
							wasFound: false,
						}
					}

					return DependencyStepResult{
						requeueAfter: 30 * time.Second,
					}
				}
				cleanDep := dep.DeepCopyObject().(client.Object)

				dependency.Set(dep)

				if IsFinalizing(cr) {
					changed, err := RemoveManagedBy(dep, cr, reconciler.Scheme())
					if client.IgnoreNotFound(err) != nil {
						return DependencyStepResult{
							err: err,
						}
					}
					if changed {
						if err := reconciler.Patch(ctx, dep, client.MergeFrom(cleanDep)); err != nil {
							return DependencyStepResult{
								err: err,
							}
						}
					}

					return DependencyStepResult{
						wasFound: true,
					}
				}

				if dependency.ShouldAddManagedByAnnotation() {
					// Setup watch if we can
					reconcilerWithWatcher, ok := reconciler.(ReconcilerWithWatcher[ControllerResourceType])
					if ok {
						result := SetupWatch(reconcilerWithWatcher, dep, true)(ctx, req)
						if result.ShouldReturn() {
							return result.FromSubStep()
						}
					}

					changed, err := AddManagedBy(dep, cr, reconciler.Scheme())
					if err != nil {
						return DependencyStepResult{
							err: err,
						}
					}
					if changed {
						if err := reconciler.Patch(ctx, dep, client.MergeFrom(cleanDep)); err != nil {
							return DependencyStepResult{
								err: err,
							}
						}
					}
				}

				if dependency.ShouldWaitForReady() && !dependency.IsReady() {
					return DependencyStepResult{
						requeueAfter: 30 * time.Second,
					}
				}

				return DependencyStepResult{
					wasFound: true,
				}
			}()

			if err := dependency.AfterReconcile(ctx, dep); err != nil {
				return DependencyStepResult{
					err: errors.Wrap(err, "failed to run AfterReconcile hook"),
				}
			}

			return funcResult
		},
	}
}
