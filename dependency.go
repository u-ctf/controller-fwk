package ctrlfwk

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GenericDependency[K client.Object] interface {
	ID() string
	New() client.Object
	Key() types.NamespacedName
	Set(obj client.Object)
	Get() client.Object
	ShouldWaitForReady() bool
	IsReady() bool
	IsOptional() bool
	Kind() string

	// Hooks
	BeforeReconcile(ctx Context[K]) error
	AfterReconcile(ctx Context[K], resource client.Object) error
}

var _ GenericDependency[client.Object] = &Dependency[client.Object, client.Object]{}

type Dependency[K, T client.Object] struct {
	userIdentifier string
	isReadyF       func(obj T) bool
	output         T
	isOptional     bool
	waitForReady   bool
	name           string
	namespace      string

	// Hooks
	beforeReconcileF func(ctx Context[K]) error
	afterReconcileF  func(ctx Context[K], resource T) error
}

func (c *Dependency[K, T]) New() client.Object {
	return NewInstanceOf(c.output)
}

func (c *Dependency[K, T]) Kind() string {
	return reflect.TypeOf(c.output).Elem().Name()
}

func (c *Dependency[K, T]) Set(obj client.Object) {
	if reflect.TypeOf(c.output) == reflect.TypeOf(obj) {
		if reflect.ValueOf(c.output).IsNil() {
			c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(T)
		}

		reflect.ValueOf(c.output).Elem().Set(reflect.ValueOf(obj).Elem())
	}
}

func (c *Dependency[K, T]) Get() client.Object {
	return c.output
}

func (c *Dependency[K, T]) IsOptional() bool {
	return c.isOptional
}

func (c *Dependency[K, T]) Key() types.NamespacedName {
	return types.NamespacedName{
		Name:      c.name,
		Namespace: c.namespace,
	}
}

func (c *Dependency[K, T]) ID() string {
	if c.userIdentifier != "" {
		return c.userIdentifier
	}
	return fmt.Sprintf("%v,%v", c.Kind(), c.Key())
}

func (c *Dependency[K, T]) ShouldWaitForReady() bool {
	return c.waitForReady
}

func (c *Dependency[K, T]) IsReady() bool {
	if c.isReadyF != nil {
		return c.isReadyF(c.output)
	}
	return false
}

func (c *Dependency[K, T]) BeforeReconcile(ctx Context[K]) error {
	if c.beforeReconcileF != nil {
		return c.beforeReconcileF(ctx)
	}
	return nil
}

func (c *Dependency[K, T]) AfterReconcile(ctx Context[K], resource client.Object) error {
	if c.afterReconcileF != nil {
		switch typedObj := resource.(type) {
		case T:
			return c.afterReconcileF(ctx, typedObj)
		default:
			var zero T
			return c.afterReconcileF(ctx, zero)
		}
	}
	return nil
}

// type UntypedDependencyResource[K client.Object] struct {
// 	*Dependency[K, *unstructured.Unstructured]
// 	gvk schema.GroupVersionKind
// }

// func NewUntypedDependencyResource[K client.Object](gvk schema.GroupVersionKind, opts ...DependencyResourceOption[K, *unstructured.Unstructured]) *UntypedDependencyResource[K] {
// 	c := &UntypedDependencyResource[K]{
// 		Dependency: NewDependency(&unstructured.Unstructured{}, opts...),
// 		gvk:        gvk,
// 	}

// 	return c
// }

// func (c *UntypedDependencyResource[K]) New() client.Object {
// 	obj := NewInstanceOf(c.output)
// 	obj.SetAPIVersion(c.gvk.GroupVersion().String())
// 	obj.SetKind(c.gvk.Kind)
// 	return obj
// }

// func (c *UntypedDependencyResource[K]) Kind() string {
// 	return c.gvk.Kind
// }
