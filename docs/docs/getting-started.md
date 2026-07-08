# Getting Started

This guide shows the current integration pattern for `github.com/u-ctf/controller-fwk`.

## Prerequisites

- Go 1.25+
- controller-runtime based controller
- A custom resource with a status field containing `Conditions []metav1.Condition` if you want to use the built-in Ready-condition helpers

Install the framework:

```bash
go get github.com/u-ctf/controller-fwk
```

## 1. Define the reconciler contract

Your reconciler must satisfy `ctrlfwk.Reconciler[K]`.

```go
type WebAppReconciler struct {
	client.Client
	RuntimeScheme *runtime.Scheme
}

var _ ctrlfwk.Reconciler[*appsv1alpha1.WebApp] = &WebAppReconciler{}

func (WebAppReconciler) For(*appsv1alpha1.WebApp) {}
```

If you plan to use dependencies or managed resources, also implement:

- `ReconcilerWithDependencies[*WebApp, WebAppContext]`
- `ReconcilerWithResources[*WebApp, WebAppContext]`

If you want dynamic watches, also embed `ctrlfwk.WatchCache` and implement `ReconcilerWithWatcher[*WebApp]`.

## 2. Create a framework context

Use `NewContext` when you only need access to the custom resource.

```go
fwkCtx := ctrlfwk.NewContext[*appsv1alpha1.WebApp](ctx, reconciler)
```

Use `NewContextWithData` when steps, dependencies, and resources need to share typed data.

```go
type WebAppData struct {
	ConfigSecret *corev1.Secret
	Deployment   *appsv1.Deployment
}

type WebAppContext = *ctrlfwk.ContextWithData[*appsv1alpha1.WebApp, *WebAppData]

fwkCtx := ctrlfwk.NewContextWithData(ctx, reconciler, &WebAppData{})
```

## 3. Build the stepper

The current API uses `NewStepperFor(ctx, logger)` plus fluent builder methods.

```go
func (reconciler *WebAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	fwkCtx := ctrlfwk.NewContext[*appsv1alpha1.WebApp](ctx, reconciler)

	stepper := ctrlfwk.NewStepperFor[*appsv1alpha1.WebApp](fwkCtx, logger).
		WithStep(ctrlfwk.NewFindControllerCustomResourceStep(fwkCtx, reconciler)).
		WithStep(validationStep()).
		WithFinalStep(ctrlfwk.NewReadyConditionFinalStep(fwkCtx, reconciler, ctrlfwk.SetReadyConditionFromResult(reconciler))).
		Build()

	return stepper.Execute(fwkCtx, req)
}
```

Use `WithFinalStep` for status updates that must run on all exit paths.

`NewEndStep` still exists, but it only runs if all previous steps returned success.

## 4. Write custom steps

Custom steps are plain values of type `ctrlfwk.Step[K, C]`.

```go
func validationStep() ctrlfwk.Step[*appsv1alpha1.WebApp, ctrlfwk.Context[*appsv1alpha1.WebApp]] {
	return ctrlfwk.NewStep(
		"validate-webapp",
		func(ctx ctrlfwk.Context[*appsv1alpha1.WebApp], logger logr.Logger, req ctrl.Request) ctrlfwk.StepResult {
			webapp := ctx.GetCustomResource()

			if webapp.Spec.Image == "" {
				return ctrlfwk.ResultEarlyReturn()
			}

			logger.Info("validated webapp", "image", webapp.Spec.Image)
			return ctrlfwk.ResultSuccess()
		},
	)
}
```

Available result helpers:

- `ResultSuccess()`
- `ResultEarlyReturn()`
- `ResultRequeueIn(duration)`
- `ResultInError(err)`

## 5. Patch status through the framework context

The framework keeps both a clean copy and a mutable copy of the custom resource in the context. Update the mutable object, then patch through the helper.

```go
func markNotReady(ctx ctrlfwk.Context[*appsv1alpha1.WebApp], reconciler *WebAppReconciler, reason string) error {
	webapp := ctx.GetCustomResource()
	meta.SetStatusCondition(&webapp.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            "validation failed",
		ObservedGeneration: webapp.GetGeneration(),
		LastTransitionTime: metav1.Now(),
	})

	return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
}
```

If you use `SetReadyConditionFromResult`, the final step can manage the generic Ready condition for you.

## 6. Add framework resource and dependency steps

Once your reconciler implements the relevant interfaces, the usual sequence is:

```go
stepper := ctrlfwk.NewStepperFor[*appsv1alpha1.WebApp](fwkCtx, logger).
	WithStep(ctrlfwk.NewFindControllerCustomResourceStep(fwkCtx, reconciler)).
	WithStep(ctrlfwk.NewResolveDynamicDependenciesStep(fwkCtx, reconciler)).
	WithStep(ctrlfwk.NewReconcileResourcesStep(fwkCtx, reconciler)).
	WithFinalStep(ctrlfwk.NewReadyConditionFinalStep(fwkCtx, reconciler, ctrlfwk.SetReadyConditionFromResult(reconciler))).
	Build()
```

## 7. Setup with manager

Without dynamic watches, the normal controller-runtime setup works:

```go
func (reconciler *WebAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.WebApp{}).
		Named("webapp").
		Complete(reconciler)
}
```

If you use `WatchCache`, build the controller instead of calling `Complete`, then set the controller on the cache:

```go
func (reconciler *WebAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrler, err := ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.WebApp{}).
		Named("webapp").
		Build(reconciler)
	if err != nil {
		return err
	}

	reconciler.WatchCache.SetController(ctrler)
	return nil
}
```

## Next steps

- [Context](./context.md) for typed reconciliation state.
- [Dependencies](./dependencies.md) for external objects.
- [Resources](./resources.md) for managed objects.
- [Watcher Interface](./watcher-interface.md) for dynamic watches.