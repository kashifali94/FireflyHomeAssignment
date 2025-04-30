package configuration

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"Savannahtakehomeassi/errors"
)

const (
	packageName = "configuration"
)

// Config holds the application configuration
type Config struct {
	TFStatePath       string
	MainTFPath        string
	CheckInterval     int
	AWSRegion         string
	AcessKeyID        string
	AccessSecret      string
	LogLevel          string
	MaxRetries        int
	RetryDelay        int
	ComparisonTimeout int
}

// Initialize sets up the configuration system
func Initialize() (*Config, error) {
	logger := zap.L().With(
		zap.String("package", packageName),
		zap.String("function", "Initialize"),
	)

	// Set default values
	viper.SetDefault("TFSTATE_PATH", "terraform.tfstate")
	viper.SetDefault("MAINTF_PATH", "main.tf")
	viper.SetDefault("CHECK_INTERVAL_MINUTES", 5)
	viper.SetDefault("AWS_REGION", "us-east-1")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("MAX_RETRIES", 3)
	viper.SetDefault("RETRY_DELAY_SECONDS", 5)
	viper.SetDefault("COMPARISON_TIMEOUT_SECONDS", 30)

	// Configure Viper to read from environment
	viper.AutomaticEnv()

	// Read from .env file
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, errors.New(errors.ErrConfigParse, "error reading config file",
				map[string]interface{}{
					"config_file": ".env",
				}, err)
		}
		logger.Info("No .env file found, using environment variables and defaults",
			zap.String("operation", "config_loading"),
		)
	}

	// Validate paths
	tfStatePath := viper.GetString("TFSTATE_PATH")
	if tfStatePath == "" {
		return nil, errors.New(errors.ErrConfigInvalid, "invalid TFSTATE_PATH",
			map[string]interface{}{
				"config_key": "TFSTATE_PATH",
			}, nil)
	}
	logger.Info("TFState path configured",
		zap.String("path", tfStatePath),
		zap.String("operation", "config_validation"),
	)

	mainTFPath := viper.GetString("MAINTF_PATH")
	if mainTFPath == "" {
		return nil, errors.New(errors.ErrConfigInvalid, "invalid MAINTF_PATH",
			map[string]interface{}{
				"config_key": "MAINTF_PATH",
			}, nil)
	}
	logger.Info("Main TF path configured",
		zap.String("path", mainTFPath),
		zap.String("operation", "config_validation"),
	)

	// Validate interval
	interval := viper.GetInt("CHECK_INTERVAL_MINUTES")
	if interval <= 0 {
		return nil, errors.New(errors.ErrConfigInvalid, "invalid CHECK_INTERVAL_MINUTES",
			map[string]interface{}{
				"config_key": "CHECK_INTERVAL_MINUTES",
				"value":      interval,
			}, nil)
	}
	logger.Info("Check interval configured",
		zap.Int("minutes", interval),
		zap.String("operation", "config_validation"),
	)

	// Validate retry settings
	maxRetries := viper.GetInt("MAX_RETRIES")
	if maxRetries < 0 {
		return nil, errors.New(errors.ErrConfigInvalid, "invalid MAX_RETRIES",
			map[string]interface{}{
				"config_key": "MAX_RETRIES",
				"value":      maxRetries,
			}, nil)
	}
	logger.Info("Max retries configured",
		zap.Int("retries", maxRetries),
		zap.String("operation", "config_validation"),
	)

	retryDelay := viper.GetInt("RETRY_DELAY_SECONDS")
	if retryDelay <= 0 {
		return nil, errors.New(errors.ErrConfigInvalid, "invalid RETRY_DELAY_SECONDS",
			map[string]interface{}{
				"config_key": "RETRY_DELAY_SECONDS",
				"value":      retryDelay,
			}, nil)
	}
	logger.Info("Retry delay configured",
		zap.Int("seconds", retryDelay),
		zap.String("operation", "config_validation"),
	)

	comparisonTimeout := viper.GetInt("COMPARISON_TIMEOUT_SECONDS")
	if comparisonTimeout <= 0 {
		return nil, errors.New(errors.ErrConfigInvalid, "invalid COMPARISON_TIMEOUT_SECONDS",
			map[string]interface{}{
				"config_key": "COMPARISON_TIMEOUT_SECONDS",
				"value":      comparisonTimeout,
			}, nil)
	}
	logger.Info("Comparison timeout configured",
		zap.Int("seconds", comparisonTimeout),
		zap.String("operation", "config_validation"),
	)

	config := &Config{
		TFStatePath:       tfStatePath,
		MainTFPath:        mainTFPath,
		CheckInterval:     interval,
		AWSRegion:         viper.GetString("AWS_REGION"),
		AccessSecret:      viper.GetString("AWS_SECRET_ACCESS_KEY"),
		AcessKeyID:        viper.GetString("AWS_ACCESS_KEY_ID"),
		LogLevel:          viper.GetString("LOG_LEVEL"),
		MaxRetries:        maxRetries,
		RetryDelay:        retryDelay,
		ComparisonTimeout: comparisonTimeout,
	}

	logger.Info("Configuration loaded successfully",
		zap.String("operation", "config_complete"),
	)
	return config, nil
}
