package instrument

import (
	"context"
	"errors"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/yyewolf/controller-fwk/mocks"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestNewInstrumentedReconciler(t *testing.T) {
	ctrlr := gomock.NewController(t)
	defer ctrlr.Finish()

	mockInstrumenter := mocks.NewMockInstrumenter(ctrlr)
	mockReconciler := mocks.NewMockTypedReconciler[reconcile.Request](ctrlr)

	instrumentedReconciler := NewInstrumentedReconciler(mockInstrumenter, mockReconciler)

	if instrumentedReconciler.Instrumenter != mockInstrumenter {
		t.Errorf("expected instrumenter to be set correctly")
	}

	if instrumentedReconciler.internalReconciler != mockReconciler {
		t.Errorf("expected internal reconciler to be set correctly")
	}
}

func TestInstrumentedReconciler_Reconcile_Success(t *testing.T) {
	ctrlr := gomock.NewController(t)
	defer ctrlr.Finish()

	mockInstrumenter := mocks.NewMockInstrumenter(ctrlr)
	mockReconciler := mocks.NewMockTypedReconciler[reconcile.Request](ctrlr)

	instrumentedReconciler := NewInstrumentedReconciler(mockInstrumenter, mockReconciler)

	ctx := context.Background()
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "test-name",
		},
	}

	expectedResult := reconcile.Result{Requeue: false}
	hub := sentry.CurrentHub().Clone()

	// Set up expectations
	mockInstrumenter.EXPECT().GetSentryHubForRequest(req).Return(hub, true)
	mockReconciler.EXPECT().Reconcile(gomock.Any(), req).Return(expectedResult, nil)
	mockInstrumenter.EXPECT().Cleanup(req)

	// Test the Reconcile method
	result, err := instrumentedReconciler.Reconcile(ctx, req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result != expectedResult {
		t.Errorf("expected result %+v, got %+v", expectedResult, result)
	}
}

func TestInstrumentedReconciler_Reconcile_WithError(t *testing.T) {
	ctrlr := gomock.NewController(t)
	defer ctrlr.Finish()

	mockInstrumenter := mocks.NewMockInstrumenter(ctrlr)
	mockReconciler := mocks.NewMockTypedReconciler[reconcile.Request](ctrlr)

	instrumentedReconciler := NewInstrumentedReconciler(mockInstrumenter, mockReconciler)

	ctx := context.Background()
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "test-name",
		},
	}

	expectedResult := reconcile.Result{Requeue: true}
	expectedError := errors.New("reconcile error")
	hub := sentry.CurrentHub().Clone()

	// Set up expectations
	mockInstrumenter.EXPECT().GetSentryHubForRequest(req).Return(hub, true)
	mockReconciler.EXPECT().Reconcile(gomock.Any(), req).Return(expectedResult, expectedError)
	mockInstrumenter.EXPECT().Cleanup(req)

	// Test the Reconcile method with error
	result, err := instrumentedReconciler.Reconcile(ctx, req)

	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}

	if result != expectedResult {
		t.Errorf("expected result %+v, got %+v", expectedResult, result)
	}
}
