package ctrlfwk

import (
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Stepper is a utility to execute a series of steps in a controller.
// It allows for easy chaining of steps and handling of errors and requeues.
// Each step can be a function that returns a StepResult, which indicates
// whether to continue, requeue, or return an error.
// The Stepper can be used in a controller's Reconcile function to manage
// the execution of multiple steps in a clean and organized manner.
type Stepper[K client.Object] struct {
	logger logr.Logger
	steps  []Step[K]
}

type StepperBuilder[K client.Object] struct {
	logger logr.Logger
	steps  []Step[K]
}

func NewStepperFor[K client.Object](logger logr.Logger) *StepperBuilder[K] {
	return &StepperBuilder[K]{
		logger: logger,
		steps:  []Step[K]{},
	}
}

// WithLogger sets the logger for the Stepper.
func (s *StepperBuilder[K]) WithStep(step Step[K]) *StepperBuilder[K] {
	s.steps = append(s.steps, step)
	return s
}

// WithLogger sets the logger for the Stepper.
func (s *StepperBuilder[K]) Build() *Stepper[K] {
	return &Stepper[K]{
		logger: s.logger,
		steps:  s.steps,
	}
}

type StepResult struct {
	earlyReturn  bool
	err          error
	requeueAfter time.Duration
}

func (result StepResult) ShouldReturn() bool {
	return result.err != nil || result.requeueAfter > 0 || result.earlyReturn
}

func (result StepResult) FromSubStep() StepResult {
	result.earlyReturn = false
	return result
}

func (result StepResult) Normal() (ctrl.Result, error) {
	if result.err != nil {
		return ctrl.Result{}, result.err
	}
	if result.requeueAfter > 0 {
		return ctrl.Result{RequeueAfter: result.requeueAfter}, nil
	}
	return ctrl.Result{}, nil
}

func ResultInError(err error) StepResult {
	return StepResult{
		err: err,
	}
}

func ResultRequeueIn(result time.Duration) StepResult {
	return StepResult{
		requeueAfter: result,
	}
}

func ResultEarlyReturn() StepResult {
	return StepResult{
		earlyReturn: true,
	}
}

func ResultSuccess() StepResult {
	return StepResult{}
}

type Step[K client.Object] struct {
	// Name is the name of the step
	Name string

	// Step is the function to execute
	Step func(ctx Context[K], logger logr.Logger, req ctrl.Request) StepResult
}

func NewStep[K client.Object](name string, step func(ctx Context[K], logger logr.Logger, req ctrl.Request) StepResult) Step[K] {
	return Step[K]{
		Name: name,
		Step: step,
	}
}

func (stepper *Stepper[K]) Execute(ctx Context[K], req ctrl.Request) (ctrl.Result, error) {
	logger := stepper.logger

	startedAt := time.Now()

	logger.Info("Inserting line return for lisibility\n\n")
	logger.Info("Starting stepper execution")

	for _, step := range stepper.steps {
		stepStartedAt := time.Now()
		result := step.Step(ctx, logger, req)
		stepDuration := time.Since(stepStartedAt)

		if result.ShouldReturn() {
			if result.err != nil {
				if IsFinalizing(ctx.GetCustomResource()) && apierrors.IsNotFound(result.err) {
					logger.Info("Resource not found during finalization, ignoring error", "step", step.Name, "stepDuration", stepDuration)
					return ResultRequeueIn(1 * time.Second).Normal()
				}

				logger.Error(result.err, "Error in step", "step", step.Name, "stepDuration", stepDuration)
			} else if result.requeueAfter > 0 {
				logger.Info("Requeueing after step", "step", step.Name, "after", result.requeueAfter, "stepDuration", stepDuration)
			} else {
				logger.Info("Early return after step", "step", step.Name, "stepDuration", stepDuration)
			}
			return result.Normal()
		}

		logger.Info("Executed step", "step", step.Name, "stepDuration", stepDuration)
	}

	logger.Info("All steps executed successfully", "duration", time.Since(startedAt))
	return ctrl.Result{}, nil
}
