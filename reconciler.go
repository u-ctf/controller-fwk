package library

import (
	"context"
	"library/tracing"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

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

type TracingReconciler[ControllerResourceType ControllerResource] interface {
	Reconciler[ControllerResourceType]
	tracing.Tracer
}

type ReconcilerWithDynamicDependencies[ControllerResourceType ControllerResource] interface {
	Reconciler[ControllerResourceType]

	GetDependencies(ctx context.Context, req ctrl.Request) ([]GenericDependencyResource, error)
}

type ReconcilerWithDynamicChildren[ControllerResourceType ControllerResource] interface {
	Reconciler[ControllerResourceType]

	GetChildren(ctx context.Context, req ctrl.Request) ([]GenericChildResource, error)
}
