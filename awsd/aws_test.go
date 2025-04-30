package awsd

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"

	"Savannahtakehomeassi/awsd/models"
	"Savannahtakehomeassi/configuration"
)

func TestNewAWSClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *configuration.Config
		expectError bool
	}{
		{
			name: "Valid Configuration",
			config: &configuration.Config{
				AWSRegion:    "us-west-2",
				AccessSecret: "test-secret",
				AcessKeyID:   "test-key",
			},
			expectError: false,
		},
		{
			name: "Empty Region",
			config: &configuration.Config{
				AWSRegion:    "",
				AccessSecret: "test-secret",
				AcessKeyID:   "test-key",
			},
			expectError: true,
		},
		{
			name: "Empty Credentials",
			config: &configuration.Config{
				AWSRegion:    "us-west-2",
				AccessSecret: "",
				AcessKeyID:   "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewAWSClient(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.NotNil(t, client.client)
			}
		})
	}
}

func TestGetAWSInstance(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name        string
		mockOutput  *ec2.DescribeInstancesOutput
		mockError   error
		expectError bool
		validate    func(t *testing.T, instance *models.AWSInstance)
	}{
		{
			name: "Success Case - Complete Instance",
			mockOutput: &ec2.DescribeInstancesOutput{
				Reservations: []types.Reservation{
					{
						Instances: []types.Instance{
							{
								InstanceId:       aws.String("i-1234567890abcdef0"),
								InstanceType:     types.InstanceTypeT2Micro,
								PrivateIpAddress: aws.String("10.0.0.1"),
								ImageId:          aws.String("ami-123"),
								KeyName:          aws.String("test-key"),
								PublicIpAddress:  aws.String("54.0.0.1"),
								PrivateDnsName:   aws.String("ip-10-0-0-1.ec2.internal"),
								LaunchTime:       aws.Time(now),
								Tags: []types.Tag{
									{Key: aws.String("Name"), Value: aws.String("test-instance")},
									{Key: aws.String("Environment"), Value: aws.String("prod")},
								},
								BlockDeviceMappings: []types.InstanceBlockDeviceMapping{
									{
										DeviceName: aws.String("/dev/xvda"),
										Ebs: &types.EbsInstanceBlockDevice{
											VolumeId: aws.String("vol-0abcd1234"),
										},
									},
								},
								SecurityGroups: []types.GroupIdentifier{
									{GroupId: aws.String("sg-1234")},
									{GroupId: aws.String("sg-5678")},
								},
								NetworkInterfaces: []types.InstanceNetworkInterface{
									{
										PrivateIpAddress: aws.String("10.0.0.1"),
										Association: &types.InstanceNetworkInterfaceAssociation{
											PublicIp: aws.String("54.0.0.1"),
										},
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
			validate: func(t *testing.T, instance *models.AWSInstance) {
				assert.Equal(t, "i-1234567890abcdef0", instance.InstanceID)
				assert.Equal(t, "t2.micro", instance.InstanceType)
				assert.Equal(t, "ami-123", instance.AMI)
				assert.Equal(t, "10.0.0.1", instance.PrivateIP)
				assert.Equal(t, "test-key", instance.KeyName)
				assert.Equal(t, "54.0.0.1", instance.PublicIP)
				assert.Equal(t, "ip-10-0-0-1.ec2.internal", instance.PrivateDnsName)
				assert.Equal(t, now.String(), instance.LaunchTime)
				assert.Len(t, instance.Tags, 2)
				assert.Equal(t, "test-instance", instance.Tags["Name"])
				assert.Equal(t, "prod", instance.Tags["Environment"])
				assert.Len(t, instance.BlockDeviceMappings, 1)
				assert.Len(t, instance.SecurityGroups, 2)
				assert.Len(t, instance.NetworkInterfaces, 1)
			},
		},
		{
			name:        "Empty Reservations",
			mockOutput:  &ec2.DescribeInstancesOutput{Reservations: []types.Reservation{}},
			expectError: true,
		},
		{
			name:        "Empty Instances",
			mockOutput:  &ec2.DescribeInstancesOutput{Reservations: []types.Reservation{{Instances: []types.Instance{}}}},
			expectError: true,
		},
		{
			name:        "AWS Error",
			mockError:   fmt.Errorf("some AWS error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockEC2Client{
				DescribeInstancesFunc: func(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
					return tt.mockOutput, tt.mockError
				},
			}

			client := &AWSClient{client: mockClient}
			instance, err := client.GetAWSInstance()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, instance)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, instance)
				if tt.validate != nil {
					tt.validate(t, instance)
				}
			}
		})
	}
}

func TestParseSecurityGroups(t *testing.T) {
	tests := []struct {
		name   string
		input  []types.GroupIdentifier
		output []models.SecurityGroup
	}{
		{
			name: "single group",
			input: []types.GroupIdentifier{
				{GroupId: str("sg-abc123")},
			},
			output: []models.SecurityGroup{
				{GroupId: "sg-abc123"},
			},
		},
		{
			name: "multiple groups",
			input: []types.GroupIdentifier{
				{GroupId: str("sg-abc123")},
				{GroupId: str("sg-def456")},
				{GroupId: str("sg-ghi789")},
			},
			output: []models.SecurityGroup{
				{GroupId: "sg-abc123"},
				{GroupId: "sg-def456"},
				{GroupId: "sg-ghi789"},
			},
		},
		{
			name:   "empty list",
			input:  []types.GroupIdentifier{},
			output: []models.SecurityGroup{},
		},
		{
			name: "nil group ID",
			input: []types.GroupIdentifier{
				{GroupId: nil},
			},
			output: []models.SecurityGroup{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSecurityGroups(tt.input)
			assert.Equal(t, tt.output, got)
		})
	}
}

func TestParseBlockDeviceMappings(t *testing.T) {
	tests := []struct {
		name     string
		input    []types.InstanceBlockDeviceMapping
		expected []models.BlockDeviceMapping
	}{
		{
			name: "single device",
			input: []types.InstanceBlockDeviceMapping{
				{
					DeviceName: str("/dev/sda1"),
					Ebs:        &types.EbsInstanceBlockDevice{VolumeId: str("vol-12345")},
				},
			},
			expected: []models.BlockDeviceMapping{
				{DeviceName: "/dev/sda1", VolumeId: "vol-12345"},
			},
		},
		{
			name: "multiple devices",
			input: []types.InstanceBlockDeviceMapping{
				{
					DeviceName: str("/dev/sda1"),
					Ebs:        &types.EbsInstanceBlockDevice{VolumeId: str("vol-12345")},
				},
				{
					DeviceName: str("/dev/sdb"),
					Ebs:        &types.EbsInstanceBlockDevice{VolumeId: str("vol-67890")},
				},
			},
			expected: []models.BlockDeviceMapping{
				{DeviceName: "/dev/sda1", VolumeId: "vol-12345"},
				{DeviceName: "/dev/sdb", VolumeId: "vol-67890"},
			},
		},
		{
			name:     "empty device list",
			input:    []types.InstanceBlockDeviceMapping{},
			expected: []models.BlockDeviceMapping{},
		},
		{
			name: "nil EBS volume",
			input: []types.InstanceBlockDeviceMapping{
				{
					DeviceName: str("/dev/sda1"),
					Ebs:        nil,
				},
			},
			expected: []models.BlockDeviceMapping{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseBlockDeviceMappings(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseNetworkInterfaces(t *testing.T) {
	tests := []struct {
		name     string
		input    []types.InstanceNetworkInterface
		expected []models.NetworkInterface
	}{
		{
			name: "Single interface",
			input: []types.InstanceNetworkInterface{
				{
					PrivateIpAddress: str("10.0.0.1"),
					Association: &types.InstanceNetworkInterfaceAssociation{
						PublicIp: str("54.1.2.3"),
					},
				},
			},
			expected: []models.NetworkInterface{
				{
					PrivateIpAddress: "10.0.0.1",
					PublicIpAddress:  "54.1.2.3",
				},
			},
		},
		{
			name: "Multiple interfaces",
			input: []types.InstanceNetworkInterface{
				{
					PrivateIpAddress: str("10.0.0.1"),
					Association: &types.InstanceNetworkInterfaceAssociation{
						PublicIp: str("54.1.2.3"),
					},
				},
				{
					PrivateIpAddress: str("10.0.0.2"),
					Association: &types.InstanceNetworkInterfaceAssociation{
						PublicIp: str("54.1.2.4"),
					},
				},
			},
			expected: []models.NetworkInterface{
				{
					PrivateIpAddress: "10.0.0.1",
					PublicIpAddress:  "54.1.2.3",
				},
				{
					PrivateIpAddress: "10.0.0.2",
					PublicIpAddress:  "54.1.2.4",
				},
			},
		},
		{
			name:     "Empty interfaces",
			input:    []types.InstanceNetworkInterface{},
			expected: []models.NetworkInterface{},
		},
		{
			name: "Nil association",
			input: []types.InstanceNetworkInterface{
				{
					PrivateIpAddress: str("10.0.0.1"),
					Association:      nil,
				},
			},
			expected: []models.NetworkInterface{
				{
					PrivateIpAddress: "10.0.0.1",
					PublicIpAddress:  "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNetworkInterfaces(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper to return pointer of any value (Go 1.18+)
func ptr[T any](v T) *T {
	return &v
}

func str(s string) *string {
	return &s
}
