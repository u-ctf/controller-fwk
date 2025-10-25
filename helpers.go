package ctrlfwk

import (
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewInstanceOf[ObjectType client.Object](object ObjectType) ObjectType {
	var newChild ObjectType
	// Use reflection to create a new instance of the child type
	childType := reflect.TypeOf(object)
	if childType == nil {
		return newChild
	}

	if childType.Kind() == reflect.Ptr {
		newChild = reflect.New(childType.Elem()).Interface().(ObjectType)
	} else {
		newChild = reflect.New(childType).Interface().(ObjectType)
	}
	return newChild
}

func isFinalizing[
	ControllerResourceType ControllerCustomResource,
](
	reconciler Reconciler[ControllerResourceType],
) bool {
	return reconciler.GetCustomResource().GetDeletionTimestamp() != nil
}

func SetAnnotation(obj client.Object, key, value string) {
	if obj.GetAnnotations() == nil {
		obj.SetAnnotations(make(map[string]string))
	}
	obj.GetAnnotations()[key] = value
}

func GetAnnotation(obj client.Object, key string) string {
	if obj.GetAnnotations() == nil {
		return ""
	}
	return obj.GetAnnotations()[key]
}
