package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "syscall"
    "time"

    "github.com/fatih/color"
    "github.com/cheggaaa/pb/v3"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "golang.org/x/term"
)

// Strategy interface for executing different strategies
type Strategy interface {
    Execute()
}

// AddSecretStrategy for adding a GitHub secret
type AddSecretStrategy struct {
    Token       string
    Repo        string
    SecretName  string
    SecretValue string
}

func (a *AddSecretStrategy) Execute() {
    env := os.Environ()
    env = append(env, fmt.Sprintf("GITHUB_TOKEN=%s", a.Token))

    // Loading Animation
    progress := pb.StartNew(100)
    go func() {
        for i := 0; i < 100; i++ {
            time.Sleep(10 * time.Millisecond)
            progress.Increment()
        }
    }()

    // Command to add the secret using GitHub CLI
    cmd := exec.Command("gh", "secret", "set", a.SecretName, "--repo", a.Repo, "--body", a.SecretValue)
    cmd.Env = env
    output, err := cmd.CombinedOutput()
    progress.Finish()

    if err != nil {
        color.New(color.FgRed).Printf("Error adding secret: %s\n", err)
        fmt.Println(string(output))
        return
    }
    color.New(color.FgGreen).Printf("Secret '%s' added to repository '%s' successfully.\n", a.SecretName, a.Repo)

    // Save secret locally for persistence
    saveSecretLocally(a.SecretName, a.SecretValue)
}

// AddWorkflowStrategy for adding a GitHub Actions workflow
type AddWorkflowStrategy struct {
    Token        string
    Repo         string
    WorkflowPath string
    Content      string
}

func (a *AddWorkflowStrategy) Execute() {
    repoDir := a.RepoDirectory()

    // Loading Animation
    progress := pb.StartNew(100)
    go func() {
        for i := 0; i < 100; i++ {
            time.Sleep(10 * time.Millisecond)
            progress.Increment()
        }
    }()

    // Check if current directory is the repository
    cwd, err := os.Getwd()
    if err != nil {
        color.New(color.FgRed).Printf("Error getting current working directory: %s\n", err)
        return
    }

    inRepoDir := false
    if filepath.Base(cwd) == repoDir {
        inRepoDir = true
    }

    if inRepoDir {
        color.New(color.FgGreen).Printf("Running inside repository directory '%s'.\n", repoDir)
    } else {
        // Clone the repository if it doesn't exist locally
        if _, err := os.Stat(repoDir); os.IsNotExist(err) {
            cloneCmd := exec.Command("git", "clone", fmt.Sprintf("https://github.com/%s.git", a.Repo), repoDir)
            cloneCmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", a.Token))
            cloneOutput, err := cloneCmd.CombinedOutput()
            progress.Finish()
            if err != nil {
                color.New(color.FgRed).Printf("Error cloning repository: %s\n", err)
                fmt.Println(string(cloneOutput))
                return
            }
            color.New(color.FgGreen).Printf("Repository '%s' cloned successfully.\n", a.Repo)
        } else {
            color.New(color.FgYellow).Printf("Repository '%s' already exists locally.\n", a.Repo)
            progress.Finish()
        }
        // Change directory to the repository
        cwd = repoDir
    }

    // Create the workflow file
    fullPath := filepath.Join(cwd, ".github", "workflows", a.WorkflowPath)
    err = os.MkdirAll(filepath.Dir(fullPath), os.ModePerm)
    if err != nil {
        color.New(color.FgRed).Printf("Error creating workflow directory: %s\n", err)
        return
    }

    err = os.WriteFile(fullPath, []byte(a.Content), 0644)
    if err != nil {
        color.New(color.FgRed).Printf("Error writing workflow file: %s\n", err)
        return
    }

    // Add, commit, and push changes with loading animation
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
            progress.Finish()
            color.New(color.FgRed).Printf("Error running command '%s': %s\n", strings.Join(args, " "), err)
            fmt.Println(string(output))
            return
        }
    }

    progress.Finish()
    color.New(color.FgGreen).Printf("Workflow '%s' added to repository '%s' successfully.\n", a.WorkflowPath, a.Repo)
}

func (a *AddWorkflowStrategy) RepoDirectory() string {
    parts := strings.Split(a.Repo, "/")
    return parts[len(parts)-1]
}

// StoreConfigStrategy for storing configuration key-value pairs
type StoreConfigStrategy struct {
    ConfigKey   string
    ConfigValue string
}

func (s *StoreConfigStrategy) Execute() {
    viper.Set(s.ConfigKey, s.ConfigValue)
    err := viper.WriteConfig()
    if err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            // Config file doesn't exist, create it
            err = viper.SafeWriteConfig()
            if err != nil {
                color.New(color.FgRed).Printf("Error creating config file: %s\n", err)
                return
            }
        } else {
            color.New(color.FgRed).Printf("Error writing config: %s\n", err)
            return
        }
    }
    color.New(color.FgYellow).Printf("Configuration '%s' saved successfully.\n", s.ConfigKey)
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
        color.New(color.FgRed).Printf("Error saving secret locally: %s\n", err)
        return
    }
    defer file.Close()
    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ")
    err = encoder.Encode(secrets)
    if err != nil {
        color.New(color.FgRed).Printf("Error encoding secrets: %s\n", err)
        return
    }

    color.New(color.FgYellow).Printf("Secret '%s' saved locally.\n", secretName)
}

// Function to print ASCII Header
func printASCIIHeader() {
    header := `
  ____ _____  __  __ 
 / ___|_   _|/ _|/ _|
| |  _  | | | |_| |_ 
| |_| | | | |  _|  _|
 \____| |_| |_| |_|  
                      
    `
    color.New(color.FgCyan).Println(header)
}

// Main function to handle commands using Cobra
func main() {
    printASCIIHeader()

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
                color.New(color.FgRed).Printf("Error creating config file: %s\n", err)
                return
            }
            color.New(color.FgYellow).Println("Config file created: config.json")
        } else {
            color.New(color.FgRed).Printf("Error reading config file: %s\n", err)
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
                color.New(color.FgBlue).Println("Enter your GitHub token:")
                byteToken, err := term.ReadPassword(int(syscall.Stdin))
                fmt.Println()
                if err != nil {
                    color.New(color.FgRed).Printf("Error reading token: %s\n", err)
                    os.Exit(1)
                }
                token = strings.TrimSpace(string(byteToken))
                viper.Set("github_token", token)
                err = viper.WriteConfig()
                if err != nil {
                    if _, ok := err.(viper.ConfigFileNotFoundError); ok {
                        err = viper.SafeWriteConfig()
                        if err != nil {
                            color.New(color.FgRed).Printf("Error creating config file: %s\n", err)
                            os.Exit(1)
                        }
                    } else {
                        color.New(color.FgRed).Printf("Error writing config: %s\n", err)
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
                color.New(color.FgRed).Println("Repository must be specified.")
                return
            }
            if !strings.Contains(repo, "/") {
                color.New(color.FgRed).Println("Invalid repository format. Please use 'owner/repo'.")
                return
            }
            if secretName == "" {
                color.New(color.FgRed).Println("Secret name must be provided.")
                return
            }
            if secretValue == "" {
                // Prompt for secret value if not provided
                color.New(color.FgBlue).Println("Enter the secret value:")
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
                color.New(color.FgRed).Println("Repository must be specified.")
                return
            }
            if !strings.Contains(repo, "/") {
                color.New(color.FgRed).Println("Invalid repository format. Please use 'owner/repo'.")
                return
            }
            if workflowName == "" {
                color.New(color.FgRed).Println("Workflow name must be provided.")
                return
            }
            if workflowContent == "" && workflowFile == "" {
                color.New(color.FgRed).Println("Either workflow content or workflow file must be provided.")
                return
            }
            if workflowContent == "" && workflowFile != "" {
                // Read content from the provided file
                contentBytes, err := os.ReadFile(workflowFile)
                if err != nil {
                    color.New(color.FgRed).Printf("Error reading workflow file '%s': %s\n", workflowFile, err)
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
                color.New(color.FgRed).Println("Configuration key must be provided.")
                return
            }
            if configValue == "" {
                color.New(color.FgRed).Println("Configuration value must be provided.")
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
        color.New(color.FgRed).Printf("Error: %s\n", err)
        os.Exit(1)
    }
}