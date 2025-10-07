package library

import (
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (status *Status) FindOrDefaultCondition(def metav1.Condition) (cond *metav1.Condition, defaulted bool) {
	cond = meta.FindStatusCondition(status.Conditions, def.Type)
	if cond != nil {
		return cond, false
	}

	return &def, true
}

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
	ControllerResourceType ControllerResource,
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
