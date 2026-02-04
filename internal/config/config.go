package config

import (
	"fmt"
	"os"
)

// Config holds the application configuration
type Config struct {
	AdminPassword   string
	DBPath          string
	ListenAddr      string
	BaseHomeDir     string
	SSHPort         int
	ExternalSSHPort int // External SSH port (e.g., mapped in Coolify)
	SessionSecret   string
	SessionDuration int // hours
}

// Load reads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		AdminPassword:   getEnv("ADMIN_PASSWORD", "admin123"),
		DBPath:          getEnv("DB_PATH", "/devbase/.internal/users.db"),
		ListenAddr:      getEnv("LISTEN_ADDR", ":8080"),
		BaseHomeDir:     getEnv("BASE_HOME_DIR", "/devbase"),
		SSHPort:         getEnvInt("SSH_PORT", 2222),
		ExternalSSHPort: getEnvInt("EXTERNAL_SSH_PORT", 2222), // External mapped port (e.g., 9311 in Coolify)
		SessionSecret:   getEnv("SESSION_SECRET", "devbase-session-secret-change-me"),
		SessionDuration: getEnvInt("SESSION_DURATION", 1), // 1 hour
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}
