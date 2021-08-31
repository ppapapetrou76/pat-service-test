package prometheus

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// WatchDBStats starts an go-routine that keep monitoring the db.Stats() and put into the prometheus metrics
// db.Stats() is the internal database statistics on this instance, including connection-pool status and counters.
func WatchDBStats(db *sql.DB, name string, refreshInterval time.Duration) (cancel context.CancelFunc, err error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	if refreshInterval <= 0 {
		return nil, errors.New("refresh interval is less or equal to zero")
	}

	tickFunc := refreshDBStatsMetricFunc(db, name)

	return startTick(tickFunc, refreshInterval)
}

func startTick(tickFunc func(), interval time.Duration) (cancel context.CancelFunc, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		timer := time.NewTicker(interval)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				tickFunc()
			}
		}
	}()

	return cancel, nil
}

func refreshDBStatsMetricFunc(db *sql.DB, name string) func() {
	labels := prometheus.Labels{ // Cache labels to avoid memory allocation
		"database": name,
	}

	return func() {
		stats := db.Stats()
		dbStatsMaxOpenConnections.With(labels).Set((float64)(stats.MaxOpenConnections))
		dbStatsOpenConnections.With(labels).Set((float64)(stats.OpenConnections))
		dbStatsInUse.With(labels).Set((float64)(stats.InUse))
		dbStatsIdle.With(labels).Set((float64)(stats.Idle))
		dbStatsWaitCount.With(labels).Set((float64)(stats.WaitCount))
		dbStatsWaitDurationInSeconds.With(labels).Set(stats.WaitDuration.Seconds())
		dbStatsMaxIdleClosed.With(labels).Set((float64)(stats.MaxIdleClosed))
		dbStatsMaxLifetimeClosed.With(labels).Set((float64)(stats.MaxLifetimeClosed))
	}
}

var (
	dbStatsMaxOpenConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_stats_max_open_connections",
			Help: "Maximum number of open connections to the database.",
		},
		[]string{"database"},
	)

	dbStatsOpenConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_stats_open_connections",
			Help: "The number of established connections both in use and idle.",
		},
		[]string{"database"},
	)

	dbStatsInUse = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_stats_in_use",
			Help: "The number of connections currently in use.",
		},
		[]string{"database"},
	)

	dbStatsIdle = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_stats_idle",
			Help: "The number of idle connections.",
		},
		[]string{"database"},
	)

	dbStatsWaitCount = promauto.NewGaugeVec(
		// nolint:promlinter
		prometheus.GaugeOpts{
			Name: "database_stats_wait_count",
			Help: "The total number of connections waited for.",
		},
		[]string{"database"},
	)

	dbStatsWaitDurationInSeconds = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_stats_wait_duration_in_seconds",
			Help: "The total time in seconds blocked waiting for a new connection.",
		},
		[]string{"database"},
	)

	dbStatsMaxIdleClosed = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_stats_max_idle_closed",
			Help: "The total number of connections closed due to SetMaxIdleConns.",
		},
		[]string{"database"},
	)

	dbStatsMaxLifetimeClosed = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_stats_max_lifetime_closed",
			Help: "The total number of connections closed due to SetConnMaxLifetime.",
		},
		[]string{"database"},
	)
)
