package ctrlfwk

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceBuilder provides a builder pattern for creating Resource[T] instances
type ResourceBuilder[T client.Object] struct {
	resource *Resource[T]
}

// NewResourceBuilder creates a new ResourceBuilder for the given type
func NewResourceBuilder[T client.Object](_ T) *ResourceBuilder[T] {
	return &ResourceBuilder[T]{
		resource: &Resource[T]{},
	}
}

// WithKey sets the resource key using a NamespacedName
func (b *ResourceBuilder[T]) WithKey(name types.NamespacedName) *ResourceBuilder[T] {
	b.resource.keyF = func() types.NamespacedName {
		return name
	}
	return b
}

// WithKeyFunc sets the resource key using a function
func (b *ResourceBuilder[T]) WithKeyFunc(f func() types.NamespacedName) *ResourceBuilder[T] {
	b.resource.keyF = f
	return b
}

// WithMutator sets the mutator function
func (b *ResourceBuilder[T]) WithMutator(f Mutator[T]) *ResourceBuilder[T] {
	b.resource.mutateF = f
	return b
}

// WithOutput sets the output object
func (b *ResourceBuilder[T]) WithOutput(obj T) *ResourceBuilder[T] {
	b.resource.output = obj
	return b
}

// WithReadinessCondition sets the readiness condition function
func (b *ResourceBuilder[T]) WithReadinessCondition(f func(obj T) bool) *ResourceBuilder[T] {
	b.resource.isReadyF = f
	return b
}

// WithSkipAndDeleteOnCondition sets the condition for skipping and deleting
func (b *ResourceBuilder[T]) WithSkipAndDeleteOnCondition(f func() bool) *ResourceBuilder[T] {
	b.resource.shouldDeleteF = f
	return b
}

// WithRequireManualDeletionForFinalize sets the manual deletion requirement function
func (b *ResourceBuilder[T]) WithRequireManualDeletionForFinalize(f func(obj T) bool) *ResourceBuilder[T] {
	b.resource.requiresDeletionF = f
	return b
}

// WithBeforeReconcile sets the before reconcile hook
func (b *ResourceBuilder[T]) WithBeforeReconcile(f func(ctx context.Context) error) *ResourceBuilder[T] {
	b.resource.beforeReconcileF = f
	return b
}

// WithAfterReconcile sets the after reconcile hook
func (b *ResourceBuilder[T]) WithAfterReconcile(f func(ctx context.Context, resource T) error) *ResourceBuilder[T] {
	b.resource.afterReconcileF = f
	return b
}

// WithAfterCreate sets the after create hook
func (b *ResourceBuilder[T]) WithAfterCreate(f func(ctx context.Context, resource T) error) *ResourceBuilder[T] {
	b.resource.onCreateF = f
	return b
}

// WithAfterUpdate sets the after update hook
func (b *ResourceBuilder[T]) WithAfterUpdate(f func(ctx context.Context, resource T) error) *ResourceBuilder[T] {
	b.resource.onUpdateF = f
	return b
}

// WithAfterDelete sets the after delete hook
func (b *ResourceBuilder[T]) WithAfterDelete(f func(ctx context.Context, resource T) error) *ResourceBuilder[T] {
	b.resource.onDeleteF = f
	return b
}

// WithAfterFinalize sets the after finalize hook
func (b *ResourceBuilder[T]) WithAfterFinalize(f func(ctx context.Context, resource T) error) *ResourceBuilder[T] {
	b.resource.onFinalizeF = f
	return b
}

// WithUserIdentifier sets a custom user identifier for the resource
func (b *ResourceBuilder[T]) WithUserIdentifier(identifier string) *ResourceBuilder[T] {
	b.resource.userIdentifier = identifier
	return b
}

// Build returns the constructed Resource[T]
func (b *ResourceBuilder[T]) Build() *Resource[T] {
	return b.resource
}
