package prometheus

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// ActivityStarting represents metrics label for starting service activity.
	ActivityStarting = "starting"
	// ActivityStopped represents metrics label for stopped service activity.
	ActivityStopped = "stopped"
)

var serviceActivities = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "service_activities_total",
		Help: "Metric informing about starting/stopped service activities",
	},
	[]string{"daemon", "activity"},
)

// ServiceMetrics represents metrics for starting/stopped activities. Due to its nature the counter expects to have
// values 0/1 for a given type of activity (starting/stopped).
type ServiceMetrics interface {
	ServiceStarting()
	ServiceStopped()
}

// NewServiceMetrics returns a metrics which is used for monitoring service activities (starting/stopped).
func NewServiceMetrics(daemon string) ServiceMetrics {
	return &serviceMetrics{
		daemon:     daemon,
		activities: serviceActivities,
	}
}

type serviceMetrics struct {
	daemon     string
	activities *prometheus.CounterVec
}

// ServiceStarting increase the starting counter.
func (m serviceMetrics) ServiceStarting() {
	labels := m.labels()
	labels["activity"] = ActivityStarting
	m.activities.With(labels).Inc()
}

// ServiceStopped increase the stopped counter.
func (m serviceMetrics) ServiceStopped() {
	labels := m.labels()
	labels["activity"] = ActivityStopped
	m.activities.With(labels).Inc()
}

func (m serviceMetrics) labels() prometheus.Labels {
	return prometheus.Labels{
		"daemon": strings.ToLower(m.daemon),
	}
}
