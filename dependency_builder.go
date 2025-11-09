package ctrlfwk

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DependencyBuilder provides a builder pattern for creating Dependency[K, T] instances
type DependencyBuilder[K client.Object, T client.Object] struct {
	dependency *Dependency[K, T]
}

// NewDependencyBuilder creates a new DependencyBuilder for the given type
func NewDependencyBuilder[K client.Object, T client.Object](ctx Context[K], _ T) *DependencyBuilder[K, T] {
	return &DependencyBuilder[K, T]{
		dependency: &Dependency[K, T]{},
	}
}

// WithOutput sets the output object
func (b *DependencyBuilder[K, T]) WithOutput(obj T) *DependencyBuilder[K, T] {
	b.dependency.output = obj
	return b
}

// WithIsReadyFunc sets the readiness condition function
func (b *DependencyBuilder[K, T]) WithIsReadyFunc(f func(obj T) bool) *DependencyBuilder[K, T] {
	b.dependency.isReadyF = f
	return b
}

// WithOptional sets whether the dependency is optional
func (b *DependencyBuilder[K, T]) WithOptional(optional bool) *DependencyBuilder[K, T] {
	b.dependency.isOptional = optional
	return b
}

// WithName sets the name of the dependency resource
func (b *DependencyBuilder[K, T]) WithName(name string) *DependencyBuilder[K, T] {
	b.dependency.name = name
	return b
}

// WithNamespace sets the namespace of the dependency resource
func (b *DependencyBuilder[K, T]) WithNamespace(namespace string) *DependencyBuilder[K, T] {
	b.dependency.namespace = namespace
	return b
}

// WithWaitForReady sets whether to wait for the dependency to be ready
func (b *DependencyBuilder[K, T]) WithWaitForReady(waitForReady bool) *DependencyBuilder[K, T] {
	b.dependency.waitForReady = waitForReady
	return b
}

// WithUserIdentifier sets a custom user identifier for the dependency
func (b *DependencyBuilder[K, T]) WithUserIdentifier(identifier string) *DependencyBuilder[K, T] {
	b.dependency.userIdentifier = identifier
	return b
}

// WithBeforeReconcile sets the before reconcile hook
func (b *DependencyBuilder[K, T]) WithBeforeReconcile(f func(ctx Context[K]) error) *DependencyBuilder[K, T] {
	b.dependency.beforeReconcileF = f
	return b
}

// WithAfterReconcile sets the after reconcile hook
func (b *DependencyBuilder[K, T]) WithAfterReconcile(f func(ctx Context[K], resource T) error) *DependencyBuilder[K, T] {
	b.dependency.afterReconcileF = f
	return b
}

// WithMutator sets a mutator function (for compatibility with test code)
func (b *DependencyBuilder[K, T]) WithMutator(f func(obj T) error) *DependencyBuilder[K, T] {
	// For compatibility with test code - dependencies typically don't mutate resources
	return b
}

// WithReadinessCondition sets the readiness condition function (alias for WithIsReadyFunc)
func (b *DependencyBuilder[K, T]) WithReadinessCondition(f func(obj T) bool) *DependencyBuilder[K, T] {
	b.dependency.isReadyF = f
	return b
}

// Build returns the constructed Dependency[K, T]
func (b *DependencyBuilder[K, T]) Build() *Dependency[K, T] {
	return b.dependency
}
