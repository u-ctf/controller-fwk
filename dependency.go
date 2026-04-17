package ctrlfwk

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GenericDependency[CustomResourceType client.Object, ContextType Context[CustomResourceType]] interface {
	// ID returns a stable identifier for this dependency instance.
	// It is primarily used for structured logging in NewResolveDynamicDependenciesStep.
	ID() string
	// New creates a new empty object instance for this dependency type.
	// NewResolveDependencyStep uses this object as the target for reconciler.Get.
	New() client.Object
	// Key returns the namespaced name used to fetch this dependency.
	// NewResolveDependencyStep passes this key to reconciler.Get.
	Key() types.NamespacedName
	// Set stores the resolved dependency object on the dependency.
	// NewResolveDependencyStep calls it after a successful Get so readiness and hooks can use the value.
	Set(obj client.Object)
	// Get returns the currently stored dependency object.
	// Consumers can use it to read the resolved dependency after the step has executed.
	Get() client.Object
	// ShouldWaitForReady indicates whether reconciliation should wait until IsReady is true.
	// In NewResolveDependencyStep, true causes a requeue until IsReady returns true.
	ShouldWaitForReady() bool
	// ShouldAddManagedByAnnotation indicates whether the managed-by annotation should be added.
	// In NewResolveDependencyStep, this controls AddManagedBy/RemoveManagedBy behavior and watch setup.
	ShouldAddManagedByAnnotation() bool
	// IsReady reports whether the stored dependency is in a ready state.
	// Its result is checked by NewResolveDependencyStep when ShouldWaitForReady is true.
	IsReady() bool
	// IsOptional indicates whether this dependency can be absent without failing reconciliation.
	// If true, a not-found dependency is treated as success in NewResolveDependencyStep.
	IsOptional() bool
	// Kind returns the Kubernetes kind name for this dependency object type.
	// It is used in step naming and as part of the default ID implementation.
	Kind() string

	// Hooks
	// BeforeReconcile is called before the dependency is fetched or mutated.
	// Use it to prepare context or perform validations.
	BeforeReconcile(ctx ContextType) error
	// AfterReconcile is always called at the end of NewResolveDependencyStep.
	// It receives the fetched resource when available, or nil/zero on early failures.
	AfterReconcile(ctx ContextType, resource client.Object) error
}

var _ GenericDependency[client.Object, Context[client.Object]] = &Dependency[client.Object, Context[client.Object], client.Object]{}

type Dependency[CustomResourceType client.Object, ContextType Context[CustomResourceType], DependencyType client.Object] struct {
	userIdentifier string
	keyF           func() types.NamespacedName
	isReadyF       func(obj DependencyType) bool
	output         DependencyType
	isOptional     bool
	waitForReady   bool
	addManagedBy   bool
	name           string
	namespace      string

	// Hooks
	beforeReconcileF func(ctx ContextType) error
	afterReconcileF  func(ctx ContextType, resource DependencyType) error
}

// New creates a new empty instance of the configured dependency object type.
func (c *Dependency[CustomResourceType, ContextType, DependencyType]) New() client.Object {
	return NewInstanceOf(c.output)
}

// Kind returns the underlying Kubernetes kind name for the dependency type.
func (c *Dependency[CustomResourceType, ContextType, DependencyType]) Kind() string {
	return reflect.TypeOf(c.output).Elem().Name()
}

// Set copies obj into the stored dependency output when the types match.
func (c *Dependency[CustomResourceType, ContextType, DependencyType]) Set(obj client.Object) {
	if reflect.TypeOf(c.output) == reflect.TypeOf(obj) {
		if reflect.ValueOf(c.output).IsNil() {
			c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(DependencyType)
		}

		reflect.ValueOf(c.output).Elem().Set(reflect.ValueOf(obj).Elem())
	}
}

// Get returns the current dependency output object.
func (c *Dependency[CustomResourceType, ContextType, DependencyType]) Get() client.Object {
	return c.output
}

// IsOptional reports whether this dependency is optional.
func (c *Dependency[CustomResourceType, ContextType, DependencyType]) IsOptional() bool {
	return c.isOptional
}

// Key returns the namespaced name used to locate this dependency.
// NewResolveDependencyStep passes this key to reconciler.Get.
func (c *Dependency[CustomResourceType, ContextType, DependencyType]) Key() types.NamespacedName {
	if c.keyF != nil {
		return c.keyF()
	}

	return types.NamespacedName{
		Name:      c.name,
		Namespace: c.namespace,
	}
}

// ID returns the user-provided identifier, or a generated one based on kind and key.
// The identifier is used for dependency-scoped log fields during dependency resolution.
func (c *Dependency[CustomResourceType, ContextType, DependencyType]) ID() string {
	if c.userIdentifier != "" {
		return c.userIdentifier
	}
	return fmt.Sprintf("%v,%v", c.Kind(), c.Key())
}

// ShouldWaitForReady indicates whether reconciliation should block until ready.
func (c *Dependency[CustomResourceType, ContextType, DependencyType]) ShouldWaitForReady() bool {
	return c.waitForReady
}

// IsReady evaluates readiness using the configured readiness function.
func (c *Dependency[CustomResourceType, ContextType, DependencyType]) IsReady() bool {
	if c.isReadyF != nil {
		return c.isReadyF(c.output)
	}
	return false
}

// BeforeReconcile runs the configured pre-reconcile hook if present.
func (c *Dependency[CustomResourceType, ContextType, DependencyType]) BeforeReconcile(ctx ContextType) error {
	if c.beforeReconcileF != nil {
		return c.beforeReconcileF(ctx)
	}
	return nil
}

// AfterReconcile runs the configured post-reconcile hook if present.
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

// ShouldAddManagedByAnnotation indicates whether managed-by should be set on the dependency.
func (c *Dependency[CustomResourceType, ContextType, DependencyType]) ShouldAddManagedByAnnotation() bool {
	return c.addManagedBy
}
