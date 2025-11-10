package ctrlfwk

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GenericDependency[CustomResourceType client.Object, ContextType Context[CustomResourceType]] interface {
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
	BeforeReconcile(ctx ContextType) error
	AfterReconcile(ctx ContextType, resource client.Object) error
}

var _ GenericDependency[client.Object, Context[client.Object]] = &Dependency[client.Object, Context[client.Object], client.Object]{}

type Dependency[CustomResourceType client.Object, ContextType Context[CustomResourceType], DependencyType client.Object] struct {
	userIdentifier string
	isReadyF       func(obj DependencyType) bool
	output         DependencyType
	isOptional     bool
	waitForReady   bool
	name           string
	namespace      string

	// Hooks
	beforeReconcileF func(ctx ContextType) error
	afterReconcileF  func(ctx ContextType, resource DependencyType) error
}

func (c *Dependency[CustomResourceType, ContextType, DependencyType]) New() client.Object {
	return NewInstanceOf(c.output)
}

func (c *Dependency[CustomResourceType, ContextType, DependencyType]) Kind() string {
	return reflect.TypeOf(c.output).Elem().Name()
}

func (c *Dependency[CustomResourceType, ContextType, DependencyType]) Set(obj client.Object) {
	if reflect.TypeOf(c.output) == reflect.TypeOf(obj) {
		if reflect.ValueOf(c.output).IsNil() {
			c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(DependencyType)
		}

		reflect.ValueOf(c.output).Elem().Set(reflect.ValueOf(obj).Elem())
	}
}

func (c *Dependency[CustomResourceType, ContextType, DependencyType]) Get() client.Object {
	return c.output
}

func (c *Dependency[CustomResourceType, ContextType, DependencyType]) IsOptional() bool {
	return c.isOptional
}

func (c *Dependency[CustomResourceType, ContextType, DependencyType]) Key() types.NamespacedName {
	return types.NamespacedName{
		Name:      c.name,
		Namespace: c.namespace,
	}
}

func (c *Dependency[CustomResourceType, ContextType, DependencyType]) ID() string {
	if c.userIdentifier != "" {
		return c.userIdentifier
	}
	return fmt.Sprintf("%v,%v", c.Kind(), c.Key())
}

func (c *Dependency[CustomResourceType, ContextType, DependencyType]) ShouldWaitForReady() bool {
	return c.waitForReady
}

func (c *Dependency[CustomResourceType, ContextType, DependencyType]) IsReady() bool {
	if c.isReadyF != nil {
		return c.isReadyF(c.output)
	}
	return false
}

func (c *Dependency[CustomResourceType, ContextType, DependencyType]) BeforeReconcile(ctx ContextType) error {
	if c.beforeReconcileF != nil {
		return c.beforeReconcileF(ctx)
	}
	return nil
}

func (c *Dependency[CustomResourceType, ContextType, DependencyType]) AfterReconcile(ctx ContextType, resource client.Object) error {
	if c.afterReconcileF != nil {
		switch typedObj := resource.(type) {
		case DependencyType:
			return c.afterReconcileF(ctx, typedObj)
		default:
			var zero DependencyType
			return c.afterReconcileF(ctx, zero)
		}
	}
	return nil
}
