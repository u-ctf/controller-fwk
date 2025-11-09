package ctrlfwk

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mutator[T client.Object] func(resource T) error

type GenericResource[K client.Object] interface {
	ID() string
	ObjectMetaGenerator() (obj client.Object, delete bool, err error)
	ShouldDeleteNow() bool
	GetMutator(obj client.Object) func() error
	Set(obj client.Object)
	Get() client.Object
	Kind() string
	IsReady(obj client.Object) bool
	RequiresManualDeletion(obj client.Object) bool

	// Hooks
	BeforeReconcile(ctx Context[K]) error
	AfterReconcile(ctx Context[K], resource client.Object) error
	OnCreate(ctx Context[K], resource client.Object) error
	OnUpdate(ctx Context[K], resource client.Object) error
	OnDelete(ctx Context[K], resource client.Object) error
	OnFinalize(ctx Context[K], resource client.Object) error
}

var _ GenericResource[client.Object] = &Resource[client.Object, client.Object]{}

type Resource[K, T client.Object] struct {
	userIdentifier string
	keyF           func() types.NamespacedName
	mutateF        Mutator[T]

	isReadyF          func(obj T) bool
	shouldDeleteF     func() bool
	requiresDeletionF func(obj T) bool
	output            T

	// Hooks
	beforeReconcileF func(ctx Context[K]) error
	afterReconcileF  func(ctx Context[K], resource T) error
	onCreateF        func(ctx Context[K], resource T) error
	onUpdateF        func(ctx Context[K], resource T) error
	onDeleteF        func(ctx Context[K], resource T) error
	onFinalizeF      func(ctx Context[K], resource T) error
}

func (c *Resource[K, T]) Kind() string {
	return reflect.TypeOf(c.output).Elem().Name()
}

func (c *Resource[K, T]) ObjectMetaGenerator() (obj client.Object, skip bool, err error) {
	if reflect.ValueOf(c.output).IsNil() {
		c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(T)
	}

	key := c.keyF()

	c.output.SetName(key.Name)
	c.output.SetNamespace(key.Namespace)

	return c.output, c.shouldDeleteF != nil && c.shouldDeleteF(), nil
}

func (c *Resource[K, T]) ID() string {
	if c.userIdentifier != "" {
		return c.userIdentifier
	}

	key := c.keyF()

	return fmt.Sprintf("%v,%v", c.Kind(), key)
}

func (c *Resource[K, T]) Set(obj client.Object) {
	if reflect.TypeOf(c.output) == reflect.TypeOf(obj) {
		if reflect.ValueOf(c.output).IsNil() {
			c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(T)
		}

		reflect.ValueOf(c.output).Elem().Set(reflect.ValueOf(obj).Elem())
	}
}

func (c *Resource[K, T]) Get() client.Object {
	return c.output
}

func (c *Resource[K, T]) IsReady(obj client.Object) bool {
	if c.isReadyF != nil {
		if typedObj, ok := obj.(T); ok {
			return c.isReadyF(typedObj)
		}
		if obj == nil {
			var zero T
			return c.isReadyF(zero)
		}
	}
	return false
}

func (c *Resource[K, T]) RequiresManualDeletion(obj client.Object) bool {
	if c.requiresDeletionF != nil {
		if typedObj, ok := obj.(T); ok {
			return c.requiresDeletionF(typedObj)
		}
		if obj == nil {
			var zero T
			return c.requiresDeletionF(zero)
		}
	}
	return false
}

func (c *Resource[K, T]) ShouldDeleteNow() bool {
	if c.shouldDeleteF != nil {
		return c.shouldDeleteF()
	}
	return false
}

func (c *Resource[K, T]) BeforeReconcile(ctx Context[K]) error {
	if c.beforeReconcileF != nil {
		return c.beforeReconcileF(ctx)
	}
	return nil
}

func (c *Resource[K, T]) AfterReconcile(ctx Context[K], resource client.Object) error {
	if c.afterReconcileF != nil {
		if typedObj, ok := resource.(T); ok {
			return c.afterReconcileF(ctx, typedObj)
		}
		if resource == nil {
			var zero T
			return c.afterReconcileF(ctx, zero)
		}
	}
	return nil
}

func (c *Resource[K, T]) OnCreate(ctx Context[K], resource client.Object) error {
	if c.onCreateF != nil {
		if typedObj, ok := resource.(T); ok {
			return c.onCreateF(ctx, typedObj)
		}
		if resource == nil {
			var zero T
			return c.onCreateF(ctx, zero)
		}
	}
	return nil
}

func (c *Resource[K, T]) OnUpdate(ctx Context[K], resource client.Object) error {
	if c.onUpdateF != nil {
		if typedObj, ok := resource.(T); ok {
			return c.onUpdateF(ctx, typedObj)
		}
		if resource == nil {
			var zero T
			return c.onUpdateF(ctx, zero)
		}
	}
	return nil
}

func (c *Resource[K, T]) OnDelete(ctx Context[K], resource client.Object) error {
	if c.onDeleteF != nil {
		if typedObj, ok := resource.(T); ok {
			return c.onDeleteF(ctx, typedObj)
		}
		if resource == nil {
			var zero T
			return c.onDeleteF(ctx, zero)
		}
	}
	return nil
}

func (c *Resource[K, T]) OnFinalize(ctx Context[K], resource client.Object) error {
	if c.onFinalizeF != nil {
		if typedObj, ok := resource.(T); ok {
			return c.onFinalizeF(ctx, typedObj)
		}
		if resource == nil {
			var zero T
			return c.onFinalizeF(ctx, zero)
		}
	}
	return nil
}

func (c *Resource[K, T]) GetMutator(obj client.Object) func() error {
	return func() error {
		if c.mutateF != nil {
			if typedObj, ok := obj.(T); ok {
				return c.mutateF(typedObj)
			}
			if obj == nil {
				var zero T
				return c.mutateF(zero)
			}
		}
		return nil
	}
}
