# Controller Framework (ctrlfwk)

[![Pipeline](https://woodpecker.yewolf.fr/api/badges/5/status.svg)](https://woodpecker.yewolf.fr/repos/5)
[![Go Version](https://img.shields.io/github/go-mod/go-version/u-ctf/controller-fwk)](https://golang.org/dl/)
[![Go Reference](https://pkg.go.dev/badge/github.com/u-ctf/controller-fwk.svg)](https://pkg.go.dev/github.com/u-ctf/controller-fwk)
[![GitHub release](https://img.shields.io/github/v/release/u-ctf/controller-fwk)](https://github.com/u-ctf/controller-fwk/releases)
[![License](https://img.shields.io/github/license/u-ctf/controller-fwk)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/u-ctf/controller-fwk)](https://goreportcard.com/report/github.com/u-ctf/controller-fwk)

A powerful and extensible framework for building Kubernetes controllers using [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime). This framework provides a structured, step-based approach to controller reconciliation with built-in support for dependencies, resources, and lifecycle hooks.

## Features

- **Step-based Reconciliation**: Organize your controller logic into discrete, manageable steps
- **Resource Management**: Declarative resource creation and management with lifecycle hooks
- **Dependency Resolution**: Handle complex dependencies between Kubernetes resources
- **Type Safety**: Full generic type support for custom resources
- **Minimal Code Changes**: Requires minimal modifications to Kubebuilder-generated controllers
- **Builder Pattern**: Fluent API for creating resources and dependencies
- **Observability**: Built-in instrumentation and logging
- **Performance**: Efficient watch caching and resource optimization
- **Testing**: Comprehensive mocking support for unit testing

## Quick Start

### Installation

```bash
go get github.com/u-ctf/controller-fwk
```

## Migration from Kubebuilder

The framework is designed to work seamlessly with existing Kubebuilder projects with minimal code changes. Below is a step-by-step guide showing the differences between a baseline Kubebuilder controller and one using controller-fwk.

### Before: Standard Kubebuilder Controller

Starting with a standard Kubebuilder-generated controller:

```go
// TestReconciler reconciles a Test object
type TestReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

func (r *TestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    _ = logf.FromContext(ctx)
    
    // TODO(user): your logic here
    
    return ctrl.Result{}, nil
}

func (r *TestReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&testv1.Test{}).
        Named("test").
        Complete(r)
}
```

### After: Controller Framework Enhanced

With minimal changes, transform it to use the controller framework:

```go
// TestReconciler reconciles a Test object
type TestReconciler struct {
    ctrl.Manager                                    // Added
    client.Client
    ctrlfwk.WatchCache                             // Added
    ctrlfwk.CustomResource[*testv1.Test]           // Added
    
    RuntimeScheme *runtime.Scheme                   // Renamed from Scheme
}

var _ ctrlfwk.Reconciler[*testv1.Test] = &TestReconciler{}

func (reconciler *TestReconciler) GetDependencies(ctx context.Context, req ctrl.Request) ([]ctrlfwk.GenericDependency, error) {
    return []ctrlfwk.GenericDependency{
        test_dependencies.NewSecretDependency(reconciler),
    }, nil
}

func (reconciler *TestReconciler) GetResources(ctx context.Context, req ctrl.Request) ([]ctrlfwk.GenericResource, error) {
    return []ctrlfwk.GenericResource{
        test_resources.NewConfigMapResource(reconciler),
    }, nil
}

func (reconciler *TestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := logf.FromContext(ctx)

    // Create a new reconciler instance for thread safety
    newReconciler := &TestReconciler{
        Manager:       reconciler.Manager,
        Client:        reconciler.Client,
        WatchCache:    reconciler.WatchCache,
        RuntimeScheme: reconciler.RuntimeScheme,
    }

    stepper := ctrlfwk.NewStepper(logger,
        ctrlfwk.WithStep(ctrlfwk.NewFindControllerCustomResourceStep(newReconciler)),
        ctrlfwk.WithStep(ctrlfwk.NewResolveDynamicDependenciesStep(newReconciler)),
        ctrlfwk.WithStep(ctrlfwk.NewReconcileResourcesStep(newReconciler)),
        ctrlfwk.WithStep(ctrlfwk.NewEndStep(newReconciler, ctrlfwk.SetReadyCondition(newReconciler))),
    )

    return stepper.Execute(ctx, req)
}

func (reconciler *TestReconciler) SetupWithManager(mgr ctrl.Manager) error {
    ctrler, err := ctrl.NewControllerManagedBy(mgr).
        For(&testv1.Test{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
        Named("test").
        Build(reconciler)                           // Changed from Complete to Build

    reconciler.WatchCache.SetController(ctrler)    // Added

    return err
}
```

### Controller Initialization Changes

In your `main.go`, make minimal adjustments to the controller initialization:

```go
// Before
if err := (&controller.TestReconciler{
    Client: mgr.GetClient(),
    Scheme: mgr.GetScheme(),
}).SetupWithManager(mgr); err != nil {
    setupLog.Error(err, "unable to create controller", "controller", "Test")
    os.Exit(1)
}

// After
if err := (&controller.TestReconciler{
    Manager:       mgr,                             // Added
    Client:        mgr.GetClient(),
    WatchCache:    ctrlfwk.NewWatchCache(),         // Added
    RuntimeScheme: mgr.GetScheme(),                 // Renamed
}).SetupWithManager(mgr); err != nil {
    setupLog.Error(err, "unable to create controller", "controller", "Test")
    os.Exit(1)
}
```

### Custom Resource Types

Your CRD types remain unchanged. The framework works with standard Kubebuilder-generated types:

```go
type TestSpec struct {
    Dependencies TestDependencies `json:"dependencies,omitempty"`
    ConfigMap    ConfigMapSpec    `json:"configMap,omitempty"`
}

type TestStatus struct {
    Conditions      []metav1.Condition `json:"conditions,omitempty"`
    ConfigMapStatus *ConfigMapStatus   `json:"configMapStatus,omitempty"`
}

type Test struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              TestSpec   `json:"spec"`
    Status            TestStatus `json:"status,omitempty"`
}
```

## Core Concepts

### Builder Pattern for Resources

The framework provides a fluent builder pattern for creating resources with comprehensive lifecycle management:

```go
func NewConfigMapResource(reconciler ctrlfwk.Reconciler[*testv1.Test]) *ctrlfwk.Resource[*corev1.ConfigMap] {
    cr := reconciler.GetCustomResource()

    return ctrlfwk.NewResourceBuilder(&corev1.ConfigMap{}).
        WithSkipAndDeleteOnCondition(func() bool {
            return !cr.Spec.ConfigMap.Enabled
        }).
        WithKeyFunc(func() types.NamespacedName {
            return types.NamespacedName{
                Name:      cr.Spec.ConfigMap.Name,
                Namespace: cr.Namespace,
            }
        }).
        WithMutator(func(resource *corev1.ConfigMap) error {
            resource.Data = make(map[string]string)
            for k, v := range cr.Spec.ConfigMap.Data {
                resource.Data[k] = v
            }
            return controllerutil.SetOwnerReference(cr, resource, reconciler.Scheme())
        }).
        WithReadinessCondition(func(_ *corev1.ConfigMap) bool { 
            return true 
        }).
        WithAfterReconcile(func(ctx context.Context, resource *corev1.ConfigMap) error {
            return updateConfigMapStatus(ctx, reconciler, resource)
        }).
        Build()
}
```

### Builder Pattern for Dependencies

Dependencies follow the same builder pattern for external resources your controller needs:

```go
func NewSecretDependency(reconciler ctrlfwk.Reconciler[*testv1.Test]) *ctrlfwk.Dependency[*corev1.Secret] {
    cr := reconciler.GetCustomResource()

    return ctrlfwk.NewDependencyBuilder(&corev1.Secret{}).
        WithName(cr.Spec.Dependencies.Secret.Name).
        WithNamespace(cr.Spec.Dependencies.Secret.Namespace).
        WithOptional(false).
        WithIsReadyFunc(func(secret *corev1.Secret) bool {
            return secret.Data["ready"] != nil
        }).
        WithWaitForReady(true).
        WithAfterReconcile(func(ctx context.Context, resource *corev1.Secret) error {
            return updateSecretStatus(ctx, reconciler, resource)
        }).
        Build()
}
```

### Stepper

The `Stepper` orchestrates the reconciliation process by executing a series of steps:

```go
stepper := ctrlfwk.NewStepper(logger,
    ctrlfwk.WithStep(ctrlfwk.NewFindControllerCustomResourceStep(reconciler)),
    ctrlfwk.WithStep(ctrlfwk.NewResolveDynamicDependenciesStep(reconciler)),
    ctrlfwk.WithStep(ctrlfwk.NewReconcileResourcesStep(reconciler)),
    ctrlfwk.WithStep(ctrlfwk.NewEndStep(reconciler, ctrlfwk.SetReadyCondition(reconciler))),
)

return stepper.Execute(ctx, req)
```

### Resources

Resources are managed declaratively with support for mutation, readiness checks, and lifecycle hooks:

```go
resource := ctrlfwk.NewResource(
    &corev1.ConfigMap{},
    ctrlfwk.ResourceWithKeyFunc(&corev1.ConfigMap{}, func() types.NamespacedName {
        return types.NamespacedName{
            Name:      cr.Spec.ConfigMap.Name,
            Namespace: cr.Namespace,
        }
    }),
    ctrlfwk.ResourceWithMutator(func(resource *corev1.ConfigMap) error {
        resource.Data = cr.Spec.ConfigMap.Data
        return controllerutil.SetOwnerReference(cr, resource, reconciler.Scheme())
    }),
    ctrlfwk.ResourceWithReadinessCondition(func(_ *corev1.ConfigMap) bool { 
        return true 
    }),
)
```

### Stepper-Based Reconciliation

The framework orchestrates reconciliation through a series of well-defined steps:

```go
stepper := ctrlfwk.NewStepper(logger,
    ctrlfwk.WithStep(ctrlfwk.NewFindControllerCustomResourceStep(reconciler)),
    ctrlfwk.WithStep(ctrlfwk.NewResolveDynamicDependenciesStep(reconciler)),
    ctrlfwk.WithStep(ctrlfwk.NewReconcileResourcesStep(reconciler)),
    ctrlfwk.WithStep(ctrlfwk.NewEndStep(reconciler, ctrlfwk.SetReadyCondition(reconciler))),
)

return stepper.Execute(ctx, req)
```

Each step handles a specific aspect of reconciliation:
- **FindControllerCustomResourceStep**: Loads the custom resource
- **ResolveDynamicDependenciesStep**: Ensures external dependencies are ready
- **ReconcileResourcesStep**: Creates and manages owned resources
- **EndStep**: Updates status and finalizes reconciliation

### Resource Lifecycle Management

Resources support comprehensive lifecycle hooks for complex scenarios:

```go
ctrlfwk.NewResourceBuilder(&appsv1.Deployment{}).
    WithBeforeReconcile(func(ctx context.Context) error {
        // Pre-reconciliation logic
        return nil
    }).
    WithAfterReconcile(func(ctx context.Context, resource *appsv1.Deployment) error {
        // Post-reconciliation logic
        return nil
    }).
    WithOnCreate(func(ctx context.Context, resource *appsv1.Deployment) error {
        // Called only when resource is first created
        return nil
    }).
    WithOnUpdate(func(ctx context.Context, resource *appsv1.Deployment) error {
        // Called only when resource is updated
        return nil
    }).
    Build()
```

### Watch Cache

The framework includes an optional `WatchCache` that automatically manages watches for dependencies and resources. This replaces the traditional `.For()` calls you typically see in Kubebuilder operators and provides several advantages:

```go
type TestReconciler struct {
    ctrl.Manager
    client.Client
    ctrlfwk.WatchCache                             // Optional but recommended
    ctrlfwk.CustomResource[*testv1.Test]
    
    RuntimeScheme *runtime.Scheme
}
```

#### Benefits of WatchCache

- **Automatic Watch Management**: No need to manually configure watches for dependencies and child resources
- **Metadata-Only Watching**: Watches only metadata to reduce memory usage and cache size
- **Dynamic Watches**: Automatically adds watches as your controller discovers new resource types
- **Reduced Boilerplate**: Eliminates the need for `.Owns()` and `.Watches()` calls in controller setup

#### How It Works

When you use the built-in steps (`NewResolveDynamicDependenciesStep` and `NewReconcileResourcesStep`), the WatchCache automatically:

1. Detects when your controller interacts with new resource types
2. Sets up efficient metadata-only watches for those resources
3. Triggers reconciliation when those resources change
4. Manages the watch lifecycle automatically

#### Traditional vs WatchCache Approach

```go
// Traditional Kubebuilder approach
func (r *TestReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&testv1.Test{}).
        Owns(&corev1.ConfigMap{}).           // Manual watch setup
        Watches(&corev1.Secret{}, ...).     // Manual watch setup
        Named("test").
        Complete(r)
}

// With WatchCache
func (reconciler *TestReconciler) SetupWithManager(mgr ctrl.Manager) error {
    ctrler, err := ctrl.NewControllerManagedBy(mgr).
        For(&testv1.Test{}).
        Named("test").
        Build(reconciler)                    // WatchCache handles the rest

    reconciler.WatchCache.SetController(ctrler)
    return err
}
```

The WatchCache is particularly beneficial for controllers that manage many different resource types or have dynamic dependencies that change based on configuration.

## Advanced Features

### Custom Steps

You can create custom steps for specific business logic:

```go
customStep := ctrlfwk.NewStep("custom-validation", func(ctx context.Context, logger logr.Logger, req ctrl.Request) ctrlfwk.StepResult {
    // Custom validation logic
    if validationFails {
        return ctrlfwk.ResultInError(errors.New("validation failed"))
    }
    return ctrlfwk.ResultSuccess()
})

stepper := ctrlfwk.NewStepper(logger,
    ctrlfwk.WithStep(ctrlfwk.NewFindControllerCustomResourceStep(reconciler)),
    ctrlfwk.WithStep(customStep), // Insert custom step
    ctrlfwk.WithStep(ctrlfwk.NewReconcileResourcesStep(reconciler)),
    ctrlfwk.WithStep(ctrlfwk.NewEndStep(reconciler, nil)),
)
```

### Status Management

The framework provides utilities for managing custom resource status:

```go
func updateConfigMapStatus(ctx context.Context, reconciler ctrlfwk.Reconciler[*testv1.Test], resource *corev1.ConfigMap) error {
    cr := reconciler.GetCustomResource()
    
    if cr.Status.ConfigMapStatus == nil {
        cr.Status.ConfigMapStatus = &testv1.ConfigMapStatus{}
    }
    
    cr.Status.ConfigMapStatus.Name = resource.Name
    
    meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
        Type:               "ConfigMap",
        Status:             metav1.ConditionTrue,
        ObservedGeneration: cr.Generation,
        Reason:             "UpToDate",
        LastTransitionTime: metav1.Now(),
    })
    
    return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
}
```

### Conditional Resource Management

Resources can be conditionally created or deleted based on spec changes:

```go
ctrlfwk.NewResourceBuilder(&corev1.ConfigMap{}).
    WithSkipAndDeleteOnCondition(func() bool {
        return !cr.Spec.ConfigMap.Enabled  // Delete if disabled
    }).
    WithKeyFunc(func() types.NamespacedName {
        // Handle name changes gracefully
        if !cr.Spec.ConfigMap.Enabled && cr.Status.ConfigMapStatus != nil {
            return types.NamespacedName{
                Name:      cr.Status.ConfigMapStatus.Name,  // Use old name for deletion
                Namespace: cr.Namespace,
            }
        }
        return types.NamespacedName{
            Name:      cr.Spec.ConfigMap.Name,
            Namespace: cr.Namespace,
        }
    }).
    Build()
```

## Testing

The framework doesn't impose any restrictions on testing. You can use standard testing libraries like Ginkgo and Gomega to write unit and integration tests for your controllers. 

Also, as a sidenote, the framework is being tested itself using a sample operator located in the `tests/operator` directory, which you can refer to for examples. This operator is meant to use every feature of the framework and serves as a practical demonstration of its capabilities. (And also makes for a good testbed for e2e tests!)

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## Support

- Documentation: https://pkg.go.dev/github.com/u-ctf/controller-fwk
- Bug Reports: https://github.com/u-ctf/controller-fwk/issues
- Feature Requests: https://github.com/u-ctf/controller-fwk/issues
- Discussions: https://github.com/u-ctf/controller-fwk/discussions

Built with care by the U-CTF team