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

	// Perform encryption
	encryptedValue, err := EncryptSecret(*publicKey.Key, secretValue)
	if err != nil {
		return "", err
	}

	return encryptedValue, nil
}

// EncryptSecret performs the encryption using NaCl's box (sealed box implementation)
func EncryptSecret(publicKey string, secretValue string) (string, error) {
	// Decode the public key from base64
	pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}

	if len(pubKeyBytes) != 32 {
		return "", errors.New("invalid public key length")
	}

	var pubKey [32]byte
	copy(pubKey[:], pubKeyBytes)

	// Generate ephemeral keypair
	var ephPub, ephPriv [32]byte
	if _, err := rand.Read(ephPub[:]); err != nil {
		return "", fmt.Errorf("failed to generate ephemeral public key: %w", err)
	}
	if _, err := rand.Read(ephPriv[:]); err != nil {
		return "", fmt.Errorf("failed to generate ephemeral private key: %w", err)
	}

	// Generate a random nonce
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the secret using box.Seal
	ciphertext := box.Seal(nil, []byte(secretValue), &nonce, &pubKey, &ephPriv)

	// Combine ephemeral public key + nonce + ciphertext
	combined := append(ephPub[:], nonce[:]...)
	combined = append(combined, ciphertext...)

	// Base64 encode the combined data
	encryptedBase64 := base64.StdEncoding.EncodeToString(combined)

	return encryptedBase64, nil
}