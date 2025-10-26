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

type Mutator[T client.Object] func(resource T) error

type GenericResource interface {
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
	BeforeReconcile(ctx context.Context) error
	AfterReconcile(ctx context.Context, resource client.Object) error
	OnCreate(ctx context.Context, resource client.Object) error
	OnUpdate(ctx context.Context, resource client.Object) error
	OnDelete(ctx context.Context, resource client.Object) error
	OnFinalize(ctx context.Context, resource client.Object) error
}

var _ GenericResource = &Resource[client.Object]{}

type Resource[T client.Object] struct {
	userIdentifier string
	keyF           func() types.NamespacedName
	mutateF        Mutator[T]

	isReadyF          func(obj T) bool
	shouldDeleteF     func() bool
	requiresDeletionF func(obj T) bool
	output            T

	// Hooks
	beforeReconcileF func(ctx context.Context) error
	afterReconcileF  func(ctx context.Context, resource T) error
	onCreateF        func(ctx context.Context, resource T) error
	onUpdateF        func(ctx context.Context, resource T) error
	onDeleteF        func(ctx context.Context, resource T) error
	onFinalizeF      func(ctx context.Context, resource T) error
}

type ResourceOption[T client.Object] func(*Resource[T])

func ResourceWithKey[T client.Object](_ T, name types.NamespacedName) ResourceOption[T] {
	return func(c *Resource[T]) {
		c.keyF = func() types.NamespacedName {
			return name
		}
	}
}

func ResourceWithKeyFunc[T client.Object](_ T, f func() types.NamespacedName) ResourceOption[T] {
	return func(c *Resource[T]) {
		c.keyF = f
	}
}

func ResourceWithMutator[T client.Object](f Mutator[T]) ResourceOption[T] {
	return func(c *Resource[T]) {
		c.mutateF = f
	}
}

func ResourceWithOutput[T client.Object](obj T) ResourceOption[T] {
	return func(c *Resource[T]) {
		c.output = obj
	}
}

func ResourceWithReadinessCondition[T client.Object](f func(obj T) bool) ResourceOption[T] {
	return func(c *Resource[T]) {
		c.isReadyF = f
	}
}

func ResourceSkipAndDeleteOnCondition[T client.Object](_ T, f func() bool) ResourceOption[T] {
	return func(c *Resource[T]) {
		c.shouldDeleteF = f
	}
}

func ResourceRequireManualDeletionForFinalize[T client.Object](f func(obj T) bool) ResourceOption[T] {
	return func(c *Resource[T]) {
		c.requiresDeletionF = f
	}
}

func ResourceBeforeReconcile[T client.Object](_ T, f func(ctx context.Context) error) ResourceOption[T] {
	return func(c *Resource[T]) {
		c.beforeReconcileF = f
	}
}

func ResourceAfterReconcile[T client.Object](_ T, f func(ctx context.Context, resource T) error) ResourceOption[T] {
	return func(c *Resource[T]) {
		c.afterReconcileF = f
	}
}

func ResourceAfterCreate[T client.Object](f func(ctx context.Context, resource T) error) ResourceOption[T] {
	return func(c *Resource[T]) {
		c.onCreateF = f
	}
}

func ResourceAfterUpdate[T client.Object](f func(ctx context.Context, resource T) error) ResourceOption[T] {
	return func(c *Resource[T]) {
		c.onUpdateF = f
	}
}

func ResourceAfterDelete[T client.Object](f func(ctx context.Context, resource T) error) ResourceOption[T] {
	return func(c *Resource[T]) {
		c.onDeleteF = f
	}
}

func ResourceAfterFinalize[T client.Object](f func(ctx context.Context, resource T) error) ResourceOption[T] {
	return func(c *Resource[T]) {
		c.onFinalizeF = f
	}
}

func NewResource[T client.Object](_ T, opts ...ResourceOption[T]) *Resource[T] {
	c := &Resource[T]{}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Resource[T]) Kind() string {
	return reflect.TypeOf(c.output).Elem().Name()
}

func (c *Resource[T]) ObjectMetaGenerator() (obj client.Object, skip bool, err error) {
	if reflect.ValueOf(c.output).IsNil() {
		c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(T)
	}

	key := c.keyF()

	c.output.SetName(key.Name)
	c.output.SetNamespace(key.Namespace)

	return c.output, c.shouldDeleteF != nil && c.shouldDeleteF(), nil
}

func (c *Resource[T]) ID() string {
	if c.userIdentifier != "" {
		return c.userIdentifier
	}

	key := c.keyF()

	return fmt.Sprintf("%v,%v", c.Kind(), key)
}

func (c *Resource[T]) Set(obj client.Object) {
	if reflect.TypeOf(c.output) == reflect.TypeOf(obj) {
		if reflect.ValueOf(c.output).IsNil() {
			c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(T)
		}

		reflect.ValueOf(c.output).Elem().Set(reflect.ValueOf(obj).Elem())
	}
}

func (c *Resource[T]) Get() client.Object {
	return c.output
}

func (c *Resource[T]) IsReady(obj client.Object) bool {
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

func (c *Resource[T]) RequiresManualDeletion(obj client.Object) bool {
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

func (c *Resource[T]) ShouldDeleteNow() bool {
	if c.shouldDeleteF != nil {
		return c.shouldDeleteF()
	}
	return false
}

func (c *Resource[T]) BeforeReconcile(ctx context.Context) error {
	if c.beforeReconcileF != nil {
		return c.beforeReconcileF(ctx)
	}
	return nil
}

func (c *Resource[T]) AfterReconcile(ctx context.Context, resource client.Object) error {
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

func (c *Resource[T]) OnCreate(ctx context.Context, resource client.Object) error {
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

func (c *Resource[T]) OnUpdate(ctx context.Context, resource client.Object) error {
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

func (c *Resource[T]) OnDelete(ctx context.Context, resource client.Object) error {
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

func (c *Resource[T]) OnFinalize(ctx context.Context, resource client.Object) error {
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

func (c *Resource[T]) GetMutator(obj client.Object) func() error {
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

type UntypedResource struct {
	*Resource[*unstructured.Unstructured]
	gvk schema.GroupVersionKind
}

func NewUntypedResource(gvk schema.GroupVersionKind, opts ...ResourceOption[*unstructured.Unstructured]) *UntypedResource {
	c := &UntypedResource{
		Resource: NewResource(&unstructured.Unstructured{}, opts...),
		gvk:      gvk,
	}

	return c
}

func (c *UntypedResource) New() client.Object {
	obj := NewInstanceOf(c.output)
	obj.SetAPIVersion(c.gvk.GroupVersion().String())
	obj.SetKind(c.gvk.Kind)
	return obj
}

func (c *UntypedResource) Kind() string {
	return c.gvk.Kind
}
