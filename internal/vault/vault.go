package vault

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const SchemaVersion = 1

// Entry represents a single password entry
type Entry struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Username    string    `json:"username"`
	Password    []byte    `json:"password"` // Stored as base64 in JSON for security
	URL         string    `json:"url"`
	Notes       string    `json:"notes"`
	BackupCodes []string  `json:"backup_codes,omitempty"` // 2FA/authenticator backup codes
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UnmarshalJSON custom unmarshaler for backward compatibility
// Handles both old string format and new []byte format
func (e *Entry) UnmarshalJSON(data []byte) error {
	// Use a temporary struct to handle both formats
	type Alias Entry
	aux := &struct {
		Password interface{} `json:"password"` // Can be string or []byte
		*Alias
	}{
		Alias: (*Alias)(e),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle password field - can be string (old format) or base64 []byte (new format)
	if aux.Password != nil {
		switch v := aux.Password.(type) {
		case string:
			// Old format: string, or new format as base64 string
			// Try to decode as base64 first
			if decoded, err := base64.StdEncoding.DecodeString(v); err == nil {
				e.Password = decoded
			} else {
				// Not base64, treat as plain string (old format)
				e.Password = []byte(v)
			}
		case []byte:
			// Already []byte
			e.Password = v
		}
	}

	return nil
}

// Vault represents the plaintext vault structure
type Vault struct {
	SchemaVersion int     `json:"schema_version"`
	VaultID       string  `json:"vault_id"`
	Entries       []Entry `json:"entries"`
}

// NewVault creates a new empty vault
func NewVault() *Vault {
	return &Vault{
		SchemaVersion: SchemaVersion,
		VaultID:       uuid.New().String(),
		Entries:       make([]Entry, 0),
	}
}

// AddEntry adds a new entry to the vault
func (v *Vault) AddEntry(name, username string, password []byte, url, notes string, backupCodes []string) *Entry {
	now := time.Now()
	// Make a copy of the password to avoid external modifications
	passwordCopy := make([]byte, len(password))
	copy(passwordCopy, password)
	
	entry := Entry{
		ID:          uuid.New().String(),
		Name:        name,
		Username:    username,
		Password:    passwordCopy,
		URL:         url,
		Notes:       notes,
		BackupCodes: backupCodes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	v.Entries = append(v.Entries, entry)
	return &entry
}

// GetEntry finds an entry by ID or name
func (v *Vault) GetEntry(identifier string) *Entry {
	for i := range v.Entries {
		if v.Entries[i].ID == identifier || v.Entries[i].Name == identifier {
			return &v.Entries[i]
		}
	}
	return nil
}

// RemoveEntry removes an entry by ID or name
func (v *Vault) RemoveEntry(identifier string) bool {
	for i, entry := range v.Entries {
		if entry.ID == identifier || entry.Name == identifier {
			v.Entries = append(v.Entries[:i], v.Entries[i+1:]...)
			return true
		}
	}
	return false
}

// ListEntries returns all entries (without passwords for listing)
type EntrySummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (v *Vault) ListEntries() []EntrySummary {
	summaries := make([]EntrySummary, len(v.Entries))
	for i, entry := range v.Entries {
		summaries[i] = EntrySummary{
			ID:        entry.ID,
			Name:      entry.Name,
			Username:  entry.Username,
			URL:       entry.URL,
			CreatedAt: entry.CreatedAt,
			UpdatedAt: entry.UpdatedAt,
		}
	}
	return summaries
}

// UpdateEntry updates an existing entry
func (v *Vault) UpdateEntry(identifier string, name, username string, password []byte, url, notes string, backupCodes []string) bool {
	entry := v.GetEntry(identifier)
	if entry == nil {
		return false
	}

	if name != "" {
		entry.Name = name
	}
	if username != "" {
		entry.Username = username
	}
	if password != nil && len(password) > 0 {
		// Make a copy to avoid external modifications
		passwordCopy := make([]byte, len(password))
		copy(passwordCopy, password)
		entry.Password = passwordCopy
	}
	if url != "" {
		entry.URL = url
	}
	if notes != "" {
		entry.Notes = notes
	}
	if backupCodes != nil {
		entry.BackupCodes = backupCodes
	}
	entry.UpdatedAt = time.Now()
	return true
}

// ToJSON serializes the vault to JSON
func (v *Vault) ToJSON() ([]byte, error) {
	return json.Marshal(v)
}

// FromJSON deserializes the vault from JSON
func FromJSON(data []byte) (*Vault, error) {
	var v Vault
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
