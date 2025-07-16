# Scale Helper Monitor

A comprehensive blockchain monitoring service with two specialized applications for tracking smart contracts and events across multiple networks.

## Applications

This repository contains two monitoring applications:

### 1. Scale Helper Monitor (`cmd/monitor/`)
- Monitors the `getScaledInputData` function across multiple blockchains
- Sends Slack alerts when the function returns `false`
- Uses `config.yaml` for configuration
- Uses `CONTRACT_ADDRESS` environment variable

### 2. Distributor Monitor (`cmd/distributor/`) ⭐ **Featured**
- Monitors `RootSubmitted` events on distributor contracts across 14+ networks
- Tracks campaign submissions with real-time Slack notifications
- **JSON-based configuration** for networks and contracts
- **State persistence** to resume monitoring after restarts
- **Docker support** for easy deployment

## 🚀 Quick Start (Distributor Monitor)

### Local Development
```bash
# 1. Configure environment
cp env.example .env
# Edit .env with your RPC URLs and Slack credentials

# 2. Configure contracts
# Update config/distributor/Contracts.json with actual contract addresses

# 3. Run
go run cmd/distributor/main.go
```

### Docker Deployment (Recommended)
```bash
# 1. Setup environment
cp docker.env.example .env
# Edit .env with your credentials

# 2. Deploy
docker-compose up -d

# 3. View logs
docker-compose logs -f distributor-monitor
```

## 📁 Configuration Structure

### Distributor Monitor
- **`config/distributor/Chains.json`** - Network metadata (14 networks, emojis, batch sizes)
- **`config/distributor/Contracts.json`** - Distributor contract addresses per network
- **`config/distributor/State.json`** - Runtime state persistence (auto-managed)
- **`distributor-config.yaml`** - Basic application settings

### Scale Helper Monitor
- **`config.yaml`** - Traditional YAML configuration
- Environment variables for contract addresses

## 🌐 Supported Networks (Distributor Monitor)

| Network | Chain ID | Emoji | Status |
|---------|----------|-------|--------|
| Ethereum | 1 | 🔷 | Active |
| Polygon | 137 | 🟣 | Active |
| BSC | 56 | 🟡 | Active |
| Arbitrum | 42161 | 🔵 | Active |
| Avalanche | 43114 | 🔺 | Active |
| Base | 8453 | 🟦 | Active |
| Berachain | 80085 | 🐻 | Testnet |
| Mantle | 5000 | 🧊 | Active |
| Optimism | 10 | 🔴 | Active |
| Sonic | 146 | ⚡ | Active |
| Unichain | 130 | 🦄 | Active |
| Ronin | 2020 | ⚔️ | Active |
| Linea | 59144 | 📐 | Active |
| Hyper EVM | 999 | ⚡ | Active |

## Environment Variables

### Distributor Monitor
**Deployment-specific only** (sensitive data):
- **`SLACK_TOKEN`** - Slack bot token
- **`SLACK_CHANNEL`** - Slack channel for alerts  
- **`{NETWORK}_NODE_URL`** - RPC endpoints (e.g., `ETH_NODE_URL`, `ARBITRUM_NODE_URL`)

**Configuration managed in JSON files** (no env vars needed):
- Contract addresses → `config/distributor/Contracts.json`
- Network metadata → `config/distributor/Chains.json`
- Batch sizes → `config/distributor/Chains.json`

### Scale Helper Monitor
- **`CONTRACT_ADDRESS`** - Smart contract address
- **`SLACK_WEBHOOK_URL`** - Slack webhook for alerts

## 🔧 Features

### Distributor Monitor
- **🌍 Multi-network**: 14+ blockchain networks supported
- **📦 Docker Ready**: Production-ready containerization
- **💾 State Persistence**: Resume monitoring after restarts
- **🎯 Smart Filtering**: Skip networks with invalid contracts
- **📊 Batch Processing**: Configurable batch sizes per network
- **🚨 Real-time Alerts**: Instant Slack notifications with emojis
- **🛡️ Robust Error Handling**: Comprehensive retry logic and timeouts

### Scale Helper Monitor  
- **Multi-chain support**: Monitor contracts on multiple networks
- **KyberSwap integration**: Fetches encoded swap data
- **Detailed alerts**: Comprehensive failure reporting
- **Configurable monitoring**: Adjustable intervals and parameters

## 📊 Monitoring Features (Distributor Monitor)

### Event Tracking
- **`RootSubmitted` Events**: Tracks new campaign submissions
- **Real-time Processing**: Immediate event detection and notification
- **Batch Optimization**: Network-specific batch sizes for efficiency

### Smart State Management
- **Automatic Persistence**: Saves last processed block per network
- **Crash Recovery**: Resumes from last known state after restarts
- **No Duplicate Processing**: Prevents re-processing old events

### Intelligent Network Handling
- **Dynamic Configuration**: Load networks from JSON config
- **Validation**: Skip invalid networks automatically
- **Timeout Management**: 30s connection, 15s block queries, 30s RPC operations

## 🐳 Docker Deployment

### Quick Setup
```bash
# Copy environment template
cp docker.env.example .env

# Edit .env with your:
# - Slack token and channel
# - RPC URLs for networks you want to monitor

# Start monitoring
docker-compose up -d
```

### Production Features
- **Security**: Non-root user, minimal Alpine image
- **Monitoring**: Health checks and structured logging  
- **Persistence**: State data survives container restarts
- **Resource Management**: Memory limits and log rotation

See [`Docker.md`](Docker.md) for complete deployment guide.

## 🛠️ Development

### Prerequisites
- Go 1.24+
- Access to blockchain RPC endpoints (Alchemy, Infura, etc.)
- Slack bot token and channel

### Building
```bash
# Install dependencies
go mod download

# Build distributor monitor
go build -o distributor-monitor ./cmd/distributor

# Build scale helper monitor  
go build -o scale-helper-monitor ./cmd/monitor
```

### Debugging Tools
```bash
# Check distributor configuration
go run scripts/check-distributor-config.go

# Test network connections
go run scripts/test-network-connection.go

# Clear saved state
go run scripts/clear-distributor-state.go
```

## 📈 Slack Alerts

### Distributor Monitor Alerts
Real-time notifications for new campaign submissions:
```
🎯 New Campaign Submitted on 🔵 Arbitrum

Campaign ID: 42
Transaction: 0x1234...abcd
Block: 158234567
Gas Used: 125,000
```

### Scale Helper Monitor Alerts
Batch alerts when `getScaledInputData` returns false:
- **Summary Statistics**: Success rates and affected chains
- **Detailed Failures**: Token pairs, amounts, error details
- **Tenderly Links**: Simulation results for debugging

## 🔗 Smart Contract Interfaces

### Distributor Monitor
Monitors contracts emitting:
```solidity
event RootSubmitted(uint256 indexed campaignId, bytes32 indexed root);
```

### Scale Helper Monitor
Expects contracts implementing:
```solidity
function getScaledInputData(
    bytes calldata inputData,
    uint256 newAmount
) external view returns (bool isSuccess, bytes memory data);
```

## 📖 Documentation

- **[Docker Deployment Guide](Docker.md)** - Complete Docker setup and troubleshooting
- **[Environment Examples](env.example)** - Configuration templates
- **[Script Documentation](scripts/README.md)** - Debugging and utility scripts

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test with both local and Docker setups
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License.
