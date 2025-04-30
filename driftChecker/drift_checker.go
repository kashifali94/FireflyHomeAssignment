package driftChecker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	awsm "Savannahtakehomeassi/awsd/models"
	"Savannahtakehomeassi/errors"
	terafm "Savannahtakehomeassi/teraform/models"
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
	logger := s.logger.With(
		zap.String("package", "driftChecker"),
		zap.String("function", "RunLoop"),
		zap.Int("interval_seconds", interval),
		zap.String("tf_state_path", tfSpath),
		zap.String("main_tf_path", mainfile),
	)

	logger.Info("Starting drift checker loop",
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
	logger := s.logger.With(
		zap.String("package", "driftChecker"),
		zap.String("function", "runDriftCheck"),
		zap.String("tf_state_path", tfPath),
		zap.String("main_tf_path", mainFile),
	)

	logger.Info("Starting drift check iteration",
		zap.String("operation", "drift_check_start"),
	)

	// Get AWS instance details
	awsInstance, err := s.awsClient.GetAWSInstance()
	if err != nil {
		logger.Error("Failed to get AWS instance details",
			zap.String("operation", "get_aws_instance"),
			zap.Error(errors.New(errors.ErrAWSInstance, "Failed to get AWS instance",
				map[string]interface{}{
					"operation": "get_aws_instance",
				}, err)),
		)
		return err
	}
	logger.Info("Successfully retrieved AWS instance details",
		zap.String("operation", "get_aws_instance"),
		zap.String("instance_id", awsInstance.InstanceID),
	)

	tfState, err := s.terraformClient.ParseTerraformInstance(tfPath)
	if err != nil {
		logger.Error("Failed to parse Terraform state",
			zap.String("operation", "terraform_state_parse"),
			zap.Error(errors.New(errors.ErrTerraformState, "Failed to parse Terraform state",
				map[string]interface{}{
					"operation": "terraform_state_parse",
					"path":      tfPath,
				}, err)),
		)
		return err
	}
	logger.Info("Successfully parsed Terraform state",
		zap.String("operation", "terraform_state_parse"),
	)

	tfConfig, err := s.terraformClient.ParseHCLConfig(mainFile)
	if err != nil {
		logger.Error("Failed to parse HCL config",
			zap.String("operation", "hcl_config_parse"),
			zap.Error(errors.New(errors.ErrTerraformConfig, "Failed to parse HCL config",
				map[string]interface{}{
					"operation": "hcl_config_parse",
					"path":      mainFile,
				}, err)),
		)
		return err
	}
	logger.Info("Successfully parsed HCL config",
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
			logger.Error("Drift check failed",
				zap.String("operation", "drift_check"),
				zap.Error(res.err),
			)
			return res.err
		}

		if len(res.drift) == 1 && res.drift[0] == "No drift detected between AWS instance and Terraform state." {
			logger.Info("No drift detected between AWS and Terraform",
				zap.String("operation", "drift_check"),
				zap.String("status", "no_drift"),
			)
		} else {
			logger.Info("Drift detected between AWS and Terraform",
				zap.String("operation", "drift_check"),
				zap.String("status", "drift_detected"),
				zap.Strings("drifts", res.drift),
			)
		}
	}

	logger.Info("Drift check completed successfully",
		zap.String("operation", "drift_check_complete"),
	)
	return nil
}

func compareAWSInstanceWithTerraform(ctx context.Context, awsInstance *awsm.AWSInstance, tfState *terafm.TerraformState) ([]string, error) {
	logger := zap.L().With(
		zap.String("function", "compareAWSInstanceWithTerraform"),
		zap.String("instance_id", awsInstance.InstanceID),
	)

	logger.Info("Starting AWS-Terraform comparison",
		zap.String("operation", "comparison_start"),
	)

	driftCh := make(chan string)
	var driftDetected []string
	var wg sync.WaitGroup

	tfInstance := findMatchingTFInstance(tfState)
	if tfInstance == nil {
		err := fmt.Errorf("no matching Terraform instance found for AWS instance %s", awsInstance.InstanceID)
		logger.Error("Failed to find matching Terraform instance",
			zap.String("operation", "instance_match"),
			zap.Error(err),
		)
		return nil, err
	}

	// Run comparisons
	run := func(f func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f()
		}()
	}

	run(func() { compareBasicFields(ctx, awsInstance, tfInstance, driftCh) })
	run(func() { compareTags(ctx, awsInstance, tfInstance, driftCh) })
	run(func() { compareBlockDevices(ctx, awsInstance, tfInstance, driftCh) })
	run(func() { compareSecurityGroups(ctx, awsInstance, tfInstance, driftCh) })
	run(func() { compareNetworkInterfaces(ctx, awsInstance, tfInstance, driftCh) })

	// Close channel after all goroutines complete
	go func() {
		wg.Wait()
		close(driftCh)
	}()

	// Collect drifts
	for {
		select {
		case <-ctx.Done():
			logger.Info("Comparison cancelled",
				zap.String("operation", "comparison_cancelled"),
			)
			return nil, ctx.Err()
		case drift, ok := <-driftCh:
			if !ok {
				if len(driftDetected) == 0 {
					driftDetected = append(driftDetected, "No drift detected between AWS instance and Terraform state.")
				}
				logger.Info("Comparison completed",
					zap.String("operation", "comparison_complete"),
					zap.Int("drift_count", len(driftDetected)),
				)
				return driftDetected, nil
			}
			driftDetected = append(driftDetected, drift)
		}
	}
}

// compareInstances for aws and tfInstance for hcl
func compareInstances(awsInst *awsm.AWSInstance, tfInst *terafm.TFInstance) ([]string, error) {
	logger := zap.L().With(
		zap.String("function", "compareInstances"),
		zap.String("instance_id", awsInst.InstanceID),
	)

	var drifts []string

	logger.Info("Starting HCL comparison",
		zap.String("operation", "hcl_comparison_start"),
	)

	if tfInst == nil {
		err := fmt.Errorf("no matching Terraform instance found for AWS instance %s", awsInst.InstanceID)
		logger.Error("Failed to find matching Terraform instance",
			zap.String("operation", "instance_match"),
			zap.Error(err),
		)
		return nil, err
	}

	if awsInst.InstanceType != tfInst.InstanceType {
		drifts = append(drifts, fmt.Sprintf("Drift in instance %s: instance_type mismatch (AWS: %s, TF: %s)", awsInst.InstanceID, awsInst.InstanceType, tfInst.InstanceType))
		logger.Info("Instance type drift detected",
			zap.String("operation", "hcl_comparison"),
			zap.String("aws_type", awsInst.InstanceType),
			zap.String("tf_type", tfInst.InstanceType),
		)
	}
	if awsInst.AMI != tfInst.AMI {
		drifts = append(drifts, fmt.Sprintf("Drift in instance %s: AMI mismatch (AWS: %s, TF: %s)", awsInst.InstanceID, awsInst.AMI, tfInst.AMI))
		logger.Info("AMI drift detected",
			zap.String("operation", "hcl_comparison"),
			zap.String("aws_ami", awsInst.AMI),
			zap.String("tf_ami", tfInst.AMI),
		)
	}
	for k, v := range tfInst.Tags {
		if awsVal, ok := awsInst.Tags[k]; !ok || awsVal != v {
			drifts = append(drifts, fmt.Sprintf("Drift in instance %s: tag %s mismatch (AWS: %s, TF: %s)", awsInst.InstanceID, k, awsVal, v))
			logger.Info("Tag drift detected",
				zap.String("operation", "hcl_comparison"),
				zap.String("tag_key", k),
				zap.String("aws_value", awsVal),
				zap.String("tf_value", v),
			)
		}
	}

	if len(drifts) == 0 {
		drifts = append(drifts, "No drift detected between AWS instance and Terraform state.")
		logger.Info("No drift detected in HCL comparison",
			zap.String("operation", "hcl_comparison_complete"),
			zap.String("status", "no_drift"),
		)
	} else {
		logger.Info("Drift detected in HCL comparison",
			zap.String("operation", "hcl_comparison_complete"),
			zap.String("status", "drift_detected"),
			zap.Int("drift_count", len(drifts)),
		)
	}

	return drifts, nil
}

func findMatchingTFInstance(tfState *terafm.TerraformState) *terafm.Instance {
	logger := zap.L().With(
		zap.String("function", "findMatchingTFInstance"),
	)

	for _, resource := range tfState.Resources {
		if resource.Type == "aws_instance" && len(resource.Instances) > 0 {
			logger.Info("Found matching Terraform instance",
				zap.String("operation", "instance_match"),
				zap.String("resource_type", resource.Type),
			)
			return &resource.Instances[0]
		}
	}
	return nil
}

func compareTags(ctx context.Context, aws *awsm.AWSInstance, tf *terafm.Instance, ch chan<- string) {
	for k, v := range tf.Attributes.Tags {
		if awsVal, ok := aws.Tags[k]; !ok || awsVal != v {
			ch <- fmt.Sprintf("Tag drift detected: %s (AWS: %s, TF: %s)", k, awsVal, v)
		}
	}
}

func compareBlockDevices(ctx context.Context, aws *awsm.AWSInstance, tf *terafm.Instance, ch chan<- string) {
	// Compare root block device
	if len(tf.Attributes.RootBlockDevice) > 0 {
		tfRoot := tf.Attributes.RootBlockDevice[0]
		for _, awsDevice := range aws.BlockDeviceMappings {
			if awsDevice.DeviceName == tfRoot.DeviceName {
				if awsDevice.VolumeId != tfRoot.VolumeID {
					ch <- fmt.Sprintf("Root block device volume ID drift detected: AWS=%s, TF=%s", awsDevice.VolumeId, tfRoot.VolumeID)
				}
				break
			}
		}
	}
}

func compareSecurityGroups(ctx context.Context, aws *awsm.AWSInstance, tf *terafm.Instance, ch chan<- string) {
	// Compare security groups
	awsSGs := make(map[string]bool)
	for _, sg := range aws.SecurityGroups {
		awsSGs[sg.GroupId] = true
	}

	for _, tfSG := range tf.Attributes.VpcSecurityGroupIDs {
		if !awsSGs[tfSG] {
			ch <- fmt.Sprintf("Security group drift detected: TF security group %s not found in AWS", tfSG)
		}
	}
}

func compareNetworkInterfaces(ctx context.Context, aws *awsm.AWSInstance, tf *terafm.Instance, ch chan<- string) {
	// Compare network interfaces
	if len(aws.NetworkInterfaces) > 0 {
		awsPrimary := aws.NetworkInterfaces[0]
		if awsPrimary.PrivateIpAddress != tf.Attributes.PrivateIP {
			ch <- fmt.Sprintf("Private IP drift detected: AWS=%s, TF=%s", awsPrimary.PrivateIpAddress, tf.Attributes.PrivateIP)
		}
		if awsPrimary.PublicIpAddress != tf.Attributes.PublicIP {
			ch <- fmt.Sprintf("Public IP drift detected: AWS=%s, TF=%s", awsPrimary.PublicIpAddress, tf.Attributes.PublicIP)
		}
	}
}

func compareBasicFields(ctx context.Context, aws *awsm.AWSInstance, tf *terafm.Instance, ch chan<- string) {
	if aws.InstanceType != tf.Attributes.InstanceType {
		ch <- fmt.Sprintf("InstanceType drift detected: AWS=%s, Terraform=%s", aws.InstanceType, tf.Attributes.InstanceType)
	}
	if aws.AMI != tf.Attributes.AMI {
		ch <- fmt.Sprintf("AMI drift detected: AWS=%s, Terraform=%s", aws.AMI, tf.Attributes.AMI)
	}
}
