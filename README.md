# Controller Framework (ctrlfwk)

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
- **Observability**: Built-in instrumentation and logging
- **Performance**: Efficient watch caching and resource optimization
- **Testing**: Comprehensive mocking support for unit testing

## Quick Start

### Installation

```bash
go get github.com/u-ctf/controller-fwk
```

### Basic Usage

```go
import ctrlfwk "github.com/u-ctf/controller-fwk"
```

## Core Concepts

### Reconciler

The framework provides a generic `Reconciler` interface that extends controller-runtime's capabilities:

```go
type Reconciler[ControllerResourceType ControllerCustomResource] interface {
    client.Client
    ctrl.Manager

    SetCustomResource(key ControllerResourceType)
    GetCustomResource() ControllerResourceType
    GetCleanCustomResource() ControllerResourceType
}
```

You can easily fulfill this interface in your controller by embedding the necessary structs:

```go
type TestReconciler struct {
	ctrl.Manager
	client.Client
	ctrlfwk.WatchCache
	ctrlfwk.CustomResource[*testv1.Test]

	RuntimeScheme *runtime.Scheme
}

var _ ctrlfwk.Reconciler[*testv1.Test] = &TestReconciler{}
```

As of now, `ctrlfwk.CustomResource[T]` must be re-initialized for each reconciliation to ensure thread safety.

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

## Complete Example

Here's a complete example of a controller that manages a ConfigMap:

### 1. Define Your Custom Resource

```go
// api/v1/test_types.go
type TestSpec struct {
    ConfigMap ConfigMapSpec `json:"configMap,omitempty"`
}

type ConfigMapSpec struct {
    Enabled bool              `json:"enabled,omitempty"`
    Name    string            `json:"name,omitempty"`
    Data    map[string]string `json:"data,omitempty"`
}

type TestStatus struct {
    Conditions      []metav1.Condition `json:"conditions,omitempty"`
    ConfigMapStatus *ConfigMapStatus   `json:"configMapStatus,omitempty"`
}

type ConfigMapStatus struct {
    Name string `json:"name,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Test struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   TestSpec   `json:"spec,omitempty"`
    Status TestStatus `json:"status,omitempty"`
}
```

### 2. Create Resource Definitions

```go
// internal/controller/test/children/configmap.go
func NewConfigMapResource(reconciler ctrlfwk.Reconciler[*testv1.Test]) *ctrlfwk.Resource[*corev1.ConfigMap] {
    cr := reconciler.GetCustomResource()

    return ctrlfwk.NewResource(
        &corev1.ConfigMap{},

        // Skip and delete the ConfigMap if disabled
        ctrlfwk.ResourceSkipAndDeleteOnCondition(&corev1.ConfigMap{}, func() bool {
            return !cr.Spec.ConfigMap.Enabled
        }),

        // Dynamic key generation
        ctrlfwk.ResourceWithKeyFunc(&corev1.ConfigMap{}, func() types.NamespacedName {
            return types.NamespacedName{
                Name:      cr.Spec.ConfigMap.Name,
                Namespace: cr.Namespace,
            }
        }),

        // Resource mutation
        ctrlfwk.ResourceWithMutator(func(resource *corev1.ConfigMap) error {
            resource.Data = make(map[string]string)
            for k, v := range cr.Spec.ConfigMap.Data {
                resource.Data[k] = v
            }
            return controllerutil.SetOwnerReference(cr, resource, reconciler.Scheme())
        }),

        // Readiness condition
        ctrlfwk.ResourceWithReadinessCondition(func(_ *corev1.ConfigMap) bool { 
            return true 
        }),

        // Post-reconciliation hook
        ctrlfwk.ResourceAfterReconcile(&corev1.ConfigMap{}, func(ctx context.Context, resource *corev1.ConfigMap) error {
            return updateConfigMapStatus(ctx, reconciler, resource)
        }),
    )
}

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

### 3. Implement the Controller

```go
// internal/controller/test_controller.go
type TestReconciler struct {
    ctrl.Manager
    client.Client
    ctrlfwk.WatchCache
    ctrlfwk.CustomResource[*testv1.Test]

    RuntimeScheme *runtime.Scheme
}

var _ ctrlfwk.Reconciler[*testv1.Test] = &TestReconciler{}

// Implement ReconcilerWithResources interface
func (r *TestReconciler) GetResources(ctx context.Context, req ctrl.Request) ([]ctrlfwk.GenericResource, error) {
    return []ctrlfwk.GenericResource{
        test_children.NewConfigMapResource(r),
    }, nil
}

// Implement ReconcilerWithDependencies interface (if needed)
func (r *TestReconciler) GetDependencies(ctx context.Context, req ctrl.Request) ([]ctrlfwk.GenericDependency, error) {
    return nil, nil
}

// Main reconciliation logic
func (r *TestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := logf.FromContext(ctx)

    // Create a new reconciler instance for this reconciliation
    newReconciler := &TestReconciler{
        Manager:       r.Manager,
        Client:        r.Client,
        WatchCache:    r.WatchCache,
        RuntimeScheme: r.RuntimeScheme,
    }

    // Create and execute the stepper
    stepper := ctrlfwk.NewStepper(logger,
        ctrlfwk.WithStep(ctrlfwk.NewFindControllerCustomResourceStep(newReconciler)),
        ctrlfwk.WithStep(ctrlfwk.NewResolveDynamicDependenciesStep(newReconciler)),
        ctrlfwk.WithStep(ctrlfwk.NewReconcileResourcesStep(newReconciler)),
        ctrlfwk.WithStep(ctrlfwk.NewEndStep(newReconciler, ctrlfwk.SetReadyCondition(newReconciler))),
    )

    return stepper.Execute(ctx, req)
}

// SetupWithManager configures the controller with the manager
func (r *TestReconciler) SetupWithManager(mgr ctrl.Manager) error {
    controller, err := ctrl.NewControllerManagedBy(mgr).
        For(&testv1.Test{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
        Named("test").
        Build(r)

    if err != nil {
        return err
    }

    r.WatchCache.SetController(controller)
    return nil
}
```

### 4. Bootstrap Your Controller

```go
// cmd/main.go
func main() {
    var metricsAddr string
    var enableLeaderElection bool
    
    flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
    flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
    flag.Parse()

    mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
        Scheme:             scheme,
        MetricsBindAddress: metricsAddr,
        Port:               9443,
        LeaderElection:     enableLeaderElection,
        LeaderElectionID:   "test.example.com",
    })
    if err != nil {
        setupLog.Error(err, "unable to start manager")
        os.Exit(1)
    }

    if err = (&controller.TestReconciler{
        Manager:       mgr,
        Client:        mgr.GetClient(),
        WatchCache:    ctrlfwk.NewWatchCache(),
        RuntimeScheme: mgr.GetScheme(),
    }).SetupWithManager(mgr); err != nil {
        setupLog.Error(err, "unable to create controller", "controller", "Test")
        os.Exit(1)
    }

    setupLog.Info("starting manager")
    if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
        setupLog.Error(err, "problem running manager")
        os.Exit(1)
    }
}
```

## Advanced Features

### Resource Lifecycle Hooks

The framework provides comprehensive lifecycle hooks for resources:

```go
ctrlfwk.NewResource(
    &appsv1.Deployment{},
    ctrlfwk.ResourceBeforeReconcile(&appsv1.Deployment{}, func(ctx context.Context) error {
        // Pre-reconciliation logic
        return nil
    }),
    ctrlfwk.ResourceAfterReconcile(&appsv1.Deployment{}, func(ctx context.Context, resource *appsv1.Deployment) error {
        // Post-reconciliation logic
        return nil
    }),
    ctrlfwk.ResourceOnCreate(&appsv1.Deployment{}, func(ctx context.Context, resource *appsv1.Deployment) error {
        // Resource creation logic, this is only called once.
        return nil
    }),
    ctrlfwk.ResourceOnUpdate(&appsv1.Deployment{}, func(ctx context.Context, resource *appsv1.Deployment) error {
        // Resource update logic, this is only called once per update.
        return nil
    }),
)
```

### Custom Steps

You can create custom steps for specific logic:

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

### Instrumentation

The framework includes built-in instrumentation capabilities:

```go
import "github.com/u-ctf/controller-fwk/instrument"

// Enable tracing for your controller
instrumenter := instrument.NewInstrumenter()
controller := instrument.NewTracerControllerFunc(instrumenter)

// Use in your controller builder
mgr.NewControllerManagedBy(mgr).Build(reconciler)
```

## Testing

The framework provides comprehensive mocking support:

```go
import (
    "github.com/u-ctf/controller-fwk/mocks"
    "go.uber.org/mock/gomock"
)

func TestReconciler(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockReconciler := mocks.NewMockReconciler[*testv1.Test](ctrl)
    
    // Setup expectations
    mockReconciler.EXPECT().GetCustomResource().Return(&testv1.Test{})
    
    // Test your logic
}
```

## Configuration

### RBAC

Don't forget to configure RBAC permissions for your controller:

```go
// +kubebuilder:rbac:groups=test.example.com,resources=tests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=test.example.com,resources=tests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=test.example.com,resources=tests/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## Support

- 📖 [Documentation](https://pkg.go.dev/github.com/u-ctf/controller-fwk)
- 🐛 [Bug Reports](https://github.com/u-ctf/controller-fwk/issues)
- 💡 [Feature Requests](https://github.com/u-ctf/controller-fwk/issues)
- 💬 [Discussions](https://github.com/u-ctf/controller-fwk/discussions)

---

Built with ❤️ by the ctrlfwk team