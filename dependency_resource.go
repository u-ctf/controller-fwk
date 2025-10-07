package library

import (
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GenericDependencyResource interface {
	New() client.Object
	Key() types.NamespacedName
	Set(obj client.Object)
	Get() client.Object
	Status(obj client.Object) *Status
	ShouldWaitForReady() bool
	IsOptional() bool
	Kind() string
}

var _ GenericDependencyResource = &DependencyResource[client.Object]{}

type DependencyResource[T client.Object] struct {
	statusGetter func(T) *Status
	output       T
	isOptional   bool
	waitForReady bool
	name         string
	namespace    string
}

type DependencyResourceOption[T client.Object] func(*DependencyResource[T])

func WithDependencyStatusGetter[T client.Object](f func(T) *Status) DependencyResourceOption[T] {
	return func(c *DependencyResource[T]) {
		c.statusGetter = f
	}
}

func WithOutput[T client.Object](obj T) DependencyResourceOption[T] {
	return func(c *DependencyResource[T]) {
		c.output = obj
	}
}

func WithOptional[T client.Object](optional bool) DependencyResourceOption[T] {
	return func(c *DependencyResource[T]) {
		c.isOptional = optional
	}
}

func WithName[T client.Object](name string) DependencyResourceOption[T] {
	return func(c *DependencyResource[T]) {
		c.name = name
	}
}

func WithNamespace[T client.Object](namespace string) DependencyResourceOption[T] {
	return func(c *DependencyResource[T]) {
		c.namespace = namespace
	}
}

func WithWaitForReady[T client.Object](waitForReady bool) DependencyResourceOption[T] {
	return func(c *DependencyResource[T]) {
		c.waitForReady = waitForReady
	}
}

func NewDependencyResource[T client.Object](_ T, opts ...DependencyResourceOption[T]) *DependencyResource[T] {
	c := &DependencyResource[T]{
		statusGetter: DefaultStatusGetter[T],
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *DependencyResource[T]) New() client.Object {
	return NewInstanceOf(c.output)
}

func (c *DependencyResource[T]) Kind() string {
	return reflect.TypeOf(c.output).Elem().Name()
}

func (c *DependencyResource[T]) Set(obj client.Object) {
	if reflect.TypeOf(c.output) == reflect.TypeOf(obj) {
		if reflect.ValueOf(c.output).IsNil() {
			c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(T)
		}

		reflect.ValueOf(c.output).Elem().Set(reflect.ValueOf(obj).Elem())
	}
}

func (c *DependencyResource[T]) Get() client.Object {
	return c.output
}

func (c *DependencyResource[T]) Status(obj client.Object) *Status {
	return c.statusGetter(obj.(T))
}

func (c *DependencyResource[T]) IsOptional() bool {
	return c.isOptional
}

func (c *DependencyResource[T]) Key() types.NamespacedName {
	return types.NamespacedName{
		Name:      c.name,
		Namespace: c.namespace,
	}
}

func (c *DependencyResource[T]) ShouldWaitForReady() bool {
	return c.waitForReady
}

type UntypedDependencyResource struct {
	*DependencyResource[*unstructured.Unstructured]
	gvk schema.GroupVersionKind
}

func NewUntypedDependencyResource(gvk schema.GroupVersionKind, opts ...DependencyResourceOption[*unstructured.Unstructured]) *UntypedDependencyResource {
	c := &UntypedDependencyResource{
		DependencyResource: NewDependencyResource(&unstructured.Unstructured{}, opts...),
		gvk:                gvk,
	}

	return c
}

func (c *UntypedDependencyResource) New() client.Object {
	obj := NewInstanceOf(c.output)
	obj.SetAPIVersion(c.gvk.GroupVersion().String())
	obj.SetKind(c.gvk.Kind)
	return obj
}
