// encrypt.go

package main

import (
	"crypto/rand"
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
	decodedPublicKey, err := base64.StdEncoding.DecodeString(*publicKey.Key)
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}

	// Check that the public key is 32 bytes (256 bits)
	if len(decodedPublicKey) != 32 {
		return "", fmt.Errorf("invalid public key length: expected 32 bytes, got %d bytes", len(decodedPublicKey))
	}

	// Convert the public key into [32]byte format
	var publicKeyBytes [32]byte
	copy(publicKeyBytes[:], decodedPublicKey)

	// Encrypt the secret value using NaCl's box.SealAnonymous
	encryptedBytes, err := box.SealAnonymous(nil, []byte(secretValue), &publicKeyBytes, rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Encode the encrypted secret to base64
	encryptedValue := base64.StdEncoding.EncodeToString(encryptedBytes)

	return encryptedValue, nil
}