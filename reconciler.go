package ctrlfwk

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ControllerCustomResource interface {
	client.Object
}

type Reconciler[ControllerResourceType ControllerCustomResource] interface {
	client.Client
	ctrl.Manager

	SetCustomResource(key ControllerResourceType)
	GetCustomResource() ControllerResourceType
	GetCleanCustomResource() ControllerResourceType
}

type ReconcilerWithWatcher[ControllerResourceType ControllerCustomResource] interface {
	Reconciler[ControllerResourceType]

	Watcher
}

type ReconcilerWithDynamicDependencies[ControllerResourceType ControllerCustomResource] interface {
	Reconciler[ControllerResourceType]

	GetDependencies(ctx context.Context, req ctrl.Request) ([]GenericDependencyResource, error)
}

type ReconcilerWithDynamicChildren[ControllerResourceType ControllerCustomResource] interface {
	Reconciler[ControllerResourceType]

	GetChildren(ctx context.Context, req ctrl.Request) ([]GenericChildResource, error)
}
