// ghm.go
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/google/go-github/v50/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/nacl/box"
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

// Implement other methods like StoreConfig, AddSecretsToRepo, AddWorkflowsToRepo as needed.

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

// AddSecretStrategy defines the parameters for adding a secret
type AddSecretStrategy struct {
	Token       string
	Repo        string // Format: "owner/repo"
	SecretName  string
	SecretValue string
	Encryptor   Encryptor
	Logger      *logrus.Logger
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
		a.Logger.Error("Invalid repository format. Use 'owner/repo'.")
		return fmt.Errorf("invalid repository format")
	}
	owner, repo := parts[0], parts[1]

	// Fetch repository public key
	publicKey, _, err := client.Actions.GetRepoPublicKey(ctx, owner, repo)
	if err != nil {
		a.Logger.Errorf("Error fetching repository public key: %v", err)
		return err
	}

	// Ensure publicKey.KeyID is not nil
	if publicKey.KeyID == nil {
		a.Logger.Error("Public Key ID is nil.")
		return fmt.Errorf("public key ID is nil")
	}

	// Encrypt the secret value
	encryptedValue, err := a.Encryptor.Encrypt(a.SecretValue, publicKey)
	if err != nil {
		a.Logger.Errorf("Error encrypting secret: %v", err)
		return err
	}

	// Create the encrypted secret struct
	encryptedSecret := &github.EncryptedSecret{
		Name:           a.SecretName,
		KeyID:          *publicKey.KeyID,
		EncryptedValue: encryptedValue,
	}

	// Create or update the secret
	_, err = client.Actions.CreateOrUpdateRepoSecret(ctx, owner, repo, encryptedSecret)
	if err != nil {
		a.Logger.Errorf("Error setting repository secret: %v", err)
		return err
	}

	a.Logger.Infof("Secret '%s' added to repository '%s' successfully.", a.SecretName, a.Repo)
	a.Logger.Infof("Secret '%s' saved locally.", a.SecretName)

	// Save secret locally for persistence
	saveSecretLocally(a.SecretName, a.SecretValue, a.Logger)

	return nil
}

// AddWorkflowStrategy defines the parameters for adding a workflow
type AddWorkflowStrategy struct {
	Token        string
	Repo         string // Format: "owner/repo"
	WorkflowName string // e.g., "ci.yml"
	Content      string // YAML content of the workflow
	Logger       *logrus.Logger
}

// Execute adds a GitHub Actions workflow to a repository
func (a *AddWorkflowStrategy) Execute() error {
	// Split repo into owner and repo
	parts := strings.Split(a.Repo, "/")
	if len(parts) != 2 {
		a.Logger.Error("Invalid repository format. Use 'owner/repo'.")
		return fmt.Errorf("invalid repository format")
	}
	owner, repo := parts[0], parts[1]

	// GitHub repository URL
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)

	// Initialize authentication for git operations
	auth := &http.BasicAuth{
		Username: "ghm",   // Can be anything except an empty string
		Password: a.Token, // GitHub Personal Access Token
	}

	// Clone the repository into a temporary directory
	tmpDir, err := ioutil.TempDir("", "ghm-repo-")
	if err != nil {
		a.Logger.Errorf("Error creating temporary directory: %v", err)
		return err
	}
	defer os.RemoveAll(tmpDir) // Clean up after cloning

	a.Logger.Info("Cloning repository into temporary directory...")

	repoGit, err := git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
		Auth:     auth,
	})
	if err != nil {
		a.Logger.Errorf("Error cloning repository: %v", err)
		return err
	}

	worktree, err := repoGit.Worktree()
	if err != nil {
		a.Logger.Errorf("Error accessing worktree: %v", err)
		return err
	}

	// Create the workflow file in .github/workflows/
	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	err = os.MkdirAll(workflowDir, os.ModePerm)
	if err != nil {
		a.Logger.Errorf("Error creating workflow directory: %v", err)
		return err
	}

	workflowPath := filepath.Join(workflowDir, a.WorkflowName)
	err = ioutil.WriteFile(workflowPath, []byte(a.Content), 0644)
	if err != nil {
		a.Logger.Errorf("Error writing workflow file: %v", err)
		return err
	}

	// Stage the workflow file
	_, err = worktree.Add(filepath.Join(".github", "workflows", a.WorkflowName))
	if err != nil {
		a.Logger.Errorf("Error adding workflow file to git: %v", err)
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
		a.Logger.Errorf("Error committing changes: %v", err)
		return err
	}

	obj, err := repoGit.CommitObject(commit)
	if err != nil {
		a.Logger.Errorf("Error getting commit object: %v", err)
		return err
	}

	a.Logger.Infof("Committed changes: %s", obj.Hash)

	// Push the commit to GitHub
	a.Logger.Info("Pushing changes to GitHub...")
	err = repoGit.Push(&git.PushOptions{
		Auth: auth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		a.Logger.Errorf("Error pushing changes: %v", err)
		return err
	}

	a.Logger.Infof("Workflow '%s' added to repository '%s' successfully.", a.WorkflowName, a.Repo)

	return nil
}

// StoreConfigStrategy defines the parameters for storing a config key-value pair
type StoreConfigStrategy struct {
	ConfigKey   string
	ConfigValue string
	Logger      *logrus.Logger
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
				s.Logger.Errorf("Error creating config file: %v", err)
				return err
			}
		} else {
			s.Logger.Errorf("Error writing config: %v", err)
			return err
		}
	}
	s.Logger.Infof("Configuration '%s' saved successfully.", s.ConfigKey)
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

// saveSecretLocally saves the secret in a local JSON file for persistence
func saveSecretLocally(secretName, secretValue string, logger *logrus.Logger) {
	secretsFile := "secrets.json"
	secrets := make(map[string]string)

	// Check if secrets.json exists
	if _, err := os.Stat(secretsFile); err == nil {
		file, err := os.Open(secretsFile)
		if err == nil {
			defer file.Close()
			jsonDecoder := json.NewDecoder(file)
			err = jsonDecoder.Decode(&secrets)
			if err != nil {
				logger.Errorf("Error decoding secrets file: %v", err)
				return
			}
		} else {
			logger.Errorf("Error opening secrets file: %v", err)
			return
		}
	}

	secrets[secretName] = secretValue

	file, err := os.Create(secretsFile)
	if err != nil {
		logger.Errorf("Error saving secret locally: %v", err)
		return
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(secrets)
	if err != nil {
		logger.Errorf("Error encoding secrets: %v", err)
		return
	}

	logger.Infof("Secret '%s' saved locally.", secretName)
}