package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	adminSessionKey = "admin_session"
	usernameKey     = "admin_username"
	sessionDuration = 1 * time.Hour
)

// AdminAuth handles admin authentication using system
type AdminAuth struct{}

// New creates a new AdminAuth instance
func New() *AdminAuth {
	return &AdminAuth{}
}

// Session represents an admin session
type Session struct {
	Token     string
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// User represents a logged-in admin user
type User struct {
	Username string
}

// Login authenticates an admin user by attempting to run a command as that user
func (a *AdminAuth) Login(username, password string) (*Session, error) {
	// Validate username
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if password == "" {
		return nil, fmt.Errorf("password is required")
	}

	// Try to authenticate by running a simple command as the user using su
	// This validates the password against the system (PAM/shadow)
	cmd := exec.Command("su", "-c", "true", username)
	cmd.Stdin = strings.NewReader(password)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}

	// Verify command succeeded (exit code 0)
	if !cmd.ProcessState.Success() {
		return nil, fmt.Errorf("authentication failed: %s", string(output))
	}

	// Create session token
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session: %w", err)
	}

	session := &Session{
		Token:     token,
		Username:  username,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sessionDuration),
	}

	return session, nil
}

// ChangePassword is not supported
func (a *AdminAuth) ChangePassword(username, oldPassword, newPassword string) error {
	return fmt.Errorf("password changes must be done using the system's 'passwd' command")
}

// ValidateSession checks if a session token is valid
func (a *AdminAuth) ValidateSession(token string) bool {
	if token == "" {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(token)
	return err == nil
}

// IsAuthenticated checks if the current request is authenticated
func (a *AdminAuth) IsAuthenticated(c *gin.Context) bool {
	token, err := c.Cookie(adminSessionKey)
	if err != nil {
		return false
	}
	return a.ValidateSession(token)
}

// RequireAuth is middleware that requires authentication
func (a *AdminAuth) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !a.IsAuthenticated(c) {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

// SetSessionCookie sets the session cookie
func SetSessionCookie(c *gin.Context, session *Session) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		adminSessionKey,
		session.Token,
		int(sessionDuration.Seconds()),
		"/",
		"",
		false,
		true,
	)
	c.SetCookie(
		usernameKey,
		session.Username,
		int(sessionDuration.Seconds()),
		"/",
		"",
		false,
		false,
	)
}

// ClearSessionCookie clears the session cookie
func ClearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		adminSessionKey,
		"",
		-1,
		"/",
		"",
		false,
		true,
	)
	c.SetCookie(
		usernameKey,
		"",
		-1,
		"/",
		"",
		false,
		false,
	)
}

// GetUsername returns the username of the currently authenticated user
func GetUsername(c *gin.Context) string {
	username, _ := c.Cookie(usernameKey)
	return username
}

// generateToken generates a random session token
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
