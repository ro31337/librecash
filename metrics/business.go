package metrics

import (
	"log"

	"github.com/VictoriaMetrics/metrics"
)

// Business metrics for LibreCash
// Note: VictoriaMetrics/metrics uses a different API than Prometheus
// We create metrics dynamically with labels included in the metric name

// RecordNewUser records a new user registration
// This should only be called when a user runs /start for the first time
func RecordNewUser(languageCode string) {
	if !IsEnabled() {
		return
	}

	// VictoriaMetrics/metrics API: include labels in metric name
	metricName := `librecash_users_new_total{language_code="` + languageCode + `"}`
	counter := metrics.GetOrCreateCounter(metricName)
	counter.Inc()
	log.Printf("[METRICS] New user registered: language=%s", languageCode)
}

// RecordUserActivity records user activity (any interaction)
func RecordUserActivity(languageCode, actionType string) {
	if !IsEnabled() {
		return
	}

	// VictoriaMetrics/metrics API: include labels in metric name
	metricName := `librecash_users_active_total{language_code="` + languageCode + `",action_type="` + actionType + `"}`
	counter := metrics.GetOrCreateCounter(metricName)
	counter.Inc()
	log.Printf("[METRICS] User activity: language=%s, action=%s", languageCode, actionType)
}

// RecordCommand records slash command usage
func RecordCommand(command, languageCode, userType string) {
	if !IsEnabled() {
		return
	}

	// VictoriaMetrics/metrics API: include labels in metric name
	metricName := `librecash_slash_commands_total{command="` + command + `",language_code="` + languageCode + `",user_type="` + userType + `"}`
	counter := metrics.GetOrCreateCounter(metricName)
	counter.Inc()
	log.Printf("[METRICS] Slash command executed: command=%s, language=%s, user_type=%s", command, languageCode, userType)
}

// GetMetricsSummary returns a summary of current metrics (for debugging)
func GetMetricsSummary() map[string]interface{} {
	if !IsEnabled() {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	return map[string]interface{}{
		"enabled":      true,
		"new_users":    "tracked via librecash_users_new_total",
		"active_users": "tracked via librecash_users_active_total",
		"commands":     "tracked via librecash_commands_total",
		"endpoint":     config.Path,
		"port":         config.Port,
	}
}
