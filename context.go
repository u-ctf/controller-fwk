package ctrlfwk

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

type Context[K client.Object] interface {
	context.Context

	ImplementsCustomResource[K]
}

type ContextWithData[K client.Object, D any] interface {
	Context[K]
	Data() D
}

type baseContext[K client.Object] struct {
	context.Context
	CustomResource[K]
}

func NewContext[K client.Object](ctx context.Context) Context[K] {
	return &baseContext[K]{
		Context:        ctx,
		CustomResource: CustomResource[K]{},
	}
}

var _ Context[*corev1.Secret] = &baseContext[*corev1.Secret]{}

type contextWithData[K client.Object, D any] struct {
	Context[K]
	data D
}

var _ ContextWithData[*corev1.Secret, int] = &contextWithData[*corev1.Secret, int]{}

func (c *contextWithData[K, D]) Data() D {
	return c.data
}

func NewContextWithData[K client.Object, D any](ctx context.Context, data D) ContextWithData[K, D] {
	return &contextWithData[K, D]{
		Context: &baseContext[K]{Context: ctx},
		data:    data,
	}
}

func ToContextWithData[D any, K client.Object](ctx Context[K]) ContextWithData[K, D] {
	return ctx.(ContextWithData[K, D])
}
