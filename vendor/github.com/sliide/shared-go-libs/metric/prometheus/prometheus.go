package prometheus

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns the handler that handle metrics endpoint requests.
func Handler() http.Handler {
	return promhttp.Handler()
}

// InitArguments holds parameters that Init uses.
type InitArguments struct {
	Service     string
	HostName    string
	Environment string
	Version     string
	GoVersion   string
	GitRevision string
	GitBranch   string
}

// Init initializes the prometheus build-info setups.
func Init(a InitArguments) error {
	buildInfo := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sliide_build_info",
			Help: fmt.Sprintf("The %s's build information.", a.Service),
			ConstLabels: prometheus.Labels{
				"service_name": strings.ToLower(a.Service),
				"hostname":     strings.ToLower(a.HostName),
				"env":          strings.ToLower(a.Environment),
				"version":      strings.ToLower(a.Version),
				"go_version":   strings.ToLower(a.GoVersion),
				"revision":     strings.ToLower(a.GitRevision),
				"branch":       strings.ToLower(a.GitBranch),
			},
		},
	)
	if err := prometheus.Register(buildInfo); err != nil {
		return err
	}

	buildInfo.Set(1)

	return nil
}

// MustInit initializes the prometheus build-info setups and throw panic if got error.
func MustInit(a InitArguments) {
	if err := Init(a); err != nil {
		panic(err)
	}
}
