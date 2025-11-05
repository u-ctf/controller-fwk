package instrument

import (
	"context"
	"time"
)

type MergedContext struct {
	ctx1 context.Context
	ctx2 context.Context
}

var _ context.Context = &MergedContext{}

func NewMergedContext(ctx1, ctx2 context.Context) *MergedContext {
	return &MergedContext{
		ctx1: ctx1,
		ctx2: ctx2,
	}
}

func (m *MergedContext) Deadline() (deadline time.Time, ok bool) {
	d1, ok1 := m.ctx1.Deadline()
	d2, ok2 := m.ctx2.Deadline()

	if !ok1 && !ok2 {
		return time.Time{}, false
	}
	if !ok1 {
		return d2, true
	}
	if !ok2 {
		return d1, true
	}
	if d1.Before(d2) {
		return d1, true
	}
	return d2, true
}

func (m *MergedContext) Done() <-chan struct{} {
	done1 := m.ctx1.Done()
	done2 := m.ctx2.Done()

	if done1 == nil && done2 == nil {
		return nil
	}
	if done1 == nil {
		return done2
	}
	if done2 == nil {
		return done1
	}

	mergedDone := make(chan struct{})
	go func() {
		select {
		case <-done1:
		case <-done2:
		}
		close(mergedDone)
	}()
	return mergedDone
}

func (m *MergedContext) Err() error {
	err1 := m.ctx1.Err()
	err2 := m.ctx2.Err()

	if err1 != nil {
		return err1
	}
	return err2
}

func (m *MergedContext) Value(key any) any {
	val1 := m.ctx1.Value(key)
	if val1 != nil {
		return val1
	}
	return m.ctx2.Value(key)
}
