package instrument

import (
	"context"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type InstrumentedReconciler struct {
	internalReconciler reconcile.TypedReconciler[reconcile.Request]
	Instrumenter
}

var _ reconcile.TypedReconciler[reconcile.Request] = &InstrumentedReconciler{}

func NewInstrumentedReconciler(t Instrumenter, r reconcile.TypedReconciler[reconcile.Request]) *InstrumentedReconciler {
	return &InstrumentedReconciler{
		internalReconciler: r,
		Instrumenter:       t,
	}
}

func (t *InstrumentedReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	ctxPtr, _ := t.GetContextForRequest(req)
	ctx, span := t.StartSpan(ctxPtr, ctx, "reconcile")
	defer span.End()

	logger := t.NewLogger(ctx)
	ctx = logf.IntoContext(ctx, logger)

	result, err := t.internalReconciler.Reconcile(ctx, req)

	t.Cleanup(ctxPtr, req)

	return result, err
}
