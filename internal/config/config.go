package config

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	Slack      SlackConfig                             `mapstructure:"slack"`
	Tenderly   TenderlyConfig                          `mapstructure:"tenderly"`
	Monitoring monitor.Config                          `mapstructure:"monitoring"`
	KyberSwap  kyberswap.Config                        `mapstructure:"kyberswap"`
	Chains     []monitor.ChainConfig                   `mapstructure:"chains"`
	TestCases  []monitor.TestCase                      `mapstructure:"test_cases"`
	Tokens     map[string]map[string]monitor.TokenInfo `mapstructure:"tokens"` // chain name -> token address -> token info
	Sources    map[string][]string                     `mapstructure:"liquidity_sources"` // chain name -> available sources
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

// KyberSwapDexResponse represents the API response structure
type KyberSwapDexResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Dexes []struct {
			ID       int    `json:"id"`
			DexID    string `json:"dexId"`
			IsEnabled bool  `json:"isEnabled"`
			Name     string `json:"name"`
			LogoURL  string `json:"logoURL"`
			Tags     interface{} `json:"tags"`
		} `json:"dexes"`
		Pagination struct {
			TotalItems int `json:"totalItems"`
		} `json:"pagination"`
	} `json:"data"`
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
func (c *Config) GetTenderlyClient(timeout time.Duration) *tenderly.Client {
	return tenderly.NewClient(c.Tenderly.AccessKey, c.Tenderly.Username, c.Tenderly.Project, timeout)
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

	// Chains config - adding all supported chains from tokens.json
	config.Chains = []monitor.ChainConfig{
		{
			Name:            "ethereum",
			ChainID:         1,
			RPCURL:          os.Getenv("ETH_NODE_URL"),
			ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		},
		{
			Name:            "polygon",
			ChainID:         137,
			RPCURL:          os.Getenv("POLYGON_NODE_URL"),
			ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		},
		{
			Name:            "bsc",
			ChainID:         56,
			RPCURL:          os.Getenv("BSC_NODE_URL"),
			ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		},
		{
			Name:            "arbitrum",
			ChainID:         42161,
			RPCURL:          os.Getenv("ARBITRUM_NODE_URL"),
			ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		},
		{
			Name:            "avalanche",
			ChainID:         43114,
			RPCURL:          os.Getenv("AVAX_NODE_URL"),
			ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		},
		{
			Name:            "base",
			ChainID:         8453,
			RPCURL:          os.Getenv("BASE_NODE_URL"),
			ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		},
		{
			Name:            "berachain",
			ChainID:         80094,
			RPCURL:          os.Getenv("BERA_NODE_URL"),
			ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		},
		{
			Name:            "mantle",
			ChainID:         5000,
			RPCURL:          os.Getenv("MANTLE_NODE_URL"),
			ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		},
		{
			Name:            "optimism",
			ChainID:         10,
			RPCURL:          os.Getenv("OPTIMISM_NODE_URL"),
			ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		},
		{
			Name:            "sonic",
			ChainID:         146,
			RPCURL:          os.Getenv("SONIC_NODE_URL"),
			ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		},
		{
			Name:            "unichain",
			ChainID:         1301,
			RPCURL:          os.Getenv("UNICHAIN_NODE_URL"),
			ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		},
	}

	// Fetch liquidity sources for each chain
	config.Sources = make(map[string][]string)
	// Parse timeout for clients
	timeout, err := time.ParseDuration(config.Monitoring.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timeout duration: %w", err)
	}
	for _, chain := range config.Chains {
		sources, err := fetchLiquiditySources(chain.Name, timeout)
		if err != nil {
			config.Sources[chain.Name] = []string{} // Set empty slice on error
		} else {
			config.Sources[chain.Name] = sources
		}
	}

	// Load tokens from JSON file
	tokens, err := loadTokens()
	if err != nil {
		return nil, fmt.Errorf("failed to load tokens: %w", err)
	}
	config.Tokens = tokens

	// Load test cases from the new nested format
	testCases, err := loadTestCases()
	if err != nil {
		return nil, fmt.Errorf("failed to load test cases: %w", err)
	}
	config.TestCases = testCases

	return &config, nil
}

func loadTokens() (map[string]map[string]monitor.TokenInfo, error) {
	// Try to read from multiple possible locations
	path := "./tokens.json"
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("failed to read tokens.json from any location: %w", err)
	}

	// First, unmarshal into the nested structure as it exists in the JSON file
	var tokens map[string]map[string]monitor.TokenInfo
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tokens JSON: %w", err)
	}

	return tokens, nil
}

func loadTestCases() ([]monitor.TestCase, error) {
	// Load the nested test cases structure
	var nestedTestCases map[string][]monitor.TestCase
	if err := viper.UnmarshalKey("test_cases", &nestedTestCases); err != nil {
		return nil, fmt.Errorf("failed to unmarshal test_cases: %w", err)
	}

	// Flatten the nested structure into a slice, adding chain_name to each test case
	var testCases []monitor.TestCase
	for chainName, chainTestCases := range nestedTestCases {
		for _, testCase := range chainTestCases {
			testCase.ChainName = chainName
			testCases = append(testCases, testCase)
		}
	}

	return testCases, nil
}

// fetchLiquiditySources fetches available liquidity sources for a given chain
func fetchLiquiditySources(chainName string, timeout time.Duration) ([]string, error) {
	url := fmt.Sprintf("https://ks-setting.kyberswap.com/api/v1/dexes?chain=%s&isEnabled=true&pageSize=100", chainName)
	
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch liquidity sources for chain %s: %w", chainName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed for chain %s with status: %s", chainName, resp.Status)
	}

	var response KyberSwapDexResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response for chain %s: %w", chainName, err)
	}

	if response.Code != 0 {
		return nil, fmt.Errorf("API returned error for chain %s: %s", chainName, response.Message)
	}

	// Extract dexId values
	var sources []string
	for _, dex := range response.Data.Dexes {
		sources = append(sources, dex.DexID)
	}

	return sources, nil
}
