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

type baseContext[K client.Object] struct {
	context.Context
	CustomResource[K]
}

// NewContext creates a new Context for the given reconciler and base context.
// K is the type of the custom resource being reconciled.
// You can use it as such:
//
//	func (reconciler *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
//		logger := logf.FromContext(ctx)
//		context := ctrlfwk.NewContext(ctx, reconciler)
func NewContext[K client.Object](ctx context.Context, reconciler Reconciler[K]) Context[K] {
	return &baseContext[K]{
		Context:        ctx,
		CustomResource: CustomResource[K]{},
	}
}

var _ Context[*corev1.Secret] = &baseContext[*corev1.Secret]{}

// ContextWithData is a context that holds additional data of type D along with the base context.
// K is the type of the custom resource being reconciled.
// D is the type of the additional data to be stored in the context.
type ContextWithData[K client.Object, D any] struct {
	Context[K]
	Data D
}

// NewContextWithData creates a new ContextWithData for the given reconciler, base context, and data.
// K is the type of the custom resource being reconciled.
// D is the type of the additional data to be stored in the context.
// You can use it as such:
//
//	func (reconciler *TestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
//		logger := logf.FromContext(ctx)
//		context := ctrlfwk.NewContextWithData(ctx, reconciler, &MyDataType{})
func NewContextWithData[K client.Object, D any](ctx context.Context, reconciler Reconciler[K], data D) *ContextWithData[K, D] {
	return &ContextWithData[K, D]{
		Context: &baseContext[K]{Context: ctx},
		Data:    data,
	}
}
