package utils

import (
	"fmt"
	"reflect"

	awsm "Savannahtakehomeassi/awsd/models"
	terafm "Savannahtakehomeassi/teraform/models"
)

// CompareAWSInstanceWithTerraform compares AWS instance data with Terraform instance data
func CompareAWSInstanceWithTerraform(awsInstance *awsm.AWSInstance, tfState *terafm.TerraformState) ([]string, error) {
	var driftDetected []string
	// Find the instance corresponding to the given AWSInstance in the Terraform state
	var tfInstance *terafm.Instance
	for _, resource := range tfState.Resources {
		if resource.Type == "aws_instance" { // Assuming the type is "aws_instance"
			tfInstance = &resource.Instances[0]
		}
	}

	// If no matching instance found in Terraform state
	if tfInstance == nil {
		return nil, fmt.Errorf("no matching Terraform instance found for AWS instance %s", awsInstance.InstanceID)
	}

	// Compare basic fields and add drift messages when mismatches occur
	if awsInstance.InstanceID != tfInstance.Attributes.InstanceID {
		driftDetected = append(driftDetected, fmt.Sprintf("InstanceID drift detected: AWS=%s, Terraform=%s", awsInstance.InstanceID, tfInstance.Attributes.InstanceID))
	}
	if awsInstance.InstanceType != tfInstance.Attributes.InstanceType {
		driftDetected = append(driftDetected, fmt.Sprintf("InstanceType drift detected: AWS=%s, Terraform=%s", awsInstance.InstanceType, tfInstance.Attributes.InstanceType))
	}
	if awsInstance.PrivateIP != tfInstance.Attributes.PrivateIP {
		driftDetected = append(driftDetected, fmt.Sprintf("PrivateIP drift detected: AWS=%s, Terraform=%s", awsInstance.PrivateIP, tfInstance.Attributes.PrivateIP))
	}
	if awsInstance.PublicIP != tfInstance.Attributes.PublicIP {
		driftDetected = append(driftDetected, fmt.Sprintf("PublicIP drift detected: AWS=%s, Terraform=%s", awsInstance.PublicIP, tfInstance.Attributes.PublicIP))
	}
	if awsInstance.KeyName != tfInstance.Attributes.KeyName {
		driftDetected = append(driftDetected, fmt.Sprintf("KeyName drift detected: AWS=%s, Terraform=%s", awsInstance.KeyName, tfInstance.Attributes.KeyName))
	}
	if awsInstance.PrivateDnsName != tfInstance.Attributes.PrivateDNS {
		driftDetected = append(driftDetected, fmt.Sprintf("PrivateDnsName drift detected: AWS=%s, Terraform=%s", awsInstance.PrivateDnsName, tfInstance.Attributes.PrivateDNS))
	}

	// Compare Tags (assuming both are in map format)
	if !reflect.DeepEqual(awsInstance.Tags, tfInstance.Attributes.Tags) {
		driftDetected = append(driftDetected, fmt.Sprintf("Tags drift detected: AWS=%v, Terraform=%v", awsInstance.Tags, tfInstance.Attributes.Tags))
	}

	// Compare Block Device Mappings (Root Block Device)
	//(TODO):Create a helper function for this
	if tfInstance.Attributes.RootBlockDevice != nil {
		// Build a map from Terraform block devices using DeviceName and VolumeID as keys
		tfBlockMap := make(map[string]terafm.RootBlockDevice)
		for _, tfBlock := range tfInstance.Attributes.RootBlockDevice {
			if tfBlock.DeviceName != "" {
				tfBlockMap[tfBlock.DeviceName] = tfBlock
			}
			if tfBlock.VolumeID != "" {
				tfBlockMap[tfBlock.VolumeID] = tfBlock
			}
		}

		for _, awsBlock := range awsInstance.BlockDeviceMappings {
			// Try to find a matching TF block using DeviceName or VolumeId
			_, matchByDeviceName := tfBlockMap[awsBlock.DeviceName]
			_, matchByVolumeID := tfBlockMap[awsBlock.VolumeId]

			if !matchByDeviceName && !matchByVolumeID {
				driftDetected = append(driftDetected,
					fmt.Sprintf("Block Device Mapping drift detected: AWS=%v, Terraform=%v", awsBlock, tfBlockMap))
			}
		}
	}

	// Compare Security Groups
	if len(awsInstance.SecurityGroups) != len(tfInstance.Attributes.SecurityGroups) {
		driftDetected = append(driftDetected, fmt.Sprintf("Security Groups count drift detected: AWS=%d, Terraform=%d", len(awsInstance.SecurityGroups), len(tfInstance.Attributes.SecurityGroups)))
	}

	//(TODO): create a function for this  as well
	flag := false
	if tfInstance.Attributes.SecurityGroups != nil {
		tfSGMap := make(map[string]bool)

		// Mark all SGs from Terraform state as true (exists)
		for _, sg := range tfInstance.Attributes.SecurityGroups {
			tfSGMap[sg] = true
		}

		// Check if each AWS SG exists in Terraform
		for _, awsSG := range awsInstance.SecurityGroups {
			if !tfSGMap[awsSG.GroupId] {
				flag = true
			}
		}

		if flag {
			driftDetected = append(driftDetected,
				fmt.Sprintf("Security Group drift detected: AWS=%s, Terraform=%v", awsInstance.SecurityGroups, tfInstance.Attributes.SecurityGroups))
		}

	}

	// Compare Network Interfaces
	if len(awsInstance.NetworkInterfaces) != 1 {
		driftDetected = append(driftDetected, fmt.Sprintf("Network Interface drift detected: Expected 1 interface, but got %d from AWS instance", len(awsInstance.NetworkInterfaces)))
	} else {
		awsNetInterface := awsInstance.NetworkInterfaces[0]
		if awsNetInterface.PrivateIpAddress != tfInstance.Attributes.PrivateIP {
			driftDetected = append(driftDetected, fmt.Sprintf("Private IP in NetworkInterface drift detected: AWS=%s, Terraform=%s", awsNetInterface.PrivateIpAddress, tfInstance.Attributes.PrivateIP))
		}
		if awsNetInterface.PublicIpAddress != tfInstance.Attributes.PublicIP {
			driftDetected = append(driftDetected, fmt.Sprintf("Public IP in NetworkInterface drift detected: AWS=%s, Terraform=%s", awsNetInterface.PublicIpAddress, tfInstance.Attributes.PublicIP))
		}
	}

	// If no drift detected, return a message indicating everything is in sync

	if len(driftDetected) == 0 {
		driftDetected = append(driftDetected, "No drift detected between AWS instance and Terraform state.")
	}

	// Return drift detection messages
	return driftDetected, nil
}
