package tracing

import (
	"os"

	"github.com/getsentry/sentry-go"
)

func InitSentry() error {
	return sentry.Init(sentry.ClientOptions{
		Dsn:              os.Getenv("SENTRY_DSN"),
		Debug:            true,
		SendDefaultPII:   true,
		EnableTracing:    true,
		TracesSampleRate: 1.0,
	})
}
