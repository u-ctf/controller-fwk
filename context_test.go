package ctrlfwk

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

type UserData struct {
	T int
}

func IOnlyTakeContext(ctx Context[*corev1.Secret]) {}

func IOnlyTakeContextWithData(ctx ContextWithData[*corev1.Secret, *UserData]) {
	ctx.Data().T = 5
}

func ITakeContextButWantData(ctx Context[*corev1.Secret]) {
	dCtx := ToContextWithData[*UserData](ctx)
	_ = dCtx.Data().T
}

func TestReconciliationWithStruct(t *testing.T) {
	r := NewContextWithData[*corev1.Secret](context.Background(), &UserData{})

	IOnlyTakeContext(r)
	IOnlyTakeContextWithData(r)
	ITakeContextButWantData(r)

	if r.Data().T != 5 {
		t.Errorf("expected data T to be 5, got %d", r.Data().T)
	}
}
