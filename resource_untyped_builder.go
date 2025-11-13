package ctrlfwk

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UntypedResourceBuilder provides a fluent builder pattern for creating managed resources
// that are not known at compile time or don't have Go type definitions.
//
// This builder is useful when your custom resource needs to manage:
//   - Custom Resource Definitions (CRDs) not included in your Go code
//   - Third-party resources from other operators
//   - Resources from different API groups or versions
//   - Dynamic resource types determined at runtime
//
// Type parameters:
//   - CustomResource: The custom resource that owns and manages this untyped resource
//   - ContextType: The context type containing the custom resource and additional data
//
// The builder works with unstructured.Unstructured objects, which can represent
// any Kubernetes resource dynamically.
//
// Unlike untyped dependencies (which are external resources you consume), untyped
// resources are resources that your controller creates and manages as part of
// implementing your custom resource's desired state.
//
// Common use cases:
//   - Managing CRDs defined in YAML but not in Go
//   - Creating third-party operator resources (e.g., Prometheus ServiceMonitors)
//   - Managing resources from different API versions
//   - Implementing operators that work with dynamic resource types
//
// Example:
//
//	// Create a Prometheus ServiceMonitor for your application
//	gvk := schema.GroupVersionKind{
//		Group:   "monitoring.coreos.com",
//		Version: "v1",
//		Kind:    "ServiceMonitor",
//	}
//	serviceMonitor := NewUntypedResourceBuilder(ctx, gvk).
//		WithKeyFunc(func() types.NamespacedName {
//			return types.NamespacedName{
//				Name:      ctx.GetCustomResource().Name + "-metrics",
//				Namespace: ctx.GetCustomResource().Namespace,
//			}
//		}).
//		WithMutator(func(obj *unstructured.Unstructured) error {
//			// Configure ServiceMonitor using unstructured helpers
//			return unstructured.SetNestedField(obj.Object, "app", "spec", "selector", "matchLabels", "app")
//		}).
//		WithReadinessCondition(func(obj *unstructured.Unstructured) bool {
//			// Check if ServiceMonitor is being scraped
//			status, found, _ := unstructured.NestedString(obj.Object, "status", "phase")
//			return found && status == "Active"
//		}).
//		Build()
type UntypedResourceBuilder[CustomResource client.Object, ContextType Context[CustomResource]] struct {
	inner *ResourceBuilder[CustomResource, ContextType, *unstructured.Unstructured]
	gvk   schema.GroupVersionKind
}

// NewUntypedResourceBuilder creates a new UntypedResourceBuilder for constructing
// managed Kubernetes resources that don't have compile-time Go types.
//
// This is particularly useful for:
//   - Managing Custom Resource Definitions (CRDs) defined in YAML but not in Go
//   - Creating third-party resources from operators you don't control
//   - Working with resources from different API versions or groups
//   - Implementing dynamic resource management based on configuration
//
// Parameters:
//   - ctx: The context containing the custom resource and additional data
//   - gvk: GroupVersionKind specifying the exact resource type to manage
//
// The GroupVersionKind must exactly match the target resource's type information.
// The resource will be managed as an unstructured.Unstructured object with
// owner references to your custom resource for proper garbage collection.
//
// Key differences from NewUntypedDependencyBuilder:
//   - Resources are CREATED and MANAGED by your controller
//   - Dependencies are CONSUMED by your controller (external resources)
//   - Resources have owner references to your custom resource
//   - Resources are deleted when your custom resource is deleted
//
// Example:
//
//	// Manage a Grafana Dashboard resource
//	gvk := schema.GroupVersionKind{
//		Group:   "grafana.integreatly.org",
//		Version: "v1beta1",
//		Kind:    "GrafanaDashboard",
//	}
//	dashboard := NewUntypedResourceBuilder(ctx, gvk).
//		WithKeyFunc(func() types.NamespacedName {
//			return types.NamespacedName{
//				Name:      ctx.GetCustomResource().Name + "-dashboard",
//				Namespace: ctx.GetCustomResource().Namespace,
//			}
//		}).
//		WithMutator(func(obj *unstructured.Unstructured) error {
//			// Configure dashboard JSON content
//			dashboardJSON := generateDashboard(ctx.GetCustomResource())
//			return unstructured.SetNestedField(obj.Object, dashboardJSON, "spec", "json")
//		}).
//		Build()
func NewUntypedResourceBuilder[CustomResource client.Object, ContextType Context[CustomResource]](ctx ContextType, gvk schema.GroupVersionKind) *UntypedResourceBuilder[CustomResource, ContextType] {
	return &UntypedResourceBuilder[CustomResource, ContextType]{
		inner: NewResourceBuilder(ctx, &unstructured.Unstructured{}),
		gvk:   gvk,
	}
}

// Build constructs and returns the final UntypedResource instance with all configured options.
//
// This method finalizes the builder pattern and creates an untyped resource that can be
// used in reconciliation steps. The returned resource contains all the configuration
// specified through the builder methods and will work with unstructured.Unstructured objects.
//
// The resource must be used with appropriate reconciliation steps (such as
// ReconcileResourcesStep) to actually perform the resource management operations.
//
// Validation:
//   - At least one of WithKey or WithKeyFunc must be called before Build()
//   - WithMutator is typically required for meaningful resource management
//
// Returns a configured UntypedResource instance ready for use in reconciliation.
func (b *UntypedResourceBuilder[CustomResource, ContextType]) Build() *UntypedResource[CustomResource, ContextType] {
	return &UntypedResource[CustomResource, ContextType]{
		Resource: b.inner.Build(),
		gvk:      b.gvk,
	}
}

// WithAfterCreate registers a hook function that executes only when an untyped resource is newly created.
//
// This function is called specifically when a resource is created for the first time,
// not when it's updated. It's useful for one-time initialization tasks specific to
// untyped resources, such as setting up external integrations or logging creation events.
//
// The function receives the newly created unstructured resource, including any fields
// populated by Kubernetes during creation.
//
// Working with unstructured objects requires using helper functions to access nested fields.
//
// Example:
//
//	.WithAfterCreate(func(ctx MyContext, obj *unstructured.Unstructured) error {
//		// Log creation of custom resource
//		logger := ctx.GetLogger()
//		name, _, _ := unstructured.NestedString(obj.Object, "metadata", "name")
//		logger.Info("Custom resource created", "name", name, "gvk", gvk)
//
//		// Register with external monitoring system
//		return ctx.RegisterWithExternalSystem(obj)
//	})
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithAfterCreate(f func(ctx ContextType, resource *unstructured.Unstructured) error) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithAfterCreate(f)
	return b
}

// WithAfterDelete registers a hook function that executes after an untyped resource is deleted.
//
// This function is called when a resource has been successfully deleted from the cluster,
// either due to a delete condition being met or during custom resource finalization.
// It can be used for cleanup tasks specific to untyped resources.
//
// The function receives the resource object as it existed just before deletion.
// Working with unstructured objects requires using helper functions to access nested fields.
//
// Example:
//
//	.WithAfterDelete(func(ctx MyContext, obj *unstructured.Unstructured) error {
//		// Clean up external references
//		name, _, _ := unstructured.NestedString(obj.Object, "metadata", "name")
//		logger := ctx.GetLogger()
//		logger.Info("Cleaning up external resources", "resource", name)
//
//		// Remove from external monitoring system
//		return ctx.UnregisterFromExternalSystem(name)
//	})
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithAfterDelete(f func(ctx ContextType, resource *unstructured.Unstructured) error) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithAfterDelete(f)
	return b
}

// WithAfterFinalize registers a hook function that executes during custom resource finalization
// for untyped resources that require special cleanup handling.
//
// This function is called when the custom resource is being deleted and the untyped resource
// needs to be cleaned up as part of the finalization process. It's particularly important
// for untyped resources that may have external dependencies or special deletion requirements.
//
// Working with unstructured objects requires using helper functions to access nested fields.
//
// Example:
//
//	.WithAfterFinalize(func(ctx MyContext, obj *unstructured.Unstructured) error {
//		// Graceful cleanup for third-party resources
//		logger := ctx.GetLogger()
//
//		// Check if resource has special cleanup requirements
//		cleanupMode, found, _ := unstructured.NestedString(obj.Object, "spec", "cleanupMode")
//		if found && cleanupMode == "preserve" {
//			logger.Info("Preserving resource during finalization")
//			return nil
//		}
//
//		// Perform graceful shutdown
//		return ctx.GracefullyShutdownExternalResource(obj)
//	})
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithAfterFinalize(f func(ctx ContextType, resource *unstructured.Unstructured) error) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithAfterFinalize(f)
	return b
}

// WithAfterReconcile registers a hook function to execute after successful untyped resource reconciliation.
//
// This function is called after the untyped resource has been successfully created, updated,
// or verified. It receives the current resource state as an unstructured.Unstructured object
// and can be used for post-processing, status updates, or triggering additional operations.
//
// Working with unstructured objects requires using helper functions to access nested fields.
//
// Example:
//
//	.WithAfterReconcile(func(ctx MyContext, obj *unstructured.Unstructured) error {
//		cr := ctx.GetCustomResource()
//
//		// Extract status information from the untyped resource
//		status, found, err := unstructured.NestedString(obj.Object, "status", "phase")
//		if err != nil {
//			return err
//		}
//
//		// Update custom resource status
//		if found {
//			cr.Status.ExternalResourcePhase = status
//			meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
//				Type:   "ExternalResourceReady",
//				Status: metav1.ConditionTrue,
//				Reason: "ResourceReconciled",
//			})
//		}
//
//		return ctx.PatchStatus()
//	})
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithAfterReconcile(f func(ctx ContextType, resource *unstructured.Unstructured) error) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithAfterReconcile(f)
	return b
}

// WithAfterUpdate registers a hook function that executes only when an untyped resource is updated.
//
// This function is called specifically when an existing untyped resource is modified,
// not when it's initially created. It's useful for tracking changes to third-party
// resources and responding to configuration updates.
//
// Working with unstructured objects requires using helper functions to access nested fields.
//
// Example:
//
//	.WithAfterUpdate(func(ctx MyContext, obj *unstructured.Unstructured) error {
//		logger := ctx.GetLogger()
//
//		// Log the update with resource details
//		name, _, _ := unstructured.NestedString(obj.Object, "metadata", "name")
//		generation, _, _ := unstructured.NestedInt64(obj.Object, "metadata", "generation")
//
//		logger.Info("External resource updated",
//			"name", name,
//			"generation", generation,
//			"gvk", gvk)
//
//		// Trigger external system update notification
//		return ctx.NotifyExternalSystemOfUpdate(obj)
//	})
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithAfterUpdate(f func(ctx ContextType, resource *unstructured.Unstructured) error) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithAfterUpdate(f)
	return b
}

// WithBeforeReconcile registers a hook function to execute before untyped resource reconciliation.
//
// This function is called before any resource operations (create, update, or delete)
// are performed on the untyped resource. It's particularly useful for validating
// that the target CRD is installed and for preparing the environment.
//
// Common use cases for untyped resources:
//   - Validating that the target CRD exists in the cluster
//   - Checking operator availability (e.g., Prometheus, Grafana operators)
//   - Setting up authentication for third-party APIs
//   - Performing environment-specific preparations
//
// Example:
//
//	.WithBeforeReconcile(func(ctx MyContext) error {
//		logger := ctx.GetLogger()
//		client := ctx.GetClient()
//
//		// Verify that the target CRD exists
//		crdName := fmt.Sprintf("%ss.%s", strings.ToLower(gvk.Kind), gvk.Group)
//		crd := &apiextensionsv1.CustomResourceDefinition{}
//		err := client.Get(ctx, types.NamespacedName{Name: crdName}, crd)
//		if err != nil {
//			return fmt.Errorf("required CRD %s not found: %w", crdName, err)
//		}
//
//		logger.Info("Proceeding with untyped resource reconciliation", "gvk", gvk)
//		return nil
//	})
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithBeforeReconcile(f func(ctx ContextType) error) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithBeforeReconcile(f)
	return b
}

// WithKey specifies a static NamespacedName for the untyped resource.
//
// This is useful when the resource name and namespace are known at build time
// and don't need to be computed dynamically based on the custom resource state.
//
// For dynamic naming based on custom resource properties, use WithKeyFunc instead.
//
// Example:
//
//	.WithKey(types.NamespacedName{
//		Name:      "monitoring-config",
//		Namespace: "monitoring",
//	}) // Static name for the monitoring resource
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithKey(name types.NamespacedName) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithKey(name)
	return b
}

// WithKeyFunc specifies a function that dynamically determines the untyped resource's NamespacedName.
//
// This function is called during reconciliation to determine where the resource should
// be created or found. For untyped resources, this is particularly important as the
// naming often needs to be coordinated with external operators or systems.
//
// Example:
//
//	.WithKeyFunc(func() types.NamespacedName {
//		cr := ctx.GetCustomResource()
//		return types.NamespacedName{
//			// Name follows third-party operator conventions
//			Name:      fmt.Sprintf("%s-%s-monitor", cr.Name, cr.Spec.Component),
//			Namespace: cr.Namespace,
//		}
//	})
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithKeyFunc(f func() types.NamespacedName) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithKeyFunc(f)
	return b
}

// WithMutator specifies the function that configures the untyped resource's desired state.
//
// The mutator function receives an unstructured.Unstructured object and should configure
// all necessary fields using the unstructured helper functions. This is where you define
// how your custom resource's spec translates into the target third-party resource configuration.
//
// Working with unstructured objects requires using helper functions like:
//   - unstructured.SetNestedField() to set individual values
//   - unstructured.SetNestedSlice() to set arrays
//   - unstructured.SetNestedMap() to set objects
//
// The mutator should be idempotent and handle both CREATE and UPDATE operations.
//
// Example:
//
//	.WithMutator(func(obj *unstructured.Unstructured) error {
//		cr := ctx.GetCustomResource()
//
//		// Set basic metadata
//		obj.SetName(cr.Name + "-servicemonitor")
//		obj.SetNamespace(cr.Namespace)
//
//		// Configure ServiceMonitor spec using unstructured helpers
//		err := unstructured.SetNestedMap(obj.Object, map[string]interface{}{
//			"app": cr.Name,
//		}, "spec", "selector", "matchLabels")
//		if err != nil {
//			return err
//		}
//
//		// Set endpoint configuration
//		endpoints := []interface{}{
//			map[string]interface{}{
//				"port": "metrics",
//				"path": "/metrics",
//			},
//		}
//		err = unstructured.SetNestedSlice(obj.Object, endpoints, "spec", "endpoints")
//		if err != nil {
//			return err
//		}
//
//		// Set owner reference for garbage collection
//		return controllerutil.SetOwnerReference(cr, obj, ctx.GetScheme())
//	})
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithMutator(f Mutator[*unstructured.Unstructured]) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithMutator(f)
	return b
}

// WithOutput specifies where to store the reconciled untyped resource after successful operations.
//
// The provided unstructured.Unstructured object will be populated with the resource's
// current state from the cluster after reconciliation completes. This is particularly
// useful for untyped resources where you need to extract status information or other
// runtime values generated by third-party operators.
//
// Example:
//
//	type MyContextData struct {
//		ServiceMonitor *unstructured.Unstructured
//	}
//
//	resource := NewUntypedResourceBuilder(ctx, gvk).
//		// ... other configuration ...
//		WithOutput(ctx.Data.ServiceMonitor). // Store reconciled resource here
//		Build()
//
//	// Later, access the reconciled resource
//	if ctx.Data.ServiceMonitor != nil {
//		status, found, _ := unstructured.NestedString(
//			ctx.Data.ServiceMonitor.Object, "status", "phase")
//	}
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithOutput(obj *unstructured.Unstructured) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithOutput(obj)
	return b
}

// WithReadinessCondition defines custom logic to determine when the untyped resource is ready.
//
// The provided function is called with the current unstructured resource state and should
// return true if the resource has reached the desired operational state. This is particularly
// important for untyped resources that may have complex initialization processes managed
// by third-party operators.
//
// Working with unstructured objects requires using helper functions to access nested fields.
//
// Example:
//
//	.WithReadinessCondition(func(obj *unstructured.Unstructured) bool {
//		// Check if Prometheus ServiceMonitor is being scraped
//		status, found, _ := unstructured.NestedString(obj.Object, "status", "conditions")
//		if !found {
//			return false
//		}
//
//		// More complex readiness check
//		lastScrape, found, _ := unstructured.NestedString(obj.Object, "status", "lastScrapeTime")
//		if !found {
//			return false
//		}
//
//		// Consider ready if scraped within last 5 minutes
//		scrapeTime, err := time.Parse(time.RFC3339, lastScrape)
//		return err == nil && time.Since(scrapeTime) < 5*time.Minute
//	})
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithReadinessCondition(f func(obj *unstructured.Unstructured) bool) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithReadinessCondition(f)
	return b
}

// WithRequireManualDeletionForFinalize specifies when an untyped resource requires manual cleanup
// during custom resource finalization.
//
// This is particularly important for untyped resources managed by third-party operators,
// as they may have complex deletion procedures or external dependencies that need special handling.
//
// Working with unstructured objects requires using helper functions to access nested fields.
//
// Example:
//
//	.WithRequireManualDeletionForFinalize(func(obj *unstructured.Unstructured) bool {
//		// Check if resource has special deletion requirements
//		deletionPolicy, found, _ := unstructured.NestedString(
//			obj.Object, "spec", "deletionPolicy")
//		if found && deletionPolicy == "Retain" {
//			return true // Requires manual cleanup
//		}
//
//		// Check for external dependencies
//		externalDeps, found, _ := unstructured.NestedSlice(
//			obj.Object, "status", "externalDependencies")
//		return found && len(externalDeps) > 0
//	})
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithRequireManualDeletionForFinalize(f func(obj *unstructured.Unstructured) bool) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithRequireManualDeletionForFinalize(f)
	return b
}

// WithSkipAndDeleteOnCondition specifies when to skip creating or delete an existing untyped resource.
//
// This is particularly useful for untyped resources that depend on optional third-party
// operators or should only be created under certain conditions. The function is evaluated
// during each reconciliation cycle.
//
// Common use cases for untyped resources:
//   - Optional monitoring resources (when Prometheus operator might not be installed)
//   - Feature flag controlled third-party integrations
//   - Environment-specific operator resources
//   - Conditional third-party service configurations
//
// Example:
//
//	.WithSkipAndDeleteOnCondition(func() bool {
//		cr := ctx.GetCustomResource()
//
//		// Only create monitoring resources if monitoring is enabled
//		if !cr.Spec.Monitoring.Enabled {
//			return true
//		}
//
//		// Check if the required operator is available
//		client := ctx.GetClient()
//		prometheusOperator := &appsv1.Deployment{}
//		err := client.Get(ctx, types.NamespacedName{
//			Name: "prometheus-operator",
//			Namespace: "monitoring",
//		}, prometheusOperator)
//
//		// Skip if operator not found or not ready
//		return err != nil || prometheusOperator.Status.ReadyReplicas == 0
//	})
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithSkipAndDeleteOnCondition(f func() bool) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithSkipAndDeleteOnCondition(f)
	return b
}

// WithUserIdentifier assigns a custom identifier for this untyped resource.
//
// This identifier is used for logging, debugging, and distinguishing between multiple
// untyped resources. It's especially important for untyped resources since the resource
// types may not be immediately obvious from logs, and you might be managing multiple
// third-party resources of different types.
//
// The identifier should be descriptive and include both the resource purpose and type.
//
// Example:
//
//	.WithUserIdentifier("prometheus-servicemonitor") // Clear purpose and type
//	.WithUserIdentifier("grafana-dashboard-app")     // Identifies both operator and purpose
//	.WithUserIdentifier("istio-virtualservice")     // Service mesh resource identifier
func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithUserIdentifier(identifier string) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithUserIdentifier(identifier)
	return b
}
