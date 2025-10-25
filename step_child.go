package ctrlfwk

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	HashAnnotation = "multi.ch/last-applied-hash"
)

func GenericGetter[
	ControllerResourceType ControllerCustomResource,
	ChildType client.Object,
](ctx context.Context, reconciler Reconciler[ControllerResourceType], desired ChildType) (actual client.Object, err error) {
	actual = NewInstanceOf(desired)
	err = reconciler.Get(ctx, client.ObjectKey{
		Name:      desired.GetName(),
		Namespace: desired.GetNamespace(),
	}, actual)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get child resource")
	}

	return actual, nil
}

func NewReconcileChildStep[
	ControllerResourceType ControllerCustomResource,
](
	reconciler Reconciler[ControllerResourceType],
	child GenericChildResource,
) Step {
	return Step{
		Name: fmt.Sprintf(StepReconcileChild, child.Kind()),
		Step: func(ctx context.Context, logger logr.Logger, req ctrl.Request) StepResult {
			if isFinalizing(reconciler) {
				// If the child does not require deletion, we can just finish here, it's gonna get garbage collected
				if !child.RequiresManualDeletion(child.Get()) {
					return ResultSuccess()
				}
			}

			if err := child.OnReconcile(ctx); err != nil {
				return ResultInError(errors.Wrap(err, "failed to run OnReconcile hook"))
			}

			desired, result := getDesiredObject(reconciler, child)(ctx, req)
			if result.ShouldReturn() {
				return result.FromSubStep()
			}

			result = handleFinalization(reconciler, child, desired)(ctx, req)
			if result.ShouldReturn() {
				return result.FromSubStep()
			}

			// Setup watch if we can
			reconcilerWithWatcher, ok := reconciler.(ReconcilerWithWatcher[ControllerResourceType])
			if ok {
				result = SetupWatch(reconcilerWithWatcher, desired, false)(ctx, req)
				if result.ShouldReturn() {
					return result.FromSubStep()
				}
			}

			patchResult, err := controllerutil.CreateOrPatch(ctx, reconciler, desired, child.GetMutator(desired))
			if err != nil {
				return ResultInError(errors.Wrap(err, "failed to create or patch child resource"))
			}

			fmt.Println("PATCH RESULT", patchResult)

			switch patchResult {
			case controllerutil.OperationResultCreated:
				if err := child.OnCreate(ctx, desired); err != nil {
					return ResultInError(errors.Wrap(err, "failed to run OnCreate hook"))
				}
			case controllerutil.OperationResultUpdated:
				if err := child.OnUpdate(ctx, desired); err != nil {
					return ResultInError(errors.Wrap(err, "failed to run OnUpdate hook"))
				}
			}

			child.Set(desired)

			if !child.IsReady(desired) {
				return ResultEarlyReturn()
			}

			return ResultSuccess()
		},
	}
}

func handleFinalization[
	ControllerResourceType ControllerCustomResource,
](
	reconciler Reconciler[ControllerResourceType],
	child GenericChildResource,
	desired client.Object,
) func(ctx context.Context, req ctrl.Request) StepResult {
	return func(ctx context.Context, req ctrl.Request) StepResult {
		if isFinalizing(reconciler) {
			if err := reconciler.Delete(ctx, desired); client.IgnoreNotFound(err) != nil {
				return ResultInError(errors.Wrap(err, "failed to delete child resource"))
			}

			if err := child.OnFinalize(ctx, desired); err != nil {
				return ResultInError(errors.Wrap(err, "failed to run OnFinalize hook"))
			}
		}

		return ResultSuccess()
	}
}

func getDesiredObject[
	ControllerResourceType ControllerCustomResource,
](
	reconciler Reconciler[ControllerResourceType],
	child GenericChildResource,
) func(ctx context.Context, req ctrl.Request) (client.Object, StepResult) {
	return func(ctx context.Context, req ctrl.Request) (client.Object, StepResult) {
		desired, delete, err := child.ObjectMetaGenerator()
		if delete {
			if desired != nil {
				if err := reconciler.Delete(ctx, desired); client.IgnoreNotFound(err) != nil {
					return nil, ResultInError(errors.Wrap(err, "failed to delete child resource"))
				}

				if err := child.OnDelete(ctx, desired); err != nil {
					return nil, ResultInError(errors.Wrap(err, "failed to run OnDelete hook"))
				}
			}
			return nil, ResultEarlyReturn()
		}
		if err != nil {
			return nil, ResultInError(errors.Wrap(err, "failed to generate child resource"))
		}

		return desired, ResultSuccess()
	}
}
