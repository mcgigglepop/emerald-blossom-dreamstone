package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vaultctl/vaultctl/internal/crypto"
	"github.com/vaultctl/vaultctl/internal/vault"
)

// LocalStorage handles local encrypted vault file operations
type LocalStorage struct {
	VaultPath string
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(vaultPath string) *LocalStorage {
	return &LocalStorage{
		VaultPath: vaultPath,
	}
}

// EnsureDir ensures the vault directory exists
func (ls *LocalStorage) EnsureDir() error {
	dir := filepath.Dir(ls.VaultPath)
	return os.MkdirAll(dir, 0700)
}

// SaveEncryptedVault saves an encrypted vault to disk
func (ls *LocalStorage) SaveEncryptedVault(ev *EncryptedVault) error {
	if err := ls.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create vault directory: %w", err)
	}

	data, err := ev.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize encrypted vault: %w", err)
	}

	if err := os.WriteFile(ls.VaultPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write vault file: %w", err)
	}

	return nil
}

// LoadEncryptedVault loads an encrypted vault from disk
func (ls *LocalStorage) LoadEncryptedVault() (*EncryptedVault, error) {
	data, err := os.ReadFile(ls.VaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("vault not found at %s. Run 'vaultctl init' first", ls.VaultPath)
		}
		return nil, fmt.Errorf("failed to read vault file: %w", err)
	}

	ev, err := EncryptedVaultFromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vault file: %w", err)
	}

	return ev, nil
}

// Exists checks if the vault file exists
func (ls *LocalStorage) Exists() bool {
	_, err := os.Stat(ls.VaultPath)
	return err == nil
}

// EncryptAndSave encrypts a vault and saves it locally
func (ls *LocalStorage) EncryptAndSave(v *vault.Vault, vaultKey []byte, ev *EncryptedVault) error {
	plaintext, err := v.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize vault: %w", err)
	}

	ciphertext, nonce, err := crypto.Encrypt(plaintext, vaultKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt vault: %w", err)
	}

	ev.Ciphertext = crypto.EncodeBase64(ciphertext)
	ev.Nonce = crypto.EncodeBase64(nonce)
	ev.SetModifiedAt(time.Now())
	ev.Version++

	return ls.SaveEncryptedVault(ev)
}

// DecryptAndLoad decrypts and loads a vault from local storage
func (ls *LocalStorage) DecryptAndLoad(masterPassword []byte) (*vault.Vault, []byte, error) {
	ev, err := ls.LoadEncryptedVault()
	if err != nil {
		return nil, nil, err
	}

	return decryptVault(ev, masterPassword)
}

// decryptVault is a helper that decrypts a vault from an EncryptedVault
func decryptVault(ev *EncryptedVault, masterPassword []byte) (*vault.Vault, []byte, error) {
	// Decode salt and encrypted vault key
	salt, err := crypto.DecodeBase64(ev.SaltMaster)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	encVaultKey, err := crypto.DecodeBase64(ev.EncVaultKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode encrypted vault key: %w", err)
	}

	// Derive master key
	kdfParams := crypto.KDFParams{
		Algo:       ev.KDFParams.Algo,
		Memory:     ev.KDFParams.Memory,
		Iterations: ev.KDFParams.Iterations,
		Parallelism: ev.KDFParams.Parallelism,
	}
	masterKey := crypto.DeriveMasterKey(masterPassword, salt, kdfParams)

	// Decrypt vault key
	var vaultKeyNonce []byte
	if ev.VaultKeyNonce != "" {
		vaultKeyNonce, err = crypto.DecodeBase64(ev.VaultKeyNonce)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode vault key nonce: %w", err)
		}
	} else {
		// Backward compatibility: if vault_key_nonce doesn't exist, use nonce
		// This handles old vaults created before we added the separate nonce field
		vaultKeyNonce, err = crypto.DecodeBase64(ev.Nonce)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode nonce: %w", err)
		}
	}
	
	vaultKey, err := crypto.DecryptVaultKey(encVaultKey, vaultKeyNonce, masterKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt vault key: %w", err)
	}

	// Decode ciphertext and nonce
	ciphertext, err := crypto.DecodeBase64(ev.Ciphertext)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonce, err := crypto.DecodeBase64(ev.Nonce)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	// Decrypt vault
	plaintext, err := crypto.Decrypt(ciphertext, nonce, vaultKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt vault: %w", err)
	}

	// Deserialize vault
	v, err := vault.FromJSON(plaintext)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize vault: %w", err)
	}

	return v, vaultKey, nil
}

