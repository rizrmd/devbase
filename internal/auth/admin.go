package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const (
	adminSessionKey = "admin_session"
	sessionDuration = 1 * time.Hour
	shadowFile      = "/etc/shadow"
)

// AdminAuth handles admin authentication using shadow file
type AdminAuth struct{}

// New creates a new AdminAuth instance
func New() *AdminAuth {
	return &AdminAuth{}
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

// Login authenticates an admin user using /etc/shadow
func (a *AdminAuth) Login(username, password string) (*Session, error) {
	// Read shadow file
	data, err := os.ReadFile(shadowFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read shadow file: %w", err)
	}

	// Find user's hash
	lines := strings.Split(string(data), "\n")
	var storedHash string
	for _, line := range lines {
		if strings.HasPrefix(line, username+":") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				storedHash = parts[1]
				break
			}
		}
	}

	if storedHash == "" {
		return nil, fmt.Errorf("user not found")
	}

	// Check if account is locked
	if storedHash == "*" || storedHash == "!" || strings.HasPrefix(storedHash, "!") {
		return nil, fmt.Errorf("account is locked")
	}

	// Verify password - only support bcrypt for now
	if !strings.HasPrefix(storedHash, "$2") {
		return nil, fmt.Errorf("unsupported hash format. Please change password to bcrypt: sudo passwd dev")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("authentication failed")
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
}

// generateToken generates a random session token
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
