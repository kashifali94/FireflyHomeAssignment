package main

import (
	awsm "Savannahtakehomeassi/awsd/models"
	"context"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"Savannahtakehomeassi/awsd"
	"Savannahtakehomeassi/configuration"
	"Savannahtakehomeassi/driftChecker"
	"Savannahtakehomeassi/logger"
	"Savannahtakehomeassi/teraform"
)

type DriftService struct {
	awsClient *awsd.AwsClient
	config    *configuration.Config
}

func NewDriftService(config *configuration.Config) (*DriftService, error) {
	awsClient, err := awsd.NewEC2Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS client: %w", err)
	}

	return &DriftService{
		awsClient: awsClient,
		config:    config,
	}, nil
}

func (s *DriftService) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.config.CheckInterval)

	defer ticker.Stop()

	// First run immediately
	if err := s.runDriftCheck(); err != nil {
		logger.Error("Error in initial drift check", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("Drift checker shutdown complete")
			return nil
		case <-ticker.C:
			if err := s.runDriftCheck(); err != nil {
				logger.Error("Error in drift check", zap.Error(err))
			}
		}
	}
}

func (s *DriftService) runDriftCheck() error {
	logger.Info("Starting drift check")

	// Fetch AWS instance with retry
	var awsInst *awsm.AWSInstance
	var err error
	for i := 0; i < s.config.MaxRetries; i++ {
		awsInst, err = awsd.GetAWSInstance(s.awsClient)
		if err == nil {
			break
		}
		if i < s.config.MaxRetries-1 {
			logger.Warn("Failed to fetch AWS instance, retrying",
				zap.Int("attempt", i+1),
				zap.Int("max_attempts", s.config.MaxRetries),
				zap.Error(err))
			time.Sleep(s.config.RetryDelay)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to fetch AWS instance after %d attempts: %w", s.config.MaxRetries, err)
	}

	// Parse Terraform state
	tfInst, err := teraform.ParseTerraformInstance(s.config.TFStatePath)
	if err != nil {
		return fmt.Errorf("failed to parse Terraform state: %w", err)
	}

	// Parse HCL config
	tfinstance, err := teraform.ParseHCLConfig(s.config.MainTFPath)
	if err != nil {
		return fmt.Errorf("failed to parse HCL config: %w", err)
	}

	// Create channels for drift comparison results and errors
	driftChannel := make(chan []string, 2)
	errorChannel := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	// Drift check with Terraform state comparison
	go func() {
		defer wg.Done()
		drift, err := driftChecker.CompareAWSInstanceWithTerraform(awsInst, tfInst)
		if err != nil {
			errorChannel <- err
			return
		}
		driftChannel <- drift
	}()

	// Drift check with Terraform HCL config comparison
	go func() {
		defer wg.Done()
		hclDrift, err := driftChecker.CompareInstances(awsInst, tfinstance)
		if err != nil {
			errorChannel <- err
			return
		}
		driftChannel <- hclDrift
	}()

	// Wait for all drift checks to complete
	wg.Wait()
	close(driftChannel)
	close(errorChannel)

	// Handle errors from drift checks
	for err := range errorChannel {
		logger.Error("Error during drift check", zap.Error(err))
	}

	// Collect and print drift results from both channels
	driftResults := make([][]string, 0, 2)
	for drift := range driftChannel {
		driftResults = append(driftResults, drift)
	}

	// Print results for both drift checks
	for _, drift := range driftResults {
		if len(drift) == 1 && drift[0] == "No drift detected between AWS instance and Terraform state." {
			logger.Info("No drift detected between AWS and Terraform")
		} else {
			logger.Info("Drift detected between AWS and Terraform")
			for i, d := range drift {
				logger.Info(fmt.Sprintf("  %d. %s", i+1, d))
			}
		}
	}

	logger.Info("Drift check completed")
	return nil
}

func main() {
	// Initialize configuration
	config, err := configuration.Initialize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Initialize(config.LogLevel); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create drift service
	service, err := NewDriftService(config)
	if err != nil {
		logger.Fatal("Failed to create drift service", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for SIGINT/SIGTERM
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		logger.Info("Received shutdown signal. Stopping drift checker...")
		cancel()
	}()

	logger.Info("Starting drift checker",
		zap.Duration("interval", config.CheckInterval),
		zap.String("tf_state_path", config.TFStatePath),
		zap.String("main_tf_path", config.MainTFPath))

	if err := service.Start(ctx); err != nil {
		logger.Error("Drift service stopped with error", zap.Error(err))
	}
}
