# Context

The framework context is the data carrier for a single reconciliation.

It holds:

- The active Go `context.Context`
- The mutable custom resource instance used by steps
- The clean custom resource snapshot used for status patch generation
- Optional typed reconciliation data shared between steps

## Base API

```go
type Context[K client.Object] interface {
	context.Context
	ImplementsCustomResource[K]
}

type ContextWithData[K client.Object, D any] struct {
	Context[K]
	Data D
}
```

Constructor functions:

```go
fwkCtx := ctrlfwk.NewContext[*myv1.Widget](ctx, reconciler)
fwkCtxWithData := ctrlfwk.NewContextWithData(ctx, reconciler, &WidgetData{})
```

## Custom resource access

The context exposes three important methods:

- `GetCustomResource()` returns the mutable in-memory object for the current reconciliation.
- `GetCleanCustomResource()` returns a clean copy suitable for merge patches.
- `SetCustomResource(obj)` stores both the mutable and clean copies.

`NewFindControllerCustomResourceStep` loads the CR and calls `SetCustomResource` for you.

```go
stepper := ctrlfwk.NewStepperFor[*myv1.Widget](fwkCtx, logger).
	WithStep(ctrlfwk.NewFindControllerCustomResourceStep(fwkCtx, reconciler)).
	Build()
```

## Using typed data

`ContextWithData` is a concrete struct with a `Data` field. The data is not accessed through a method.

```go
type WidgetData struct {
	ConfigSecret *corev1.Secret
	Deployment   *appsv1.Deployment
	Validated    bool
}

type WidgetContext = *ctrlfwk.ContextWithData[*myv1.Widget, *WidgetData]

func (reconciler *WidgetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	fwkCtx := ctrlfwk.NewContextWithData(ctx, reconciler, &WidgetData{})

	stepper := ctrlfwk.NewStepperFor[*myv1.Widget](fwkCtx, logger).
		WithStep(ctrlfwk.NewFindControllerCustomResourceStep(fwkCtx, reconciler)).
		Build()

	return stepper.Execute(fwkCtx, req)
}
```

Then, inside steps:

```go
func validationStep() ctrlfwk.Step[*myv1.Widget, WidgetContext] {
	return ctrlfwk.NewStep(
		"validate",
		func(ctx WidgetContext, logger logr.Logger, req ctrl.Request) ctrlfwk.StepResult {
			ctx.Data.Validated = true
			logger.Info("validation complete")
			return ctrlfwk.ResultSuccess()
		},
	)
}
```

## Status patching

Patch through `PatchCustomResourceStatus(ctx, reconciler)` after updating the object returned by `GetCustomResource()`.

```go
func setPhase(ctx ctrlfwk.Context[*myv1.Widget], reconciler *WidgetReconciler, phase string) error {
	widget := ctx.GetCustomResource()
	widget.Status.Phase = phase

	return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
}
```

The helper computes a merge patch from the clean copy to the updated object, then refreshes the context with the patched object.

## Ready condition helpers

Two helpers exist for the common `Ready` condition:

- `SetReadyCondition(reconciler)` returns `func(obj K) (bool, error)` and is intended for `NewEndStep`.
- `SetReadyConditionFromResult(reconciler)` returns `func(obj K, lastStepName string, lastStepResult StepResult) (bool, error)` and is intended for `NewReadyConditionFinalStep`.

The second helper is the better default because it can mark the resource as not ready on requeues, early returns, and errors.

## Recommended aliases

Aliases make the generic signatures manageable:

```go
type WidgetData struct {
	ConfigSecret *corev1.Secret
	Deployment   *appsv1.Deployment
}

type WidgetContext = *ctrlfwk.ContextWithData[*myv1.Widget, *WidgetData]
type WidgetDependency = ctrlfwk.GenericDependency[*myv1.Widget, WidgetContext]
type WidgetResource = ctrlfwk.GenericResource[*myv1.Widget, WidgetContext]
type WidgetStep = ctrlfwk.Step[*myv1.Widget, WidgetContext]
```

## Practical rules

- Create one framework context per reconciliation.
- Do not bypass the framework by storing a separate CR pointer elsewhere.
- Call `NewFindControllerCustomResourceStep` before steps that read or patch the custom resource.
- Store shared state in `ctx.Data`, not in package globals.