package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize Logrus logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	// Conditionally print the ASCII header
	if os.Getenv("TESTING") != "1" {
		printASCIIHeader(logger)
	}

	// Initialize Viper config
	initConfig(logger)

	// Initialize and execute the CLI root command
	rootCmd := initRootCmd(logger)

	if err := rootCmd.Execute(); err != nil {
		logger.Errorf("Error: %v", err)
		os.Exit(1)
	}
}