package teraform

import (
	"encoding/json"
	"fmt"
	"os"

	"Savannahtakehomeassi/teraform/models"
)

// ParseTerraformInstance parses the Terraform state file for an EC2 instance
func ParseTerraformInstance(filePath string) (*models.TerraformState, error) {
	// Read the Terraform state file
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	//fmt.Println(string(file))
	var tfState models.TerraformState
	if err := json.Unmarshal(file, &tfState); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	return &tfState, nil
}
