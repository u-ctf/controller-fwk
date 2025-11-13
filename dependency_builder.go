package ctrlfwk

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DependencyBuilder provides a fluent builder pattern for creating Dependency instances.
// It enables declarative construction of dependencies with type safety and method chaining.
//
// Type parameters:
//   - CustomResourceType: The custom resource that owns this dependency
//   - ContextType: The context type containing the custom resource and additional data
//   - DependencyType: The Kubernetes resource type this dependency represents
type DependencyBuilder[CustomResourceType client.Object, ContextType Context[CustomResourceType], DependencyType client.Object] struct {
	dependency *Dependency[CustomResourceType, ContextType, DependencyType]
}

// NewDependencyBuilder creates a new DependencyBuilder for constructing dependencies
// on external Kubernetes resources.
//
// Dependencies are external resources that your custom resource depends on to function
// correctly. They are resolved during the dependency resolution step of reconciliation.
//
// Parameters:
//   - ctx: The context containing the custom resource and additional data
//   - _: A zero-value instance used for type inference (e.g., &corev1.Secret{})
//
// The dependency will only be resolved when used with ResolveDependencyStep or
// ResolveDynamicDependenciesStep during reconciliation.
//
// Common use cases:
//   - Waiting for secrets or configmaps to be available
//   - Ensuring other custom resources are ready
//   - Validating external service availability
//
// Example:
//
//	// Wait for a secret to contain required data
//	dep := NewDependencyBuilder(ctx, &corev1.Secret{}).
//		WithName("database-credentials").
//		WithNamespace(ctx.GetCustomResource().Namespace).
//		WithWaitForReady(true).
//		WithIsReadyFunc(func(secret *corev1.Secret) bool {
//			return secret.Data["username"] != nil && secret.Data["password"] != nil
//		}).
//		WithOutput(ctx.Data.DatabaseSecret).
//		Build()
func NewDependencyBuilder[
	CustomResourceType client.Object,
	ContextType Context[CustomResourceType],
	DependencyType client.Object,
](
	ctx ContextType,
	_ DependencyType,
) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	return &DependencyBuilder[CustomResourceType, ContextType, DependencyType]{
		dependency: &Dependency[CustomResourceType, ContextType, DependencyType]{},
	}
}

// WithOutput specifies where to store the resolved dependency resource.
//
// The provided object will be populated with the dependency's data after successful
// resolution. This allows other parts of your reconciliation logic to access the
// dependency's current state.
//
// The output object should be a field in your context's data structure to ensure
// it's accessible throughout the reconciliation process.
//
// Note: Setting an output is optional. If you only need to verify the dependency
// exists and is ready, you can use WithAfterReconcile for post-resolution actions
// without storing the full resource.
//
// Example:
//
//	type MyContextData struct {
//		DatabaseSecret *corev1.Secret
//	}
//
//	dep := NewDependencyBuilder(ctx, &corev1.Secret{}).
//		WithName("database-creds").
//		WithOutput(ctx.Data.DatabaseSecret). // Store resolved secret here
//		Build()
//
// This allows you to access ctx.Data.DatabaseSecret later in your reconciliation logic.
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithOutput(obj DependencyType) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.output = obj
	return b
}

// WithIsReadyFunc defines custom logic to determine if the dependency is ready for use.
//
// The provided function is called with the resolved dependency resource and should
// return true if the dependency meets your readiness criteria, false otherwise.
//
// If no readiness function is provided, the dependency is considered ready as soon
// as it exists in the cluster.
//
// Common readiness patterns:
//   - Checking for specific data fields in secrets/configmaps
//   - Validating status conditions on custom resources
//   - Ensuring required labels or annotations are present
//
// Example:
//
//	.WithIsReadyFunc(func(secret *corev1.Secret) bool {
//		// Secret is ready when it contains required database credentials
//		return secret.Data["host"] != nil &&
//		       secret.Data["username"] != nil &&
//		       secret.Data["password"] != nil
//	})
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithIsReadyFunc(f func(obj DependencyType) bool) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.isReadyF = f
	return b
}

// WithOptional configures whether this dependency is required for reconciliation.
//
// When set to true, the dependency resolution will continue even if this dependency
// is missing or not ready. When false (default), missing or unready dependencies
// will cause reconciliation to requeue and wait.
//
// Use optional dependencies for:
//   - Feature flags or optional configurations
//   - Dependencies that provide enhanced functionality but aren't required
//   - Resources that may not exist in all environments
//
// Example:
//
//	// Optional feature configuration
//	dep := NewDependencyBuilder(ctx, &corev1.ConfigMap{}).
//		WithName("optional-features").
//		WithOptional(true). // Don't block reconciliation if missing
//		Build()
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithOptional(optional bool) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.isOptional = optional
	return b
}

// WithName specifies the name of the Kubernetes resource to depend on.
//
// This is the metadata.name field of the target resource. The name is required
// for dependency resolution and should match exactly with the existing resource.
//
// Example:
//
//	.WithName("database-credentials") // Look for resource named "database-credentials"
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithName(name string) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.name = name
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
//	.WithNamespace("kube-system") // Look in kube-system namespace
//	.WithNamespace(ctx.GetCustomResource().Namespace) // Same namespace as custom resource
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithNamespace(namespace string) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.namespace = namespace
	return b
}

// WithWaitForReady determines whether reconciliation should wait for this dependency
// to become ready before proceeding.
//
// When true (recommended), reconciliation will requeue if the dependency exists but
// is not yet ready according to the readiness function. When false, the dependency
// is only checked for existence.
//
// This is particularly useful for:
//   - Resources that need initialization time
//   - External services that may be temporarily unavailable
//   - Custom resources with complex readiness conditions
//
// Example:
//
//	.WithWaitForReady(true) // Wait for dependency to be ready, don't just check existence
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithWaitForReady(waitForReady bool) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.waitForReady = waitForReady
	return b
}

// WithUserIdentifier assigns a custom identifier for this dependency.
//
// This identifier is used for logging, debugging, and distinguishing between
// multiple dependencies of the same type. If not provided, a default identifier
// will be generated based on the resource type and name.
//
// Useful for:
//   - Debugging dependency resolution issues
//   - Providing meaningful names in logs
//   - Distinguishing between similar dependencies
//
// Example:
//
//	.WithUserIdentifier("database-connection-secret") // Custom name for logs
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithUserIdentifier(identifier string) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.userIdentifier = identifier
	return b
}

// WithBeforeReconcile registers a hook function to execute before dependency resolution.
//
// This function is called before attempting to resolve the dependency and can be used
// for setup tasks, validation, or logging. If the function returns an error,
// dependency resolution will be aborted.
//
// Common use cases:
//   - Validating preconditions
//   - Setting up authentication
//   - Logging dependency resolution attempts
//
// Example:
//
//	.WithBeforeReconcile(func(ctx MyContext) error {
//		logger := ctx.GetLogger()
//		logger.Info("Resolving database secret dependency")
//		return nil
//	})
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithBeforeReconcile(f func(ctx ContextType) error) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.beforeReconcileF = f
	return b
}

// WithAfterReconcile registers a hook function to execute after successful dependency resolution.
//
// This function is called with the resolved dependency resource and can be used for
// post-processing, validation, or updating your custom resource's status. If the
// function returns an error, the reconciliation will fail.
//
// The resource parameter contains the current state of the resolved dependency.
//
// Common use cases:
//   - Extracting and caching dependency data
//   - Updating custom resource status with dependency information
//   - Triggering additional processing based on dependency state
//
// Example:
//
//	.WithAfterReconcile(func(ctx MyContext, secret *corev1.Secret) error {
//		// Cache database connection string from secret
//		connStr := string(secret.Data["connection-string"])
//		ctx.Data.DatabaseConnectionString = connStr
//		return updateCustomResourceStatus(ctx, "DatabaseReady", "Connected")
//	})
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithAfterReconcile(f func(ctx ContextType, resource DependencyType) error) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.afterReconcileF = f
	return b
}

// WithReadinessCondition is an alias for WithIsReadyFunc that defines custom readiness logic.
//
// This method provides the same functionality as WithIsReadyFunc but with a more
// descriptive name. Use whichever method name feels more natural in your context.
//
// See WithIsReadyFunc for detailed documentation and examples.
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithReadinessCondition(f func(obj DependencyType) bool) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.isReadyF = f
	return b
}

// WithAddManagedByAnnotation controls whether to add a "managed-by" annotation
// to the dependency resource.
//
// When enabled, this adds metadata to help identify which controller is managing
// or depending on this resource. This is useful for:
//   - Debugging resource relationships
//   - Resource lifecycle management
//   - Avoiding conflicts between controllers
//
// Also, when enabled, if the reconciler has a Watcher configured, it will automatically
// watch for changes to this dependency resource and trigger reconciliations accordingly.
//
// This is not enabled by default to avoid unnecessary annotations on resources.
//
// The annotation typically follows the format:
//
//	"dependencies.ctrlfwk.com/managed-by": "[{'name':'<controller-name>','namespace':'<controller-namespace>','gvk':{'group':'<group>','version':'<version>','kind':'<kind>'}}]"
//
// Example:
//
//	.WithAddManagedByAnnotation(true) // Mark this resource as managed by our controller
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithAddManagedByAnnotation(add bool) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.addManagedBy = add
	return b
}

// Build constructs and returns the final Dependency instance with all configured options.
//
// This method finalizes the builder pattern and creates the dependency that can be
// used in reconciliation steps. The returned dependency contains all the configuration
// specified through the builder methods.
//
// The dependency must be used with appropriate reconciliation steps (such as
// ResolveDynamicDependenciesStep) to actually perform the dependency resolution.
//
// Returns a configured Dependency instance ready for use in reconciliation.
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) Build() *Dependency[CustomResourceType, ContextType, DependencyType] {
	return b.dependency
}
