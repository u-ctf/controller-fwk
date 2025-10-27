package ctrlfwk

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

func NewResolveDynamicDependenciesStep[
	ControllerResourceType ControllerCustomResource,
](
	reconciler ReconcilerWithDependencies[ControllerResourceType],
) Step {
	return Step{
		Name: StepResolveDependencies,
		Step: func(ctx context.Context, logger logr.Logger, req ctrl.Request) StepResult {
			dependencies, err := reconciler.GetDependencies(ctx, req)
			if err != nil {
				return ResultInError(errors.Wrap(err, "failed to get dependencies"))
			}

			var returnResults []StepResult

			for _, dependency := range dependencies {
				subStepLogger := logger.WithValues("dependency", dependency.ID())

				subStepLogger.Info("Resolving dependency")
				subStep := NewResolveDependencyStep(reconciler, dependency)
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
