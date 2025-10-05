package metrics

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/VictoriaMetrics/metrics"
)

// Configuration for metrics collection
type Config struct {
	Enabled bool
	Port    int
	Path    string
}

// Global configuration
var config Config

// Initialize metrics system
func Init() error {
	// Load configuration from environment variables
	config = Config{
		Enabled: getEnvBool("METRICS_ENABLED", true),
		Port:    getEnvInt("METRICS_PORT", 8081),
		Path:    getEnvString("METRICS_PATH", "/metrics"),
	}

	if !config.Enabled {
		log.Printf("[METRICS] Metrics collection is disabled")
		return nil
	}

	log.Printf("[METRICS] Initializing metrics system on port %d", config.Port)

	// Start HTTP server for metrics endpoint
	go startMetricsServer()

	log.Printf("[METRICS] Metrics system initialized successfully")
	return nil
}

// Start HTTP server for metrics endpoint
func startMetricsServer() {
	mux := http.NewServeMux()

	// Metrics endpoint
	mux.HandleFunc(config.Path, func(w http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(w, true)
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := fmt.Sprintf("127.0.0.1:%d", config.Port)
	log.Printf("[METRICS] Starting metrics server on %s%s", addr, config.Path)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("[METRICS] Error starting metrics server: %v", err)
	}
}

// IsEnabled returns true if metrics collection is enabled
func IsEnabled() bool {
	return config.Enabled
}

// Helper functions for environment variables
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvString(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
