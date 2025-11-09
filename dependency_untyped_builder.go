package ctrlfwk

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UntypedDependencyBuilder provides a builder pattern for creating Dependency[K, T] instances
type UntypedDependencyBuilder[K client.Object] struct {
	inner *DependencyBuilder[K, *unstructured.Unstructured]
	gvk   schema.GroupVersionKind
}

// NewUntypedDependencyBuilder creates a new UntypedDependencyBuilder for the given type
func NewUntypedDependencyBuilder[K client.Object](ctx Context[K], gvk schema.GroupVersionKind) *UntypedDependencyBuilder[K] {
	return &UntypedDependencyBuilder[K]{
		inner: NewDependencyBuilder(ctx, &unstructured.Unstructured{}),
		gvk:   gvk,
	}
}

func (b *UntypedDependencyBuilder[K]) Build() *UntypedDependency[K] {
	return &UntypedDependency[K]{
		Dependency: b.inner.Build(),
		gvk:        b.gvk,
	}
}

func (b *UntypedDependencyBuilder[K]) WithAfterReconcile(f func(ctx Context[K], resource *unstructured.Unstructured) error) *UntypedDependencyBuilder[K] {
	b.inner = b.inner.WithAfterReconcile(f)
	return b
}

func (b *UntypedDependencyBuilder[K]) WithBeforeReconcile(f func(ctx Context[K]) error) *UntypedDependencyBuilder[K] {
	b.inner = b.inner.WithBeforeReconcile(f)
	return b
}

func (b *UntypedDependencyBuilder[K]) WithIsReadyFunc(f func(obj *unstructured.Unstructured) bool) *UntypedDependencyBuilder[K] {
	b.inner = b.inner.WithIsReadyFunc(f)
	return b
}

func (b *UntypedDependencyBuilder[K]) WithMutator(f func(obj *unstructured.Unstructured) error) *UntypedDependencyBuilder[K] {
	b.inner = b.inner.WithMutator(f)
	return b
}

func (b *UntypedDependencyBuilder[K]) WithName(name string) *UntypedDependencyBuilder[K] {
	b.inner = b.inner.WithName(name)
	return b
}

func (b *UntypedDependencyBuilder[K]) WithNamespace(namespace string) *UntypedDependencyBuilder[K] {
	b.inner = b.inner.WithNamespace(namespace)
	return b
}

func (b *UntypedDependencyBuilder[K]) WithOptional(optional bool) *UntypedDependencyBuilder[K] {
	b.inner = b.inner.WithOptional(optional)
	return b
}

func (b *UntypedDependencyBuilder[K]) WithOutput(obj *unstructured.Unstructured) *UntypedDependencyBuilder[K] {
	b.inner = b.inner.WithOutput(obj)
	return b
}

func (b *UntypedDependencyBuilder[K]) WithReadinessCondition(f func(obj *unstructured.Unstructured) bool) *UntypedDependencyBuilder[K] {
	b.inner = b.inner.WithReadinessCondition(f)
	return b
}

func (b *UntypedDependencyBuilder[K]) WithUserIdentifier(identifier string) *UntypedDependencyBuilder[K] {
	b.inner = b.inner.WithUserIdentifier(identifier)
	return b
}

func (b *UntypedDependencyBuilder[K]) WithWaitForReady(waitForReady bool) *UntypedDependencyBuilder[K] {
	b.inner = b.inner.WithWaitForReady(waitForReady)
	return b
}
