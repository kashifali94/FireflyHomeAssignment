package driftChecker

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"sync"

	awsm "Savannahtakehomeassi/awsd/models"
	terafm "Savannahtakehomeassi/teraform/models"
)

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
