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

// Global color functions for easy reuse
var (
    red    = color.New(color.FgRed).SprintFunc()
    green  = color.New(color.FgGreen).SprintFunc()
    yellow = color.New(color.FgYellow).SprintFunc()
    blue   = color.New(color.FgBlue).SprintFunc()
    cyan   = color.New(color.FgCyan).SprintFunc()
    bold   = color.New(color.Bold).SprintFunc()
)

// Function to print a colorful ASCII Header
func printASCIIHeader() {
    header := `
      ____ _   _ __  __ 
     / ___| |_| |  \/  |
    | |  _|  _  | |\/| |
    | |_| | | | | |  | |
     \____|_| |_|_|  |_| 
                              
    `
    color.New(color.FgCyan).Println(header)
    color.New(color.FgMagenta).Println("    GitHub Management CLI (ghm)")
    color.New(color.FgCyan).Println("=======================================")
}

// Initialize Viper Configuration
func initConfig() {
    viper.SetConfigName("config")
    viper.SetConfigType("json")
    viper.AddConfigPath(".")
    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            if err := viper.SafeWriteConfig(); err != nil {
                red("Error creating config file:", err)
                os.Exit(1)
            }
            yellow("Config file created: config.json")
        } else {
            red("Error reading config file:", err)
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
                blue("Enter your GitHub token:")
                byteToken, err := term.ReadPassword(int(syscall.Stdin))
                fmt.Println()
                if err != nil {
                    red("Error reading token:", err)
                    os.Exit(1)
                }
                token = strings.TrimSpace(string(byteToken))
                viper.Set("github_token", token)
                if err := viper.WriteConfig(); err != nil {
                    if _, ok := err.(viper.ConfigFileNotFoundError); ok {
                        if err := viper.SafeWriteConfig(); err != nil {
                            red("Error creating config file:", err)
                            os.Exit(1)
                        }
                    } else {
                        red("Error writing config:", err)
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

// Initialize Add Secret Command
func initAddSecretCmd(token *string) *cobra.Command {
    var repo, secretName, secretValue string

    addSecretCmd := &cobra.Command{
        Use:   "add-secret",
        Short: "Add a secret to a GitHub repository",
        Run: func(cmd *cobra.Command, args []string) {
            if repo == "" {
                red("Repository must be specified.")
                return
            }
            if !strings.Contains(repo, "/") {
                red("Invalid repository format. Please use 'owner/repo'.")
                return
            }
            if secretName == "" {
                red("Secret name must be provided.")
                return
            }
            if secretValue == "" {
                blue("Enter the secret value:")
                reader := bufio.NewReader(os.Stdin)
                secretValueInput, _ := reader.ReadString('\n')
                secretValue = strings.TrimSpace(secretValueInput)
            }
            strategy := &AddSecretStrategy{
                Token:       *token,
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

    return addSecretCmd
}

// Initialize Add Workflow Command
func initAddWorkflowCmd(token *string) *cobra.Command {
    var repo, workflowName, workflowContent, workflowFile string

    addWorkflowCmd := &cobra.Command{
        Use:   "add-workflow",
        Short: "Add a GitHub Actions workflow to a repository",
        Run: func(cmd *cobra.Command, args []string) {
            if repo == "" {
                red("Repository must be specified.")
                return
            }
            if !strings.Contains(repo, "/") {
                red("Invalid repository format. Please use 'owner/repo'.")
                return
            }
            if workflowName == "" {
                red("Workflow name must be provided.")
                return
            }
            if workflowContent == "" && workflowFile == "" {
                red("Either workflow content or workflow file must be provided.")
                return
            }
            if workflowContent == "" && workflowFile != "" {
                contentBytes, err := os.ReadFile(workflowFile)
                if err != nil {
                    red("Error reading workflow file '%s': %s\n", workflowFile, err)
                    return
                }
                workflowContent = string(contentBytes)
            }
            strategy := &AddWorkflowStrategy{
                Token:        *token,
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

    return addWorkflowCmd
}

// Initialize Store Config Command
func initStoreConfigCmd() *cobra.Command {
    var configKey, configValue string

    storeConfigCmd := &cobra.Command{
        Use:   "store-config",
        Short: "Store a configuration key-value pair",
        Run: func(cmd *cobra.Command, args []string) {
            if configKey == "" {
                red("Configuration key must be provided.")
                return
            }
            if configValue == "" {
                red("Configuration value must be provided.")
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

    return storeConfigCmd
}


// StartProgress initializes and returns a single-line progress bar
func StartProgress(total int) *pb.ProgressBar {
    bar := pb.New(total).
        SetTemplateString("{{bar . }} {{percent . }}") // Correct template
    bar.Start()
    return bar
}

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

    // Initialize progress bar
    progress := StartProgress(100)
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
        red("Error adding secret:", err)
        fmt.Println(string(output))
        return
    }
    green("Secret '%s' added to repository '%s' successfully.", a.SecretName, a.Repo)

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

    // Initialize progress bar
    progress := StartProgress(100)
    go func() {
        for i := 0; i < 100; i++ {
            time.Sleep(10 * time.Millisecond)
            progress.Increment()
        }
    }()

    // Check if current directory is the repository
    cwd, err := os.Getwd()
    if err != nil {
        red("Error getting current working directory:", err)
        return
    }

    inRepoDir := false
    if filepath.Base(cwd) == repoDir {
        inRepoDir = true
    }

    if inRepoDir {
        green("Running inside repository directory '%s'.", repoDir)
    } else {
        // Clone the repository if it doesn't exist locally
        if _, err := os.Stat(repoDir); os.IsNotExist(err) {
            cloneCmd := exec.Command("git", "clone", fmt.Sprintf("https://github.com/%s.git", a.Repo), repoDir)
            cloneCmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", a.Token))
            cloneOutput, err := cloneCmd.CombinedOutput()
            progress.Finish()
            if err != nil {
                red("Error cloning repository:", err)
                fmt.Println(string(cloneOutput))
                return
            }
            green("Repository '%s' cloned successfully.", a.Repo)
        } else {
            yellow("Repository '%s' already exists locally.", a.Repo)
            progress.Finish()
        }
        // Change directory to the repository
        cwd = repoDir
    }

    // Create the workflow file
    fullPath := filepath.Join(cwd, ".github", "workflows", a.WorkflowPath)
    err = os.MkdirAll(filepath.Dir(fullPath), os.ModePerm)
    if err != nil {
        red("Error creating workflow directory:", err)
        return
    }

    err = os.WriteFile(fullPath, []byte(a.Content), 0644)
    if err != nil {
        red("Error writing workflow file:", err)
        return
    }

    // Add, commit, and push changes with progress
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
            red("Error running command '%s': %s", strings.Join(args, " "), err)
            fmt.Println(string(output))
            return
        }
    }

    progress.Finish()
    green("Workflow '%s' added to repository '%s' successfully.", a.WorkflowPath, a.Repo)
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
                red("Error creating config file:", err)
                return
            }
        } else {
            red("Error writing config:", err)
            return
        }
    }
    yellow("Configuration '%s' saved successfully.", s.ConfigKey)
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
        red("Error saving secret locally:", err)
        return
    }
    defer file.Close()
    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ")
    err = encoder.Encode(secrets)
    if err != nil {
        red("Error encoding secrets:", err)
        return
    }

    yellow("Secret '%s' saved locally.", secretName)
}

func main() {
    printASCIIHeader()
    initConfig()
    rootCmd := initRootCmd()

    if err := rootCmd.Execute(); err != nil {
        red("Error:", err)
        os.Exit(1)
    }
}