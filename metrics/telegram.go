package metrics

import (
	"log"

	"github.com/VictoriaMetrics/metrics"
)

// RecordTelegramMessage records Telegram message delivery tracking
func RecordTelegramMessage(messageType, status, errorCode string) {
	if !IsEnabled() {
		return
	}

	// VictoriaMetrics/metrics API: include labels in metric name
	metricName := `librecash_telegram_messages_total{message_type="` + messageType + `",status="` + status + `",error_code="` + errorCode + `"}`
	counter := metrics.GetOrCreateCounter(metricName)
	counter.Inc()
	log.Printf("[METRICS] Telegram message: type=%s, status=%s, error=%s", messageType, status, errorCode)
}
