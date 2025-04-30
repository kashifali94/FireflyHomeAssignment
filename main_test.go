package main

import (
	"context"
	"testing"
	"time"

	awsm "Savannahtakehomeassi/awsd/models"
	"Savannahtakehomeassi/configuration"
	"Savannahtakehomeassi/driftChecker"
	"Savannahtakehomeassi/logger"
	terafm "Savannahtakehomeassi/teraform/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testCase struct {
	name          string
	setup         func(t *testing.T) (*configuration.Config, *driftChecker.MockAWSClient, *driftChecker.MockTerraformClient, *zap.Logger)
	expectedError bool
	timeout       time.Duration
}

func TestMainSetup(t *testing.T) {
	// Initialize logger for test
	err := logger.Initialize("info")
	require.NoError(t, err)
	defer logger.Sync()

	logger := zap.L().With(zap.String("package", "test"))

	testCases := []testCase{
		{
			name: "successful configuration loading",
			setup: func(t *testing.T) (*configuration.Config, *driftChecker.MockAWSClient, *driftChecker.MockTerraformClient, *zap.Logger) {
				config, err := configuration.Initialize()
				require.NoError(t, err)
				return config, nil, nil, logger
			},
			expectedError: false,
		},
		{
			name: "successful AWS client creation",
			setup: func(t *testing.T) (*configuration.Config, *driftChecker.MockAWSClient, *driftChecker.MockTerraformClient, *zap.Logger) {
				config, err := configuration.Initialize()
				require.NoError(t, err)
				awsClient := new(driftChecker.MockAWSClient)
				return config, awsClient, nil, logger
			},
			expectedError: false,
		},
		{
			name: "successful Terraform client creation",
			setup: func(t *testing.T) (*configuration.Config, *driftChecker.MockAWSClient, *driftChecker.MockTerraformClient, *zap.Logger) {
				config, err := configuration.Initialize()
				require.NoError(t, err)
				tfClient := new(driftChecker.MockTerraformClient)
				return config, nil, tfClient, logger
			},
			expectedError: false,
		},
		{
			name: "successful DriftService creation and running",
			setup: func(t *testing.T) (*configuration.Config, *driftChecker.MockAWSClient, *driftChecker.MockTerraformClient, *zap.Logger) {
				config, err := configuration.Initialize()
				require.NoError(t, err)

				// Create and set up mock AWS client
				awsClient := new(driftChecker.MockAWSClient)
				awsClient.On("GetAWSInstance").Return(&awsm.AWSInstance{
					InstanceID:   "i-1234567890abcdef0",
					InstanceType: "t2.micro",
					PrivateIP:    "10.0.0.1",
					PublicIP:     "54.0.0.1",
					KeyName:      "test-key",
					Tags: map[string]string{
						"Name": "test-instance",
					},
					SecurityGroups: []awsm.SecurityGroup{
						{GroupId: "sg-12345678"},
					},
					NetworkInterfaces: []awsm.NetworkInterface{
						{
							PrivateIpAddress: "10.0.0.1",
							PublicIpAddress:  "54.0.0.1",
						},
					},
				}, nil)

				// Create and set up mock Terraform client
				tfClient := new(driftChecker.MockTerraformClient)
				tfClient.On("ParseTerraformInstance", config.TFStatePath).Return(&terafm.TerraformState{
					Resources: []terafm.Resource{
						{
							Type: "aws_instance",
							Instances: []terafm.Instance{
								{
									Attributes: terafm.InstanceAttributes{
										InstanceID:   "i-1234567890abcdef0",
										InstanceType: "t2.micro",
										PrivateIP:    "10.0.0.1",
										PublicIP:     "54.0.0.1",
										KeyName:      "test-key",
										Tags: map[string]string{
											"Name": "test-instance",
										},
										SecurityGroups: []string{"sg-12345678"},
									},
								},
							},
						},
					},
				}, nil)
				tfClient.On("ParseHCLConfig", config.MainTFPath).Return(&terafm.TFInstance{
					ID:           "i-1234567890abcdef0",
					InstanceType: "t2.micro",
					AMI:          "ami-12345678",
					Tags: map[string]string{
						"Name": "test-instance",
					},
				}, nil)

				return config, awsClient, tfClient, logger
			},
			expectedError: false,
			timeout:       2 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, awsClient, tfClient, logger := tc.setup(t)

			switch tc.name {
			case "successful configuration loading":
				assert.NotNil(t, config)
				assert.NotEmpty(t, config.TFStatePath)
				assert.NotEmpty(t, config.MainTFPath)
				assert.Greater(t, config.CheckInterval, 0)

			case "successful AWS client creation":
				assert.NotNil(t, awsClient)

			case "successful Terraform client creation":
				assert.NotNil(t, tfClient)

			case "successful DriftService creation and running":
				driftService := driftChecker.NewDriftService(awsClient, tfClient, logger)
				assert.NotNil(t, driftService)

				ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
				defer cancel()

				errChan := make(chan error, 1)
				go func() {
					err := driftService.RunLoop(ctx, config.TFStatePath, config.MainTFPath, config.CheckInterval)
					errChan <- err
				}()

				select {
				case err := <-errChan:
					if tc.expectedError {
						assert.Error(t, err)
					} else {
						assert.NoError(t, err)
					}
				case <-ctx.Done():
					// Expected timeout
				}
			}
		})
	}
}
