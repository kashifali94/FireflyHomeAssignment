package awsd

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/viper"
	"log"

	"Savannahtakehomeassi/awsd/models"
)

type AwsClient struct {
	client EC2API
}

func NewEC2ClientWithConfig(cfg aws.Config) *AwsClient {
	return &AwsClient{
		client: ec2.NewFromConfig(cfg),
	}
}

// NewEC2Client creates and returns a configured EC2 client for local development with LocalStack
func NewEC2Client() (*AwsClient, error) {

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(viper.GetString("AWS_REGION")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(viper.GetString("AWS_SECRET_ACCESS_KEY"),
			viper.GetString("AWS_ACCESS_KEY_ID"), "")),
		config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{URL: viper.GetString("LOCALSTACK_URL"), SigningRegion: region}, nil
			}),
		),
	)
	if err != nil {
		return nil, err
	}

	return NewEC2ClientWithConfig(cfg), nil
}

// GetAWSInstance fetches AWS EC2 instance details
func GetAWSInstance(awS *AwsClient) (*models.AWSInstance, error) {
	// Describe the instance
	output, err := awS.client.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{},
	})

	if err != nil {
		return nil, err
	}

	// Check if the instance exists in the response
	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		log.Print("unable to find the instance")
		return nil, errors.New("no instances found")
	}

	// Extract instance details
	i := output.Reservations[0].Instances[0]
	log.Printf("found the instance: %s", *i.InstanceId)

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
		PrivateIP:           aws.ToString(i.PrivateIpAddress), // Safely dereferencing pointer
		KeyName:             aws.ToString(i.KeyName),          // Safely dereferencing pointer
		Tags:                tags,
		PublicIP:            aws.ToString(i.PublicIpAddress),
		LaunchTime:          i.LaunchTime.String(),
		PrivateDnsName:      aws.ToString(i.PrivateDnsName),
		BlockDeviceMappings: parseBlockDeviceMappings(i.BlockDeviceMappings),
		SecurityGroups:      parseSecurityGroups(i.SecurityGroups),
		NetworkInterfaces:   parseNetworkInterfaces(i.NetworkInterfaces),
	}

	log.Printf("AWS: response is parsed successfully")

	return awsInstance, nil
}

// Helper function to parse block device mappings
func parseBlockDeviceMappings(mappings []types.InstanceBlockDeviceMapping) []models.BlockDeviceMapping {
	result := make([]models.BlockDeviceMapping, 0)
	for _, mapping := range mappings {
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
		result = append(result, models.NetworkInterface{
			PrivateIpAddress: *iface.PrivateIpAddress,
			PublicIpAddress:  *iface.Association.PublicIp,
		})
	}
	return result
}
