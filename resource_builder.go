package ctrlfwk

import (
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceBuilder provides a builder pattern for creating Resource[CustomResource, ContextType, ResourceType] instances
type ResourceBuilder[CustomResource client.Object, ContextType Context[CustomResource], ResourceType client.Object] struct {
	resource *Resource[CustomResource, ContextType, ResourceType]
}

// NewResourceBuilder creates a new ResourceBuilder for the given type
func NewResourceBuilder[CustomResource client.Object, ContextType Context[CustomResource], ResourceType client.Object](ctx ContextType, _ ResourceType) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	return &ResourceBuilder[CustomResource, ContextType, ResourceType]{
		resource: &Resource[CustomResource, ContextType, ResourceType]{},
	}
}

// WithKey sets the resource key using a NamespacedName
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithKey(name types.NamespacedName) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.keyF = func() types.NamespacedName {
		return name
	}
	return b
}

// WithKeyFunc sets the resource key using a function
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithKeyFunc(f func() types.NamespacedName) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.keyF = f
	return b
}

// WithMutator sets the mutator function
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithMutator(f Mutator[ResourceType]) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.mutateF = f
	return b
}

// WithOutput sets the output object
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithOutput(obj ResourceType) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.output = obj
	return b
}

// WithReadinessCondition sets the readiness condition function
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithReadinessCondition(f func(obj ResourceType) bool) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.isReadyF = f
	return b
}

// WithSkipAndDeleteOnCondition sets the condition for skipping and deleting
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithSkipAndDeleteOnCondition(f func() bool) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.shouldDeleteF = f
	return b
}

// WithRequireManualDeletionForFinalize sets the manual deletion requirement function
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithRequireManualDeletionForFinalize(f func(obj ResourceType) bool) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.requiresDeletionF = f
	return b
}

// WithBeforeReconcile sets the before reconcile hook
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithBeforeReconcile(f func(ctx ContextType) error) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.beforeReconcileF = f
	return b
}

// WithAfterReconcile sets the after reconcile hook
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithAfterReconcile(f func(ctx ContextType, resource ResourceType) error) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.afterReconcileF = f
	return b
}

// WithAfterCreate sets the after create hook
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithAfterCreate(f func(ctx ContextType, resource ResourceType) error) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.onCreateF = f
	return b
}

// WithAfterUpdate sets the after update hook
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithAfterUpdate(f func(ctx ContextType, resource ResourceType) error) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.onUpdateF = f
	return b
}

// WithAfterDelete sets the after delete hook
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithAfterDelete(f func(ctx ContextType, resource ResourceType) error) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.onDeleteF = f
	return b
}

// WithAfterFinalize sets the after finalize hook
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithAfterFinalize(f func(ctx ContextType, resource ResourceType) error) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.onFinalizeF = f
	return b
}

// WithUserIdentifier sets a custom user identifier for the resource
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) WithUserIdentifier(identifier string) *ResourceBuilder[CustomResource, ContextType, ResourceType] {
	b.resource.userIdentifier = identifier
	return b
}

// Build returns the constructed Resource[CustomResource, ContextType, ResourceType]
func (b *ResourceBuilder[CustomResource, ContextType, ResourceType]) Build() *Resource[CustomResource, ContextType, ResourceType] {
	return b.resource
}
