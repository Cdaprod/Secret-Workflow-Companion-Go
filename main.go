// main.go
package main

import (
	"os"
	"sync"
	"flag"
	"fmt"

	"github.com/sirupsen/logrus"
	//"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/manifoldco/promptui"
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
		HeaderColor.Println(header) // Print the header with color
	})
}

// GetGitHubToken prompts the user for their GitHub token and saves it in the config
func GetGitHubToken(logger *logrus.Logger) (string, error) {
    prompt := promptui.Prompt{
        Label: "Enter your GitHub token",
        Mask: '*', // Mask input with asterisks
    }
    
    token, err := prompt.Run()
    if err != nil {
        logger.Errorf("Error reading GitHub token: %v", err)
        return "", err
    }

    // Store the token in the config
    storeConfig := &StoreConfigStrategy{
        ConfigKey:   "github_token",
        ConfigValue: token,
        Logger:      logger,
    }

    if err := storeConfig.Execute(); err != nil {
        logger.Errorf("Error storing GitHub token: %v", err)
        return "", err
    }

    return token, nil
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

    // Get GitHub token
    githubToken, err := GetGitHubToken(logger)
    if err != nil {
        logger.Fatalf("Failed to get GitHub token: %v", err)
    }

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

    // Now you can use githubToken where needed
    logger.Infof("GitHub token obtained: %s", githubToken) // Use the token as needed
}