package ctrlfwk_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	ctrlfwk "github.com/u-ctf/controller-fwk"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type stepperTestContext struct {
	context.Context
	ctrlfwk.CustomResource[*corev1.ConfigMap]
}

func newStepperTestContext() *stepperTestContext {
	ctx := &stepperTestContext{Context: context.Background()}
	ctx.SetCustomResource(&corev1.ConfigMap{})
	return ctx
}

func TestStepper_WithFinalStep_RunsAfterSuccess(t *testing.T) {
	ctx := newStepperTestContext()

	called := false
	gotLastStepName := ""
	gotSuccess := false

	stepper := ctrlfwk.NewStepperFor(ctx, logr.Discard()).
		WithStep(ctrlfwk.NewStep("prepare", func(ctx *stepperTestContext, logger logr.Logger, req ctrl.Request) ctrlfwk.StepResult {
			return ctrlfwk.ResultSuccess()
		})).
		WithFinalStep(ctrlfwk.NewFinalStep("finalize", func(ctx *stepperTestContext, logger logr.Logger, req ctrl.Request, lastStepName string, lastStepResult ctrlfwk.StepResult) error {
			called = true
			gotLastStepName = lastStepName
			gotSuccess = lastStepResult.IsSuccess()
			return nil
		})).
		Build()

	result, err := stepper.Execute(ctx, ctrl.Request{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != (ctrl.Result{}) {
		t.Fatalf("expected empty result, got %#v", result)
	}
	if !called {
		t.Fatal("expected final step to be called")
	}
	if gotLastStepName != "prepare" {
		t.Fatalf("expected last step name to be %q, got %q", "prepare", gotLastStepName)
	}
	if !gotSuccess {
		t.Fatal("expected final step to observe a successful last step")
	}
}

func TestStepper_WithFinalStep_RunsAfterError(t *testing.T) {
	ctx := newStepperTestContext()
	stepErr := errors.New("boom")

	called := false

	stepper := ctrlfwk.NewStepperFor(ctx, logr.Discard()).
		WithStep(ctrlfwk.NewStep("resolve dependency", func(ctx *stepperTestContext, logger logr.Logger, req ctrl.Request) ctrlfwk.StepResult {
			return ctrlfwk.ResultInError(stepErr)
		})).
		WithFinalStep(ctrlfwk.NewFinalStep("finalize", func(ctx *stepperTestContext, logger logr.Logger, req ctrl.Request, lastStepName string, lastStepResult ctrlfwk.StepResult) error {
			called = true
			if lastStepName != "resolve dependency" {
				return errors.New("unexpected last step name")
			}
			if !errors.Is(lastStepResult.Error(), stepErr) {
				return errors.New("unexpected last step error")
			}
			return nil
		})).
		Build()

	_, err := stepper.Execute(ctx, ctrl.Request{})
	if !called {
		t.Fatal("expected final step to be called")
	}
	if !errors.Is(err, stepErr) {
		t.Fatalf("expected returned error to contain step error, got %v", err)
	}
}

func TestStepper_WithFinalStep_JoinsFinalStepError(t *testing.T) {
	ctx := newStepperTestContext()
	stepErr := errors.New("step failed")
	finalErr := errors.New("final step failed")

	stepper := ctrlfwk.NewStepperFor(ctx, logr.Discard()).
		WithStep(ctrlfwk.NewStep("reconcile resource", func(ctx *stepperTestContext, logger logr.Logger, req ctrl.Request) ctrlfwk.StepResult {
			return ctrlfwk.ResultInError(stepErr)
		})).
		WithFinalStep(ctrlfwk.NewFinalStep("finalize", func(ctx *stepperTestContext, logger logr.Logger, req ctrl.Request, lastStepName string, lastStepResult ctrlfwk.StepResult) error {
			return finalErr
		})).
		Build()

	_, err := stepper.Execute(ctx, ctrl.Request{})
	if !errors.Is(err, stepErr) {
		t.Fatalf("expected returned error to contain step error, got %v", err)
	}
	if !errors.Is(err, finalErr) {
		t.Fatalf("expected returned error to contain final step error, got %v", err)
	}
}

func TestNewReadyConditionFinalStep_SkipsWhenControllerResourceWasNotLoaded(t *testing.T) {
	ctx := &stepperTestContext{Context: context.Background()}
	called := false

	finalStep := ctrlfwk.NewReadyConditionFinalStep[*corev1.ConfigMap](
		ctx,
		ctrlfwk.Reconciler[*corev1.ConfigMap](nil),
		func(obj *corev1.ConfigMap, lastStepName string, lastStepResult ctrlfwk.StepResult) (bool, error) {
			called = true
			return true, nil
		},
	)

	err := finalStep.Step(ctx, logr.Discard(), ctrl.Request{}, ctrlfwk.StepFindControllerCustomResource, ctrlfwk.ResultEarlyReturn())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if called {
		t.Fatal("expected ready-condition callback to be skipped when the controller resource was not loaded")
	}
}
