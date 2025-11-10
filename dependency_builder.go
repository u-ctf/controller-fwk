package ctrlfwk

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DependencyBuilder provides a builder pattern for creating Dependency[K,C, T] instances
type DependencyBuilder[CustomResourceType client.Object, ContextType Context[CustomResourceType], DependencyType client.Object] struct {
	dependency *Dependency[CustomResourceType, ContextType, DependencyType]
}

// NewDependencyBuilder creates a new DependencyBuilder for the given type
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

// WithOutput sets the output object
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithOutput(obj DependencyType) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.output = obj
	return b
}

// WithIsReadyFunc sets the readiness condition function
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithIsReadyFunc(f func(obj DependencyType) bool) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.isReadyF = f
	return b
}

// WithOptional sets whether the dependency is optional
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithOptional(optional bool) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.isOptional = optional
	return b
}

// WithName sets the name of the dependency resource
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithName(name string) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.name = name
	return b
}

// WithNamespace sets the namespace of the dependency resource
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithNamespace(namespace string) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.namespace = namespace
	return b
}

// WithWaitForReady sets whether to wait for the dependency to be ready
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithWaitForReady(waitForReady bool) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.waitForReady = waitForReady
	return b
}

// WithUserIdentifier sets a custom user identifier for the dependency
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithUserIdentifier(identifier string) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.userIdentifier = identifier
	return b
}

// WithBeforeReconcile sets the before reconcile hook
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithBeforeReconcile(f func(ctx ContextType) error) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.beforeReconcileF = f
	return b
}

// WithAfterReconcile sets the after reconcile hook
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithAfterReconcile(f func(ctx ContextType, resource DependencyType) error) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.afterReconcileF = f
	return b
}

// WithReadinessCondition sets the readiness condition function (alias for WithIsReadyFunc)
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) WithReadinessCondition(f func(obj DependencyType) bool) *DependencyBuilder[CustomResourceType, ContextType, DependencyType] {
	b.dependency.isReadyF = f
	return b
}

// Build returns the constructed Dependency[K,C, T]
func (b *DependencyBuilder[CustomResourceType, ContextType, DependencyType]) Build() *Dependency[CustomResourceType, ContextType, DependencyType] {
	return b.dependency
}
