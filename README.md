# Scale Helper Monitor

A monitoring service that checks the `getScaledInputData` function across multiple blockchains and sends Slack alerts when the function returns false.

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
- **Graceful shutdown**: Handles SIGINT/SIGTERM for clean shutdowns
- **Environment variable support**: Easy deployment with environment variables

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

Create a `.env` file or set these environment variables:

````bash
# Slack Configuration
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK

# Ethereum Mainnet
ETH_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY
ETH_CONTRACT_ADDRESS=0x1234567890abcdef1234567890abcdef12345678

# Polygon
POLYGON_RPC_URL=https://polygon-mainnet.g.alchemy.com/v2/YOUR_API_KEY
POLYGON_CONTRACT_ADDRESS=0x1234567890abcdef1234567890abcdef12345678

# Binance Smart Chain
BSC_RPC_URL=https://bsc-dataseed.binance.org
BSC_CONTRACT_ADDRESS=0x1234567890abcdef1234567890abcdef12345678

# Arbitrum
ARBITRUM_RPC_URL=https://arb-mainnet.g.alchemy.com/v2/YOUR_API_KEY
ARBITRUM_CONTRACT_ADDRESS=0x1234567890abcdef1234567890abcdef12345678

### Configuration File

The service reads from `config.yaml`. You can customize:

- **Monitoring interval**: How often to check contracts
- **API timeout**: Timeout for API and RPC calls
- **Test tokens**: Token pairs to monitor on each chain
- **Chain configurations**: RPC URLs and contract addresses

## Usage

### Running the Service

```bash
# Run with default configuration
go run .

# Build and run binary
go build -o scale-helper-monitor
./scale-helper-monitor
````

### Docker Support

```bash
# Build Docker image
docker build -t scale-helper-monitor .

# Run with environment variables
docker run --env-file .env scale-helper-monitor
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

When `getScaledInputData` returns `false`, the service sends detailed Slack alerts containing:

- Chain name and ID
- Token pair information (symbols and addresses)
- Transaction amounts in USD
- Input data
- Returned data from the contract
- Timestamp and error details

## Monitoring Logic

For each configured token pair, the service:

1. **Fetches route**: Calls KyberSwap API to get encoded swap data
2. **Scales amount**: Creates a new amount (110% of original) to test scaling
3. **Calls contract**: Invokes `getScaledInputData` with the encoded data and new amount
4. **Checks result**: If `isSuccess` is `false`, sends a Slack alert
5. **Logs activity**: Records all operations with structured logging
