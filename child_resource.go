package library

import (
	"context"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GenericChildResource interface {
	Generator(ctx context.Context, req ctrl.Request) (obj client.Object, skip bool, err error)
	Set(obj client.Object)
	Get() client.Object
	Status(obj client.Object) *Status
	Kind() string
}

var _ GenericChildResource = &ChildResource[client.Object]{}

type ChildResource[T client.Object] struct {
	generatorF   ChildGenerator[T]
	statusGetter func(T) *Status
	output       T
}

type ChildResourceOption[T client.Object] func(*ChildResource[T])

func WithChildGenerator[T client.Object](f ChildGenerator[T]) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.generatorF = f
	}
}

func WithChildStatusGetter[T client.Object](f func(T) *Status) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.statusGetter = f
	}
}

func WithChildOutput[T client.Object](obj T) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.output = obj
	}
}

func DefaultStatusGetter[T client.Object](obj T) *Status {
	return &Status{
		Conditions: []metav1.Condition{
			{
				Type:   ConditionTypeReady,
				Status: metav1.ConditionTrue,
			},
		},
	}
}

func NewChildResource[T client.Object](_ T, opts ...ChildResourceOption[T]) *ChildResource[T] {
	c := &ChildResource[T]{
		statusGetter: DefaultStatusGetter[T],
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *ChildResource[T]) Kind() string {
	return reflect.TypeOf(c.output).Elem().Name()
}

func (c *ChildResource[T]) Generator(ctx context.Context, req ctrl.Request) (obj client.Object, skip bool, err error) {
	return c.generatorF(ctx, req)
}

func (c *ChildResource[T]) Set(obj client.Object) {
	if reflect.TypeOf(c.output) == reflect.TypeOf(obj) {
		if reflect.ValueOf(c.output).IsNil() {
			c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(T)
		}

		reflect.ValueOf(c.output).Elem().Set(reflect.ValueOf(obj).Elem())
	}
}

func (c *ChildResource[T]) Get() client.Object {
	return c.output
}

func (c *ChildResource[T]) Status(obj client.Object) *Status {
	return c.statusGetter(obj.(T))
}
