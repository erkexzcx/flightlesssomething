package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

var discordOAuthConfig *oauth2.Config

// InitDiscordOAuth initializes the Discord OAuth2 configuration
func InitDiscordOAuth(clientID, clientSecret, redirectURL string) {
	discordOAuthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"identify"},
		Endpoint: oauth2.Endpoint{ //nolint:gosec // G101: These are public OAuth endpoint URLs, not credentials
			AuthURL:  "https://discord.com/api/oauth2/authorize",
			TokenURL: "https://discord.com/api/oauth2/token",
		},
	}
}

// DiscordUser represents a Discord user from the API
type DiscordUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// HandleLogin initiates Discord OAuth login
func HandleLogin(c *gin.Context) {
	session := sessions.Default(c)

	if session.Get("UserID") != nil {
		c.JSON(http.StatusOK, gin.H{"message": "already logged in"})
		return
	}

	state, err := generateRandomString()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	session.Set("OAuthState", state)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, discordOAuthConfig.AuthCodeURL(state))
}

// HandleLoginCallback handles the Discord OAuth callback
func HandleLoginCallback(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)

		savedState := session.Get("OAuthState")
		if savedState == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}
		stateStr, ok := savedState.(string)
		if !ok || c.Query("state") != stateStr {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}

		token, err := discordOAuthConfig.Exchange(context.Background(), c.Query("code"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange code"})
			return
		}

		// Create HTTP request with context
		req, err := http.NewRequestWithContext(c.Request.Context(), "GET", "https://discord.com/api/users/@me", http.NoBody)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
			return
		}
		client := discordOAuthConfig.Client(context.Background(), token)
		res, err := client.Do(req) //nolint:gosec // G704: URL is a hardcoded constant, not user-controlled
		if err != nil || res.StatusCode != 200 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
			return
		}
		defer func() {
			if closeErr := res.Body.Close(); closeErr != nil {
				// Log error but continue - this is cleanup
				fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
			}
		}()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
			return
		}

		var discordUser DiscordUser
		if err := json.Unmarshal(body, &discordUser); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse user info"})
			return
		}

		user := User{
			DiscordID: discordUser.ID,
			Username:  discordUser.Username,
		}

		result := db.DB.Model(&user).Where("discord_id = ?", discordUser.ID).Updates(&user)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		if result.RowsAffected == 0 {
			if err := db.DB.Create(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
				return
			}
		}

		if err := db.DB.Where("discord_id = ?", discordUser.ID).First(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
			return
		}

		// Check if user is banned
		if user.IsBanned {
			c.JSON(http.StatusForbidden, gin.H{"error": "your account has been banned"})
			return
		}

		// Update last web activity
		now := time.Now()
		user.LastWebActivityAt = &now
		db.DB.Save(&user)

		session.Set("UserID", user.ID)
		session.Set("Username", user.Username)
		session.Set("IsAdmin", user.IsAdmin)
		if err := session.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
			return
		}

		c.Redirect(http.StatusTemporaryRedirect, "/")
	}
}

// HandleAdminLogin handles admin login with username and password
func HandleAdminLogin(config *Config, db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if admin login is rate limited (global lock)
		limiter := GetAdminLoginLimiter()
		if limiter.IsLocked("admin_login") {
			remaining := limiter.GetRemainingTime("admin_login")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":            "too many failed login attempts",
				"retry_after_secs": int(remaining.Seconds()),
			})
			return
		}

		var loginReq struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"` //nolint:gosec // G117: Request binding field, not a hardcoded secret
		}

		if err := c.ShouldBindJSON(&loginReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		if loginReq.Username != config.AdminUsername || loginReq.Password != config.AdminPassword {
			// Record failed login attempt
			limiter.Allow("admin_login")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		// Successful login - reset the rate limiter
		limiter.Reset("admin_login")

		// Get or create admin user
		var adminUser User
		result := db.DB.Where("discord_id = ?", "admin").First(&adminUser)
		if result.Error != nil {
			// Create admin user
			adminUser = User{
				DiscordID: "admin",
				Username:  "Admin",
				IsAdmin:   true,
			}
			if err := db.DB.Create(&adminUser).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create admin user"})
				return
			}
		}

		// Ensure admin flag is set
		if !adminUser.IsAdmin {
			adminUser.IsAdmin = true
			db.DB.Save(&adminUser)
		}

		// Update last web activity
		now := time.Now()
		adminUser.LastWebActivityAt = &now
		db.DB.Save(&adminUser)

		session := sessions.Default(c)
		session.Set("UserID", adminUser.ID)
		session.Set("Username", adminUser.Username)
		session.Set("IsAdmin", true)
		if err := session.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "admin login successful"})
	}
}

// HandleLogout logs out the current user
func HandleLogout(secureCookie bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		session.Clear()
		// Set MaxAge to -1 to delete the cookie
		session.Options(sessions.Options{
			Path:     "/",
			MaxAge:   -1,
			Secure:   secureCookie,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		if err := session.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "logout successful"})
	}
}

// HandleGetCurrentUser returns the current user's session information
func HandleGetCurrentUser(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("UserID")

	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": session.Get("Username"),
		"is_admin": session.Get("IsAdmin"),
	})
}

// RequireAuth is a middleware that requires authentication
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("UserID")

		if userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		c.Set("UserID", userID)
		c.Set("Username", session.Get("Username"))
		c.Set("IsAdmin", session.Get("IsAdmin"))
		c.Next()
	}
}

// RequireAdmin is a middleware that requires admin privileges
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		isAdmin := session.Get("IsAdmin")

		if isAdmin == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin privileges required"})
			c.Abort()
			return
		}
		adminFlag, ok := isAdmin.(bool)
		if !ok || !adminFlag {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin privileges required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// generateRandomString generates a random hex string
func generateRandomString() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
