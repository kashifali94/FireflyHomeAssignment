package teraform

import (
	"Savannahtakehomeassi/logger"
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"Savannahtakehomeassi/teraform/models"
)

// ParseTerraformInstance parses the Terraform state file for an EC2 instance
func ParseTerraformInstance(filePath string) (*models.TerraformState, error) {
	// Read the Terraform state file
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var tfState models.TerraformState
	if err := json.Unmarshal(file, &tfState); err != nil {
		return nil, err
	}

	logger.Info("TerraForm: response is parsed successfully")
	return &tfState, nil
}

func ParseHCLConfig(filename string) (*models.TFInstance, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	logger.Info("Parsed HCL config file successfully")
	return &instance, nil
}
