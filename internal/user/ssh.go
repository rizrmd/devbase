package user

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// SSHKey represents an SSH public key
type SSHKey struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	PublicKey string `json:"public_key"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// SSHManager handles SSH key operations
type SSHManager struct {
	db          *sql.DB
	baseHomeDir string
}

// NewSSHManager creates a new SSHManager
func NewSSHManager(db *sql.DB, baseHomeDir string) *SSHManager {
	return &SSHManager{
		db:          db,
		baseHomeDir: baseHomeDir,
	}
}

// AddKey adds an SSH public key for a user
func (s *SSHManager) AddKey(username, publicKey, name string) error {
	// Verify user exists
	var exists bool
	err := s.db.QueryRow("SELECT COUNT(*) > 0 FROM users WHERE username = ?", username).Scan(&exists)
	if err != nil || !exists {
		return fmt.Errorf("user not found")
	}

	// Insert into database
	result, err := s.db.Exec(
		"INSERT INTO ssh_keys (username, public_key, name) VALUES (?, ?, ?)",
		username, publicKey, name,
	)
	if err != nil {
		return fmt.Errorf("failed to save key: %w", err)
	}

	// Write to authorized_keys file
	if err := s.writeAuthorizedKeys(username); err != nil {
		// Rollback database insert
		s.db.Exec("DELETE FROM ssh_keys WHERE id = (SELECT last_insert_rowid())")
		return fmt.Errorf("failed to write authorized_keys: %w", err)
	}

	// Set proper permissions
	homeDir := filepath.Join(s.baseHomeDir, username)
	sshDir := filepath.Join(homeDir, ".ssh")
	authKeysFile := filepath.Join(sshDir, "authorized_keys")

	// Ensure .ssh directory has correct permissions
	if err := os.Chmod(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to set .ssh permissions: %w", err)
	}

	// Ensure authorized_keys has correct permissions
	if err := os.Chmod(authKeysFile, 0600); err != nil {
		return fmt.Errorf("failed to set authorized_keys permissions: %w", err)
	}

	// Set ownership to the user
	if err := s.setFileOwnership(username, sshDir, authKeysFile); err != nil {
		return fmt.Errorf("failed to set file ownership: %w", err)
	}

	_, _ = result.RowsAffected()
	return nil
}

// writeAuthorizedKeys writes all keys for a user to authorized_keys file
func (s *SSHManager) writeAuthorizedKeys(username string) error {
	// Fetch all keys for user
	rows, err := s.db.Query("SELECT public_key FROM ssh_keys WHERE username = ?", username)
	if err != nil {
		return err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return err
		}
		keys = append(keys, key)
	}

	// Create .ssh directory if it doesn't exist
	homeDir := filepath.Join(s.baseHomeDir, username)
	sshDir := filepath.Join(homeDir, ".ssh")

	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	// Write keys to authorized_keys
	authKeysFile := filepath.Join(sshDir, "authorized_keys")
	file, err := os.Create(authKeysFile)
	if err != nil {
		return fmt.Errorf("failed to create authorized_keys: %w", err)
	}
	defer file.Close()

	for _, key := range keys {
		if _, err := file.WriteString(key + "\n"); err != nil {
			return fmt.Errorf("failed to write key: %w", err)
		}
	}

	return nil
}

// setFileOwnership sets file ownership to the specified user
func (s *SSHManager) setFileOwnership(username string, paths ...string) error {
	for _, path := range paths {
		cmd := exec.Command("sudo", "chown", "-R", fmt.Sprintf("%s:%s", username, username), path)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("chown failed for %s: %s: %w", path, string(output), err)
		}
	}
	return nil
}

// ListKeys returns all SSH keys for a user
func (s *SSHManager) ListKeys(username string) ([]SSHKey, error) {
	rows, err := s.db.Query(
		"SELECT id, username, public_key, name, created_at FROM ssh_keys WHERE username = ? ORDER BY created_at DESC",
		username,
	)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	var keys []SSHKey
	for rows.Next() {
		var k SSHKey
		err := rows.Scan(&k.ID, &k.Username, &k.PublicKey, &k.Name, &k.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		keys = append(keys, k)
	}

	return keys, nil
}

// DeleteKey removes an SSH key
func (s *SSHManager) DeleteKey(keyID int, username string) error {
	// Verify key belongs to user
	var owner string
	err := s.db.QueryRow("SELECT username FROM ssh_keys WHERE id = ?", keyID).Scan(&owner)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("key not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	if owner != username {
		return fmt.Errorf("key does not belong to user")
	}

	_ = owner // Use the variable

	// Delete from database
	_, err = s.db.Exec("DELETE FROM ssh_keys WHERE id = ?", keyID)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	// Rewrite authorized_keys
	if err := s.writeAuthorizedKeys(username); err != nil {
		return fmt.Errorf("failed to update authorized_keys: %w", err)
	}

	return nil
}

// GetKey returns a single SSH key
func (s *SSHManager) GetKey(keyID int, username string) (*SSHKey, error) {
	var k SSHKey

	err := s.db.QueryRow(
		"SELECT id, username, public_key, name, created_at FROM ssh_keys WHERE id = ?",
		keyID,
	).Scan(&k.ID, &k.Username, &k.PublicKey, &k.Name, &k.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("key not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if k.Username != username {
		return nil, fmt.Errorf("key does not belong to user")
	}

	return &k, nil
}
