package teraform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"Savannahtakehomeassi/teraform/models"
)

func TestParseTerraformInstance(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewTerraformClient()

	sampleState := models.TerraformState{
		Version:          4,
		TerraformVersion: "1.6.2",
		Serial:           1,
		Lineage:          "abc123",
		Resources: []models.Resource{
			{
				Mode:     "managed",
				Type:     "aws_instance",
				Name:     "example",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []models.Instance{
					{
						SchemaVersion: 1,
						Attributes: models.InstanceAttributes{
							AMI:            "ami-12345678",
							InstanceID:     "i-1234567890abcdef0",
							InstanceType:   "t2.micro",
							PrivateIP:      "10.0.0.1",
							PublicIP:       "3.3.3.3",
							KeyName:        "my-key",
							Tags:           map[string]string{"Name": "test-instance"},
							SecurityGroups: []string{"sg-abc123"},
						},
					},
				},
			},
		},
	}

	validJSON, err := json.Marshal(sampleState)
	require.NoError(t, err)

	validFile := filepath.Join(tmpDir, "terraform.tfstate")
	require.NoError(t, os.WriteFile(validFile, validJSON, 0644))

	invalidFile := filepath.Join(tmpDir, "invalid.tfstate")
	require.NoError(t, os.WriteFile(invalidFile, []byte("not-a-json"), 0644))

	tests := []struct {
		name        string
		filePath    string
		expectError bool
		verify      func(t *testing.T, state *models.TerraformState)
	}{
		{
			name:        "Valid Terraform state",
			filePath:    validFile,
			expectError: false,
			verify: func(t *testing.T, state *models.TerraformState) {
				assert.Equal(t, "t2.micro", state.Resources[0].Instances[0].Attributes.InstanceType)
				assert.Equal(t, "test-instance", state.Resources[0].Instances[0].Attributes.Tags["Name"])
			},
		},
		{
			name:        "Missing file",
			filePath:    filepath.Join(tmpDir, "missing.tfstate"),
			expectError: true,
		},
		{
			name:        "Invalid JSON format",
			filePath:    invalidFile,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state, err := client.ParseTerraformInstance(tc.filePath)
			if tc.expectError {
				require.Error(t, err)
				assert.Nil(t, state)
			} else {
				require.NoError(t, err)
				require.NotNil(t, state)
				tc.verify(t, state)
			}
		})
	}
}

func TestParseHCLConfig(t *testing.T) {
	client := NewTerraformClient()

	tests := []struct {
		name        string
		content     string
		expected    *models.TFInstance
		expectError bool
	}{
		{
			name: "Valid HCL with tags",
			content: `
resource "aws_instance" "example" {
  ami           = "ami-123456"
  instance_type = "t2.micro"
  tags = {
    Name = "example"
    Env  = "dev"
  }
}
`,
			expected: &models.TFInstance{
				AMI:          "ami-123456",
				InstanceType: "t2.micro",
				Tags: map[string]string{
					"Name": "example",
					"Env":  "dev",
				},
			},
		},
		{
			name: "Missing tags block",
			content: `
resource "aws_instance" "example" {
  ami           = "ami-789012"
  instance_type = "t3.medium"
}
`,
			expected: &models.TFInstance{
				AMI:          "ami-789012",
				InstanceType: "t3.medium",
				Tags:         map[string]string{},
			},
		},
		{
			name: "Irregular spacing and inline comments",
			content: `
resource "aws_instance" "example" {
  ami="ami-irregular"   # inline comment
  instance_type =    "t3.small"   
  tags = { 
    Project="drift-detector"
    Team =  "devops"  // team tag
  }
}
`,
			expected: &models.TFInstance{
				AMI:          "ami-irregular",
				InstanceType: "t3.small",
				Tags: map[string]string{
					"Project": "drift-detector",
					"Team":    "devops",
				},
			},
		},
		{
			name: "No aws_instance block",
			content: `
resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}
`,
			expected: &models.TFInstance{
				Tags: map[string]string{},
			},
		},
		{
			name:        "Corrupted file format",
			content:     `{{{{{`,
			expectError: false, // Will parse nothing, but won't return an error since scanner just sees junk
			expected: &models.TFInstance{
				Tags: map[string]string{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile := writeTempFile(t, tc.content)
			defer os.Remove(tmpFile)

			instance, err := client.ParseHCLConfig(tmpFile)
			if tc.expectError {
				require.Error(t, err)
				assert.Nil(t, instance)
			} else {
				require.NoError(t, err)
				require.NotNil(t, instance)
				assert.Equal(t, tc.expected.AMI, instance.AMI)
				assert.Equal(t, tc.expected.InstanceType, instance.InstanceType)
				assert.Equal(t, tc.expected.Tags, instance.Tags)
			}
		})
	}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)
	return tmpFile
}
