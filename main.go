package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/viper"

	"Savannahtakehomeassi/awsd"
	"Savannahtakehomeassi/configuration"
	"Savannahtakehomeassi/driftChecker"
	"Savannahtakehomeassi/teraform"
)

func main() {
	// Configure logger with timestamps
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Initialize configuration
	configuration.Initialize()

	tfPath := viper.GetString("TFSTATE_PATH")
	maintfPath := viper.GetString("MAINTF_PATH")
	interval := viper.GetInt("CHECK_INTERVAL_MINUTES")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for SIGINT/SIGTERM
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		log.Println("Received shutdown signal. Stopping drift checker...")
		cancel()
	}()

	log.Printf("⏳ Starting drift checker every %d seconds(s)...\n", interval)
	runLoop(ctx, tfPath, maintfPath, interval)
}

func runLoop(ctx context.Context, tfPath, maintfPath string, interval int) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// First run immediately
	runDriftCheck(tfPath, maintfPath)

	for {
		select {
		case <-ctx.Done():
			log.Println("Drift checker shutdown complete.")
			return
		case <-ticker.C:
			runDriftCheck(tfPath, maintfPath)
		}
	}
}

func runDriftCheck(tfPath, maintfPath string) {
	log.Println("Starting drift check...")

	// Initialize AWS client
	awsClient, err := awsd.NewEC2Client()
	if err != nil {
		log.Fatalf("Failed to create AWS client: %v\n", err)
		return
	}

	// Fetch AWS instance
	awsInst, err := awsd.GetAWSInstance(awsClient)
	if err != nil {
		log.Fatalf("Failed to fetch AWS instance: %v\n", err)
		return
	}

	// Parse Terraform state
	tfInst, err := teraform.ParseTerraformInstance(tfPath)
	if err != nil {
		log.Fatalf("Failed to parse Terraform state: %v\n", err)
		return
	}

	// Parse HCL config
	tfinstance, err := teraform.ParseHCLConfig(maintfPath)
	if err != nil {
		log.Fatalf("Failed to parse HCL config: %v\n", err)
		return
	}

	// Create channels for drift comparison results and errors
	driftChannel := make(chan []string, 2) // For drift results
	errorChannel := make(chan error, 2)    // For errors

	// Create a wait group for synchronization
	var wg sync.WaitGroup
	wg.Add(2) // Two drift checks

	// Drift check with Terraform state comparison
	go func() {
		defer wg.Done()
		drift, err := driftChecker.CompareAWSInstanceWithTerraform(awsInst, tfInst)
		if err != nil {
			errorChannel <- err
			return
		}
		driftChannel <- drift
	}()

	// Drift check with Terraform HCL config comparison
	go func() {
		defer wg.Done()
		hclDrift, err := driftChecker.CompareInstances(awsInst, tfinstance)
		if err != nil {
			errorChannel <- err
			return
		}
		driftChannel <- hclDrift
	}()

	// Wait for all drift checks to complete
	wg.Wait()
	close(driftChannel)
	close(errorChannel)

	// Handle errors from drift checks
	for err := range errorChannel {
		log.Printf("Error during drift check: %v\n", err)
	}

	// Collect and print drift results from both channels
	driftResults := make([][]string, 0, 2)
	for drift := range driftChannel {
		driftResults = append(driftResults, drift)
	}

	// Print results for both drift checks
	for _, drift := range driftResults {
		if len(drift) == 1 && drift[0] == "No drift detected between AWS instance and Terraform state." {
			log.Println("No drift detected between AWS and Terraform.")
		} else {
			log.Println("Drift detected between AWS and Terraform:")
			for i, d := range drift {
				log.Printf("  %d. %s\n", i+1, d)
			}
		}
	}

	log.Println("Drift check completed.")
}
