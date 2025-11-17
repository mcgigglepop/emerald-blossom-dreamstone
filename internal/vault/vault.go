package vault

import (
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
	Password    string    `json:"password"`
	URL         string    `json:"url"`
	Notes       string    `json:"notes"`
	BackupCodes []string  `json:"backup_codes,omitempty"` // 2FA/authenticator backup codes
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
func (v *Vault) AddEntry(name, username, password, url, notes string, backupCodes []string) *Entry {
	now := time.Now()
	entry := Entry{
		ID:          uuid.New().String(),
		Name:        name,
		Username:    username,
		Password:    password,
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
func (v *Vault) UpdateEntry(identifier string, name, username, password, url, notes string, backupCodes []string) bool {
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
	if password != "" {
		entry.Password = password
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
