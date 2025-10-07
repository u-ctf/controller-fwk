package library

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newResolveDependencyStep[
	ControllerResourceType ControllerResource,
](
	reconciler Reconciler[ControllerResourceType],
	dependency GenericDependencyResource,
) Step {
	return Step{
		Name: fmt.Sprintf(StepResolveDependency, dependency.Kind()),
		Step: func(ctx context.Context, req ctrl.Request) StepResult {
			controller := reconciler.GetCustomResource()
			controllerStatus := controller.GetStatus()

			depKey := dependency.Key()
			dep := dependency.New()
			dep.SetName(depKey.Name)
			dep.SetNamespace(depKey.Namespace)

			dependencyRef, err := NewObjectReference(reconciler, dep, metav1.ConditionUnknown, controller.GetGeneration())
			if err != nil {
				return ResultInError(errors.Wrap(err, "failed to create dependency resource ref"))
			}

			err = reconciler.Get(ctx, depKey, dep)
			if err != nil {
				dependencyRef.ObservedGeneration = controller.GetGeneration()
				dependencyRef.Status = metav1.ConditionFalse
				dependencyRef.Reason = ReasonNotFound
				dependencyRef.Message = err.Error()

				changed := controllerStatus.Dependencies.Set(dependencyRef)
				if changed {
					if err := reconciler.Status().Update(ctx, controller); err != nil {
						return ResultInError(errors.Wrap(err, "failed to update status"))
					}
				}

				if client.IgnoreNotFound(err) != nil {
					return ResultInError(errors.Wrap(err, "failed to get dependency resource"))
				}

				if isFinalizing(reconciler) {
					return ResultSuccess()
				}

				return ResultRequeueIn(30 * time.Second)
			}

			dependency.Set(dep)

			if isFinalizing(reconciler) {
				changed, err := RemoveManagedBy(dep, controller, reconciler.GetScheme())
				if err != nil {
					return ResultInError(err)
				}
				if changed {
					if err := reconciler.Update(ctx, dep); err != nil {
						return ResultInError(err)
					}
				}

				return ResultSuccess()
			}

			// Setup watch if not already set
			result := SetupWatch(reconciler, dep, true)(ctx, req)
			if result.ShouldReturn() {
				return result.FromSubStep()
			}

			changed, err := AddManagedBy(dep, controller, reconciler.GetScheme())
			if err != nil {
				return ResultInError(err)
			}
			if changed {
				if err := reconciler.Update(ctx, dep); err != nil {
					return ResultInError(err)
				}
			}

			if dependency.ShouldWaitForReady() {
				result := waitForDependencyReady(reconciler, dependency, dependencyRef, dep)(ctx, req)
				if result.ShouldReturn() {
					return result
				}
			}

			dependencyRef.Status = metav1.ConditionTrue
			dependencyRef.Reason = ""
			dependencyRef.Message = ""
			dependencyRef.ObservedGeneration = controller.GetGeneration()
			changed = controllerStatus.Dependencies.Set(dependencyRef)
			if changed {
				if err := reconciler.Status().Update(ctx, controller); err != nil {
					return ResultInError(errors.Wrap(err, "failed to update status"))
				}
			}

			return ResultSuccess()
		},
	}
}

func waitForDependencyReady[
	ControllerResourceType ControllerResource,
](
	reconciler Reconciler[ControllerResourceType],
	dependency GenericDependencyResource,
	dependencyRef *ObjectReference,
	resource client.Object,
) func(ctx context.Context, req ctrl.Request) StepResult {
	return func(ctx context.Context, req ctrl.Request) StepResult {
		controller := reconciler.GetCustomResource()
		controllerStatus := controller.GetStatus()

		// Wait for actual to be ready
		dependencyStatus := dependency.Status(resource)
		readyCondition := meta.FindStatusCondition(dependencyStatus.Conditions, ConditionTypeReady)

		if readyCondition != nil {
			dependencyRef.Status = readyCondition.Status
			if readyCondition.Status == metav1.ConditionTrue {
				dependencyRef.Reason = ""
				dependencyRef.Message = ""
			} else {
				dependencyRef.Reason = readyCondition.Reason
				dependencyRef.Message = readyCondition.Message
			}
		} else {
			dependencyRef.Status = metav1.ConditionUnknown
			dependencyRef.Reason = ReasonUnknown
			dependencyRef.Message = "the dependency resource is not ready"
		}

		changed := controllerStatus.Dependencies.Set(dependencyRef)
		if changed {
			err := reconciler.Status().Update(ctx, controller)
			if err != nil {
				return ResultInError(errors.Wrap(err, "failed to update status"))
			}
		}

		if readyCondition == nil || readyCondition.Status != metav1.ConditionTrue {
			return ResultEarlyReturn()
		}

		return ResultSuccess()
	}
}
