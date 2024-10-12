// main.go
package main

import (
	"os"
	"fmt"

	"github.com/sirupsen/logrus"
	//"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	// Initialize configuration
	initConfig()

	// Print ASCII Header
	printASCIIHeader()

	// Initialize root command
	rootCmd := initRootCmd(logger)

	// Execute root command
	if err := rootCmd.Execute(); err != nil {
		logger.Fatalf("Error executing command: %v", err)
	}
}

// printASCIIHeader prints a colorful ASCII header
func printASCIIHeader() {
	header := `
 _____ _    _  __  __ 
/ ____| |  (_) |  \/  |
| |    | | ___| | \  / |
| |    | |/ / | | |\/| |
| |____|   <| | | |  | |
 \_____|_|\_\_|_|_|  |_|`
	fmt.Println(header)
}

// initConfig initializes the configuration with viper
func initConfig() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Config file not found; using defaults")
		} else {
			fmt.Printf("Error reading config file: %s\n", err)
		}
	}
}