package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	"scale-helper-monitor/internal/config"
	"scale-helper-monitor/internal/monitor"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// Don't fail if .env doesn't exist, just log a warning
		logrus.WithError(err).Warn("No .env file found, using environment variables only")
	} else {
		logrus.Info("Successfully loaded .env file")
	}

	var runOnce bool
	// Check environment variable as alternative to flag
	if os.Getenv("RUN_ONCE") == "true" {
		runOnce = true
	}

	// Setup logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logger.SetLevel(logrus.InfoLevel)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	if runOnce {
		logger.Info("Starting Scale Helper Monitor (one-shot mode)")
	} else {
		logger.Info("Starting Scale Helper Monitor (continuous mode)")
	}

	// Parse timeout for clients
	timeout, err := time.ParseDuration(cfg.Monitoring.Timeout)
	if err != nil {
		logger.WithError(err).Fatal("Invalid timeout duration")
	}

	// Create clients
	kyberClient := cfg.GetKyberSwapClient(timeout, logger)
	slackClient := cfg.GetSlackClient(timeout, logger)
	tenderlyClient := cfg.GetTenderlyClient(timeout)

	// Create monitor
	monitorService, err := monitor.NewMonitor(
		&cfg.Monitoring,
		cfg.TestCases,
		cfg.Tokens,
		cfg.OnlyScaleDownDexs,
		cfg.Sources,
		cfg.Chains,
		kyberClient,
		slackClient,
		tenderlyClient,
		logger,
	)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create monitor")
	}
	defer monitorService.Close()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if runOnce {
		// One-shot mode: run monitoring once and exit
		logger.Info("Running monitoring once...")
		err := monitorService.RunMonitoringOnce(ctx)
		if err != nil {
			logger.WithError(err).Error("Monitoring failed")
			os.Exit(1)
		}
		logger.Info("Monitoring completed successfully")
		return
	}

	// Continuous mode: original behavior with signals and loops
	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start monitoring in a goroutine
	monitorDone := make(chan error, 1)
	go func() {
		monitorDone <- monitorService.RunMonitoring(ctx)
	}()

	// Wait for shutdown signal or monitoring error
	select {
	case sig := <-sigChan:
		logger.WithField("signal", sig).Info("Received shutdown signal")
		cancel()

		// Wait for monitoring to stop
		err := <-monitorDone
		if err != nil && err != context.Canceled {
			logger.WithError(err).Error("Monitoring stopped with error")
		}

	case err := <-monitorDone:
		if err != nil {
			logger.WithError(err).Error("Monitoring stopped with error")
			os.Exit(1)
		}
	}

	logger.Info("Scale Helper Monitor stopped")
}
