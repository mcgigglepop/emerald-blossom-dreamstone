package storage

import (
	"encoding/json"
	"time"
)

// EncryptedVault represents the encrypted vault format stored on disk and in DynamoDB
type EncryptedVault struct {
	SchemaVersion  int       `json:"schema_version"`
	VaultID        string    `json:"vault_id"`
	SaltMaster     string    `json:"salt_master"`      // base64
	EncVaultKey    string    `json:"enc_vault_key"`    // base64
	VaultKeyNonce  string    `json:"vault_key_nonce"`  // base64 - nonce for vault key encryption
	KDFParams      KDFParams `json:"kdf_params"`
	Cipher         string    `json:"cipher"`
	Ciphertext     string    `json:"ciphertext"`       // base64
	Nonce          string    `json:"nonce"`            // base64 - nonce for vault ciphertext
	ModifiedAt     string    `json:"modified_at"`     // ISO 8601
	Version        int64     `json:"version"`
}

// KDFParams holds Argon2id parameters
type KDFParams struct {
	Algo       string `json:"algo"`
	Memory     uint32 `json:"memory"`
	Iterations uint32 `json:"iterations"`
	Parallelism uint8 `json:"parallelism"`
}

// ToJSON serializes the encrypted vault to JSON
func (ev *EncryptedVault) ToJSON() ([]byte, error) {
	return json.Marshal(ev)
}

// FromJSON deserializes the encrypted vault from JSON
func EncryptedVaultFromJSON(data []byte) (*EncryptedVault, error) {
	var ev EncryptedVault
	if err := json.Unmarshal(data, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

// GetModifiedAtTime parses the ModifiedAt timestamp
func (ev *EncryptedVault) GetModifiedAtTime() (time.Time, error) {
	return time.Parse(time.RFC3339, ev.ModifiedAt)
}

// SetModifiedAt sets the ModifiedAt timestamp
func (ev *EncryptedVault) SetModifiedAt(t time.Time) {
	ev.ModifiedAt = t.Format(time.RFC3339)
}

