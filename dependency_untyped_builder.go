package ctrlfwk

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UntypedDependencyBuilder provides a builder pattern for creating Dependency[K, T] instances
type UntypedDependencyBuilder[CustomResourceType client.Object, ContextType Context[CustomResourceType]] struct {
	inner *DependencyBuilder[CustomResourceType, ContextType, *unstructured.Unstructured]
	gvk   schema.GroupVersionKind
}

// NewUntypedDependencyBuilder creates a new UntypedDependencyBuilder for the given type
func NewUntypedDependencyBuilder[CustomResourceType client.Object, ContextType Context[CustomResourceType]](ctx ContextType, gvk schema.GroupVersionKind) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	return &UntypedDependencyBuilder[CustomResourceType, ContextType]{
		inner: NewDependencyBuilder(ctx, &unstructured.Unstructured{}),
		gvk:   gvk,
	}
}

func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) Build() *UntypedDependency[CustomResourceType, ContextType] {
	return &UntypedDependency[CustomResourceType, ContextType]{
		Dependency: b.inner.Build(),
		gvk:        b.gvk,
	}
}

func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithAfterReconcile(f func(ctx ContextType, resource *unstructured.Unstructured) error) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithAfterReconcile(f)
	return b
}

func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithBeforeReconcile(f func(ctx ContextType) error) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithBeforeReconcile(f)
	return b
}

func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithIsReadyFunc(f func(obj *unstructured.Unstructured) bool) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithIsReadyFunc(f)
	return b
}

func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithName(name string) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithName(name)
	return b
}

func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithNamespace(namespace string) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithNamespace(namespace)
	return b
}

func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithOptional(optional bool) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithOptional(optional)
	return b
}

func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithOutput(obj *unstructured.Unstructured) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithOutput(obj)
	return b
}

func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithReadinessCondition(f func(obj *unstructured.Unstructured) bool) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithReadinessCondition(f)
	return b
}

func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithUserIdentifier(identifier string) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithUserIdentifier(identifier)
	return b
}

func (b *UntypedDependencyBuilder[CustomResourceType, ContextType]) WithWaitForReady(waitForReady bool) *UntypedDependencyBuilder[CustomResourceType, ContextType] {
	b.inner = b.inner.WithWaitForReady(waitForReady)
	return b
}
