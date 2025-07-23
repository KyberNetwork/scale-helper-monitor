package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	Networks []Network `yaml:"networks"`
	Slack    struct {
		Token   string `yaml:"token"`
		Channel string `yaml:"channel"`
	} `yaml:"slack"`
	Monitoring struct {
		PollInterval int    `yaml:"poll_interval"`
		BatchSize    int    `yaml:"batch_size"`
		StateFile    string `yaml:"state_file"`
	} `yaml:"monitoring"`
	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logging"`
}

// LoadConfig loads configuration from YAML file and overrides with environment variables
func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Override with environment variables if set
	overrideWithEnv(&config)

	return &config, nil
}

// overrideWithEnv overrides config values with environment variables if they are set
func overrideWithEnv(config *Config) {
	// Slack configuration
	if token := os.Getenv("SLACK_TOKEN"); token != "" {
		config.Slack.Token = token
	}
	if channel := os.Getenv("SLACK_CHANNEL"); channel != "" {
		config.Slack.Channel = channel
	}

	// Monitoring configuration
	if pollInterval := os.Getenv("MONITORING_POLL_INTERVAL"); pollInterval != "" {
		if interval, err := strconv.Atoi(pollInterval); err == nil {
			config.Monitoring.PollInterval = interval
		}
	}

	// Logging configuration
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Logging.Format = format
	}
}

// GetNetworks returns the list of networks to monitor, loading from environment if needed
func (c *Config) GetNetworks() []Network {
	return LoadNetworksFromEnv(c.Networks, c.Monitoring.BatchSize)
}

// Validate performs basic validation on the configuration
func (c *Config) Validate() error {
	if c.Slack.Token == "" {
		return fmt.Errorf("slack token is required")
	}

	if c.Slack.Channel == "" {
		return fmt.Errorf("slack channel is required")
	}

	if c.Monitoring.PollInterval <= 0 {
		c.Monitoring.PollInterval = 30 // Default to 30 seconds
	}

	if c.Monitoring.BatchSize <= 0 {
		c.Monitoring.BatchSize = 100 // Default to 100 blocks
	}

	if c.Monitoring.StateFile == "" {
		c.Monitoring.StateFile = "config/distributor/State.json" // Default state file path
	}

	if c.Logging.Level == "" {
		c.Logging.Level = "info" // Default to info level
	}

	if c.Logging.Format == "" {
		c.Logging.Format = "text" // Default to text format
	}

	return nil
}
