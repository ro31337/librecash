package bugsink

import (
	"errors"
	"librecash/config"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

func TestInitDisabled(t *testing.T) {
	// Save original config
	originalConfig := *config.C()
	defer func() {
		// Restore original config (this is a simplified approach)
		*config.C() = originalConfig
	}()

	// Test with BugSink disabled
	cfg := config.C()
	cfg.BugSink_Enabled = false

	err := Init()
	if err != nil {
		t.Errorf("Init() with disabled BugSink should not return error, got: %v", err)
	}

	if IsEnabled() {
		t.Error("BugSink should be disabled when BugSink_Enabled is false")
	}
}

func TestInitMissingDSN(t *testing.T) {
	// Save original config
	originalConfig := *config.C()
	defer func() {
		*config.C() = originalConfig
	}()

	// Test with enabled but missing DSN
	cfg := config.C()
	cfg.BugSink_Enabled = true
	cfg.BugSink_DSN = ""

	err := Init()
	if err != nil {
		t.Errorf("Init() with missing DSN should not return error, got: %v", err)
	}

	if IsEnabled() {
		t.Error("BugSink should be disabled when DSN is empty")
	}
}

func TestCaptureErrorWhenDisabled(t *testing.T) {
	// Ensure BugSink is disabled
	initialized = false
	enabled = false

	// This should not panic or cause issues
	testErr := errors.New("test error")
	CaptureError(testErr, map[string]interface{}{
		"test": true,
	})

	// If we reach here, the function handled the disabled state correctly
}

func TestCaptureMessageWhenDisabled(t *testing.T) {
	// Ensure BugSink is disabled
	initialized = false
	enabled = false

	// This should not panic or cause issues
	CaptureMessage("test message", map[string]interface{}{
		"test": true,
	})

	// If we reach here, the function handled the disabled state correctly
}

func TestRecoverWhenDisabled(t *testing.T) {
	// Ensure BugSink is disabled
	initialized = false
	enabled = false

	// This should not panic or cause issues
	defer func() {
		// If Recover() works correctly, it should catch the panic
		// and not let it propagate to this defer
		if r := recover(); r != nil {
			t.Errorf("Recover() should have caught the panic when BugSink is disabled, but it propagated: %v", r)
		}
	}()

	// Call Recover() in a separate function to test it
	func() {
		defer Recover()
		// Trigger a panic
		panic("test panic")
	}()

	// If we reach here, Recover() successfully caught the panic
}

func TestSetUserWhenDisabled(t *testing.T) {
	// Ensure BugSink is disabled
	initialized = false
	enabled = false

	// This should not panic or cause issues
	SetUser(123, "testuser")

	// If we reach here, the function handled the disabled state correctly
}

func TestSetTagWhenDisabled(t *testing.T) {
	// Ensure BugSink is disabled
	initialized = false
	enabled = false

	// This should not panic or cause issues
	SetTag("test", "value")

	// If we reach here, the function handled the disabled state correctly
}

func TestAddBreadcrumbWhenDisabled(t *testing.T) {
	// Ensure BugSink is disabled
	initialized = false
	enabled = false

	// This should not panic or cause issues
	AddBreadcrumb("test message", "test", sentry.LevelInfo)

	// If we reach here, the function handled the disabled state correctly
}

func TestFlushWhenDisabled(t *testing.T) {
	// Ensure BugSink is disabled
	initialized = false
	enabled = false

	// Should return true when disabled
	result := Flush(1 * time.Second)
	if !result {
		t.Error("Flush() should return true when BugSink is disabled")
	}
}

func TestCloseWhenDisabled(t *testing.T) {
	// Ensure BugSink is disabled
	initialized = false
	enabled = false

	// This should not panic or cause issues
	Close()

	// If we reach here, the function handled the disabled state correctly
}

// Integration test - only run if BugSink is properly configured
func TestIntegrationWithMockDSN(t *testing.T) {
	// Save original config
	originalConfig := *config.C()
	defer func() {
		// Clean up
		Close()
		*config.C() = originalConfig
	}()

	// Test with mock DSN (this won't actually send to BugSink but tests the SDK)
	cfg := config.C()
	cfg.BugSink_Enabled = true
	cfg.BugSink_DSN = "http://test@localhost:5577/1"
	cfg.BugSink_Environment = "test"
	cfg.BugSink_Release = "test-1.0.0"

	// This might fail if BugSink is not running, but we can test the initialization logic
	err := Init()
	if err != nil {
		// Expected if BugSink is not running, so we'll skip the rest
		t.Skipf("BugSink not available for integration test: %v", err)
		return
	}

	if !IsEnabled() {
		t.Error("BugSink should be enabled after successful initialization")
	}

	// Test error capture
	testErr := errors.New("integration test error")
	CaptureError(testErr, map[string]interface{}{
		"test":      "integration",
		"component": "test",
		"timestamp": time.Now(),
	})

	// Test message capture
	CaptureMessage("integration test message", map[string]interface{}{
		"test":      "integration",
		"component": "test",
	})

	// Test user context
	SetUser(12345, "testuser")

	// Test tags
	SetTag("test", "integration")

	// Test breadcrumbs
	AddBreadcrumb("test breadcrumb", "test", sentry.LevelInfo)

	// Test flush
	flushed := Flush(2 * time.Second)
	if !flushed {
		t.Log("Flush returned false - events might not have been sent")
	}

	// Test close
	Close()
}
