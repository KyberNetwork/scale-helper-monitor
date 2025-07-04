package tenderly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Constants for special addresses
const (
	ZERO_ADDRESS   = "0x0000000000000000000000000000000000000000"
	NATIVE_ADDRESS = "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"
)

// NewClient creates a new Tenderly client
func NewClient(accessKey, username, project string) *Client {
	return &Client{
		accessKey: accessKey,
		username:  username,
		project:   project,
		baseURL:   "https://api.tenderly.co/api/v1",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateStateObjectsForSwap creates state objects for token balances and approvals
func (c *Client) CreateStateObjectsForSwap(tokenIn, routerAddress, fromAddress, amount string, chainName string, balanceSlot *map[string]map[string]map[string]string) (map[string]interface{}, error) {
	stateObjects := make(map[string]interface{})
	
	stateObjects[fromAddress] = map[string]interface{}{
		"balance": "0xffffffffffffffffffffffffffff",
	}
	
	// Skip token state manipulation for native token
	if strings.EqualFold(tokenIn, NATIVE_ADDRESS) {
		return stateObjects, nil
	}

	storageSlot := (*balanceSlot)[fromAddress][chainName][tokenIn]

	if storageSlot == "" {
		return nil, fmt.Errorf("balance slot not found for token %s on chain %s", tokenIn, chainName)
	}
	
	// Set token balance following sim.py pattern
	stateObjects[tokenIn] = map[string]interface{}{
		"storage": map[string]string{
			storageSlot: "0x7fffffffffffffff0123456789abcdef", 
		},
	}
	
	return stateObjects, nil
}

// SimulateTransactionBundle simulates a transaction using Tenderly's simulate-bundle endpoint
func (c *Client) SimulateTransactionBundle(ctx context.Context, bundleReq *SimulationBundleRequest) (*SimulationBundleResponse, error) {
	url := fmt.Sprintf("%s/account/%s/project/%s/simulate-bundle", c.baseURL, c.username, c.project)
	
	jsonData, err := json.Marshal(bundleReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal simulation bundle request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Access-Key", c.accessKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("simulation failed with status %d %s", resp.StatusCode, resp.Body)
	}

	var bundleResp SimulationBundleResponse
	if err := json.NewDecoder(resp.Body).Decode(&bundleResp); err != nil {
		return nil, fmt.Errorf("failed to decode simulation bundle response: %w", err)
	}

	return &bundleResp, nil
}

// SimulateTransaction provides a simpler interface that matches our existing code
func (c *Client) SimulateTransaction(ctx context.Context, networkID, tokenIn ,from, to, input, value string, stateObjects map[string]interface{}) (bool, string, string, error) {
	approvalReq := c.CreateApprovalData(networkID, from, to, tokenIn)
	swapReq := &SimulationRequest{
		NetworkID:    networkID,
		From:         from,
		To:           to,
		GasLimit:          9999999999999, 
		Value:        value,
		Input:        input,
		Save:         true,
		SaveIfFails:  true,
		SimulationType: "quick",
		StateObjects: stateObjects,
	}
	
	bundleResp, err := c.SimulateTransactionBundle(ctx, &SimulationBundleRequest{
		Simulations: []SimulationRequest{*approvalReq, *swapReq},
	})
	if err != nil {
		return false, "", "", err
	}
	
	if len(bundleResp.SimulationResults) == 0 {
		return false, "", "", fmt.Errorf("no simulation results returned")
	}
	
	result := bundleResp.SimulationResults[1]
	
	// Generate Tenderly URL
	tenderlyURL := fmt.Sprintf("https://dashboard.tenderly.co/%s/%s/simulator/%s", 
		c.username, c.project, result.Simulation.ID)
	
	// Check if transaction was successful
	if result.Transaction != nil {
		return result.Transaction.Status, result.Simulation.ErrorMessage, tenderlyURL, nil
	}
	
	// If transaction is nil, it failed
	return false, result.Simulation.ErrorMessage, tenderlyURL, nil
}

// GetChainNetworkID converts chain ID to Tenderly network ID
func GetChainNetworkID(chainID int) string {
	networkMap := map[int]string{
		1:     "1",     // Ethereum
		56:    "56",    // BSC
		137:   "137",   // Polygon
		43114: "43114", // Avalanche
		250:   "250",   // Fantom
		42161: "42161", // Arbitrum
		10:    "10",    // Optimism
		8453:  "8453",  // Base
		// Add more as needed
	}
	
	if networkID, exists := networkMap[chainID]; exists {
		return networkID
	}
	return fmt.Sprintf("%d", chainID)
}

// CreateApprovalData creates approval transaction data for token approvals
func (c *Client) CreateApprovalData(networkID, sender, routerAddress, tokenToApprove string) *SimulationRequest {
	amount := "ffffffffffffffffffffffffffffffff"
	
	// Remove 0x prefix from router address
	routerWithoutPrefix := strings.TrimPrefix(routerAddress, "0x")
	
	// ERC20 approve function: approve(address spender, uint256 amount)
	// Function selector: 0x095ea7b3
	input := fmt.Sprintf("0x095ea7b3000000000000000000000000%s00000000000000000000000000000000%s", 
		routerWithoutPrefix, amount)
	
	return &SimulationRequest{
		Save:            true,
		SaveIfFails:   true,
		SimulationType: "quick",
		NetworkID:      networkID,
		From:            sender,
		GasLimit:          9999999999999,
		Value:           "0x0",
		To:              tokenToApprove,
		Input:           input,
	}
}