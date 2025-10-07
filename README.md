# Library

The Library is a collection of helpers to help with the development of the operators. It is used by all the operators to simplify the code and avoid duplication.

## Stepper

The stepper is a simple helper to help with the reconciliation of the resources. It is used to create a stepper that can be used to execute functions in order and simplify logging. A step looks like this:

```go
type Step struct {
	// Name is the name of the step
	Name string

	// Step is the function to execute
	Step func(ctx context.Context, req ctrl.Request) StepResult
}
```

A sample step might look like this:

```go
func (reconciler *AppReconciler) NewAlterStatusStep() library.Step {
	return library.Step{
		Name: "Alter Status For No Reason",
		Step: func(ctx context.Context, req ctrl.Request) library.StepResult {
			reconciler.app.Status.Field = 3

			if err := reconciler.Status().Update(ctx, &reconciler.app); err != nil {
				return library.ResultInError(err)
			}

			return library.ResultSuccess()
		},
	}
}
```

## Reconciler

In order to be used with the library, the reconciler must implement the `library.Reconciler` interface. This interface is used to create a reconciler that can be used with the library.

```go
type ControllerResource interface {
	GetStatus() *Status

	client.Object
}

type Reconciler[ControllerResourceType ControllerResource] interface {
	GetController() controller.TypedController[reconcile.Request]
	GetFinalizer() string
	GetCustomResource() ControllerResourceType
	SetCustomResource(ControllerResourceType)

	client.Client
	ctrl.Manager
	Watcher
}
```

These are meant to be simple and not add a lot of boilerplate to the operator.

## Children

In order to reconcile children, an operator must implement the `ReconcilerWithDynamicChildren` interface:

```go
type ReconcilerWithDynamicChildren[ControllerResourceType ControllerResource] interface {
	Reconciler[ControllerResourceType]

	GetChildren(ctx context.Context, req ctrl.Request) ([]GenericChildResource, error)
}
```

For example, an Operator that creates a ConfigMap as a child resource might look like this:

```go
func (reconciler *Reconciler) GetChildren(ctx context.Context, req ctrl.Request) ([]library.GenericChildResource, error) {
	return []library.GenericChildResource{
		library.NewChildResource(
			&corev1.ConfigMap{},
			library.WithChildOutput(&reconciler.configMap),
			library.WithChildGenerator(reconciler.configMapGenerator),
		),
    }
}
```

The generator function is used by the library to create OR update the child resource depending on the state of the child resource. The generator function is a function that takes a context and a request and returns a child resource.

In order to simplify the status its also set in the `childResources` field in the `status` of the CR :
```yaml
status:
    childResources:
      - apiVersion: v1
        group: ""
        kind: ConfigMap
        name: app-sample
        namespace: default
        observedGeneration: 15
        status: "True"
        transitionTime: "2025-04-26T07:57:00Z"
        uid: 16d49a29-eb8f-462e-8460-2895f6c14bd4
```

This status also shows you if any error occurred during the reconciliation of the child resource. The status is set to `True` if the child resource is in a good state and `False` if there was an error or if the child resource is not in a good state.

## Dependencies

In order to reconcile dependencies, an operator must implement the `ReconcilerWithDynamicDependencies` interface:

```go
type ReconcilerWithDynamicDependencies[ControllerResourceType ControllerResource] interface {
	Reconciler[ControllerResourceType]

	GetDependencies(ctx context.Context, req ctrl.Request) ([]GenericDependencyResource, error)
}
```

For example, an Operator that requires a ConfigMap as a dependency might look like this:

```go
func (reconciler *Reconciler) GetDependencies(ctx context.Context, req ctrl.Request) ([]library.GenericDependencyResource, error) {
	return []library.GenericDependencyResource{
		library.NewDependencyResource(
			&corev1.ConfigMap{},
			library.WithName[*corev1.ConfigMap]("test"),
			library.WithNamespace[*corev1.ConfigMap](req.Namespace),
		),
	}, nil
}
```

It is can support cross-namespace dependencies as well.

As per the children, the status of the dependency is set in the `status` of the CR. The status is set to `True` if the dependency is in a good state and `False` if there was an error or if the dependency is not in a good state.
```yaml
status:
    dependencies:
      - apiVersion: v1
        group: ""
        kind: ConfigMap
        name: app-sample
        namespace: default
        observedGeneration: 15
        status: "True"
        transitionTime: "2025-04-26T07:57:00Z"
```

## Contracts

Contracts are meant to get a struct from an unstructured object. This is useful when you want to get a struct from a CRD that is not known at compile time. For example, the Route operator needs to get the `routeContract` from the target. The contract looks like this:

```yaml
status:
  routeContract:
    serviceRef:
      name: app-sample
      port: 80
```

The code to get the contract looks like this:

```go 
routeContract, err := library.GetContract[routev1.RouteContract](target, "routeContract")
```

## Watch Cache

Reconciler implement by default a watch cache. This is to simplify the watching logic. The "reconcile child" and "get dependency" steps use this watch cache to register new resources to watch, this means that the operator must have the RBAC to do so.
