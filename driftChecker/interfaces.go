package driftChecker

import (
	awsm "Savannahtakehomeassi/awsd/models"
	terafm "Savannahtakehomeassi/teraform/models"
	"context"
)

// AWSClient defines the interface for AWS operations
type AWSClient interface {
	GetAWSInstance() (*awsm.AWSInstance, error)
}

// TerraformClient defines the interface for Terraform operations
type TerraformClient interface {
	ParseTerraformInstance(path string) (*terafm.TerraformState, error)
	ParseHCLConfig(path string) (*terafm.TFInstance, error)
}

// DriftChecker defines the interface for drift checking operations
type DriftChecker interface {
	RunLoop(ctx context.Context, tfSpath, mainfile string, interval int) error
	runDriftCheck(ctx context.Context, tfPath, mainFile string) error
}
