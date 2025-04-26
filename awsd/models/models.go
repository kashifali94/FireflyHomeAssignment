package models

// AWSInstance represents the structure of an EC2 instance
type AWSInstance struct {
	InstanceID          string
	InstanceType        string
	PrivateIP           string
	PublicIP            string
	KeyName             string
	LaunchTime          string
	PrivateDnsName      string
	AMI                 string
	BlockDeviceMappings []BlockDeviceMapping
	SecurityGroups      []SecurityGroup
	NetworkInterfaces   []NetworkInterface
	Tags                map[string]string
}

// BlockDeviceMapping represents a block device mapping in AWS
type BlockDeviceMapping struct {
	DeviceName string
	VolumeId   string
}

// SecurityGroup represents a security group associated with an instance
type SecurityGroup struct {
	GroupId string
}

// NetworkInterface represents a network interface associated with an instance
type NetworkInterface struct {
	PrivateIpAddress string
	PublicIpAddress  string
}
