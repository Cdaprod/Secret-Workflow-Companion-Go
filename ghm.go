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
}

// GHMImpl is the concrete implementation of the GHM interface
type GHMImpl struct {
	Token     string
	Encryptor Encryptor
}

// NewGHM creates a new instance of GHMImpl
func NewGHM(token string) GHM {
	return &GHMImpl{
		Token:     token,
		Encryptor: &EncryptorImpl{},
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
	}
	return strategy.Execute()
}