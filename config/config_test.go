package config

import (
	"os"
	"path"
	"reflect"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

type envVariables struct {
	agentData  string
	agentBatch string
	telemBatch string
	timeout    string
	region     string
	logLevel   string
	acctId     string
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name string
		want Config
		vars envVariables
	}{
		{
			name: "default",
			want: defaultConfig(),
			vars: envVariables{
				agentData:  "",
				agentBatch: "",
				telemBatch: "",
				timeout:    "",
				region:     "",
				logLevel:   "",
				acctId:     "",
			},
		},
		{
			name: "default env vars",
			want: defaultConfig(),
			vars: envVariables{
				agentData:  "true",
				agentBatch: "1",
				telemBatch: "1",
				timeout:    "10s",
				region:     "",
				logLevel:   "info",
				acctId:     "",
			},
		},
		{
			name: "override all defaults",
			want: func() Config {
				return Config{
					CollectAgentData:        false,
					DataCollectionTimeout:   600 * time.Millisecond,
					AgentTelemetryBatchSize: 5,
					TelemetryAPIBatchSize:   8,
					LogLevel:                log.WarnLevel,
					AgentTelemetryRegion:    "test",
					AccountID:               "12",
					ExtensionName:           path.Base(os.Args[0]),
				}
			}(),
			vars: envVariables{
				agentData:  "false",
				agentBatch: "5",
				telemBatch: "8",
				timeout:    "600ms",
				region:     "test",
				logLevel:   "warn",
				acctId:     "12",
			},
		},
		{
			name: "timeout too low",
			want: func() Config {
				return Config{
					CollectAgentData:        false,
					DataCollectionTimeout:   600 * time.Millisecond,
					AgentTelemetryBatchSize: 5,
					TelemetryAPIBatchSize:   8,
					LogLevel:                log.WarnLevel,
					AgentTelemetryRegion:    "test",
					AccountID:               "12",
					ExtensionName:           path.Base(os.Args[0]),
				}
			}(),
			vars: envVariables{
				agentData:  "false",
				agentBatch: "5",
				telemBatch: "8",
				timeout:    "300ms",
				region:     "test",
				logLevel:   "warn",
				acctId:     "12",
			},
		},
		{
			name: "invalid agent telemetry batch size",
			want: func() Config {
				return Config{
					CollectAgentData:        false,
					DataCollectionTimeout:   600 * time.Millisecond,
					AgentTelemetryBatchSize: defaultAgentTelemtryBatchSize,
					TelemetryAPIBatchSize:   8,
					LogLevel:                log.WarnLevel,
					AgentTelemetryRegion:    "test",
					AccountID:               "12",
					ExtensionName:           path.Base(os.Args[0]),
				}
			}(),
			vars: envVariables{
				agentData:  "false",
				agentBatch: "invalid",
				telemBatch: "8",
				timeout:    "300ms",
				region:     "test",
				logLevel:   "warn",
				acctId:     "12",
			},
		},
		{
			name: "invalid telemetry api batch size",
			want: func() Config {
				return Config{
					CollectAgentData:        false,
					DataCollectionTimeout:   600 * time.Millisecond,
					AgentTelemetryBatchSize: 5,
					TelemetryAPIBatchSize:   defaultTelemtryAPIBatchSize,
					LogLevel:                log.WarnLevel,
					AgentTelemetryRegion:    "test",
					AccountID:               "12",
					ExtensionName:           path.Base(os.Args[0]),
				}
			}(),
			vars: envVariables{
				agentData:  "false",
				agentBatch: "5",
				telemBatch: "invalid",
				timeout:    "300ms",
				region:     "test",
				logLevel:   "warn",
				acctId:     "12",
			},
		},
	}
	for _, tt := range tests {
		os.Setenv(agentDataEnabledVariable, tt.vars.agentData)
		os.Setenv(agentDataBatchSizeVariable, tt.vars.agentBatch)
		os.Setenv(clientRetryTimeoutVariable, tt.vars.timeout)
		os.Setenv(agentTelemetryRegionVariable, tt.vars.region)
		os.Setenv(extensionLogLevelVariable, tt.vars.logLevel)
		os.Setenv(telAPIBatchSizeVariable, tt.vars.telemBatch)
		os.Setenv(nrAccountIDVariable, tt.vars.acctId)

		t.Run(tt.name, func(t *testing.T) {
			if got := GetConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetConfig() = %v, want %v", got, tt.want)
			}
		})

		os.Unsetenv(agentDataEnabledVariable)
		os.Unsetenv(agentDataBatchSizeVariable)
		os.Unsetenv(clientRetryTimeoutVariable)
		os.Unsetenv(agentTelemetryRegionVariable)
		os.Unsetenv(extensionLogLevelVariable)
		os.Unsetenv(telAPIBatchSizeVariable)
		os.Unsetenv(nrAccountIDVariable)
	}
}
