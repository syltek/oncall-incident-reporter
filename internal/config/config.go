// Package config provides configuration management for the application
package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

const (
	defaultConfigFile     = "config.yaml"
	slackTokenEnv         = "SLACK_TOKEN"
	slackSigningSecretEnv = "SLACK_SIGNING_SECRET"
	configFileEnv         = "CONFIG_FILE"
)

// Config holds the complete application configuration
type Config struct {
	Metadata      *Metadata       `mapstructure:"metadata"`
	SlackConfig   *SlackConfig   `mapstructure:"slack_config"`
	Endpoints     *Endpoints      `mapstructure:"endpoints"`
	Modal         *Modal          `mapstructure:"modal"`
}

type Metadata struct {
	Service       string          `mapstructure:"service"`
	Environment   string          `mapstructure:"environment"`
	Team          string          `mapstructure:"team"`
}

// SlackConfig holds Slack-specific configuration
type SlackConfig struct {
	Token         string `mapstructure:"slack_token"`
	SigningSecret string `mapstructure:"slack_signing_secret"`
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
	return LoadConfigWithViper(viper.New())
}

// LoadConfigWithViper configures the Viper instance with initial settings
func LoadConfigWithViper(v *viper.Viper) (*Config, error) {
	// Bind env variables before any other operations
	v.BindEnv("slack_config.slack_token", slackTokenEnv)
	v.BindEnv("slack_config.slack_signing_secret", slackSigningSecretEnv)

	if err := setupViper(v); err != nil {
		return nil, fmt.Errorf("setup viper: %w", err)
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
	// Read config file from environment variable
	configFile := os.Getenv(configFileEnv)
    
	if configFile == "" {
		configFile = defaultConfigFile
	}
	v.SetConfigFile(configFile)
	return nil
}

// readConfiguration attempts to read the configuration file
func readConfiguration(v *viper.Viper) error {
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("read config file: %w", err)
		}
		// Return error since config file is required
		return fmt.Errorf("config file not found: %w", err)
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
	if c.Metadata == nil {
		return fmt.Errorf("metadata is required")
	}

	if c.SlackConfig == nil {
		return fmt.Errorf("slack config is required")
	}

	if c.Endpoints == nil {
		return fmt.Errorf("endpoints are required")
	}

	if c.Modal == nil {
		return fmt.Errorf("modal is required")
	}

	if c.SlackConfig.SigningSecret == "" {
		return fmt.Errorf("slack signing secret is required, please set SLACK_SIGNING_SECRET environment variable")
	}

	if c.SlackConfig.Token == "" {
		return fmt.Errorf("slack token is required, please set SLACK_TOKEN environment variable")
	}

	return nil
}

// Add these helper methods to Config struct for easier testing
func (c *Config) IsValid() bool {
	return c.validate() == nil
}

// Add getters for required fields to make tests more readable
func (c *Config) GetSlackToken() string {
	return c.SlackConfig.Token
}

func (c *Config) GetEnvironment() string {
	if c.Metadata == nil {
		return ""
	}
	return c.Metadata.Environment
}
