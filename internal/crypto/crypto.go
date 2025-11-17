package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	// Salt sizes
	SaltSize = 32

	// Key sizes
	MasterKeySize = 32
	VaultKeySize  = 32

	// Nonce size for XChaCha20-Poly1305
	NonceSize = 24

	// Argon2id parameters
	DefaultMemory      = 64 * 1024 // 64 MB
	DefaultIterations  = 3
	DefaultParallelism = 1
)

// KDFParams holds Argon2id parameters
type KDFParams struct {
	Algo       string `json:"algo"`
	Memory     uint32 `json:"memory"`
	Iterations uint32 `json:"iterations"`
	Parallelism uint8 `json:"parallelism"`
}

// DefaultKDFParams returns sensible default parameters
func DefaultKDFParams() KDFParams {
	return KDFParams{
		Algo:       "argon2id",
		Memory:     DefaultMemory,
		Iterations: DefaultIterations,
		Parallelism: DefaultParallelism,
	}
}

// DeriveMasterKey derives a master key from a password using Argon2id
func DeriveMasterKey(password []byte, salt []byte, params KDFParams) []byte {
	return argon2.IDKey(password, salt, params.Iterations, params.Memory, params.Parallelism, MasterKeySize)
}

// GenerateSalt generates a random salt
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// GenerateVaultKey generates a random vault key
func GenerateVaultKey() ([]byte, error) {
	key := make([]byte, VaultKeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate vault key: %w", err)
	}
	return key, nil
}

// EncryptVaultKey encrypts the vault key with the master key
func EncryptVaultKey(vaultKey []byte, masterKey []byte) ([]byte, []byte, error) {
	aead, err := chacha20poly1305.NewX(masterKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, vaultKey, nil)
	return ciphertext, nonce, nil
}

// DecryptVaultKey decrypts the vault key with the master key
func DecryptVaultKey(encryptedVaultKey []byte, nonce []byte, masterKey []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(nonce) != NonceSize {
		return nil, errors.New("invalid nonce size")
	}

	plaintext, err := aead.Open(nil, nonce, encryptedVaultKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt vault key: %w", err)
	}

	return plaintext, nil
}

// Encrypt encrypts data using XChaCha20-Poly1305
func Encrypt(plaintext []byte, key []byte) ([]byte, []byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt decrypts data using XChaCha20-Poly1305
func Decrypt(ciphertext []byte, nonce []byte, key []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(nonce) != NonceSize {
		return nil, errors.New("invalid nonce size")
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncodeBase64 encodes bytes to base64 string
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes base64 string to bytes
func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// ConstantTimeCompare performs constant-time comparison
func ConstantTimeCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// Zeroize overwrites a byte slice with zeros to clear sensitive data from memory
func Zeroize(data []byte) {
	if data != nil {
		for i := range data {
			data[i] = 0
		}
	}
}

