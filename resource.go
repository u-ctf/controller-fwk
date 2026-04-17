package ctrlfwk

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mutator[ResourceType client.Object] func(resource ResourceType) error

type GenericResource[CustomResource client.Object, ContextType Context[CustomResource]] interface {
	// ID returns a stable identifier for this resource instance.
	// It is primarily used for structured logging in NewReconcileResourcesStep.
	ID() string
	// ObjectMetaGenerator returns the desired object metadata and whether this resource should be deleted now.
	// NewReconcileResourceStep uses it to build the desired object before create-or-patch.
	ObjectMetaGenerator() (obj client.Object, delete bool, err error)
	// ShouldDeleteNow indicates whether getDesiredObject should delete this resource and early-return.
	ShouldDeleteNow() bool
	// GetMutator returns the mutation function passed to controllerutil.CreateOrPatch.
	GetMutator(obj client.Object) func() error
	// Set stores the reconciled resource object after create-or-patch.
	Set(obj client.Object)
	// Get returns the currently stored resource object.
	// NewReconcileResourceStep uses it during finalization checks.
	Get() client.Object
	// Kind returns the Kubernetes kind name for this resource type.
	// It is used in step naming and as part of the default ID implementation.
	Kind() string
	// IsReady reports whether the reconciled resource is ready.
	// NewReconcileResourceStep early-returns when this is false.
	IsReady(obj client.Object) bool
	// RequiresManualDeletion indicates whether finalization should explicitly delete the resource.
	// If false, finalization skips manual deletion and relies on garbage collection.
	RequiresManualDeletion(obj client.Object) bool
	// CanBePaused enables pause-label handling around mutation.
	// When true, NewReconcileResourceStep skips mutation if the pause label is present.
	CanBePaused() bool

	// Hooks
	// BeforeReconcile runs before desired state generation and mutation.
	BeforeReconcile(ctx ContextType) error
	// AfterReconcile always runs at the end of NewReconcileResourceStep.
	AfterReconcile(ctx ContextType, resource client.Object) error
	// OnCreate runs when CreateOrPatch reports a newly created resource.
	OnCreate(ctx ContextType, resource client.Object) error
	// OnUpdate runs when CreateOrPatch reports an updated resource.
	OnUpdate(ctx ContextType, resource client.Object) error
	// OnDelete runs when ObjectMetaGenerator requests immediate deletion and delete succeeds.
	OnDelete(ctx ContextType, resource client.Object) error
	// OnFinalize runs during CR finalization, after or instead of explicit deletion.
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

// ObjectMetaGenerator initializes the desired object identity and reports delete-now intent.
// getDesiredObject uses this during each reconcile iteration.
func (c *Resource[CustomResource, ContextType, ResourceType]) ObjectMetaGenerator() (obj client.Object, skip bool, err error) {
	if reflect.ValueOf(c.output).IsNil() {
		c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(ResourceType)
	}

	key := c.keyF()

	c.output.SetName(key.Name)
	c.output.SetNamespace(key.Namespace)

	return c.output, c.shouldDeleteF != nil && c.shouldDeleteF(), nil
}

// ID returns the user-provided identifier, or a generated one based on kind and key.
// The identifier is used for resource-scoped log fields during reconciliation.
func (c *Resource[CustomResource, ContextType, ResourceType]) ID() string {
	if c.userIdentifier != "" {
		return c.userIdentifier
	}

	key := c.keyF()

	return fmt.Sprintf("%v,%v", c.Kind(), key)
}

// Set copies obj into the stored output when the types match.
// NewReconcileResourceStep calls this after CreateOrPatch.
func (c *Resource[CustomResource, ContextType, ResourceType]) Set(obj client.Object) {
	if reflect.TypeOf(c.output) == reflect.TypeOf(obj) {
		if reflect.ValueOf(c.output).IsNil() {
			c.output = reflect.New(reflect.TypeOf(c.output).Elem()).Interface().(ResourceType)
		}

		reflect.ValueOf(c.output).Elem().Set(reflect.ValueOf(obj).Elem())
	}
}

// Get returns the current resource output object.
func (c *Resource[CustomResource, ContextType, ResourceType]) Get() client.Object {
	return c.output
}

// IsReady evaluates readiness using the configured readiness function.
// NewReconcileResourceStep uses this to decide whether to early-return.
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

// RequiresManualDeletion reports whether finalization must explicitly delete this resource.
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

// ShouldDeleteNow reports whether this resource should be deleted in getDesiredObject.
func (c *Resource[CustomResource, ContextType, ResourceType]) ShouldDeleteNow() bool {
	if c.shouldDeleteF != nil {
		return c.shouldDeleteF()
	}
	return false
}

// BeforeReconcile runs the configured pre-reconcile hook if present.
func (c *Resource[CustomResource, ContextType, ResourceType]) BeforeReconcile(ctx ContextType) error {
	if c.beforeReconcileF != nil {
		return c.beforeReconcileF(ctx)
	}
	return nil
}

// AfterReconcile runs the configured post-reconcile hook if present.
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

// OnCreate runs the configured creation hook when a resource is created.
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

// OnUpdate runs the configured update hook when a resource is updated.
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

// OnDelete runs the configured delete hook for delete-now flows.
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

// OnFinalize runs the configured finalize hook during CR finalization.
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

// GetMutator wraps the configured mutate function for CreateOrPatch.
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

// CanBePaused reports whether pause-label checks should skip mutation for this resource.
func (c *Resource[CustomResource, ContextType, ResourceType]) CanBePaused() bool {
	if c.canBePausedF != nil {
		return c.canBePausedF()
	}
	return false
}
