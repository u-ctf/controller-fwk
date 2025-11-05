package instrument

import (
	"context"
	"testing"
)

func TestMergeContext(t *testing.T) {
	contextA, cancelA := context.WithTimeout(context.Background(), 0)
	contextA = context.WithValue(contextA, "key1", "value1")
	contextA = context.WithValue(contextA, "key2", "value2")

	contextB := context.WithValue(context.Background(), "keyA", "valueA")
	contextB = context.WithValue(contextB, "key3", "value3")
	contextB = context.WithValue(contextB, "key4", "value4")

	ctx := NewMergedContext(contextA, contextB)

	if ctx.Value("key1") != "value1" {
		t.Errorf("expected key1 to be value1, got %v", ctx.Value("key1"))
	}
	if ctx.Value("key2") != "value2" {
		t.Errorf("expected key2 to be value2, got %v", ctx.Value("key2"))
	}
	if ctx.Value("key3") != "value3" {
		t.Errorf("expected key3 to be value3, got %v", ctx.Value("key3"))
	}
	if ctx.Value("key4") != "value4" {
		t.Errorf("expected key4 to be value4, got %v", ctx.Value("key4"))
	}
	if ctx.Value("keyA") != "valueA" {
		t.Errorf("expected keyA to be valueA, got %v", ctx.Value("keyA"))
	}

	cancelA()
}
