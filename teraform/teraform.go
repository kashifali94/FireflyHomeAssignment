package teraform

import (
	"Savannahtakehomeassi/errors"
	"Savannahtakehomeassi/teraform/models"
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

const (
	packageName = "teraform"
)

// TerraformClient represents a client for Terraform operations
type TerraformClient struct{}

// NewTerraformClient creates a new Terraform client
func NewTerraformClient() *TerraformClient {
	logger := zap.L().With(
		zap.String("package", packageName),
		zap.String("function", "NewTerraformClient"),
	)

	logger.Info("Terraform client created successfully")
	return &TerraformClient{}
}

// ParseTerraformInstance parses the Terraform state file for an EC2 instance
func (c *TerraformClient) ParseTerraformInstance(filePath string) (*models.TerraformState, error) {
	logger := zap.L().With(
		zap.String("package", packageName),
		zap.String("function", "ParseTerraformInstance"),
		zap.String("file_path", filePath),
	)

	// Read the Terraform state file
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.New(errors.ErrTerraformState, "failed to read terraform state file",
			map[string]interface{}{
				"operation": "file_read",
				"file_path": filePath,
			}, err)
	}

	var tfState models.TerraformState
	if err := json.Unmarshal(file, &tfState); err != nil {
		return nil, errors.New(errors.ErrTerraformState, "failed to parse terraform state",
			map[string]interface{}{
				"operation": "json_unmarshal",
				"file_path": filePath,
			}, err)
	}

	logger.Info("Terraform state parsed successfully",
		zap.String("operation", "state_parse"),
	)
	return &tfState, nil
}

// ParseHCLConfig parses the HCL configuration file
func (c *TerraformClient) ParseHCLConfig(filename string) (*models.TFInstance, error) {
	logger := zap.L().With(
		zap.String("package", packageName),
		zap.String("function", "ParseHCLConfig"),
		zap.String("file_path", filename),
	)

	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.New(errors.ErrTerraformConfig, "failed to open HCL config file",
			map[string]interface{}{
				"operation": "file_open",
				"file_path": filename,
			}, err)
	}
	defer file.Close()

	var instance models.TFInstance
	instance.Tags = make(map[string]string)

	scanner := bufio.NewScanner(file)
	var insideResource, insideTags bool

	reKV := regexp.MustCompile(`^\s*(\w+)\s*=\s*["']?([^"']+)["']?`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and blank lines
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Remove inline comments
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		if strings.HasPrefix(line, "resource") && strings.Contains(line, `"aws_instance"`) {
			insideResource = true
			continue
		}

		if insideResource && strings.HasPrefix(line, "tags") && strings.Contains(line, "{") {
			insideTags = true
			continue
		}

		if insideTags {
			if strings.HasPrefix(line, "}") {
				insideTags = false
				continue
			}
			if match := reKV.FindStringSubmatch(line); len(match) == 3 {
				instance.Tags[match[1]] = match[2]
			}
			continue
		}

		if insideResource {
			if strings.HasPrefix(line, "}") {
				insideResource = false
				continue
			}
			if match := reKV.FindStringSubmatch(line); len(match) == 3 {
				key, val := match[1], match[2]
				switch key {
				case "ami":
					instance.AMI = val
				case "instance_type":
					instance.InstanceType = val
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.New(errors.ErrTerraformConfig, "failed to scan HCL config file",
			map[string]interface{}{
				"operation": "file_scan",
				"file_path": filename,
			}, err)
	}

	logger.Info("HCL config parsed successfully",
		zap.String("operation", "config_parse"),
		zap.String("ami", instance.AMI),
		zap.String("instance_type", instance.InstanceType),
	)
	return &instance, nil
}
