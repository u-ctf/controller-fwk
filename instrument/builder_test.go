package instrument

import (
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"github.com/u-ctf/controller-fwk/mocks"
	"go.uber.org/mock/gomock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuilder_ShouldReplaceController(t *testing.T) {
	ctrlr := gomock.NewController(t)
	defer ctrlr.Finish()

	mockController := mocks.NewMockTypedController[reconcile.Request](ctrlr)
	mockController.EXPECT().Start(gomock.Any()).Return(nil).AnyTimes()
	mockController.EXPECT().Watch(gomock.Any()).Return(nil).AnyTimes()
	mockController.EXPECT().GetLogger().Return(logr.Discard()).AnyTimes()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
	if err != nil {
		t.Fatalf("unable to create manager: %v", err)
	}

	blder := builder.TypedControllerManagedBy[reconcile.Request](mgr).
		Named("oui").
		For(&corev1.Pod{})

	var worked bool
	fn := func(name string, mgr manager.Manager, options controller.TypedOptions[reconcile.Request]) (controller.TypedController[reconcile.Request], error) {
		worked = true
		return mockController, nil
	}

	forceSetNewController(blder, fn)

	_, err = blder.Build(mocks.NewMockTypedReconciler[reconcile.Request](ctrlr))
	if err != nil {
		t.Fatalf("unexpected error from Build: %v", err)
	}

	if !worked {
		t.Errorf("expected the newController function to be replaced")
	}
}

func TestBuilder_ShouldReplaceWatches(t *testing.T) {
	ctrlr := gomock.NewController(t)
	defer ctrlr.Finish()

	mockController := mocks.NewMockTypedController[reconcile.Request](ctrlr)
	mockController.EXPECT().Start(gomock.Any()).Return(nil).AnyTimes()
	mockController.EXPECT().GetLogger().Return(logr.Discard()).AnyTimes()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
	if err != nil {
		t.Fatalf("unable to create manager: %v", err)
	}

	mockedInstrumenter := mocks.NewMockInstrumenter(ctrlr)
	mockedInstrumenter.EXPECT().GetContextForEvent(gomock.Any()).Return(nil).AnyTimes()
	mockedInstrumenter.EXPECT().GetContextForRequest(gomock.Any()).Return(nil, false).AnyTimes()
	mockedInstrumenter.EXPECT().NewQueue(gomock.Any()).Return(nil).AnyTimes()

	var hdler handler.TypedEventHandler[client.Object, reconcile.Request]
	reflect.ValueOf(&hdler).Elem().Set(reflect.ValueOf(&handler.EnqueueRequestForObject{}))
	mockedInstrumenter.EXPECT().InstrumentRequestHandler(gomock.Any()).Return(NewInstrumentedEventHandler(mockedInstrumenter, hdler)).Times(2)

	var newController = func(name string, mgr manager.Manager, options controller.TypedOptions[reconcile.Request]) (controller.TypedController[reconcile.Request], error) {
		return &InstrumentedController{
			internalController: mockController,
			Instrumenter:       mockedInstrumenter,
		}, nil
	}

	blder := InstrumentedControllerManagedBy(mockedInstrumenter, mgr)
	forceSetNewController(blder.Builder, newController)

	ls, _ := predicate.LabelSelectorPredicate(v1.LabelSelector{
		MatchLabels: map[string]string{
			"app": "my-app",
		},
	})

	blder = blder.
		Named("oui").
		For(&corev1.Pod{}).
		Owns(&corev1.Secret{}, builder.WithPredicates(ls))

	gomock.InOrder(
		mockController.EXPECT().Watch(gomock.Any()).Do(func(src source.TypedSource[reconcile.Request]) {
			// verify that the handler was wrapped
			v := reflect.ValueOf(src).Elem()
			hdlrField := v.FieldByName("Handler")
			if !hdlrField.IsValid() || hdlrField.Elem().Type() != reflect.TypeOf((*instrumentedEventHandler[client.Object])(nil)) {
				t.Errorf("expected the Handler field to be of type *instrument.instrumentedEventHandler[sigs.k8s.io/controller-runtime/pkg/client.Object], got %v", hdlrField.Elem().Type())
			}

			predicatesField := v.FieldByName("Predicates")
			if !predicatesField.IsValid() || predicatesField.Len() != 0 {
				t.Errorf("expected the Predicates field to be empty, got %v", predicatesField.Len())
			}
		}).Return(nil),
		mockController.EXPECT().Watch(gomock.Any()).Do(func(src source.TypedSource[reconcile.Request]) {
			// verify that the handler was wrapped
			v := reflect.ValueOf(src).Elem()
			hdlrField := v.FieldByName("Handler")
			if !hdlrField.IsValid() || hdlrField.Elem().Type() != reflect.TypeOf((*instrumentedEventHandler[client.Object])(nil)) {
				t.Errorf("expected the Handler field to be of type *instrument.instrumentedEventHandler[sigs.k8s.io/controller-runtime/pkg/client.Object], got %v", hdlrField.Elem().Type())
			}
		}).Return(nil),
	)

	_, err = blder.Build(mocks.NewMockTypedReconciler[reconcile.Request](ctrlr))
	if err != nil {
		t.Fatalf("unexpected error from Build: %v", err)
	}
}
