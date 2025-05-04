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
	"Savannahtakehomeassi/errors"
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
		panic(errors.New(errors.ErrConfigParse, "Failed to initialize logger",
			map[string]interface{}{
				"operation": "logger_init",
			}, err))
	}
	defer logger.Sync()

	logger := zap.L().With(zap.String("package", packageName))
	logger.Info("Application starting",
		zap.String("operation", "startup"),
	)

	// Load configuration
	config, err := configuration.Initialize()
	if err != nil {
		logger.Error("Failed to load configuration",
			zap.String("operation", "config_load"),
			zap.Error(errors.New(errors.ErrConfigParse, "Configuration initialization failed",
				map[string]interface{}{
					"operation": "config_init",
				}, err)),
		)
		os.Exit(1)
	}
	logger.Info("Configuration loaded successfully",
		zap.String("operation", "config_load"),
		zap.String("tf_state_path", config.TFStatePath),
		zap.String("main_tf_path", config.MainTFPath),
		zap.Int("check_interval", config.CheckInterval),
	)

	// Create AWS client
	awsClient, err := awsd.NewAWSClient(config)
	if err != nil {
		logger.Error("Failed to create AWS client",
			zap.String("operation", "aws_client_creation"),
			zap.Error(errors.New(errors.ErrAWSClient, "AWS client creation failed",
				map[string]interface{}{
					"operation": "aws_client_init",
				}, err)),
		)
		os.Exit(1)
	}
	logger.Info("AWS client created successfully",
		zap.String("operation", "aws_client_creation"),
	)

	// Create Terraform client
	terraformClient := teraform.NewTerraformClient()
	logger.Info("Terraform client created successfully",
		zap.String("operation", "terraform_client_creation"),
	)

	// Create DriftService
	driftService := driftChecker.NewDriftService(awsClient, terraformClient, logger)
	logger.Info("DriftService created successfully",
		zap.String("operation", "drift_service_creation"),
	)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start drift checker in a goroutine
	errChan := make(chan error, 1)
	go func() {
		logger.Info("Starting drift checker service",
			zap.String("operation", "drift_service_start"),
		)
		err := driftService.RunLoop(ctx, config.TFStatePath, config.MainTFPath, config.CheckInterval)
		if err != nil {
			errChan <- errors.New(errors.ErrDriftChecker, "Drift service run loop failed",
				map[string]interface{}{
					"operation": "drift_service_run",
				}, err)
		}
	}()

	// Wait for either a signal or an error
	select {
	case sig := <-sigChan:
		logger.Info("Received signal, initiating shutdown",
			zap.String("operation", "shutdown"),
			zap.String("signal", sig.String()),
		)
		cancel()
		// Give some time for cleanup
		time.Sleep(2 * time.Second)
		logger.Info("Shutdown complete",
			zap.String("operation", "shutdown_complete"),
		)
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
