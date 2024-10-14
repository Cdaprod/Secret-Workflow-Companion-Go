// main.go
package main

import (
	"os"
	"sync"
	"fmt"

	"github.com/sirupsen/logrus"
	//"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Mutex to ensure thread safety in case the program is multithreaded
var once sync.Once

// printASCIIHeader prints a colorful ASCII header only once per run
func printASCIIHeader() {
	header := `
╔────────────────────────────────────────────────────────────────────────────╗
│ ██████╗██████╗  █████╗ ██████╗ ██████╗  ██████╗ ██████╗                    │
│██╔════╝██╔══██╗██╔══██╗██╔══██╗██╔══██╗██╔═══██╗██╔══██╗                   │
│██║     ██║  ██║███████║██████╔╝██████╔╝██║   ██║██║  ██║                   │
│██║     ██║  ██║██╔══██║██╔═══╝ ██╔══██╗██║   ██║██║  ██║                   │
│╚██████╗██████╔╝██║  ██║██║     ██║  ██║╚██████╔╝██████╔╝                   │
│ ╚═════╝╚═════╝ ╚═╝  ╚═╝╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚═════╝                    │
│                                                                            │
│ ██████╗ ██╗████████╗██╗  ██╗██╗   ██╗██████╗                               │
│██╔════╝ ██║╚══██╔══╝██║  ██║██║   ██║██╔══██╗                              │
│██║  ███╗██║   ██║   ███████║██║   ██║██████╔╝                              │
│██║   ██║██║   ██║   ██╔══██║██║   ██║██╔══██╗                              │
│╚██████╔╝██║   ██║   ██║  ██║╚██████╔╝██████╔╝                              │
│ ╚═════╝ ╚═╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝ ╚═════╝                               │
│                                                                            │
│ ██████╗ ██████╗ ███╗   ███╗██████╗  █████╗ ███╗   ██╗██╗ ██████╗ ███╗   ██╗│
│██╔════╝██╔═══██╗████╗ ████║██╔══██╗██╔══██╗████╗  ██║██║██╔═══██╗████╗  ██║│
│██║     ██║   ██║██╔████╔██║██████╔╝███████║██╔██╗ ██║██║██║   ██║██╔██╗ ██║│
│██║     ██║   ██║██║╚██╔╝██║██╔═══╝ ██╔══██║██║╚██╗██║██║██║   ██║██║╚██╗██║│
│╚██████╗╚██████╔╝██║ ╚═╝ ██║██║     ██║  ██║██║ ╚████║██║╚██████╔╝██║ ╚████║│
│ ╚═════╝ ╚═════╝ ╚═╝     ╚═╝╚═╝     ╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝ ╚═════╝ ╚═╝  ╚═══╝│
╚────────────────────────────────────────────────────────────────────────────╝
`
	once.Do(func() {
		HeaderColor(header) // Print the header with color
	})
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


func main() {
	// Parse flags to check if TUI should run
	runTUIFlag := flag.Bool("tui", false, "Run the TUI (terminal user interface)")
	flag.Parse()

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

	// Check if TUI should be launched
	if *runTUIFlag {
		// Run the TUI if the flag is set
		runTUI(logger)
	} else {
		// Initialize and run the default CLI command
		rootCmd := initRootCmd(logger)
		if err := rootCmd.Execute(); err != nil {
			logger.Fatalf("Error executing command: %v", err)
		}
	}
}