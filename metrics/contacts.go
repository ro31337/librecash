package metrics

import (
	"log"

	"github.com/VictoriaMetrics/metrics"
)

// RecordContactRequest records contact information requests
func RecordContactRequest(listingType, languageCode, userType string) {
	if !IsEnabled() {
		return
	}

	// VictoriaMetrics/metrics API: include labels in metric name
	metricName := `librecash_contacts_requested_total{listing_type="` + listingType + `",language_code="` + languageCode + `",user_type="` + userType + `"}`
	counter := metrics.GetOrCreateCounter(metricName)
	counter.Inc()
	log.Printf("[METRICS] Contact request: listing_type=%s, language=%s, user_type=%s", listingType, languageCode, userType)
}
