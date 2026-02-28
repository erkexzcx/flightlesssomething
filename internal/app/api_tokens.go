package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const maxTokensPerUser = 10

// HandleListAPITokens lists all API tokens for the current user
func HandleListAPITokens(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("UserID")

		var tokens []APIToken
		if err := db.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&tokens).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve tokens"})
			return
		}

		c.JSON(http.StatusOK, tokens)
	}
}

// HandleCreateAPIToken creates a new API token for the current user
func HandleCreateAPIToken(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("UserID")

		var req struct {
			Name string `json:"name" binding:"required,min=1,max=100"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// Check if user has reached the token limit
		var count int64
		if err := db.DB.Model(&APIToken{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check token count"})
			return
		}

		if count >= maxTokensPerUser {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maximum number of tokens reached (10)"})
			return
		}

		// Generate a random token
		token, err := generateAPIToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}

		apiToken := APIToken{
			UserID: userID,
			Token:  token,
			Name:   req.Name,
		}

		if err := db.DB.Create(&apiToken).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
			return
		}

		c.JSON(http.StatusCreated, apiToken)
	}
}

// HandleDeleteAPIToken deletes an API token
func HandleDeleteAPIToken(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("UserID")
		tokenID := c.Param("id")

		// Verify the token belongs to the current user
		var token APIToken
		if err := db.DB.Where("id = ? AND user_id = ?", tokenID, userID).First(&token).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
			return
		}

		if err := db.DB.Delete(&token).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "token deleted"})
	}
}

// generateAPIToken generates a random API token
func generateAPIToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// RequireAuthOrToken is a middleware that requires either session authentication or API token
func RequireAuthOrToken(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First try session authentication
		session := sessions.Default(c)
		userID := session.Get("UserID")

		if userID != nil {
			// Validate user against database to ensure account still exists and is not banned.
			// This prevents stale sessions from granting access to deleted, banned, or demoted users.
			var user User
			if err := db.DB.First(&user, userID).Error; err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
				c.Abort()
				return
			}

			if user.IsBanned {
				c.JSON(http.StatusForbidden, gin.H{"error": "your account has been banned"})
				c.Abort()
				return
			}

			// Set context from database (not from potentially stale session values)
			c.Set("UserID", user.ID)
			c.Set("Username", user.Username)
			c.Set("IsAdmin", user.IsAdmin)
			c.Set("AuthMethod", "session") // Track authentication method

			// Update user's last web activity timestamp
			// Note: This adds a DB write on every web request. For high-traffic scenarios,
			// consider batching updates or using a background process.
			now := time.Now()
			db.DB.Model(&User{}).Where("id = ?", user.ID).Update("last_web_activity_at", now)

			c.Next()
			return
		}

		// Try API token authentication
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		// Expected format: "Bearer <token>"
		const prefix = "Bearer "
		if len(authHeader) <= len(prefix) || authHeader[:len(prefix)] != prefix {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			c.Abort()
			return
		}

		token := authHeader[len(prefix):]

		// Find the API token
		var apiToken APIToken
		if err := db.DB.Preload("User").Where("token = ?", token).First(&apiToken).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Check if user is banned
		if apiToken.User.IsBanned {
			c.JSON(http.StatusForbidden, gin.H{"error": "your account has been banned"})
			c.Abort()
			return
		}

		// Update last used timestamp for token and user
		// Note: This adds DB writes on every API request. For high-traffic scenarios,
		// consider batching updates or using a background process.
		now := time.Now()
		apiToken.LastUsedAt = &now
		if err := db.DB.Save(&apiToken).Error; err != nil {
			// Log error but don't fail authentication - the user is valid
			// This is a non-critical failure that only affects tracking
			if cErr := c.Error(err); cErr != nil {
				// Log but continue - error handling in Gin context is best-effort
				fmt.Printf("Warning: failed to set context error: %v\n", cErr)
			}
		}

		// Update user's last API activity timestamp
		db.DB.Model(&User{}).Where("id = ?", apiToken.UserID).Update("last_api_activity_at", now)

		// Set user context
		c.Set("UserID", apiToken.UserID)
		c.Set("Username", apiToken.User.Username)
		c.Set("IsAdmin", apiToken.User.IsAdmin)
		c.Set("AuthMethod", "api_token") // Track authentication method
		c.Next()
	}
}
