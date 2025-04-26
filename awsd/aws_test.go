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
)

func TestGetAWSInstance(t *testing.T) {
	tests := []struct {
		name        string
		mockOutput  *ec2.DescribeInstancesOutput
		mockError   error
		expectError bool
	}{
		{
			name: "Success Case",
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
								LaunchTime:       aws.Time(time.Now()),
								Tags: []types.Tag{
									{Key: aws.String("Name"), Value: aws.String("test-instance")},
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
		},
		{
			name:        "Empty Reservations",
			mockOutput:  &ec2.DescribeInstancesOutput{Reservations: []types.Reservation{}},
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

			client := &AwsClient{client: mockClient}
			instance, err := GetAWSInstance(client)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError && instance == nil {
				t.Errorf("expected instance but got nil")
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
			name:   "empty list",
			input:  []types.GroupIdentifier{},
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
			name:     "empty device list",
			input:    []types.InstanceBlockDeviceMapping{},
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
			name:     "Empty interfaces",
			input:    []types.InstanceNetworkInterface{},
			expected: []models.NetworkInterface{},
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
