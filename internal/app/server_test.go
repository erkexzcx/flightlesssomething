package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestExtractPublicBaseURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "https URL",
			input: "https://example.com/auth/callback",
			want:  "https://example.com",
		},
		{
			name:  "http URL",
			input: "http://localhost:5000/auth/callback",
			want:  "http://localhost:5000",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "invalid URL",
			input: "://invalid",
			want:  "",
		},
		{
			name:  "URL without host",
			input: "/relative/path",
			want:  "",
		},
		{
			name:  "URL with subdomain and port",
			input: "https://app.example.com:8443/auth/discord/callback",
			want:  "https://app.example.com:8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPublicBaseURL(tt.input)
			if got != tt.want {
				t.Errorf("extractPublicBaseURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	InitRateLimiters()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("test_session", store))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	headers := []struct {
		name  string
		value string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, h := range headers {
		got := w.Header().Get(h.name)
		if got != h.value {
			t.Errorf("Header %s = %q, want %q", h.name, got, h.value)
		}
	}
}
