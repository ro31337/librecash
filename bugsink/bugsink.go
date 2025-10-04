package bugsink

import (
	"fmt"
	"librecash/config"
	"log"
	"time"

	"github.com/getsentry/sentry-go"
)

var (
	initialized bool
	enabled     bool
)

// Initialize BugSink error tracking
func Init() error {
	cfg := config.C()

	// Check if BugSink is enabled
	if !cfg.BugSink_Enabled {
		log.Println("[BUGSINK] BugSink error tracking is disabled")
		enabled = false
		return nil
	}

	if cfg.BugSink_DSN == "" {
		log.Println("[BUGSINK] BugSink DSN not provided, disabling error tracking")
		enabled = false
		return nil
	}

	log.Printf("[BUGSINK] Initializing BugSink error tracking...")
	log.Printf("[BUGSINK] Environment: %s", cfg.BugSink_Environment)
	log.Printf("[BUGSINK] Release: %s", cfg.BugSink_Release)

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.BugSink_DSN,
		Debug:            cfg.BugSink_Environment == "development",
		Environment:      cfg.BugSink_Environment,
		Release:          cfg.BugSink_Release,
		AttachStacktrace: true,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Add custom tags to all events
			if event.Tags == nil {
				event.Tags = make(map[string]string)
			}
			event.Tags["service"] = "librecash"
			event.Tags["component"] = "telegram-bot"
			return event
		},
	})

	if err != nil {
		return fmt.Errorf("failed to initialize BugSink: %w", err)
	}

	initialized = true
	enabled = true
	log.Println("[BUGSINK] BugSink error tracking initialized successfully")

	// Test connection by sending a test message
	CaptureMessage("BugSink initialized successfully", map[string]interface{}{
		"component": "initialization",
		"test":      true,
	})

	return nil
}

// IsEnabled returns true if BugSink is enabled and initialized
func IsEnabled() bool {
	return enabled && initialized
}

// CaptureError captures an error with additional context
func CaptureError(err error, context map[string]interface{}) {
	if !IsEnabled() {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		// Add context data
		for key, value := range context {
			scope.SetContext(key, map[string]interface{}{key: value})
		}

		// Add level based on error type
		scope.SetLevel(sentry.LevelError)

		// Capture the exception
		sentry.CaptureException(err)
	})
}

// CaptureMessage captures a message with additional context
func CaptureMessage(message string, context map[string]interface{}) {
	if !IsEnabled() {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		// Add context data
		for key, value := range context {
			scope.SetContext(key, map[string]interface{}{key: value})
		}

		// Add level
		scope.SetLevel(sentry.LevelInfo)

		// Capture the message
		sentry.CaptureMessage(message)
	})
}

// CapturePanic recovers from a panic and reports it to BugSink
func CapturePanic() {
	if !IsEnabled() {
		return
	}

	if err := recover(); err != nil {
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetLevel(sentry.LevelFatal)
			scope.SetContext("panic", map[string]interface{}{
				"recovered_value": fmt.Sprintf("%v", err),
			})
			sentry.CaptureException(fmt.Errorf("panic recovered: %v", err))
		})

		// Re-panic after capturing
		panic(err)
	}
}

// Recover captures panics but doesn't re-panic them
func Recover() {
	if err := recover(); err != nil {
		if IsEnabled() {
			sentry.WithScope(func(scope *sentry.Scope) {
				scope.SetLevel(sentry.LevelFatal)
				scope.SetContext("panic", map[string]interface{}{
					"recovered_value": fmt.Sprintf("%v", err),
				})
				sentry.CaptureException(fmt.Errorf("panic recovered: %v", err))
			})
		}

		log.Printf("[BUGSINK] Panic recovered and reported: %v", err)
	}
}

// SetUser sets user context for subsequent error reports
func SetUser(userID int64, username string) {
	if !IsEnabled() {
		return
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID:       fmt.Sprintf("%d", userID),
			Username: username,
		})
	})
}

// SetTag sets a tag for subsequent error reports
func SetTag(key, value string) {
	if !IsEnabled() {
		return
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag(key, value)
	})
}

// AddBreadcrumb adds a breadcrumb to the error trail
func AddBreadcrumb(message string, category string, level sentry.Level) {
	if !IsEnabled() {
		return
	}

	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Message:   message,
		Category:  category,
		Level:     level,
		Timestamp: time.Now(),
	})
}

// Flush flushes any pending events to BugSink
func Flush(timeout time.Duration) bool {
	if !IsEnabled() {
		return true
	}

	return sentry.Flush(timeout)
}

// Close gracefully shuts down BugSink
func Close() {
	if !IsEnabled() {
		return
	}

	log.Println("[BUGSINK] Flushing pending events before shutdown...")
	Flush(2 * time.Second)
	log.Println("[BUGSINK] BugSink error tracking closed")
}
