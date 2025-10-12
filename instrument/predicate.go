package instrument

import (
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type instrumentedPredicate struct {
	tracer    *instrumenter
	inner     predicate.Predicate
	innerName string
}

var _ predicate.Predicate = &instrumentedPredicate{}

func NewInstrumentedPredicate(tracer *instrumenter, inner predicate.Predicate) predicate.Predicate {
	return &instrumentedPredicate{
		tracer:    tracer,
		inner:     inner,
		innerName: fmt.Sprintf("%T", inner),
	}
}

// Create returns true if the Create event should be processed
func (p *instrumentedPredicate) Create(event event.TypedCreateEvent[client.Object]) bool {
	hub := p.tracer.GetOrCreateSentryHubForEvent(event)
	ctx := context.Background()
	ctx = sentry.SetHubOnContext(ctx, hub)

	span := sentry.StartSpan(ctx, fmt.Sprintf("event.create.predicate.%T", p.inner))
	defer span.Finish()

	result := p.inner.Create(event)

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "controller.event",
		Message:  "Processed create event",
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"result": result,
		},
	}, nil)

	return result
}

// Delete returns true if the Delete event should be processed
func (p *instrumentedPredicate) Delete(event event.TypedDeleteEvent[client.Object]) bool {
	hub := p.tracer.GetOrCreateSentryHubForEvent(event)
	ctx := context.Background()
	ctx = sentry.SetHubOnContext(ctx, hub)

	span := sentry.StartSpan(ctx, fmt.Sprintf("event.delete.predicate.%T", p.inner))
	defer span.Finish()

	result := p.inner.Delete(event)

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "controller.event",
		Message:  "Processed delete event",
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"result": result,
		},
	}, nil)

	return result
}

// Update returns true if the Update event should be processed
func (p *instrumentedPredicate) Update(event event.TypedUpdateEvent[client.Object]) bool {
	hub := p.tracer.GetOrCreateSentryHubForEvent(event)
	ctx := context.Background()
	ctx = sentry.SetHubOnContext(ctx, hub)

	span := sentry.StartSpan(ctx, fmt.Sprintf("event.update.predicate.%T", p.inner))
	defer span.Finish()

	result := p.inner.Update(event)

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "controller.event",
		Message:  "Processed update event",
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"result": result,
		},
	}, nil)

	return result
}

// Generic returns true if the Generic event should be processed
func (p *instrumentedPredicate) Generic(event event.TypedGenericEvent[client.Object]) bool {
	hub := p.tracer.GetOrCreateSentryHubForEvent(event)
	ctx := context.Background()
	ctx = sentry.SetHubOnContext(ctx, hub)

	span := sentry.StartSpan(ctx, fmt.Sprintf("event.generic.predicate.%T", p.inner))
	defer span.Finish()

	result := p.inner.Generic(event)

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "controller.event",
		Message:  "Processed generic event",
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"result": result,
		},
	}, nil)

	return result
}
