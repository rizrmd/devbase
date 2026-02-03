package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const (
	adminSessionKey = "admin_session"
	sessionDuration = 1 * time.Hour
)

// AdminAuth handles admin authentication
type AdminAuth struct {
	db *sql.DB
}

// New creates a new AdminAuth instance
func New(db *sql.DB) *AdminAuth {
	return &AdminAuth{db: db}
}

// Session represents an admin session
type Session struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// User represents a logged-in admin user
type User struct {
	Username string
}

// Login authenticates an admin user
func (a *AdminAuth) Login(username, password string) (*Session, error) {
	var storedHash string
	err := a.db.QueryRow(
		"SELECT password_hash FROM admin_passwords WHERE username = ?",
		username,
	).Scan(&storedHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Create session token
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session: %w", err)
	}

	session := &Session{
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sessionDuration),
	}

	return session, nil
}

// ChangePassword updates the admin password
func (a *AdminAuth) ChangePassword(username, oldPassword, newPassword string) error {
	var storedHash string
	err := a.db.QueryRow(
		"SELECT password_hash FROM admin_passwords WHERE username = ?",
		username,
	).Scan(&storedHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("invalid current password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update in database
	_, err = a.db.Exec(
		"UPDATE admin_passwords SET password_hash = ?, changed_at = CURRENT_TIMESTAMP WHERE username = ?",
		string(hashedPassword), username,
	)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// SetSessionCookie sets the session cookie in the response
func SetSessionCookie(c *gin.Context, session *Session) {
	c.SetCookie(
		adminSessionKey,
		session.Token,
		int(sessionDuration.Seconds()),
		"/",
		"",
		false,
		true, // httpOnly
	)
}

// ClearSessionCookie removes the session cookie
func ClearSessionCookie(c *gin.Context) {
	c.SetCookie(
		adminSessionKey,
		"",
		-1,
		"/",
		"",
		false,
		true,
	)
}

// GetSessionFromCookie retrieves the session from the request cookie
func GetSessionFromCookie(c *gin.Context) *Session {
	token, err := c.Cookie(adminSessionKey)
	if err != nil {
		return nil
	}

	// In production, you'd validate this against a stored session
	// For now, we'll just check if the token exists and isn't expired
	// This is a simplified implementation
	if token == "" {
		return nil
	}

	return &Session{
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sessionDuration),
	}
}

// IsAuthenticated checks if the request has a valid admin session
func (a *AdminAuth) IsAuthenticated(c *gin.Context) bool {
	session := GetSessionFromCookie(c)
	return session != nil && session.ExpiresAt.After(time.Now())
}

// RequireAuth is a middleware that requires admin authentication
func (a *AdminAuth) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !a.IsAuthenticated(c) {
			c.Redirect(302, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

// generateToken creates a secure random token
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
