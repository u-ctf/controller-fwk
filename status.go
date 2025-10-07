package library

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Status struct {
	Dependencies   ObjectReferenceList `json:"dependencies,omitempty"`
	ChildResources ObjectReferenceList `json:"childResources,omitempty"`
	Conditions     []metav1.Condition  `json:"conditions,omitempty"`
	LastStep       string              `json:"lastStep,omitempty"`
}
