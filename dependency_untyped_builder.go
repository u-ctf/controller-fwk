package ctrlfwk

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UntypedDependencyBuilder provides a fluent builder pattern for creating dependencies
// on resources that are not known at compile time or don't have Go type definitions.
//
// This builder is useful when working with:
//   - Custom Resource Definitions (CRDs) not included in your Go code
//   - Third-party resources without Go client types
//   - Dynamic resource types determined at runtime
//   - Resources from different API groups or versions
//
// Type parameters:
//   - CustomResourceType: The custom resource that owns this dependency
//   - ContextType: The context type containing the custom resource and additional data
//
// The builder works with unstructured.Unstructured objects, which can represent
// any Kubernetes resource dynamically.
//
// Example:
//
//	// Depend on a custom resource defined by a CRD
//	gvk := schema.GroupVersionKind{
//		Group:   "example.com",
//		Version: "v1",
//		Kind:    "Database",
//	}
//	dep := NewUntypedDependencyBuilder(ctx, gvk).
//		WithName("my-database").
//		WithNamespace("default").
//		WithIsReadyFunc(func(obj *unstructured.Unstructured) bool {
//			// Check custom readiness condition
//			status, found, _ := unstructured.NestedString(obj.Object, "status", "phase")
//			return found && status == "Ready"
//		}).
//		Build()
type UntypedDependencyBuilder[CustomResourceType client.Object, ContextType Context[CustomResourceType]] struct {
	inner *DependencyBuilder[CustomResourceType, ContextType, *unstructured.Unstructured]
	gvk   schema.GroupVersionKind
}

// NewUntypedDependencyBuilder creates a new UntypedDependencyBuilder for constructing
// dependencies on Kubernetes resources that don't have compile-time Go types.
//
// This is particularly useful for:
//   - Custom Resource Definitions (CRDs) defined in YAML but not in Go
//   - Third-party resources from operators you don't control
//   - Resources from different API versions or groups
//   - Dynamic resource types determined at runtime
//
// Parameters:
//   - ctx: The context containing the custom resource and additional data
//   - gvk: GroupVersionKind specifying the exact resource type to depend on
//
// The GroupVersionKind must exactly match the target resource's type information.
// The dependency will be resolved as an unstructured.Unstructured object.
//
// Example:
//
//	// Depend on a Prometheus ServiceMonitor resource
//	gvk := schema.GroupVersionKind{
//		Group:   "monitoring.coreos.com",
//		Version: "v1",
//		Kind:    "ServiceMonitor",
//	}
//	dep := NewUntypedDependencyBuilder(ctx, gvk).
//		WithName("my-app-metrics").
//		WithNamespace(ctx.GetCustomResource().Namespace).
//		WithOptional(true). // Don't fail if Prometheus operator not installed
//		Build()
func NewUntypedDependencyBuilder[CustomResourceType client.Object, ContextType Context[CustomResourceType]](ctx ContextType, gvk schema.GroupVersionKind) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	return &UntypedDependencyBuilder[CustomResourceType, ContextType]{
		inner: NewDependencyBuilder(ctx, &unstructured.Unstructured{}),
		gvk:   gvk,
	}
}

// Build constructs and returns the final UntypedDependency instance with all configured options.
//
// This method finalizes the builder pattern and creates an untyped dependency that can be
// used in reconciliation steps. The returned dependency contains all the configuration
// specified through the builder methods and will work with unstructured.Unstructured objects.
//
// The dependency must be used with appropriate reconciliation steps (such as
// ResolveDynamicDependenciesStep) to actually perform the dependency resolution.
//
// Returns a configured UntypedDependency instance ready for use in reconciliation.
func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) Build() *UntypedDependency[CustomResourceType, ContextType] {
	return &UntypedDependency[CustomResourceType, ContextType]{
		Dependency: b.inner.Build(),
		gvk:        b.gvk,
	}
}

// WithAfterReconcile registers a hook function to execute after successful dependency resolution.
//
// This function is called with the resolved dependency as an unstructured.Unstructured object
// and can be used for post-processing, validation, or updating your custom resource's status.
// If the function returns an error, the reconciliation will fail.
//
// Working with unstructured objects requires using helper functions to access nested fields:
//
// Example:
//
//	.WithAfterReconcile(func(ctx MyContext, obj *unstructured.Unstructured) error {
//		// Extract status from custom resource
//		status, found, err := unstructured.NestedString(obj.Object, "status", "endpoint")
//		if err != nil {
//			return err
//		}
//		if found {
//			ctx.Data.DatabaseEndpoint = status
//		}
//		return nil
//	})
func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithAfterReconcile(f func(ctx ContextType, resource *unstructured.Unstructured) error) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithAfterReconcile(f)
	return b
}

// WithBeforeReconcile registers a hook function to execute before dependency resolution.
//
// This function is called before attempting to resolve the untyped dependency and can be used
// for setup tasks, validation, or logging. If the function returns an error,
// dependency resolution will be aborted.
//
// Common use cases:
//   - Validating that the target CRD is installed
//   - Setting up authentication for third-party resources
//   - Logging dependency resolution attempts for debugging
//
// Example:
//
//	.WithBeforeReconcile(func(ctx MyContext) error {
//		logger := ctx.GetLogger()
//		logger.Info("Resolving third-party resource dependency", "gvk", gvk)
//		return nil
//	})
func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithBeforeReconcile(f func(ctx ContextType) error) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithBeforeReconcile(f)
	return b
}

// WithIsReadyFunc defines custom logic to determine if the untyped dependency is ready for use.
//
// The provided function is called with the resolved dependency as an unstructured.Unstructured
// object and should return true if the dependency meets your readiness criteria.
//
// Working with unstructured objects requires using helper functions to access nested fields.
// Common patterns include checking status conditions, phase fields, or custom markers.
//
// Example:
//
//	.WithIsReadyFunc(func(obj *unstructured.Unstructured) bool {
//		// Check if a custom resource has reached "Ready" state
//		phase, found, _ := unstructured.NestedString(obj.Object, "status", "phase")
//		if !found {
//			return false
//		}
//		return phase == "Ready" || phase == "Running"
//	})
func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithIsReadyFunc(f func(obj *unstructured.Unstructured) bool) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithIsReadyFunc(f)
	return b
}

// WithName specifies the name of the Kubernetes resource to depend on.
//
// This is the metadata.name field of the target resource. The name is required
// for dependency resolution and should match exactly with the existing resource.
//
// Example:
//
//	.WithName("my-database-instance") // Look for resource named "my-database-instance"
func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithName(name string) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithName(name)
	return b
}

// WithNamespace specifies the namespace where the dependency resource is located.
//
// This is the metadata.namespace field of the target resource. If not specified,
// the dependency will be looked up in the same namespace as the custom resource.
//
// For cluster-scoped resources, this field is ignored.
//
// Example:
//
//	.WithNamespace("monitoring") // Look in monitoring namespace
//	.WithNamespace(ctx.GetCustomResource().Namespace) // Same namespace as custom resource
func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithNamespace(namespace string) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithNamespace(namespace)
	return b
}

// WithOptional configures whether this dependency is required for reconciliation.
//
// When set to true, the dependency resolution will continue even if this dependency
// is missing or not ready. When false (default), missing or unready dependencies
// will cause reconciliation to requeue and wait.
//
// This is particularly useful for untyped dependencies on optional operators or CRDs:
//   - Prometheus monitoring resources (when Prometheus operator is optional)
//   - Service mesh resources (when Istio/Linkerd might not be installed)
//   - Third-party integrations that enhance but don't break functionality
//
// Example:
//
//	.WithOptional(true) // Don't fail if the CRD isn't installed
func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithOptional(optional bool) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithOptional(optional)
	return b
}

// WithOutput specifies where to store the resolved untyped dependency resource.
//
// The provided unstructured.Unstructured object will be populated with the dependency's
// data after successful resolution. This allows other parts of your reconciliation
// logic to access the dependency's current state.
//
// The output object should be a field in your context's data structure to ensure
// it's accessible throughout the reconciliation process.
//
// Example:
//
//	type MyContextData struct {
//		DatabaseInstance *unstructured.Unstructured
//	}
//
//	dep := NewUntypedDependencyBuilder(ctx, gvk).
//		WithName("my-db").
//		WithOutput(ctx.Data.DatabaseInstance). // Store resolved resource here
//		Build()
func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithOutput(obj *unstructured.Unstructured) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithOutput(obj)
	return b
}

// WithReadinessCondition is an alias for WithIsReadyFunc that defines custom readiness logic.
//
// This method provides the same functionality as WithIsReadyFunc but with a more
// descriptive name for untyped dependencies. Use whichever method name feels more
// natural in your context.
//
// See WithIsReadyFunc for detailed documentation and examples of working with
// unstructured.Unstructured objects.
func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithReadinessCondition(f func(obj *unstructured.Unstructured) bool) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithReadinessCondition(f)
	return b
}

// WithUserIdentifier assigns a custom identifier for this untyped dependency.
//
// This identifier is used for logging, debugging, and distinguishing between
// multiple untyped dependencies. If not provided, a default identifier will be
// generated based on the GroupVersionKind and resource name.
//
// This is especially useful for untyped dependencies since the resource types
// may not be immediately obvious from logs.
//
// Example:
//
//	.WithUserIdentifier("prometheus-servicemonitor") // Clear name for logs and debugging
func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithUserIdentifier(identifier string) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithUserIdentifier(identifier)
	return b
}

// WithWaitForReady determines whether reconciliation should wait for this dependency
// to become ready before proceeding.
//
// When true (recommended), reconciliation will requeue if the dependency exists but
// is not yet ready according to the readiness function. When false, the dependency
// is only checked for existence.
//
// This is particularly important for untyped dependencies on custom resources that
// may have complex initialization or external dependencies.
//
// Example:
//
//	.WithWaitForReady(true) // Wait for custom resource to be fully initialized
func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithWaitForReady(waitForReady bool) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithWaitForReady(waitForReady)
	return b
}

// WithAddManagedByAnnotation controls whether to add a "managed-by" annotation
// to the untyped dependency resource.
//
// When enabled, this adds metadata to help identify which controller is managing
// or depending on this resource. This is especially useful for untyped dependencies
// since the relationship between controllers and third-party resources may not be obvious.
//
// The annotation typically follows the format:
//
//	"app.kubernetes.io/managed-by": "<controller-name>"
//
// Example:
//
//	.WithAddManagedByAnnotation(true) // Mark dependency relationship for debugging
func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithAddManagedByAnnotation(add bool) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithAddManagedByAnnotation(add)
	return b
}
