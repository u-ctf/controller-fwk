package ctrlfwk

import (
	"context"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ControllerCustomResource interface {
	client.Object
}

type Reconciler[ControllerResourceType ControllerCustomResource] interface {
	client.Client

	SetCustomResource(key ControllerResourceType)
	GetCustomResource() ControllerResourceType
	GetCleanCustomResource() ControllerResourceType
}

type ReconcilerWithWatcher[ControllerResourceType ControllerCustomResource] interface {
	Reconciler[ControllerResourceType]

	Watcher
}

type ReconcilerWithDependencies[ControllerResourceType ControllerCustomResource] interface {
	Reconciler[ControllerResourceType]

	GetDependencies(ctx context.Context, req ctrl.Request) ([]GenericDependency, error)
}

type ReconcilerWithResources[ControllerResourceType ControllerCustomResource] interface {
	Reconciler[ControllerResourceType]

	GetResources(ctx context.Context, req ctrl.Request) ([]GenericResource, error)
}

type ReconcilerWithEventRecorder[ControllerResourceType ControllerCustomResource] interface {
	Reconciler[ControllerResourceType]

	record.EventRecorder
}
