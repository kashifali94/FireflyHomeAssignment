package driftChecker

import (
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	awsm "Savannahtakehomeassi/awsd/models"
	"Savannahtakehomeassi/logger"
	terafm "Savannahtakehomeassi/teraform/models"
)

func TestCompareInstances(t *testing.T) {
	awsInstance := &awsm.AWSInstance{
		InstanceID:   "i-12345",
		InstanceType: "t2.micro",
		PrivateIP:    "10.0.0.1",
		PublicIP:     "54.214.227.242",
		AMI:          "ami-12345",
		Tags:         map[string]string{"env": "production"},
		NetworkInterfaces: []awsm.NetworkInterface{
			{
				PrivateIpAddress: "10.0.0.1",
				PublicIpAddress:  "54.214.227.242",
			},
		},
	}

	tfState := &terafm.TFInstance{
		ID:           "i-12345",
		InstanceType: "t2.micro",
		AMI:          "ami-12345",
		Tags:         map[string]string{"env": "production"},
	}

	// Test no drift case
	drift, err := compareInstances(awsInstance, tfState)
	assert.NoError(t, err)
	assert.Len(t, drift, 1)
	assert.Equal(t, "No drift detected between AWS instance and Terraform state.", drift[0])

	// Test drift in instance type
	tfState.InstanceType = "t2.large"
	drift, err = compareInstances(awsInstance, tfState)
	assert.NoError(t, err)
	assert.Len(t, drift, 1)
	assert.Equal(t, "Drift in instance i-12345: instance_type mismatch (AWS: t2.micro, TF: t2.large)", drift[0])
}

func TestCompareInstances_NewFormat(t *testing.T) {
	// Initialize logger
	if err := logger.Initialize("info"); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	tests := []struct {
		name     string
		aws      *awsm.AWSInstance
		tf       *terafm.TFInstance
		expected []string
	}{
		{
			name: "no drift",
			aws: &awsm.AWSInstance{
				InstanceID:   "i-12345",
				InstanceType: "t2.micro",
				AMI:          "ami-12345",
				Tags:         map[string]string{"env": "production"},
			},
			tf: &terafm.TFInstance{
				ID:           "i-12345",
				InstanceType: "t2.micro",
				AMI:          "ami-12345",
				Tags:         map[string]string{"env": "production"},
			},
			expected: []string{"No drift detected between AWS instance and Terraform state."},
		},
		{
			name: "instance type drift",
			aws: &awsm.AWSInstance{
				InstanceID:   "i-12345",
				InstanceType: "t2.micro",
				AMI:          "ami-12345",
				Tags:         map[string]string{"env": "production"},
			},
			tf: &terafm.TFInstance{
				ID:           "i-12345",
				InstanceType: "t2.large",
				AMI:          "ami-12345",
				Tags:         map[string]string{"env": "production"},
			},
			expected: []string{"Drift in instance i-12345: instance_type mismatch (AWS: t2.micro, TF: t2.large)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compare instances
			drifts, err := compareInstances(tt.aws, tt.tf)
			require.NoError(t, err)
			if len(drifts) > 0 {
				logger.Info("Drift detected in HCL comparison",
					zap.String("function", "compareInstances"),
					zap.String("instance_id", tt.aws.InstanceID),
					zap.String("operation", "hcl_comparison_complete"),
					zap.String("status", "drift_detected"),
					zap.Int("drift_count", len(drifts)))
			} else {
				logger.Info("No drift detected in HCL comparison",
					zap.String("function", "compareInstances"),
					zap.String("instance_id", tt.aws.InstanceID),
					zap.String("operation", "hcl_comparison_complete"),
					zap.String("status", "no_drift"))
			}
			require.Equal(t, tt.expected, drifts)
		})
	}
}
