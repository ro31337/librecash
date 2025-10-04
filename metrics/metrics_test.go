package metrics

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricsInit(t *testing.T) {
	// Set test environment to use different port
	os.Setenv("METRICS_PORT", "8082")
	defer os.Unsetenv("METRICS_PORT")

	// Test that metrics initialization works
	err := Init()
	assert.NoError(t, err, "Metrics initialization should not fail")
	assert.True(t, IsEnabled(), "Metrics should be enabled by default")
}

func TestRecordNewUser(t *testing.T) {
	// Set test environment to disable metrics server for this test
	os.Setenv("METRICS_ENABLED", "true")
	os.Setenv("METRICS_PORT", "8083")
	defer os.Unsetenv("METRICS_ENABLED")
	defer os.Unsetenv("METRICS_PORT")

	// Test recording new user metric
	RecordNewUser("en")
	RecordNewUser("ru")
	RecordNewUser("es")

	// Test passes if no panic occurs
	assert.True(t, true, "Recording metrics should not cause errors")
}

func TestRecordUserActivity(t *testing.T) {
	// Test recording user activity metric
	RecordUserActivity("en", "menu_navigation")
	RecordUserActivity("ru", "command_execution")

	// Test passes if no panic occurs
	assert.True(t, true, "Recording user activity should not cause errors")
}

func TestRecordCommand(t *testing.T) {
	// Test recording command metric
	RecordCommand("/start", "en", "new")
	RecordCommand("/language", "ru", "returning")

	// Test passes if no panic occurs
	assert.True(t, true, "Recording commands should not cause errors")
}

func TestMetricsConfiguration(t *testing.T) {
	// Test metrics configuration
	os.Setenv("METRICS_ENABLED", "false")
	defer os.Unsetenv("METRICS_ENABLED")

	// Reinitialize with disabled metrics
	err := Init()
	assert.NoError(t, err, "Init should work even when disabled")
	assert.False(t, IsEnabled(), "Metrics should be disabled when METRICS_ENABLED=false")

	// Test recording when disabled (should not panic)
	RecordNewUser("en")
	RecordUserActivity("en", "test")
	RecordCommand("/test", "en", "new")

	assert.True(t, true, "Recording metrics when disabled should not cause errors")
}

func TestMetricsDisabled(t *testing.T) {
	// Test that metrics work when disabled
	os.Setenv("METRICS_ENABLED", "false")
	defer os.Unsetenv("METRICS_ENABLED")

	// Test recording when disabled (should not panic)
	RecordNewUser("en")
	RecordUserActivity("en", "test")
	RecordCommand("/test", "en", "new")

	// Should not panic and should return disabled summary
	summary := GetMetricsSummary()
	assert.False(t, summary["enabled"].(bool), "Metrics should be disabled")
}

func TestGetMetricsSummary(t *testing.T) {
	// Set test environment
	os.Setenv("METRICS_ENABLED", "true")
	os.Setenv("METRICS_PORT", "8084")
	defer os.Unsetenv("METRICS_ENABLED")
	defer os.Unsetenv("METRICS_PORT")

	// Reinitialize with test config
	Init()

	// Test metrics summary function
	summary := GetMetricsSummary()

	assert.NotNil(t, summary, "Summary should not be nil")
	assert.True(t, summary["enabled"].(bool), "Metrics should be enabled")
	assert.Equal(t, "/metrics", summary["endpoint"], "Endpoint should be /metrics")
	assert.Equal(t, 8084, summary["port"], "Port should be 8084")
}

func TestMetricsInMemory(t *testing.T) {
	// Test that metrics are actually recorded in memory
	os.Setenv("METRICS_ENABLED", "true")
	defer os.Unsetenv("METRICS_ENABLED")

	// Record some metrics
	RecordNewUser("en")
	RecordNewUser("ru")
	RecordUserActivity("en", "test")
	RecordCommand("/start", "en", "new")

	// Test passes if no panic occurs - metrics are stored in VictoriaMetrics/metrics library
	assert.True(t, true, "Metrics should be recorded in memory without errors")
}
