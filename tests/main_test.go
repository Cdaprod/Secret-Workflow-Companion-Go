// tests/main_test.go

package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Helper function to execute the ghm command with arguments
func executeGhmCommand(args ...string) (string, string, error) {
    // Assuming the tests are run from the tests/ directory
    cmd := exec.Command("../ghm", args...)

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()
    return stdout.String(), stderr.String(), err
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
    stdout, stderr, err := executeGhmCommand("add-secret")

    require.Error(t, err, "Executing add-secret without flags should return an error")
    assert.Contains(t, stderr, "Repository must be specified.")
}

// Test the add-secret command with all required flags
func TestAddSecret(t *testing.T) {
    // Setup: Ensure the repository exists or mock its existence
    // For this test, we'll assume the repository "owner/test-repo" exists

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
    secretsFile := filepath.Join("..", "secrets.json")
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
    configFile := filepath.Join("..", "config.json")
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
    stdout, stderr, err := executeGhmCommand("add-workflow")

    require.Error(t, err, "Executing add-workflow without flags should return an error")
    assert.Contains(t, stderr, "Repository must be specified.")
}

// Test the add-workflow command with all required flags
func TestAddWorkflow(t *testing.T) {
    // Setup: Ensure the repository exists or mock its existence
    // For this test, we'll assume the repository "owner/test-repo" exists

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
    workflowPath := filepath.Join("..", repo, ".github", "workflows", workflowName)
    defer os.RemoveAll(filepath.Join("..", repo)) // Clean up after test

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