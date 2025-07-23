package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// Network represents a blockchain network configuration
type Network struct {
	Name            string `yaml:"name"`
	ChainID         int64  `yaml:"chain_id"`
	RPCURL          string `yaml:"rpc_url"`
	ContractAddress string `yaml:"contract_address"`
	StartBlock      uint64 `yaml:"start_block"`
	BatchSize       int    `yaml:"batch_size"`
}

// ChainConfig represents a chain configuration from Chains.json
type ChainConfig struct {
	Name             string `json:"name"`
	ChainID          int64  `json:"chainId"`
	Emoji            string `json:"emoji"`
	EnvPrefix        string `json:"envPrefix"`
	DefaultBatchSize int    `json:"defaultBatchSize"`
	BlockTime        string `json:"blockTime"`
	Category         string `json:"category"`
}

// ChainsData represents the structure of Chains.json
type ChainsData struct {
	Chains []ChainConfig `json:"chains"`
}

// ContractsData represents the structure of Contracts.json
type ContractsData struct {
	Contracts map[string]string `json:"contracts"`
	Comment   string            `json:"_comment"`
}

var (
	chainsData    *ChainsData
	chainsMap     map[int64]ChainConfig
	contractsData *ContractsData
)

// loadChainsConfig loads chain configurations from Chains.json
func loadChainsConfig() error {
	if chainsData != nil {
		return nil // Already loaded
	}

	data, err := ioutil.ReadFile("config/distributor/Chains.json")
	if err != nil {
		return fmt.Errorf("failed to read Chains.json: %v", err)
	}

	chainsData = &ChainsData{}
	err = json.Unmarshal(data, chainsData)
	if err != nil {
		return fmt.Errorf("failed to parse Chains.json: %v", err)
	}

	// Create a map for quick lookups
	chainsMap = make(map[int64]ChainConfig)
	for _, chain := range chainsData.Chains {
		chainsMap[chain.ChainID] = chain
	}

	return nil
}

// loadContractsConfig loads contract configurations from Contracts.json
func loadContractsConfig() error {
	if contractsData != nil {
		return nil // Already loaded
	}

	data, err := ioutil.ReadFile("config/distributor/Contracts.json")
	if err != nil {
		return fmt.Errorf("failed to read Contracts.json: %v", err)
	}

	contractsData = &ContractsData{}
	err = json.Unmarshal(data, contractsData)
	if err != nil {
		return fmt.Errorf("failed to parse Contracts.json: %v", err)
	}

	return nil
}

// LoadNetworksFromEnv loads network configurations from environment variables
func LoadNetworksFromEnv(configNetworks []Network, globalBatchSize int) []Network {
	// Load chains configuration
	if err := loadChainsConfig(); err != nil {
		fmt.Printf("Warning: Failed to load chains config: %v\n", err)
		return []Network{}
	}

	// Load contracts configuration
	if err := loadContractsConfig(); err != nil {
		fmt.Printf("Warning: Failed to load contracts config: %v\n", err)
		// If contracts config fails, we'll use environment variables for contract addresses
	}

	// If networks are defined in config, use them; otherwise use environment variables
	if len(configNetworks) > 0 {
		// Apply default batch sizes to config networks if not set
		for i := range configNetworks {
			if configNetworks[i].BatchSize <= 0 {
				// Try to get default from chains config
				if chainConfig, exists := chainsMap[configNetworks[i].ChainID]; exists {
					configNetworks[i].BatchSize = chainConfig.DefaultBatchSize
				} else {
					configNetworks[i].BatchSize = globalBatchSize
				}
			}
		}
		return configNetworks
	}

	// Build networks from environment variables using chains config
	var validNetworks []Network

	for _, chainConfig := range chainsData.Chains {
		rpcURL := os.Getenv(chainConfig.EnvPrefix + "_NODE_URL")

		// Get contract address from Contracts.json only
		var contractAddress string
		if contractsData != nil {
			if addr, exists := contractsData.Contracts[chainConfig.Name]; exists {
				contractAddress = addr
			}
		}

		// Skip networks without RPC URL or with address 0 (not deployed yet)
		if rpcURL == "" || contractAddress == "" || contractAddress == "0x0000000000000000000000000000000000000000" {
			continue
		}

		// Use default batch size from chains config
		batchSize := chainConfig.DefaultBatchSize

		network := Network{
			Name:            chainConfig.Name,
			ChainID:         chainConfig.ChainID,
			RPCURL:          rpcURL,
			ContractAddress: contractAddress,
			BatchSize:       batchSize,
		}

		validNetworks = append(validNetworks, network)
	}

	return validNetworks
}

// GetConfiguredNetworks returns a summary of which networks are properly configured
func GetConfiguredNetworks() []string {
	networks := LoadNetworksFromEnv([]Network{}, 100) // Pass a default global batch size
	var configured []string

	for _, network := range networks {
		emoji := GetNetworkEmoji(network.ChainID)
		configured = append(configured, fmt.Sprintf("%s %s (Chain ID: %d)", emoji, network.Name, network.ChainID))
	}

	return configured
}

// ValidateNetworkConfig checks if the required environment variables are set for at least one network
func ValidateNetworkConfig() error {
	networks := LoadNetworksFromEnv([]Network{}, 100) // Pass a default global batch size

	if len(networks) == 0 {
		return fmt.Errorf("no networks configured - please set RPC URL and contract address for at least one network")
	}

	return nil
}

// GetNetworkEmoji returns an emoji for the given chain ID
func GetNetworkEmoji(chainID int64) string {
	if err := loadChainsConfig(); err != nil {
		return "⚡" // Default emoji if config can't be loaded
	}

	if chainConfig, exists := chainsMap[chainID]; exists {
		return chainConfig.Emoji
	}

	return "⚡" // Default emoji for unknown chains
}

// SupportedNetworks returns a list of all supported networks with their metadata
func SupportedNetworks() []Network {
	if err := loadChainsConfig(); err != nil {
		return []Network{} // Return empty if config can't be loaded
	}

	var networks []Network
	for _, chainConfig := range chainsData.Chains {
		networks = append(networks, Network{
			Name:    chainConfig.Name,
			ChainID: chainConfig.ChainID,
		})
	}

	return networks
}

// GetChainConfig returns the chain configuration for a given chain ID
func GetChainConfig(chainID int64) (ChainConfig, bool) {
	if err := loadChainsConfig(); err != nil {
		return ChainConfig{}, false
	}

	config, exists := chainsMap[chainID]
	return config, exists
}

// GetAllChainConfigs returns all chain configurations
func GetAllChainConfigs() ([]ChainConfig, error) {
	if err := loadChainsConfig(); err != nil {
		return nil, err
	}

	return chainsData.Chains, nil
}

// GetContractAddress returns the contract address for a given network name
func GetContractAddress(networkName string) (string, bool) {
	if err := loadContractsConfig(); err != nil {
		return "", false
	}

	address, exists := contractsData.Contracts[networkName]
	return address, exists
}

// GetAllContracts returns all contract addresses
func GetAllContracts() (map[string]string, error) {
	if err := loadContractsConfig(); err != nil {
		return nil, err
	}

	return contractsData.Contracts, nil
}
