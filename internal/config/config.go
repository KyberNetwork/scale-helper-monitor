package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"scale-helper-monitor/internal/clients/kyberswap"
	"scale-helper-monitor/internal/clients/slack"
	"scale-helper-monitor/internal/clients/tenderly"
	"scale-helper-monitor/internal/monitor"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Slack      SlackConfig              `mapstructure:"slack"`
	Tenderly   TenderlyConfig           `mapstructure:"tenderly"`
	Monitoring monitor.Config           `mapstructure:"monitoring"`
	KyberSwap  kyberswap.Config         `mapstructure:"kyberswap"`
	Chains     []monitor.ChainConfig    `mapstructure:"chains"`
	TestTokens []monitor.TestToken      `mapstructure:"test_tokens"`
	BalanceSlot map[string]map[string]map[string]string        `mapstructure:"balance_slot"`
}

// SlackConfig represents Slack configuration
type SlackConfig struct {
	WebhookURL string `mapstructure:"webhook_url"`
}

// TenderlyConfig represents Tenderly configuration
type TenderlyConfig struct {
	AccessKey string `mapstructure:"access_key"`
	Username  string `mapstructure:"username"`
	Project   string `mapstructure:"project"`
}

// GetSlackClient creates a Slack client from the configuration
func (c *Config) GetSlackClient(timeout time.Duration, logger *logrus.Logger) *slack.Client {
	return slack.NewClient(c.Slack.WebhookURL, timeout, logger)
}

// GetKyberSwapClient creates a KyberSwap client from the configuration
func (c *Config) GetKyberSwapClient(timeout time.Duration, logger *logrus.Logger) *kyberswap.Client {
	return kyberswap.NewClient(c.KyberSwap, timeout, logger)
}

// GetTenderlyClient creates a Tenderly client from the configuration
func (c *Config) GetTenderlyClient() *tenderly.Client {
	return tenderly.NewClient(c.Tenderly.AccessKey, c.Tenderly.Username, c.Tenderly.Project)
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("$HOME/.scale-helper-monitor")
	viper.AddConfigPath("/etc/scale-helper-monitor/")

	// Enable environment variable substitution
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Set environment variables for viper to use
	setEnvironmentVariables()

	// Create config struct and manually populate with environment variables
	var config Config
	
	// Slack config
	config.Slack.WebhookURL = os.Getenv("SLACK_WEBHOOK_URL")
	
	// Tenderly config
	config.Tenderly.AccessKey = os.Getenv("TENDERLY_ACCESS_KEY")
	config.Tenderly.Username = os.Getenv("TENDERLY_USERNAME")
	config.Tenderly.Project = os.Getenv("TENDERLY_PROJECT")
	
	// Monitoring config
	config.Monitoring.Interval = viper.GetString("monitoring.interval")
	config.Monitoring.Timeout = viper.GetString("monitoring.timeout")
	
	// KyberSwap config
	config.KyberSwap.APIBaseURL = viper.GetString("kyberswap.api_base_url")
	config.KyberSwap.ClientID = viper.GetString("kyberswap.client_id")
	
	// Chains config
	config.Chains = []monitor.ChainConfig{
		{
			Name:            "ethereum",
			ChainID:         1,
			RPCURL:          os.Getenv("ETH_NODE_URL"),
			ContractAddress: os.Getenv("ETH_CONTRACT_ADDRESS"),
		},
		{
			Name:            "polygon", 
			ChainID:         137,
			RPCURL:          os.Getenv("POLYGON_NODE_URL"),
			ContractAddress: os.Getenv("POLYGON_CONTRACT_ADDRESS"),
		},
		{
			Name:            "bsc",
			ChainID:         56,
			RPCURL:          os.Getenv("BSC_NODE_URL"),
			ContractAddress: os.Getenv("BSC_CONTRACT_ADDRESS"),
		},
		{
			Name:            "arbitrum",
			ChainID:         42161,
			RPCURL:          os.Getenv("ARBITRUM_NODE_URL"),
			ContractAddress: os.Getenv("ARBITRUM_CONTRACT_ADDRESS"),
		},
	}

	// Load balance slots from JSON file
	balanceSlots, err := loadBalanceSlots()
	if err != nil {
		return nil, fmt.Errorf("failed to load balance slots: %w", err)
	}
	config.BalanceSlot = balanceSlots
	
	// Test tokens config
	if err := viper.UnmarshalKey("test_tokens", &config.TestTokens); err != nil {
		return nil, fmt.Errorf("failed to unmarshal test_tokens: %w", err)
	}

	return &config, nil
}

func loadBalanceSlots() (map[string]map[string]map[string]string, error) {
	// Try to read from multiple possible locations
	path := "./balance_slots.json"
	data, err := os.ReadFile(path)
	
	if err != nil {
		return nil, fmt.Errorf("failed to read balance_slots.json from any location: %w", err)
	}
	
	var balanceSlots map[string]map[string]map[string]string
	if err := json.Unmarshal(data, &balanceSlots); err != nil {
		return nil, fmt.Errorf("failed to unmarshal balance slots JSON: %w", err)
	}
	// fmt.Println(balanceSlots)
	
	return balanceSlots, nil
}

func setEnvironmentVariables() {
	envVars := []string{
		"ETH_NODE_URL", "POLYGON_NODE_URL", "BSC_NODE_URL", "ARBITRUM_NODE_URL",
		"ETH_CONTRACT_ADDRESS", "POLYGON_CONTRACT_ADDRESS", "BSC_CONTRACT_ADDRESS", "ARBITRUM_CONTRACT_ADDRESS",
		"SLACK_WEBHOOK_URL",
		"TENDERLY_ACCESS_KEY", "TENDERLY_USERNAME", "TENDERLY_PROJECT",
	}

	for _, env := range envVars {
		if value := os.Getenv(env); value != "" {
			// Convert environment variable name to viper key format
			key := strings.ToLower(strings.ReplaceAll(env, "_", "."))
			viper.Set(key, value)
		}
	}
} 