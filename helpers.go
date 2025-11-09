package ctrlfwk

import (
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewInstanceOf[T client.Object](object T) T {
	var newObject T
	// Use reflection to create a new instance of the object type
	objectType := reflect.TypeOf(object)
	if objectType == nil {
		return newObject
	}

	if objectType.Kind() == reflect.Ptr {
		newObject = reflect.New(objectType.Elem()).Interface().(T)
	} else {
		newObject = reflect.New(objectType).Interface().(T)
	}
	return newObject
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
