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

type ChildMutator[ChildType client.Object] func(child ChildType) error

type GenericChildResource interface {
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
	OnReconcile(ctx context.Context) error
	OnCreate(ctx context.Context, child client.Object) error
	OnUpdate(ctx context.Context, child client.Object) error
	OnDelete(ctx context.Context, child client.Object) error
	OnFinalize(ctx context.Context, child client.Object) error
}

var _ GenericChildResource = &ChildResource[client.Object]{}

type ChildResource[T client.Object] struct {
	userIdentifier string
	keyF           func() types.NamespacedName
	mutateF        ChildMutator[T]

	isReadyF          func(obj T) bool
	shouldDeleteF     func() bool
	requiresDeletionF func(obj T) bool
	output            T

	// Hooks
	onReconcileF func(ctx context.Context) error
	onCreateF    func(ctx context.Context, child T) error
	onUpdateF    func(ctx context.Context, child T) error
	onDeleteF    func(ctx context.Context, child T) error
	onFinalizeF  func(ctx context.Context, child T) error
}

type ChildResourceOption[T client.Object] func(*ChildResource[T])

func WithChildKey[T client.Object](_ T, name types.NamespacedName) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.keyF = func() types.NamespacedName {
			return name
		}
	}
}

func WithChildKeyFunc[T client.Object](_ T, f func() types.NamespacedName) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.keyF = f
	}
}

func WithChildMutator[T client.Object](f ChildMutator[T]) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.mutateF = f
	}
}

func WithChildOutput[T client.Object](obj T) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.output = obj
	}
}

func WithChildReadyCheck[T client.Object](f func(obj T) bool) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.isReadyF = f
	}
}

func WithChildShouldDelete[T client.Object](_ T, f func() bool) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.shouldDeleteF = f
	}
}

func WithChildDeletionCheck[T client.Object](f func(obj T) bool) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.requiresDeletionF = f
	}
}

func WithChildOnReconcile[T client.Object](_ T, f func(ctx context.Context) error) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.onReconcileF = f
	}
}

func WithChildOnCreate[T client.Object](f func(ctx context.Context, child T) error) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.onCreateF = f
	}
}

func WithChildOnUpdate[T client.Object](f func(ctx context.Context, child T) error) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.onUpdateF = f
	}
}

func WithChildOnDelete[T client.Object](f func(ctx context.Context, child T) error) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.onDeleteF = f
	}
}

func WithChildOnFinalize[T client.Object](f func(ctx context.Context, child T) error) ChildResourceOption[T] {
	return func(c *ChildResource[T]) {
		c.onFinalizeF = f
	}
}

func NewChildResource[T client.Object](_ T, opts ...ChildResourceOption[T]) *ChildResource[T] {
	c := &ChildResource[T]{}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *ChildResource[T]) Kind() string {
	return reflect.TypeOf(c.output).Elem().Name()
}

func (c *ChildResource[T]) ObjectMetaGenerator() (obj client.Object, skip bool, err error) {
	if reflect.ValueOf(c.output).IsNil() {
		c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(T)
	}

	key := c.keyF()

	c.output.SetName(key.Name)
	c.output.SetNamespace(key.Namespace)

	return c.output, c.shouldDeleteF != nil && c.shouldDeleteF(), nil
}

func (c *ChildResource[T]) ID() string {
	if c.userIdentifier != "" {
		return c.userIdentifier
	}

	key := c.keyF()

	return fmt.Sprintf("%v,%v", c.Kind(), key)
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

func (c *ChildResource[T]) IsReady(obj client.Object) bool {
	if c.isReadyF != nil {
		if typedObj, ok := obj.(T); ok {
			return c.isReadyF(typedObj)
		}
	}
	return false
}

func (c *ChildResource[T]) RequiresManualDeletion(obj client.Object) bool {
	if c.requiresDeletionF != nil {
		if typedObj, ok := obj.(T); ok {
			return c.requiresDeletionF(typedObj)
		}
	}
	return false
}

func (c *ChildResource[T]) ShouldDeleteNow() bool {
	if c.shouldDeleteF != nil {
		return c.shouldDeleteF()
	}
	return false
}

func (c *ChildResource[T]) OnReconcile(ctx context.Context) error {
	if c.onReconcileF != nil {
		return c.onReconcileF(ctx)
	}
	return nil
}

func (c *ChildResource[T]) OnCreate(ctx context.Context, child client.Object) error {
	if c.onCreateF != nil {
		if typedObj, ok := child.(T); ok {
			return c.onCreateF(ctx, typedObj)
		}
	}
	return nil
}

func (c *ChildResource[T]) OnUpdate(ctx context.Context, child client.Object) error {
	if c.onUpdateF != nil {
		if typedObj, ok := child.(T); ok {
			return c.onUpdateF(ctx, typedObj)
		}
	}
	return nil
}

func (c *ChildResource[T]) OnDelete(ctx context.Context, child client.Object) error {
	if c.onDeleteF != nil {
		if typedObj, ok := child.(T); ok {
			return c.onDeleteF(ctx, typedObj)
		}
	}
	return nil
}

func (c *ChildResource[T]) OnFinalize(ctx context.Context, child client.Object) error {
	if c.onFinalizeF != nil {
		if typedObj, ok := child.(T); ok {
			return c.onFinalizeF(ctx, typedObj)
		}
	}
	return nil
}

func (c *ChildResource[T]) GetMutator(obj client.Object) func() error {
	return func() error {
		if c.mutateF != nil {
			if typedObj, ok := obj.(T); ok {
				return c.mutateF(typedObj)
			}
		}
		return nil
	}
}

type UntypedChildResource struct {
	*DependencyResource[*unstructured.Unstructured]
	gvk schema.GroupVersionKind
}

func NewUntypedChildResource(gvk schema.GroupVersionKind, opts ...DependencyResourceOption[*unstructured.Unstructured]) *UntypedChildResource {
	c := &UntypedChildResource{
		DependencyResource: NewDependencyResource(&unstructured.Unstructured{}, opts...),
		gvk:                gvk,
	}

	return c
}

func (c *UntypedChildResource) New() client.Object {
	obj := NewInstanceOf(c.output)
	obj.SetAPIVersion(c.gvk.GroupVersion().String())
	obj.SetKind(c.gvk.Kind)
	return obj
}

func (c *UntypedChildResource) Kind() string {
	return c.gvk.Kind
}
