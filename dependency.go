package ctrlfwk

import (
	"context"
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

	// Hooks
	BeforeReconcile(ctx context.Context) error
	AfterReconcile(ctx context.Context, resource client.Object) error
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

	// Hooks
	beforeReconcileF func(ctx context.Context) error
	afterReconcileF  func(ctx context.Context, resource T) error
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

func DependencyWithOptional[T client.Object](_ T, optional bool) DependencyResourceOption[T] {
	return func(c *Dependency[T]) {
		c.isOptional = optional
	}
}

func DependencyWithName[T client.Object](_ T, name string) DependencyResourceOption[T] {
	return func(c *Dependency[T]) {
		c.name = name
	}
}

func DependencyWithNamespace[T client.Object](_ T, namespace string) DependencyResourceOption[T] {
	return func(c *Dependency[T]) {
		c.namespace = namespace
	}
}

func DependencyWithWaitForReady[T client.Object](_ T, waitForReady bool) DependencyResourceOption[T] {
	return func(c *Dependency[T]) {
		c.waitForReady = waitForReady
	}
}

func DependencyBeforeReconcile[T client.Object](_ T, f func(ctx context.Context) error) DependencyResourceOption[T] {
	return func(c *Dependency[T]) {
		c.beforeReconcileF = f
	}
}

func DependencyAfterReconcile[T client.Object](_ T, f func(ctx context.Context, resource T) error) DependencyResourceOption[T] {
	return func(c *Dependency[T]) {
		c.afterReconcileF = f
	}
}

func DependencyWithReadinessCondition[T client.Object](_ T, f func(obj T) bool) DependencyResourceOption[T] {
	return func(c *Dependency[T]) {
		c.isReadyF = f
	}
}

func NewDependency[T client.Object](_ T, opts ...DependencyResourceOption[T]) *Dependency[T] {
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

func (c *Dependency[T]) BeforeReconcile(ctx context.Context) error {
	if c.beforeReconcileF != nil {
		return c.beforeReconcileF(ctx)
	}
	return nil
}

func (c *Dependency[T]) AfterReconcile(ctx context.Context, resource client.Object) error {
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

type UntypedDependencyResource struct {
	*Dependency[*unstructured.Unstructured]
	gvk schema.GroupVersionKind
}

func NewUntypedDependencyResource(gvk schema.GroupVersionKind, opts ...DependencyResourceOption[*unstructured.Unstructured]) *UntypedDependencyResource {
	c := &UntypedDependencyResource{
		Dependency: NewDependency(&unstructured.Unstructured{}, opts...),
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
