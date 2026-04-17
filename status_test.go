package ctrlfwk_test

import (
	"errors"
	"strings"
	"testing"

	ctrlfwk "github.com/u-ctf/controller-fwk"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type readyStatusTestObjectStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type readyStatusTestObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            readyStatusTestObjectStatus `json:"status,omitempty"`
}

func (obj *readyStatusTestObject) DeepCopyObject() runtime.Object {
	if obj == nil {
		return nil
	}

	copy := *obj
	copy.ObjectMeta = *obj.ObjectMeta.DeepCopy()
	if obj.Status.Conditions != nil {
		copy.Status.Conditions = append([]metav1.Condition(nil), obj.Status.Conditions...)
	}

	return &copy
}

func TestSetReadyConditionFromResult_SetsReadyTrueOnSuccess(t *testing.T) {
	obj := &readyStatusTestObject{ObjectMeta: metav1.ObjectMeta{Generation: 7}}

	changed, err := ctrlfwk.SetReadyConditionFromResult[*readyStatusTestObject](nil)(obj, "reconcile resources", ctrlfwk.ResultSuccess())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !changed {
		t.Fatal("expected ready condition to be updated")
	}

	condition := meta.FindStatusCondition(obj.Status.Conditions, "Ready")
	if condition == nil {
		t.Fatal("expected Ready condition to be present")
	}
	if condition.Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready condition status True, got %s", condition.Status)
	}
	if condition.Reason != "Reconciled" {
		t.Fatalf("expected Ready condition reason %q, got %q", "Reconciled", condition.Reason)
	}
	if condition.Message != "The resource is ready" {
		t.Fatalf("expected Ready condition message %q, got %q", "The resource is ready", condition.Message)
	}
	if condition.ObservedGeneration != 7 {
		t.Fatalf("expected observed generation 7, got %d", condition.ObservedGeneration)
	}
}

func TestSetReadyConditionFromResult_SetsReadyFalseOnError(t *testing.T) {
	obj := &readyStatusTestObject{ObjectMeta: metav1.ObjectMeta{Generation: 3}}
	stepErr := errors.New("dependency not ready")

	changed, err := ctrlfwk.SetReadyConditionFromResult[*readyStatusTestObject](nil)(obj, "resolve dependencies", ctrlfwk.ResultInError(stepErr))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !changed {
		t.Fatal("expected ready condition to be updated")
	}

	condition := meta.FindStatusCondition(obj.Status.Conditions, "Ready")
	if condition == nil {
		t.Fatal("expected Ready condition to be present")
	}
	if condition.Status != metav1.ConditionFalse {
		t.Fatalf("expected Ready condition status False, got %s", condition.Status)
	}
	if condition.Reason != "ReconciliationFailed" {
		t.Fatalf("expected Ready condition reason %q, got %q", "ReconciliationFailed", condition.Reason)
	}
	if !strings.Contains(condition.Message, "resolve dependencies") {
		t.Fatalf("expected Ready condition message to mention step name, got %q", condition.Message)
	}
	if !strings.Contains(condition.Message, stepErr.Error()) {
		t.Fatalf("expected Ready condition message to mention error, got %q", condition.Message)
	}
	if condition.ObservedGeneration != 3 {
		t.Fatalf("expected observed generation 3, got %d", condition.ObservedGeneration)
	}
}
