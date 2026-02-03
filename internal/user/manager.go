package user

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// User represents a devbase user
type User struct {
	Username  string `json:"username"`
	HomeDir   string `json:"home_dir"`
	CreatedAt string `json:"created_at,omitempty"`
}

// Manager handles user management operations using system commands only
type Manager struct {
	baseHomeDir string
}

// New creates a new Manager instance
func New(baseHomeDir string) *Manager {
	return &Manager{
		baseHomeDir: baseHomeDir,
	}
}

// CreateUser creates a new Linux user with home directory
func (m *Manager) CreateUser(username, password string) error {
	// Validate username
	if err := validateUsername(username); err != nil {
		return err
	}

	// Check if user already exists
	if m.UserExists(username) {
		return fmt.Errorf("user already exists")
	}

	// Create user with system commands
	homeDir := filepath.Join(m.baseHomeDir, username)
	if err := m.createSystemUser(username, password, homeDir); err != nil {
		return fmt.Errorf("failed to create system user: %w", err)
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

// ListUsers returns all users in /devbase directory
func (m *Manager) ListUsers() ([]User, error) {
	// Read /devbase directory to find user home directories
	entries, err := os.ReadDir(m.baseHomeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read base directory: %w", err)
	}

	var users []User
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		username := entry.Name()

		// Skip hidden directories
		if strings.HasPrefix(username, ".") {
			continue
		}

		// Verify this is an actual system user by checking passwd
		if !m.isSystemUser(username) {
			continue
		}

		homeDir := filepath.Join(m.baseHomeDir, username)

		// Get creation time from home directory
		info, err := entry.Info()
		if err != nil {
			continue
		}

		users = append(users, User{
			Username:  username,
			HomeDir:   homeDir,
			CreatedAt: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	return users, nil
}

// isSystemUser checks if username exists in /etc/passwd
func (m *Manager) isSystemUser(username string) bool {
	cmd := exec.Command("getent", "passwd", username)
	return cmd.Run() == nil
}

// GetUser retrieves a user by username
func (m *Manager) GetUser(username string) (*User, error) {
	if !m.UserExists(username) {
		return nil, fmt.Errorf("user not found")
	}

	homeDir := filepath.Join(m.baseHomeDir, username)

	// Get home directory info
	info, err := os.Stat(homeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read home directory: %w", err)
	}

	return &User{
		Username:  username,
		HomeDir:   homeDir,
		CreatedAt: info.ModTime().Format("2006-01-02 15:04:05"),
	}, nil
}

// UpdateUser is a no-op (no email to update)
func (m *Manager) UpdateUser(username, _ string) error {
	if !m.UserExists(username) {
		return fmt.Errorf("user not found")
	}
	// No-op since we removed email field
	return nil
}

// DeleteUser deletes a user
func (m *Manager) DeleteUser(username string) error {
	// Check if user exists
	if !m.UserExists(username) {
		return fmt.Errorf("user not found")
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

// UserExists checks if a user exists in the system
func (m *Manager) UserExists(username string) bool {
	return m.isSystemUser(username)
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
