package ctrlfwk

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UntypedResourceBuilder provides a builder pattern for creating Resource[K, *unstructured.Unstructured] instances
type UntypedResourceBuilder[CustomResource client.Object, ContextType Context[CustomResource]] struct {
	inner *ResourceBuilder[CustomResource, ContextType, *unstructured.Unstructured]
	gvk   schema.GroupVersionKind
}

// NewUntypedResourceBuilder creates a new UntypedResourceBuilder for *unstructured.Unstructuredhe given *unstructured.Unstructuredype
func NewUntypedResourceBuilder[CustomResource client.Object, ContextType Context[CustomResource]](ctx ContextType, gvk schema.GroupVersionKind) *UntypedResourceBuilder[CustomResource, ContextType] {
	return &UntypedResourceBuilder[CustomResource, ContextType]{
		inner: NewResourceBuilder(ctx, &unstructured.Unstructured{}),
		gvk:   gvk,
	}
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) Build() *UntypedResource[CustomResource, ContextType] {
	return &UntypedResource[CustomResource, ContextType]{
		Resource: b.inner.Build(),
		gvk:      b.gvk,
	}
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithAfterCreate(f func(ctx ContextType, resource *unstructured.Unstructured) error) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithAfterCreate(f)
	return b
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithAfterDelete(f func(ctx ContextType, resource *unstructured.Unstructured) error) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithAfterDelete(f)
	return b
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithAfterFinalize(f func(ctx ContextType, resource *unstructured.Unstructured) error) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithAfterFinalize(f)
	return b
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithAfterReconcile(f func(ctx ContextType, resource *unstructured.Unstructured) error) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithAfterReconcile(f)
	return b
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithAfterUpdate(f func(ctx ContextType, resource *unstructured.Unstructured) error) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithAfterUpdate(f)
	return b
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithBeforeReconcile(f func(ctx ContextType) error) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithBeforeReconcile(f)
	return b
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithKey(name types.NamespacedName) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithKey(name)
	return b
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithKeyFunc(f func() types.NamespacedName) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithKeyFunc(f)
	return b
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithMutator(f Mutator[*unstructured.Unstructured]) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithMutator(f)
	return b
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithOutput(obj *unstructured.Unstructured) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithOutput(obj)
	return b
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithReadinessCondition(f func(obj *unstructured.Unstructured) bool) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithReadinessCondition(f)
	return b
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithRequireManualDeletionForFinalize(f func(obj *unstructured.Unstructured) bool) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithRequireManualDeletionForFinalize(f)
	return b
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithSkipAndDeleteOnCondition(f func() bool) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithSkipAndDeleteOnCondition(f)
	return b
}

func (b *UntypedResourceBuilder[CustomResource, ContextType]) WithUserIdentifier(identifier string) *UntypedResourceBuilder[CustomResource, ContextType] {
	b.inner = b.inner.WithUserIdentifier(identifier)
	return b
}
