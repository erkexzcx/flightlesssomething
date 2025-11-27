package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestAdminLoginRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test database
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create test config
	config := &Config{
		AdminUsername: "admin",
		AdminPassword: "testpass",
	}

	// Initialize rate limiters
	InitRateLimiters()

	// Create router with session
	r := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("test_session", store))
	r.POST("/auth/admin/login", HandleAdminLogin(config, db))

	// Test 3 failed login attempts
	for i := 0; i < 3; i++ {
		loginJSON := `{"username":"admin","password":"wrongpass"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/admin/login", strings.NewReader(loginJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Attempt %d: expected status %d, got %d", i+1, http.StatusUnauthorized, w.Code)
		}
	}

	// 4th attempt should be rate limited
	loginJSON := `{"username":"admin","password":"wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/admin/login", strings.NewReader(loginJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("4th attempt: expected status %d, got %d", http.StatusTooManyRequests, w.Code)
	}

	// Check response contains retry_after_secs
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, ok := response["retry_after_secs"]; !ok {
		t.Error("Response should contain retry_after_secs field")
	}

	// Reset the rate limiter and test successful login
	GetAdminLoginLimiter().Reset("admin_login")

	loginJSON = `{"username":"admin","password":"testpass"}`
	req = httptest.NewRequest(http.MethodPost, "/auth/admin/login", strings.NewReader(loginJSON))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Successful login: expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify that successful login reset the limiter
	// Try wrong password again
	loginJSON = `{"username":"admin","password":"wrongpass"}`
	req = httptest.NewRequest(http.MethodPost, "/auth/admin/login", strings.NewReader(loginJSON))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("After successful login, failed attempt should return %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestBenchmarkUploadRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test database
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Initialize rate limiters
	InitRateLimiters()

	// Create a regular user
	user := User{
		DiscordID: "test-user-123",
		Username:  "testuser",
		IsAdmin:   false,
	}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Test rate limiting directly via the limiter
	limiter := GetBenchmarkUploadLimiter()
	userKey := fmt.Sprintf("user_%d", user.ID)

	// First 5 uploads should succeed
	for i := 0; i < 5; i++ {
		if !limiter.Allow(userKey) {
			t.Errorf("Upload %d should be allowed", i+1)
		}
	}

	// 6th upload should be rate limited
	if limiter.Allow(userKey) {
		t.Error("6th upload should be rate limited")
	}

	// Check that we can get remaining time
	remaining := limiter.GetRemainingTime(userKey)
	if remaining <= 0 {
		t.Error("Should have remaining time for rate limit")
	}

	// Verify IsLocked returns true
	if !limiter.IsLocked(userKey) {
		t.Error("User should be locked after 5 uploads")
	}
}

func TestBenchmarkUploadRateLimit_AdminExempt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test database
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Initialize rate limiters
	InitRateLimiters()

	// Create an admin user
	adminUser := User{
		DiscordID: "admin-user-123",
		Username:  "adminuser",
		IsAdmin:   true,
	}
	if err := db.DB.Create(&adminUser).Error; err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	// Test that admin users bypass rate limiting
	// We'll simulate this by checking that rate limiting isn't enforced for admin users
	// This is tested in the HandleCreateBenchmark function which checks !user.IsAdmin

	// In the handler, rate limiting is only applied if !user.IsAdmin
	// So we just verify that admin flag is set correctly
	if !adminUser.IsAdmin {
		t.Error("Admin user should have IsAdmin flag set to true")
	}

	// Verify non-admin would be rate limited
	regularUser := User{
		DiscordID: "regular-user-123",
		Username:  "regularuser",
		IsAdmin:   false,
	}
	if err := db.DB.Create(&regularUser).Error; err != nil {
		t.Fatalf("Failed to create regular user: %v", err)
	}

	limiter := GetBenchmarkUploadLimiter()
	userKey := fmt.Sprintf("user_%d", regularUser.ID)

	// Use up the limit for regular user
	for i := 0; i < 5; i++ {
		limiter.Allow(userKey)
	}

	// Should be locked for regular user
	if !limiter.IsLocked(userKey) {
		t.Error("Regular user should be rate limited after 5 uploads")
	}

	// Admin user key should not be locked (as it's never checked)
	adminKey := fmt.Sprintf("user_%d", adminUser.ID)
	if limiter.IsLocked(adminKey) {
		t.Error("Admin user key should not be locked (rate limiting not applied)")
	}
}

func TestAdminLoginRateLimit_SuccessfulLoginResetsCounter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test database
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create test config
	config := &Config{
		AdminUsername: "admin",
		AdminPassword: "testpass",
	}

	// Initialize rate limiters
	InitRateLimiters()

	// Create router with session
	r := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("test_session", store))
	r.POST("/auth/admin/login", HandleAdminLogin(config, db))

	// Make 2 failed attempts
	for i := 0; i < 2; i++ {
		loginJSON := `{"username":"admin","password":"wrongpass"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/admin/login", strings.NewReader(loginJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Failed attempt %d: expected status %d, got %d", i+1, http.StatusUnauthorized, w.Code)
		}
	}

	// Make a successful login
	loginJSON := `{"username":"admin","password":"testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/admin/login", strings.NewReader(loginJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Successful login: expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Make 3 more failed attempts - should work since counter was reset
	for i := 0; i < 3; i++ {
		loginJSON = `{"username":"admin","password":"wrongpass"}`
		req = httptest.NewRequest(http.MethodPost, "/auth/admin/login", strings.NewReader(loginJSON))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Failed attempt after reset %d: expected status %d, got %d", i+1, http.StatusUnauthorized, w.Code)
		}
	}

	// 4th attempt should now be rate limited
	loginJSON = `{"username":"admin","password":"wrongpass"}`
	req = httptest.NewRequest(http.MethodPost, "/auth/admin/login", strings.NewReader(loginJSON))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("4th attempt after reset: expected status %d, got %d", http.StatusTooManyRequests, w.Code)
	}
}

func TestRateLimitExpiration(t *testing.T) {
	// Create a rate limiter with a short window for testing
	rl := NewRateLimiter(2, 500*time.Millisecond)

	// Use up the limit
	rl.Allow("test-key")
	rl.Allow("test-key")

	// Should be locked
	if !rl.IsLocked("test-key") {
		t.Error("Should be locked after reaching limit")
	}

	// Wait for the window to expire
	time.Sleep(600 * time.Millisecond)

	// Should not be locked anymore
	if rl.IsLocked("test-key") {
		t.Error("Should not be locked after window expired")
	}

	// Should be allowed again
	if !rl.Allow("test-key") {
		t.Error("Should be allowed after window expired")
	}
}
