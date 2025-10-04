package metrics

import (
	"log"

	"github.com/VictoriaMetrics/metrics"
)

// RecordListing records listing creation and management with USD amount tracking
func RecordListing(operation, listingType, amountUSD, languageCode string) {
	if !IsEnabled() {
		return
	}

	// Note: Amount is currently selected via buttons (predefined USD values)
	// Future enhancement: Allow keyboard input for custom amounts
	metricName := `librecash_listings_total{operation="` + operation + `",listing_type="` + listingType + `",amount_usd="` + amountUSD + `",language_code="` + languageCode + `"}`
	counter := metrics.GetOrCreateCounter(metricName)
	counter.Inc()
	log.Printf("[METRICS] Listing: operation=%s, type=%s, amount_usd=%s, language=%s", operation, listingType, amountUSD, languageCode)
}
