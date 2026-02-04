package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

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

	// Ensure base home directory exists
	if err := os.MkdirAll(cfg.BaseHomeDir, 0755); err != nil {
		log.Fatalf("Failed to create base home directory: %v", err)
	}

	// Initialize managers
	authManager := auth.New()
	userManager := user.New(cfg.BaseHomeDir)

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
			"Title":            "Login - DevBase User Manager",
			"Error":            c.Query("error"),
			"IsAuthenticated":  false,
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
					"Title":           "Users - DevBase User Manager",
					"Error":           fmt.Sprintf("Failed to load users: %v", err),
					"IsAuthenticated": true,
					"Username":        auth.GetUsername(c),
				})
				return
			}

			c.HTML(http.StatusOK, "users.html", gin.H{
				"Title":           "Users - DevBase User Manager",
				"Users":           users,
				"Success":         c.Query("success"),
				"IsAuthenticated": true,
				"Username":        auth.GetUsername(c),
			})
		})

		protected.GET("/users/new", func(c *gin.Context) {
			c.HTML(http.StatusOK, "user_form.html", gin.H{
				"Title":           "Create User - DevBase User Manager",
				"Action":          "/users",
				"Method":          "POST",
				"IsAuthenticated": true,
				"Username":        auth.GetUsername(c),
			})
		})

		protected.POST("/users", func(c *gin.Context) {
			var newUser struct {
				Username string `form:"username" binding:"required"`
				Password string `form:"password" binding:"required"`
			}

			if err := c.ShouldBind(&newUser); err != nil {
				c.HTML(http.StatusBadRequest, "user_form.html", gin.H{
					"Title":           "Create User - DevBase User Manager",
					"Action":          "/users",
					"Method":          "POST",
					"Error":           "Please fill in all required fields",
					"IsAuthenticated": true,
					"Username":        auth.GetUsername(c),
				})
				return
			}

			if err := userManager.CreateUser(newUser.Username, newUser.Password); err != nil {
				c.HTML(http.StatusInternalServerError, "user_form.html", gin.H{
					"Title":           "Create User - DevBase User Manager",
					"Action":          "/users",
					"Method":          "POST",
					"Error":           fmt.Sprintf("Failed to create user: %v", err),
					"IsAuthenticated": true,
					"Username":        auth.GetUsername(c),
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

			// Get the external host from the request (remove port if present)
			host := c.GetHeader("X-Forwarded-Host")
			if host == "" {
				host = c.Request.Host
			}
			// Remove port from host if present
			if idx := strings.Index(host, ":"); idx != -1 {
				host = host[:idx]
			}

			c.HTML(http.StatusOK, "user_form.html", gin.H{
				"Title":           fmt.Sprintf("Edit User %s", username),
				"User":            u,
				"Action":          fmt.Sprintf("/users/%s", username),
				"Method":          "POST",
				"IsEdit":          true,
				"Success":         c.Query("success"),
				"Error":           c.Query("error"),
				"IsAuthenticated": true,
				"Username":        auth.GetUsername(c),
				"SSHPort":         cfg.ExternalSSHPort,
				"SSHHost":         host,
			})
		})

		protected.POST("/users/:username", func(c *gin.Context) {
			username := c.Param("username")

			// Update is a no-op since we removed email
			if err := userManager.UpdateUser(username, ""); err != nil {
				c.Redirect(http.StatusFound, fmt.Sprintf("/users/%s?error=Failed+to+update+user", username))
				return
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

		protected.POST("/users/:username/password", func(c *gin.Context) {
			username := c.Param("username")
			var passwordData struct {
				Password string `form:"password" binding:"required"`
			}

			if err := c.ShouldBind(&passwordData); err != nil {
				c.Redirect(http.StatusFound, fmt.Sprintf("/users/%s?error=Password+is+required", username))
				return
			}

			if err := userManager.ChangePassword(username, passwordData.Password); err != nil {
				c.Redirect(http.StatusFound, fmt.Sprintf("/users/%s?error=Failed+to+change+password", username))
				return
			}

			c.Redirect(http.StatusFound, fmt.Sprintf("/users/%s?success=Password+changed", username))
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
			}

			if err := c.ShouldBindJSON(&newUser); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			if err := userManager.CreateUser(newUser.Username, newUser.Password); err != nil {
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
