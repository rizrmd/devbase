package main

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"

	"github.com/rizdev/devbase/internal/auth"
	"github.com/rizdev/devbase/internal/config"
	"github.com/rizdev/devbase/internal/user"
)

//go:embed templates/*
var templatesFS embed.FS

func init() {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize database
	db, err := initDatabase(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize managers
	authManager := auth.New()
	userManager := user.New(db, cfg.BaseHomeDir)
	sshManager := user.NewSSHManager(db, cfg.BaseHomeDir)

	// Ensure base home directory exists
	if err := os.MkdirAll(cfg.BaseHomeDir, 0755); err != nil {
		log.Fatalf("Failed to create base home directory: %v", err)
	}

	// Setup Gin router
	router := gin.Default()

	// Load HTML templates from embedded filesystem
	tmpl := template.Must(template.New("").ParseFS(templatesFS, "templates/*.html"))
	router.SetHTMLTemplate(tmpl)

	// Public routes
	router.GET("/", func(c *gin.Context) {
		if authManager.IsAuthenticated(c) {
			c.Redirect(http.StatusFound, "/users")
		} else {
			c.Redirect(http.StatusFound, "/login")
		}
	})

	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"Title":         "Login - DevBase User Manager",
			"Error":         c.Query("error"),
			"IsAuthenticated": false,
		})
	})

	router.POST("/login", func(c *gin.Context) {
		var credentials struct {
			Username string `form:"username"`
			Password string `form:"password"`
		}
		if err := c.ShouldBind(&credentials); err != nil {
			c.Redirect(http.StatusFound, "/login?error=Invalid+form+data")
			return
		}

		session, err := authManager.Login(credentials.Username, credentials.Password)
		if err != nil {
			c.Redirect(http.StatusFound, "/login?error=Invalid+username+or+password")
			return
		}

		auth.SetSessionCookie(c, session)
		c.Redirect(http.StatusFound, "/users")
	})

	router.GET("/logout", func(c *gin.Context) {
		auth.ClearSessionCookie(c)
		c.Redirect(http.StatusFound, "/login")
	})

	// Protected routes
	protected := router.Group("")
	protected.Use(authManager.RequireAuth())
	{
		protected.GET("/users", func(c *gin.Context) {
			users, err := userManager.ListUsers()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "users.html", gin.H{
					"Title":         "Users - DevBase User Manager",
					"Error":         fmt.Sprintf("Failed to load users: %v", err),
					"IsAuthenticated": true,
				})
				return
			}

			c.HTML(http.StatusOK, "users.html", gin.H{
				"Title":         "Users - DevBase User Manager",
				"Users":         users,
				"Success":       c.Query("success"),
				"IsAuthenticated": true,
			})
		})

		protected.GET("/users/new", func(c *gin.Context) {
			c.HTML(http.StatusOK, "user_form.html", gin.H{
				"Title":         "Create User - DevBase User Manager",
				"Action":        "/users",
				"Method":        "POST",
				"IsAuthenticated": true,
			})
		})

		protected.POST("/users", func(c *gin.Context) {
			var newUser struct {
				Username string `form:"username" binding:"required"`
				Password string `form:"password" binding:"required"`
				Email    string `form:"email"`
			}

			if err := c.ShouldBind(&newUser); err != nil {
				c.HTML(http.StatusBadRequest, "user_form.html", gin.H{
					"Title":         "Create User - DevBase User Manager",
					"Action":        "/users",
					"Method":        "POST",
					"Error":         "Please fill in all required fields",
					"IsAuthenticated": true,
				})
				return
			}

			if err := userManager.CreateUser(newUser.Username, newUser.Password, newUser.Email); err != nil {
				c.HTML(http.StatusInternalServerError, "user_form.html", gin.H{
					"Title":         "Create User - DevBase User Manager",
					"Action":        "/users",
					"Method":        "POST",
					"Error":         fmt.Sprintf("Failed to create user: %v", err),
					"IsAuthenticated": true,
				})
				return
			}

			c.Redirect(http.StatusFound, "/users?success=User+created+successfully")
		})

		protected.GET("/users/:username", func(c *gin.Context) {
			username := c.Param("username")
			u, err := userManager.GetUser(username)
			if err != nil {
				c.Redirect(http.StatusFound, "/users")
				return
			}

			keys, err := sshManager.ListKeys(username)
			if err != nil {
				keys = []user.SSHKey{}
			}

			c.HTML(http.StatusOK, "user_form.html", gin.H{
				"Title":         fmt.Sprintf("Edit User %s", username),
				"User":          u,
				"Keys":          keys,
				"Action":        fmt.Sprintf("/users/%s", username),
				"Method":        "POST",
				"IsEdit":        true,
				"Success":       c.Query("success"),
				"Error":         c.Query("error"),
				"IsAuthenticated": true,
			})
		})

		protected.POST("/users/:username", func(c *gin.Context) {
			username := c.Param("username")
			var updateData struct {
				Email string `form:"email"`
			}

			if err := c.ShouldBind(&updateData); err == nil {
				if err := userManager.UpdateUser(username, updateData.Email); err != nil {
					c.Redirect(http.StatusFound, fmt.Sprintf("/users/%s?error=Failed+to+update+user", username))
					return
				}
			}

			c.Redirect(http.StatusFound, fmt.Sprintf("/users/%s?success=User+updated", username))
		})

		protected.DELETE("/users/:username", func(c *gin.Context) {
			username := c.Param("username")
			if err := userManager.DeleteUser(username); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		protected.POST("/users/:username/keys", func(c *gin.Context) {
			username := c.Param("username")
			var newKey struct {
				PublicKey string `form:"public_key" binding:"required"`
				Name      string `form:"name"`
			}

			if err := c.ShouldBind(&newKey); err != nil {
				c.Redirect(http.StatusFound, fmt.Sprintf("/users/%s?error=Invalid+key+data", username))
				return
			}

			if err := sshManager.AddKey(username, newKey.PublicKey, newKey.Name); err != nil {
				c.Redirect(http.StatusFound, fmt.Sprintf("/users/%s?error=Failed+to+add+key", username))
				return
			}

			c.Redirect(http.StatusFound, fmt.Sprintf("/users/%s?success=SSH+key+added", username))
		})

		protected.DELETE("/users/:username/keys/:id", func(c *gin.Context) {
			username := c.Param("username")
			keyID := 0
			fmt.Sscanf(c.Param("id"), "%d", &keyID)

			if err := sshManager.DeleteKey(keyID, username); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"success": true})
		})
	}

	// API routes
	api := router.Group("/api")
	api.Use(authManager.RequireAuth())
	{
		api.GET("/users", func(c *gin.Context) {
			users, err := userManager.ListUsers()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, users)
		})

		api.POST("/users", func(c *gin.Context) {
			var newUser struct {
				Username string `json:"username" binding:"required"`
				Password string `json:"password" binding:"required"`
				Email    string `json:"email"`
			}

			if err := c.ShouldBindJSON(&newUser); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			if err := userManager.CreateUser(newUser.Username, newUser.Password, newUser.Email); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusCreated, gin.H{"success": true, "username": newUser.Username})
		})

		api.GET("/users/:username", func(c *gin.Context) {
			u, err := userManager.GetUser(c.Param("username"))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}
			c.JSON(http.StatusOK, u)
		})

		api.DELETE("/users/:username", func(c *gin.Context) {
			if err := userManager.DeleteUser(c.Param("username")); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"success": true})
		})
	}

	// Start server
	addr := cfg.ListenAddr
	log.Printf("Starting DevBase User Manager on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// initDatabase initializes the SQLite database
func initDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Read and execute schema
	schemaPath := filepath.Join(filepath.Dir(filepath.Dir(dbPath)), "data", "schema.sql")
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		// If running from compiled binary, schema is embedded
		// For now, just create tables inline
		if err := createTables(db); err != nil {
			return nil, fmt.Errorf("failed to create tables: %w", err)
		}
	} else {
		schema, err := os.ReadFile(schemaPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read schema: %w", err)
		}
		if _, err := db.Exec(string(schema)); err != nil {
			return nil, fmt.Errorf("failed to execute schema: %w", err)
		}
	}

	return db, nil
}

// createTables creates database tables inline
func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_active BOOLEAN DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS passwords (
		username TEXT PRIMARY KEY,
		password_hash TEXT NOT NULL,
		changed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS ssh_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL,
		public_key TEXT NOT NULL,
		name TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
	);
	`

	_, err := db.Exec(schema)
	return err
}
