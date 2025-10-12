package instrument

import (
	"context"

	"github.com/getsentry/sentry-go"
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
	hub, _ := t.GetSentryHubForRequest(req)
	ctx = sentry.SetHubOnContext(ctx, hub)

	tx := sentry.StartTransaction(ctx, "reconcile")
	defer tx.Finish()

	result, err := t.internalReconciler.Reconcile(ctx, req)
	if err != nil {
		hub.CaptureException(err)
	}

	t.Cleanup(req)

	return result, err
}
