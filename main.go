// main.go

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v50/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// Define global color functions for consistent terminal output
var (
	red    = color.New(color.FgRed).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	blue   = color.New(color.FgBlue).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	bold   = color.New(color.Bold).SprintFunc()
)

// Strategy interface defines the Execute method for different CLI commands
type Strategy interface {
	Execute() error
}

// AddSecretStrategy defines the parameters for adding a secret
type AddSecretStrategy struct {
	Token       string
	Repo        string // Format: "owner/repo"
	SecretName  string
	SecretValue string
	Encryptor   Encryptor
}

// AddWorkflowStrategy defines the parameters for adding a workflow
type AddWorkflowStrategy struct {
	Token        string
	Repo         string // Format: "owner/repo"
	WorkflowName string // e.g., "ci.yml"
	Content      string // YAML content of the workflow
}

// StoreConfigStrategy defines the parameters for storing a config key-value pair
type StoreConfigStrategy struct {
	ConfigKey   string
	ConfigValue string
}

// Execute adds a secret to a GitHub repository
func (a *AddSecretStrategy) Execute() error {
	ctx := context.Background()

	// Initialize GitHub client with OAuth2 token
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: a.Token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Split repo into owner and repo
	parts := strings.Split(a.Repo, "/")
	if len(parts) != 2 {
		fmt.Fprintln(os.Stderr, red("Invalid repository format. Use 'owner/repo'."))
		return fmt.Errorf("invalid repository format")
	}
	owner, repo := parts[0], parts[1]

	// Fetch repository public key
	publicKey, _, err := client.Actions.GetRepoPublicKey(ctx, owner, repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error fetching repository public key:"), err)
		return err
	}

	// Encrypt the secret value
	encryptedValue, err := a.Encryptor.Encrypt(a.SecretValue, publicKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error encrypting secret:"), err)
		return err
	}

	// Check if KeyID is not nil
	if publicKey.KeyID == nil {
		fmt.Fprintln(os.Stderr, red("Public Key ID is nil."))
		return fmt.Errorf("public key ID is nil")
	}

	// Create or update the secret
	secret := &github.EncryptedSecret{
		EncryptedValue: &encryptedValue,
		KeyID:          *publicKey.KeyID, // Dereference the pointer
	}

	// Correct number of arguments: ctx, owner, repo, secretName, secret
	_, _, err = client.Actions.CreateOrUpdateRepoSecret(ctx, owner, repo, a.SecretName, secret)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error setting repository secret:"), err)
		return err
	}

	fmt.Println(green("Secret '%s' added to repository '%s' successfully.", a.SecretName, a.Repo))
	fmt.Println(yellow("Secret '%s' saved locally.", a.SecretName))

	// Save secret locally for persistence
	saveSecretLocally(a.SecretName, a.SecretValue)

	return nil
}

// Execute adds a GitHub Actions workflow to a repository
func (a *AddWorkflowStrategy) Execute() error {
	ctx := context.Background()

	// Split repo into owner and repo
	parts := strings.Split(a.Repo, "/")
	if len(parts) != 2 {
		fmt.Fprintln(os.Stderr, red("Invalid repository format. Use 'owner/repo'."))
		return fmt.Errorf("invalid repository format")
	}
	owner, repo := parts[0], parts[1]

	// GitHub repository URL
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)

	// Initialize authentication for git operations
	auth := &http.BasicAuth{
		Username: "ghm",     // Can be anything except an empty string
		Password: a.Token,   // GitHub Personal Access Token
	}

	// Clone the repository into a temporary directory
	tmpDir, err := ioutil.TempDir("", "ghm-repo-")
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error creating temporary directory:"), err)
		return err
	}
	defer os.RemoveAll(tmpDir) // Clean up after cloning

	fmt.Println(cyan("Cloning repository into temporary directory..."))

	repoGit, err := git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
		Auth:     auth,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error cloning repository:"), err)
		return err
	}

	worktree, err := repoGit.Worktree()
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error accessing worktree:"), err)
		return err
	}

	// Create the workflow file in .github/workflows/
	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	err = os.MkdirAll(workflowDir, os.ModePerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error creating workflow directory:"), err)
		return err
	}

	workflowPath := filepath.Join(workflowDir, a.WorkflowName)
	err = ioutil.WriteFile(workflowPath, []byte(a.Content), 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error writing workflow file:"), err)
		return err
	}

	// Stage the workflow file
	_, err = worktree.Add(filepath.Join(".github", "workflows", a.WorkflowName))
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error adding workflow file to git:"), err)
		return err
	}

	// Commit the changes
	commitMsg := "Add GitHub Actions workflow"
	commit, err := worktree.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "ghm",
			Email: "ghm@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error committing changes:"), err)
		return err
	}

	obj, err := repoGit.CommitObject(commit)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error getting commit object:"), err)
		return err
	}

	fmt.Println(green("Committed changes: %s", obj.Hash))

	// Push the commit to GitHub
	fmt.Println(cyan("Pushing changes to GitHub..."))
	err = repoGit.Push(&git.PushOptions{
		Auth: auth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		fmt.Fprintln(os.Stderr, red("Error pushing changes:"), err)
		return err
	}

	fmt.Println(green("Workflow '%s' added to repository '%s' successfully.", a.WorkflowName, a.Repo))

	return nil
}

// Execute stores a configuration key-value pair
func (s *StoreConfigStrategy) Execute() error {
	viper.Set(s.ConfigKey, s.ConfigValue)
	err := viper.WriteConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file doesn't exist, create it
			err = viper.SafeWriteConfig()
			if err != nil {
				fmt.Fprintln(os.Stderr, red("Error creating config file:"), err)
				return err
			}
		} else {
			fmt.Fprintln(os.Stderr, red("Error writing config:"), err)
			return err
		}
	}
	fmt.Println(yellow("Configuration '%s' saved successfully.", s.ConfigKey))
	return nil
}

// Initialize Add Secret Command
func initAddSecretCmd(token *string) *cobra.Command {
	var repo, secretName, secretValue string

	addSecretCmd := &cobra.Command{
		Use:   "add-secret",
		Short: "Add a secret to a GitHub repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repo == "" {
				fmt.Fprintln(os.Stderr, red("Repository must be specified."))
				return fmt.Errorf("repository not specified")
			}
			if !strings.Contains(repo, "/") {
				fmt.Fprintln(os.Stderr, red("Invalid repository format. Use 'owner/repo'."))
				return fmt.Errorf("invalid repository format")
			}
			if secretName == "" {
				fmt.Fprintln(os.Stderr, red("Secret name must be provided."))
				return fmt.Errorf("secret name not provided")
			}
			if secretValue == "" {
				fmt.Fprint(os.Stdout, blue("Enter the secret value: "))
				reader := bufio.NewReader(os.Stdin)
				secretValueInput, err := reader.ReadString('\n')
				if err != nil {
					fmt.Fprintln(os.Stderr, red("Error reading secret value:"), err)
					return err
				}
				secretValue = strings.TrimSpace(secretValueInput)
			}
			strategy := &AddSecretStrategy{
				Token:       *token,
				Repo:        repo,
				SecretName:  secretName,
				SecretValue: secretValue,
				Encryptor:   &EncryptorImpl{},
			}
			return strategy.Execute()
		},
	}

	addSecretCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository name in 'owner/repo' format")
	addSecretCmd.Flags().StringVarP(&secretName, "name", "n", "", "Name of the secret")
	addSecretCmd.Flags().StringVarP(&secretValue, "value", "v", "", "Value of the secret")

	return addSecretCmd
}

// Initialize Add Workflow Command
func initAddWorkflowCmd(token *string) *cobra.Command {
	var repo, workflowName, workflowContent, workflowFile string

	addWorkflowCmd := &cobra.Command{
		Use:   "add-workflow",
		Short: "Add a GitHub Actions workflow to a repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repo == "" {
				fmt.Fprintln(os.Stderr, red("Repository must be specified."))
				return fmt.Errorf("repository not specified")
			}
			if !strings.Contains(repo, "/") {
				fmt.Fprintln(os.Stderr, red("Invalid repository format. Use 'owner/repo'."))
				return fmt.Errorf("invalid repository format")
			}
			if workflowName == "" {
				fmt.Fprintln(os.Stderr, red("Workflow name must be provided."))
				return fmt.Errorf("workflow name not provided")
			}
			if workflowContent == "" && workflowFile == "" {
				fmt.Fprintln(os.Stderr, red("Either workflow content or workflow file must be provided."))
				return fmt.Errorf("workflow content or file not provided")
			}
			if workflowContent == "" && workflowFile != "" {
				contentBytes, err := ioutil.ReadFile(workflowFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, red("Error reading workflow file '%s': %s\n"), workflowFile, err)
					return err
				}
				workflowContent = string(contentBytes)
			}
			strategy := &AddWorkflowStrategy{
				Token:        *token,
				Repo:         repo,
				WorkflowName: workflowName,
				Content:      workflowContent,
			}
			return strategy.Execute()
		},
	}

	addWorkflowCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository name in 'owner/repo' format")
	addWorkflowCmd.Flags().StringVarP(&workflowName, "name", "n", "", "Name of the workflow file (e.g., ci.yml)")
	addWorkflowCmd.Flags().StringVarP(&workflowContent, "content", "c", "", "Content of the workflow file")
	addWorkflowCmd.Flags().StringVarP(&workflowFile, "file", "f", "", "Path to the workflow file to read content from")

	return addWorkflowCmd
}

// Initialize Store Config Command
func initStoreConfigCmd() *cobra.Command {
	var configKey, configValue string

	storeConfigCmd := &cobra.Command{
		Use:   "store-config",
		Short: "Store a configuration key-value pair",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configKey == "" {
				fmt.Fprintln(os.Stderr, red("Configuration key must be provided."))
				return fmt.Errorf("configuration key not provided")
			}
			if configValue == "" {
				fmt.Fprintln(os.Stderr, red("Configuration value must be provided."))
				return fmt.Errorf("configuration value not provided")
			}
			strategy := &StoreConfigStrategy{
				ConfigKey:   configKey,
				ConfigValue: configValue,
			}
			return strategy.Execute()
		},
	}

	storeConfigCmd.Flags().StringVarP(&configKey, "key", "k", "", "Configuration key")
	storeConfigCmd.Flags().StringVarP(&configValue, "value", "v", "", "Configuration value")

	return storeConfigCmd
}

// saveSecretLocally saves the secret in a local JSON file for persistence
func saveSecretLocally(secretName, secretValue string) {
	secretsFile := "secrets.json"
	secrets := make(map[string]string)

	// Check if secrets.json exists
	if _, err := os.Stat(secretsFile); err == nil {
		file, err := os.Open(secretsFile)
		if err == nil {
			defer file.Close()
			json.NewDecoder(file).Decode(&secrets)
		}
	}

	secrets[secretName] = secretValue

	file, err := os.Create(secretsFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error saving secret locally:"), err)
		return
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(secrets)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("Error encoding secrets:"), err)
		return
	}

	fmt.Println(yellow("Secret '%s' saved locally.", secretName))
}

// printASCIIHeader prints a colorful ASCII header
func printASCIIHeader() {
	header := `
          ____ _     _  __  __ 
         / ___| |__ (_)|  \/  |
        | |  _| '_ \| || |\/| |
        | |_| | | | | || |  | |
         \____|_| |_|_||_|  |_| 
                                  
        `
	color.New(color.FgCyan).Println(header)
	color.New(color.FgMagenta).Println("    GitHub Management CLI (ghm)")
	color.New(color.FgCyan).Println("=======================================")
}

// initConfig initializes Viper configuration
func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := viper.SafeWriteConfig(); err != nil {
				fmt.Fprintln(os.Stderr, red("Error creating config file:"), err)
				os.Exit(1)
			}
			fmt.Println(yellow("Config file created: config.json"))
		} else {
			fmt.Fprintln(os.Stderr, red("Error reading config file:"), err)
			os.Exit(1)
		}
	}
}

// Initialize Root Command
func initRootCmd() *cobra.Command {
	var token string

	rootCmd := &cobra.Command{
		Use:   "ghm",
		Short: "GitHub Management CLI",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Ensure GitHub token is available
			token = viper.GetString("github_token")
			if token == "" {
				fmt.Fprint(os.Stdout, blue("Enter your GitHub token: "))
				reader := bufio.NewReader(os.Stdin)
				byteToken, err := reader.ReadString('\n')
				if err != nil {
					fmt.Fprintln(os.Stderr, red("Error reading token:"), err)
					os.Exit(1)
				}
				token = strings.TrimSpace(string(byteToken))
				viper.Set("github_token", token)
				if err := viper.WriteConfig(); err != nil {
					if _, ok := err.(viper.ConfigFileNotFoundError); ok {
						if err := viper.SafeWriteConfig(); err != nil {
							fmt.Fprintln(os.Stderr, red("Error creating config file:"), err)
							os.Exit(1)
						}
					} else {
						fmt.Fprintln(os.Stderr, red("Error writing config:"), err)
						os.Exit(1)
					}
				}
			}
		},
	}

	// Add subcommands
	rootCmd.AddCommand(initAddSecretCmd(&token))
	rootCmd.AddCommand(initAddWorkflowCmd(&token))
	rootCmd.AddCommand(initStoreConfigCmd())

	return rootCmd
}

func main() {
	// Conditionally print the ASCII header
	if os.Getenv("TESTING") != "1" {
		printASCIIHeader()
	}

	// Initialize Viper config
	initConfig()

	// Initialize and execute the root command
	rootCmd := initRootCmd()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, red("Error:"), err)
		os.Exit(1)
	}
}