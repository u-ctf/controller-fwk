package ctrlfwk

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

func NewReconcileChildrenStep[
	ControllerResourceType ControllerCustomResource,
](
	reconciler ReconcilerWithDynamicChildren[ControllerResourceType],
) Step {
	return Step{
		Name: StepReconcileChildren,
		Step: func(ctx context.Context, logger logr.Logger, req ctrl.Request) StepResult {
			children, err := reconciler.GetChildren(ctx, req)
			if err != nil {
				return ResultInError(errors.Wrap(err, "failed to get children"))
			}

			var returnResults []StepResult

			for _, child := range children {
				subStepLogger := logger.WithValues("child", child.ID())

				subStepLogger.Info("Reconciling child")
				subStep := NewReconcileChildStep(reconciler, child)
				result := subStep.Step(ctx, subStepLogger, req)
				if result.ShouldReturn() {
					subStepLogger.Info("Child reconciliation resulted in early return or error")
					returnResults = append(returnResults, result)
					continue
				}
				subStepLogger.Info("Reconciled child successfully")
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
