package ctrlfwk

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DependencyBuilder provides a builder pattern for creating Dependency[T] instances
type DependencyBuilder[T client.Object] struct {
	dependency *Dependency[T]
}

// NewDependencyBuilder creates a new DependencyBuilder for the given type
func NewDependencyBuilder[T client.Object](_ T) *DependencyBuilder[T] {
	return &DependencyBuilder[T]{
		dependency: &Dependency[T]{},
	}
}

// WithOutput sets the output object
func (b *DependencyBuilder[T]) WithOutput(obj T) *DependencyBuilder[T] {
	b.dependency.output = obj
	return b
}

// WithIsReadyFunc sets the readiness condition function
func (b *DependencyBuilder[T]) WithIsReadyFunc(f func(obj T) bool) *DependencyBuilder[T] {
	b.dependency.isReadyF = f
	return b
}

// WithOptional sets whether the dependency is optional
func (b *DependencyBuilder[T]) WithOptional(optional bool) *DependencyBuilder[T] {
	b.dependency.isOptional = optional
	return b
}

// WithName sets the name of the dependency resource
func (b *DependencyBuilder[T]) WithName(name string) *DependencyBuilder[T] {
	b.dependency.name = name
	return b
}

// WithNamespace sets the namespace of the dependency resource
func (b *DependencyBuilder[T]) WithNamespace(namespace string) *DependencyBuilder[T] {
	b.dependency.namespace = namespace
	return b
}

// WithWaitForReady sets whether to wait for the dependency to be ready
func (b *DependencyBuilder[T]) WithWaitForReady(waitForReady bool) *DependencyBuilder[T] {
	b.dependency.waitForReady = waitForReady
	return b
}

// WithUserIdentifier sets a custom user identifier for the dependency
func (b *DependencyBuilder[T]) WithUserIdentifier(identifier string) *DependencyBuilder[T] {
	b.dependency.userIdentifier = identifier
	return b
}

// WithBeforeReconcile sets the before reconcile hook
func (b *DependencyBuilder[T]) WithBeforeReconcile(f func(ctx context.Context) error) *DependencyBuilder[T] {
	b.dependency.beforeReconcileF = f
	return b
}

// WithAfterReconcile sets the after reconcile hook
func (b *DependencyBuilder[T]) WithAfterReconcile(f func(ctx context.Context, resource T) error) *DependencyBuilder[T] {
	b.dependency.afterReconcileF = f
	return b
}

// WithMutator sets a mutator function (for compatibility with test code)
func (b *DependencyBuilder[T]) WithMutator(f func(obj T) error) *DependencyBuilder[T] {
	// For compatibility with test code - dependencies typically don't mutate resources
	return b
}

// WithReadinessCondition sets the readiness condition function (alias for WithIsReadyFunc)
func (b *DependencyBuilder[T]) WithReadinessCondition(f func(obj T) bool) *DependencyBuilder[T] {
	b.dependency.isReadyF = f
	return b
}

// Build returns the constructed Dependency[T]
func (b *DependencyBuilder[T]) Build() *Dependency[T] {
	return b.dependency
}
