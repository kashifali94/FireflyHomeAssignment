package main

import (
	"fmt"
	"log"

	"github.com/spf13/viper"

	"Savannahtakehomeassi/awsd"
	"Savannahtakehomeassi/configuration"
	"Savannahtakehomeassi/teraform"
	"Savannahtakehomeassi/utils"
)

func main() {
	configuration.Initialize()
	tfPath := viper.GetString("TFSTATE_PATH")

	awsClient, err := awsd.NewEC2Client()
	if err != nil {
		log.Fatalf("unable to create aws client:%v", err)
	}

	awsInst, err := awsd.GetAWSInstance(awsClient)
	if err != nil {
		log.Fatalf("AWS error: %v", err)
	}

	tfInst, err := teraform.ParseTerraformInstance(tfPath)
	if err != nil {
		log.Fatalf("Terraform error: %v", err)
	}

	drift, err := utils.CompareAWSInstanceWithTerraform(awsInst, tfInst)
	if len(drift) == 0 {
		fmt.Println("No drift detected!")
	} else {
		fmt.Println("Drift detected:")
		for _, values := range drift {
			fmt.Println(values)
		}
	}
}
