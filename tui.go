// tui.go
package main

import (
    "context"
    "fmt"
    //"os"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/manifoldco/promptui"
    "github.com/sirupsen/logrus"
    "github.com/spf13/viper"
)

// Define TUI tabs
var tabs = []string{"Secrets", "Workflows", "Repositories", "Settings", "Help"}

// Define the TUI model
type model struct {
    tabs        []string
    activeTab   int
    logger      *logrus.Logger
    reposConfig *ReposConfig
    // Additional state can be added here for each tab
}

// Initialize the TUI model
func newModel(logger *logrus.Logger, reposConfig *ReposConfig) model {
    return model{
        tabs:        tabs,
        activeTab:   0,
        logger:      logger,
        reposConfig: reposConfig,
    }
}

// Init is part of the Bubble Tea interface
func (m model) Init() tea.Cmd {
    return nil
}

// Update is part of the Bubble Tea interface
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit
        case "left", "h":
            m.activeTab = max(m.activeTab-1, 0)
        case "right", "l":
            m.activeTab = min(m.activeTab+1, len(m.tabs)-1)
        case "enter":
            // Handle actions based on the active tab
            switch m.activeTab {
            case 0:
                // Secrets Tab
                m.logger.Info("Secrets tab selected")
                // Implement secrets management interaction
                runAddSecretInteraction(m)
            case 1:
                // Workflows Tab
                m.logger.Info("Workflows tab selected")
                // Implement workflows management interaction
                runAddWorkflowInteraction(m)
            case 2:
                // Repositories Tab
                m.logger.Info("Repositories tab selected")
                // Implement repository overview interaction
                // For simplicity, no action
            case 3:
                // Settings Tab
                m.logger.Info("Settings tab selected")
                // Implement settings management interaction
                runStoreConfigInteraction(m)
            case 4:
                // Help Tab
                m.logger.Info("Help tab selected")
                // No action, help is already displayed
            }
        }
    }
    return m, nil
}

// View renders the TUI
func (m model) View() string {
    var tabUI string
    var separator = " | "

    for i, t := range m.tabs {
        if i == m.activeTab {
            tabUI += lipgloss.NewStyle().
                Bold(true).
                Foreground(lipgloss.Color("205")).
                Render(t) + separator
        } else {
            tabUI += lipgloss.NewStyle().
                Foreground(lipgloss.Color("240")).
                Render(t) + separator
        }
    }

    // Remove the trailing separator
    tabUI = strings.TrimSuffix(tabUI, separator)

    // Render content based on activeTab
    var content string
    switch m.activeTab {
    case 0:
        content = renderSecretsTab(m)
    case 1:
        content = renderWorkflowsTab(m)
    case 2:
        content = renderRepositoriesTab(m)
    case 3:
        content = renderSettingsTab(m)
    case 4:
        content = renderHelpTab(m)
    default:
        content = "Unknown Tab"
    }

    // Combine tabs and content
    return lipgloss.JoinVertical(lipgloss.Left, tabUI, content)
}

// Render functions for each tab
func renderSecretsTab(m model) string {
    return "Secrets Management Interface:\n\nPress 'Enter' to add a secret."
}

func renderWorkflowsTab(m model) string {
    return "Workflows Management Interface:\n\nPress 'Enter' to add a workflow."
}

func renderRepositoriesTab(m model) string {
    // Example: List repositories
    if len(m.reposConfig.Repositories) == 0 {
        return "No repositories configured."
    }

    var repoList string
    for repo, config := range m.reposConfig.Repositories {
        repoList += fmt.Sprintf("Repository: %s\n  Secrets: %s\n  Workflows: %s\n\n",
            repo,
            strings.Join(config.Secrets, ", "),
            strings.Join(config.Workflows, ", "))
    }
    return repoList
}

func renderSettingsTab(m model) string {
    return "Settings Interface:\n\nPress 'Enter' to store a configuration key-value pair."
}

func renderHelpTab(m model) string {
    helpText := `
GitHub Management CLI Help

- Secrets Tab:
  Manage your GitHub secrets across repositories.
  - Press 'Enter' to add a new secret.

- Workflows Tab:
  Manage your GitHub Actions workflows across repositories.
  - Press 'Enter' to add a new workflow.

- Repositories Tab:
  View all repositories and their associated secrets and workflows.

- Settings Tab:
  Configure your GitHub tokens and default settings.
  - Press 'Enter' to store a new configuration.

- Help Tab:
  Display this help information.

Navigation:
- Use Left/Right arrows or 'h'/'l' to switch between tabs.
- Press 'Enter' to interact with the active tab.
- Press 'q' or 'Ctrl+C' to quit the application.
`
    return helpText
}

// Interaction functions
func runAddSecretInteraction(m model) {
    // Prompt for repository
    repo, err := promptInput("Enter repository (owner/repo): ")
    if err != nil {
        m.logger.Errorf("Error reading repository: %v", err)
        return
    }

    // Prompt for secret name
    secretName, err := promptInput("Enter secret name: ")
    if err != nil {
        m.logger.Errorf("Error reading secret name: %v", err)
        return
    }

    // Prompt for secret value
    secretValue, err := promptPassword("Enter secret value: ")
    if err != nil {
        m.logger.Errorf("Error reading secret value: %v", err)
        return
    }

    // Execute the AddSecret command
    ghm := NewGHM(viper.GetString("github_token"), m.logger)
    err = ghm.AddSecret(context.Background(), repo, secretName, secretValue)
    if err != nil {
        m.logger.Errorf("Error adding secret: %v", err)
        fmt.Println("Failed to add secret.")
    } else {
        SuccessColor.Println("Secret added successfully.")
    }

    // Reload reposConfig
    updatedReposConfig, err := LoadReposConfig(m.logger)
    if err != nil {
        m.logger.Errorf("Error reloading repos config: %v", err)
        return
    }
    m.reposConfig = updatedReposConfig
}

func runAddWorkflowInteraction(m model) {
    // Prompt for repository
    repo, err := promptInput("Enter repository (owner/repo): ")
    if err != nil {
        m.logger.Errorf("Error reading repository: %v", err)
        return
    }

    // Prompt for workflow name
    workflowName, err := promptInput("Enter workflow file name (e.g., ci.yml): ")
    if err != nil {
        m.logger.Errorf("Error reading workflow name: %v", err)
        return
    }

    // Prompt for workflow content
    workflowContent, err := promptMultiLineInput("Enter workflow YAML content (end with an empty line):")
    if err != nil {
        m.logger.Errorf("Error reading workflow content: %v", err)
        return
    }

    // Execute the AddWorkflow command
    ghm := NewGHM(viper.GetString("github_token"), m.logger)
    err = ghm.AddWorkflow(context.Background(), repo, workflowName, workflowContent)
    if err != nil {
        m.logger.Errorf("Error adding workflow: %v", err)
        fmt.Println("Failed to add workflow.")
    } else {
        SuccessColor.Println("Workflow added successfully.")
    }

    // Reload reposConfig
    updatedReposConfig, err := LoadReposConfig(m.logger)
    if err != nil {
        m.logger.Errorf("Error reloading repos config: %v", err)
        return
    }
    m.reposConfig = updatedReposConfig
}

func runStoreConfigInteraction(m model) {
    // Prompt for config key
    configKey, err := promptInput("Enter configuration key: ")
    if err != nil {
        m.logger.Errorf("Error reading config key: %v", err)
        return
    }

    // Prompt for config value
    configValue, err := promptInput("Enter configuration value: ")
    if err != nil {
        m.logger.Errorf("Error reading config value: %v", err)
        return
    }

    // Execute the StoreConfig command
    ghm := NewGHM(viper.GetString("github_token"), m.logger)
    err = ghm.StoreConfig(context.Background(), configKey, configValue)
    if err != nil {
        m.logger.Errorf("Error storing config: %v", err)
        fmt.Println("Failed to store configuration.")
    } else {
        SuccessColor.Println("Configuration stored successfully.")
    }
}

// Prompt helper functions
func promptInput(label string) (string, error) {
    prompt := promptui.Prompt{
        Label: label,
    }
    return prompt.Run()
}

func promptPassword(label string) (string, error) {
    prompt := promptui.Prompt{
        Label:    label,
        Mask:     '*',
        Validate: func(input string) error {
            if len(input) == 0 {
                return fmt.Errorf("value cannot be empty")
            }
            return nil
        },
    }
    return prompt.Run()
}

func promptMultiLineInput(label string) (string, error) {
    fmt.Println(label)
    var lines []string
    for {
        var input string
        _, err := fmt.Scanln(&input)
        if err != nil {
            return "", err
        }
        if input == "" {
            break
        }
        lines = append(lines, input)
    }
    return strings.Join(lines, "\n"), nil
}

// Run the TUI program
func runTUI(logger *logrus.Logger) error {
    // Load reposConfig
    reposConfig, err := LoadReposConfig(logger)
    if err != nil {
        logger.Errorf("Error loading repositories config: %v", err)
        fmt.Println("Failed to load repositories config. Exiting TUI.")
        return err // Return the error
    }

    p := tea.NewProgram(newModel(logger, reposConfig))
    if err := p.Start(); err != nil {
        logger.Errorf("Error running TUI: %v", err)
        return fmt.Errorf("Error running TUI: %v", err) // Return the error
    }
    return nil // Return nil if no error
}

// Helper functions
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}