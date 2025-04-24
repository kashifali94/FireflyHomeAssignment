package configuration

import (
	"log"

	"github.com/spf13/viper"
)

// Intialize will get the values from .env file and use the configuration in the app
func Initialize() {
	// Set up Viper to read from .env
	// it is set according to the docker container env if you want to run it from here
	//you need to change the path to  .env
	viper.SetConfigFile("/app/.env") // Specify the .env file
	err := viper.ReadInConfig()      // Read the .env file
	if err != nil {
		log.Fatalf("Error reading .env file: %v", err)
	}

	// Alternatively, allow Viper to pick up environment variables automatically
	viper.AutomaticEnv()

}
