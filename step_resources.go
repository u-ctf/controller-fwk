package ctrlfwk

import (
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

func NewReconcileResourcesStep[
	ControllerResourceType ControllerCustomResource,
	ContextType Context[ControllerResourceType],
](
	_ ContextType,
	reconciler ReconcilerWithResources[ControllerResourceType, ContextType],
) Step[ControllerResourceType, ContextType] {
	return Step[ControllerResourceType, ContextType]{
		Name: StepReconcileResources,
		Step: func(ctx ContextType, logger logr.Logger, req ctrl.Request) StepResult {
			resources, err := reconciler.GetResources(ctx, req)
			if err != nil {
				return ResultInError(errors.Wrap(err, "failed to get resources"))
			}

			var returnResults []StepResult

			for _, resource := range resources {
				subStepLogger := logger.WithValues("resource", resource.ID())

				subStep := NewReconcileResourceStep(ctx, reconciler, resource)
				result := subStep.Step(ctx, subStepLogger, req)
				if result.ShouldReturn() {
					subStepLogger.Info("Resource reconciliation resulted in early return or error")
					returnResults = append(returnResults, result)
					continue
				}
				subStepLogger.Info("Reconciled resource successfully")
			}

			// Return result errors first
			for _, result := range returnResults {
				if result.err != nil {
					return result
				}
			}

			for _, result := range returnResults {
				if result.ShouldReturn() {
					return result
				}
			}

			return ResultSuccess()
		},
	}
}
