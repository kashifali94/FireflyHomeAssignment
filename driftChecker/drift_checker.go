package driftChecker

import (
	"Savannahtakehomeassi/logger"
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"Savannahtakehomeassi/errors"
)

// DriftService handles drift checking operations
type DriftService struct {
	awsClient       AWSClient
	terraformClient TerraformClient
	logger          *zap.Logger
}

// NewDriftService creates a new DriftService instance
func NewDriftService(awsClient AWSClient, terraformClient TerraformClient, logger *zap.Logger) *DriftService {
	return &DriftService{
		awsClient:       awsClient,
		terraformClient: terraformClient,
		logger:          logger,
	}
}

// RunLoop runs the drift checking loop
func (s *DriftService) RunLoop(ctx context.Context, tfSpath, mainfile string, interval int) error {
	s.logger.Info("Starting drift checker loop",
		zap.String("operation", "loop_start"),
	)

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// First run immediately
	if err := s.runDriftCheck(ctx, tfSpath, mainfile); err != nil {
		return errors.New(errors.ErrDriftChecker, "Initial drift check failed",
			map[string]interface{}{
				"operation": "initial_drift_check",
			}, err)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("Drift checker shutdown complete",
				zap.String("operation", "loop_shutdown"),
			)
			return nil
		case <-ticker.C:
			if err := s.runDriftCheck(ctx, tfSpath, mainfile); err != nil {
				return errors.New(errors.ErrDriftChecker, "Periodic drift check failed",
					map[string]interface{}{
						"operation": "periodic_drift_check",
					}, err)
			}
		}
	}
}

// runDriftCheck performs a single drift check iteration
func (s *DriftService) runDriftCheck(ctx context.Context, tfPath, mainFile string) error {
	s.logger.Info("Starting drift check iteration",
		zap.String("operation", "drift_check_start"),
	)

	// Get AWS instance details
	awsInstance, err := s.awsClient.GetAWSInstance()
	if err != nil {
		s.logger.Error("Failed to get AWS instance details",
			zap.String("operation", "get_aws_instance"),
			zap.Error(errors.New(errors.ErrAWSInstance, "Failed to get AWS instance",
				map[string]interface{}{
					"operation": "get_aws_instance",
				}, err)),
		)
		return err
	}
	s.logger.Info("Successfully retrieved AWS instance details",
		zap.String("operation", "get_aws_instance"),
		zap.String("instance_id", awsInstance.InstanceID),
	)

	tfState, err := s.terraformClient.ParseTerraformInstance(tfPath)
	if err != nil {
		s.logger.Error("Failed to parse Terraform state",
			zap.String("operation", "terraform_state_parse"),
			zap.Error(errors.New(errors.ErrTerraformState, "Failed to parse Terraform state",
				map[string]interface{}{
					"operation": "terraform_state_parse",
					"path":      tfPath,
				}, err)),
		)
		return err
	}
	s.logger.Info("Successfully parsed Terraform state",
		zap.String("operation", "terraform_state_parse"),
	)

	tfConfig, err := s.terraformClient.ParseHCLConfig(mainFile)
	if err != nil {
		s.logger.Error("Failed to parse HCL config",
			zap.String("operation", "hcl_config_parse"),
			zap.Error(errors.New(errors.ErrTerraformConfig, "Failed to parse HCL config",
				map[string]interface{}{
					"operation": "hcl_config_parse",
					"path":      mainFile,
				}, err)),
		)
		return err
	}
	s.logger.Info("Successfully parsed HCL config",
		zap.String("operation", "hcl_config_parse"),
	)

	// Channels for collecting results
	type result struct {
		drift []string
		err   error
	}
	results := make(chan result, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			results <- result{nil, errors.New(errors.ErrDriftChecker, "drift check cancelled",
				map[string]interface{}{
					"operation": "drift_check",
					"context":   "cancelled",
				}, nil)}
			return
		default:
			drift, err := compareAWSInstanceWithTerraform(ctx, awsInstance, tfState)
			results <- result{drift, err}
		}
	}()

	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			results <- result{nil, errors.New(errors.ErrDriftChecker, "HCL drift check cancelled",
				map[string]interface{}{
					"operation": "hcl_drift_check",
					"context":   "cancelled",
				}, nil)}
			return
		default:
			drift, err := compareInstances(awsInstance, tfConfig)
			results <- result{drift, err}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	// Handle results safely
	for res := range results {
		if res.err != nil {
			s.logger.Error("Drift check failed",
				zap.String("operation", "drift_check"),
				zap.Error(res.err),
			)
			return res.err
		}

		if len(res.drift) == 1 && res.drift[0] == "No drift detected between AWS instance and Terraform state." {
			s.logger.Info("No drift detected between AWS and Terraform",
				zap.String("operation", "drift_check"),
				zap.String("status", "no_drift"),
			)
		} else {
			s.logger.Info("Drift detected between AWS and Terraform",
				zap.String("operation", "drift_check"),
				zap.String("status", "drift_detected"),
				zap.Strings("drifts", res.drift),
			)
		}
	}

	s.logger.Info("Drift check completed successfully",
		zap.String("operation", "drift_check_complete"),
	)
	return nil
}
