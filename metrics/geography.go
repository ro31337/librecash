package metrics

import (
	"fmt"
	"log"

	"github.com/VictoriaMetrics/metrics"
)

// RecordUserLocation records user location data collection with exact coordinates
func RecordUserLocation(lat, lon float64, languageCode string) {
	if !IsEnabled() {
		return
	}

	// Submit exact coordinates as provided by user
	latStr := fmt.Sprintf("%.6f", lat)
	lonStr := fmt.Sprintf("%.6f", lon)

	metricName := `librecash_user_locations_total{latitude="` + latStr + `",longitude="` + lonStr + `",language_code="` + languageCode + `"}`
	counter := metrics.GetOrCreateCounter(metricName)
	counter.Inc()
	log.Printf("[METRICS] User location: lat=%s, lon=%s, language=%s", latStr, lonStr, languageCode)
}
