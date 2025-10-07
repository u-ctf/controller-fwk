package library

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Stepper is a utility to execute a series of steps in a controller.
// It allows for easy chaining of steps and handling of errors and requeues.
// Each step can be a function that returns a StepResult, which indicates
// whether to continue, requeue, or return an error.
// The Stepper can be used in a controller's Reconcile function to manage
// the execution of multiple steps in a clean and organized manner.
type Stepper struct {
	logger logr.Logger
	steps  []Step
}

type StepperOptions func(*Stepper)

// WithLogger sets the logger for the Stepper.
func WithStep(step Step) StepperOptions {
	return func(s *Stepper) {
		s.steps = append(s.steps, step)
	}
}

func NewStepper(logger logr.Logger, opts ...StepperOptions) *Stepper {
	stepper := &Stepper{
		logger: logger,
		steps:  []Step{},
	}

	for _, opt := range opts {
		opt(stepper)
	}

	return stepper
}

type StepResult struct {
	earlyReturn  bool
	err          error
	requeue      bool
	requeueAfter time.Duration
}

func (result StepResult) ShouldReturn() bool {
	return result.err != nil || result.requeue || result.requeueAfter > 0 || result.earlyReturn
}

func (result StepResult) FromSubStep() StepResult {
	result.earlyReturn = false
	return result
}

func (result StepResult) Normal() (ctrl.Result, error) {
	if result.err != nil {
		return ctrl.Result{}, result.err
	}
	if result.requeue {
		return ctrl.Result{Requeue: true}, nil
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
		requeue:      true,
		requeueAfter: result,
	}
}

func ResultRequeue() StepResult {
	return StepResult{
		requeue: true,
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

type Step struct {
	// Name is the name of the step
	Name string

	// Step is the function to execute
	Step func(ctx context.Context, req ctrl.Request) StepResult
}

func NewStep(name string, step func(ctx context.Context, req ctrl.Request) StepResult) Step {
	return Step{
		Name: name,
		Step: step,
	}
}

func (stepper *Stepper) Execute(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := stepper.logger

	startedAt := time.Now()

	logger.Info("\n\nStarting stepper execution")

	for _, step := range stepper.steps {
		// time.Sleep(5 * time.Second)
		logger.Info("Executing step", "step", step.Name)

		stepStartedAt := time.Now()
		result := step.Step(ctx, req)
		stepDuration := time.Since(stepStartedAt)

		if result.ShouldReturn() {
			if result.err != nil {
				logger.Error(result.err, "Error in step", "step", step.Name, "stepDuration", stepDuration)
			} else if result.requeue {
				logger.Info("Requeueing after step", "step", step.Name, "stepDuration", stepDuration)
			} else if result.requeueAfter > 0 {
				logger.Info("Requeueing after step", "step", step.Name, "after", result.requeueAfter, "stepDuration", stepDuration)
			}
			return result.Normal()
		}

		logger.Info("Executed step", "step", step.Name, "stepDuration", stepDuration)
	}

	logger.Info("All steps executed successfully", "duration", time.Since(startedAt))
	return ctrl.Result{}, nil
}
