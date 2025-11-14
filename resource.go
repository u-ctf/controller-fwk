package ctrlfwk

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mutator[ResourceType client.Object] func(resource ResourceType) error

type GenericResource[CustomResource client.Object, ContextType Context[CustomResource]] interface {
	ID() string
	ObjectMetaGenerator() (obj client.Object, delete bool, err error)
	ShouldDeleteNow() bool
	GetMutator(obj client.Object) func() error
	Set(obj client.Object)
	Get() client.Object
	Kind() string
	IsReady(obj client.Object) bool
	RequiresManualDeletion(obj client.Object) bool
	CanBePaused() bool

	// Hooks
	BeforeReconcile(ctx ContextType) error
	AfterReconcile(ctx ContextType, resource client.Object) error
	OnCreate(ctx ContextType, resource client.Object) error
	OnUpdate(ctx ContextType, resource client.Object) error
	OnDelete(ctx ContextType, resource client.Object) error
	OnFinalize(ctx ContextType, resource client.Object) error
}

var _ GenericResource[client.Object, Context[client.Object]] = &Resource[client.Object, Context[client.Object], client.Object]{}

type Resource[CustomResource client.Object, ContextType Context[CustomResource], ResourceType client.Object] struct {
	userIdentifier string
	keyF           func() types.NamespacedName
	mutateF        Mutator[ResourceType]

	isReadyF          func(obj ResourceType) bool
	shouldDeleteF     func() bool
	requiresDeletionF func(obj ResourceType) bool
	output            ResourceType
	canBePausedF      func() bool

	// Hooks
	beforeReconcileF func(ctx ContextType) error
	afterReconcileF  func(ctx ContextType, resource ResourceType) error
	onCreateF        func(ctx ContextType, resource ResourceType) error
	onUpdateF        func(ctx ContextType, resource ResourceType) error
	onDeleteF        func(ctx ContextType, resource ResourceType) error
	onFinalizeF      func(ctx ContextType, resource ResourceType) error
}

func (c *Resource[CustomResource, ContextType, ResourceType]) Kind() string {
	return reflect.TypeOf(c.output).Elem().Name()
}

func (c *Resource[CustomResource, ContextType, ResourceType]) ObjectMetaGenerator() (obj client.Object, skip bool, err error) {
	if reflect.ValueOf(c.output).IsNil() {
		c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(ResourceType)
	}

	key := c.keyF()

	c.output.SetName(key.Name)
	c.output.SetNamespace(key.Namespace)

	return c.output, c.shouldDeleteF != nil && c.shouldDeleteF(), nil
}

func (c *Resource[CustomResource, ContextType, ResourceType]) ID() string {
	if c.userIdentifier != "" {
		return c.userIdentifier
	}

	key := c.keyF()

	return fmt.Sprintf("%v,%v", c.Kind(), key)
}

func (c *Resource[CustomResource, ContextType, ResourceType]) Set(obj client.Object) {
	if reflect.TypeOf(c.output) == reflect.TypeOf(obj) {
		if reflect.ValueOf(c.output).IsNil() {
			c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(ResourceType)
		}

		reflect.ValueOf(c.output).Elem().Set(reflect.ValueOf(obj).Elem())
	}
}

func (c *Resource[CustomResource, ContextType, ResourceType]) Get() client.Object {
	return c.output
}

func (c *Resource[CustomResource, ContextType, ResourceType]) IsReady(obj client.Object) bool {
	if c.isReadyF != nil {
		if typedObj, ok := obj.(ResourceType); ok {
			return c.isReadyF(typedObj)
		}
		if obj == nil {
			var zero ResourceType
			return c.isReadyF(zero)
		}
	}
	return false
}

func (c *Resource[CustomResource, ContextType, ResourceType]) RequiresManualDeletion(obj client.Object) bool {
	if c.requiresDeletionF != nil {
		if typedObj, ok := obj.(ResourceType); ok {
			return c.requiresDeletionF(typedObj)
		}
		if obj == nil {
			var zero ResourceType
			return c.requiresDeletionF(zero)
		}
	}
	return false
}

func (c *Resource[CustomResource, ContextType, ResourceType]) ShouldDeleteNow() bool {
	if c.shouldDeleteF != nil {
		return c.shouldDeleteF()
	}
	return false
}

func (c *Resource[CustomResource, ContextType, ResourceType]) BeforeReconcile(ctx ContextType) error {
	if c.beforeReconcileF != nil {
		return c.beforeReconcileF(ctx)
	}
	return nil
}

func (c *Resource[CustomResource, ContextType, ResourceType]) AfterReconcile(ctx ContextType, resource client.Object) error {
	if c.afterReconcileF != nil {
		if typedObj, ok := resource.(ResourceType); ok {
			return c.afterReconcileF(ctx, typedObj)
		}
		if resource == nil {
			var zero ResourceType
			return c.afterReconcileF(ctx, zero)
		}
	}
	return nil
}

func (c *Resource[CustomResource, ContextType, ResourceType]) OnCreate(ctx ContextType, resource client.Object) error {
	if c.onCreateF != nil {
		if typedObj, ok := resource.(ResourceType); ok {
			return c.onCreateF(ctx, typedObj)
		}
		if resource == nil {
			var zero ResourceType
			return c.onCreateF(ctx, zero)
		}
	}
	return nil
}

func (c *Resource[CustomResource, ContextType, ResourceType]) OnUpdate(ctx ContextType, resource client.Object) error {
	if c.onUpdateF != nil {
		if typedObj, ok := resource.(ResourceType); ok {
			return c.onUpdateF(ctx, typedObj)
		}
		if resource == nil {
			var zero ResourceType
			return c.onUpdateF(ctx, zero)
		}
	}
	return nil
}

func (c *Resource[CustomResource, ContextType, ResourceType]) OnDelete(ctx ContextType, resource client.Object) error {
	if c.onDeleteF != nil {
		if typedObj, ok := resource.(ResourceType); ok {
			return c.onDeleteF(ctx, typedObj)
		}
		if resource == nil {
			var zero ResourceType
			return c.onDeleteF(ctx, zero)
		}
	}
	return nil
}

func (c *Resource[CustomResource, ContextType, ResourceType]) OnFinalize(ctx ContextType, resource client.Object) error {
	if c.onFinalizeF != nil {
		if typedObj, ok := resource.(ResourceType); ok {
			return c.onFinalizeF(ctx, typedObj)
		}
		if resource == nil {
			var zero ResourceType
			return c.onFinalizeF(ctx, zero)
		}
	}
	return nil
}

func (c *Resource[CustomResource, ContextType, ResourceType]) GetMutator(obj client.Object) func() error {
	return func() error {
		if c.mutateF != nil {
			if typedObj, ok := obj.(ResourceType); ok {
				return c.mutateF(typedObj)
			}
			if obj == nil {
				var zero ResourceType
				return c.mutateF(zero)
			}
		}
		return nil
	}
}

func (c *Resource[CustomResource, ContextType, ResourceType]) CanBePaused() bool {
	if c.canBePausedF != nil {
		return c.canBePausedF()
	}
	return false
}
