package sentry

import (
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
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			return event
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

func Flush(timeout time.Duration) bool {
	return sentry.Flush(timeout)
}
