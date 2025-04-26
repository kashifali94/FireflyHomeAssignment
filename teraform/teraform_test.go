package teraform

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"Savannahtakehomeassi/teraform/models"
)

func TestParseTerraformInstance(t *testing.T) {
	tmpDir := t.TempDir()

	// Create full sample TerraformState
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
							PrivateDNS:     "ip-10-0-0-1.ec2.internal",
							PublicDNS:      "ec2-3-3-3-3.compute-1.amazonaws.com",
							RootBlockDevice: []models.RootBlockDevice{
								{
									DeviceName:          "/dev/xvda",
									VolumeID:            "vol-abc123",
									VolumeSize:          8,
									VolumeType:          "gp2",
									DeleteOnTermination: true,
									Encrypted:           false,
								},
							},
						},
					},
				},
			},
		},
	}

	// Marshal to JSON
	validJSON, err := json.Marshal(sampleState)
	assert.NoError(t, err)

	// Write it to a file
	validFile := filepath.Join(tmpDir, "terraform.tfstate")
	err = os.WriteFile(validFile, validJSON, 0644)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		filePath       string
		expectError    bool
		expectedInstID string
		expectedType   string
		expectedTag    string
	}{
		{
			name:           "Valid state file",
			filePath:       validFile,
			expectError:    false,
			expectedInstID: "i-1234567890abcdef0",
			expectedType:   "t2.micro",
			expectedTag:    "test-instance",
		},
		{
			name:        "Missing file",
			filePath:    filepath.Join(tmpDir, "missing.tfstate"),
			expectError: true,
		},
		{
			name:        "Invalid JSON",
			filePath:    filepath.Join(tmpDir, "invalid.tfstate"),
			expectError: true,
		},
	}

	// Write invalid JSON
	err = os.WriteFile(filepath.Join(tmpDir, "invalid.tfstate"), []byte("invalid-json"), 0644)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, err := ParseTerraformInstance(tt.filePath)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, state)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, state)
				res := state.Resources[0].Instances[0].Attributes
				assert.Equal(t, tt.expectedInstID, res.InstanceID)
				assert.Equal(t, tt.expectedType, res.InstanceType)
				assert.Equal(t, tt.expectedTag, res.Tags["Name"])
			}
		})
	}
}

func TestParseHCLConfig(t *testing.T) {
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
			expectError: false,
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
			expectError: false,
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
			expectError: false,
		},
		{
			name: "No aws_instance block",
			content: `
resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}
`,
			expected:    &models.TFInstance{Tags: map[string]string{}},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "main-*.tf")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tc.content)
			require.NoError(t, err)
			tmpFile.Close()

			instance, err := ParseHCLConfig(tmpFile.Name())
			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected.AMI, instance.AMI)
			require.Equal(t, tc.expected.InstanceType, instance.InstanceType)
			require.Equal(t, tc.expected.Tags, instance.Tags)
		})
	}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.tf")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return tmpFile
}
