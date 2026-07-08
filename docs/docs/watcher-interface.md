# Watcher Interface

The watcher integration lets the framework register watches dynamically for resource types discovered during reconciliation.

This removes most manual `.Owns(...)` and `.Watches(...)` calls when you use the framework's built-in resource and dependency steps.

## Core types

```go
type Watcher interface {
	ctrl.Manager

	AddWatchSource(key WatchCacheKey)
	IsWatchingSource(key WatchCacheKey) bool
	GetController() controller.TypedController[reconcile.Request]
}

type WatchCache struct {
	cache      map[WatchCacheKey]bool
	controller controller.TypedController[reconcile.Request]
	ctrl.Manager
}
```

Create it with:

```go
WatchCache: ctrlfwk.NewWatchCache(mgr),
```

## Cache options and scope

The dynamic watch cache is still backed by controller-runtime's manager cache.

That means watch scope is decided in two layers:

- The framework decides which GVKs to watch at runtime.
- The manager cache decides which objects of those GVKs are actually cached and observed.

Use the watch-cache option builders when you want to reduce the cache scope for performance, tenancy, or security reasons.

These options are passed when creating the manager, before `NewWatchCache(mgr)` is used.

```go
cacheOptions := ctrlfwk.NewWatchCacheOptionsBuilder().
	WithScheme(scheme).
	Build()

mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
	Scheme: scheme,
	Cache:  cacheOptions,
})
if err != nil {
	return err
}
```

## Watching a specific namespace

The most common use case is a namespace-scoped operator that should not cache objects cluster-wide.

Use `WithDefaultNamespaces(...)` when most watched objects should be restricted to one or more namespaces:

```go
cacheOptions := ctrlfwk.NewWatchCacheOptionsBuilder().
	WithDefaultNamespaces(map[string]cache.Config{
		"tenant-a": {},
	}).
	Build()
```

Use this when:

- the operator is deployed per tenant or per environment
- dependencies and managed resources are expected to live in the same namespace set
- you want to avoid creating cluster-wide informers for common types like Secrets or ConfigMaps

You can also scope only one object type to a namespace while leaving others broader:

```go
secretCache := ctrlfwk.NewWatchCacheByObjectBuilder().
	WithNamespace("shared-secrets", cache.Config{}).
	Build()

cacheOptions := ctrlfwk.NewWatchCacheOptionsBuilder().
	WithByObjectFor(&corev1.Secret{}, secretCache).
	Build()
```

Use this when a dependency type lives in a dedicated namespace, such as:

- shared credentials in `shared-secrets`
- platform-owned ConfigMaps in `platform-system`
- certificates managed in a central namespace

## Watching objects with specific labels

Use `WithDefaultLabelSelector(...)` to apply a label filter to all cached objects by default:

```go
cacheOptions := ctrlfwk.NewWatchCacheOptionsBuilder().
	WithDefaultLabelSelector(labels.SelectorFromSet(labels.Set{
		"app.kubernetes.io/part-of": "widget-operator",
	})).
	Build()
```

This is useful when the cluster contains many objects of the same type, but your controller should only react to the subset it manages.

For object-specific filtering, combine `WithByObjectFor(...)` with `NewWatchCacheByObjectBuilder()`:

```go
configMapCache := ctrlfwk.NewWatchCacheByObjectBuilder().
	WithLabelSelector(labels.SelectorFromSet(labels.Set{
		"tenant":    "tenant-a",
		"component": "api",
	})).
	Build()

cacheOptions := ctrlfwk.NewWatchCacheOptionsBuilder().
	WithByObjectFor(&corev1.ConfigMap{}, configMapCache).
	Build()
```

Use this when:

- only labelled ConfigMaps or Secrets should trigger reconciles
- the resource type is shared by many controllers in the same cluster
- you already stamp resources with tenancy or ownership labels in your mutators

## Combining namespace and label filters

Per-object options can combine both constraints:

```go
secretCache := ctrlfwk.NewWatchCacheByObjectBuilder().
	WithNamespace("shared-secrets", cache.Config{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			"tenant": "tenant-a",
		}),
	}).
	Build()

cacheOptions := ctrlfwk.NewWatchCacheOptionsBuilder().
	WithByObjectFor(&corev1.Secret{}, secretCache).
	Build()
```

This is the right pattern when the controller should only watch a narrow subset of a high-cardinality type.

Typical examples:

- only Secrets for one tenant in a shared namespace
- only ConfigMaps for one controller component
- only platform resources carrying an opt-in label

## Defaults vs per-object rules

Choose the option level based on how broad the rule is:

- `WithDefaultNamespaces(...)` and `WithDefaultLabelSelector(...)` for cluster-wide defaults across watched types
- `WithByObjectFor(...)` when only one resource type should be narrowed
- `NewWatchCacheByObjectBuilder()` when a specific type needs its own namespace or label rules

In practice, per-object rules are usually safer because they avoid accidentally filtering out unrelated custom resources or dependencies.

## Practical caveats

- Cache filters affect what dynamic watches can observe. If an object is outside the configured namespace or label selector, a runtime watch will not see it.
- Configure the manager cache before constructing the reconciler's `WatchCache`.
- Use label filters only on labels your controller or platform consistently applies.
- When debugging a missing reconcile, verify both the dynamic watch registration and the cache selector scope.

## Reconciler setup

Embed `WatchCache` and implement `ReconcilerWithWatcher[K]`.

```go
type WidgetReconciler struct {
	client.Client
	ctrlfwk.WatchCache
	RuntimeScheme *runtime.Scheme
}

var _ ctrlfwk.ReconcilerWithWatcher[*myv1.Widget] = &WidgetReconciler{}
```

In `SetupWithManager`, use `Build` so you can save the controller back to the watch cache:

```go
func (reconciler *WidgetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrler, err := ctrl.NewControllerManagedBy(mgr).
		For(&myv1.Widget{}).
		Named("widget").
		Build(reconciler)
	if err != nil {
		return err
	}

	reconciler.WatchCache.SetController(ctrler)
	return nil
}
```

## How dynamic watches are added

The framework calls `SetupWatch(...)` from built-in steps:

- `NewReconcileResourceStep` registers watches for managed resources.
- `NewResolveDependencyStep` registers watches for dependencies only when managed-by annotations are enabled.

Watches are deduplicated by `WatchCacheKey`, which is built from the GVK and the watch type.

## Metadata-only watching

Watches are registered against `metav1.PartialObjectMetadata`, not full objects.

That keeps the cache lighter and avoids holding entire resource specs in memory for every watched type.

The default predicate only reacts to meaningful changes:

- update when `resourceVersion` changes
- delete always
- create never

## Dependencies vs resources

Managed resources and dependencies are wired differently:

- Managed resources enqueue owner requests.
- Dependencies enqueue requests by reading the `dependencies.ctrlfwk.com/managed-by` annotation.

That means dependency-triggered reconciles only work when the dependency was configured with `WithAddManagedByAnnotation(true)`.

## Example reconciliation pipeline

```go
func (reconciler *WidgetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	fwkCtx := ctrlfwk.NewContext[*myv1.Widget](ctx, reconciler)

	stepper := ctrlfwk.NewStepperFor[*myv1.Widget](fwkCtx, logger).
		WithStep(ctrlfwk.NewFindControllerCustomResourceStep(fwkCtx, reconciler)).
		WithStep(ctrlfwk.NewResolveDynamicDependenciesStep(fwkCtx, reconciler)).
		WithStep(ctrlfwk.NewReconcileResourcesStep(fwkCtx, reconciler)).
		WithFinalStep(ctrlfwk.NewReadyConditionFinalStep(fwkCtx, reconciler, ctrlfwk.SetReadyConditionFromResult(reconciler))).
		Build()

	return stepper.Execute(fwkCtx, req)
}
```

## Practical rules

- Use `Build`, not `Complete`, when you need to call `WatchCache.SetController`.
- Dependencies only become watch sources when managed-by annotations are enabled.
- Resource watches work automatically once the reconciler implements `ReconcilerWithWatcher[K]`.