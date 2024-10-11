package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/fatih/color"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "golang.org/x/term"
    "syscall"
)

type Strategy interface {
    Execute()
}

// AddSecretStrategy concrete strategy for adding a GitHub secret
type AddSecretStrategy struct {
    Token       string
    Repo        string
    SecretName  string
    SecretValue string
}

// Execute method for AddSecretStrategy
func (a *AddSecretStrategy) Execute() {
    env := os.Environ()
    env = append(env, fmt.Sprintf("GITHUB_TOKEN=%s", a.Token))

    // Command to add the secret using GitHub CLI
    cmd := exec.Command("gh", "secret", "set", a.SecretName, "--repo", a.Repo, "--body", a.SecretValue)
    cmd.Env = env
    output, err := cmd.CombinedOutput()
    if err != nil {
        color.Red("Error adding secret: %s", err)
        fmt.Println(string(output))
        return
    }
    color.Green("Secret '%s' added to repository '%s' successfully.", a.SecretName, a.Repo)

    // Save secret locally for persistence
    saveSecretLocally(a.SecretName, a.SecretValue)
}

// AddWorkflowStrategy concrete strategy for adding a GitHub Actions workflow file
type AddWorkflowStrategy struct {
    Token        string
    Repo         string
    WorkflowPath string
    Content      string
}

// Execute method for AddWorkflowStrategy
func (a *AddWorkflowStrategy) Execute() {
    repoDir := a.RepoDirectory()

    // Check if current directory is the repository
    cwd, err := os.Getwd()
    if err != nil {
        color.Red("Error getting current working directory: %s", err)
        return
    }

    inRepoDir := false
    if filepath.Base(cwd) == repoDir {
        inRepoDir = true
    }

    if inRepoDir {
        color.Green("Running inside repository directory '%s'.", repoDir)
    } else {
        // Clone the repository if it doesn't exist locally
        if _, err := os.Stat(repoDir); os.IsNotExist(err) {
            cloneCmd := exec.Command("git", "clone", fmt.Sprintf("https://github.com/%s.git", a.Repo), repoDir)
            cloneCmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", a.Token))
            cloneOutput, err := cloneCmd.CombinedOutput()
            if err != nil {
                color.Red("Error cloning repository: %s", err)
                fmt.Println(string(cloneOutput))
                return
            }
            color.Green("Repository '%s' cloned successfully.", a.Repo)
        } else {
            color.Yellow("Repository '%s' already exists locally.", a.Repo)
        }
        // Change directory to the repository
        cwd = repoDir
    }

    // Create the workflow file
    fullPath := filepath.Join(cwd, ".github", "workflows", a.WorkflowPath)
    err = os.MkdirAll(filepath.Dir(fullPath), os.ModePerm)
    if err != nil {
        color.Red("Error creating workflow directory: %s", err)
        return
    }

    err = os.WriteFile(fullPath, []byte(a.Content), 0644)
    if err != nil {
        color.Red("Error writing workflow file: %s", err)
        return
    }

    // Add, commit, and push changes
    gitCommands := [][]string{
        {"git", "add", "."},
        {"git", "commit", "-m", "Add new GitHub Actions workflow"},
        {"git", "push"},
    }

    for _, args := range gitCommands {
        cmd := exec.Command(args[0], args[1:]...)
        cmd.Dir = cwd
        cmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", a.Token))
        output, err := cmd.CombinedOutput()
        if err != nil {
            color.Red("Error running command '%s': %s", strings.Join(args, " "), err)
            fmt.Println(string(output))
            return
        }
    }

    color.Green("Workflow '%s' added to repository '%s' successfully.", a.WorkflowPath, a.Repo)
}

// RepoDirectory returns the local directory name of the repository
func (a *AddWorkflowStrategy) RepoDirectory() string {
    parts := strings.Split(a.Repo, "/")
    return parts[len(parts)-1]
}

// StoreConfigStrategy concrete strategy for storing repository configurations
type StoreConfigStrategy struct {
    ConfigKey   string
    ConfigValue string
}

// Execute method for StoreConfigStrategy
func (s *StoreConfigStrategy) Execute() {
    viper.Set(s.ConfigKey, s.ConfigValue)
    err := viper.WriteConfig()
    if err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            // Config file doesn't exist, create it
            err = viper.SafeWriteConfig()
            if err != nil {
                color.Red("Error creating config file: %s", err)
                return
            }
        } else {
            color.Red("Error writing config: %s", err)
            return
        }
    }
    color.Yellow("Configuration '%s' saved successfully.", s.ConfigKey)
}

// Function to save the secret locally in a JSON file
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
        color.Red("Error saving secret locally: %s", err)
        return
    }
    defer file.Close()
    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ")
    err = encoder.Encode(secrets)
    if err != nil {
        color.Red("Error encoding secrets: %s", err)
        return
    }

    color.Yellow("Secret '%s' saved locally.", secretName)
}

// Main function to handle commands using Cobra
func main() {
    // Initialize Viper
    viper.SetConfigName("config")
    viper.SetConfigType("json")
    viper.AddConfigPath(".")
    viper.AutomaticEnv()

    // Read in existing config or create a new one
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            // Config file not found; create it
            err = viper.SafeWriteConfig()
            if err != nil {
                color.Red("Error creating config file: %s", err)
                return
            }
            color.Yellow("Config file created: config.json")
        } else {
            color.Red("Error reading config file: %s", err)
            return
        }
    }

    var token string

    rootCmd := &cobra.Command{
        Use:   "ghm",
        Short: "GitHub Management CLI",
        PersistentPreRun: func(cmd *cobra.Command, args []string) {
            // Ensure GitHub token is available
            token = viper.GetString("github_token")
            if token == "" {
                color.Blue("Enter your GitHub token:")
                byteToken, err := term.ReadPassword(int(syscall.Stdin))
                fmt.Println()
                if err != nil {
                    color.Red("Error reading token: %s", err)
                    os.Exit(1)
                }
                token = strings.TrimSpace(string(byteToken))
                viper.Set("github_token", token)
                err = viper.WriteConfig()
                if err != nil {
                    if _, ok := err.(viper.ConfigFileNotFoundError); ok {
                        err = viper.SafeWriteConfig()
                        if err != nil {
                            color.Red("Error creating config file: %s", err)
                            os.Exit(1)
                        }
                    } else {
                        color.Red("Error writing config: %s", err)
                        os.Exit(1)
                    }
                }
            }
        },
    }

    // Add Secret Command
    var repo string
    var secretName string
    var secretValue string

    addSecretCmd := &cobra.Command{
        Use:   "add-secret",
        Short: "Add a secret to a GitHub repository",
        Run: func(cmd *cobra.Command, args []string) {
            if repo == "" {
                color.Red("Repository must be specified.")
                return
            }
            if !strings.Contains(repo, "/") {
                color.Red("Invalid repository format. Please use 'owner/repo'.")
                return
            }
            if secretName == "" {
                color.Red("Secret name must be provided.")
                return
            }
            if secretValue == "" {
                // Prompt for secret value if not provided
                color.Blue("Enter the secret value:")
                reader := bufio.NewReader(os.Stdin)
                secretValueInput, _ := reader.ReadString('\n')
                secretValue = strings.TrimSpace(secretValueInput)
            }
            strategy := &AddSecretStrategy{
                Token:       token,
                Repo:        repo,
                SecretName:  secretName,
                SecretValue: secretValue,
            }
            strategy.Execute()
        },
    }
    addSecretCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository name in 'owner/repo' format")
    addSecretCmd.Flags().StringVarP(&secretName, "name", "n", "", "Name of the secret")
    addSecretCmd.Flags().StringVarP(&secretValue, "value", "v", "", "Value of the secret")

    // Add Workflow Command
    var workflowName string
    var workflowContent string
    var workflowFile string

    addWorkflowCmd := &cobra.Command{
        Use:   "add-workflow",
        Short: "Add a GitHub Actions workflow to a repository",
        Run: func(cmd *cobra.Command, args []string) {
            if repo == "" {
                color.Red("Repository must be specified.")
                return
            }
            if !strings.Contains(repo, "/") {
                color.Red("Invalid repository format. Please use 'owner/repo'.")
                return
            }
            if workflowName == "" {
                color.Red("Workflow name must be provided.")
                return
            }
            if workflowContent == "" && workflowFile == "" {
                color.Red("Either workflow content or workflow file must be provided.")
                return
            }
            if workflowContent == "" && workflowFile != "" {
                // Read content from the provided file
                contentBytes, err := os.ReadFile(workflowFile)
                if err != nil {
                    color.Red("Error reading workflow file '%s': %s", workflowFile, err)
                    return
                }
                workflowContent = string(contentBytes)
            }
            strategy := &AddWorkflowStrategy{
                Token:        token,
                Repo:         repo,
                WorkflowPath: workflowName,
                Content:      workflowContent,
            }
            strategy.Execute()
        },
    }
    addWorkflowCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repository name in 'owner/repo' format")
    addWorkflowCmd.Flags().StringVarP(&workflowName, "name", "n", "", "Name of the workflow file (e.g., ci.yml)")
    addWorkflowCmd.Flags().StringVarP(&workflowContent, "content", "c", "", "Content of the workflow file")
    addWorkflowCmd.Flags().StringVarP(&workflowFile, "file", "f", "", "Path to the workflow file to read content from")

    // Store Config Command
    var configKey string
    var configValue string

    storeConfigCmd := &cobra.Command{
        Use:   "store-config",
        Short: "Store a configuration key-value pair",
        Run: func(cmd *cobra.Command, args []string) {
            if configKey == "" {
                color.Red("Configuration key must be provided.")
                return
            }
            if configValue == "" {
                color.Red("Configuration value must be provided.")
                return
            }
            strategy := &StoreConfigStrategy{
                ConfigKey:   configKey,
                ConfigValue: configValue,
            }
            strategy.Execute()
        },
    }
    storeConfigCmd.Flags().StringVarP(&configKey, "key", "k", "", "Configuration key")
    storeConfigCmd.Flags().StringVarP(&configValue, "value", "v", "", "Configuration value")

    // Add subcommands to root command
    rootCmd.AddCommand(addSecretCmd)
    rootCmd.AddCommand(addWorkflowCmd)
    rootCmd.AddCommand(storeConfigCmd)

    // Execute the root command
    if err := rootCmd.Execute(); err != nil {
        color.Red("Error: %s", err)
        os.Exit(1)
    }
}