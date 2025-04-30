package driftChecker

// Client defines the interface for drift checking operations
type Client interface {
	GetAWSClient() interface{}
	GetTerraformClient() interface{}
	GetTerraformStatePath() string
	GetTerraformConfigPath() string
}
