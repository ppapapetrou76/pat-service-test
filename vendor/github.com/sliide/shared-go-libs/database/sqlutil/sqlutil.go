package sqlutil

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"gorm.io/gorm"

	healthcheck "github.com/sliide/service-healthcheck"
	"github.com/sliide/shared-go-libs/metric/prometheus"
)

// Watch db stats every n seconds.
const dbRefreshInterval = time.Second * 5

// MonitoringParams is used to pass parameters to the InitDBMonitoring function
type MonitoringParams struct {
	// This will be used in metrics injection. It doesn't have to be the actual DB name
	DBName string
	// This tells prometheus how frequently to watch for db stats.
	DBRefreshInterval time.Duration
	// This is a list of models we would like to check that the DB has full permissions.
	Models []interface{}
}

func (params MonitoringParams) validate() error {
	if params.DBName == "" {
		return errors.New("db name is required")
	}

	return nil
}

func (params *MonitoringParams) defaultValues() {
	if params.DBRefreshInterval.Nanoseconds() <= 0 {
		params.DBRefreshInterval = dbRefreshInterval
	}
}

// InitDBMonitoring accepts a gorm.DB reference and some monitoring parameters and returns a health check / monitoring
// function healthcheck.CheckingFunc that can be used by our services.
// If an error occurs then it is returned with a proper wrapped message.
func InitDBMonitoring(gormDB *gorm.DB, params *MonitoringParams) (healthcheck.CheckingFunc, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.defaultValues()
	db, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get DB connection: %w", err)
	}

	tableNames, err := tableNames(gormDB, params.Models)
	if err != nil {
		return nil, fmt.Errorf("failed to get DB table names: %w", err)
	}

	_, err = prometheus.WatchDBStats(db, params.DBName, params.DBRefreshInterval)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Prometheus DB stats monitoring: %w", err)
	}

	return healthcheck.PostgresTableFullPermissionCheck(db, tableNames), nil
}

func tableNames(db *gorm.DB, models []interface{}) ([]string, error) {
	names := make([]string, 0, len(models))
	errs := new(multierror.Error)
	for _, model := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			errs = multierror.Append(err, errs)
		}
		names = append(names, stmt.Schema.Table)
	}

	return names, errs.ErrorOrNil()
}
