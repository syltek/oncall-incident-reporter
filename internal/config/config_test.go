package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name       string
		configFile string
		envVars    map[string]string
		wantErr    bool
		validate   func(*testing.T, *Config)
	}{
		{
			name:       "valid config",
			configFile: "testdata/valid_config.yaml",
			envVars: map[string]string{
				"SLACK_TOKEN":          "xoxb-test-token",
				"SLACK_SIGNING_SECRET": "test-signing-secret",
			},
			wantErr: false,
			validate: func(t *testing.T, c *Config) {
				assert.True(t, c.IsValid())
				assert.Equal(t, "xoxb-test-token", c.GetSlackToken())
				assert.Equal(t, "development", c.GetEnvironment())
			},
		},
		{
			name:       "missing required fields",
			configFile: "testdata/invalid_config.yaml",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set config file
			t.Setenv("CONFIG_FILE", tt.configFile)

			v := viper.New()

			// Set test environment variables
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			// Override config file
			v.SetConfigFile(tt.configFile)

			cfg, err := LoadConfigWithViper(v)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}
