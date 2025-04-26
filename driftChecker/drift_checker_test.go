package driftChecker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	awsm "Savannahtakehomeassi/awsd/models"
	terafm "Savannahtakehomeassi/teraform/models"
)

func TestCompareAWSInstanceWithTerraform(t *testing.T) {
	tests := []struct {
		name         string
		awsInstance  *awsm.AWSInstance
		tfState      *terafm.TerraformState
		expectDrift  bool
		expectError  bool
		expectedMsgs []string
	}{
		{
			name: "No drift - all fields match",
			awsInstance: &awsm.AWSInstance{
				InstanceID:     "i-123",
				InstanceType:   "t2.micro",
				PrivateIP:      "10.0.0.1",
				PublicIP:       "3.3.3.3",
				KeyName:        "my-key",
				PrivateDnsName: "ip-10-0-0-1.ec2.internal",
				Tags: map[string]string{
					"Name": "test-instance",
				},
				BlockDeviceMappings: []awsm.BlockDeviceMapping{
					{DeviceName: "/dev/xvda", VolumeId: "vol-1"},
				},
				SecurityGroups: []awsm.SecurityGroup{
					{GroupId: "sg-123"},
				},
				NetworkInterfaces: []awsm.NetworkInterface{
					{PrivateIpAddress: "10.0.0.1", PublicIpAddress: "3.3.3.3"},
				},
			},
			tfState: &terafm.TerraformState{
				Resources: []terafm.Resource{
					{
						Type: "aws_instance",
						Instances: []terafm.Instance{
							{
								Attributes: terafm.InstanceAttributes{
									InstanceID:   "i-123",
									InstanceType: "t2.micro",
									PrivateIP:    "10.0.0.1",
									PublicIP:     "3.3.3.3",
									KeyName:      "my-key",
									PrivateDNS:   "ip-10-0-0-1.ec2.internal",
									Tags: map[string]string{
										"Name": "test-instance",
									},
									RootBlockDevice: []terafm.RootBlockDevice{
										{DeviceName: "/dev/xvda", VolumeID: "vol-1"},
									},
									SecurityGroups: []string{"sg-123"},
								},
							},
						},
					},
				},
			},
			expectDrift:  false,
			expectError:  false,
			expectedMsgs: []string{"No drift detected between AWS instance and Terraform state."},
		},
		{
			name: "Drift in InstanceType and Tags",
			awsInstance: &awsm.AWSInstance{
				InstanceID:   "i-456",
				InstanceType: "t3.small",
				Tags: map[string]string{
					"Name": "wrong-tag",
				},
				PrivateIP: "192.168.0.1",
				PublicIP:  "1.1.1.1",
				BlockDeviceMappings: []awsm.BlockDeviceMapping{
					{DeviceName: "/dev/xvda", VolumeId: "vol-2"},
				},
				SecurityGroups: []awsm.SecurityGroup{
					{GroupId: "sg-789"},
				},
				NetworkInterfaces: []awsm.NetworkInterface{
					{PrivateIpAddress: "192.168.0.1", PublicIpAddress: "1.1.1.1"},
				},
			},
			tfState: &terafm.TerraformState{
				Resources: []terafm.Resource{
					{
						Type: "aws_instance",
						Instances: []terafm.Instance{
							{
								Attributes: terafm.InstanceAttributes{
									InstanceID:   "i-456",
									InstanceType: "t2.micro",
									Tags: map[string]string{
										"Name": "correct-tag",
									},
									RootBlockDevice: []terafm.RootBlockDevice{
										{DeviceName: "/dev/xvda", VolumeID: "vol-3"},
									},
									SecurityGroups: []string{"sg-456"},
									PrivateIP:      "192.168.0.1",
									PublicIP:       "1.1.1.1",
								},
							},
						},
					},
				},
			},
			expectDrift: true,
			expectError: false,
			expectedMsgs: []string{
				"InstanceType drift detected: AWS=t3.small, Terraform=t2.micro",
				"Tags drift detected: AWS=map[Name:wrong-tag], Terraform=map[Name:correct-tag]",
				"Security Group drift detected: AWS=[{sg-789}], Terraform=[sg-456]",
			},
		},
		{
			name:         "Empty state",
			awsInstance:  &awsm.AWSInstance{InstanceID: "i-empty"},
			tfState:      &terafm.TerraformState{},
			expectDrift:  false,
			expectedMsgs: nil,
			expectError:  true,
		},
		{
			name: "No drift - all fields match",
			awsInstance: &awsm.AWSInstance{
				InstanceID:     "i-123",
				InstanceType:   "t2.micro",
				PrivateIP:      "10.0.0.1",
				PublicIP:       "3.3.3.3",
				KeyName:        "my-key",
				PrivateDnsName: "ip-10-0-0-1.ec2.internal",
				Tags: map[string]string{
					"Name": "test-instance",
				},
				BlockDeviceMappings: []awsm.BlockDeviceMapping{
					{DeviceName: "/dev/xvda", VolumeId: "vol-1"},
				},
				SecurityGroups: []awsm.SecurityGroup{
					{GroupId: "sg-123"},
				},
				NetworkInterfaces: []awsm.NetworkInterface{
					{PrivateIpAddress: "10.0.0.1", PublicIpAddress: "3.3.3.3"},
				},
			},
			tfState: &terafm.TerraformState{
				Resources: []terafm.Resource{
					{
						Type: "aws_instance",
						Instances: []terafm.Instance{
							{
								Attributes: terafm.InstanceAttributes{
									InstanceID:   "i-123",
									InstanceType: "t2.micro",
									PrivateIP:    "10.0.0.1",
									PublicIP:     "3.3.3.3",
									KeyName:      "my-key",
									PrivateDNS:   "ip-10-0-0-1.ec2.internal",
									Tags: map[string]string{
										"Name": "test-instance",
									},
									RootBlockDevice: []terafm.RootBlockDevice{
										{DeviceName: "/dev/xvda", VolumeID: "vol-1"},
									},
									SecurityGroups: []string{"sg-123"},
								},
							},
						},
					},
				},
			},
			expectDrift:  false,
			expectError:  false,
			expectedMsgs: []string{"No drift detected between AWS instance and Terraform state."},
		},
		{
			name: "AWS Instance ID does not match Terraform Instance ID",
			awsInstance: &awsm.AWSInstance{
				InstanceID:     "i-123",
				InstanceType:   "t2.micro",
				PrivateIP:      "10.0.0.1",
				PublicIP:       "3.3.3.3",
				KeyName:        "my-key",
				PrivateDnsName: "ip-10-0-0-1.ec2.internal",
				Tags: map[string]string{
					"Name": "test-instance",
				},
				BlockDeviceMappings: []awsm.BlockDeviceMapping{
					{DeviceName: "/dev/xvda", VolumeId: "vol-1"},
				},
				SecurityGroups: []awsm.SecurityGroup{
					{GroupId: "sg-123"},
				},
				NetworkInterfaces: []awsm.NetworkInterface{
					{PrivateIpAddress: "10.0.0.1", PublicIpAddress: "3.3.3.3"},
				},
			},
			tfState: &terafm.TerraformState{
				Resources: []terafm.Resource{
					{
						Type: "aws_instance",
						Instances: []terafm.Instance{
							{
								Attributes: terafm.InstanceAttributes{
									InstanceID:   "i-12311",
									InstanceType: "t2.micro",
									PrivateIP:    "10.0.0.1",
									PublicIP:     "3.3.3.3",
									KeyName:      "my-key",
									PrivateDNS:   "ip-10-0-0-1.ec2.internal",
									Tags: map[string]string{
										"Name": "test-instance",
									},
									RootBlockDevice: []terafm.RootBlockDevice{
										{DeviceName: "/dev/xvda", VolumeID: "vol-1"},
									},
									SecurityGroups: []string{"sg-123"},
								},
							},
						},
					},
				},
			},
			expectDrift: true,
			expectError: false,
			expectedMsgs: []string{
				"InstanceID drift detected: AWS=i-123, Terraform=i-12311",
			},
		},
		{
			name: "AWS BlockDeviceMapping does not match Terraform BlockDeviceMapping",
			awsInstance: &awsm.AWSInstance{
				InstanceID:     "i-123",
				InstanceType:   "t2.micro",
				PrivateIP:      "10.0.0.1",
				PublicIP:       "3.3.3.3",
				KeyName:        "my-key",
				PrivateDnsName: "ip-10-0-0-1.ec2.internal",
				Tags: map[string]string{
					"Name": "test-instance",
				},
				BlockDeviceMappings: []awsm.BlockDeviceMapping{
					{DeviceName: "/dev/xvda", VolumeId: "vol-1"},
					{DeviceName: "/dev1/xvda", VolumeId: "vol-2"},
				},
				SecurityGroups: []awsm.SecurityGroup{
					{GroupId: "sg-123"},
				},
				NetworkInterfaces: []awsm.NetworkInterface{
					{PrivateIpAddress: "10.0.0.1", PublicIpAddress: "3.3.3.3"},
				},
			},
			tfState: &terafm.TerraformState{
				Resources: []terafm.Resource{
					{
						Type: "aws_instance",
						Instances: []terafm.Instance{
							{
								Attributes: terafm.InstanceAttributes{
									InstanceID:   "i-123",
									InstanceType: "t2.micro",
									PrivateIP:    "10.0.0.1",
									PublicIP:     "3.3.3.3",
									KeyName:      "my-key",
									PrivateDNS:   "ip-10-0-0-1.ec2.internal",
									Tags: map[string]string{
										"Name": "test-instance",
									},
									RootBlockDevice: []terafm.RootBlockDevice{
										{DeviceName: "/dev/xvda", VolumeID: "vol-1"},
									},
									SecurityGroups: []string{"sg-123"},
								},
							},
						},
					},
				},
			},
			expectDrift:  false,
			expectError:  false,
			expectedMsgs: []string{"Block device drift detected: AWS={/dev1/xvda vol-2}, Terraform=map[/dev/xvda:{false /dev/xvda false vol-1 0 } vol-1:{false /dev/xvda false vol-1 0 }]"},
		},
		{
			name: "AWS KeyName does not match Terraform KeyName",
			awsInstance: &awsm.AWSInstance{
				InstanceID:     "i-123",
				InstanceType:   "t2.micro",
				PrivateIP:      "10.0.0.1",
				PublicIP:       "3.3.3.3",
				KeyName:        "my-key123",
				PrivateDnsName: "ip-10-0-0-1.ec2.internal",
				Tags: map[string]string{
					"Name": "test-instance",
				},
				BlockDeviceMappings: []awsm.BlockDeviceMapping{
					{DeviceName: "/dev/xvda", VolumeId: "vol-1"},
				},
				SecurityGroups: []awsm.SecurityGroup{
					{GroupId: "sg-123"},
				},
				NetworkInterfaces: []awsm.NetworkInterface{
					{PrivateIpAddress: "10.0.0.1", PublicIpAddress: "3.3.3.3"},
				},
			},
			tfState: &terafm.TerraformState{
				Resources: []terafm.Resource{
					{
						Type: "aws_instance",
						Instances: []terafm.Instance{
							{
								Attributes: terafm.InstanceAttributes{
									InstanceID:   "i-123",
									InstanceType: "t2.micro",
									PrivateIP:    "10.0.0.1",
									PublicIP:     "3.3.3.3",
									KeyName:      "my-key",
									PrivateDNS:   "ip-10-0-0-1.ec2.internal",
									Tags: map[string]string{
										"Name": "test-instance",
									},
									RootBlockDevice: []terafm.RootBlockDevice{
										{DeviceName: "/dev/xvda", VolumeID: "vol-1"},
									},
									SecurityGroups: []string{"sg-123"},
								},
							},
						},
					},
				},
			},
			expectDrift:  false,
			expectError:  false,
			expectedMsgs: []string{"KeyName drift detected: AWS=my-key123, Terraform=my-key"},
		},
		{
			name: "AWS PrivateDnsName does not match Terraform PrivateDnsName",
			awsInstance: &awsm.AWSInstance{
				InstanceID:     "i-123",
				InstanceType:   "t2.micro",
				PrivateIP:      "10.0.0.1",
				PublicIP:       "3.3.3.3",
				KeyName:        "my-key",
				PrivateDnsName: "ip-10-0-0-1.ec2.internal-test",
				Tags: map[string]string{
					"Name": "test-instance",
				},
				BlockDeviceMappings: []awsm.BlockDeviceMapping{
					{DeviceName: "/dev/xvda", VolumeId: "vol-1"},
				},
				SecurityGroups: []awsm.SecurityGroup{
					{GroupId: "sg-123"},
				},
				NetworkInterfaces: []awsm.NetworkInterface{
					{PrivateIpAddress: "10.0.0.1", PublicIpAddress: "3.3.3.3"},
				},
			},
			tfState: &terafm.TerraformState{
				Resources: []terafm.Resource{
					{
						Type: "aws_instance",
						Instances: []terafm.Instance{
							{
								Attributes: terafm.InstanceAttributes{
									InstanceID:   "i-123",
									InstanceType: "t2.micro",
									PrivateIP:    "10.0.0.1",
									PublicIP:     "3.3.3.3",
									KeyName:      "my-key",
									PrivateDNS:   "ip-10-0-0-1.ec2.internal",
									Tags: map[string]string{
										"Name": "test-instance",
									},
									RootBlockDevice: []terafm.RootBlockDevice{
										{DeviceName: "/dev/xvda", VolumeID: "vol-1"},
									},
									SecurityGroups: []string{"sg-123"},
								},
							},
						},
					},
				},
			},
			expectDrift:  false,
			expectError:  false,
			expectedMsgs: []string{"PrivateDNS drift detected: AWS=ip-10-0-0-1.ec2.internal-test, Terraform=ip-10-0-0-1.ec2.internal"},
		},
		{
			name: "AWS SecurityGroups count does not match Terraform count SecurityGroups",
			awsInstance: &awsm.AWSInstance{
				InstanceID:     "i-123",
				InstanceType:   "t2.micro",
				PrivateIP:      "10.0.0.1",
				PublicIP:       "3.3.3.3",
				KeyName:        "my-key",
				PrivateDnsName: "ip-10-0-0-1.ec2.internal",
				Tags: map[string]string{
					"Name": "test-instance",
				},
				BlockDeviceMappings: []awsm.BlockDeviceMapping{
					{DeviceName: "/dev/xvda", VolumeId: "vol-1"},
				},
				SecurityGroups: []awsm.SecurityGroup{
					{GroupId: "sg-123"},
				},
				NetworkInterfaces: []awsm.NetworkInterface{
					{PrivateIpAddress: "10.0.0.1", PublicIpAddress: "3.3.3.3"},
				},
			},
			tfState: &terafm.TerraformState{
				Resources: []terafm.Resource{
					{
						Type: "aws_instance",
						Instances: []terafm.Instance{
							{
								Attributes: terafm.InstanceAttributes{
									InstanceID:   "i-123",
									InstanceType: "t2.micro",
									PrivateIP:    "10.0.0.1",
									PublicIP:     "3.3.3.3",
									KeyName:      "my-key",
									PrivateDNS:   "ip-10-0-0-1.ec2.internal",
									Tags: map[string]string{
										"Name": "test-instance",
									},
									RootBlockDevice: []terafm.RootBlockDevice{
										{DeviceName: "/dev/xvda", VolumeID: "vol-1"},
									},
									SecurityGroups: []string{"sg-123", "sg-1234"},
								},
							},
						},
					},
				},
			},
			expectDrift:  false,
			expectError:  false,
			expectedMsgs: []string{"Security Groups count drift detected: AWS=1, Terraform=2"},
		},
		{
			name: "AWS SecurityGroups does not match Terraform SecurityGroups",
			awsInstance: &awsm.AWSInstance{
				InstanceID:     "i-123",
				InstanceType:   "t2.micro",
				PrivateIP:      "10.0.0.1",
				PublicIP:       "3.3.3.3",
				KeyName:        "my-key",
				PrivateDnsName: "ip-10-0-0-1.ec2.internal",
				Tags: map[string]string{
					"Name": "test-instance",
				},
				BlockDeviceMappings: []awsm.BlockDeviceMapping{
					{DeviceName: "/dev/xvda", VolumeId: "vol-1"},
				},
				SecurityGroups: []awsm.SecurityGroup{
					{GroupId: "sg-123"},
					{GroupId: "sg-12345"},
				},
				NetworkInterfaces: []awsm.NetworkInterface{
					{PrivateIpAddress: "10.0.0.1", PublicIpAddress: "3.3.3.3"},
				},
			},
			tfState: &terafm.TerraformState{
				Resources: []terafm.Resource{
					{
						Type: "aws_instance",
						Instances: []terafm.Instance{
							{
								Attributes: terafm.InstanceAttributes{
									InstanceID:   "i-123",
									InstanceType: "t2.micro",
									PrivateIP:    "10.0.0.1",
									PublicIP:     "3.3.3.3",
									KeyName:      "my-key",
									PrivateDNS:   "ip-10-0-0-1.ec2.internal",
									Tags: map[string]string{
										"Name": "test-instance",
									},
									RootBlockDevice: []terafm.RootBlockDevice{
										{DeviceName: "/dev/xvda", VolumeID: "vol-1"},
									},
									SecurityGroups: []string{"sg-123", "sg-1234"},
								},
							},
						},
					},
				},
			},
			expectDrift:  false,
			expectError:  false,
			expectedMsgs: []string{"Security Group drift detected: AWS=[{sg-123} {sg-12345}], Terraform=[sg-123 sg-1234]"},
		},
		{
			name: "AWS Private and Public IPS does not match Terraform Private and Public IPS",
			awsInstance: &awsm.AWSInstance{
				InstanceID:     "i-123",
				InstanceType:   "t2.micro",
				PrivateIP:      "10.0.0.12",
				PublicIP:       "3.3.3.4",
				KeyName:        "my-key",
				PrivateDnsName: "ip-10-0-0-1.ec2.internal",
				Tags: map[string]string{
					"Name": "test-instance",
				},
				BlockDeviceMappings: []awsm.BlockDeviceMapping{
					{DeviceName: "/dev/xvda", VolumeId: "vol-1"},
				},
				SecurityGroups: []awsm.SecurityGroup{
					{GroupId: "sg-123"},
				},
				NetworkInterfaces: []awsm.NetworkInterface{
					{PrivateIpAddress: "10.0.0.1", PublicIpAddress: "3.3.3.3"},
				},
			},
			tfState: &terafm.TerraformState{
				Resources: []terafm.Resource{
					{
						Type: "aws_instance",
						Instances: []terafm.Instance{
							{
								Attributes: terafm.InstanceAttributes{
									InstanceID:   "i-123",
									InstanceType: "t2.micro",
									PrivateIP:    "10.0.0.1",
									PublicIP:     "3.3.3.3",
									KeyName:      "my-key",
									PrivateDNS:   "ip-10-0-0-1.ec2.internal",
									Tags: map[string]string{
										"Name": "test-instance",
									},
									RootBlockDevice: []terafm.RootBlockDevice{
										{DeviceName: "/dev/xvda", VolumeID: "vol-1"},
									},
									SecurityGroups: []string{"sg-123"},
								},
							},
						},
					},
				},
			},
			expectDrift:  false,
			expectError:  false,
			expectedMsgs: []string{"PrivateIP drift detected: AWS=10.0.0.12, Terraform=10.0.0.1", "PublicIP drift detected: AWS=3.3.3.4, Terraform=3.3.3.3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs, err := CompareAWSInstanceWithTerraform(tt.awsInstance, tt.tfState)

			if tt.expectError {
				assert.Error(t, err, "Expected error, but got none")
				return
			} else {
				assert.NoError(t, err)
			}

			if tt.expectDrift {
				assert.NotEmpty(t, msgs)
				for _, expected := range tt.expectedMsgs {
					found := false
					for _, actual := range msgs {
						if actual == expected {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected drift message not found: %s", expected)
				}
			} else {
				assert.Equal(t, tt.expectedMsgs, msgs)
			}
		})
	}

}
