package ctrlfwk

import (
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

func NewResolveDynamicDependenciesStep[
	ControllerResourceType ControllerCustomResource,
	ContextType Context[ControllerResourceType],
](
	_ ContextType,
	reconciler ReconcilerWithDependencies[ControllerResourceType, ContextType],
) Step[ControllerResourceType, ContextType] {
	return Step[ControllerResourceType, ContextType]{
		Name: StepResolveDependencies,
		Step: func(ctx ContextType, logger logr.Logger, req ctrl.Request) StepResult {
			dependencies, err := reconciler.GetDependencies(ctx, req)
			if err != nil {
				return ResultInError(errors.Wrap(err, "failed to get dependencies"))
			}

			var returnResults []StepResult

			// Add the finalizer to clean up "managed by" references
			// in dependencies when the CR is deleted
			subStep := NewAddFinalizerStep(ctx, reconciler, FinalizerDependenciesManagedBy)
			result := subStep.Step(ctx, logger, req)
			if result.ShouldReturn() {
				return result.FromSubStep()
			}

			for _, dependency := range dependencies {
				subStepLogger := logger.WithValues("dependency", dependency.ID())

				subStep := NewResolveDependencyStep(ctx, reconciler, dependency)
				result := subStep.Step(ctx, subStepLogger, req)
				if result.ShouldReturn() {
					subStepLogger.Info("Dependency resolution resulted in early return or error")
					returnResults = append(returnResults, result)
					continue
				}
				subStepLogger.Info("Resolved dependency successfully")
			}

			// Return result errors first
			for _, result := range returnResults {
				if result.err != nil {
					if IsFinalizing(ctx.GetCustomResource()) && apierrors.IsNotFound(result.err) {
						continue
					}
					return result
				}
			}

			// Remove the finalizer, ExecuteFinalizerStep will handle actual removal when finalizing
			subStep = NewExecuteFinalizerStep(ctx, reconciler, FinalizerDependenciesManagedBy, NilFinalizerFunc)
			result = subStep.Step(ctx, logger, req)
			if result.ShouldReturn() {
				return result.FromSubStep()
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
