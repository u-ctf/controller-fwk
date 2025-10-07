package library

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func (child *ObjectReference) DeepCopyInto(out *ObjectReference) {
	*out = *child
	out.APIVersion = child.APIVersion
	out.Kind = child.Kind
	out.Group = child.Group
	out.Name = child.Name
	out.UID = child.UID
	out.Status = child.Status
	out.Reason = child.Reason
}

func (child *ObjectReference) DeepCopy() *ObjectReference {
	if child == nil {
		return nil
	}
	out := new(ObjectReference)
	child.DeepCopyInto(out)
	return out
}

func (list *ObjectReferenceList) DeepCopyInto(out *ObjectReferenceList) {
	*out = make(ObjectReferenceList, len(*list))
	for i := range *list {
		(*list)[i].DeepCopyInto(&(*out)[i])
	}
}

func (list *ObjectReferenceList) DeepCopy() *ObjectReferenceList {
	if list == nil {
		return nil
	}
	out := new(ObjectReferenceList)
	list.DeepCopyInto(out)
	return out
}

func (status *Status) DeepCopyInto(out *Status) {
	*out = *status
	if status.Conditions != nil {
		in, out := &status.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if status.ChildResources != nil {
		in, out := &status.ChildResources, &out.ChildResources
		*out = make(ObjectReferenceList, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (status *Status) DeepCopy() *Status {
	if status == nil {
		return nil
	}
	out := new(Status)
	status.DeepCopyInto(out)
	return out
}
