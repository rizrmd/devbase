package user

import (
	"database/sql"
	"fmt"
	"os/exec"
	"strings"

	_ "modernc.org/sqlite"
)

// User represents a devbase user
type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	IsActive  bool   `json:"is_active"`
}

// Manager handles user management operations
type Manager struct {
	db          *sql.DB
	baseHomeDir string
}

// New creates a new Manager instance
func New(db *sql.DB, baseHomeDir string) *Manager {
	return &Manager{
		db:          db,
		baseHomeDir: baseHomeDir,
	}
}

// CreateUser creates a new user with home directory and system access
func (m *Manager) CreateUser(username, password, email string) error {
	// Validate username
	if err := validateUsername(username); err != nil {
		return err
	}

	// Check if user already exists
	var exists bool
	err := m.db.QueryRow("SELECT COUNT(*) > 0 FROM users WHERE username = ?", username).Scan(&exists)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if exists {
		return fmt.Errorf("user already exists")
	}

	// Create Linux user
	homeDir := fmt.Sprintf("%s/%s", m.baseHomeDir, username)
	if err := m.createSystemUser(username, password, homeDir); err != nil {
		return fmt.Errorf("failed to create system user: %w", err)
	}

	// Save to database
	_, err = m.db.Exec(
		"INSERT INTO users (username, email) VALUES (?, ?)",
		username, email,
	)
	if err != nil {
		// Rollback: delete system user
		m.deleteSystemUser(username)
		return fmt.Errorf("failed to save user to database: %w", err)
	}

	return nil
}

// createSystemUser creates a Linux user with home directory
func (m *Manager) createSystemUser(username, password, homeDir string) error {
	// Create user with home directory
	cmd := exec.Command("sudo", "useradd", "-m", "-d", homeDir, "-s", "/bin/bash", username)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("useradd failed: %s: %w", string(output), err)
	}

	// Add to sudo group
	cmd = exec.Command("sudo", "usermod", "-aG", "sudo", username)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("usermod failed: %s: %w", string(output), err)
	}

	// Set password
	cmd = exec.Command("sudo", "chpasswd")
	cmd.Stdin = strings.NewReader(fmt.Sprintf("%s:%s", username, password))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chpasswd failed: %s: %w", string(output), err)
	}

	return nil
}

// ListUsers returns all users
func (m *Manager) ListUsers() ([]User, error) {
	rows, err := m.db.Query("SELECT id, username, email, created_at, is_active FROM users ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var isActive int
		err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt, &isActive)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		u.IsActive = isActive == 1
		users = append(users, u)
	}

	return users, nil
}

// GetUser retrieves a user by username
func (m *Manager) GetUser(username string) (*User, error) {
	var u User
	var isActive int

	err := m.db.QueryRow(
		"SELECT id, username, email, created_at, is_active FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt, &isActive)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	u.IsActive = isActive == 1
	return &u, nil
}

// UpdateUser updates user email
func (m *Manager) UpdateUser(username, email string) error {
	result, err := m.db.Exec(
		"UPDATE users SET email = ? WHERE username = ?",
		email, username,
	)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// DeleteUser deletes a user
func (m *Manager) DeleteUser(username string) error {
	// Check if user exists
	_, err := m.GetUser(username)
	if err != nil {
		return err
	}

	// Delete from database first
	_, err = m.db.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		return fmt.Errorf("failed to delete from database: %w", err)
	}

	// Delete system user (this also removes home directory with -r flag)
	if err := m.deleteSystemUser(username); err != nil {
		return fmt.Errorf("failed to delete system user: %w", err)
	}

	return nil
}

// deleteSystemUser removes a Linux user
func (m *Manager) deleteSystemUser(username string) error {
	cmd := exec.Command("sudo", "userdel", "-r", username)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("userdel failed: %s: %w", string(output), err)
	}
	return nil
}

// validateUsername checks if username is valid
func validateUsername(username string) error {
	if len(username) < 3 || len(username) > 32 {
		return fmt.Errorf("username must be between 3 and 32 characters")
	}

	// Check for alphanumeric and underscore only
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return fmt.Errorf("username can only contain letters, numbers, underscores, and hyphens")
		}
	}

	// Must start with letter
	if !((username[0] >= 'a' && username[0] <= 'z') || (username[0] >= 'A' && username[0] <= 'Z')) {
		return fmt.Errorf("username must start with a letter")
	}

	return nil
}

// UserExists checks if a user exists
func (m *Manager) UserExists(username string) bool {
	var exists bool
	err := m.db.QueryRow("SELECT COUNT(*) > 0 FROM users WHERE username = ?", username).Scan(&exists)
	return err == nil && exists
}
