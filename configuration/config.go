package configuration

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	TFStatePath       string
	MainTFPath        string
	CheckInterval     time.Duration
	AWSRegion         string
	LogLevel          string
	MaxRetries        int
	RetryDelay        time.Duration
	ComparisonTimeout time.Duration
}

// Initialize sets up the configuration system
func Initialize() (*Config, error) {
	// Set default values
	viper.SetDefault("TFSTATE_PATH", "terraform.tfstate")
	viper.SetDefault("MAINTF_PATH", "main.tf")
	viper.SetDefault("CHECK_INTERVAL_MINUTES", 5)
	viper.SetDefault("AWS_REGION", "us-east-1")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("MAX_RETRIES", 3)
	viper.SetDefault("RETRY_DELAY_SECONDS", 5)
	viper.SetDefault("COMPARISON_TIMEOUT_SECONDS", 30)

	// Read from config file if it exists

	viper.SetConfigFile(".env") // Specify the .env file
	// Read the .env file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Validate paths
	tfStatePath := viper.GetString("TFSTATE_PATH")
	if tfStatePath == "" {
		return nil, errors.New("invalid TFSTATE_PATH")
	}

	mainTFPath := viper.GetString("MAINTF_PATH")
	if mainTFPath == "" {
		return nil, errors.New("invalid MAINTF_PATH")
	}

	// Validate interval
	interval := viper.GetInt("CHECK_INTERVAL_SECONDS")
	if interval <= 0 {
		return nil, fmt.Errorf("invalid CHECK_INTERVAL_SECONDS: must be positive")
	}

	// Validate retry settings
	maxRetries := viper.GetInt("MAX_RETRIES")
	if maxRetries < 0 {
		return nil, fmt.Errorf("invalid MAX_RETRIES: must be non-negative")
	}

	retryDelay := viper.GetInt("RETRY_DELAY_SECONDS")
	if retryDelay <= 0 {
		return nil, fmt.Errorf("invalid RETRY_DELAY_SECONDS: must be positive")
	}

	comparisonTimeout := viper.GetInt("COMPARISON_TIMEOUT_SECONDS")
	if comparisonTimeout <= 0 {
		return nil, fmt.Errorf("invalid COMPARISON_TIMEOUT_SECONDS: must be positive")
	}

	return &Config{
		TFStatePath:       tfStatePath,
		MainTFPath:        mainTFPath,
		CheckInterval:     time.Duration(interval) * time.Second,
		AWSRegion:         viper.GetString("AWS_REGION"),
		LogLevel:          viper.GetString("LOG_LEVEL"),
		MaxRetries:        maxRetries,
		RetryDelay:        time.Duration(retryDelay) * time.Second,
		ComparisonTimeout: time.Duration(comparisonTimeout) * time.Second,
	}, nil
}
