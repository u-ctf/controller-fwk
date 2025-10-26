package ctrlfwk

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GenericDependency interface {
	ID() string
	New() client.Object
	Key() types.NamespacedName
	Set(obj client.Object)
	Get() client.Object
	ShouldWaitForReady() bool
	IsReady() bool
	IsOptional() bool
	Kind() string
}

var _ GenericDependency = &Dependency[client.Object]{}

type Dependency[T client.Object] struct {
	userIdentifier string
	isReadyF       func(obj T) bool
	output         T
	isOptional     bool
	waitForReady   bool
	name           string
	namespace      string
}

type DependencyResourceOption[T client.Object] func(*Dependency[T])

func DependencyWithOutput[T client.Object](obj T) DependencyResourceOption[T] {
	return func(c *Dependency[T]) {
		c.output = obj
	}
}

func DependencyWithIsReadyFunc[T client.Object](f func(obj T) bool) DependencyResourceOption[T] {
	return func(c *Dependency[T]) {
		c.isReadyF = f
	}
}

func DependencyWithOptional[T client.Object](optional bool) DependencyResourceOption[T] {
	return func(c *Dependency[T]) {
		c.isOptional = optional
	}
}

func DependencyWithName[T client.Object](name string) DependencyResourceOption[T] {
	return func(c *Dependency[T]) {
		c.name = name
	}
}

func DependencyWithNamespace[T client.Object](namespace string) DependencyResourceOption[T] {
	return func(c *Dependency[T]) {
		c.namespace = namespace
	}
}

func DependencyWithWaitForReady[T client.Object](waitForReady bool) DependencyResourceOption[T] {
	return func(c *Dependency[T]) {
		c.waitForReady = waitForReady
	}
}

func NewDependencyResource[T client.Object](_ T, opts ...DependencyResourceOption[T]) *Dependency[T] {
	c := &Dependency[T]{}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Dependency[T]) New() client.Object {
	return NewInstanceOf(c.output)
}

func (c *Dependency[T]) Kind() string {
	return reflect.TypeOf(c.output).Elem().Name()
}

func (c *Dependency[T]) Set(obj client.Object) {
	if reflect.TypeOf(c.output) == reflect.TypeOf(obj) {
		if reflect.ValueOf(c.output).IsNil() {
			c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(T)
		}

		reflect.ValueOf(c.output).Elem().Set(reflect.ValueOf(obj).Elem())
	}
}

func (c *Dependency[T]) Get() client.Object {
	return c.output
}

func (c *Dependency[T]) IsOptional() bool {
	return c.isOptional
}

func (c *Dependency[T]) Key() types.NamespacedName {
	return types.NamespacedName{
		Name:      c.name,
		Namespace: c.namespace,
	}
}

func (c *Dependency[T]) ID() string {
	if c.userIdentifier != "" {
		return c.userIdentifier
	}
	return fmt.Sprintf("%v,%v", c.Kind(), c.Key())
}

func (c *Dependency[T]) ShouldWaitForReady() bool {
	return c.waitForReady
}

func (c *Dependency[T]) IsReady() bool {
	if c.isReadyF != nil {
		return c.isReadyF(c.output)
	}
	return false
}

type UntypedDependencyResource struct {
	*Dependency[*unstructured.Unstructured]
	gvk schema.GroupVersionKind
}

func NewUntypedDependencyResource(gvk schema.GroupVersionKind, opts ...DependencyResourceOption[*unstructured.Unstructured]) *UntypedDependencyResource {
	c := &UntypedDependencyResource{
		Dependency: NewDependencyResource(&unstructured.Unstructured{}, opts...),
		gvk:        gvk,
	}

	return c
}

func (c *UntypedDependencyResource) New() client.Object {
	obj := NewInstanceOf(c.output)
	obj.SetAPIVersion(c.gvk.GroupVersion().String())
	obj.SetKind(c.gvk.Kind)
	return obj
}

func (c *UntypedDependencyResource) Kind() string {
	return c.gvk.Kind
}
