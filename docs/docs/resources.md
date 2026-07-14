# Resources

Resources are Kubernetes objects your controller creates or updates to realize the desired state of the custom resource.

Examples:

- Deployments
- Services
- ConfigMaps generated from spec data
- Third-party CRs created by your controller

## Reconciler contract

Implement `ReconcilerWithResources[K, C]`:

```go
type WidgetContext = *ctrlfwk.ContextWithData[*myv1.Widget, *WidgetData]
type WidgetResource = ctrlfwk.GenericResource[*myv1.Widget, WidgetContext]

func (reconciler *WidgetReconciler) GetResources(ctx WidgetContext, req ctrl.Request) ([]WidgetResource, error) {
	widget := ctx.GetCustomResource()

	return []WidgetResource{
		ctrlfwk.NewResourceBuilder(ctx, &appsv1.Deployment{}).
			WithKeyFunc(func() types.NamespacedName {
				return types.NamespacedName{
					Name:      widget.Name + "-deployment",
					Namespace: widget.Namespace,
				}
			}).
			WithMutator(func(dep *appsv1.Deployment) error {
				dep.Spec.Replicas = &widget.Spec.Replicas
				dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{"app": widget.Name}}
				dep.Spec.Template.ObjectMeta.Labels = map[string]string{"app": widget.Name}
				dep.Spec.Template.Spec.Containers = []corev1.Container{{
					Name:  "app",
					Image: widget.Spec.Image,
				}}
				return controllerutil.SetOwnerReference(widget, dep, reconciler.RuntimeScheme)
			}).
			WithReadinessCondition(func(dep *appsv1.Deployment) bool {
				if dep.Spec.Replicas == nil {
					return dep.Status.ReadyReplicas > 0
				}
				return dep.Status.ReadyReplicas == *dep.Spec.Replicas
			}).
			WithOutput(ctx.Data.Deployment).
			Build(),
	}, nil
}
```

Use the built-in step to reconcile all returned resources:

```go
ctrlfwk.NewReconcileResourcesStep(fwkCtx, reconciler)
```

## Typed resource builder

`NewResourceBuilder(ctx, &appsv1.Deployment{})` exposes these core methods:

- `WithKey(...)`
- `WithKeyFunc(...)`
- `WithMutator(...)`
- `WithOutput(obj)`
- `WithReadinessCondition(...)`
- `WithSkipAndDeleteOnCondition(...)`
- `WithRequireManualDeletionForFinalize(...)`
- `WithBeforeReconcile(...)`
- `WithAfterReconcile(...)`
- `WithAfterCreate(...)`
- `WithAfterUpdate(...)`
- `WithAfterDelete(...)`
- `WithAfterFinalize(...)`
- `WithUserIdentifier(...)`
- `WithCanBePaused(true)`
- `WithCanBePausedFunc(...)`
- `WithMutatorErrorHandler(...)`

## Mutator error handling

By default, errors from the `WithMutator` callback propagate directly to `CreateOrPatch`. Use `WithMutatorErrorHandler` to transform, log, or suppress them before they cause the step to fail.

One common use case is ignoring transient errors caused by a namespace being terminated:

```go
type pausableResource struct{}

func (p pausableResource) GetPausableResources(context.Context) ... {}

ctrlfwk.NewResourceBuilder(ctx, &corev1.ConfigMap{}).
    WithMutator(func(cm *corev1.ConfigMap) error {
        return controllerutil.SetOwnerReference(ctx.GetCustomResource(), cm, reconciler.Scheme())
    }).
    WithMutatorErrorHandler(func(err error) error {
        if apierrors.HasStatusCause(err, corev1.NamespaceTerminatingCause) {
            return nil // ignore because the resource will be garbage-collected
        }
        return err
    }).
    Build()
```

The `WithMutatorErrorHandler` method is also available on `NewUntypedResourceBuilder`.

## Untyped resource builder

Use `NewUntypedResourceBuilder` for resources without generated Go types.

```go
serviceMonitorGVK := schema.GroupVersionKind{
	Group:   "monitoring.coreos.com",
	Version: "v1",
	Kind:    "ServiceMonitor",
}

resource := ctrlfwk.NewUntypedResourceBuilder(ctx, serviceMonitorGVK).
	WithKeyFunc(func() types.NamespacedName {
		widget := ctx.GetCustomResource()
		return types.NamespacedName{Name: widget.Name + "-metrics", Namespace: widget.Namespace}
	}).
	WithMutator(func(obj *unstructured.Unstructured) error {
		return unstructured.SetNestedMap(obj.Object, map[string]interface{}{
			"matchLabels": map[string]interface{}{"app": ctx.GetCustomResource().Name},
		}, "spec", "selector")
	}).
	WithReadinessCondition(func(obj *unstructured.Unstructured) bool {
		return true
	}).
	Build()
```

## Readiness is explicit

This is the most important detail to get right: resources are only considered ready when `WithReadinessCondition` returns true.

If you do not set a readiness condition, `resource.IsReady(...)` evaluates to false and the resource step will early-return every reconciliation.

For resources that should be considered ready immediately after a successful create-or-patch, make that explicit:

```go
.WithReadinessCondition(func(*corev1.ConfigMap) bool {
	return true
})
```

## Lifecycle semantics

`NewReconcileResourceStep` handles:

- `BeforeReconcile`
- desired object generation
- optional delete-now flow via `WithSkipAndDeleteOnCondition`
- optional dynamic watch setup when the reconciler implements `ReconcilerWithWatcher`
- `CreateOrPatch`
- `AfterCreate` and `AfterUpdate`
- readiness evaluation
- `AfterReconcile`

During custom-resource finalization:

- If `WithRequireManualDeletionForFinalize` is not configured or returns false, the framework relies on garbage collection and only runs `AfterFinalize`.
- If it returns true, the framework explicitly deletes the resource before running `AfterFinalize`.

## Hooks

Use these hook names, which match the current builder API:

- `WithAfterCreate(...)`
- `WithAfterUpdate(...)`
- `WithAfterDelete(...)`
- `WithAfterFinalize(...)`

There are no `WithOnCreate`, `WithOnUpdate`, `WithOnDelete`, or `WithOnFinalize` methods in the current API.

## Conditional resources

Skip creation and delete an existing object when a condition becomes true:

```go
ctrlfwk.NewResourceBuilder(ctx, &corev1.Service{}).
	WithKeyFunc(func() types.NamespacedName {
		widget := ctx.GetCustomResource()
		return types.NamespacedName{Name: widget.Name, Namespace: widget.Namespace}
	}).
	WithMutator(func(svc *corev1.Service) error {
		return controllerutil.SetOwnerReference(ctx.GetCustomResource(), svc, reconciler.RuntimeScheme)
	}).
	WithSkipAndDeleteOnCondition(func() bool {
		return !ctx.GetCustomResource().Spec.ExposeService
	}).
	WithReadinessCondition(func(*corev1.Service) bool { return true }).
	Build()
```

## Pausable resources

`WithCanBePaused(true)` makes the resource respect the framework pause label when reconciling that individual object.

If the live resource already has the pause label, the framework skips mutation for that resource.

## Practical rules

- Use resources only for objects your controller owns.
- Always set owner references in the mutator when garbage collection should apply.
- Always define a readiness condition, even if it is just `return true`.
- Use `WithOutput` when later hooks or steps need the live object.