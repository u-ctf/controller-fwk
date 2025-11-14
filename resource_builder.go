package ctrlfwk

import (
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceBuilder provides a fluent builder pattern for creating Resource instances that your
// custom resource controller manages during reconciliation.
//
// Resources represent Kubernetes objects that your controller creates, updates, and manages
// as part of achieving the desired state for your custom resource. Unlike dependencies,
// resources are owned by your custom resource and are created/managed by your controller.
//
// Type parameters:
//   - CustomResource: The custom resource that owns and manages this resource
//   - ContextType: The context type containing the custom resource and additional data
//   - ResourceType: The Kubernetes resource type this builder creates (e.g., Deployment, Service)
//
// Common use cases:
//   - Creating Deployments for your application custom resource
//   - Managing Services, ConfigMaps, and Secrets
//   - Setting up RBAC resources (Roles, RoleBindings)
//   - Creating PersistentVolumeClaims for stateful applications
//
// Example:
//
//	// Create a Deployment resource for your custom resource
//	deployment := NewResourceBuilder(ctx, &appsv1.Deployment{}).
//		WithKeyFunc(func() types.NamespacedName {
//			return types.NamespacedName{
//				Name:      ctx.GetCustomResource().Name + "-deployment",
//				Namespace: ctx.GetCustomResource().Namespace,
//			}
//		}).
//		WithMutator(func(deployment *appsv1.Deployment) error {
//			// Configure deployment spec based on custom resource
//			deployment.Spec.Replicas = ctx.GetCustomResource().Spec.Replicas
//			return controllerutil.SetOwnerReference(ctx.GetCustomResource(), deployment, scheme)
//		}).
//		WithReadinessCondition(func(deployment *appsv1.Deployment) bool {
//			return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas
//		}).
//		Build()
type ResourceBuilder[CustomResource client.Object, ContextType Context[CustomResource], ResourceType client.Object] struct {
	resource *Resource[CustomResource, ContextType, ResourceType]
}

// NewResourceBuilder creates a new ResourceBuilder for constructing managed Kubernetes resources.
//
// Resources are Kubernetes objects that your controller creates and manages to implement
// the desired state defined by your custom resource. They are typically owned by your
// custom resource and have owner references set for garbage collection.
//
// Parameters:
//   - ctx: The context containing the custom resource and additional data
//   - _: A zero-value instance used for type inference (e.g., &appsv1.Deployment{})
//
// The resource will be reconciled when used with ReconcileResourcesStep or
// ReconcileResourceStep during the reconciliation process.
//
// Key differences from dependencies:
//   - Resources are CREATED and MANAGED by your controller
//   - Dependencies are CONSUMED by your controller (external resources)
//   - Resources typically have owner references to your custom resource
//   - Resources are deleted when your custom resource is deleted
//
// Common resource patterns:
//   - Application deployments and services
//   - Configuration resources (ConfigMaps, Secrets)
//   - RBAC resources for your application
//   - Storage resources (PVCs) for stateful applications
//
// Example:
//
//	// Create a Service resource for your web application
//	service := NewResourceBuilder(ctx, &corev1.Service{}).
//		WithKeyFunc(func() types.NamespacedName {
//			return types.NamespacedName{
//				Name:      ctx.GetCustomResource().Name + "-service",
//				Namespace: ctx.GetCustomResource().Namespace,
//			}
//		}).
//		WithMutator(func(svc *corev1.Service) error {
//			svc.Spec.Selector = map[string]string{"app": ctx.GetCustomResource().Name}
//			svc.Spec.Ports = []corev1.ServicePort{{
//				Port:       80,
//				TargetPort: intstr.FromInt(8080),
//				Protocol:   corev1.ProtocolTCP,
//			}}
//			return controllerutil.SetOwnerReference(ctx.GetCustomResource(), svc, scheme)
//		}).
//		Build()
func NewResourceBuilder[CustomResource client.Object, ContextType Context[CustomResource], ResourceType client.Object](ctx ContextType, _ ResourceType) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	return &ResourceBuilder[CustomResource, ContextType, ResourceType]{
		resource: &Resource[CustomResource, ContextType, ResourceType]{},
	}
}

// WithKey specifies a static NamespacedName for the resource.
//
// This is useful when the resource name and namespace are known at build time
// and don't need to be computed dynamically based on the custom resource state.
//
// For dynamic naming based on custom resource properties, use WithKeyFunc instead.
//
// Example:
//
//	.WithKey(types.NamespacedName{
//		Name:      "my-app-service",
//		Namespace: "default",
//	}) // Static name for the service
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithKey(name types.NamespacedName) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.keyF = func() types.NamespacedName {
		return name
	}
	return b
}

// WithKeyFunc specifies a function that dynamically determines the resource's NamespacedName.
//
// This function is called during reconciliation to determine where the resource should
// be created or found. It's evaluated each time the resource is reconciled, allowing
// for dynamic naming based on the current state of the custom resource.
//
// The function typically derives the name from the custom resource's metadata or spec,
// and should return a consistent result for the same custom resource state.
//
// Common patterns:
//   - Prefixing with custom resource name: ctx.GetCustomResource().Name + "-suffix"
//   - Using custom resource namespace: ctx.GetCustomResource().Namespace
//   - Conditional naming based on custom resource spec
//   - When the name is stored in the spec, you might wanna refer to the status when the spec field is updated or disappears
//
// Example:
//
//	.WithKeyFunc(func() types.NamespacedName {
//		cr := ctx.GetCustomResource()
//		return types.NamespacedName{
//			Name:      cr.Name + "-" + cr.Spec.Component, // Dynamic based on spec
//			Namespace: cr.Namespace,                        // Same namespace as CR
//		}
//	})
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithKeyFunc(f func() types.NamespacedName) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.keyF = f
	return b
}

// WithMutator specifies the function that configures the resource's desired state.
//
// The mutator function is called whenever the resource needs to be created or updated.
// It receives the resource object and should configure all necessary fields to match
// the desired state defined by your custom resource.
//
// The mutator should:
//   - Set all required fields on the resource
//   - Configure the resource based on custom resource spec
//   - Set owner references for garbage collection
//   - Apply labels, annotations, and other metadata
//   - Return an error if configuration fails
//
// The mutator is called for both CREATE and UPDATE operations, so it should be
// idempotent and handle both scenarios gracefully.
//
// Example:
//
//	.WithMutator(func(deployment *appsv1.Deployment) error {
//		cr := ctx.GetCustomResource()
//
//		// Configure deployment based on custom resource
//		deployment.Spec.Replicas = cr.Spec.Replicas
//		deployment.Spec.Selector = &metav1.LabelSelector{
//			MatchLabels: map[string]string{"app": cr.Name},
//		}
//		deployment.Spec.Template.Spec.Containers = []corev1.Container{{
//			Name:  "app",
//			Image: cr.Spec.Image,
//		}}
//
//		// Set owner reference for garbage collection
//		return controllerutil.SetOwnerReference(cr, deployment, ctx.GetScheme())
//	})
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithMutator(f Mutator[ResourceType]) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.mutateF = f
	return b
}

// WithOutput specifies where to store the reconciled resource after successful operations.
//
// The provided object will be populated with the resource's current state from the
// cluster after reconciliation completes. This allows other parts of your controller
// logic to access the resource's runtime state, such as generated fields, status
// information, or cluster-assigned values.
//
// The output object should be a field in your context's data structure to ensure
// it's accessible throughout the reconciliation process.
//
// Common use cases:
//   - Accessing service ClusterIP after creation
//   - Reading generated secret data
//   - Getting deployment status for custom resource status updates
//   - Obtaining persistent volume claim details
//
// Example:
//
//	type MyContextData struct {
//		AppService *corev1.Service
//	}
//
//	service := NewResourceBuilder(ctx, &corev1.Service{}).
//		// ... other configuration ...
//		WithOutput(ctx.Data.AppService). // Store reconciled service here
//		Build()
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithOutput(obj ResourceType) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.output = obj
	return b
}

// WithReadinessCondition defines custom logic to determine when the resource is ready.
//
// The provided function is called with the current resource state and should return
// true if the resource has reached the desired operational state. This affects when
// the overall custom resource is considered ready and can influence reconciliation flow.
//
// If no readiness condition is provided, the resource is considered ready as soon
// as it exists and has been successfully created or updated.
//
// Common readiness patterns:
//   - Deployments: Check that ReadyReplicas == DesiredReplicas
//   - Services: Verify endpoints are populated
//   - Jobs: Check for successful completion
//   - Custom resources: Examine status conditions
//
// Example:
//
//	.WithReadinessCondition(func(deployment *appsv1.Deployment) bool {
//		// Deployment is ready when all replicas are ready
//		if deployment.Spec.Replicas == nil {
//			return deployment.Status.ReadyReplicas > 0
//		}
//		return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas &&
//		       deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas
//	})
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithReadinessCondition(f func(obj ResourceType) bool) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.isReadyF = f
	return b
}

// WithSkipAndDeleteOnCondition specifies when to skip creating or delete an existing resource.
//
// The provided function is evaluated during reconciliation. When it returns true:
//   - If the resource doesn't exist, it will be skipped (not created)
//   - If the resource exists, it will be deleted
//
// This is useful for conditional resource management based on custom resource configuration,
// feature flags, or environmental conditions.
//
// The condition function is called each reconciliation cycle, so resources can be
// dynamically created or removed based on changing conditions.
//
// Common use cases:
//   - Feature toggles that enable/disable optional components
//   - Environment-specific resources (dev vs prod)
//   - Conditional scaling or resource allocation
//   - Migration scenarios where resources are phased out
//
// Example:
//
//	.WithSkipAndDeleteOnCondition(func() bool {
//		cr := ctx.GetCustomResource()
//		// Only create monitoring service if monitoring is enabled
//		return !cr.Spec.Monitoring.Enabled
//	})
//
//	// Another example: conditional based on environment
//	.WithSkipAndDeleteOnCondition(func() bool {
//		// Skip expensive resources in development environment
//		return ctx.GetCustomResource().Spec.Environment == "development"
//	})
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithSkipAndDeleteOnCondition(f func() bool) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.shouldDeleteF = f
	return b
}

// WithRequireManualDeletionForFinalize specifies when a resource requires manual cleanup
// during custom resource finalization.
//
// When the custom resource is being deleted, this function is called to determine if
// the resource requires special handling before the custom resource can be fully removed.
// If the function returns true, the resource must be manually cleaned up before
// finalization can complete.
//
// This is typically used for resources that:
//   - Have external dependencies that need cleanup
//   - Store important data that needs backup
//   - Have complex deletion procedures
//   - Need graceful shutdown processes
//
// The function receives the current resource state to make decisions based on the
// resource's current condition or configuration.
//
// Example:
//
//	.WithRequireManualDeletionForFinalize(func(pvc *corev1.PersistentVolumeClaim) bool {
//		// Require manual deletion for PVCs that contain important data
//		if important, exists := pvc.Annotations["data.important"]; exists {
//			return important == "true"
//		}
//		return false
//	})
//
//	// Another example: based on resource state
//	.WithRequireManualDeletionForFinalize(func(deployment *appsv1.Deployment) bool {
//		// Require manual deletion if deployment is still running
//		return deployment.Status.Replicas > 0
//	})
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithRequireManualDeletionForFinalize(f func(obj ResourceType) bool) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.requiresDeletionF = f
	return b
}

// WithBeforeReconcile registers a hook function to execute before resource reconciliation.
//
// This function is called before any resource operations (create, update, or delete)
// are performed. It can be used for setup tasks, validation, or preparation work.
// If the function returns an error, resource reconciliation will be aborted.
//
// Common use cases:
//   - Validating preconditions for resource creation
//   - Setting up external dependencies
//   - Performing cleanup of old resources
//   - Logging reconciliation attempts
//   - Initializing shared state
//
// Example:
//
//	.WithBeforeReconcile(func(ctx MyContext) error {
//		logger := ctx.GetLogger()
//		cr := ctx.GetCustomResource()
//
//		logger.Info("Reconciling application deployment", "version", cr.Spec.Version)
//
//		// Validate configuration before proceeding
//		if cr.Spec.Replicas < 0 {
//			return fmt.Errorf("replica count cannot be negative: %d", cr.Spec.Replicas)
//		}
//
//		return nil
//	})
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithBeforeReconcile(f func(ctx ContextType) error) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.beforeReconcileF = f
	return b
}

// WithAfterReconcile registers a hook function to execute after successful resource reconciliation.
//
// This function is called after the resource has been successfully created, updated, or
// verified to exist with the correct configuration. It receives the current resource
// state from the cluster and can be used for post-processing, status updates, or
// triggering additional operations.
//
// If the function returns an error, the reconciliation will fail even though the
// resource operation itself was successful.
//
// Common use cases:
//   - Updating custom resource status with resource information
//   - Triggering external system notifications
//   - Caching resource data for future use
//   - Logging successful operations
//   - Initiating dependent operations
//
// Example:
//
//	.WithAfterReconcile(func(ctx MyContext, service *corev1.Service) error {
//		cr := ctx.GetCustomResource()
//
//		// Update custom resource status with service information
//		if cr.Status.ServiceStatus == nil {
//			cr.Status.ServiceStatus = &ServiceStatus{}
//		}
//		cr.Status.ServiceStatus.Name = service.Name
//		cr.Status.ServiceStatus.ClusterIP = service.Spec.ClusterIP
//
//		// Set ready condition
//		meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
//			Type:   "ServiceReady",
//			Status: metav1.ConditionTrue,
//			Reason: "ServiceCreated",
//		})
//
//		return ctx.PatchStatus()
//	})
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithAfterReconcile(f func(ctx ContextType, resource ResourceType) error) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.afterReconcileF = f
	return b
}

// WithAfterCreate registers a hook function that executes only when a resource is newly created.
//
// This function is called specifically when a resource is created for the first time,
// not when it's updated. It's useful for one-time initialization tasks, logging
// creation events, or triggering operations that should only happen on resource creation.
//
// The function receives the newly created resource in its current state from the cluster,
// including any fields that were populated by Kubernetes (like UID, creation timestamp).
//
// Common use cases:
//   - Logging resource creation events
//   - One-time initialization tasks
//   - Sending creation notifications to external systems
//   - Recording metrics for new resource creation
//   - Triggering initial configuration workflows
//
// Example:
//
//	.WithAfterCreate(func(ctx MyContext, deployment *appsv1.Deployment) error {
//		logger := ctx.GetLogger()
//		logger.Info("Deployment created successfully",
//			"name", deployment.Name,
//			"namespace", deployment.Namespace,
//			"uid", deployment.UID,
//			"replicas", *deployment.Spec.Replicas)
//
//		// Send notification to monitoring system
//		return ctx.NotifyExternalSystem("deployment_created", deployment.Name)
//	})
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithAfterCreate(f func(ctx ContextType, resource ResourceType) error) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.onCreateF = f
	return b
}

// WithAfterUpdate registers a hook function that executes only when a resource is updated.
//
// This function is called specifically when an existing resource is modified, not when
// it's initially created. It's useful for tracking changes, logging update events, or
// triggering operations that should only happen when configuration changes.
//
// The function receives the updated resource in its current state from the cluster
// after the update operation has completed successfully.
//
// Common use cases:
//   - Logging configuration changes
//   - Triggering rolling updates or restarts
//   - Notifying external systems of changes
//   - Recording metrics for resource modifications
//   - Validating update results
//
// Example:
//
//	.WithAfterUpdate(func(ctx MyContext, deployment *appsv1.Deployment) error {
//		logger := ctx.GetLogger()
//		cr := ctx.GetCustomResource()
//
//		logger.Info("Deployment updated successfully",
//			"name", deployment.Name,
//			"generation", deployment.Generation,
//			"replicas", *deployment.Spec.Replicas)
//
//		// Update custom resource status with latest generation
//		cr.Status.DeploymentGeneration = deployment.Generation
//		return ctx.PatchStatus()
//	})
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithAfterUpdate(f func(ctx ContextType, resource ResourceType) error) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.onUpdateF = f
	return b
}

// WithAfterDelete registers a hook function that executes after a resource is deleted.
//
// This function is called when a resource has been successfully deleted from the cluster,
// either due to a delete condition being met or during custom resource finalization.
// It can be used for cleanup tasks, logging deletion events, or notifying external systems.
//
// The function receives the resource object as it existed just before deletion.
// At this point, the resource no longer exists in the cluster.
//
// Common use cases:
//   - Logging resource deletion events
//   - Cleaning up external resources or dependencies
//   - Notifying monitoring or auditing systems
//   - Recording metrics for resource deletion
//   - Updating custom resource status to reflect deletion
//
// Example:
//
//	.WithAfterDelete(func(ctx MyContext, pvc *corev1.PersistentVolumeClaim) error {
//		logger := ctx.GetLogger()
//		logger.Info("PersistentVolumeClaim deleted",
//			"name", pvc.Name,
//			"namespace", pvc.Namespace,
//			"storageClass", *pvc.Spec.StorageClassName)
//
//		// Clean up any backup data associated with this PVC
//		return ctx.CleanupBackupData(pvc.Name)
//	})
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithAfterDelete(f func(ctx ContextType, resource ResourceType) error) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.onDeleteF = f
	return b
}

// WithAfterFinalize registers a hook function that executes during custom resource finalization.
//
// This function is called when the custom resource is being deleted and the resource
// needs to be cleaned up as part of the finalization process. It's used for graceful
// shutdown procedures and cleanup of resources that require special handling.
//
// The function should perform any necessary cleanup operations and ensure that
// external dependencies are properly handled before the custom resource is fully removed.
//
// Common use cases:
//   - Graceful shutdown of applications
//   - Backup of important data before deletion
//   - Cleanup of external resources or registrations
//   - Notifying external systems of resource removal
//   - Removing finalizers from dependent resources
//
// Example:
//
//	.WithAfterFinalize(func(ctx MyContext, deployment *appsv1.Deployment) error {
//		logger := ctx.GetLogger()
//
//		// Gracefully scale down before deletion
//		if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas > 0 {
//			logger.Info("Scaling down deployment before finalization")
//			zero := int32(0)
//			deployment.Spec.Replicas = &zero
//			return ctx.Update(deployment)
//		}
//
//		// Perform final cleanup
//		return ctx.CleanupExternalResources(deployment.Name)
//	})
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithAfterFinalize(f func(ctx ContextType, resource ResourceType) error) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.onFinalizeF = f
	return b
}

// WithUserIdentifier assigns a custom identifier for this resource.
//
// This identifier is used for logging, debugging, and distinguishing between multiple
// resources of the same type within your controller. If not provided, a default
// identifier will be generated based on the resource type.
//
// The identifier appears in logs and error messages, making it easier to track
// specific resources during debugging and troubleshooting.
//
// Useful for:
//   - Distinguishing between multiple deployments (e.g., "frontend", "backend")
//   - Providing meaningful names for different services
//   - Creating clear audit trails in logs
//   - Simplifying debugging of complex resource hierarchies
//
// Example:
//
//	.WithUserIdentifier("frontend-deployment") // Clear name for logs
//	.WithUserIdentifier("database-service")    // Easy to identify in debugging
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithUserIdentifier(identifier string) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.userIdentifier = identifier
	return b
}

// CanBePaused specifies whether this resource supports pausing reconciliation.
//
// When set to true, the resource will respect the paused state of the custom resource.
// If the custom resource is marked as paused (e.g., via a label), reconciliation
// for this resource will be skipped until the pause is lifted.
//
// This is useful for scenarios where you want to temporarily halt changes to
// certain resources without deleting them or affecting other parts of the system.
//
// Common use cases:
//   - Temporarily halting updates during maintenance windows
//   - Pausing non-critical resources while troubleshooting issues
//   - Allowing manual intervention before resuming automated management
//
// Example:
//
//	.WithCanBePaused(true) // Enable pausing for this resource
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithCanBePaused(canBePaused bool) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.canBePausedF = func() bool {
		return canBePaused
	}
	return b
}

// WithCanBePausedFunc specifies a function to determine if this resource supports pausing reconciliation.
//
// The provided function is called during reconciliation to check if the resource
// should respect the paused state of the custom resource. If it returns true,
// reconciliation for this resource will be skipped when the custom resource is paused.
//
// This allows for dynamic control over which resources can be paused based on
// the current state or configuration of the custom resource.
//
// Example:
//
//	.WithCanBePausedFunc(func() bool {
//	    // Custom logic to determine if the resource can be paused
//	    return someCondition
//	})
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithCanBePausedFunc(f func() bool) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.canBePausedF = f
	return b
}

// Build constructs and returns the final Resource instance with all configured options.
//
// This method finalizes the builder pattern and creates a resource that can be used
// in reconciliation steps. The returned resource contains all the configuration
// specified through the builder methods.
//
// The resource must be used with appropriate reconciliation steps (such as
// ReconcileResourcesStep) to actually perform the resource management operations.
//
// Validation:
//   - At least one of WithKey or WithKeyFunc must be called before Build()
//   - WithMutator is typically required for meaningful resource management
//
// Returns a configured Resource instance ready for use in reconciliation.
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) Build() *Resource[CustomResource, ContextType, ResourceType] {
	return b.resource
}
