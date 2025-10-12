package instrument

import (
	"reflect"
	"unsafe"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type InstrumentedBuilder struct {
	manager manager.Manager

	*builder.Builder
	Instrumenter
}

// forceSetNewController uses reflection and unsafe to set the unexported newController field of a
// controller.Builder. This is necessary because controller.Builder does not provide a way to set
// a custom controller constructor function.
func forceSetNewController[request comparable](obj *builder.TypedBuilder[request], fn func(name string, mgr manager.Manager, options controller.TypedOptions[request]) (controller.TypedController[reconcile.Request], error)) {
	v := reflect.ValueOf(obj).Elem()
	f := v.FieldByName("newController")

	if f.Type() != reflect.TypeOf(fn) {
		panic("field newController has unexpected type")
	}

	// Get pointer to the field even if it’s unexported
	ptr := unsafe.Pointer(f.UnsafeAddr())
	realValue := reflect.NewAt(f.Type(), ptr).Elem()
	realValue.Set(reflect.ValueOf(fn))
}

// InstrumentedControllerManagedBy returns a new controller builder that will be started by the provided Manager.
func InstrumentedControllerManagedBy(t Instrumenter, m manager.Manager) *InstrumentedBuilder {
	blder := builder.TypedControllerManagedBy[reconcile.Request](m)
	blder.WithOptions(controller.TypedOptions[reconcile.Request]{
		NewQueue: t.NewQueue(m),
	})
	forceSetNewController(blder, NewTracerControllerFunc(t))

	return &InstrumentedBuilder{
		Instrumenter: t,
		Builder:      blder,
		manager:      m,
	}
}

func (blder *InstrumentedBuilder) For(object client.Object, opts ...builder.ForOption) *InstrumentedBuilder {
	blder.Builder = blder.Builder.For(object, opts...)
	return blder
}

func (blder *InstrumentedBuilder) Build(r reconcile.TypedReconciler[reconcile.Request]) (controller.TypedController[reconcile.Request], error) {
	return blder.Builder.Build(NewInstrumentedReconciler(blder.Instrumenter, r))
}

func (blder *InstrumentedBuilder) Complete(r reconcile.TypedReconciler[reconcile.Request]) error {
	return blder.Builder.Complete(r)
}

func (blder *InstrumentedBuilder) Named(name string) *InstrumentedBuilder {
	blder.Builder = blder.Builder.Named(name)
	return blder
}

func (blder *InstrumentedBuilder) Owns(object client.Object, opts ...builder.OwnsOption) *InstrumentedBuilder {
	blder.Builder = blder.Builder.Owns(object, opts...)
	return blder
}

func (blder *InstrumentedBuilder) Watches(object client.Object, eventHandler handler.TypedEventHandler[client.Object, reconcile.Request], opts ...builder.WatchesOption) *InstrumentedBuilder {
	blder.Builder = blder.Builder.Watches(object, eventHandler, opts...)
	return blder
}

func (blder *InstrumentedBuilder) WatchesMetadata(object client.Object, eventHandler handler.TypedEventHandler[client.Object, reconcile.Request], opts ...builder.WatchesOption) *InstrumentedBuilder {
	blder.Builder = blder.Builder.WatchesMetadata(object, eventHandler, opts...)
	return blder
}

func (blder *InstrumentedBuilder) WatchesRawSource(src source.TypedSource[reconcile.Request]) *InstrumentedBuilder {
	blder.Builder = blder.Builder.WatchesRawSource(src)
	return blder
}

func (blder *InstrumentedBuilder) WithEventFilter(p predicate.Predicate) *InstrumentedBuilder {
	blder.Builder = blder.Builder.WithEventFilter(p)
	return blder
}

func (blder *InstrumentedBuilder) WithLogConstructor(logConstructor func(*reconcile.Request) logr.Logger) *InstrumentedBuilder {
	blder.Builder = blder.Builder.WithLogConstructor(logConstructor)
	return blder
}

func (blder *InstrumentedBuilder) WithOptions(options controller.TypedOptions[reconcile.Request]) *InstrumentedBuilder {
	options.NewQueue = blder.Instrumenter.NewQueue(blder.manager)
	blder.Builder = blder.Builder.WithOptions(options)
	return blder
}
