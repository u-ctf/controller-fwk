package instrument

import (
	"context"
	"reflect"
	"unsafe"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type InstrumentedController struct {
	internalController controller.TypedController[reconcile.Request]
	Instrumenter
}

var _ controller.TypedController[reconcile.Request] = &InstrumentedController{}

func NewTracerControllerFunc(t Instrumenter) func(name string, mgr manager.Manager, options controller.TypedOptions[reconcile.Request]) (controller.TypedController[reconcile.Request], error) {
	return func(name string, mgr manager.Manager, options controller.TypedOptions[reconcile.Request]) (controller.TypedController[reconcile.Request], error) {
		internal, err := controller.NewTyped(name, mgr, options)
		if err != nil {
			return nil, err
		}

		return &InstrumentedController{
			internalController: internal,
			Instrumenter:       t,
		}, nil
	}
}

func (t *InstrumentedController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	return t.internalController.Reconcile(ctx, req)
}

func (t *InstrumentedController) Watch(src source.TypedSource[reconcile.Request]) error {
	// Use reflect to check if src has a field named "Handler"
	v := reflect.ValueOf(src).Elem()
	hdlrField := v.FieldByName("Handler")
	if hdlrField.IsValid() && hdlrField.Type() == reflect.TypeOf((*handler.TypedEventHandler[client.Object, reconcile.Request])(nil)).Elem() {
		// Get the func to wrap it, it's exported since the name starts with a capital letter
		ptr := unsafe.Pointer(hdlrField.UnsafeAddr())
		realValue := reflect.NewAt(hdlrField.Type(), ptr).Elem()
		originalHandler := realValue.Interface().(handler.TypedEventHandler[client.Object, reconcile.Request])
		wrappedHandler := t.InstrumentRequestHandler(originalHandler)
		realValue.Set(reflect.ValueOf(wrappedHandler))
	}

	predicatesField := v.FieldByName("Predicates")
	if predicatesField.IsValid() && predicatesField.Type() == reflect.TypeOf([]predicate.Predicate{}) {
		// Get the func to wrap it, it's exported since the name starts with a capital letter
		ptr := unsafe.Pointer(predicatesField.UnsafeAddr())
		realValue := reflect.NewAt(predicatesField.Type(), ptr).Elem()
		originalPredicates := realValue.Interface().([]predicate.Predicate)
		wrappedPredicates := make([]predicate.Predicate, len(originalPredicates))
		for i, p := range originalPredicates {
			wrappedPredicates[i] = t.InstrumentPredicate(p)
		}
		realValue.Set(reflect.ValueOf(wrappedPredicates))
	}

	return t.internalController.Watch(src)
}

func (t *InstrumentedController) Start(ctx context.Context) error {
	return t.internalController.Start(ctx)
}

func (t *InstrumentedController) GetLogger() logr.Logger {
	return t.internalController.GetLogger()
}
