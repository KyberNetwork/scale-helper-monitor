# Docker Deployment Guide

This guide explains how to run the Distributor Monitor using Docker.

## Quick Start

### 1. Prepare Environment Variables

Copy the Docker environment example:
```bash
cp docker.env.example .env
```

Edit `.env` and update with your actual values:
- **SLACK_TOKEN**: Your Slack bot token (required)  
- **SLACK_CHANNEL**: Slack channel for alerts (required)
- **Network RPC URLs**: Add API keys for the networks you want to monitor

### 2. Configure Contract Addresses  

Update `config/distributor/Contracts.json` with the actual distributor contract addresses for your networks. Networks with address `0x0000000000000000000000000000000000000000` will be automatically skipped.

### 3. Build and Run

**Using Docker Compose (Recommended):**
```bash
# Build and start the monitor
docker-compose up -d

# View logs  
docker-compose logs -f distributor-monitor

# Stop the monitor
docker-compose down
```

**Using Docker directly:**
```bash
# Build the image
docker build -t distributor-monitor .

# Run the container
docker run -d \
  --name distributor-monitor \
  --env-file .env \
  -v $(pwd)/config:/app/config \
  distributor-monitor
```

## Configuration

### Network Configuration
- **Chain metadata**: `config/distributor/Chains.json` (emojis, batch sizes, block times)
- **Contract addresses**: `config/distributor/Contracts.json` (distributor contracts per network)  
- **Runtime state**: `config/distributor/State.json` (automatically managed)

### Environment Variables
Only deployment-specific values need environment variables:
- **Slack credentials**: `SLACK_TOKEN`, `SLACK_CHANNEL`
- **RPC URLs**: `{NETWORK}_NODE_URL` (e.g., `ETH_NODE_URL`, `ARBITRUM_NODE_URL`)

### Volumes
- `./config:/app/config` - Configuration files (mounted read-write)
- `distributor-state:/app/config/distributor` - State persistence (Docker volume)

## Monitoring and Logs

### View Logs
```bash
# Real-time logs
docker-compose logs -f distributor-monitor

# Last 100 lines  
docker-compose logs --tail=100 distributor-monitor
```

### Health Check
```bash
# Check container health
docker-compose ps

# Manual health check
docker exec distributor-monitor ps aux | grep distributor-monitor
```

### Container Management
```bash
# Restart the monitor
docker-compose restart distributor-monitor

# Stop and remove
docker-compose down  

# Rebuild after code changes
docker-compose up --build -d
```

## Troubleshooting

### Common Issues

1. **Configuration not found**
   - Ensure `config/` directory is properly mounted
   - Check that JSON files exist and are readable

2. **Network connection failures**  
   - Verify RPC URLs in `.env` file
   - Check API key limits and authentication

3. **Permission errors**
   - The container runs as non-root user (UID 1001)
   - Ensure config files are readable by the container

### Debug Commands
```bash
# Check configuration  
docker-compose exec distributor-monitor ls -la config/distributor/

# Validate environment
docker-compose exec distributor-monitor env | grep -E "(SLACK|NODE_URL)"

# Interactive shell
docker-compose exec distributor-monitor sh
```

## Production Deployment

### Resource Requirements
- **Memory**: 256MB minimum, 512MB recommended
- **CPU**: 0.5 cores minimum, 1 core recommended  
- **Storage**: 100MB for application, 10MB for state data

### Recommended Settings
```yaml
# docker-compose.yml additions for production
deploy:
  resources:
    limits:
      memory: 512M
      cpus: '1.0'
    reservations:
      memory: 256M  
      cpus: '0.5'
```

### Monitoring Integration
The container includes health checks and structured logging:
- **Health endpoint**: Container health status via `ps` command
- **Log format**: JSON for easy parsing by log aggregators  
- **Log rotation**: 10MB max size, 3 files retained

### Security
- Runs as non-root user (UID 1001)
- Minimal Alpine Linux base image
- No unnecessary packages or tools
- Configuration mounted read-only where possible 