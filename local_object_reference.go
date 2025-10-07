package library

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ObjectReference represents a child resource of a parent resource.
// It contains metadata about the child resource, including its API version,
// kind, group, name, UID, status, and reason.
// This struct is used to track the status of child resources in the parent resource's status.
// It is typically used in the status subresource of a Kubernetes custom resource.
type ObjectReference struct {
	APIVersion         string                 `json:"apiVersion"`
	Kind               string                 `json:"kind"`
	Group              string                 `json:"group"`
	Name               string                 `json:"name"`
	Namespace          string                 `json:"namespace"`
	Status             metav1.ConditionStatus `json:"status"`
	TransitionTime     metav1.Time            `json:"transitionTime"`
	ObservedGeneration int64                  `json:"observedGeneration"`

	// +optional
	UID string `json:"uid,omitempty"`
	// +optional
	Reason string `json:"reason,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
}

func (obj *ObjectReference) GroupVersionKind() schema.GroupVersionKind {
	gv, _ := schema.ParseGroupVersion(obj.APIVersion)
	return schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    obj.Kind,
	}
}

func NewObjectReference[
	ControllerResourceType ControllerResource,
](reconciler Reconciler[ControllerResourceType], ref client.Object, status metav1.ConditionStatus, generation int64) (*ObjectReference, error) {
	gvks, _, err := reconciler.Scheme().ObjectKinds(ref)
	if err != nil {
		return nil, err
	}
	if len(gvks) == 0 {
		return nil, fmt.Errorf("no GVK found for object %T", ref)
	}
	gvk := gvks[0]

	return &ObjectReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Group:              gvk.Group,
		Name:               ref.GetName(),
		Namespace:          ref.GetNamespace(),
		UID:                string(ref.GetUID()),
		Status:             status,
		ObservedGeneration: generation,
	}, nil
}

func EmptyObjectReference[
	ControllerResourceType ControllerResource,
](reconciler Reconciler[ControllerResourceType], ref client.Object) (*ObjectReference, error) {
	gvks, _, err := reconciler.Scheme().ObjectKinds(ref)
	if err != nil {
		return nil, err
	}
	if len(gvks) == 0 {
		return nil, fmt.Errorf("no GVK found for object %T", ref)
	}
	gvk := gvks[0]

	return &ObjectReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Group:      gvk.Group,
		Name:       ref.GetName(),
		Namespace:  ref.GetNamespace(),
		UID:        string(ref.GetUID()),
	}, nil
}

func (obj *ObjectReference) SetReason(reason string) {
	obj.Reason = reason
}

func (obj *ObjectReference) Equal(other *ObjectReference) bool {
	if obj == nil || other == nil {
		return false
	}
	return obj.APIVersion == other.APIVersion &&
		obj.Kind == other.Kind &&
		obj.Group == other.Group &&
		obj.Name == other.Name &&
		obj.Namespace == other.Namespace
}

func (obj *ObjectReference) Changed(other *ObjectReference) bool {
	return obj.APIVersion != other.APIVersion ||
		obj.Kind != other.Kind ||
		obj.Group != other.Group ||
		obj.Name != other.Name ||
		obj.Namespace != other.Namespace ||
		obj.UID != other.UID ||
		obj.Status != other.Status ||
		obj.ObservedGeneration != other.ObservedGeneration ||
		obj.Reason != other.Reason ||
		obj.Message != other.Message
}

// ObjectReferenceList is a list of ChildResource.
// It is used to represent a collection of child resources in the status of a parent resource.
// This struct is typically used in the status subresource of a Kubernetes custom resource.
type ObjectReferenceList []ObjectReference

// CRUD Operations for ChildResourceList, the unicity should be on the GK/name, not on the UID.
// Set adds a child resource to the list if it doesn't already exist.
func (list *ObjectReferenceList) Set(obj *ObjectReference) bool {
	obj.TransitionTime = metav1.Now()

	for i, c := range *list {
		if c.Group == obj.Group && c.Kind == obj.Kind && c.Name == obj.Name {
			previous := (*list)[i]
			changed := previous.Changed(obj)
			if changed {
				// fmt.Printf("%+#v %+#v\n", previous, obj)
				(*list)[i] = *obj
			}
			return changed
		}
	}
	*list = append(*list, *obj)
	return true
}

// Remove removes a child resource from the list by its GK/name.
func (list *ObjectReferenceList) Remove(obj *ObjectReference) bool {
	for i, c := range *list {
		if c.Group == obj.Group && c.Kind == obj.Kind && c.Name == obj.Name {
			*list = append((*list)[:i], (*list)[i+1:]...)
			return true
		}
	}
	return false
}

// Get retrieves a child resource from the list by its GK/name.
func (list *ObjectReferenceList) Get(group, kind, name string) (*ObjectReference, bool) {
	for _, c := range *list {
		if c.Group == group && c.Kind == kind && c.Name == name {
			return &c, true
		}
	}
	return nil, false
}
