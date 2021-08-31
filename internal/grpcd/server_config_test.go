package grpcd

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestNewServerConfigs(t *testing.T) {
	type args struct {
		params ServerConfigParams
		opts   []ServerConfigsOpts
	}

	db := new(gorm.DB)
	tests := []struct {
		name     string
		args     args
		expected ServerConfigs
	}{
		{
			name: "should return configuration without opts",
			args: args{
				params: ServerConfigParams{
					Name:       "some-service-Name",
					ListenAddr: "localhost:8080",
					DB:         db,
				},
			},
			expected: ServerConfigs{
				name:                  "some-service-Name",
				listenAddr:            "localhost:8080",
				logger:                logrus.NewEntry(logrus.StandardLogger()),
				maxConnectionAge:      time.Second * 60,
				maxConnectionAgeGrace: time.Second * 10,
				db:                    db,
			},
		},
		{
			name: "should return configuration with all available opts",
			args: args{
				params: ServerConfigParams{
					Name:       "some-service-Name",
					ListenAddr: "localhost:8080",
					DB:         db,
				},
				opts: []ServerConfigsOpts{
					SetLogger(logrus.NewEntry(logrus.StandardLogger())),
					SetMaxConnectionAge(time.Second * 2),
					SetMaxConnectionAgeGrace(time.Hour * 10),
				},
			},
			expected: ServerConfigs{
				name:                  "some-service-Name",
				listenAddr:            "localhost:8080",
				logger:                logrus.NewEntry(logrus.StandardLogger()),
				maxConnectionAge:      time.Second * 2,
				maxConnectionAgeGrace: time.Hour * 10,
				db:                    db,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := NewServerConfigs(tt.args.params, tt.args.opts...)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
