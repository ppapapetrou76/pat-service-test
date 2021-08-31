package main

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	_ "google.golang.org/grpc/encoding/gzip"
	"gorm.io/gorm"

	"github.com/sliide/logstash"
	healthcheck "github.com/sliide/service-healthcheck"
	"github.com/sliide/shared-go-libs/metric/prometheus"
	"github.com/sliide/template-grpc-service/internal/configs"
	"github.com/sliide/template-grpc-service/internal/grpcd"
)

const (
	checkingInterval = 100 * time.Millisecond
)

// Those variables indicates the build info, should be assign in the build stage.
var (
	Version     string
	GitRevision string
	GitBranch   string
)

type resources struct {
	// this db is used to access datastore(s) used by this service only.
	db *gorm.DB
}

func main() {
	sys, err := configs.Load()
	if err != nil {
		logrus.WithError(err).Fatalf("Failed to load system config")
	}

	if err := initLogstash(sys); err != nil {
		logrus.WithError(err).Fatalf("Failed to initialise logstash")
	}

	res, err := initResources(sys)
	if err != nil {
		logrus.WithError(err).Fatalf("Failed to initialise resources")
	}

	s, err := initServer(sys, res)
	if err != nil {
		logrus.WithError(err).Fatalf("Failed to initialise server")
	}

	if err := initMonitoring(sys, s, res); err != nil {
		logrus.WithError(err).Fatalf("Failed to initialise monitoring")
	}

	// Run the server
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			logrus.WithError(err).Fatalf("Failed to listen and serve the server")
		}
	}()

	// Wait terminal signal
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	logrus.Info("Stopping server")

	s.GracefulStop()
}

func initLogstash(sys configs.Config) error {
	return logstash.InitWithOutput(
		sys.LogLevel,
		sys.Env,
		sys.Service,
		os.Stdout,
	)
}

func initResources(sys configs.Config) (*resources, error) {
	return &resources{}, nil
}

func initServer(sys configs.Config, res *resources) (*grpcd.Server, error) {
	l := logrus.NewEntry(logrus.StandardLogger())
	listenAddr := sys.ListenAddr
	params := grpcd.ServerConfigParams{
		Name:       sys.Service,
		ListenAddr: listenAddr,
		DB:         res.db,
	}
	cfg := grpcd.NewServerConfigs(params,
		grpcd.SetLogger(l.WithField("service_version", fmt.Sprintf("%s (%s)", Version, runtime.Version()))))

	logrus.WithFields(logrus.Fields{
		"listen_addr":  listenAddr,
		"version":      Version,
		"go_version":   runtime.Version(),
		"git_revision": GitRevision,
		"git_branch":   GitBranch,
	}).Info("Starting server")

	return grpcd.NewServer(cfg)
}

func initMonitoring(sys configs.Config, s *grpcd.Server, res *resources) error {
	// We don't need to check the monitoring endpoints are working or not,
	// the external monitoring tools (e.g. Sensu) will raise warnings
	// if cannot access these endpoints.

	// Readiness check
	// We only check the service starts serving or not,
	// do not need to care the service is 100% healthy,
	// or all dependencies are working fine.
	isReady := &atomic.Value{}
	isReady.Store(false)

	go func() {
		for {
			if s.Serving() {
				break
			}
			time.Sleep(checkingInterval)
		}
		time.Sleep(1 * time.Second)
		isReady.Store(true)
	}()

	if sys.PprofEnabled {
		go func() {
			// Export to a different port from monitoring, because profiling should have high-level security settings
			h := mux.NewRouter()

			h.Handle("/", http.RedirectHandler("/debug/pprof/", http.StatusTemporaryRedirect))
			h.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
			h.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
			h.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
			h.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
			h.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
			logrus.Println(http.ListenAndServe(":6060", h))
		}()
	}

	prometheus.MustInit(prometheus.InitArguments{
		Service:     sys.Service,
		HostName:    sys.Hostname,
		Environment: sys.Env,
		Version:     Version,
		GoVersion:   runtime.Version(),
		GitRevision: GitRevision,
		GitBranch:   GitBranch,
	})

	logrus.AddHook(prometheus.NewLogsMetrics().Hook())

	go func() {
		h := mux.NewRouter()
		// Readiness endpoint for k8s
		h.Handle("/ready", healthcheck.Readiness(isReady))

		// Health check endpoint
		h.Handle("/", http.RedirectHandler("/healthcheck", http.StatusTemporaryRedirect))
		h.Handle("/healthcheck", func() http.Handler {
			hc := healthcheck.New(healthcheck.Params{
				Service:     sys.Service,
				Environment: sys.Env,
				Version:     Version,
			})

			hc.AddCheck("http server", healthcheck.DaemonServingCheck(s))

			return healthcheck.HandlerWithLogger(hc, logrus.NewEntry(logrus.StandardLogger()))
		}())

		// Prometheus metrics endpoint
		h.Handle("/metrics", prometheus.Handler())
		logrus.Println("Monitoring Healthcheck", http.ListenAndServe(":2112", h))
	}()

	return nil
}
