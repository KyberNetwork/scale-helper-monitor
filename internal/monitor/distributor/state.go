package monitor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// NetworkState represents the monitoring state for a network
type NetworkState struct {
	NetworkName   string `json:"network_name"`
	ChainID       int64  `json:"chain_id"`
	LastProcessed uint64 `json:"last_processed_block"`
	LastUpdated   string `json:"last_updated"`
}

// StateManager handles persistence of monitoring state
type StateManager struct {
	stateFile string
}

// NewStateManager creates a new state manager
func NewStateManager(stateFile string) *StateManager {
	return &StateManager{
		stateFile: stateFile,
	}
}

// SaveNetworkState saves the last processed block for a network
func (sm *StateManager) SaveNetworkState(networkName string, chainID int64, lastBlock uint64) error {
	// Load existing states
	states, err := sm.LoadAllStates()
	if err != nil {
		states = make(map[string]NetworkState)
	}

	// Update state for this network
	states[networkName] = NetworkState{
		NetworkName:   networkName,
		ChainID:       chainID,
		LastProcessed: lastBlock,
		LastUpdated:   time.Now().Format("2006-01-02 15:04:05 UTC"),
	}

	// Save to file
	return sm.saveStatesToFile(states)
}

// LoadNetworkState loads the last processed block for a network
func (sm *StateManager) LoadNetworkState(networkName string) (uint64, error) {
	states, err := sm.LoadAllStates()
	if err != nil {
		return 0, err
	}

	if state, exists := states[networkName]; exists {
		return state.LastProcessed, nil
	}

	return 0, fmt.Errorf("no state found for network %s", networkName)
}

// LoadAllStates loads all network states from file
func (sm *StateManager) LoadAllStates() (map[string]NetworkState, error) {
	states := make(map[string]NetworkState)

	// Check if file exists
	if _, err := os.Stat(sm.stateFile); os.IsNotExist(err) {
		return states, nil // Return empty map if file doesn't exist
	}

	// Read file
	data, err := ioutil.ReadFile(sm.stateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %v", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, &states); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %v", err)
	}

	return states, nil
}

// saveStatesToFile saves states to the JSON file
func (sm *StateManager) saveStatesToFile(states map[string]NetworkState) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(sm.stateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %v", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal states: %v", err)
	}

	// Write to file
	if err := ioutil.WriteFile(sm.stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %v", err)
	}

	return nil
}

// GetStateFilePath returns the path to the state file
func (sm *StateManager) GetStateFilePath() string {
	return sm.stateFile
}

// DisplayCurrentState logs the current state of all networks
func (sm *StateManager) DisplayCurrentState(logger *logrus.Logger) {
	states, err := sm.LoadAllStates()
	if err != nil {
		logger.Errorf("Failed to load states: %v", err)
		return
	}

	if len(states) == 0 {
		logger.Info("No saved states found - starting fresh")
		return
	}

	logger.Info("Current saved states:")
	for networkName, state := range states {
		logger.Infof("  %s (Chain %d): Last processed block %d (Updated: %s)",
			networkName, state.ChainID, state.LastProcessed, state.LastUpdated)
	}
}

// ClearState removes all saved state (for debugging purposes)
func (sm *StateManager) ClearState() error {
	if err := os.Remove(sm.stateFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear state file: %v", err)
	}
	return nil
}
