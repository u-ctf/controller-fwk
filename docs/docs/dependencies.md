# Dependencies

Dependencies are existing Kubernetes objects your controller consumes.

Examples:

- A Secret containing credentials
- A ConfigMap with shared configuration
- A third-party CR managed by another operator

Dependencies are not created by the framework. They are fetched, optionally annotated, optionally watched, and optionally waited on.

## Reconciler contract

Implement `ReconcilerWithDependencies[K, C]`:

```go
type WidgetContext = *ctrlfwk.ContextWithData[*myv1.Widget, *WidgetData]
type WidgetDependency = ctrlfwk.GenericDependency[*myv1.Widget, WidgetContext]

func (reconciler *WidgetReconciler) GetDependencies(ctx WidgetContext, req ctrl.Request) ([]WidgetDependency, error) {
	return []WidgetDependency{
		ctrlfwk.NewDependencyBuilder(ctx, &corev1.Secret{}).
			WithName(ctx.GetCustomResource().Spec.SecretRef.Name).
			WithNamespace(ctx.GetCustomResource().Namespace).
			WithWaitForReady(true).
			WithIsReadyFunc(func(secret *corev1.Secret) bool {
				return len(secret.Data["username"]) > 0 && len(secret.Data["password"]) > 0
			}).
			WithOutput(ctx.Data.ConfigSecret).
			Build(),
	}, nil
}
```

Use the built-in step to resolve everything returned from `GetDependencies`:

```go
ctrlfwk.NewResolveDynamicDependenciesStep(fwkCtx, reconciler)
```

## Typed dependency builder

`NewDependencyBuilder(ctx, &corev1.Secret{})` exposes these core methods:

- `WithKey(...)`
- `WithKeyFunc(...)`
- `WithName(...)`
- `WithNamespace(...)`
- `WithWaitForReady(true)`
- `WithIsReadyFunc(...)`
- `WithReadinessCondition(...)`
- `WithOptional(true)`
- `WithOutput(obj)`
- `WithBeforeReconcile(...)`
- `WithAfterReconcile(...)`
- `WithUserIdentifier(...)`
- `WithAddManagedByAnnotation(true)`

## Untyped dependency builder

Use `NewUntypedDependencyBuilder` for resources without generated Go types.

```go
gvk := schema.GroupVersionKind{
	Group:   "monitoring.coreos.com",
	Version: "v1",
	Kind:    "ServiceMonitor",
}

dep := ctrlfwk.NewUntypedDependencyBuilder(ctx, gvk).
	WithName("widget-metrics").
	WithNamespace(ctx.GetCustomResource().Namespace).
	WithOptional(true).
	WithIsReadyFunc(func(obj *unstructured.Unstructured) bool {
		phase, found, _ := unstructured.NestedString(obj.Object, "status", "phase")
		return found && phase == "Ready"
	}).
	Build()
```

## Resolution semantics

`NewResolveDependencyStep` behaves like this:

- If the object is missing and the dependency is optional, the step succeeds.
- If the object is missing and the dependency is required, the step requeues after 30 seconds.
- If `WithWaitForReady(true)` is set and readiness returns false, the step requeues after 30 seconds.
- `AfterReconcile` runs even when the dependency step returns early.

## Managed-by annotations and dynamic watches

Dependency watches are only set up when both conditions are true:

1. The reconciler implements `ReconcilerWithWatcher[K]`.
2. The dependency enables `WithAddManagedByAnnotation(true)`.

When enabled, the framework:

- Adds the `dependencies.ctrlfwk.com/managed-by` annotation to the dependency.
- Registers a metadata watch for that dependency type.
- Removes the managed-by reference during finalization.

If you do not enable managed-by annotations, the framework still resolves the dependency, but it will not automatically reconnect dependency events back to your controller.

## Hooks

Dependencies have two lifecycle hooks:

- `WithBeforeReconcile(func(ctx C) error)`
- `WithAfterReconcile(func(ctx C, resource T) error)`

Typical uses:

- Compute derived data and store it in `ctx.Data`
- Update status on missing or invalid dependencies
- Emit events through a reconciler that also implements `ReconcilerWithEventRecorder`

## Example with status update

```go
ctrlfwk.NewDependencyBuilder(ctx, &corev1.Secret{}).
	WithName("database-credentials").
	WithNamespace(ctx.GetCustomResource().Namespace).
	WithWaitForReady(true).
	WithIsReadyFunc(func(secret *corev1.Secret) bool {
		return len(secret.Data["dsn"]) > 0
	}).
	WithAfterReconcile(func(ctx WidgetContext, secret *corev1.Secret) error {
		if secret == nil {
			return nil
		}

		ctx.Data.ConfigSecret = secret
		return nil
	}).
	Build()
```

## Practical rules

- Use dependencies for objects you do not own.
- Use resources for objects you create and reconcile.
- Only enable `WithWaitForReady(true)` when you also provide a meaningful readiness function.
- Only rely on dependency-triggered reconciles when `WithAddManagedByAnnotation(true)` is set.