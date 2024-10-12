package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-github/v50/github"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEncryptor is a mock implementation of the Encryptor interface
type MockEncryptor struct {
	mock.Mock
}

func (m *MockEncryptor) Encrypt(secretValue string, publicKey *github.PublicKey) (string, error) {
	args := m.Called(secretValue, publicKey)
	return args.String(0), args.Error(1)
}

// MockGHM is a mock implementation of the GHM interface
type MockGHM struct {
	mock.Mock
}

func (m *MockGHM) AddSecret(ctx context.Context, repo, secretName, secretValue string) error {
	args := m.Called(ctx, repo, secretName, secretValue)
	return args.Error(0)
}

func (m *MockGHM) AddWorkflow(ctx context.Context, repo, workflowName, content string) error {
	args := m.Called(ctx, repo, workflowName, content)
	return args.Error(0)
}

func (m *MockGHM) StoreConfig(ctx context.Context, key, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func TestAddSecretStrategy_Execute(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	mockEncryptor := new(MockEncryptor)
	mockEncryptor.On("Encrypt", "supersecretvalue1", mock.Anything).Return("encrypted_value1", nil)

	strategy := &AddSecretStrategy{
		Token:       "test_token",
		Repo:        "owner/repo",
		SecretName:  "SECRET_KEY_1",
		SecretValue: "supersecretvalue1",
		Encryptor:   mockEncryptor,
		Logger:      logger,
	}

	err := strategy.Execute()
	assert.NoError(t, err)
	mockEncryptor.AssertExpectations(t)
}

func TestStoreConfigStrategy_Execute(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	strategy := &StoreConfigStrategy{
		ConfigKey:   "github_token",
		ConfigValue: "test_token",
		Logger:      logger,
	}

	err := strategy.Execute()
	assert.NoError(t, err)

	// Verify that the config was written
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	assert.NoError(t, err)
	token := viper.GetString("github_token")
	assert.Equal(t, "test_token", token)
}

func TestLoadAndSaveReposConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	// Create a temporary repos.json
	tempFile, err := ioutil.TempFile("", "repos.json")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	reposConfig := &ReposConfig{
		Repositories: map[string]RepoConfig{
			"owner/repo": {
				Secrets:    []string{"SECRET_KEY_1"},
				Workflows:  []string{"ci.yml"},
				LastUpdate: "2024-10-11T10:00:00Z",
			},
		},
	}

	// Save reposConfig to tempFile
	err = SaveReposConfig(reposConfig, logger)
	assert.NoError(t, err)

	// Load reposConfig from tempFile
	loadedConfig, err := LoadReposConfig(logger)
	assert.NoError(t, err)
	assert.Equal(t, reposConfig, loadedConfig)
}