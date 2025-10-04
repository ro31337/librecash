package metrics

import (
	"fmt"
	"librecash/objects"
	"log"

	"github.com/VictoriaMetrics/metrics"
)

// RecordMenuTransition records user menu state transitions using numeric menu IDs
func RecordMenuTransition(fromState, toState objects.MenuId, languageCode string) {
	if !IsEnabled() {
		return
	}

	// Use numeric menu IDs as strings for labels
	fromStateStr := fmt.Sprintf("%d", fromState)
	toStateStr := fmt.Sprintf("%d", toState)

	// VictoriaMetrics/metrics API: include labels in metric name
	metricName := `librecash_menu_transitions_total{from_state="` + fromStateStr + `",to_state="` + toStateStr + `",language_code="` + languageCode + `"}`
	counter := metrics.GetOrCreateCounter(metricName)
	counter.Inc()
	log.Printf("[METRICS] Menu transition: from=%s, to=%s, language=%s", fromStateStr, toStateStr, languageCode)
}
