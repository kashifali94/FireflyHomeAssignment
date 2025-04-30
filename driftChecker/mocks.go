package driftChecker

import (
	"github.com/stretchr/testify/mock"

	awsm "Savannahtakehomeassi/awsd/models"
	terafm "Savannahtakehomeassi/teraform/models"
)

// MockAWSClient is a mock implementation of AWSClient
type MockAWSClient struct {
	mock.Mock
}

// GetAWSInstance mocks the GetAWSInstance method
func (m *MockAWSClient) GetAWSInstance() (*awsm.AWSInstance, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*awsm.AWSInstance), args.Error(1)
}

// MockTerraformClient is a mock implementation of TerraformClient
type MockTerraformClient struct {
	mock.Mock
}

// ParseTerraformInstance mocks the ParseTerraformInstance method
func (m *MockTerraformClient) ParseTerraformInstance(path string) (*terafm.TerraformState, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*terafm.TerraformState), args.Error(1)
}

// ParseHCLConfig mocks the ParseHCLConfig method
func (m *MockTerraformClient) ParseHCLConfig(path string) (*terafm.TFInstance, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*terafm.TFInstance), args.Error(1)
}
