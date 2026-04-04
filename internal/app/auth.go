package app

import (
	"crypto/rand"
	"crypto/subtle"
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

		// Check if Discord rejected the authorization (user denied, app disabled, etc.)
		if errParam := c.Query("error"); errParam != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "authorization denied"})
			return
		}

		savedState := session.Get("OAuthState")
		if savedState == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}
		stateStr, ok := savedState.(string)
		if !ok || subtle.ConstantTimeCompare([]byte(c.Query("state")), []byte(stateStr)) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}
		// Clear the one-time OAuth state nonce now that it has been validated.
		// Save the session immediately so the nonce is invalidated even if the
		// downstream Exchange call fails — preserving the one-time-use guarantee.
		session.Delete("OAuthState")
		if err := session.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
			return
		}

		token, err := discordOAuthConfig.Exchange(c.Request.Context(), c.Query("code"))
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
		client := discordOAuthConfig.Client(c.Request.Context(), token)
		res, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
			return
		}
		defer func() {
			if closeErr := res.Body.Close(); closeErr != nil {
				// Log error but continue - this is cleanup
				fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
			}
		}()
		if res.StatusCode != 200 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
			return
		}

		body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20)) // limit Discord response to 1 MB
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
		db.DB.Model(&User{}).Where("id = ?", user.ID).Update("last_web_activity_at", now)

		session.Clear()
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
		// Rate-limit admin login attempts per source IP to prevent a single attacker
		// from locking out all admins by exhausting a global slot.
		limiter := GetAdminLoginLimiter()
		clientIP := c.ClientIP()
		if allowed, remaining := limiter.AllowWithRemaining(clientIP); !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":            "too many failed login attempts",
				"retry_after_secs": int(remaining.Seconds()),
			})
			return
		}

		var loginReq struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&loginReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// Use constant-time comparison to prevent timing side-channel attacks.
		usernameOK := subtle.ConstantTimeCompare([]byte(loginReq.Username), []byte(config.AdminUsername))
		passwordOK := subtle.ConstantTimeCompare([]byte(loginReq.Password), []byte(config.AdminPassword))
		if usernameOK&passwordOK != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		// Successful login - reset the rate limiter for this IP
		limiter.Reset(clientIP)

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

		// Prevent banned admin from logging in before making any DB changes
		if adminUser.IsBanned {
			c.JSON(http.StatusForbidden, gin.H{"error": "your account has been banned"})
			return
		}

		// Ensure admin flag is set
		if !adminUser.IsAdmin {
			db.DB.Model(&User{}).Where("id = ?", adminUser.ID).Update("is_admin", true)
		}

		// Update last web activity
		now := time.Now()
		db.DB.Model(&User{}).Where("id = ?", adminUser.ID).Update("last_web_activity_at", now)

		session := sessions.Default(c)
		session.Clear()
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

// HandleGetCurrentUser returns the current user's information, validated against the database.
func HandleGetCurrentUser(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("UserID")

		if userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		// Re-validate against DB to catch banned/deleted/demoted users.
		var user User
		if err := db.DB.First(&user, userID).Error; err != nil || user.IsBanned {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id":  user.ID,
			"username": user.Username,
			"is_admin": user.IsAdmin,
		})
	}
}

// RequireAdmin is a middleware that requires admin privileges.
// Must be used after RequireAuthOrToken which sets "IsAdmin" in context from the database.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("IsAdmin")

		if !exists {
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
