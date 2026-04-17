package ctrlfwk

import (
	"errors"
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
type Stepper[K client.Object, C Context[K]] struct {
	logger logr.Logger
	steps  []Step[K, C]
	final  *FinalStep[K, C]
}

type StepperBuilder[K client.Object, C Context[K]] struct {
	logger logr.Logger
	steps  []Step[K, C]
	final  *FinalStep[K, C]
}

// NewStepperFor creates a builder for a stepper bound to the provided logger.
func NewStepperFor[K client.Object, C Context[K]](ctx C, logger logr.Logger) *StepperBuilder[K, C] {
	return &StepperBuilder[K, C]{
		logger: logger,
		steps:  []Step[K, C]{},
	}
}

// WithStep appends a normal reconciliation step to the execution chain.
func (s *StepperBuilder[K, C]) WithStep(step Step[K, C]) *StepperBuilder[K, C] {
	s.steps = append(s.steps, step)
	return s
}

// WithFinalStep registers a final step that always runs after the last executed step.
//
// The final step is invoked on success, early return, requeue, and error paths. It receives
// the name and result of the last executed step so it can derive status updates such as
// readiness conditions.
//
// If multiple final steps are configured, the last one wins.
func (s *StepperBuilder[K, C]) WithFinalStep(step FinalStep[K, C]) *StepperBuilder[K, C] {
	s.final = &step
	return s
}

// Build constructs the Stepper from the configured steps and final step.
func (s *StepperBuilder[K, C]) Build() *Stepper[K, C] {
	return &Stepper[K, C]{
		logger: s.logger,
		steps:  s.steps,
		final:  s.final,
	}
}

type StepResult struct {
	earlyReturn  bool
	err          error
	requeueAfter time.Duration
}

// Error returns the underlying step error, if any.
func (result StepResult) Error() error {
	return result.err
}

// RequeueAfter returns the requested requeue duration, if any.
func (result StepResult) RequeueAfter() time.Duration {
	return result.requeueAfter
}

// IsEarlyReturn reports whether the step requested an early return without error.
func (result StepResult) IsEarlyReturn() bool {
	return result.earlyReturn
}

// IsSuccess reports whether the step completed without error, requeue, or early return.
func (result StepResult) IsSuccess() bool {
	return !result.ShouldReturn()
}

// ShouldReturn reports whether reconciliation should stop after this step.
func (result StepResult) ShouldReturn() bool {
	return result.err != nil || result.requeueAfter > 0 || result.earlyReturn
}

// FromSubStep clears early return semantics when bubbling a nested step result upward.
func (result StepResult) FromSubStep() StepResult {
	result.earlyReturn = false
	return result
}

// Normal converts the step result into a controller-runtime reconcile result.
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

// FinalStep is an always-run callback executed after the last normal step attempt.
//
// It is intended for cross-cutting cleanup or status updates that must happen even when a
// previous step returned an error or requested a requeue.
type FinalStep[K client.Object, C Context[K]] struct {
	// Name is the name used for logging the final step execution.
	Name string

	// Step executes after the last normal step and receives that step's name and result.
	Step func(ctx C, logger logr.Logger, req ctrl.Request, lastStepName string, lastStepResult StepResult) error
}

// NewFinalStep creates an always-run final step.
func NewFinalStep[K client.Object, C Context[K]](
	name string,
	step func(ctx C, logger logr.Logger, req ctrl.Request, lastStepName string, lastStepResult StepResult) error,
) FinalStep[K, C] {
	return FinalStep[K, C]{
		Name: name,
		Step: step,
	}
}

type Step[K client.Object, C Context[K]] struct {
	// Name is the name of the step
	Name string

	// Step is the function to execute
	Step func(ctx C, logger logr.Logger, req ctrl.Request) StepResult
}

func NewStep[K client.Object, C Context[K]](name string, step func(ctx C, logger logr.Logger, req ctrl.Request) StepResult) Step[K, C] {
	return Step[K, C]{
		Name: name,
		Step: step,
	}
}

func mergeFinalStepResult(result StepResult, finalErr error) StepResult {
	if finalErr == nil {
		return result
	}

	if result.err != nil {
		result.err = errors.Join(result.err, finalErr)
		return result
	}

	return ResultInError(finalErr)
}

func (stepper *Stepper[K, C]) executeFinalStep(ctx C, req ctrl.Request, lastStepName string, lastStepResult StepResult) error {
	if stepper.final == nil || stepper.final.Step == nil {
		return nil
	}

	stepStartedAt := time.Now()
	err := stepper.final.Step(ctx, stepper.logger, req, lastStepName, lastStepResult)
	stepDuration := time.Since(stepStartedAt)

	if err != nil {
		stepper.logger.Error(err, "Error in final step", "step", stepper.final.Name, "stepDuration", stepDuration, "lastStep", lastStepName)
		return err
	}

	stepper.logger.Info("Executed final step", "step", stepper.final.Name, "stepDuration", stepDuration, "lastStep", lastStepName)
	return nil
}

func (stepper *Stepper[K, C]) Execute(ctx C, req ctrl.Request) (ctrl.Result, error) {
	logger := stepper.logger

	startedAt := time.Now()
	lastStepName := ""
	lastStepResult := ResultSuccess()

	logger.Info("Starting stepper execution")

	for _, step := range stepper.steps {
		stepStartedAt := time.Now()
		result := step.Step(ctx, logger, req)
		stepDuration := time.Since(stepStartedAt)
		lastStepName = step.Name
		lastStepResult = result

		if result.ShouldReturn() {
			resultToReturn := result

			if result.err != nil {
				if IsFinalizing(ctx.GetCustomResource()) && apierrors.IsNotFound(result.err) {
					logger.Info("Resource not found during finalization, ignoring error", "step", step.Name, "stepDuration", stepDuration)
					resultToReturn = ResultRequeueIn(1 * time.Second)
				} else {
					logger.Error(result.err, "Error in step", "step", step.Name, "stepDuration", stepDuration)
				}
			} else if result.requeueAfter > 0 {
				logger.Info("Requeueing after step", "step", step.Name, "after", result.requeueAfter, "stepDuration", stepDuration)
			} else {
				logger.Info("Early return after step", "step", step.Name, "stepDuration", stepDuration)
			}

			finalErr := stepper.executeFinalStep(ctx, req, lastStepName, lastStepResult)
			return mergeFinalStepResult(resultToReturn, finalErr).Normal()
		}

		logger.Info("Executed step", "step", step.Name, "stepDuration", stepDuration)
	}

	if err := stepper.executeFinalStep(ctx, req, lastStepName, lastStepResult); err != nil {
		return ResultInError(err).Normal()
	}

	logger.Info("All steps executed successfully", "duration", time.Since(startedAt))
	return ctrl.Result{}, nil
}
