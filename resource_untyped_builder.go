package ctrlfwk

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UntypedResourceBuilder provides a builder pattern for creating Resource[K, *unstructured.Unstructured] instances
type UntypedResourceBuilder[K client.Object] struct {
	inner *ResourceBuilder[K, *unstructured.Unstructured]
	gvk   schema.GroupVersionKind
}

// NewUntypedResourceBuilder creates a new UntypedResourceBuilder for *unstructured.Unstructuredhe given *unstructured.Unstructuredype
func NewUntypedResourceBuilder[K client.Object](ctx Context[K], gvk schema.GroupVersionKind) *UntypedResourceBuilder[K] {
	return &UntypedResourceBuilder[K]{
		inner: NewResourceBuilder(ctx, &unstructured.Unstructured{}),
		gvk:   gvk,
	}
}

func (b *UntypedResourceBuilder[K]) Build() *UntypedResource[K] {
	return &UntypedResource[K]{
		Resource: b.inner.Build(),
		gvk:      b.gvk,
	}
}

func (b *UntypedResourceBuilder[K]) WithAfterCreate(f func(ctx Context[K], resource *unstructured.Unstructured) error) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithAfterCreate(f)
	return b
}

func (b *UntypedResourceBuilder[K]) WithAfterDelete(f func(ctx Context[K], resource *unstructured.Unstructured) error) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithAfterDelete(f)
	return b
}

func (b *UntypedResourceBuilder[K]) WithAfterFinalize(f func(ctx Context[K], resource *unstructured.Unstructured) error) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithAfterFinalize(f)
	return b
}

func (b *UntypedResourceBuilder[K]) WithAfterReconcile(f func(ctx Context[K], resource *unstructured.Unstructured) error) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithAfterReconcile(f)
	return b
}

func (b *UntypedResourceBuilder[K]) WithAfterUpdate(f func(ctx Context[K], resource *unstructured.Unstructured) error) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithAfterUpdate(f)
	return b
}

func (b *UntypedResourceBuilder[K]) WithBeforeReconcile(f func(ctx Context[K]) error) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithBeforeReconcile(f)
	return b
}

func (b *UntypedResourceBuilder[K]) WithKey(name types.NamespacedName) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithKey(name)
	return b
}

func (b *UntypedResourceBuilder[K]) WithKeyFunc(f func() types.NamespacedName) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithKeyFunc(f)
	return b
}

func (b *UntypedResourceBuilder[K]) WithMutator(f Mutator[*unstructured.Unstructured]) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithMutator(f)
	return b
}

func (b *UntypedResourceBuilder[K]) WithOutput(obj *unstructured.Unstructured) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithOutput(obj)
	return b
}

func (b *UntypedResourceBuilder[K]) WithReadinessCondition(f func(obj *unstructured.Unstructured) bool) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithReadinessCondition(f)
	return b
}

func (b *UntypedResourceBuilder[K]) WithRequireManualDeletionForFinalize(f func(obj *unstructured.Unstructured) bool) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithRequireManualDeletionForFinalize(f)
	return b
}

func (b *UntypedResourceBuilder[K]) WithSkipAndDeleteOnCondition(f func() bool) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithSkipAndDeleteOnCondition(f)
	return b
}

func (b *UntypedResourceBuilder[K]) WithUserIdentifier(identifier string) *UntypedResourceBuilder[K] {
	b.inner = b.inner.WithUserIdentifier(identifier)
	return b
}
