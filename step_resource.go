package ctrlfwk

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func NewReconcileResourceStep[
	ControllerResourceType ControllerCustomResource,
	ContextType Context[ControllerResourceType],
](
	_ ContextType,
	reconciler Reconciler[ControllerResourceType],
	resource GenericResource[ControllerResourceType, ContextType],
) Step[ControllerResourceType, ContextType] {
	return Step[ControllerResourceType, ContextType]{
		Name: fmt.Sprintf(StepReconcileResource, resource.Kind()),
		Step: func(ctx ContextType, logger logr.Logger, req ctrl.Request) StepResult {
			var desired client.Object
			var result StepResult

			funcResult := func() StepResult {
				cr := ctx.GetCustomResource()

				if IsFinalizing(cr) {
					// If the resource does not require deletion, we can just finish here, it's gonna get garbage collected
					if !resource.RequiresManualDeletion(resource.Get()) {
						if err := resource.OnFinalize(ctx, desired); err != nil {
							return ResultInError(errors.Wrap(err, "failed to run OnFinalize hook"))
						}

						return ResultSuccess()
					}
				}

				if err := resource.BeforeReconcile(ctx); err != nil {
					return ResultInError(errors.Wrap(err, "failed to run BeforeReconcile hook"))
				}

				desired, result = getDesiredObject(reconciler, resource)(ctx, req)
				if result.ShouldReturn() {
					return result.FromSubStep()
				}

				if resource.CanBePaused() {
					labels := cr.GetLabels()
					if labels != nil {
						if _, ok := labels[LabelReconciliationPaused]; ok {
							logger.Info("Reconciliation is paused for this resource, skipping reconciliation step")
							return ResultSuccess()
						}
					}
				}

				if IsFinalizing(cr) {
					if err := reconciler.Delete(ctx, desired); client.IgnoreNotFound(err) != nil {
						return ResultInError(errors.Wrap(err, "failed to delete resource"))
					}

					if err := resource.OnFinalize(ctx, desired); err != nil {
						return ResultInError(errors.Wrap(err, "failed to run OnFinalize hook"))
					}

					return ResultSuccess()
				}

				// Setup watch if we can
				reconcilerWithWatcher, ok := reconciler.(ReconcilerWithWatcher[ControllerResourceType])
				if ok {
					result = SetupWatch(reconcilerWithWatcher, desired, false)(ctx, req)
					if result.ShouldReturn() {
						return result.FromSubStep()
					}
				}

				patchResult, err := controllerutil.CreateOrPatch(ctx, reconciler, desired, resource.GetMutator(desired))
				if err != nil {
					return ResultInError(errors.Wrap(err, "failed to create or patch resource"))
				}

				resource.Set(desired)

				switch patchResult {
				case controllerutil.OperationResultCreated:
					if err := resource.OnCreate(ctx, desired); err != nil {
						return ResultInError(errors.Wrap(err, "failed to run OnCreate hook"))
					}
				case controllerutil.OperationResultUpdated:
					if err := resource.OnUpdate(ctx, desired); err != nil {
						return ResultInError(errors.Wrap(err, "failed to run OnUpdate hook"))
					}
				}

				if !resource.IsReady(desired) {
					return ResultEarlyReturn()
				}

				return ResultSuccess()
			}()

			if err := resource.AfterReconcile(ctx, desired); err != nil {
				return ResultInError(errors.Wrap(err, "failed to run AfterReconcile hook"))
			}

			return funcResult
		},
	}
}

func getDesiredObject[
	ControllerResourceType ControllerCustomResource,
	ContextType Context[ControllerResourceType],
](
	reconciler Reconciler[ControllerResourceType],
	resource GenericResource[ControllerResourceType, ContextType],
) func(ctx ContextType, req ctrl.Request) (client.Object, StepResult) {
	return func(ctx ContextType, req ctrl.Request) (client.Object, StepResult) {
		desired, delete, err := resource.ObjectMetaGenerator()
		if delete {
			if desired != nil && desired.GetName() != "" {
				err := reconciler.Delete(ctx, desired)
				if client.IgnoreNotFound(err) != nil {
					return nil, ResultInError(errors.Wrap(err, "failed to delete resource"))
				}

				if err == nil {
					if err := resource.OnDelete(ctx, desired); err != nil {
						return nil, ResultInError(errors.Wrap(err, "failed to run OnDelete hook"))
					}
				}
			}
			return nil, ResultEarlyReturn()
		}
		if err != nil {
			return nil, ResultInError(errors.Wrap(err, "failed to generate resource"))
		}

		return desired, ResultSuccess()
	}
}
