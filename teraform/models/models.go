package models

import "github.com/hashicorp/hcl/v2"

// Root structure of the Terraform state file
type TerraformState struct {
	Version          int                    `json:"version"`
	TerraformVersion string                 `json:"terraform_version"`
	Serial           int                    `json:"serial"`
	Lineage          string                 `json:"lineage"`
	Outputs          map[string]interface{} `json:"outputs"`
	Resources        []Resource             `json:"resources"`
}

// Resource represents a single resource in the Terraform state file
type Resource struct {
	Mode      string     `json:"mode"`
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Provider  string     `json:"provider"`
	Instances []Instance `json:"instances"`
}

// Instance represents a specific instance of a resource
type Instance struct {
	SchemaVersion       int                `json:"schema_version"`
	Attributes          InstanceAttributes `json:"attributes"`
	SensitiveAttributes []interface{}      `json:"sensitive_attributes"`
	Private             string             `json:"private"`
	Dependencies        []string           `json:"dependencies"`
}

// InstanceAttributes contains all the instance's attributes (like ami, instance_type, etc.)
type InstanceAttributes struct {
	AMI                       string            `json:"ami"`
	ARN                       string            `json:"arn"`
	AssociatePublicIP         bool              `json:"associate_public_ip_address"`
	AvailabilityZone          string            `json:"availability_zone"`
	InstanceID                string            `json:"id"`
	InstanceType              string            `json:"instance_type"`
	PrivateIP                 string            `json:"private_ip"`
	PublicIP                  string            `json:"public_ip"`
	KeyName                   string            `json:"key_name"`
	RootBlockDevice           []RootBlockDevice `json:"root_block_device"`
	SecurityGroups            []string          `json:"security_groups"`
	Tags                      map[string]string `json:"tags"`
	VpcSecurityGroupIDs       []string          `json:"vpc_security_group_ids"`
	PrimaryNetworkInterfaceID string            `json:"primary_network_interface_id"`
	PrivateDNS                string            `json:"private_dns"`
	PublicDNS                 string            `json:"public_dns"`
}

// RootBlockDevice represents the root block device for an EC2 instance
type RootBlockDevice struct {
	DeleteOnTermination bool   `json:"delete_on_termination"`
	DeviceName          string `json:"device_name"`
	Encrypted           bool   `json:"encrypted"`
	VolumeID            string `json:"volume_id"`
	VolumeSize          int    `json:"volume_size"`
	VolumeType          string `json:"volume_type"`
}

// SecurityGroup represents an AWS security group
type SecurityGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AWSInstanceBlock struct {
	AMI          string            `hcl:"ami"`
	InstanceType string            `hcl:"instance_type"`
	Tags         map[string]string `hcl:"tags,attr"`
}

type OutputBlock struct {
	Name  string         `hcl:"name,label"`
	Value hcl.Expression `hcl:"value,attr"`
}

type ResourceBlock struct {
	Type string `hcl:"type,label"`
	Name string `hcl:"name,label"`
	Body hcl.Body
}

type Config struct {
	Resources []ResourceBlock `hcl:"resource,block"`
	Outputs   []OutputBlock   `hcl:"output,block"`
}

type TFInstance struct {
	ID           string
	InstanceType string
	AMI          string
	Tags         map[string]string
}
