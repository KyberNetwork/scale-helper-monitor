# Scale Helper Monitor

A monitoring service that checks the `getScaledInputData` function across multiple blockchains and sends batch Slack alerts when the function returns false.

## Overview

This service:

- Fetches routing data from KyberSwap API for various token pairs
- Calls the `getScaledInputData` function on smart contracts deployed across multiple chains
- Monitors the function's return value and sends Slack alerts when it returns `false`
- Provides detailed logging and error reporting

## Features

- **Multi-chain support**: Monitor contracts on Ethereum, Polygon, BSC, Arbitrum, and more
- **KyberSwap integration**: Automatically fetches encoded swap data from KyberSwap API
- **Slack alerts**: Detailed alerts with token information, chain details, and error context
- **Configurable monitoring**: Adjust intervals, timeouts, and token pairs

## Prerequisites

- Go 1.19+
- Access to blockchain RPC endpoints (Alchemy, Infura, etc.)
- Slack webhook URL for alerts
- Contract addresses for each chain you want to monitor

## Installation

1. Clone the repository:

```bash
git clone <repository-url>
cd scale-helper-monitor
```

2. Initialize Go modules:

```bash
go mod init scale-helper-monitor
go mod tidy
```

## Configuration

### Environment Variables

Checkout `env.example`

### Configuration File

The service reads from `config.yaml`. You can customize:

- **Monitoring interval**: How often to check contracts
- **API timeout**: Timeout for API and RPC calls
- **Test cases**: Test case to monitor on each chain

## Usage

### Running the Service

```bash
# Run with default configuration
go run .

# Build and run binary
go build -o scale-helper-monitor
./scale-helper-monitor
```

## Smart Contract Interface

The service expects contracts to implement:

```solidity
function getScaledInputData(
    bytes calldata inputData,
    uint256 newAmount
) external view returns (bool isSuccess, bytes memory data);
```

## API Integration

The service integrates with [KyberSwap Aggregator API](https://docs.kyberswap.com/kyberswap-solutions/kyberswap-aggregator/aggregator-api-specification/evm-swaps) to:

1. Fetch encoded swap data for token pairs
2. Get real-time pricing information
3. Generate test cases for the monitoring function

## Slack Alerts

When `getScaledInputData` returns `false`, the service collects all failures from a monitoring run and sends a single batch alert containing:

### Alert Summary

- **Total failures count**
- **Total test cases executed**
- **Success rate percentage**
- **Affected chains list**

### Individual Failure Details

For each failure, the alert includes:

- Chain name and ID
- Token pair information (symbols and addresses)
- Original and scaled transaction amounts
- Input data and returned data from the contract
- Error details and timestamp
- Tenderly simulation links (original and scaled swaps)

### Route Sequence Information

Detailed swap route breakdown showing:

- **Number of sequence steps**
- **Pool information for each step**:
  - Pool type (e.g., UniswapV2, UniswapV3, Curve)
  - Exchange name
  - Pool-specific details

### Example Alert Structure

```
🚨 Scale Helper Monitor Alert - 2 Failures

📊 Alert Summary
Total Failures: 2
Total Test Cases: 10
Success Rate: 80.00%
Affected Chains: ethereum, arbitrum

❌ Failure 1: ethereum
Chain: ethereum
Token Addresses: In: 0x..., Out: 0x...
Sequence Details:
--- Step 1 ---
Pool 1:
  Type: UniswapV3
  Exchange: Uniswap V3
--- Step 2 ---
Pool 1:
  Type: Curve
  Exchange: Curve Finance
```

## Monitoring Logic

For each configured token pair, the service:

1. **Fetches route**: Calls KyberSwap API to get encoded swap data with detailed sequence information
2. **Scales amount**: Creates a new amount (random percentage) to test scaling
3. **Simulates original**: Uses Tenderly to verify the original swap works
4. **Calls contract**: Invokes `getScaledInputData` with the encoded data and new amount
5. **Simulates scaled**: Uses Tenderly to test the scaled swap data
6. **Sends batch alert**: After all test cases complete, sends a single consolidated alert if there are failures
7. **Logs activity**: Records all operations with structured logging and success rate tracking
