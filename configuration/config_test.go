package configuration_test

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"Savannahtakehomeassi/configuration"
)

// createTempEnvFile writes a temporary .env file and returns its path
func createTempEnvFile(t *testing.T, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", ".env.test")
	if err != nil {
		t.Fatalf("failed to create temp env file: %v", err)
	}

	_, err = tmpFile.WriteString(content)
	if err != nil {
		t.Fatalf("failed to write to temp env file: %v", err)
	}

	err = tmpFile.Close()
	if err != nil {
		t.Fatalf("failed to close temp env file: %v", err)
	}

	return tmpFile.Name()
}

func TestInitialize_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		envFile    string // if set, will write a .env file with this content
		expectErr  bool
		assertions func(*testing.T, *configuration.Config)
	}{
		{
			name: "Valid configuration from environment variables",
			env: map[string]string{
				"TFSTATE_PATH":               "test.tfstate",
				"MAINTF_PATH":                "main.tf",
				"CHECK_INTERVAL_MINUTES":     "10",
				"AWS_REGION":                 "us-west-2",
				"AWS_ACCESS_KEY_ID":          "AKIAEXAMPLE",
				"AWS_SECRET_ACCESS_KEY":      "secret123",
				"LOG_LEVEL":                  "debug",
				"MAX_RETRIES":                "5",
				"RETRY_DELAY_SECONDS":        "10",
				"COMPARISON_TIMEOUT_SECONDS": "60",
			},
			expectErr: false,
			assertions: func(t *testing.T, cfg *configuration.Config) {
				assert.Equal(t, "test.tfstate", cfg.TFStatePath)
				assert.Equal(t, "main.tf", cfg.MainTFPath)
				assert.Equal(t, 10, cfg.CheckInterval)
				assert.Equal(t, "us-west-2", cfg.AWSRegion)
				assert.Equal(t, "AKIAEXAMPLE", cfg.AcessKeyID)
				assert.Equal(t, "secret123", cfg.AccessSecret)
				assert.Equal(t, "debug", cfg.LogLevel)
				assert.Equal(t, 5, cfg.MaxRetries)
				assert.Equal(t, 10, cfg.RetryDelay)
				assert.Equal(t, 60, cfg.ComparisonTimeout)
			},
		},
		{
			name: "Configuration from temp .env file",
			envFile: `
TFSTATE_PATH=envfile.tfstate
MAINTF_PATH=envfile_main.tf
CHECK_INTERVAL_MINUTES=15
AWS_REGION=ap-south-1
AWS_ACCESS_KEY_ID=ENVKEY
AWS_SECRET_ACCESS_KEY=ENVSECRET
LOG_LEVEL=error
MAX_RETRIES=2
RETRY_DELAY_SECONDS=6
COMPARISON_TIMEOUT_SECONDS=25
`,
			expectErr: false,
			assertions: func(t *testing.T, cfg *configuration.Config) {
				assert.Equal(t, "envfile.tfstate", cfg.TFStatePath)
				assert.Equal(t, "envfile_main.tf", cfg.MainTFPath)
				assert.Equal(t, 15, cfg.CheckInterval)
				assert.Equal(t, "ap-south-1", cfg.AWSRegion)
				assert.Equal(t, "ENVKEY", cfg.AcessKeyID)
				assert.Equal(t, "ENVSECRET", cfg.AccessSecret)
				assert.Equal(t, "error", cfg.LogLevel)
				assert.Equal(t, 2, cfg.MaxRetries)
				assert.Equal(t, 6, cfg.RetryDelay)
				assert.Equal(t, 25, cfg.ComparisonTimeout)
			},
		},
		{
			name: "Invalid CHECK_INTERVAL_MINUTES from env",
			env: map[string]string{
				"TFSTATE_PATH":           "file.tfstate",
				"MAINTF_PATH":            "main.tf",
				"CHECK_INTERVAL_MINUTES": "-1",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			var tempEnvFile string

			// Write .env file if content is specified
			if tt.envFile != "" {
				tempEnvFile = createTempEnvFile(t, tt.envFile)
				defer os.Remove(tempEnvFile)
				viper.SetConfigFile(tempEnvFile)
			}

			// Set environment variables
			for k, v := range tt.env {
				_ = os.Setenv(k, v)
			}

			cfg, err := configuration.Initialize()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.assertions != nil {
					tt.assertions(t, cfg)
				}
			}

			// Clean up
			for k := range tt.env {
				_ = os.Unsetenv(k)
			}
		})
	}
}

func TestInitialize_WithTempEnvFile(t *testing.T) {
	envContent := `
TFSTATE_PATH=custom.tfstate
MAINTF_PATH=custom_main.tf
CHECK_INTERVAL_MINUTES=10
AWS_REGION=eu-west-1
AWS_ACCESS_KEY_ID=TESTKEY
AWS_SECRET_ACCESS_KEY=TESTSECRET
LOG_LEVEL=warn
MAX_RETRIES=4
RETRY_DELAY_SECONDS=7
COMPARISON_TIMEOUT_SECONDS=45
`
	envFilePath := createTempEnvFile(t, envContent)
	defer os.Remove(envFilePath)

	viper.Reset()
	viper.SetConfigFile(envFilePath)

	cfg, err := configuration.Initialize()
	assert.NoError(t, err)
	assert.Equal(t, "custom.tfstate", cfg.TFStatePath)
	assert.Equal(t, "custom_main.tf", cfg.MainTFPath)
	assert.Equal(t, 10, cfg.CheckInterval)
	assert.Equal(t, "eu-west-1", cfg.AWSRegion)
	assert.Equal(t, "TESTKEY", cfg.AcessKeyID)
	assert.Equal(t, "TESTSECRET", cfg.AccessSecret)
	assert.Equal(t, "warn", cfg.LogLevel)
	assert.Equal(t, 4, cfg.MaxRetries)
	assert.Equal(t, 7, cfg.RetryDelay)
	assert.Equal(t, 45, cfg.ComparisonTimeout)
}
