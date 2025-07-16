package monitor

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	distributor "scale-helper-monitor/internal/config/distributor"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/slack-go/slack"
)

// Start begins monitoring for events on a specific network
func (m *Monitor) Start(ctx context.Context) error {
	m.Logger.Infof("🚀 Starting distributor monitor for %s network (Chain ID: %d)...", m.Network.Name, m.Network.ChainID)

	// Test RPC connection first with timeout
	connectionCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	chainID, err := m.Client.ChainID(connectionCtx)
	if err != nil {
		return fmt.Errorf("failed to connect to %s RPC: %v", m.Network.Name, err)
	}

	if chainID.Int64() != m.Network.ChainID {
		m.Logger.Warnf("⚠️  Chain ID mismatch for %s: expected %d, got %d", m.Network.Name, m.Network.ChainID, chainID.Int64())
	} else {
		m.Logger.Infof("✅ Successfully connected to %s (Chain ID: %d)", m.Network.Name, chainID.Int64())
	}

	// Get contract address
	contractAddress := common.HexToAddress(m.Network.ContractAddress)

	// Create event signature hash
	eventSignature := []byte("RootSubmitted(bytes32,bytes32,uint256)")
	eventSignatureHash := crypto.Keccak256Hash(eventSignature)

	// Get starting block - try to load from state first
	startBlock := m.Network.StartBlock
	if savedBlock, err := m.StateManager.LoadNetworkState(m.Network.Name); err == nil {
		// Use saved block + 1 to continue from where we left off
		startBlock = savedBlock + 1
		m.Logger.Infof("Resuming from saved state for %s: block %d", m.Network.Name, startBlock)
	} else if startBlock == 0 {
		// No saved state and no configured start block, use latest
		latestBlock, err := m.Client.HeaderByNumber(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to get latest block for %s: %v", m.Network.Name, err)
		}
		startBlock = latestBlock.Number.Uint64()
		m.Logger.Infof("No saved state found for %s, starting from latest block: %d", m.Network.Name, startBlock)
	} else {
		m.Logger.Infof("No saved state found for %s, starting from configured block: %d", m.Network.Name, startBlock)
	}

	m.Logger.Infof("Monitoring %s contract %s starting from block %d", m.Network.Name, contractAddress.Hex(), startBlock)

	// Start monitoring loop
	ticker := time.NewTicker(time.Duration(m.Config.Monitoring.PollInterval) * time.Second)
	defer ticker.Stop()

	currentBlock := startBlock

	for {
		select {
		case <-ctx.Done():
			m.Logger.Infof("Context cancelled, stopping %s monitor...", m.Network.Name)
			// Save final state before stopping
			if currentBlock > 0 {
				if err := m.StateManager.SaveNetworkState(m.Network.Name, m.Network.ChainID, currentBlock-1); err != nil {
					m.Logger.Errorf("Failed to save final state for %s: %v", m.Network.Name, err)
				} else {
					m.Logger.Infof("Saved final state for %s: block %d", m.Network.Name, currentBlock-1)
				}
			}
			return ctx.Err()
		case <-ticker.C:
			// Get latest block with timeout
			blockCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			latestBlock, err := m.Client.HeaderByNumber(blockCtx, nil)
			cancel()

			if err != nil {
				m.Logger.Errorf("❌ Failed to get latest block for %s: %v", m.Network.Name, err)
				// Add a small delay before retrying to avoid spamming
				time.Sleep(5 * time.Second)
				continue
			}

			// Calculate end block for this batch
			toBlock := latestBlock.Number.Uint64()
			if currentBlock > toBlock {
				continue // No new blocks
			}

			// Track if we processed any blocks for state saving
			originalCurrentBlock := currentBlock

			// Limit batch size
			if toBlock-currentBlock > uint64(m.Network.BatchSize) {
				toBlock = currentBlock + uint64(m.Network.BatchSize)
			}

			m.Logger.Infof("🔍 Checking %s blocks %d to %d (latest: %d)", m.Network.Name, currentBlock, toBlock, latestBlock.Number.Uint64())

			// Query for events
			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(currentBlock)),
				ToBlock:   big.NewInt(int64(toBlock)),
				Addresses: []common.Address{contractAddress},
				Topics:    [][]common.Hash{{eventSignatureHash}},
			}

			// Query logs with timeout
			queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			logs, err := m.Client.FilterLogs(queryCtx, query)
			cancel()

			if err != nil {
				m.Logger.Errorf("❌ Failed to filter logs for %s: %v", m.Network.Name, err)
				continue
			}

			if len(logs) > 0 {
				m.Logger.Infof("📝 Found %d log(s) for %s in blocks %d-%d", len(logs), m.Network.Name, currentBlock, toBlock)
			}

			// Process each log
			for _, log := range logs {
				event, err := m.parseRootSubmittedEvent(log)
				if err != nil {
					m.Logger.Errorf("Failed to parse event: %v", err)
					continue
				}

				m.Logger.Infof("Found RootSubmitted event on %s: CampaignId=%s, Block=%d",
					m.Network.Name, fmt.Sprintf("0x%x", event.CampaignId), event.BlockNumber)

				// Send Slack notification
				if err := m.sendSlackNotification(event); err != nil {
					m.Logger.Errorf("Failed to send Slack notification: %v", err)
				}
			}

			// Update current block for next iteration
			currentBlock = toBlock + 1

			// Save state after processing blocks (only if we actually processed some blocks)
			if toBlock >= originalCurrentBlock {
				if err := m.StateManager.SaveNetworkState(m.Network.Name, m.Network.ChainID, toBlock); err != nil {
					m.Logger.Errorf("Failed to save state for %s: %v", m.Network.Name, err)
				} else {
					m.Logger.Debugf("Saved state for %s: block %d", m.Network.Name, toBlock)
				}
			}
		}
	}
}

// parseRootSubmittedEvent parses the RootSubmitted event from a log entry
func (m *Monitor) parseRootSubmittedEvent(log types.Log) (*RootSubmittedEvent, error) {
	// Parse the event using ABI
	// campaignId is indexed (in Topics), pendingRoot and effectiveTimestamp are in Data
	event := struct {
		PendingRoot        [32]byte
		EffectiveTimestamp *big.Int
	}{}

	err := m.ContractABI.UnpackIntoInterface(&event, "RootSubmitted", log.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %v", err)
	}

	// Extract campaignId from Topics[1] (Topics[0] is the event signature hash)
	var campaignId [32]byte
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("missing campaignId in event topics")
	}
	copy(campaignId[:], log.Topics[1].Bytes())

	// Get block timestamp - fallback to current time if block retrieval fails
	var blockTimestamp time.Time
	block, err := m.Client.BlockByNumber(context.Background(), big.NewInt(int64(log.BlockNumber)))
	if err != nil {
		m.Logger.Warnf("Failed to get block %d timestamp for %s: %v - using current time", log.BlockNumber, m.Network.Name, err)
		blockTimestamp = time.Now()
	} else {
		blockTimestamp = time.Unix(int64(block.Time()), 0)
	}

	return &RootSubmittedEvent{
		CampaignId:         campaignId,
		PendingRoot:        event.PendingRoot,
		EffectiveTimestamp: event.EffectiveTimestamp,
		BlockNumber:        log.BlockNumber,
		TxHash:             log.TxHash.Hex(),
		Timestamp:          blockTimestamp,
		Network:            m.Network,
	}, nil
}

// sendSlackNotification sends a notification to Slack
func (m *Monitor) sendSlackNotification(event *RootSubmittedEvent) error {
	// Format the effective timestamp
	effectiveTime := time.Unix(event.EffectiveTimestamp.Int64(), 0)

	// Get network emoji based on chain ID
	emoji := distributor.GetNetworkEmoji(event.Network.ChainID)

	// Create Slack attachment with detailed information
	attachment := slack.Attachment{
		Color: "good",
		Title: fmt.Sprintf("%s Root Submitted", emoji),
		Text:  fmt.Sprintf("A new root has been submitted on %s network", event.Network.Name),
		Fields: []slack.AttachmentField{
			{
				Title: "Campaign ID",
				Value: fmt.Sprintf("`0x%x`", event.CampaignId),
				Short: true,
			},
			{
				Title: "Pending Root",
				Value: fmt.Sprintf("`0x%x`", event.PendingRoot),
				Short: true,
			},
			{
				Title: "Effective Timestamp",
				Value: effectiveTime.Format("2006-01-02 15:04:05 UTC"),
				Short: true,
			},
			{
				Title: "Block Number",
				Value: fmt.Sprintf("%d", event.BlockNumber),
				Short: true,
			},
			{
				Title: "Transaction Hash",
				Value: fmt.Sprintf("`%s`", event.TxHash),
				Short: false,
			},
			{
				Title: "Block Timestamp",
				Value: event.Timestamp.Format("2006-01-02 15:04:05 UTC"),
				Short: true,
			},
		},
		Footer: "Distributor Monitor",
	}

	_, _, err := m.SlackClient.PostMessage(
		m.Config.Slack.Channel,
		slack.MsgOptionAttachments(attachment),
	)

	if err != nil {
		return fmt.Errorf("failed to post Slack message: %v", err)
	}

	m.Logger.Infof("Slack notification sent successfully for %s", event.Network.Name)
	return nil
}

// Start begins monitoring all configured networks
func (mnm *MultiNetworkMonitor) Start(ctx context.Context) error {
	mnm.Logger.Infof("Starting multi-network distributor monitor for %d networks...", len(mnm.Monitors))

	if len(mnm.Monitors) > 0 {
		mnm.Logger.Infof("State will be saved to: %s", mnm.Monitors[0].StateManager.GetStateFilePath())
		// Display current state for all networks
		mnm.Monitors[0].StateManager.DisplayCurrentState(mnm.Logger)
	}

	// Create a context that can be cancelled
	monitorCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start each monitor in a separate goroutine
	for i, monitor := range mnm.Monitors {
		mnm.Logger.Infof("Starting monitor %d/%d for %s network (Chain ID: %d)...", i+1, len(mnm.Monitors), monitor.Network.Name, monitor.Network.ChainID)

		go func(m *Monitor, index int) {
			mnm.Logger.Infof("Goroutine %d: Starting %s monitor...", index+1, m.Network.Name)

			if err := m.Start(monitorCtx); err != nil && err != context.Canceled {
				mnm.Logger.Errorf("Monitor %d (%s) stopped with error: %v", index+1, m.Network.Name, err)
			} else if err == context.Canceled {
				mnm.Logger.Infof("Monitor %d (%s) stopped due to context cancellation", index+1, m.Network.Name)
			} else {
				mnm.Logger.Infof("Monitor %d (%s) stopped normally", index+1, m.Network.Name)
			}
		}(monitor, i)

		// Small delay between starting monitors to avoid overwhelming logs
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for context cancellation
	<-ctx.Done()
	mnm.Logger.Info("Stopping all network monitors...")

	// Cancel all monitors
	cancel()

	// Give monitors a moment to clean up and save final state
	time.Sleep(2 * time.Second)

	mnm.Logger.Info("All network monitors stopped")
	return ctx.Err()
}

// CreateContractABI creates the contract ABI for the RootSubmitted event
func CreateContractABI() (abi.ABI, error) {
	// Define the ABI for the RootSubmitted event
	const abiJSON = `[
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": true,
					"internalType": "bytes32",
					"name": "campaignId",
					"type": "bytes32"
				},
				{
					"indexed": false,
					"internalType": "bytes32",
					"name": "pendingRoot",
					"type": "bytes32"
				},
				{
					"indexed": false,
					"internalType": "uint256",
					"name": "effectiveTimestamp",
					"type": "uint256"
				}
			],
			"name": "RootSubmitted",
			"type": "event"
		}
	]`

	contractABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to parse ABI: %v", err)
	}

	return contractABI, nil
}
