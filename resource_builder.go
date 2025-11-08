package ctrlfwk

import (
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceBuilder provides a builder pattern for creating Resource[K, T] instances
type ResourceBuilder[K, T client.Object] struct {
	resource *Resource[K, T]
}

// NewResourceBuilder creates a new ResourceBuilder for the given type
func NewResourceBuilder[K, T client.Object](ctx Context[K], _ T) *ResourceBuilder[K, T] {
	return &ResourceBuilder[K, T]{
		resource: &Resource[K, T]{},
	}
}

// WithKey sets the resource key using a NamespacedName
func (b *ResourceBuilder[K, T]) WithKey(name types.NamespacedName) *ResourceBuilder[K, T] {
	b.resource.keyF = func() types.NamespacedName {
		return name
	}
	return b
}

// WithKeyFunc sets the resource key using a function
func (b *ResourceBuilder[K, T]) WithKeyFunc(f func() types.NamespacedName) *ResourceBuilder[K, T] {
	b.resource.keyF = f
	return b
}

// WithMutator sets the mutator function
func (b *ResourceBuilder[K, T]) WithMutator(f Mutator[T]) *ResourceBuilder[K, T] {
	b.resource.mutateF = f
	return b
}

// WithOutput sets the output object
func (b *ResourceBuilder[K, T]) WithOutput(obj T) *ResourceBuilder[K, T] {
	b.resource.output = obj
	return b
}

// WithReadinessCondition sets the readiness condition function
func (b *ResourceBuilder[K, T]) WithReadinessCondition(f func(obj T) bool) *ResourceBuilder[K, T] {
	b.resource.isReadyF = f
	return b
}

// WithSkipAndDeleteOnCondition sets the condition for skipping and deleting
func (b *ResourceBuilder[K, T]) WithSkipAndDeleteOnCondition(f func() bool) *ResourceBuilder[K, T] {
	b.resource.shouldDeleteF = f
	return b
}

// WithRequireManualDeletionForFinalize sets the manual deletion requirement function
func (b *ResourceBuilder[K, T]) WithRequireManualDeletionForFinalize(f func(obj T) bool) *ResourceBuilder[K, T] {
	b.resource.requiresDeletionF = f
	return b
}

// WithBeforeReconcile sets the before reconcile hook
func (b *ResourceBuilder[K, T]) WithBeforeReconcile(f func(ctx Context[K]) error) *ResourceBuilder[K, T] {
	b.resource.beforeReconcileF = f
	return b
}

// WithAfterReconcile sets the after reconcile hook
func (b *ResourceBuilder[K, T]) WithAfterReconcile(f func(ctx Context[K], resource T) error) *ResourceBuilder[K, T] {
	b.resource.afterReconcileF = f
	return b
}

// WithAfterCreate sets the after create hook
func (b *ResourceBuilder[K, T]) WithAfterCreate(f func(ctx Context[K], resource T) error) *ResourceBuilder[K, T] {
	b.resource.onCreateF = f
	return b
}

// WithAfterUpdate sets the after update hook
func (b *ResourceBuilder[K, T]) WithAfterUpdate(f func(ctx Context[K], resource T) error) *ResourceBuilder[K, T] {
	b.resource.onUpdateF = f
	return b
}

// WithAfterDelete sets the after delete hook
func (b *ResourceBuilder[K, T]) WithAfterDelete(f func(ctx Context[K], resource T) error) *ResourceBuilder[K, T] {
	b.resource.onDeleteF = f
	return b
}

// WithAfterFinalize sets the after finalize hook
func (b *ResourceBuilder[K, T]) WithAfterFinalize(f func(ctx Context[K], resource T) error) *ResourceBuilder[K, T] {
	b.resource.onFinalizeF = f
	return b
}

// WithUserIdentifier sets a custom user identifier for the resource
func (b *ResourceBuilder[K, T]) WithUserIdentifier(identifier string) *ResourceBuilder[K, T] {
	b.resource.userIdentifier = identifier
	return b
}

// Build returns the constructed Resource[K, T]
func (b *ResourceBuilder[K, T]) Build() *Resource[K, T] {
	return b.resource
}
