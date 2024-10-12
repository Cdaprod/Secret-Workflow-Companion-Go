// ghm.go
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v50/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/oauth2"
)

// Encryptor interface abstracts the encryption functionality
type Encryptor interface {
	Encrypt(secretValue string, publicKey *github.PublicKey) (string, error)
}

// EncryptorImpl is the concrete implementation of Encryptor
type EncryptorImpl struct{}

// Encrypt encrypts the secret value using the provided GitHub repository public key
func (e *EncryptorImpl) Encrypt(secretValue string, publicKey *github.PublicKey) (string, error) {
	if publicKey == nil || publicKey.Key == nil {
		return "", errors.New("invalid public key provided")
	}

	// Decode the public key from base64
	keyBytes, err := base64.StdEncoding.DecodeString(*publicKey.Key)
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}

	// Check if the key is for NaCl encryption
	if len(keyBytes) == 32 {
		return e.encryptNaCl(secretValue, keyBytes)
	}

	// Otherwise, try RSA encryption
	return e.encryptRSA(secretValue, keyBytes)
}

// encryptNaCl encrypts the secret value using NaCl's box.SealAnonymous
func (e *EncryptorImpl) encryptNaCl(secretValue string, publicKey []byte) (string, error) {
	var publicKeyBytes [32]byte
	copy(publicKeyBytes[:], publicKey)

	encryptedBytes, err := box.SealAnonymous(nil, []byte(secretValue), &publicKeyBytes, rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt secret using NaCl: %w", err)
	}

	encryptedValue := base64.StdEncoding.EncodeToString(encryptedBytes)
	return encryptedValue, nil
}

// encryptRSA encrypts the secret value using RSA-OAEP with SHA-256
func (e *EncryptorImpl) encryptRSA(secretValue string, publicKey []byte) (string, error) {
	parsedPubKey, err := x509.ParsePKIXPublicKey(publicKey)
	if err != nil {
		rsaPubKey, parseErr := x509.ParsePKCS1PublicKey(publicKey)
		if parseErr != nil {
			return "", fmt.Errorf("failed to parse RSA public key: %w", err)
		}
		parsedPubKey = rsaPubKey
	}

	rsaPubKey, ok := parsedPubKey.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("public key is not an RSA public key")
	}

	label := []byte("")
	hash := sha256.New()
	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, rsaPubKey, []byte(secretValue), label)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt secret using RSA: %w", err)
	}

	encryptedValue := base64.StdEncoding.EncodeToString(ciphertext)
	return encryptedValue, nil
}

// GHM interface defines the methods for the GitHub Management CLI (ghm)
type GHM interface {
	AddSecret(ctx context.Context, repo, secretName, secretValue string) error
	AddWorkflow(ctx context.Context, repo, workflowName, content string) error
	StoreConfig(ctx context.Context, key, value string) error
	AddSecretsToRepo(ctx context.Context, targetRepo string, secretNames []string, reposConfig *ReposConfig) error
	AddWorkflowsToRepo(ctx context.Context, targetRepo string, workflowNames []string, reposConfig *ReposConfig) error
}

// GHMImpl is the concrete implementation of the GHM interface
type GHMImpl struct {
	Token     string
	Encryptor Encryptor
	Logger    *logrus.Logger
}

// NewGHM creates a new instance of GHMImpl
func NewGHM(token string, logger *logrus.Logger) GHM {
	return &GHMImpl{
		Token:     token,
		Encryptor: &EncryptorImpl{},
		Logger:    logger,
	}
}

// AddSecret adds a secret to the GitHub repository
func (g *GHMImpl) AddSecret(ctx context.Context, repo, secretName, secretValue string) error {
	strategy := &AddSecretStrategy{
		Token:       g.Token,
		Repo:        repo,
		SecretName:  secretName,
		SecretValue: secretValue,
		Encryptor:   g.Encryptor,
		Logger:      g.Logger,
	}
	return strategy.Execute()
}

// AddWorkflow adds a workflow file to the GitHub repository
func (g *GHMImpl) AddWorkflow(ctx context.Context, repo, workflowName, content string) error {
	strategy := &AddWorkflowStrategy{
		Token:        g.Token,
		Repo:         repo,
		WorkflowName: workflowName,
		Content:      content,
		Logger:       g.Logger,
	}
	return strategy.Execute()
}

// StoreConfig stores a configuration key-value pair
func (g *GHMImpl) StoreConfig(ctx context.Context, key, value string) error {
	strategy := &StoreConfigStrategy{
		ConfigKey:   key,
		ConfigValue: value,
		Logger:      g.Logger,
	}
	return strategy.Execute()
}

// AddSecretsToRepo adds multiple secrets to a target repository
func (g *GHMImpl) AddSecretsToRepo(ctx context.Context, targetRepo string, secretNames []string, reposConfig *ReposConfig) error {
	for _, secretName := range secretNames {
		// Retrieve secret value from secrets.json
		secretValue, err := g.getSecretValue(secretName)
		if err != nil {
			g.Logger.Errorf("Error retrieving secret '%s': %v", secretName, err)
			continue
		}

		// Add secret to the target repository
		err = g.AddSecret(ctx, targetRepo, secretName, secretValue)
		if err != nil {
			g.Logger.Errorf("Error adding secret '%s' to '%s': %v", secretName, targetRepo, err)
			continue
		}

		// Update reposConfig
		repoConfig, exists := reposConfig.Repositories[targetRepo]
		if !exists {
			repoConfig = RepoConfig{
				Secrets:    []string{},
				Workflows:  []string{},
				LastUpdate: time.Now().Format(time.RFC3339),
			}
		}
		repoConfig.Secrets = append(repoConfig.Secrets, secretName)
		repoConfig.LastUpdate = time.Now().Format(time.RFC3339)
		reposConfig.Repositories[targetRepo] = repoConfig

		g.Logger.Infof("Secret '%s' added to repository '%s'.", secretName, targetRepo)
	}

	return nil
}

// AddWorkflowsToRepo adds multiple workflows to a target repository
func (g *GHMImpl) AddWorkflowsToRepo(ctx context.Context, targetRepo string, workflowNames []string, reposConfig *ReposConfig) error {
	for _, workflowName := range workflowNames {
		// Retrieve workflow content from workflows.json
		workflowContent, err := g.getWorkflowContent(workflowName)
		if err != nil {
			g.Logger.Errorf("Error retrieving workflow '%s': %v", workflowName, err)
			continue
		}

		// Add workflow to the target repository
		err = g.AddWorkflow(ctx, targetRepo, workflowName, workflowContent)
		if err != nil {
			g.Logger.Errorf("Error adding workflow '%s' to '%s': %v", workflowName, targetRepo, err)
			continue
		}

		// Update reposConfig
		repoConfig, exists := reposConfig.Repositories[targetRepo]
		if !exists {
			repoConfig = RepoConfig{
				Secrets:    []string{},
				Workflows:  []string{},
				LastUpdate: time.Now().Format(time.RFC3339),
			}
		}
		repoConfig.Workflows = append(repoConfig.Workflows, workflowName)
		repoConfig.LastUpdate = time.Now().Format(time.RFC3339)
		reposConfig.Repositories[targetRepo] = repoConfig

		g.Logger.Infof("Workflow '%s' added to repository '%s'.", workflowName, targetRepo)
	}

	return nil
}

// ReposConfig holds the mapping between repositories and their added secrets/workflows
type ReposConfig struct {
	Repositories map[string]RepoConfig `json:"repositories"`
}

// RepoConfig holds the secrets and workflows added to a repository
type RepoConfig struct {
	Secrets    []string `json:"secrets"`
	Workflows  []string `json:"workflows"`
	LastUpdate string   `json:"last_update"`
}

// LoadReposConfig loads the repos.json configuration file
func LoadReposConfig(logger *logrus.Logger) (*ReposConfig, error) {
	configFile := "repos.json"
	reposConfig := &ReposConfig{
		Repositories: make(map[string]RepoConfig),
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Create an empty repos.json file
		file, err := os.Create(configFile)
		if err != nil {
			logger.Errorf("Error creating repos.json: %v", err)
			return nil, err
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(reposConfig)
		if err != nil {
			logger.Errorf("Error encoding repos.json: %v", err)
			return nil, err
		}
		return reposConfig, nil
	}

	// Read existing repos.json
	file, err := os.Open(configFile)
	if err != nil {
		logger.Errorf("Error opening repos.json: %v", err)
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(reposConfig)
	if err != nil {
		logger.Errorf("Error decoding repos.json: %v", err)
		return nil, err
	}

	return reposConfig, nil
}

// SaveReposConfig saves the repos.json configuration file
func SaveReposConfig(reposConfig *ReposConfig, logger *logrus.Logger) error {
	configFile := "repos.json"

	file, err := os.Create(configFile)
	if err != nil {
		logger.Errorf("Error creating repos.json: %v", err)
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(reposConfig)
	if err != nil {
		logger.Errorf("Error encoding repos.json: %v", err)
		return err
	}

	return nil
}

// getSecretValue retrieves the secret value from secrets.json
func (g *GHMImpl) getSecretValue(secretName string) (string, error) {
	secretsFile := "secrets.json"
	secrets := make(map[string]string)

	if _, err := os.Stat(secretsFile); os.IsNotExist(err) {
		return "", fmt.Errorf("secrets.json does not exist")
	}

	file, err := os.Open(secretsFile)
	if err != nil {
		g.Logger.Errorf("Error opening secrets.json: %v", err)
		return "", err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&secrets)
	if err != nil {
		g.Logger.Errorf("Error decoding secrets.json: %v", err)
		return "", err
	}

	secretValue, exists := secrets[secretName]
	if !exists {
		return "", fmt.Errorf("secret '%s' not found", secretName)
	}

	return secretValue, nil
}

// getWorkflowContent retrieves the workflow content from workflows.json
func (g *GHMImpl) getWorkflowContent(workflowName string) (string, error) {
	workflowsFile := "workflows.json"
	workflows := make(map[string]string)

	if _, err := os.Stat(workflowsFile); os.IsNotExist(err) {
		return "", fmt.Errorf("workflows.json does not exist")
	}

	file, err := os.Open(workflowsFile)
	if err != nil {
		g.Logger.Errorf("Error opening workflows.json: %v", err)
		return "", err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&workflows)
	if err != nil {
		g.Logger.Errorf("Error decoding workflows.json: %v", err)
		return "", err
	}

	workflowContent, exists := workflows[workflowName]
	if !exists {
		return "", fmt.Errorf("workflow '%s' not found", workflowName)
	}

	return workflowContent, nil
}