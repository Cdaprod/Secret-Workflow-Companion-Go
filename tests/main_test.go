// tests/main_test.go

package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "strings"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI escape codes from strings
func stripANSI(s string) string {
    return ansi.ReplaceAllString(s, "")
}

// TestMain sets up the testing environment
func TestMain(m *testing.M) {
    // Setup: Create config.json with a dummy token
    config := map[string]string{
        "github_token": "dummy_token",
    }
    configData, _ := json.Marshal(config)
    configPath := "config.json"
    err := os.WriteFile(configPath, configData, 0644)
    if err != nil {
        fmt.Println("Failed to create config.json:", err)
        os.Exit(1)
    }
    defer os.Remove(configPath)

    // Setup: Create a fake 'gh' executable
    fakeGhPath := filepath.Join("..", "gh")
    ghScript := `#!/bin/bash
if [ "$1" == "secret" ] && [ "$2" == "set" ]; then
    echo "Secret set successfully."
    exit 0
elif [ "$1" == "secret" ]; then
    echo "Invalid secret command."
    exit 1
elif [ "$1" == "auth" ]; then
    echo "authenticated"
    exit 0
else
    echo "Unknown gh command."
    exit 1
fi
`
    err = os.WriteFile(fakeGhPath, []byte(ghScript), 0755)
    if err != nil {
        fmt.Println("Failed to create fake gh:", err)
        os.Exit(1)
    }
    defer os.Remove(fakeGhPath)

    // Setup: Create a fake 'git' executable
    fakeGitPath := filepath.Join("..", "git")
    gitScript := `#!/bin/bash
if [ "$1" == "clone" ]; then
    # Simulate git clone by creating the target directory
    target_dir="$3"
    mkdir -p "$target_dir/.github/workflows"
    echo "Mock git clone into $target_dir"
    exit 0
else
    echo "Mock git command executed: $@"
    exit 0
fi
`
    err = os.WriteFile(fakeGitPath, []byte(gitScript), 0755)
    if err != nil {
        fmt.Println("Failed to create fake git:", err)
        os.Exit(1)
    }
    defer os.Remove(fakeGitPath)

    // Prepend parent directory to PATH so that 'gh' and 'git' are used
    originalPath := os.Getenv("PATH")
    newPath := fmt.Sprintf("../:%s", originalPath)
    os.Setenv("PATH", newPath)
    defer os.Setenv("PATH", originalPath)

    // Build the ghm binary
    buildCmd := exec.Command("go", "build", "-o", "../ghm", "../main.go")
    buildOutput, err := buildCmd.CombinedOutput()
    if err != nil {
        fmt.Printf("Failed to build ghm: %s\n", string(buildOutput))
        os.Exit(1)
    }

    // Run tests
    code := m.Run()

    // Cleanup: Remove the ghm binary and fake gh/git
    os.Remove("../ghm")
    os.Remove(fakeGhPath)
    os.Remove(fakeGitPath)

    os.Exit(code)
}

// executeGhmCommand runs the ghm command with provided arguments and returns stdout, stderr, and error
func executeGhmCommand(args ...string) (string, string, error) {
    cmd := exec.Command("../ghm", args...)

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()
    return stripANSI(stdout.String()), stripANSI(stderr.String()), err
}

// Test the root command to display help
func TestRootHelp(t *testing.T) {
    stdout, stderr, err := executeGhmCommand("--help")

    require.NoError(t, err, "Executing ghm --help should not return an error")
    require.Empty(t, stderr, "Executing ghm --help should not output to stderr")

    // Check if help message contains expected sections
    assert.Contains(t, stdout, "Usage:")
    assert.Contains(t, stdout, "Available Commands:")
    assert.Contains(t, stdout, "add-secret")
    assert.Contains(t, stdout, "add-workflow")
    assert.Contains(t, stdout, "store-config")
}

// Test the add-secret command with missing required flags
func TestAddSecretMissingFlags(t *testing.T) {
    stdout, stderr, err := executeGhmCommand("add-secret") // Don't ignore stdout

    require.Error(t, err, "Executing add-secret without flags should return an error")
    assert.True(t, strings.Contains(stderr, "Repository must be specified.") || strings.Contains(stdout, "Repository must be specified."),
        "Error message 'Repository must be specified.' not found in stdout or stderr")
}

// Test the add-secret command with all required flags
func TestAddSecret(t *testing.T) {
    // Define test parameters
    repo := "owner/test-repo"
    secretName := "TEST_SECRET"
    secretValue := "testvalue123"

    // Execute the add-secret command
    stdout, stderr, err := executeGhmCommand(
        "add-secret",
        "--repo", repo,
        "--name", secretName,
        "--value", secretValue,
    )

    // Assertions
    require.NoError(t, err, "Executing add-secret should not return an error")
    require.Empty(t, stderr, "Executing add-secret should not output to stderr")

    assert.Contains(t, stdout, fmt.Sprintf("Secret '%s' added to repository '%s' successfully.", secretName, repo))
    assert.Contains(t, stdout, fmt.Sprintf("Secret '%s' saved locally.", secretName))

    // Verify that the secret is saved locally in secrets.json
    secretsFile := "secrets.json"
    defer os.Remove(secretsFile) // Clean up after test

    data, err := os.ReadFile(secretsFile)
    require.NoError(t, err, "Reading secrets.json should not return an error")

    var secrets map[string]string
    err = json.Unmarshal(data, &secrets)
    require.NoError(t, err, "Unmarshalling secrets.json should not return an error")

    assert.Equal(t, secretValue, secrets[secretName], "Secret value should match the input")
}

// Test the store-config command
func TestStoreConfig(t *testing.T) {
    configKey := "api_key"
    configValue := "12345ABCDE"

    stdout, stderr, err := executeGhmCommand(
        "store-config",
        "--key", configKey,
        "--value", configValue,
    )

    require.NoError(t, err, "Executing store-config should not return an error")
    require.Empty(t, stderr, "Executing store-config should not output to stderr")

    assert.Contains(t, stdout, fmt.Sprintf("Configuration '%s' saved successfully.", configKey))

    // Verify that the configuration is saved using Viper in config.json
    configFile := "config.json"
    defer os.Remove(configFile) // Clean up after test

    data, err := os.ReadFile(configFile)
    require.NoError(t, err, "Reading config.json should not return an error")

    var config map[string]string
    err = json.Unmarshal(data, &config)
    require.NoError(t, err, "Unmarshalling config.json should not return an error")

    assert.Equal(t, configValue, config[configKey], "Configuration value should match the input")
}

// Test the add-workflow command with missing required flags
func TestAddWorkflowMissingFlags(t *testing.T) {
    stdout, stderr, err := executeGhmCommand("add-workflow") // Don't ignore stdout

    require.Error(t, err, "Executing add-workflow without flags should return an error")
    assert.True(t, strings.Contains(stderr, "Repository must be specified.") || strings.Contains(stdout, "Repository must be specified."),
        "Error message 'Repository must be specified.' not found in stdout or stderr")
}

// Test the add-workflow command with all required flags
func TestAddWorkflow(t *testing.T) {
    // Define test parameters
    repo := "owner/test-repo"
    workflowName := "ci.yml"
    workflowContent := `
name: CI

on:
  push:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v2
    - name: Set up Go
      uses: actions/setup-go@v2
      with:
        go-version: 1.18
    - name: Build
      run: go build -v ./...
`

    // Execute the add-workflow command
    stdout, stderr, err := executeGhmCommand(
        "add-workflow",
        "--repo", repo,
        "--name", workflowName,
        "--content", workflowContent,
    )

    // Assertions
    require.NoError(t, err, "Executing add-workflow should not return an error")
    require.Empty(t, stderr, "Executing add-workflow should not output to stderr")

    assert.Contains(t, stdout, fmt.Sprintf("Workflow '%s' added to repository '%s' successfully.", workflowName, repo))

    // Verify that the workflow file exists in the repository
    workflowPath := filepath.Join(repo, ".github", "workflows", workflowName)
    defer os.RemoveAll(repo) // Clean up after test

    _, err = os.Stat(workflowPath)
    require.NoError(t, err, "Workflow file should exist in the repository")

    // Optionally, read and verify the workflow content
    data, err := os.ReadFile(workflowPath)
    require.NoError(t, err, "Reading workflow file should not return an error")

    assert.Contains(t, string(data), "name: CI", "Workflow content should contain the correct name")
    assert.Contains(t, string(data), "go-version: 1.18", "Workflow content should contain the correct Go version")
}

// Test colorized outputs (optional)
func TestColorizedOutputs(t *testing.T) {
    // This test checks if the colored output contains ANSI escape codes
    // It does not verify the colors themselves, as that would require terminal rendering

    stdout, stderr, err := executeGhmCommand("add-secret", "--repo", "owner/test-repo", "--name", "COLOR_TEST_SECRET", "--value", "colorvalue123")

    require.NoError(t, err, "Executing add-secret should not return an error")
    require.Empty(t, stderr, "Executing add-secret should not output to stderr")

    // ANSI escape code regex can be used to check for color codes
    hasANSI := strings.Contains(stdout, "\033[")
    assert.True(t, hasANSI, "Output should contain ANSI escape codes")
}

// MockEncryptor is a mock implementation of the Encryptor interface
type MockEncryptor struct{}

// Encrypt returns a dummy encrypted string
func (m *MockEncryptor) Encrypt(secretValue string, publicKey *github.PublicKey) (string, error) {
    return "dummy_encrypted_value", nil
}

// TestAddSecret tests the add-secret command
func TestAddSecret(t *testing.T) {
    // Setup mock GitHub server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/actions/secrets/public-key") {
            response := map[string]interface{}{
                "key_id": "dummykeyid",
                "key":    "dummypublickey",
            }
            json.NewEncoder(w).Encode(response)
            return
        }
        if r.Method == "PUT" && strings.HasSuffix(r.URL.Path, "/actions/secrets/TEST_SECRET") {
            response := map[string]interface{}{
                "name": "TEST_SECRET",
            }
            json.NewEncoder(w).Encode(response)
            return
        }
        w.WriteHeader(http.StatusNotFound)
    }))
    defer server.Close()

    // Override the GitHub API URL in the application
    os.Setenv("GITHUB_API_URL", server.URL)
    defer os.Unsetenv("GITHUB_API_URL")

    // Override the GitHub token
    os.Setenv("GITHUB_TOKEN", "dummy_token")
    defer os.Unsetenv("GITHUB_TOKEN")

    // Initialize Viper with the mock GitHub API URL
    viper.Set("github_token", "dummy_token")

    // Execute the add-secret command
    stdout, stderr, err := executeGhmCommand(
        "add-secret",
        "--repo", "owner/repo",
        "--name", "TEST_SECRET",
        "--value", "testvalue123",
    )

    // Assertions
    require.NoError(t, err, "Executing add-secret should not return an error")
    require.Empty(t, stderr, "Executing add-secret should not output to stderr")

    assert.Contains(t, stdout, "Secret 'TEST_SECRET' added to repository 'owner/repo' successfully.")
    assert.Contains(t, stdout, "Secret 'TEST_SECRET' saved locally.")

    // Verify that the secret is saved locally in secrets.json
    secretsFile := filepath.Join("..", "secrets.json")
    defer os.Remove(secretsFile) // Clean up after test

    data, err := os.ReadFile(secretsFile)
    require.NoError(t, err, "Reading secrets.json should not return an error")

    var secrets map[string]string
    err = json.Unmarshal(data, &secrets)
    require.NoError(t, err, "Unmarshalling secrets.json should not return an error")

    assert.Equal(t, "testvalue123", secrets["TEST_SECRET"], "Secret value should match the input")
}

// executeGhmCommand runs the ghm command with provided arguments and returns stdout, stderr, and error
func executeGhmCommand(args ...string) (string, string, error) {
    cmd := exec.Command("../ghm", args...)

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()
    return stdout.String(), stderr.String(), err
}