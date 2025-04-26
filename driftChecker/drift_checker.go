package driftChecker

import (
	"Savannahtakehomeassi/logger"
	"fmt"
	"log"
	"reflect"
	"sync"

	awsm "Savannahtakehomeassi/awsd/models"
	terafm "Savannahtakehomeassi/teraform/models"
)

// CompareAWSInstanceWithTerraform compares AWS instance data with Terraform instance data
func CompareAWSInstanceWithTerraform(awsInstance *awsm.AWSInstance, tfState *terafm.TerraformState) ([]string, error) {
	driftCh := make(chan string)
	var driftDetected []string
	var wg sync.WaitGroup

	logger.Info("Drift checker started")

	tfInstance := findMatchingTFInstance(tfState)
	if tfInstance == nil {
		return nil, fmt.Errorf("no matching Terraform instance found for AWS instance %s", awsInstance.InstanceID)
	}

	// Launch comparison routines using the helper
	runComparison(&wg, driftCh, func() {
		compareBasicFields(awsInstance, tfInstance, driftCh)
	})
	runComparison(&wg, driftCh, func() {
		compareTags(awsInstance, tfInstance, driftCh)
	})
	runComparison(&wg, driftCh, func() {
		compareBlockDevices(awsInstance, tfInstance, driftCh)
	})
	runComparison(&wg, driftCh, func() {
		compareSecurityGroups(awsInstance, tfInstance, driftCh)
	})
	runComparison(&wg, driftCh, func() {
		compareNetworkInterfaces(awsInstance, tfInstance, driftCh)
	})

	go func() {
		wg.Wait()
		close(driftCh)
	}()

	for drift := range driftCh {
		driftDetected = append(driftDetected, drift)
	}

	if len(driftDetected) == 0 {
		driftDetected = append(driftDetected, "No drift detected between AWS instance and Terraform state.")
	}

	return driftDetected, nil
}

// generic function to launch goroutine safely
func runComparison(wg *sync.WaitGroup, ch chan<- string, fn func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn()
	}()
}

func findMatchingTFInstance(tfState *terafm.TerraformState) *terafm.Instance {
	for _, resource := range tfState.Resources {
		if resource.Type == "aws_instance" && len(resource.Instances) > 0 {
			return &resource.Instances[0]
		}
	}
	return nil
}

func compareBasicFields(aws *awsm.AWSInstance, tf *terafm.Instance, ch chan<- string) {
	if aws.InstanceID != tf.Attributes.InstanceID {
		ch <- fmt.Sprintf("InstanceID drift detected: AWS=%s, Terraform=%s", aws.InstanceID, tf.Attributes.InstanceID)
	}
	if aws.InstanceType != tf.Attributes.InstanceType {
		ch <- fmt.Sprintf("InstanceType drift detected: AWS=%s, Terraform=%s", aws.InstanceType, tf.Attributes.InstanceType)
	}
	if aws.PrivateIP != tf.Attributes.PrivateIP {
		ch <- fmt.Sprintf("PrivateIP drift detected: AWS=%s, Terraform=%s", aws.PrivateIP, tf.Attributes.PrivateIP)
	}
	if aws.PublicIP != tf.Attributes.PublicIP {
		ch <- fmt.Sprintf("PublicIP drift detected: AWS=%s, Terraform=%s", aws.PublicIP, tf.Attributes.PublicIP)
	}
	if aws.KeyName != tf.Attributes.KeyName {
		ch <- fmt.Sprintf("KeyName drift detected: AWS=%s, Terraform=%s", aws.KeyName, tf.Attributes.KeyName)
	}
	if aws.PrivateDnsName != tf.Attributes.PrivateDNS {
		ch <- fmt.Sprintf("PrivateDNS drift detected: AWS=%s, Terraform=%s", aws.PrivateDnsName, tf.Attributes.PrivateDNS)
	}
}

func compareTags(aws *awsm.AWSInstance, tf *terafm.Instance, ch chan<- string) {
	if !reflect.DeepEqual(aws.Tags, tf.Attributes.Tags) {
		ch <- fmt.Sprintf("Tags drift detected: AWS=%v, Terraform=%v", aws.Tags, tf.Attributes.Tags)
	}
}

func compareBlockDevices(aws *awsm.AWSInstance, tf *terafm.Instance, ch chan<- string) {
	if tf.Attributes.RootBlockDevice == nil {
		return
	}
	tfBlockMap := make(map[string]terafm.RootBlockDevice)
	for _, tfBlock := range tf.Attributes.RootBlockDevice {
		if tfBlock.DeviceName != "" {
			tfBlockMap[tfBlock.DeviceName] = tfBlock
		}
		if tfBlock.VolumeID != "" {
			tfBlockMap[tfBlock.VolumeID] = tfBlock
		}
	}
	for _, awsBlock := range aws.BlockDeviceMappings {
		if _, found := tfBlockMap[awsBlock.DeviceName]; !found {
			if _, foundVol := tfBlockMap[awsBlock.VolumeId]; !foundVol {
				ch <- fmt.Sprintf("Block device drift detected: AWS=%v, Terraform=%v", awsBlock, tfBlockMap)
			}
		}
	}
}

func compareSecurityGroups(aws *awsm.AWSInstance, tf *terafm.Instance, ch chan<- string) {
	if len(aws.SecurityGroups) != len(tf.Attributes.SecurityGroups) {
		ch <- fmt.Sprintf("Security Groups count drift detected: AWS=%d, Terraform=%d", len(aws.SecurityGroups), len(tf.Attributes.SecurityGroups))
		return
	}
	flag := false
	if tf.Attributes.SecurityGroups != nil {
		tfSGMap := make(map[string]bool)

		// Mark all SGs from Terraform state as true (exists)
		for _, sg := range tf.Attributes.SecurityGroups {
			tfSGMap[sg] = true
		}

		// Check if each AWS SG exists in Terraform
		for _, awsSG := range aws.SecurityGroups {
			if !tfSGMap[awsSG.GroupId] {
				flag = true
			}
		}

		if flag {
			ch <- fmt.Sprintf("Security Group drift detected: AWS=%s, Terraform=%v", aws.SecurityGroups, tf.Attributes.SecurityGroups)
		}

	}
}

func compareNetworkInterfaces(aws *awsm.AWSInstance, tf *terafm.Instance, ch chan<- string) {
	if len(aws.NetworkInterfaces) != 1 {
		ch <- fmt.Sprintf("Network Interface count drift detected: Expected 1, got %d", len(aws.NetworkInterfaces))
		return
	}
	iface := aws.NetworkInterfaces[0]
	if iface.PrivateIpAddress != tf.Attributes.PrivateIP {
		ch <- fmt.Sprintf("NetworkInterface PrivateIP drift detected: AWS=%s, Terraform=%s", iface.PrivateIpAddress, tf.Attributes.PrivateIP)
	}
	if iface.PublicIpAddress != tf.Attributes.PublicIP {
		ch <- fmt.Sprintf("NetworkInterface PublicIP drift detected: AWS=%s, Terraform=%s", iface.PublicIpAddress, tf.Attributes.PublicIP)
	}
}

func CompareInstances(awsInst *awsm.AWSInstance, tfInst *terafm.TFInstance) ([]string, error) {
	var drifts []string

	log.Print("Drift comparison between AWS and Terraform HCL")
	if tfInst == nil {
		return nil, fmt.Errorf("no matching Terraform instance found for AWS instance %s", awsInst.InstanceID)
	}

	if awsInst.InstanceType != tfInst.InstanceType {
		drifts = append(drifts, fmt.Sprintf("Drift in instance %s: instance_type mismatch (AWS: %s, TF: %s)", awsInst.InstanceID, awsInst.InstanceType, tfInst.InstanceType))
	}
	if awsInst.AMI != tfInst.AMI {
		drifts = append(drifts, fmt.Sprintf("Drift in instance %s: AMI mismatch (AWS: %s, TF: %s)", awsInst.InstanceID, awsInst.AMI, tfInst.AMI))
	}
	for k, v := range tfInst.Tags {
		if awsVal, ok := awsInst.Tags[k]; !ok || awsVal != v {
			drifts = append(drifts, fmt.Sprintf("Drift in instance %s: tag %s mismatch (AWS: %s, TF: %s)", awsInst.InstanceID, k, awsVal, v))
		}
	}

	if len(drifts) == 0 {
		drifts = append(drifts, "No drift detected between AWS instance and Terraform state.")
	}

	return drifts, nil
}
