package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Savannahtakehomeassi/awsd"
	"Savannahtakehomeassi/configuration"
	"Savannahtakehomeassi/driftChecker"
	"Savannahtakehomeassi/logger"
	"Savannahtakehomeassi/teraform"

	"go.uber.org/zap"
)

const (
	packageName = "main"
)

func main() {
	// Initialize logger
	if err := logger.Initialize("info"); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger := zap.L().With(zap.String("package", packageName))

	// Load configuration
	config, err := configuration.Initialize()
	if err != nil {
		logger.Error("Failed to load configuration",
			zap.String("operation", "config_load"),
			zap.Error(err),
		)
		os.Exit(1)
	}

	// Create AWS client
	awsClient, err := awsd.NewAWSClient(config)
	if awsClient == nil {
		logger.Error("Failed to create AWS client",
			zap.String("operation", "aws_client_creation"),
		)
		os.Exit(1)
	}

	// Create Terraform client
	terraformClient := teraform.NewTerraformClient()

	// Create DriftService
	driftService := driftChecker.NewDriftService(awsClient, terraformClient, logger)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start drift checker in a goroutine
	errChan := make(chan error, 1)
	go func() {
		err := driftService.RunLoop(ctx, config.TFStatePath, config.MainTFPath, config.CheckInterval)
		if err != nil {
			errChan <- err
		}
	}()

	// Wait for either a signal or an error
	select {
	case sig := <-sigChan:
		logger.Info("Received signal, shutting down",
			zap.String("operation", "shutdown"),
			zap.String("signal", sig.String()),
		)
		cancel()
		// Give some time for cleanup
		time.Sleep(2 * time.Second)
	case err := <-errChan:
		if err != nil {
			logger.Error("Drift checker error",
				zap.String("operation", "drift_check"),
				zap.Error(err),
			)
			os.Exit(1)
		}
	}
}
