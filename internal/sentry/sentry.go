package sentry

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	sentryfiber "github.com/getsentry/sentry-go/fiber"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func InitSentry(dsn string, environment string, release string) (fiber.Handler, error) {
	if dsn == "" {
		log.Warn().Msg("Sentry DSN not configured; error tracking disabled")
		return nil, nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      environment,
		Release:          release,
		TracesSampleRate: 0.1,
		EnableLogs:       true,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			if event.Level == sentry.LevelError || event.Level == sentry.LevelFatal {
				if event.Tags == nil {
					event.Tags = make(map[string]string)
				}
				event.Tags["error_type"] = "unhandled_panic"
			}
			return event
		},
		TracesSampler: func(ctx sentry.SamplingContext) float64 {
			return 0.1
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize Sentry: %w", err)
	}

	log.Info().Str("environment", environment).Msg("Sentry initialized for error tracking")
	handler := sentryfiber.New(sentryfiber.Options{
		Repanic:         true,
		WaitForDelivery: false,
		Timeout:         5 * time.Second,
	})

	return handler, nil
}

func GetHubFromContext(c *fiber.Ctx) *sentry.Hub {
	return sentryfiber.GetHubFromContext(c)
}

func CaptureException(c *fiber.Ctx, err error) {
	if hub := GetHubFromContext(c); hub != nil {
		hub.CaptureException(err)
	}
}

func CaptureMessage(c *fiber.Ctx, message string) {
	if hub := GetHubFromContext(c); hub != nil {
		hub.CaptureMessage(message)
	}
}

func SetTag(c *fiber.Ctx, key, value string) {
	if hub := GetHubFromContext(c); hub != nil {
		hub.Scope().SetTag(key, value)
	}
}

func SetExtra(c *fiber.Ctx, key string, value interface{}) {
	if hub := GetHubFromContext(c); hub != nil {
		hub.Scope().SetExtra(key, value)
	}
}

func CaptureErrorWithContext(c *fiber.Ctx, err error, statusCode int, operation string) {
	if hub := GetHubFromContext(c); hub != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			scope.SetTag("operation", operation)
			scope.SetTag("http_status", fmt.Sprintf("%d", statusCode))
			scope.SetExtra("request_method", c.Method())
			scope.SetExtra("request_path", c.Path())
			hub.CaptureException(err)
		})
	}
}

func CaptureExceptionWithContext(ctx context.Context, err error, operation string) {
	if span := GetSpanFromContext(ctx); span != nil {
		span.SetTag("error", "true")
		span.SetTag("operation", operation)
	}

	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			scope.SetTag("operation", operation)
			hub.CaptureException(err)
		})
	}
}

func StartTransaction(c *fiber.Ctx, name string) *sentry.Span {
	// Handle nil context for background jobs
	if c == nil {
		return StartBackgroundTransaction(context.Background(), name)
	}

	options := []sentry.SpanOption{
		sentry.WithOpName("http.server"),
		sentry.WithTransactionSource(sentry.SourceURL),
	}
	tx := sentry.StartTransaction(c.Context(), name, options...)
	return tx
}

// StartBackgroundTransaction creates a transaction for background jobs/workers
func StartBackgroundTransaction(ctx context.Context, name string) *sentry.Span {
	options := []sentry.SpanOption{
		sentry.WithOpName("task"),
		sentry.WithTransactionSource(sentry.SourceTask),
	}
	tx := sentry.StartTransaction(ctx, name, options...)
	return tx
}

func StartSpan(ctx context.Context, operation string, description string) *sentry.Span {
	span := sentry.StartSpan(ctx, operation)
	span.Description = description
	return span
}

func GetTransactionFromContext(ctx context.Context) *sentry.Span {
	return sentry.TransactionFromContext(ctx)
}

func GetSpanFromContext(ctx context.Context) *sentry.Span {
	return sentry.SpanFromContext(ctx)
}

func Flush(timeout time.Duration) bool {
	return sentry.Flush(timeout)
}
