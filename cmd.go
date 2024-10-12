// cmd.go
package main

import (
	"context"
	"encoding/json" // Added to resolve undefined: json
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// Initialize Root Command
func initRootCmd(logger *logrus.Logger) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ghm",
		Short: "GitHub Management CLI",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Ensure GitHub token is available
			token := viper.GetString("github_token")
			if token == "" {
				fmt.Print("Enter your GitHub token: ")
				byteToken, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println() // Move to the next line after input
				if err != nil {
					logger.Fatalf("Error reading token: %v", err)
				}
				token = strings.TrimSpace(string(byteToken))
				viper.Set("github_token", token)
				if err := viper.WriteConfig(); err != nil {
					if _, ok := err.(viper.ConfigFileNotFoundError); ok {
						if err := viper.SafeWriteConfig(); err != nil {
							logger.Fatalf("Error creating config file: %v", err)
						}
					} else {
						logger.Fatalf("Error writing config: %v", err)
					}
				}
				logger.Info("GitHub token saved successfully.")
			} else {
				logger.Info("GitHub token loaded from config.")
			}
		},
	}

	// Add subcommands
	rootCmd.AddCommand(initAddSecretCmd(logger))
	rootCmd.AddCommand(initAddWorkflowCmd(logger))
	rootCmd.AddCommand(initStoreConfigCmd(logger))
	rootCmd.AddCommand(initAddSavedSecretCmd(logger))
	rootCmd.AddCommand(initAddSavedWorkflowCmd(logger))
	rootCmd.AddCommand(initListReposCmd(logger))

	return rootCmd
}

// Initialize Add Secret Command
func initAddSecretCmd(logger *logrus.Logger) *cobra.Command {
	var repo, secretName, secretValue string

	addSecretCmd := &cobra.Command{
		Use:   "add-secret",
		Short: "Add a secret to a GitHub repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repo == "" {
				logger.Error("Repository must be specified.")
				return fmt.Errorf("repository not specified")
			}
			if !strings.Contains(repo, "/") {
				logger.Error("Invalid repository format. Use 'owner/repo'.")
				return fmt.Errorf("invalid repository format")
			}
			if secretName == "" {
				logger.Error("Secret name must be provided.")
				return fmt.Errorf("secret name not provided")
			}
			if secretValue == "" {
				fmt.Print("Enter the secret value: ")
				byteSecret, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println() // Move to the next line after input
				if err != nil {
					logger.Errorf("Error reading secret value: %v", err)
					return err
				}
				secretValue = strings.TrimSpace(string(byteSecret))
			}

			ghm := NewGHM(viper.GetString("github_token"), logger) // Pass both arguments
			return ghm.AddSecret(context.Background(), repo, secretName, secretValue)
		},
	}

	addSecretCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository name in 'owner/repo' format")
	addSecretCmd.Flags().StringVarP(&secretName, "name", "n", "", "Name of the secret")
	addSecretCmd.Flags().StringVarP(&secretValue, "value", "v", "", "Value of the secret")

	return addSecretCmd
}

// Initialize Add Workflow Command
func initAddWorkflowCmd(logger *logrus.Logger) *cobra.Command {
	var repo, workflowName, workflowContent, workflowFile string

	addWorkflowCmd := &cobra.Command{
		Use:   "add-workflow",
		Short: "Add a GitHub Actions workflow to a repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repo == "" {
				logger.Error("Repository must be specified.")
				return fmt.Errorf("repository not specified")
			}
			if !strings.Contains(repo, "/") {
				logger.Error("Invalid repository format. Use 'owner/repo'.")
				return fmt.Errorf("invalid repository format")
			}
			if workflowName == "" {
				logger.Error("Workflow name must be provided.")
				return fmt.Errorf("workflow name not provided")
			}
			if workflowContent == "" && workflowFile == "" {
				logger.Error("Either workflow content or workflow file must be provided.")
				return fmt.Errorf("workflow content or file not provided")
			}
			if workflowContent == "" && workflowFile != "" {
				contentBytes, err := ioutil.ReadFile(workflowFile)
				if err != nil {
					logger.Errorf("Error reading workflow file '%s': %v", workflowFile, err)
					return err
				}
				workflowContent = string(contentBytes)
			}

			ghm := NewGHM(viper.GetString("github_token"), logger) // Pass both arguments
			return ghm.AddWorkflow(context.Background(), repo, workflowName, workflowContent)
		},
	}

	addWorkflowCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository name in 'owner/repo' format")
	addWorkflowCmd.Flags().StringVarP(&workflowName, "name", "n", "", "Name of the workflow file (e.g., ci.yml)")
	addWorkflowCmd.Flags().StringVarP(&workflowContent, "content", "c", "", "Content of the workflow file")
	addWorkflowCmd.Flags().StringVarP(&workflowFile, "file", "f", "", "Path to the workflow file to read content from")

	return addWorkflowCmd
}

// Initialize Store Config Command
func initStoreConfigCmd(logger *logrus.Logger) *cobra.Command {
	var configKey, configValue string

	storeConfigCmd := &cobra.Command{
		Use:   "store-config",
		Short: "Store a configuration key-value pair",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configKey == "" {
				logger.Error("Configuration key must be provided.")
				return fmt.Errorf("configuration key not provided")
			}
			if configValue == "" {
				logger.Error("Configuration value must be provided.")
				return fmt.Errorf("configuration value not provided")
			}
			ghm := NewGHM(viper.GetString("github_token"), logger) // Pass both arguments
			return ghm.StoreConfig(context.Background(), configKey, configValue)
		},
	}

	storeConfigCmd.Flags().StringVarP(&configKey, "key", "k", "", "Configuration key")
	storeConfigCmd.Flags().StringVarP(&configValue, "value", "v", "", "Configuration value")

	return storeConfigCmd
}

// Initialize Add Saved Secret Command
func initAddSavedSecretCmd(logger *logrus.Logger) *cobra.Command {
	var targetRepo string

	addSavedSecretCmd := &cobra.Command{
		Use:   "add-saved-secrets",
		Short: "Interactively add saved secrets to a target GitHub repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetRepo == "" {
				logger.Error("Target repository must be specified.")
				return fmt.Errorf("target repository not specified")
			}
			if !strings.Contains(targetRepo, "/") {
				logger.Error("Invalid repository format. Use 'owner/repo'.")
				return fmt.Errorf("invalid repository format")
			}

			// Load saved secrets
			secrets, err := loadSavedSecrets(logger)
			if err != nil {
				logger.Errorf("Error loading saved secrets: %v", err)
				return err
			}
			if len(secrets) == 0 {
				logger.Info("No saved secrets found.")
				return nil
			}

			// Interactive selection
			selectedSecrets, err := promptSelectItems("Select Secrets to Add", secrets)
			if err != nil {
				logger.Errorf("Error selecting secrets: %v", err)
				return err
			}
			if len(selectedSecrets) == 0 {
				logger.Info("No secrets selected.")
				return nil
			}

			// Load repos.json
			reposConfig, err := LoadReposConfig(logger)
			if err != nil {
				logger.Errorf("Error loading repos config: %v", err)
				return err
			}

			// Add selected secrets to the target repository
			ghm := NewGHM(viper.GetString("github_token"), logger) // Pass both arguments
			err = ghm.AddSecretsToRepo(context.Background(), targetRepo, selectedSecrets, reposConfig)
			if err != nil {
				logger.Errorf("Error adding secrets to repository: %v", err)
				return err
			}

			// Save repos.json
			err = SaveReposConfig(reposConfig, logger)
			if err != nil {
				logger.Errorf("Error saving repos config: %v", err)
				return err
			}

			return nil
		},
	}

	addSavedSecretCmd.Flags().StringVarP(&targetRepo, "repo", "r", "", "Target repository in 'owner/repo' format")

	return addSavedSecretCmd
}

// Initialize Add Saved Workflow Command
func initAddSavedWorkflowCmd(logger *logrus.Logger) *cobra.Command {
	var targetRepo string

	addSavedWorkflowCmd := &cobra.Command{
		Use:   "add-saved-workflows",
		Short: "Interactively add saved workflows to a target GitHub repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetRepo == "" {
				logger.Error("Target repository must be specified.")
				return fmt.Errorf("target repository not specified")
			}
			if !strings.Contains(targetRepo, "/") {
				logger.Error("Invalid repository format. Use 'owner/repo'.")
				return fmt.Errorf("invalid repository format")
			}

			// Load saved workflows
			workflows, err := loadSavedWorkflows(logger)
			if err != nil {
				logger.Errorf("Error loading saved workflows: %v", err)
				return err
			}
			if len(workflows) == 0 {
				logger.Info("No saved workflows found.")
				return nil
			}

			// Interactive selection
			selectedWorkflows, err := promptSelectItems("Select Workflows to Add", workflows)
			if err != nil {
				logger.Errorf("Error selecting workflows: %v", err)
				return err
			}
			if len(selectedWorkflows) == 0 {
				logger.Info("No workflows selected.")
				return nil
			}

			// Load repos.json
			reposConfig, err := LoadReposConfig(logger)
			if err != nil {
				logger.Errorf("Error loading repos config: %v", err)
				return err
			}

			// Add selected workflows to the target repository
			ghm := NewGHM(viper.GetString("github_token"), logger) // Pass both arguments
			err = ghm.AddWorkflowsToRepo(context.Background(), targetRepo, selectedWorkflows, reposConfig)
			if err != nil {
				logger.Errorf("Error adding workflows to repository: %v", err)
				return err
			}

			// Save repos.json
			err = SaveReposConfig(reposConfig, logger)
			if err != nil {
				logger.Errorf("Error saving repos config: %v", err)
				return err
			}

			return nil
		},
	}

	addSavedWorkflowCmd.Flags().StringVarP(&targetRepo, "repo", "r", "", "Target repository in 'owner/repo' format")

	return addSavedWorkflowCmd
}

// Initialize List Repositories Command
func initListReposCmd(logger *logrus.Logger) *cobra.Command {
	listReposCmd := &cobra.Command{
		Use:   "list-repos",
		Short: "List all repositories and their added secrets/workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			reposConfig, err := LoadReposConfig(logger)
			if err != nil {
				logger.Errorf("Error loading repos config: %v", err)
				return err
			}

			if len(reposConfig.Repositories) == 0 {
				logger.Info("No repositories configured.")
				return nil
			}

			for repo, config := range reposConfig.Repositories {
				fmt.Printf("Repository: %s\n", repo)
				fmt.Printf("  Last Update: %s\n", config.LastUpdate)
				fmt.Printf("  Secrets:\n")
				for _, secret := range config.Secrets {
					fmt.Printf("    - %s\n", secret)
				}
				fmt.Printf("  Workflows:\n")
				for _, workflow := range config.Workflows {
					fmt.Printf("    - %s\n", workflow)
				}
				fmt.Println()
			}

			return nil
		},
	}

	return listReposCmd
}

// loadSavedSecrets loads secrets from secrets.json
func loadSavedSecrets(logger *logrus.Logger) ([]string, error) {
	secretsFile := "secrets.json"
	secrets := make(map[string]string)

	if _, err := os.Stat(secretsFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("secrets.json does not exist")
	}

	file, err := os.Open(secretsFile)
	if err != nil {
		logger.Errorf("Error opening secrets.json: %v", err)
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&secrets)
	if err != nil {
		logger.Errorf("Error decoding secrets.json: %v", err)
		return nil, err
	}

	var secretNames []string
	for name := range secrets {
		secretNames = append(secretNames, name)
	}

	return secretNames, nil
}

// loadSavedWorkflows loads workflows from workflows.json
func loadSavedWorkflows(logger *logrus.Logger) ([]string, error) {
	workflowsFile := "workflows.json"
	workflows := make(map[string]string)

	if _, err := os.Stat(workflowsFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("workflows.json does not exist")
	}

	file, err := os.Open(workflowsFile)
	if err != nil {
		logger.Errorf("Error opening workflows.json: %v", err)
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&workflows)
	if err != nil {
		logger.Errorf("Error decoding workflows.json: %v", err)
		return nil, err
	}

	var workflowNames []string
	for name := range workflows {
		workflowNames = append(workflowNames, name)
	}

	return workflowNames, nil
}

// promptSelectItems presents an interactive menu for selection
func promptSelectItems(label string, items []string) ([]string, error) {
	selectedItems := []string{}

	for {
		prompt := promptui.Select{
			Label: label,
			Items: items,
			Size:  10,
		}

		result, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt || err == promptui.ErrEOF {
				return selectedItems, nil
			}
			return selectedItems, err
		}

		// Confirm selection
		confirmPrompt := promptui.Prompt{
			Label:     fmt.Sprintf("Add '%s'?", items[result]),
			IsConfirm: true,
		}

		confirm, err := confirmPrompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt || err == promptui.ErrEOF {
				return selectedItems, nil
			}
			return selectedItems, err
		}

		if strings.ToLower(confirm) == "y" || confirm == "" {
			selectedItems = append(selectedItems, items[result])
			fmt.Printf("Added '%s' to the selection.\n", items[result])
		}

		// Ask if the user wants to continue selecting
		continuePrompt := promptui.Prompt{
			Label:     "Select another item?",
			IsConfirm: true,
			Default:   "y",
		}

		cont, err := continuePrompt.Run()
		if err != nil || strings.ToLower(cont) != "y" {
			break
		}
	}

	return selectedItems, nil
}