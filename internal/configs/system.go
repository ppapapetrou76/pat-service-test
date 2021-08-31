package configs

import (
	"fmt"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	// `HOSTNAME` is provided by OS and should always be non-empty
	Hostname     string `env:"HOSTNAME"`
	Env          string `env:"ENV" envDefault:"dev"`
	Service      string `env:"SERVICE" envDefault:"template-grpc-service"`
	LogLevel     string `env:"LOG_LEVEL" envDefault:"INFO"`
	ListenAddr   string `env:"SERVER_LISTEN_ADDR" envDefault:"0.0.0.0:8080"`
	PprofEnabled bool   `env:"PPROF_ENABLED" envDefault:"true"`
	RdsURL       string `env:"RDS_URL,required"`
}

func Load() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}
