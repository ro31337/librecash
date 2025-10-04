package metrics

import (
	"log"
	"strconv"

	"github.com/VictoriaMetrics/metrics"
)

// RecordFanoutMessage records fanout messages sent to multiple users
func RecordFanoutMessage(messageType, languageCode string, success bool) {
	if !IsEnabled() {
		return
	}

	// VictoriaMetrics/metrics API: include labels in metric name
	metricName := `librecash_fanout_messages_total{message_type="` + messageType + `",language_code="` + languageCode + `",success="` + strconv.FormatBool(success) + `"}`
	counter := metrics.GetOrCreateCounter(metricName)
	counter.Inc()
	log.Printf("[METRICS] Fanout message: type=%s, language=%s, success=%t", messageType, languageCode, success)
}
