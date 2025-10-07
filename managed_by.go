package library

import (
	"context"
	"encoding/json"
	"slices"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	AnnotationRef = "multi.ch/managed-by"
)

type ManagedBy struct {
	Name      string                  `json:"name"`
	Namespace string                  `json:"namespace"`
	GVK       schema.GroupVersionKind `json:"gvk"`
}

// GetManagedBy returns the list of resource that manages the current object
func GetManagedBy(obj client.Object) ([]ManagedBy, error) {
	annotations := obj.GetAnnotations()
	v, ok := annotations[AnnotationRef]
	if !ok {
		return []ManagedBy{}, nil
	}

	var out []ManagedBy
	err := json.Unmarshal([]byte(v), &out)
	if err != nil {
		return nil, err
	}

	return out, err
}

func AddManagedBy(obj client.Object, controlledBy client.Object, scheme *runtime.Scheme) (changed bool, err error) {
	gvk, err := apiutil.GVKForObject(controlledBy, scheme)
	if err != nil {
		return false, err
	}

	references, err := GetManagedBy(obj)
	if err != nil {
		return false, err
	}

	// Early return if ref is already present
	ref := ManagedBy{
		Name:      controlledBy.GetName(),
		Namespace: controlledBy.GetNamespace(),
		GVK:       gvk,
	}
	if slices.Contains(references, ref) {
		return false, nil
	}

	references = append(references, ref)

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotationValue, err := json.Marshal(references)
	if err != nil {
		return false, err
	}

	annotations[AnnotationRef] = string(annotationValue)

	obj.SetAnnotations(annotations)

	return true, nil
}

func RemoveManagedBy(obj client.Object, controlledBy client.Object, scheme *runtime.Scheme) (changed bool, err error) {
	gvk, err := apiutil.GVKForObject(controlledBy, scheme)
	if err != nil {
		return false, err
	}

	references, err := GetManagedBy(obj)
	if err != nil {
		return false, err
	}

	// Early return if ref is not present
	ref := ManagedBy{
		Name:      controlledBy.GetName(),
		Namespace: controlledBy.GetNamespace(),
		GVK:       gvk,
	}
	if !slices.Contains(references, ref) {
		return false, nil
	}

	references = slices.DeleteFunc(references, func(val ManagedBy) bool {
		return val == ref
	})

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotationValue, err := json.Marshal(references)
	if err != nil {
		return false, err
	}

	annotations[AnnotationRef] = string(annotationValue)

	obj.SetAnnotations(annotations)

	return true, nil
}

func GetManagedByReconcileRequests(ownedBy client.Object, scheme *runtime.Scheme) (func(ctx context.Context, obj client.Object) []reconcile.Request, error) {
	gvk, err := apiutil.GVKForObject(ownedBy, scheme)
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		references, err := GetManagedBy(obj)
		if err != nil {
			return nil
		}

		var requests []reconcile.Request

		for _, ref := range references {
			if ref.GVK != gvk {
				continue
			}

			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: ref.Name,
					Name:      ref.Namespace,
				},
			})
		}
		return requests
	}, err
}
