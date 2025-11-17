package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vaultctl/vaultctl/internal/crypto"
)

const (
	// Default session timeout (30 minutes)
	DefaultSessionTimeout = 30 * time.Minute
	// Session file permissions (read/write for user only)
	SessionFileMode = 0600
)

// SessionData represents the encrypted session data
type SessionData struct {
	EncryptedVaultKey string    `json:"encrypted_vault_key"` // base64
	Nonce             string    `json:"nonce"`                // base64
	SessionKey        string    `json:"session_key"`          // base64 - encrypted session key
	SessionKeyNonce   string    `json:"session_key_nonce"`    // base64 - nonce for session key encryption
	CreatedAt         time.Time `json:"created_at"`
	ExpiresAt         time.Time `json:"expires_at"`
}

// SessionManager handles session management
type SessionManager struct {
	sessionPath string
	sessionKey  []byte
	timeout     time.Duration
}

// NewSessionManager creates a new session manager
func NewSessionManager(sessionPath string, timeout time.Duration) *SessionManager {
	return &SessionManager{
		sessionPath: sessionPath,
		timeout:     timeout,
	}
}

// getMasterKey derives a master key from user-specific data for encrypting session keys
func (sm *SessionManager) getMasterKey() ([]byte, error) {
	// Use user's home directory as a source for key derivation
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Also use username for additional entropy
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}

	// Create a salt from home directory and username
	salt := []byte(fmt.Sprintf("%s:%s:vaultctl", homeDir, username))

	// Derive a key using a simple hash (for session key encryption only)
	// This is not for password derivation, just for encrypting the session key
	key := crypto.DeriveMasterKey([]byte(homeDir+username), salt, crypto.KDFParams{
		Algo:       "argon2id",
		Memory:     32 * 1024, // 32 MB
		Iterations: 2,
		Parallelism: 1,
	})

	return key, nil
}

// GetSessionKey gets or creates a session key, loading from session file if available
func (sm *SessionManager) GetSessionKey() ([]byte, error) {
	if sm.sessionKey != nil {
		return sm.sessionKey, nil
	}

	// Try to load from session file
	if _, err := os.Stat(sm.sessionPath); err == nil {
		data, err := os.ReadFile(sm.sessionPath)
		if err == nil {
			var sessionData SessionData
			if json.Unmarshal(data, &sessionData) == nil && sessionData.SessionKey != "" {
				// Decrypt the session key
				masterKey, err := sm.getMasterKey()
				if err != nil {
					return nil, fmt.Errorf("failed to get master key: %w", err)
				}

				encrypted, err := crypto.DecodeBase64(sessionData.SessionKey)
				if err != nil {
					return nil, fmt.Errorf("failed to decode session key: %w", err)
				}

				nonce, err := crypto.DecodeBase64(sessionData.SessionKeyNonce)
				if err != nil {
					return nil, fmt.Errorf("failed to decode session key nonce: %w", err)
				}

				sessionKey, err := crypto.Decrypt(encrypted, nonce, masterKey)
				if err == nil {
					sm.sessionKey = sessionKey
					return sessionKey, nil
				}
			}
		}
	}

	// Generate new session key
	sessionKey, err := crypto.GenerateVaultKey() // 32 bytes
	if err != nil {
		return nil, fmt.Errorf("failed to generate session key: %w", err)
	}

	sm.sessionKey = sessionKey
	return sessionKey, nil
}

// SaveSession saves the vault key encrypted with session key
func (sm *SessionManager) SaveSession(vaultKey []byte) error {
	sessionKey, err := sm.GetSessionKey()
	if err != nil {
		return fmt.Errorf("failed to get session key: %w", err)
	}

	// Encrypt vault key with session key
	encrypted, nonce, err := crypto.Encrypt(vaultKey, sessionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt vault key: %w", err)
	}

	// Encrypt and store the session key itself (so it persists across processes)
	masterKey, err := sm.getMasterKey()
	if err != nil {
		return fmt.Errorf("failed to get master key: %w", err)
	}

	encryptedSessionKey, sessionKeyNonce, err := crypto.Encrypt(sessionKey, masterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt session key: %w", err)
	}

	now := time.Now()
	sessionData := SessionData{
		EncryptedVaultKey: crypto.EncodeBase64(encrypted),
		Nonce:             crypto.EncodeBase64(nonce),
		SessionKey:        crypto.EncodeBase64(encryptedSessionKey),
		SessionKeyNonce:   crypto.EncodeBase64(sessionKeyNonce),
		CreatedAt:         now,
		ExpiresAt:         now.Add(sm.timeout),
	}

	// Ensure directory exists
	dir := filepath.Dir(sm.sessionPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Write session file
	data, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	if err := os.WriteFile(sm.sessionPath, data, SessionFileMode); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// LoadSession loads and decrypts the vault key from session
func (sm *SessionManager) LoadSession() ([]byte, error) {
	// Check if session file exists
	if _, err := os.Stat(sm.sessionPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no active session")
	}

	// Read session file
	data, err := os.ReadFile(sm.sessionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var sessionData SessionData
	if err := json.Unmarshal(data, &sessionData); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	// Check if session expired
	if time.Now().After(sessionData.ExpiresAt) {
		sm.ClearSession()
		return nil, fmt.Errorf("session expired")
	}

	// Decrypt the session key from session data
	if sessionData.SessionKey == "" || sessionData.SessionKeyNonce == "" {
		return nil, fmt.Errorf("session key not found in session data")
	}

	masterKey, err := sm.getMasterKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get master key: %w", err)
	}

	encrypted, err := crypto.DecodeBase64(sessionData.SessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode session key: %w", err)
	}

	nonce, err := crypto.DecodeBase64(sessionData.SessionKeyNonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decode session key nonce: %w", err)
	}

	sessionKey, err := crypto.Decrypt(encrypted, nonce, masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt session key: %w", err)
	}
	sm.sessionKey = sessionKey

	// Decrypt vault key
	encryptedVaultKey, err := crypto.DecodeBase64(sessionData.EncryptedVaultKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted vault key: %w", err)
	}

	vaultKeyNonce, err := crypto.DecodeBase64(sessionData.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	vaultKey, err := crypto.Decrypt(encryptedVaultKey, vaultKeyNonce, sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt vault key: %w", err)
	}

	return vaultKey, nil
}

// ClearSession removes the session file
func (sm *SessionManager) ClearSession() error {
	if _, err := os.Stat(sm.sessionPath); os.IsNotExist(err) {
		return nil // Already cleared
	}

	if err := os.Remove(sm.sessionPath); err != nil {
		return fmt.Errorf("failed to remove session file: %w", err)
	}

	sm.sessionKey = nil

	return nil
}

// HasActiveSession checks if there's an active session
func (sm *SessionManager) HasActiveSession() bool {
	vaultKey, err := sm.LoadSession()
	return err == nil && vaultKey != nil
}

// GetSessionPath returns the session file path
func (sm *SessionManager) GetSessionPath() string {
	return sm.sessionPath
}

