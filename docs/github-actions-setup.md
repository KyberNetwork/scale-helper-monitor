# GitHub Actions Setup Guide

This guide will help you set up automated monitoring using GitHub Actions that runs every 30 minutes.

## Prerequisites

1. GitHub repository with your scale-helper-monitor code
2. Access to blockchain RPC endpoints (Infura, Alchemy, QuickNode, etc.)
3. Tenderly account and API credentials
4. Slack webhook URL for alerts
5. Deployed scale helper contracts on all target chains

## Step 1: Configure Repository Secrets

Go to your GitHub repository → Settings → Secrets and variables → Actions → Repository secrets

Add the following secrets:

### External Services

```
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK
TENDERLY_ACCESS_KEY=your_tenderly_access_key
TENDERLY_USERNAME=your_tenderly_username
TENDERLY_PROJECT=your_tenderly_project_name
```

### Blockchain RPC URLs

```
ETH_NODE_URL=https://mainnet.infura.io/v3/YOUR_KEY
POLYGON_NODE_URL=https://polygon-mainnet.infura.io/v3/YOUR_KEY
BSC_NODE_URL=https://bsc-dataseed.binance.org/
ARBITRUM_NODE_URL=https://arbitrum-mainnet.infura.io/v3/YOUR_KEY
AVALANCHE_NODE_URL=https://api.avax.network/ext/bc/C/rpc
BASE_NODE_URL=https://mainnet.base.org
BERACHAIN_NODE_URL=https://artio.rpc.berachain.com/
MANTLE_NODE_URL=https://rpc.mantle.xyz
OPTIMISM_NODE_URL=https://mainnet.optimism.io
SONIC_NODE_URL=https://rpc.sonic.chain
UNICHAIN_NODE_URL=https://sepolia.unichain.org
```

### Contract Addresses

```
ETH_CONTRACT_ADDRESS=0xYourContractAddressOnEthereum
POLYGON_CONTRACT_ADDRESS=0xYourContractAddressOnPolygon
BSC_CONTRACT_ADDRESS=0xYourContractAddressOnBSC
ARBITRUM_CONTRACT_ADDRESS=0xYourContractAddressOnArbitrum
AVALANCHE_CONTRACT_ADDRESS=0xYourContractAddressOnAvalanche
BASE_CONTRACT_ADDRESS=0xYourContractAddressOnBase
BERACHAIN_CONTRACT_ADDRESS=0xYourContractAddressOnBerachain
MANTLE_CONTRACT_ADDRESS=0xYourContractAddressOnMantle
OPTIMISM_CONTRACT_ADDRESS=0xYourContractAddressOnOptimism
SONIC_CONTRACT_ADDRESS=0xYourContractAddressOnSonic
UNICHAIN_CONTRACT_ADDRESS=0xYourContractAddressOnUnichain
```

## Step 2: Configure Your Application

Make sure your `config.yaml` has the monitoring interval set to run once:

```yaml
monitoring:
  interval: "1s" # Will run once and exit
  timeout: "30s"
```

## Step 3: Test the Workflow

### Manual Testing

1. Go to Actions tab in your GitHub repository
2. Select "Scale Helper Monitor" workflow
3. Click "Run workflow" to test manually
4. Check the logs to ensure everything works

### Automatic Schedule

The workflow will automatically run every 30 minutes starting from when you push the workflow file.

## Step 4: Monitor the Results

### View Workflow Runs

- Go to Actions tab to see all workflow runs
- Click on any run to see detailed logs
- Failed runs will upload log artifacts

### Slack Alerts

- Scale helper failures will be sent to your configured Slack channel
- Successful runs will be logged but won't send alerts

## Workflow Features

### Triggers

- **Scheduled**: Every 30 minutes via cron
- **Manual**: Can be triggered manually via workflow_dispatch
- **Push**: Runs on code changes to ensure the workflow works

### Error Handling

- 25-minute timeout to prevent overlap with next scheduled run
- Graceful timeout handling - doesn't fail if monitoring takes longer
- Log artifacts uploaded on failures for debugging

### Resource Management

- Uses Go build cache for faster builds
- Efficient Ubuntu runner
- Parallel test execution for faster monitoring cycles

## Troubleshooting

### Common Issues

**1. Secrets not working**

- Ensure secret names match exactly (case-sensitive)
- Verify all required secrets are added

**2. RPC connection failures**

- Check if RPC URLs are accessible
- Verify API keys are valid and have sufficient credits
- Some chains might need different RPC endpoints

**3. Contract call failures**

- Verify contract addresses are correct for each chain
- Ensure contracts are deployed and verified

**4. Slack notifications not working**

- Test webhook URL manually
- Check webhook permissions

### Debug Steps

1. Check the workflow logs in GitHub Actions
2. Review the uploaded log artifacts for failed runs
3. Test individual components locally
4. Verify all environment variables are set correctly

## Cost Considerations

- GitHub Actions provides 2,000 free minutes/month for public repos
- Each run takes ~2-5 minutes
- Running every 30 minutes = ~48 runs/day = ~1,440 runs/month = ~2,880-7,200 minutes/month
- Consider adjusting frequency if approaching limits

## Security Best Practices

1. **Never commit secrets to code**
2. **Use repository secrets for sensitive data**
3. **Regularly rotate API keys**
4. **Monitor secret usage in audit logs**
5. **Limit repository access to necessary personnel**

## Advanced Configuration

### Custom Schedules

Modify the cron expression in `.github/workflows/monitoring.yml`:

```yaml
schedule:
  - cron: "0 */2 * * *" # Every 2 hours
  - cron: "0 9,17 * * 1-5" # 9 AM and 5 PM on weekdays
```

### Environment-Specific Configs

Create separate workflows for different environments (staging, production) with different secrets and configurations.

### Notifications

Add more notification channels by modifying the monitoring code:

- Discord webhooks
- Email notifications
- PagerDuty alerts
- Custom webhook endpoints
