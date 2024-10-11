// encrypt.go

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/google/go-github/v50/github"
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

	// Parse the public key (Assuming it's in PKIX format)
	parsedPubKey, err := x509.ParsePKIXPublicKey(keyBytes)
	if err != nil {
		// Try parsing as PKCS1
		rsaPubKey, parseErr := x509.ParsePKCS1PublicKey(keyBytes)
		if parseErr != nil {
			return "", fmt.Errorf("failed to parse RSA public key: %w", err)
		}
		parsedPubKey = rsaPubKey
	}

	rsaPubKey, ok := parsedPubKey.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("public key is not an RSA public key")
	}

	// Encrypt the secret value using RSA-OAEP with SHA-256
	label := []byte("") // No label
	hash := sha256.New()
	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, rsaPubKey, []byte(secretValue), label)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Encode the ciphertext to base64
	encryptedValue := base64.StdEncoding.EncodeToString(ciphertext)

	return encryptedValue, nil
}