package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestGenerateRandomString(t *testing.T) {
	s1, err := generateRandomString()
	if err != nil {
		t.Fatalf("generateRandomString() error = %v", err)
	}

	if len(s1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("generateRandomString() length = %d, want 32", len(s1))
	}

	// Generate another and ensure they're different
	s2, err := generateRandomString()
	if err != nil {
		t.Fatalf("generateRandomString() error = %v", err)
	}

	if s1 == s2 {
		t.Error("generateRandomString() generated identical strings")
	}
}

func TestInitDiscordOAuth(t *testing.T) {
	InitDiscordOAuth("test-client-id", "test-secret", "http://localhost/callback")

	if discordOAuthConfig == nil {
		t.Error("InitDiscordOAuth() did not initialize config")
	}

	if discordOAuthConfig.ClientID != "test-client-id" {
		t.Errorf("ClientID = %s, want test-client-id", discordOAuthConfig.ClientID)
	}

	if len(discordOAuthConfig.Scopes) == 0 {
		t.Error("Scopes not set")
	}
}

func TestHandleGetCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("unauthenticated", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		// Setup sessions
		store := cookie.NewStore([]byte("test-secret"))
		r.Use(sessions.Sessions("test_session", store))
		r.GET("/api/auth/me", HandleGetCurrentUser)

		// Make request
		c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("authenticated", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		// Setup sessions
		store := cookie.NewStore([]byte("test-secret"))
		r.Use(sessions.Sessions("test_session", store))

		// Setup route and session data
		r.GET("/api/auth/me", func(ctx *gin.Context) {
			session := sessions.Default(ctx)
			session.Set("UserID", uint(123))
			session.Set("Username", "testuser")
			session.Set("IsAdmin", false)
			if err := session.Save(); err != nil {
				t.Errorf("Failed to save session: %v", err)
			}
			HandleGetCurrentUser(ctx)
		})

		// Make request
		c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}
