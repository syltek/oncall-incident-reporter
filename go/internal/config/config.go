// Package config provides configuration management for the application
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

const (
	defaultConfigFile = "config.yaml"
	slackTokenEnv    = "SLACK_TOKEN"
	slackChannelEnv  = "SLACK_CHANNEL_ID"
	configFileEnv    = "CONFIG_FILE"
)

// Config holds the complete application configuration
type Config struct {
	SlackConfig   *SlackConfig   `mapstructure:"slack_config"`
	Endpoints     Endpoints      `mapstructure:"endpoints"`
	Modal         Modal          `mapstructure:"modal"`
	MonitorConfig *MonitorConfig `mapstructure:"monitor_config"`
}

// MonitorConfig holds monitor-specific configuration
type MonitorConfig struct {
	ID                   int64   `mapstructure:"id"`
	NewCriticalThreshold float64 `mapstructure:"new_critical_threshold"`
	AlertRecipient       string  `mapstructure:"alert_recipient"`
}

// SlackConfig holds Slack-specific configuration
type SlackConfig struct {
	Token         string `mapstructure:"token"`
	ChannelID     string `mapstructure:"channel_id"`
	MessageFormat string `mapstructure:"message_format"`
}

// Endpoints holds API endpoint configurations
type Endpoints struct {
	SlackCommand     string `mapstructure:"slack_command"`
	SlackModalParser string `mapstructure:"slack_modal_parser"`
}

// Modal represents the modal dialog configuration
type Modal struct {
	Title  string  `mapstructure:"title"`
	Inputs []Input `mapstructure:"inputs"`
}

type Option struct {
	Text string `mapstructure:"text"`
}

// Input represents a single input field configuration in the modal
type Input struct {
	Key         string `mapstructure:"key"`
	Label       string `mapstructure:"label"`
	Placeholder string `mapstructure:"placeholder"`
	Required    bool   `mapstructure:"required"`
	Type        string `mapstructure:"type"`
	Options     []Option `mapstructure:"options"`
}
// LoadConfig reads and parses the configuration from files and environment variables
func LoadConfig() (*Config, error) {
    v := viper.New() // Create a new Viper instance for better isolation
    
    if err := setupViper(v); err != nil {
        return nil, fmt.Errorf("setup viper: %w", err)
    }
    
    if err := bindEnvironmentVariables(v); err != nil {
        return nil, fmt.Errorf("bind environment variables: %w", err)
    }
    
    if err := readConfiguration(v); err != nil {
        return nil, fmt.Errorf("read configuration: %w", err)
    }
    
    config, err := unmarshalConfig(v)
    if err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }
    
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

    return config, nil
}

// setupViper configures the Viper instance with initial settings
func setupViper(v *viper.Viper) error {
    configFile := v.GetString(configFileEnv)
    if configFile == "" {
        configFile = defaultConfigFile
    }
    v.SetConfigFile(configFile)
    return nil
}

// bindEnvironmentVariables binds environment variables to configuration keys
func bindEnvironmentVariables(v *viper.Viper) error {
    if err := v.BindEnv("slack_config.token", slackTokenEnv); err != nil {
        return fmt.Errorf("bind slack token: %w", err)
    }
    
    if err := v.BindEnv("slack_config.channel_id", slackChannelEnv); err != nil {
        return fmt.Errorf("bind slack channel ID: %w", err)
    }
    
    v.AutomaticEnv()
    return nil
}

// readConfiguration attempts to read the configuration file
func readConfiguration(v *viper.Viper) error {
    if err := v.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return fmt.Errorf("read config file: %w", err)
        }
        // Config file not found - using environment variables only
        return nil
    }
    return nil
}

// unmarshalConfig unmarshals the configuration into a Config struct
func unmarshalConfig(v *viper.Viper) (*Config, error) {
    var config Config
    if err := v.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }
    return &config, nil
}

// validate validates the configuration
func (c *Config) validate() error {
    if c.SlackConfig != nil {
        if c.SlackConfig.Token == "" {
            return fmt.Errorf("slack token is required when slack_config is present")
        }
        // ChannelID and MessageFormat can be empty
    }

    if c.MonitorConfig != nil {
        if c.MonitorConfig.ID == 0 {
            return fmt.Errorf("monitor ID is required when monitor_config is present")
        }
        if c.MonitorConfig.NewCriticalThreshold == 0 {
            return fmt.Errorf("new critical threshold is required when monitor_config is present")
        }
        if c.MonitorConfig.AlertRecipient == "" {
            return fmt.Errorf("alert recipient is required when monitor_config is present")
        }
    }

    return nil
}
