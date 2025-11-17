package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultctl/vaultctl/internal/crypto"
	"github.com/vaultctl/vaultctl/internal/storage"
	"github.com/vaultctl/vaultctl/internal/vault"
)

// decryptVaultFromEncrypted decrypts a vault from an EncryptedVault structure
func decryptVaultFromEncrypted(ev *storage.EncryptedVault, masterPassword []byte) (*vault.Vault, []byte, error) {
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

// saveVault saves the unlocked vault to local storage and optionally syncs to DynamoDB
func saveVault(cmd *cobra.Command, syncToDynamo bool) error {
	if unlockedVault == nil {
		return fmt.Errorf("vault is not unlocked")
	}

	// Load current encrypted vault to preserve metadata
	ev, err := localStore.LoadEncryptedVault()
	if err != nil {
		return fmt.Errorf("failed to load encrypted vault: %w", err)
	}

	// Encrypt and save locally
	if err := localStore.EncryptAndSave(unlockedVault, vaultKey, ev); err != nil {
		return fmt.Errorf("failed to save vault: %w", err)
	}

	// Sync to DynamoDB if requested
	if syncToDynamo && dynamoStore != nil {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = cmd.Root().Context()
		}
		if err := dynamoStore.SaveVault(ctx, ev, ev.Version-1); err != nil {
			return fmt.Errorf("failed to sync to DynamoDB: %w", err)
		}
	}

	return nil
}

