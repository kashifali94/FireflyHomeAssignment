package main

import (
	"context"
	"errors"
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
	require.NoError(t, err, "Failed to initialize logger")
	defer logger.Sync()

	logger := zap.L().With(zap.String("package", "test"))

	testCases := []testCase{
		{
			name: "successful configuration loading",
			setup: func(t *testing.T) (*configuration.Config, *driftChecker.MockAWSClient, *driftChecker.MockTerraformClient, *zap.Logger) {
				config, err := configuration.Initialize()
				if err != nil {
					logger.Error("Failed to initialize configuration",
						zap.Error(errors.New("configuration initialization failed")))
					require.NoError(t, err)
				}
				logger.Info("Configuration loaded successfully")
				return config, nil, nil, logger
			},
			expectedError: false,
		},
		{
			name: "successful AWS client creation",
			setup: func(t *testing.T) (*configuration.Config, *driftChecker.MockAWSClient, *driftChecker.MockTerraformClient, *zap.Logger) {
				config, err := configuration.Initialize()
				if err != nil {
					logger.Error("Failed to initialize configuration",
						zap.Error(errors.New("configuration initialization failed")))
					require.NoError(t, err)
				}
				awsClient := new(driftChecker.MockAWSClient)
				logger.Info("AWS client created successfully")
				return config, awsClient, nil, logger
			},
			expectedError: false,
		},
		{
			name: "successful Terraform client creation",
			setup: func(t *testing.T) (*configuration.Config, *driftChecker.MockAWSClient, *driftChecker.MockTerraformClient, *zap.Logger) {
				config, err := configuration.Initialize()
				if err != nil {
					logger.Error("Failed to initialize configuration",
						zap.Error(errors.New("configuration initialization failed")))
					require.NoError(t, err)
				}
				tfClient := new(driftChecker.MockTerraformClient)
				logger.Info("Terraform client created successfully")
				return config, nil, tfClient, logger
			},
			expectedError: false,
		},
		{
			name: "successful DriftService creation and running",
			setup: func(t *testing.T) (*configuration.Config, *driftChecker.MockAWSClient, *driftChecker.MockTerraformClient, *zap.Logger) {
				config, err := configuration.Initialize()
				if err != nil {
					logger.Error("Failed to initialize configuration",
						zap.Error(errors.New("configuration initialization failed")))
					require.NoError(t, err)
				}
				logger.Info("Configuration loaded successfully")

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
				logger.Info("AWS client mock setup completed")

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
				logger.Info("Terraform client mock setup completed")

				return config, awsClient, tfClient, logger
			},
			expectedError: false,
			timeout:       2 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger.Info("Starting test case", zap.String("test_case", tc.name))

			config, awsClient, tfClient, logger := tc.setup(t)

			switch tc.name {
			case "successful configuration loading":
				assert.NotNil(t, config, "Configuration should not be nil")
				assert.NotEmpty(t, config.TFStatePath, "TFStatePath should not be empty")
				assert.NotEmpty(t, config.MainTFPath, "MainTFPath should not be empty")
				assert.Greater(t, config.CheckInterval, 0, "CheckInterval should be greater than 0")
				logger.Info("Configuration validation completed successfully")

			case "successful AWS client creation":
				assert.NotNil(t, awsClient, "AWS client should not be nil")
				logger.Info("AWS client validation completed successfully")

			case "successful Terraform client creation":
				assert.NotNil(t, tfClient, "Terraform client should not be nil")
				logger.Info("Terraform client validation completed successfully")

			case "successful DriftService creation and running":
				driftService := driftChecker.NewDriftService(awsClient, tfClient, logger)
				assert.NotNil(t, driftService, "DriftService should not be nil")
				logger.Info("DriftService created successfully")

				ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
				defer cancel()

				errChan := make(chan error, 1)
				go func() {
					logger.Info("Starting DriftService run loop")
					err := driftService.RunLoop(ctx, config.TFStatePath, config.MainTFPath, config.CheckInterval)
					if err != nil {
						logger.Error("DriftService run loop failed",
							zap.Error(errors.New("drift service run loop failed")))
					}
					errChan <- err
				}()

				select {
				case err := <-errChan:
					if tc.expectedError {
						assert.Error(t, err, "Expected an error but got none")
					} else {
						assert.NoError(t, err, "Unexpected error occurred")
					}
				case <-ctx.Done():
					logger.Info("Test completed due to timeout",
						zap.Duration("timeout", tc.timeout))
				}
			}
		})
	}
}
