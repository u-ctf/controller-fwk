---
slug: /
title: Overview
---

# Controller Framework (ctrlfwk)

Controller Framework is a generic layer on top of controller-runtime for writing Kubernetes controllers as an ordered sequence of typed reconciliation steps.

The current API centers on four building blocks:

- `Context[K]` and `ContextWithData[K, D]` for storing the custom resource and any reconciliation-scoped data.
- `Step[K, C]` and `FinalStep[K, C]` for composing reconciliation logic.
- `NewStepperFor(ctx, logger)` for building the execution pipeline.
- Builder-based dependency and resource definitions for external objects and managed objects.

## Reconciliation model

The typical flow is:

1. Create a framework context.
2. Build a stepper with normal steps.
3. Register a final step for status updates that must run on success, error, requeue, or early return.
4. Execute the stepper with the framework context.

```go
func (r *WidgetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	fwkCtx := ctrlfwk.NewContext[*widgetv1.Widget](ctx, r)

	stepper := ctrlfwk.NewStepperFor[*widgetv1.Widget](fwkCtx, logger).
		WithStep(ctrlfwk.NewFindControllerCustomResourceStep(fwkCtx, r)).
		WithStep(ctrlfwk.NewResolveDynamicDependenciesStep(fwkCtx, r)).
		WithStep(ctrlfwk.NewReconcileResourcesStep(fwkCtx, r)).
		WithFinalStep(ctrlfwk.NewReadyConditionFinalStep(fwkCtx, r, ctrlfwk.SetReadyConditionFromResult(r))).
		Build()

	return stepper.Execute(fwkCtx, req)
}
```

## Important behavior

- `NewEndStep` is a normal step. It only runs if all prior steps succeeded.
- `NewReadyConditionFinalStep` is the preferred way to keep the Ready condition in sync on all exit paths.
- `PatchCustomResourceStatus(ctx, reconciler)` patches from the clean copy stored in the framework context.
- `SetReadyCondition` and `SetReadyConditionFromResult` assume your CR status has a `Conditions []metav1.Condition` field.

## Interface contracts

At minimum, a reconciler must satisfy `ctrlfwk.Reconciler[K]`:

```go
type WidgetReconciler struct {
	client.Client
	RuntimeScheme *runtime.Scheme
}

var _ ctrlfwk.Reconciler[*widgetv1.Widget] = &WidgetReconciler{}

func (WidgetReconciler) For(*widgetv1.Widget) {}
```

Additional interfaces enable more framework features:

- `ReconcilerWithDependencies[K, C]` exposes `GetDependencies`.
- `ReconcilerWithResources[K, C]` exposes `GetResources`.
- `ReconcilerWithWatcher[K]` enables dynamic watch registration.

## Documentation map

- [Getting Started](./getting-started.md) for the current minimal controller pattern.
- [Context](./context.md) for custom-resource state and typed reconciliation data.
- [Dependencies](./dependencies.md) for external object resolution.
- [Resources](./resources.md) for managed object reconciliation.
- [Watcher Interface](./watcher-interface.md) for dynamic watches.
- [Instrumentation](./instrumentation.md) for tracing and log propagation.