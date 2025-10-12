package instrument

import (
	"testing"
	"weak"

	"github.com/getsentry/sentry-go"
	"go.uber.org/mock/gomock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockPredicate struct {
	createResult  bool
	updateResult  bool
	deleteResult  bool
	genericResult bool
}

func (m *mockPredicate) Create(event.TypedCreateEvent[client.Object]) bool {
	return m.createResult
}

func (m *mockPredicate) Update(event.TypedUpdateEvent[client.Object]) bool {
	return m.updateResult
}

func (m *mockPredicate) Delete(event.TypedDeleteEvent[client.Object]) bool {
	return m.deleteResult
}

func (m *mockPredicate) Generic(event.TypedGenericEvent[client.Object]) bool {
	return m.genericResult
}

func TestNewInstrumentedPredicate(t *testing.T) {
	ctrlr := gomock.NewController(t)
	defer ctrlr.Finish()

	innerPredicate := &mockPredicate{createResult: true}

	// Cast to *instrumenter to access the concrete type
	tracer := &instrumenter{}
	instrumentedPred := NewInstrumentedPredicate(tracer, innerPredicate)

	// Verify the type
	ip, ok := instrumentedPred.(*instrumentedPredicate)
	if !ok {
		t.Fatalf("expected *instrumentedPredicate, got %T", instrumentedPred)
	}

	if ip.inner != innerPredicate {
		t.Errorf("expected inner predicate to be set correctly")
	}

	if ip.tracer != tracer {
		t.Errorf("expected tracer to be set correctly")
	}

	expectedName := "*instrument.mockPredicate"
	if ip.innerName != expectedName {
		t.Errorf("expected inner name to be %q, got %q", expectedName, ip.innerName)
	}
}

func TestInstrumentedPredicate_Create(t *testing.T) {
	ctrlr := gomock.NewController(t)
	defer ctrlr.Finish()

	tracer := &instrumenter{
		hubCache: map[string]weak.Pointer[sentry.Hub]{},
	}
	innerPredicate := &mockPredicate{createResult: true}

	ip := &instrumentedPredicate{
		tracer:    tracer,
		inner:     innerPredicate,
		innerName: "mockPredicate",
	}

	// Create a test event
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	createEvent := event.TypedCreateEvent[client.Object]{
		Object: pod,
	}

	// Test that the result matches the inner predicate
	result := ip.Create(createEvent)

	if result != innerPredicate.createResult {
		t.Errorf("expected result %v, got %v", innerPredicate.createResult, result)
	}
}
