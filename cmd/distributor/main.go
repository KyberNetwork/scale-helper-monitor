package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	distributor "scale-helper-monitor/internal/config/distributor"
	monitor "scale-helper-monitor/internal/monitor/distributor"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

// NewMonitor creates a new monitor instance for a specific network
func NewMonitor(cfg *distributor.Config, network *distributor.Network, logger *logrus.Logger) (*monitor.Monitor, error) {
	// Connect to network RPC client
	client, err := ethclient.Dial(network.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s client: %v", network.Name, err)
	}

	// Initialize Slack client
	slackClient := slack.New(cfg.Slack.Token)

	// Create contract ABI for the RootSubmitted event
	contractABI, err := monitor.CreateContractABI()
	if err != nil {
		return nil, fmt.Errorf("failed to create contract ABI: %v", err)
	}

	// Initialize state manager
	stateManager := monitor.NewStateManager(cfg.Monitoring.StateFile)

	return &monitor.Monitor{
		Client:       client,
		SlackClient:  slackClient,
		Config:       cfg,
		Network:      network,
		Logger:       logger,
		ContractABI:  contractABI,
		StateManager: stateManager,
	}, nil
}

// NewMultiNetworkMonitor creates a new multi-network monitor
func NewMultiNetworkMonitor(cfg *distributor.Config) (*monitor.MultiNetworkMonitor, error) {
	// Setup logger
	logger := logrus.New()
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	if cfg.Logging.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{})
	}

	// Load networks from environment variables if not set in config
	networks := cfg.GetNetworks()

	// Log which networks are configured
	if len(networks) == 0 {
		fmt.Fprintf(os.Stderr, "No networks configured. Please set environment variables for at least one network.\n")
		fmt.Fprintf(os.Stderr, "Example: ETH_NODE_URL and ETH_CONTRACT_ADDRESS for Ethereum\n")
		os.Exit(1)
	}

	logger.Infof("📋 Loaded %d network configurations:", len(networks))
	for i, network := range networks {
		emoji := distributor.GetNetworkEmoji(network.ChainID)
		logger.Infof("  %d. %s %s (Chain ID: %d) - Contract: %s", i+1, emoji, network.Name, network.ChainID, network.ContractAddress)
	}

	// Create monitors for each network
	var monitors []*monitor.Monitor
	for i := range networks {
		network := &networks[i]
		if network.RPCURL == "" {
			logger.Warnf("Skipping %s network: no RPC URL configured", network.Name)
			continue
		}
		if network.ContractAddress == "" {
			logger.Warnf("Skipping %s network: no contract address configured", network.Name)
			continue
		}

		monitor, err := NewMonitor(cfg, network, logger)
		if err != nil {
			logger.Errorf("Failed to create monitor for %s: %v", network.Name, err)
			continue
		}
		monitors = append(monitors, monitor)
		logger.Infof("✅ Created monitor for %s network (Chain ID: %d) - Contract: %s", network.Name, network.ChainID, network.ContractAddress)
	}

	if len(monitors) == 0 {
		return nil, fmt.Errorf("no valid network configurations found")
	}

	return &monitor.MultiNetworkMonitor{
		Monitors: monitors,
		Config:   cfg,
		Logger:   logger,
	}, nil
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: Error loading .env file: %v\n", err)
		fmt.Println("Continuing with system environment variables...")
	}

	// Load configuration
	cfg, err := distributor.LoadConfig("distributor-config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Config validation error: %v\n", err)
		os.Exit(1)
	}

	// Create multi-network monitor
	multiMonitor, err := NewMultiNetworkMonitor(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating multi-network monitor: %v\n", err)
		os.Exit(1)
	}

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		multiMonitor.Logger.Info("Received interrupt signal, shutting down...")
		cancel()
	}()

	// Start monitoring
	if err := multiMonitor.Start(ctx); err != nil && err != context.Canceled {
		multiMonitor.Logger.Errorf("Multi-network monitor error: %v", err)
		os.Exit(1)
	}

	multiMonitor.Logger.Info("Multi-network monitor stopped")
}
