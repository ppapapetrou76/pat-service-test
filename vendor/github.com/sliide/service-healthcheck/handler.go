package healthcheck

import (
	"encoding/json"
	"net/http"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// Readiness returns a handler that returns the readiness state base on the atomic value,
// the handler only returns http.StatusOK if the isReady value stored the true boolean value
func Readiness(isReady *atomic.Value) http.HandlerFunc {
	if isReady == nil {
		return func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		}
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		if b, ok := isReady.Load().(bool); !ok || !b {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// Handler returns a handler that handle the healthcheck
func Handler(hc HealthChecker) http.Handler {
	if hc == nil {
		return handlerServerError(`{"error": "Failed to find the health checker"}`)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		results := hc.RunChecks(req.Context())
		writeResults(w, results)
	})
}

// HandlerWithLogger returns a handler that handle the health check and logs the health check results
func HandlerWithLogger(hc HealthChecker, logger *logrus.Entry) http.Handler {
	if hc == nil {
		return handlerServerError(`{"error": "Failed to find the health checker"}`)
	}

	if logger == nil {
		return handlerServerError(`{"error": "Logger not provided"}`)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		results := hc.RunChecks(req.Context())
		writeResults(w, results)
		logResults(logger, results)
	})
}

func writeResults(w http.ResponseWriter, results CheckingResults) {
	// Add extra fields for the human readable
	output := struct {
		CheckingResults

		DurationInSeconds float64 `json:"duration_in_seconds"`
		IsHealthy         bool    `json:"is_healthy"`
		IsDegraded        bool    `json:"is_degraded"`
	}{
		results,

		results.Duration.Seconds(),
		results.IsHealthy(),
		results.IsDegraded(),
	}

	b, err := json.Marshal(output)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error": "Failed to marshal json"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func handlerServerError(errorMsg string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(errorMsg))
	})
}

func logResults(l *logrus.Entry, results CheckingResults) {
	l.WithFields(logrus.Fields{
		"state":                  results.GetState().String(),
		"total_duration_seconds": results.Duration.Seconds(),
		"checks":                 results.Checks,
	}).Info("Healthcheck")
}
