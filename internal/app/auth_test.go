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

func TestSessionCookieFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		secure     bool
		wantSecure bool
	}{
		{"secure cookie for HTTPS", true, true},
		{"non-secure cookie for HTTP", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			store := cookie.NewStore([]byte("test-secret"))
			store.Options(sessions.Options{
				Path:     "/",
				HttpOnly: true,
				Secure:   tt.secure,
				SameSite: http.SameSiteLaxMode,
			})
			r.Use(sessions.Sessions("test_session", store))

			r.GET("/test", func(ctx *gin.Context) {
				session := sessions.Default(ctx)
				session.Set("key", "value")
				if err := session.Save(); err != nil {
					t.Errorf("Failed to save session: %v", err)
				}
				ctx.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			r.ServeHTTP(w, req)

			cookies := w.Result().Cookies()
			if len(cookies) == 0 {
				t.Fatal("Expected session cookie to be set")
			}

			sessionCookie := cookies[0]
			if sessionCookie.HttpOnly != true {
				t.Error("Expected HttpOnly to be true")
			}
			if sessionCookie.Secure != tt.wantSecure {
				t.Errorf("Expected Secure=%v, got %v", tt.wantSecure, sessionCookie.Secure)
			}
			if sessionCookie.SameSite != http.SameSiteLaxMode {
				t.Errorf("Expected SameSite=Lax, got %v", sessionCookie.SameSite)
			}
		})
	}
}

func TestLogoutCookieFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	store := cookie.NewStore([]byte("test-secret"))
	store.Options(sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("test_session", store))
	r.POST("/auth/logout", HandleLogout(false))

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.HttpOnly != true {
			t.Error("Expected logout cookie HttpOnly to be true")
		}
		if c.SameSite != http.SameSiteLaxMode {
			t.Errorf("Expected logout cookie SameSite=Lax, got %v", c.SameSite)
		}
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
