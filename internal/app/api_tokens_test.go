package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Setup sessions
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("test_session", store))

	return r
}

func TestAPITokenOperations(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "token_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(tmpDir); removeErr != nil {
			t.Logf("Warning: failed to remove temp directory: %v", removeErr)
		}
	}()

	// Initialize test database
	db, err := InitDB(tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Create a test user
	user := User{
		DiscordID: "test123",
		Username:  "testuser",
		IsAdmin:   false,
	}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	r := setupTestRouter()

	// Setup routes
	authorized := r.Group("/api")
	authorized.Use(func(c *gin.Context) {
		// Mock authenticated user
		c.Set("UserID", user.ID)
		c.Set("Username", user.Username)
		c.Set("IsAdmin", user.IsAdmin)
		c.Next()
	})
	{
		authorized.GET("/tokens", HandleListAPITokens(db))
		authorized.POST("/tokens", HandleCreateAPIToken(db))
		authorized.DELETE("/tokens/:id", HandleDeleteAPIToken(db))
	}

	t.Run("create_token", func(t *testing.T) {
		reqBody := map[string]string{"name": "Test Token"}
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/tokens", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response APIToken
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response.Name != "Test Token" {
			t.Errorf("Expected name 'Test Token', got '%s'", response.Name)
		}

		if len(response.Token) != 64 {
			t.Errorf("Expected token length 64, got %d", len(response.Token))
		}
	})

	t.Run("list_tokens", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/tokens", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []APIToken
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(response) == 0 {
			t.Error("Expected at least one token")
		}
	})

	t.Run("token_limit", func(t *testing.T) {
		// Create 9 more tokens to reach the limit of 10
		for i := 0; i < 9; i++ {
			token := APIToken{
				UserID: user.ID,
				Token:  generateTestToken(),
				Name:   "Token " + strconv.Itoa(i),
			}
			db.DB.Create(&token)
		}

		// Try to create the 11th token
		reqBody := map[string]string{"name": "Over Limit Token"}
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/tokens", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for exceeding limit, got %d", w.Code)
		}
	})

	t.Run("delete_token", func(t *testing.T) {
		// Get first token
		var token APIToken
		db.DB.Where("user_id = ?", user.ID).First(&token)

		req := httptest.NewRequest(http.MethodDelete, "/api/tokens/"+strconv.Itoa(int(token.ID)), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify token was deleted
		var count int64
		db.DB.Model(&APIToken{}).Where("id = ?", token.ID).Count(&count)
		if count != 0 {
			t.Error("Token should have been deleted")
		}
	})
}

func TestAPITokenAuthentication(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "token_auth_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(tmpDir); removeErr != nil {
			t.Logf("Warning: failed to remove temp directory: %v", removeErr)
		}
	}()

	// Initialize test database
	db, err := InitDB(tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Create benchmark directory
	benchmarksDir := filepath.Join(tmpDir, "benchmarks")
	if err := os.MkdirAll(benchmarksDir, 0755); err != nil {
		t.Fatalf("Failed to create benchmarks directory: %v", err)
	}

	// Create a test user and token
	user := User{
		DiscordID: "test456",
		Username:  "tokenuser",
		IsAdmin:   false,
	}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	token := APIToken{
		UserID: user.ID,
		Token:  "test_token_12345678901234567890123456789012",
		Name:   "Test Auth Token",
	}
	if err := db.DB.Create(&token).Error; err != nil {
		t.Fatalf("Failed to create test token: %v", err)
	}

	r := setupTestRouter()

	// Setup protected route
	authorized := r.Group("/api")
	authorized.Use(RequireAuthOrToken(db))
	{
		authorized.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"user_id":  c.GetUint("UserID"),
				"username": c.GetString("Username"),
			})
		})
	}

	t.Run("auth_with_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("Authorization", "Bearer test_token_12345678901234567890123456789012")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["username"] != user.Username {
			t.Errorf("Expected username '%s', got '%v'", user.Username, response["username"])
		}
	})

	t.Run("auth_without_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("auth_with_invalid_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("Authorization", "Bearer invalid_token")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("last_used_at_updated", func(t *testing.T) {
		// Get initial last_used_at
		var tokenBefore APIToken
		db.DB.Where("id = ?", token.ID).First(&tokenBefore)

		// Make a request with the token
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("Authorization", "Bearer test_token_12345678901234567890123456789012")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		// Get updated last_used_at
		var tokenAfter APIToken
		db.DB.Where("id = ?", token.ID).First(&tokenAfter)

		if tokenAfter.LastUsedAt == nil {
			t.Error("LastUsedAt should be set after use")
		}

		if tokenBefore.LastUsedAt != nil && !tokenAfter.LastUsedAt.After(*tokenBefore.LastUsedAt) {
			t.Error("LastUsedAt should be updated after use")
		}
	})
}

func generateTestToken() string {
	token, err := generateAPIToken()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate test token: %v", err))
	}
	return token
}
