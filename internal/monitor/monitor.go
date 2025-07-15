package monitor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"

	"scale-helper-monitor/internal/clients/kyberswap"
	"scale-helper-monitor/internal/clients/slack"
	"scale-helper-monitor/internal/clients/tenderly"
)

// Monitor represents the main monitoring service
type Monitor struct {
	config           *Config
	testCases        []TestCase
	tokens           map[string]map[string]TokenInfo // chain name -> token address -> token info
	chains           []ChainConfig
	liquiditySources map[string][]string
	kyberClient      *kyberswap.Client
	slackClient      *slack.Client
	tenderlyClient   *tenderly.Client
	ethClients       map[string]*ethclient.Client
	contractABI      abi.ABI
	logger           *logrus.Logger
}

// NewMonitor creates a new monitoring service
func NewMonitor(
	config *Config,
	testCases []TestCase,
	tokens map[string]map[string]TokenInfo,
	liquiditySources map[string][]string,
	chains []ChainConfig,
	kyberClient *kyberswap.Client,
	slackClient *slack.Client,
	tenderlyClient *tenderly.Client,
	logger *logrus.Logger,
) (*Monitor, error) {

	// Create Ethereum clients for each chain
	ethClients := make(map[string]*ethclient.Client)
	for _, chain := range chains {
		if chain.RPCURL == "" {
			logger.WithField("chain", chain.Name).Warn("RPC URL not configured for chain, skipping")
			continue
		}

		client, err := ethclient.Dial(chain.RPCURL)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"chain": chain.Name,
				"error": err,
			}).Error("Failed to connect to RPC endpoint")
			continue
		}

		ethClients[chain.Name] = client
	}

	// Create contract ABI
	contractABI, err := createContractABI()
	if err != nil {
		return nil, fmt.Errorf("failed to create contract ABI: %w", err)
	}

	return &Monitor{
		config:           config,
		chains:           chains,
		liquiditySources: liquiditySources,
		kyberClient:      kyberClient,
		slackClient:      slackClient,
		tenderlyClient:   tenderlyClient,
		ethClients:       ethClients,
		contractABI:      contractABI,
		logger:           logger,
		tokens:           tokens,
		testCases:        testCases,
	}, nil
}

// createContractABI creates the ABI for the getScaledInputData function
func createContractABI() (abi.ABI, error) {
	abiJSON := `[{
		"inputs": [
			{
				"internalType": "bytes",
				"name": "inputData",
				"type": "bytes"
			},
			{
				"internalType": "uint256",
				"name": "newAmount",
				"type": "uint256"
			}
		],
		"name": "getScaledInputData",
		"outputs": [
			{
				"internalType": "bool",
				"name": "isSuccess",
				"type": "bool"
			},
			{
				"internalType": "bytes",
				"name": "data",
				"type": "bytes"
			}
		],
		"stateMutability": "view",
		"type": "function"
	}]`

	return abi.JSON(strings.NewReader(abiJSON))
}

// MonitorChain monitors a specific chain with a test token pair
func (m *Monitor) MonitorChain(ctx context.Context, testCase TestCase) (*Result, error) {
	amount, err := strconv.ParseFloat(testCase.Amount, 64)

	if err != nil {
		return nil, fmt.Errorf("invalid amount format: %s", testCase.Amount)
	}

	decimals, ok := new(big.Int).SetString(m.tokens[testCase.ChainName][testCase.TokenIn].Decimals, 10)
	if !ok {
		return nil, fmt.Errorf("invalid decimals format: %s", m.tokens[testCase.ChainName][testCase.TokenIn].Decimals)
	}

	// Check if amount is a integer
	if amount == float64(int64(amount)) {
		// For whole numbers, create string with zeros appended
		zeros := strings.Repeat("0", int(decimals.Int64()))
		testCase.Amount = testCase.Amount + zeros
	} else {
		amountInt := int64(amount * math.Pow10(int(decimals.Int64())))
		testCase.Amount = big.NewInt(amountInt).String()
	}

	// Find the chain config
	var chainConfig *ChainConfig
	for _, chain := range m.chains {
		if chain.Name == testCase.ChainName {
			chainConfig = &chain
			break
		}
	}

	if chainConfig == nil {
		return nil, fmt.Errorf("chain %s not found in configuration", testCase.ChainName)
	}

	// Get Ethereum client
	ethClient, exists := m.ethClients[chainConfig.Name]
	if !exists {
		return nil, fmt.Errorf("ethereum client not available for chain %s", chainConfig.Name)
	}

	// Fetch route from KyberSwap
	routeEncodedData, route, err := m.kyberClient.GetRoute(
		chainConfig.Name,
		testCase.TokenIn,
		testCase.TokenOut,
		testCase.Amount,
		m.liquiditySources[chainConfig.Name],
		testCase.IncludedSources,
	)

	if err != nil {
		return &Result{
			ChainName: testCase.ChainName,
			TokenIn:   m.tokens[testCase.ChainName][testCase.TokenIn].Symbol,
			TokenOut:  m.tokens[testCase.ChainName][testCase.TokenOut].Symbol,
			Amount:    testCase.Amount,
			Error:     fmt.Sprintf("Failed to fetch route: %v", err),
		}, err
	}

	// Step 1: Simulate original swap with Tenderly
	fromAddress := "0xdeAD00000000000000000000000000000000dEAd"

	// Create state objects for fake balances and
	stateObjects, err := m.tenderlyClient.CreateStateObjectsForSwap(
		testCase.TokenIn,
		routeEncodedData.RouterAddress,
		fromAddress,
		routeEncodedData.AmountIn,
		chainConfig.Name,
		m.tokens[chainConfig.Name][testCase.TokenIn].Slot,
	)
	if err != nil {
		return &Result{
			ChainName: chainConfig.Name,
			TokenIn:   m.tokens[testCase.ChainName][testCase.TokenIn].Symbol,
			TokenOut:  m.tokens[testCase.ChainName][testCase.TokenOut].Symbol,
			Amount:    testCase.Amount,
			Error:     fmt.Sprintf("Failed to create state objects: %v", err),
		}, err
	}

	// Simulate original swap
	originalSuccess, originalError, originalTenderlyURL, err := m.tenderlyClient.SimulateTransaction(
		ctx,
		tenderly.GetChainNetworkID(chainConfig.ChainID),
		testCase.TokenIn,
		fromAddress,
		routeEncodedData.RouterAddress,
		routeEncodedData.Data,
		routeEncodedData.TransactionValue,
		stateObjects,
	)
	if err != nil {
		return &Result{
			ChainName: chainConfig.Name,
			TokenIn:   m.tokens[testCase.ChainName][testCase.TokenIn].Symbol,
			TokenOut:  m.tokens[testCase.ChainName][testCase.TokenOut].Symbol,
			Amount:    testCase.Amount,
			Error:     fmt.Sprintf("Original Tenderly simulation failed: %v", err),
		}, err
	}

	// Check if original simulation succeeded
	if !originalSuccess {
		errorMsg := "Original swap simulation failed"
		if originalError != "" {
			errorMsg = fmt.Sprintf("Original swap failed: %s", originalError, "Tenderly URL: ", originalTenderlyURL)
		}

		return &Result{
			ChainName:           chainConfig.Name,
			TokenIn:             testCase.TokenIn,
			TokenOut:            testCase.TokenOut,
			Amount:              testCase.Amount,
			Error:               errorMsg,
			OriginalTenderlyURL: originalTenderlyURL,
		}, fmt.Errorf(errorMsg)
	}

	// Step 2: Call scale helper to get modified data
	inputData, err := hexutil.Decode(routeEncodedData.Data)
	if err != nil {
		return &Result{
			ChainName:           chainConfig.Name,
			TokenIn:             testCase.TokenIn,
			TokenOut:            testCase.TokenOut,
			Amount:              testCase.Amount,
			Error:               fmt.Sprintf("Failed to decode input data: %v", err),
			OriginalTenderlyURL: originalTenderlyURL,
		}, err
	}

	originalAmount, ok := new(big.Int).SetString(routeEncodedData.AmountIn, 10)

	if !ok {
		return &Result{
			ChainName:           chainConfig.Name,
			TokenIn:             testCase.TokenIn,
			TokenOut:            testCase.TokenOut,
			Amount:              testCase.Amount,
			Error:               "Failed to parse input amount",
			OriginalTenderlyURL: originalTenderlyURL,
		}, fmt.Errorf("failed to parse input amount")
	}
	newAmount := originalAmount
	scaleDown := rand.Intn(1000000)%2 == 0
	percentage := rand.Intn(20)
	if scaleDown {
		newAmount = new(big.Int).Mul(originalAmount, big.NewInt(int64(100-percentage)))
		newAmount = new(big.Int).Div(newAmount, big.NewInt(100))
	} else {
		newAmount = new(big.Int).Mul(originalAmount, big.NewInt(int64(100+percentage)))
		newAmount = new(big.Int).Div(newAmount, big.NewInt(100))
	}

	// Call the scale helper contract
	scaleResult, err := m.callGetScaledInputData(ctx, ethClient, chainConfig.ContractAddress, inputData, newAmount)
	if err != nil {
		return &Result{
			ChainName:           chainConfig.Name,
			TokenIn:             m.tokens[testCase.ChainName][testCase.TokenIn].Symbol,
			TokenOut:            m.tokens[testCase.ChainName][testCase.TokenOut].Symbol,
			Amount:              testCase.Amount,
			InputData:           routeEncodedData.Data,
			NewAmount:           newAmount.String(),
			Error:               fmt.Sprintf("Scale Failed: %v", err),
			Route:               route.Route,
			OriginalTenderlyURL: originalTenderlyURL,
		}, err
	}

	if !scaleResult.IsSuccess {
		scaleErr := &CallGetScaledInputDataError{
			ChainName: chainConfig.Name,
			Message:   "Scale helper returned false",
		}
		return &Result{
			ChainName:           chainConfig.Name,
			TokenIn:             testCase.TokenIn,
			TokenOut:            testCase.TokenOut,
			Amount:              testCase.Amount,
			IsSuccess:           scaleResult.IsSuccess,
			ReturnedData:        hexutil.Encode(scaleResult.Data),
			InputData:           routeEncodedData.Data,
			NewAmount:           newAmount.String(),
			Route:               route.Route,
			Error:               "Scale helper returned false",
			OriginalTenderlyURL: originalTenderlyURL,
		}, scaleErr
	}

	// Step 3: Simulate scaled swap with Tenderly
	scaledData := hexutil.Encode(scaleResult.Data)

	// Create state objects for scaled amount
	scaledStateObjects, err := m.tenderlyClient.CreateStateObjectsForSwap(
		testCase.TokenIn,
		routeEncodedData.RouterAddress,
		fromAddress,
		newAmount.String(),
		chainConfig.Name,
		m.tokens[chainConfig.Name][testCase.TokenIn].Slot,
	)
	if err != nil {
		return &Result{
			ChainName: chainConfig.Name,
			TokenIn:   m.tokens[testCase.ChainName][testCase.TokenIn].Symbol,
			TokenOut:  m.tokens[testCase.ChainName][testCase.TokenOut].Symbol,
			Amount:    testCase.Amount,
			Error:     fmt.Sprintf("Failed to create scaled state objects: %v", err),
		}, err
	}

	// Simulate scaled swap
	scaledSuccess, scaledError, scaledTenderlyURL, err := m.tenderlyClient.SimulateTransaction(
		ctx,
		tenderly.GetChainNetworkID(chainConfig.ChainID),
		testCase.TokenIn,
		fromAddress,
		routeEncodedData.RouterAddress,
		scaledData,
		routeEncodedData.TransactionValue,
		scaledStateObjects,
	)
	if err != nil {
		return &Result{
			ChainName:           chainConfig.Name,
			TokenIn:             testCase.TokenIn,
			TokenOut:            testCase.TokenOut,
			Amount:              testCase.Amount,
			IsSuccess:           scaleResult.IsSuccess,
			ReturnedData:        scaledData,
			InputData:           routeEncodedData.Data,
			NewAmount:           newAmount.String(),
			Route:               route.Route,
			Error:               fmt.Sprintf("Scaled Tenderly simulation failed: %v", err),
			OriginalTenderlyURL: originalTenderlyURL,
		}, err
	}

	// Step 4: Check if scaled simulation failed - if so, this triggers alert
	if !scaledSuccess {
		errorMsg := "Scaled swap simulation failed"
		if scaledError != "" {
			errorMsg = fmt.Sprintf("Scaled swap failed: %s", scaledError)
		}

		scaleErr := &CallGetScaledInputDataError{
			ChainName: chainConfig.Name,
			Message:   errorMsg,
		}

		return &Result{
			ChainName:           chainConfig.Name,
			TokenIn:             testCase.TokenIn,
			TokenOut:            testCase.TokenOut,
			Amount:              testCase.Amount,
			IsSuccess:           false, // Scaled simulation failed
			ReturnedData:        scaledData,
			InputData:           routeEncodedData.Data,
			NewAmount:           newAmount.String(),
			Route:               route.Route,
			Error:               errorMsg,
			OriginalTenderlyURL: originalTenderlyURL,
			ScaledTenderlyURL:   scaledTenderlyURL,
		}, scaleErr
	}

	return &Result{
		ChainName:           chainConfig.Name,
		TokenIn:             testCase.TokenIn,
		TokenOut:            testCase.TokenOut,
		Amount:              testCase.Amount,
		IsSuccess:           true,
		ReturnedData:        scaledData,
		InputData:           routeEncodedData.Data,
		NewAmount:           newAmount.String(),
		Route:               route.Route,
		OriginalTenderlyURL: originalTenderlyURL,
		ScaledTenderlyURL:   scaledTenderlyURL,
	}, nil
}

// callGetScaledInputData calls the getScaledInputData function on the contract
func (m *Monitor) callGetScaledInputData(ctx context.Context, client *ethclient.Client, contractAddress string, inputData []byte, newAmount *big.Int) (*ContractCallResult, error) {
	// Find the chain ID from the contract address
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %v", err)
	}

	var chainName string
	for _, chain := range m.chains {
		if chain.ChainID == int(chainID.Int64()) {
			chainName = chain.Name
			break
		}
	}

	// If chain not found, log error with more details
	if contractAddress == "" {
		m.logger.WithFields(logrus.Fields{
			"chainName": chainName,
			"chainID":   chainID.String(),
		}).Error("Chain not found")
		return nil, fmt.Errorf("chain not found")
	}

	// Pack the function call
	data, err := m.contractABI.Pack("getScaledInputData", inputData, newAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to pack function call: %v", err)
	}

	// Create call message
	msg := ethereum.CallMsg{
		To:   &common.Address{},
		Data: data, // fix later
	}

	// Parse contract address
	contractAddr := common.HexToAddress(contractAddress)
	msg.To = &contractAddr

	// Make the call
	result, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		// Check if this is a contract revert vs RPC failure
		errMsg := err.Error()
		if strings.Contains(errMsg, "execution reverted") ||
			strings.Contains(errMsg, "revert") ||
			strings.Contains(errMsg, "invalid opcode") ||
			strings.Contains(errMsg, "out of gas") {
			// This is a contract revert, create CallGetScaledInputDataError
			scaleErr := &CallGetScaledInputDataError{
				ChainName: chainName,
				Message:   errMsg,
			}
			return nil, scaleErr
		}
		// This is an RPC failure, return regular error
		return nil, fmt.Errorf("RPC call failed: %v", err)
	}

	// Unpack the result
	unpacked, err := m.contractABI.Unpack("getScaledInputData", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result")
	}

	// Extract values
	if len(unpacked) != 2 {
		return nil, fmt.Errorf("unexpected result length: %d", len(unpacked))
	}

	isSuccess, ok := unpacked[0].(bool)
	if !ok {
		return nil, fmt.Errorf("failed to parse scaled success")
	}

	data, ok = unpacked[1].([]byte)
	if !ok {
		return nil, fmt.Errorf("failed to parse data")
	}

	return &ContractCallResult{
		IsSuccess: isSuccess,
		Data:      data,
	}, nil
}

// RunMonitoringOnce runs monitoring for all test cases once and exits
func (m *Monitor) RunMonitoringOnce(ctx context.Context) error {
	m.logger.Info("Running one-shot monitoring check")

	var failures []slack.MonitoringResult

	// Monitor each test case
	for i, testCase := range m.testCases {
		result, err := m.MonitorChain(ctx, testCase)
		if err != nil {
			// Only collect failures for CallGetScaledInputDataError (scale helper or simulation failures)
			var scaleHelperErr *CallGetScaledInputDataError
			if errors.As(err, &scaleHelperErr) && result != nil {
				// Add to failures collection instead of sending individual alert
				failures = append(failures, result)
				m.logger.WithError(err).Error("Monitoring check failed")
			} else {
				// For other errors (API failures, network issues), just log
				m.logger.WithError(err).Warn("Monitoring check encountered error")
			}
			continue
		}

		// Log the result
		m.logger.WithFields(logrus.Fields{
			"chain":    result.ChainName,
			"tokenIn":  m.tokens[result.ChainName][result.TokenIn].Symbol,
			"tokenOut": m.tokens[result.ChainName][result.TokenOut].Symbol,
		}).Info(fmt.Sprintf("Test case %d completed", i+1))
	}

	if alertErr := m.slackClient.SendAlert(failures, len(m.testCases)); alertErr != nil {
		m.logger.WithError(alertErr).Error("Failed to send Slack alert")
	}
	m.logger.WithFields(logrus.Fields{
		"Total test cases": len(m.testCases),
		"Run on chains":    len(m.chains),
		"Failures":         len(failures),
		"Success Rate":     fmt.Sprintf("%.2f%%", float64(len(m.testCases)-len(failures))/float64(len(m.testCases))*100),
	}).Info("Monitoring check completed")

	m.logger.Info("One-shot monitoring completed")
	return nil
}

// RunMonitoring runs the monitoring loop
func (m *Monitor) RunMonitoring(ctx context.Context) error {
	// Parse interval
	interval, err := time.ParseDuration(m.config.Interval)
	if err != nil {
		return fmt.Errorf("invalid interval duration: %w", err)
	}

	m.logger.WithField("interval", interval).Info("Starting monitoring loop")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Monitoring loop stopped")
			return ctx.Err()

		case <-ticker.C:
			var failures []slack.MonitoringResult

			for i, testCase := range m.testCases {
				result, err := m.MonitorChain(ctx, testCase)
				if err != nil {
					// Only collect failures for CallGetScaledInputDataError (scale helper or simulation failures)
					var scaleHelperErr *CallGetScaledInputDataError
					if errors.As(err, &scaleHelperErr) && result != nil {
						failures = append(failures, result)
						m.logger.WithError(err).Error("Monitoring check failed")
					} else {
						// For other errors (API failures, network issues), just log
						m.logger.WithError(err).Warn("Monitoring check encountered error")
					}
					continue
				}

				// Log the result
				m.logger.WithFields(logrus.Fields{
					"chain":    result.ChainName,
					"tokenIn":  m.tokens[result.ChainName][result.TokenIn].Symbol,
					"tokenOut": m.tokens[result.ChainName][result.TokenOut].Symbol,
				}).Info(fmt.Sprintf("Test case %d completed", i+1))
			}

			// Send batch alert if there are any failures
			if len(failures) > 0 {
				if alertErr := m.slackClient.SendAlert(failures, len(m.testCases)); alertErr != nil {
					m.logger.WithError(alertErr).Error("Failed to send Slack alert")
				}
				m.logger.WithFields(logrus.Fields{
					"Total test cases": len(m.testCases),
					"Run on chains":    len(m.chains),
					"Failures":         len(failures),
				}).Info("Monitoring check completed")
			} else {
				m.logger.Info("All test cases passed successfully on all chains")
			}
		}
	}
}

// Close closes all connections
func (m *Monitor) Close() {
	for chainName, client := range m.ethClients {
		if client != nil {
			client.Close()
			m.logger.WithField("chain", chainName).Info("Closed RPC connection")
		}
	}
}
