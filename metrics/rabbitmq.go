package metrics

import (
	"log"
	"strconv"

	"github.com/VictoriaMetrics/metrics"
)

// RecordRabbitMQMessage records RabbitMQ message processing
func RecordRabbitMQMessage(operation, queue string, success bool) {
	if !IsEnabled() {
		return
	}

	// VictoriaMetrics/metrics API: include labels in metric name
	metricName := `librecash_rabbitmq_messages_total{operation="` + operation + `",queue="` + queue + `",success="` + strconv.FormatBool(success) + `"}`
	counter := metrics.GetOrCreateCounter(metricName)
	counter.Inc()
	log.Printf("[METRICS] RabbitMQ message: operation=%s, queue=%s, success=%t", operation, queue, success)
}
