package grpcd

import (
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	// defaultTimeoutRPC specifies a time limit for processing a gRPC call.
	defaultTimeoutRPC = time.Second * 5
	// defaultMaxConnectionAge is a duration for the maximum amount of time a connection may exist before it will be closed by sending a GoAway.
	defaultMaxConnectionAge = time.Second * 60
	// defaultMaxConnectionAgeGrace allows pending RPCs to complete before forcibly closing connections.
	defaultMaxConnectionAgeGrace = time.Second * 10
)

// ServerConfigs defines the initial configs for the content Server.
type ServerConfigs struct {
	name       string
	listenAddr string

	logger *logrus.Entry

	maxConnectionAge      time.Duration
	maxConnectionAgeGrace time.Duration

	db *gorm.DB
}

// ServerConfigParams represents params for creating a ServerConfigs object.
type ServerConfigParams struct {
	Name       string
	ListenAddr string
	DB         *gorm.DB
}

// ServerConfigsOpts defines a function that can change properties of a ServerConfigs concrete object.
type ServerConfigsOpts func(cfg *ServerConfigs)

// SetLogger sets the logger attribute of a ServerConfigs.
func SetLogger(logger *logrus.Entry) ServerConfigsOpts {
	return func(cfg *ServerConfigs) {
		cfg.logger = logger
	}
}

// SetMaxConnectionAge sets the maxConnectionAge attribute of a ServerConfigs.
func SetMaxConnectionAge(value time.Duration) ServerConfigsOpts {
	return func(cfg *ServerConfigs) {
		cfg.maxConnectionAge = value
	}
}

// SetMaxConnectionAgeGrace sets the maxConnectionAgeGrace attribute of a ServerConfigs.
func SetMaxConnectionAgeGrace(value time.Duration) ServerConfigsOpts {
	return func(cfg *ServerConfigs) {
		cfg.maxConnectionAgeGrace = value
	}
}

// NewServerConfigs returns a new ServerConfigs object initialized with ServerConfigParams, and the default
// values for other attributes.
// Clients can also provide optional parameters to override one or more default values.
func NewServerConfigs(params ServerConfigParams, opts ...ServerConfigsOpts) ServerConfigs {
	srvConfig := ServerConfigs{
		name:                  params.Name,
		listenAddr:            params.ListenAddr,
		db:                    params.DB,
		logger:                logrus.NewEntry(logrus.StandardLogger()),
		maxConnectionAge:      defaultMaxConnectionAge,
		maxConnectionAgeGrace: defaultMaxConnectionAgeGrace,
	}

	for _, o := range opts {
		o(&srvConfig)
	}

	return srvConfig
}
