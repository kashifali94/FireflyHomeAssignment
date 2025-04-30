package awsd

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"Savannahtakehomeassi/awsd/models"
	"Savannahtakehomeassi/configuration"
	"Savannahtakehomeassi/errors"
)

const (
	packageName = "awsd"
)

type AWSClient struct {
	client EC2API
}

// NewAWSClient creates a new AWS client
func NewAWSClient(conf *configuration.Config) (*AWSClient, error) {
	logger := zap.L().With(
		zap.String("package", packageName),
		zap.String("function", "NewAWSClient"),
	)

	// Validate configuration
	if conf == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	if conf.AWSRegion == "" {
		return nil, fmt.Errorf("AWS region cannot be empty")
	}

	if conf.AccessSecret == "" || conf.AcessKeyID == "" {
		return nil, fmt.Errorf("AWS credentials cannot be empty")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(conf.AWSRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(conf.AccessSecret, conf.AcessKeyID, "")),
		config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{URL: viper.GetString("LOCALSTACK_URL"), SigningRegion: region}, nil
			}),
		),
	)
	if err != nil {
		logger.Error("Failed to create AWS client",
			zap.String("operation", "client_creation"),
			zap.Error(err),
		)
		return nil, err
	}

	logger.Info("AWS client created successfully")
	return &AWSClient{
		client: ec2.NewFromConfig(cfg),
	}, nil
}

// GetAWSInstance fetches AWS EC2 instance details
func (c *AWSClient) GetAWSInstance() (*models.AWSInstance, error) {
	logger := zap.L().With(
		zap.String("package", packageName),
		zap.String("function", "GetAWSInstance"),
	)

	// Describe the instance
	output, err := c.client.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{},
	})

	if err != nil {
		return nil, errors.New(errors.ErrAWSInstance, "failed to describe instances",
			map[string]interface{}{
				"operation": "describe_instances",
			}, err)
	}

	// Check if the instance exists in the response
	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		logger.Error("No instances found")
		return nil, errors.New(errors.ErrAWSInstance, "no instances found",
			map[string]interface{}{
				"operation": "instance_lookup",
			}, nil)
	}

	// Extract instance details
	i := output.Reservations[0].Instances[0]
	logger.Info("Instance found", zap.String("instance_id", *i.InstanceId))

	// Map tags
	tags := make(map[string]string)
	for _, tag := range i.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	// Return the AWSInstance model with relevant details
	awsInstance := &models.AWSInstance{
		InstanceID:          *i.InstanceId,
		InstanceType:        string(i.InstanceType),
		AMI:                 *i.ImageId,
		PrivateIP:           aws.ToString(i.PrivateIpAddress),
		KeyName:             aws.ToString(i.KeyName),
		Tags:                tags,
		PublicIP:            aws.ToString(i.PublicIpAddress),
		LaunchTime:          i.LaunchTime.String(),
		PrivateDnsName:      aws.ToString(i.PrivateDnsName),
		BlockDeviceMappings: parseBlockDeviceMappings(i.BlockDeviceMappings),
		SecurityGroups:      parseSecurityGroups(i.SecurityGroups),
		NetworkInterfaces:   parseNetworkInterfaces(i.NetworkInterfaces),
	}

	logger.Info("AWS instance details parsed successfully",
		zap.String("instance_id", awsInstance.InstanceID),
		zap.String("instance_type", awsInstance.InstanceType),
	)
	return awsInstance, nil
}

// Helper function to parse block device mappings
func parseBlockDeviceMappings(mappings []types.InstanceBlockDeviceMapping) []models.BlockDeviceMapping {
	result := make([]models.BlockDeviceMapping, 0)
	for _, mapping := range mappings {
		if mapping.DeviceName == nil || mapping.Ebs == nil || mapping.Ebs.VolumeId == nil {
			continue
		}
		result = append(result, models.BlockDeviceMapping{
			DeviceName: *mapping.DeviceName,
			VolumeId:   *mapping.Ebs.VolumeId,
		})
	}
	return result
}

// Helper function to parse security groups
func parseSecurityGroups(groups []types.GroupIdentifier) []models.SecurityGroup {
	result := make([]models.SecurityGroup, 0)
	for _, group := range groups {
		if group.GroupId == nil {
			continue
		}
		result = append(result, models.SecurityGroup{
			GroupId: *group.GroupId,
		})
	}
	return result
}

// Helper function to parse network interfaces
func parseNetworkInterfaces(interfaces []types.InstanceNetworkInterface) []models.NetworkInterface {
	result := make([]models.NetworkInterface, 0)
	for _, iface := range interfaces {
		if iface.PrivateIpAddress == nil {
			continue
		}
		ni := models.NetworkInterface{
			PrivateIpAddress: *iface.PrivateIpAddress,
		}
		if iface.Association != nil && iface.Association.PublicIp != nil {
			ni.PublicIpAddress = *iface.Association.PublicIp
		}
		result = append(result, ni)
	}
	return result
}
