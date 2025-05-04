package driftChecker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	awsm "Savannahtakehomeassi/awsd/models"
	"Savannahtakehomeassi/logger"
	terafm "Savannahtakehomeassi/teraform/models"
)

func TestDriftService_RunLoop(t *testing.T) {
	logger, _ := zap.NewProduction() // Creating a logger for the tests
	defer logger.Sync()              // Ensure logs are flushed

	tests := []struct {
		name          string
		awsMock       *MockAWSClient
		tfMock        *MockTerraformClient
		mockAWSError  error
		mockTFError   error
		expectErr     bool
		expectTimeout bool
		mockAWS       *awsm.AWSInstance
		mockTerraform *terafm.TerraformState
	}{
		{
			name:          "Test no drift",
			awsMock:       new(MockAWSClient),
			tfMock:        new(MockTerraformClient),
			mockAWSError:  nil,
			mockTFError:   nil,
			expectErr:     false,
			expectTimeout: true,
			mockAWS:       &awsm.AWSInstance{InstanceID: "i-12345", InstanceType: "t2.micro"},
			mockTerraform: &terafm.TerraformState{Resources: []terafm.Resource{
				{
					Type: "aws_instance",
					Instances: []terafm.Instance{
						{Attributes: terafm.InstanceAttributes{InstanceID: "i-12345", InstanceType: "t2.micro"}},
					},
				},
			}},
		},
		{
			name:          "Test drift detected",
			awsMock:       new(MockAWSClient),
			tfMock:        new(MockTerraformClient),
			mockAWSError:  nil,
			mockTFError:   nil,
			expectErr:     false,
			expectTimeout: true,
			mockAWS:       &awsm.AWSInstance{InstanceID: "i-12345", InstanceType: "t2.large"},
			mockTerraform: &terafm.TerraformState{Resources: []terafm.Resource{
				{
					Type: "aws_instance",
					Instances: []terafm.Instance{
						{Attributes: terafm.InstanceAttributes{InstanceID: "i-12345", InstanceType: "t2.micro"}},
					},
				},
			}},
		},
		{
			name:         "Test AWS client error",
			awsMock:      new(MockAWSClient),
			tfMock:       new(MockTerraformClient),
			mockAWSError: errors.New("AWS error"),
			mockTFError:  nil,
			expectErr:    true,
		},
		{
			name:         "Test Terraform client error",
			awsMock:      new(MockAWSClient),
			tfMock:       new(MockTerraformClient),
			mockAWSError: nil,
			mockTFError:  errors.New("Terraform error"),
			expectErr:    true,
			mockAWS:      &awsm.AWSInstance{InstanceID: "i-12345", InstanceType: "t2.micro"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			// Reset mock expectations
			tt.awsMock.ExpectedCalls = nil
			tt.tfMock.ExpectedCalls = nil

			tt.awsMock.On("GetAWSInstance").Return(tt.mockAWS, tt.mockAWSError)

			// Only set up Terraform mocks if we expect to reach them
			if tt.mockAWSError == nil {
				tt.tfMock.On("ParseTerraformInstance", mock.Anything).Return(tt.mockTerraform, tt.mockTFError)

				// Only set up ParseHCLConfig mock if we expect it to be called
				if tt.mockTFError == nil {
					// Create a TFInstance from the TerraformState for the HCL config
					var tfInstance *terafm.TFInstance
					if tt.mockTerraform != nil && len(tt.mockTerraform.Resources) > 0 && len(tt.mockTerraform.Resources[0].Instances) > 0 {
						instance := tt.mockTerraform.Resources[0].Instances[0]
						tfInstance = &terafm.TFInstance{
							ID:           instance.Attributes.InstanceID,
							InstanceType: instance.Attributes.InstanceType,
							AMI:          instance.Attributes.AMI,
							Tags:         instance.Attributes.Tags,
						}
					}
					tt.tfMock.On("ParseHCLConfig", mock.Anything).Return(tfInstance, tt.mockTFError)
				}
			}

			// Create DriftService with mocked clients
			service := &DriftService{
				awsClient:       tt.awsMock,
				terraformClient: tt.tfMock,
				logger:          logger,
			}

			// Run the test with a short interval
			err := service.RunLoop(ctx, "path/to/tfstate", "path/to/mainfile", 1)

			if tt.expectTimeout {
				// For timeout cases, we expect a context cancellation error
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "drift check cancelled")
			} else if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Assertions to ensure that all mock expectations were met
			tt.awsMock.AssertExpectations(t)
			tt.tfMock.AssertExpectations(t)
		})
	}
}

func TestDriftService_runDriftCheck(t *testing.T) {
	tests := []struct {
		name        string
		awsInstance *awsm.AWSInstance
		awsError    error
		tfState     *terafm.TerraformState
		tfInstance  *terafm.TFInstance
		tfPath      string
		mainFile    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful drift check",
			awsInstance: &awsm.AWSInstance{
				InstanceID:     "i-12345",
				InstanceType:   "t2.micro",
				AMI:            "ami-12345",
				PrivateIP:      "10.0.0.1",
				PublicIP:       "54.214.227.242",
				PrivateDnsName: "ip-10-0-0-1.ec2.internal",
				Tags:           map[string]string{"Name": "TestInstance"},
				NetworkInterfaces: []awsm.NetworkInterface{
					{
						PrivateIpAddress: "10.0.0.1",
						PublicIpAddress:  "54.214.227.242",
					},
				},
			},
			awsError: nil,
			tfState: &terafm.TerraformState{
				Resources: []terafm.Resource{
					{
						Type: "aws_instance",
						Instances: []terafm.Instance{
							{
								Attributes: terafm.InstanceAttributes{
									InstanceID:   "i-0b0f62398bf34f224",
									InstanceType: "t2.micro",
									AMI:          "ami-12345678",
									PrivateIP:    "10.249.67.6",
									PublicIP:     "54.214.227.242",
									PrivateDNS:   "ip-10-249-67-6.ec2.internal",
									Tags:         map[string]string{"Name": "TestInstance"},
								},
							},
						},
					},
				},
			},
			tfInstance: &terafm.TFInstance{
				ID:           "i-0b0f62398bf34f224",
				InstanceType: "t2.micro",
				AMI:          "ami-12345678",
				Tags:         map[string]string{"Name": "TestInstance"},
			},
			tfPath:      "terraform.tfstate",
			mainFile:    "main.tf",
			expectError: false,
		},
		{
			name:        "AWS instance not found",
			awsInstance: nil,
			awsError:    errors.New("AWS instance not found"),
			tfState:     nil,
			tfInstance:  nil,
			tfPath:      "terraform.tfstate",
			mainFile:    "main.tf",
			expectError: true,
			errorMsg:    "AWS instance not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock clients
			awsClient := new(MockAWSClient)
			tfClient := new(MockTerraformClient)
			if err := logger.Initialize("info"); err != nil {
				panic("Failed to initialize logger: " + err.Error())
			}
			defer logger.Sync()

			logger := zap.L().With(zap.String("package", "packageName"))

			// Setup mock expectations
			awsClient.On("GetAWSInstance").Return(tt.awsInstance, tt.awsError)
			if tt.awsError == nil && tt.tfState != nil {
				tfClient.On("ParseTerraformInstance", tt.tfPath).Return(tt.tfState, nil)
				tfClient.On("ParseHCLConfig", tt.mainFile).Return(tt.tfInstance, nil)
			}

			// Create service instance
			service := NewDriftService(awsClient, tfClient, logger)

			// Create context
			ctx := context.Background()

			// Run the test
			err := service.runDriftCheck(ctx, tt.tfPath, tt.mainFile)

			// Assert results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			awsClient.AssertExpectations(t)
			if tt.awsError == nil {
				tfClient.AssertExpectations(t)
			}
		})
	}
}
