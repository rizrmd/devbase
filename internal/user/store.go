package user

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PersistedUser represents user data that needs to persist across container restarts
type PersistedUser struct {
	Username    string `json:"username"`
	Password    string `json:"password"` // Plain text password (will be used to set via chpasswd)
	HomeDir     string `json:"home_dir"`
	CreatedAt   string `json:"created_at"`
	UID         string `json:"uid,omitempty"`
	GID         string `json:"gid,omitempty"`
}

// UserStore handles persistent storage of user data
type UserStore struct {
	filePath string
	mu       sync.RWMutex
}

// NewUserStore creates a new UserStore instance
func NewUserStore(internalDir string) *UserStore {
	return &UserStore{
		filePath: filepath.Join(internalDir, "users.json"),
	}
}

// Load loads all users from the persistent store
func (s *UserStore) Load() ([]PersistedUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If file doesn't exist, return empty list
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return []PersistedUser{}, nil
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read users file: %w", err)
	}

	var users []PersistedUser
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("failed to parse users file: %w", err)
	}

	return users, nil
}

// Save saves users to the persistent store
func (s *UserStore) Save(users []PersistedUser) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}

	// Write to temporary file first, then rename for atomicity
	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write users file: %w", err)
	}

	if err := os.Rename(tmpPath, s.filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename users file: %w", err)
	}

	return nil
}

// AddUser adds a new user to the store
func (s *UserStore) AddUser(username, password, homeDir string) error {
	users, err := s.Load()
	if err != nil {
		return err
	}

	// Check if user already exists
	for _, u := range users {
		if u.Username == username {
			return fmt.Errorf("user already exists in store")
		}
	}

	users = append(users, PersistedUser{
		Username:  username,
		Password:  password,
		HomeDir:   homeDir,
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	})

	return s.Save(users)
}

// RemoveUser removes a user from the store
func (s *UserStore) RemoveUser(username string) error {
	users, err := s.Load()
	if err != nil {
		return err
	}

	var updated []PersistedUser
	found := false
	for _, u := range users {
		if u.Username != username {
			updated = append(updated, u)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("user not found in store")
	}

	return s.Save(updated)
}

// UpdatePassword updates a user's password in the store
func (s *UserStore) UpdatePassword(username, newPassword string) error {
	users, err := s.Load()
	if err != nil {
		return err
	}

	found := false
	for i := range users {
		if users[i].Username == username {
			users[i].Password = newPassword
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("user not found in store")
	}

	return s.Save(users)
}

// Exists checks if a user exists in the store
func (s *UserStore) Exists(username string) bool {
	users, err := s.Load()
	if err != nil {
		return false
	}

	for _, u := range users {
		if u.Username == username {
			return true
		}
	}

	return false
}
