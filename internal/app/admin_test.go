package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestEnsureSystemAdmin(t *testing.T) {
	// Create a temporary database
	tmpDir := t.TempDir()
	db, err := InitDB(tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	t.Run("creates new system admin", func(t *testing.T) {
		err := EnsureSystemAdmin(db, "testadmin", "testpass")
		if err != nil {
			t.Fatalf("Failed to ensure system admin: %v", err)
		}

		// Verify the admin was created
		var admin User
		if err := db.DB.Where("discord_id = ?", "admin").First(&admin).Error; err != nil {
			t.Fatalf("Failed to find admin user: %v", err)
		}

		if admin.Username != "testadmin" {
			t.Errorf("Expected username 'testadmin', got '%s'", admin.Username)
		}

		if !admin.IsAdmin {
			t.Error("Expected admin flag to be true")
		}
	})

	t.Run("updates existing system admin username", func(t *testing.T) {
		err := EnsureSystemAdmin(db, "newadminname", "testpass")
		if err != nil {
			t.Fatalf("Failed to update system admin: %v", err)
		}

		// Verify the admin was updated
		var admin User
		if err := db.DB.Where("discord_id = ?", "admin").First(&admin).Error; err != nil {
			t.Fatalf("Failed to find admin user: %v", err)
		}

		if admin.Username != "newadminname" {
			t.Errorf("Expected username 'newadminname', got '%s'", admin.Username)
		}

		// Verify there's only one system admin
		var count int64
		db.DB.Model(&User{}).Where("discord_id = ?", "admin").Count(&count)
		if count != 1 {
			t.Errorf("Expected exactly 1 system admin, got %d", count)
		}
	})

	t.Run("ensures admin flag is set", func(t *testing.T) {
		// Manually unset the admin flag
		var admin User
		db.DB.Where("discord_id = ?", "admin").First(&admin)
		admin.IsAdmin = false
		db.DB.Save(&admin)

		// Run EnsureSystemAdmin again
		err := EnsureSystemAdmin(db, "adminuser", "testpass")
		if err != nil {
			t.Fatalf("Failed to ensure system admin: %v", err)
		}

		// Verify the admin flag was restored
		db.DB.Where("discord_id = ?", "admin").First(&admin)
		if !admin.IsAdmin {
			t.Error("Expected admin flag to be restored to true")
		}
	})
}

func TestAuditLogCreation(t *testing.T) {
	// Create a temporary database
	tmpDir := t.TempDir()
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

	t.Run("creates audit log entry", func(t *testing.T) {
		err := CreateAuditLog(db, user.ID, "Test Action", "Test description", "user", 1)
		if err != nil {
			t.Fatalf("Failed to create audit log: %v", err)
		}

		// Verify the log was created
		var log AuditLog
		if err := db.DB.Where("user_id = ?", user.ID).First(&log).Error; err != nil {
			t.Fatalf("Failed to find audit log: %v", err)
		}

		if log.Action != "Test Action" {
			t.Errorf("Expected action 'Test Action', got '%s'", log.Action)
		}

		if log.Description != "Test description" {
			t.Errorf("Expected description 'Test description', got '%s'", log.Description)
		}
	})

	t.Run("logs benchmark creation", func(t *testing.T) {
		// Create a test benchmark
		benchmark := Benchmark{
			UserID:      user.ID,
			Title:       "Test Benchmark",
			Description: "Test description",
		}
		if err := db.DB.Create(&benchmark).Error; err != nil {
			t.Fatalf("Failed to create test benchmark: %v", err)
		}

		LogBenchmarkCreated(db, user.ID, benchmark.ID, benchmark.Title)

		// Verify the log was created
		var log AuditLog
		if err := db.DB.Where("user_id = ? AND target_type = ? AND target_id = ?", user.ID, "benchmark", benchmark.ID).First(&log).Error; err != nil {
			t.Fatalf("Failed to find audit log: %v", err)
		}

		if log.Action != "Benchmark Created" {
			t.Errorf("Expected action 'Benchmark Created', got '%s'", log.Action)
		}
	})
}

func TestAuditLogModel(t *testing.T) {
	// Create a temporary database
	tmpDir := t.TempDir()
	db, err := InitDB(tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Create a test user
	user := User{
		DiscordID: "test456",
		Username:  "testuser2",
		IsAdmin:   false,
	}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create an audit log
	log := AuditLog{
		UserID:      user.ID,
		Action:      "Test Action",
		Description: "Test description",
		TargetType:  "user",
		TargetID:    user.ID,
	}
	if err := db.DB.Create(&log).Error; err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	// Retrieve the log with user preloaded
	var retrievedLog AuditLog
	if err := db.DB.Preload("User").First(&retrievedLog, log.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve audit log: %v", err)
	}

	if retrievedLog.User.Username != user.Username {
		t.Errorf("Expected user username '%s', got '%s'", user.Username, retrievedLog.User.Username)
	}

	if retrievedLog.CreatedAtHumanized == "" {
		t.Error("Expected CreatedAtHumanized to be set by AfterFind hook")
	}
}

func TestDatabaseMigration(t *testing.T) {
	// Create a temporary database
	tmpDir := t.TempDir()
	db, err := InitDB(tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Verify AuditLog table exists by creating a record
	log := AuditLog{
		UserID:      1,
		Action:      "Test",
		Description: "Test",
		TargetType:  "test",
		TargetID:    1,
	}
	if err := db.DB.Create(&log).Error; err != nil {
		t.Fatalf("Failed to create audit log - table may not exist: %v", err)
	}

	// Clean up
	db.DB.Delete(&log)
}

func TestSelfProtectionDeleteUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("prevents admin from deleting their own account", func(t *testing.T) {
		tmpDir := t.TempDir()
		db, err := InitDB(tmpDir)
		if err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}

		// Create admin user
		admin := User{
			DiscordID: "admin123",
			Username:  "admin",
			IsAdmin:   true,
		}
		if err := db.DB.Create(&admin).Error; err != nil {
			t.Fatalf("Failed to create admin user: %v", err)
		}

		// Setup test request
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		// Setup route with middleware that sets UserID
		r.DELETE("/api/admin/users/:id", func(ctx *gin.Context) {
			ctx.Set("UserID", admin.ID)
			HandleDeleteUser(db)(ctx)
		})

		// Make request to delete self
		c.Request = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/users/%d", admin.ID), nil)
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		// Parse response
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["error"] != "cannot delete your own account" {
			t.Errorf("Expected error message about self-deletion, got: %v", response["error"])
		}

		// Verify admin still exists
		var checkAdmin User
		if err := db.DB.First(&checkAdmin, admin.ID).Error; err != nil {
			t.Error("Admin user was deleted when it should have been prevented")
		}
	})

	t.Run("allows admin to delete other users", func(t *testing.T) {
		tmpDir := t.TempDir()
		db, err := InitDB(tmpDir)
		if err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}

		// Create admin user
		admin := User{
			DiscordID: "admin123",
			Username:  "admin",
			IsAdmin:   true,
		}
		if createErr := db.DB.Create(&admin).Error; createErr != nil {
			t.Fatalf("Failed to create admin user: %v", createErr)
		}

		// Create another user to delete
		otherUser := User{
			DiscordID: "user456",
			Username:  "regularuser",
			IsAdmin:   false,
		}
		if createErr := db.DB.Create(&otherUser).Error; createErr != nil {
			t.Fatalf("Failed to create test user: %v", createErr)
		}

		// Setup test request
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		// Setup route with middleware that sets UserID
		r.DELETE("/api/admin/users/:id", func(ctx *gin.Context) {
			ctx.Set("UserID", admin.ID)
			HandleDeleteUser(db)(ctx)
		})

		// Delete the other user (should succeed)
		c.Request = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/users/%d", otherUser.ID), nil)
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify user was deleted
		var deletedUser User
		err = db.DB.First(&deletedUser, otherUser.ID).Error
		if err == nil {
			t.Error("User should have been deleted")
		}
	})
}

func TestSelfProtectionBanUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("prevents admin from banning themselves", func(t *testing.T) {
		tmpDir := t.TempDir()
		db, err := InitDB(tmpDir)
		if err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}

		// Create admin user
		admin := User{
			DiscordID: "admin456",
			Username:  "admin",
			IsAdmin:   true,
		}
		if createErr := db.DB.Create(&admin).Error; createErr != nil {
			t.Fatalf("Failed to create admin user: %v", createErr)
		}

		// Setup test request
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		// Setup route with middleware that sets UserID
		r.PUT("/api/admin/users/:id/ban", func(ctx *gin.Context) {
			ctx.Set("UserID", admin.ID)
			HandleBanUser(db)(ctx)
		})

		// Try to ban self
		requestBody, err := json.Marshal(map[string]interface{}{
			"banned": true,
		})
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		c.Request = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/users/%d/ban", admin.ID), bytes.NewBuffer(requestBody))
		c.Request.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		// Parse response
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["error"] != "cannot ban your own account" {
			t.Errorf("Expected error message about self-ban, got: %v", response["error"])
		}

		// Verify admin is not banned
		var checkAdmin User
		if err := db.DB.First(&checkAdmin, admin.ID).Error; err != nil {
			t.Fatal("Failed to retrieve admin user")
		}
		if checkAdmin.IsBanned {
			t.Error("Admin should not be banned")
		}
	})

	t.Run("allows admin to unban themselves (if somehow banned)", func(t *testing.T) {
		tmpDir := t.TempDir()
		db, err := InitDB(tmpDir)
		if err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}

		// Create admin user
		admin := User{
			DiscordID: "admin789",
			Username:  "admin",
			IsAdmin:   true,
			IsBanned:  true, // Pre-banned
		}
		if createErr := db.DB.Create(&admin).Error; createErr != nil {
			t.Fatalf("Failed to create admin user: %v", createErr)
		}

		// Setup test request
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		// Setup route with middleware that sets UserID
		r.PUT("/api/admin/users/:id/ban", func(ctx *gin.Context) {
			ctx.Set("UserID", admin.ID)
			HandleBanUser(db)(ctx)
		})

		// Try to unban self (should be allowed)
		requestBody, err := json.Marshal(map[string]interface{}{
			"banned": false,
		})
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		c.Request = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/users/%d/ban", admin.ID), bytes.NewBuffer(requestBody))
		c.Request.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify admin is unbanned
		var checkAdmin User
		db.DB.First(&checkAdmin, admin.ID)
		if checkAdmin.IsBanned {
			t.Error("Admin should be unbanned")
		}
	})

	t.Run("allows admin to ban other users", func(t *testing.T) {
		tmpDir := t.TempDir()
		db, err := InitDB(tmpDir)
		if err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}

		// Create admin user
		admin := User{
			DiscordID: "admin012",
			Username:  "admin",
			IsAdmin:   true,
		}
		if createErr := db.DB.Create(&admin).Error; createErr != nil {
			t.Fatalf("Failed to create admin user: %v", createErr)
		}

		// Create another user to ban
		otherUser := User{
			DiscordID: "user789",
			Username:  "regularuser",
			IsAdmin:   false,
		}
		if createErr := db.DB.Create(&otherUser).Error; createErr != nil {
			t.Fatalf("Failed to create test user: %v", createErr)
		}

		// Setup test request
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		// Setup route with middleware that sets UserID
		r.PUT("/api/admin/users/:id/ban", func(ctx *gin.Context) {
			ctx.Set("UserID", admin.ID)
			HandleBanUser(db)(ctx)
		})

		// Ban the other user (should succeed)
		requestBody, err := json.Marshal(map[string]interface{}{
			"banned": true,
		})
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		c.Request = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/users/%d/ban", otherUser.ID), bytes.NewBuffer(requestBody))
		c.Request.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify user was banned
		var bannedUser User
		db.DB.First(&bannedUser, otherUser.ID)
		if !bannedUser.IsBanned {
			t.Error("User should be banned")
		}
	})
}

func TestSelfProtectionToggleAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("prevents admin from revoking their own admin privileges", func(t *testing.T) {
		tmpDir := t.TempDir()
		db, err := InitDB(tmpDir)
		if err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}

		// Create admin user
		admin := User{
			DiscordID: "admin345",
			Username:  "admin",
			IsAdmin:   true,
		}
		if createErr := db.DB.Create(&admin).Error; createErr != nil {
			t.Fatalf("Failed to create admin user: %v", createErr)
		}

		// Setup test request
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		// Setup route with middleware that sets UserID
		r.PUT("/api/admin/users/:id/admin", func(ctx *gin.Context) {
			ctx.Set("UserID", admin.ID)
			HandleToggleUserAdmin(db)(ctx)
		})

		// Try to revoke own admin privileges
		requestBody, err := json.Marshal(map[string]interface{}{
			"is_admin": false,
		})
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		c.Request = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/users/%d/admin", admin.ID), bytes.NewBuffer(requestBody))
		c.Request.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		// Parse response
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["error"] != "cannot revoke your own admin privileges" {
			t.Errorf("Expected error message about self-demotion, got: %v", response["error"])
		}

		// Verify admin flag is still true
		var checkAdmin User
		if err := db.DB.First(&checkAdmin, admin.ID).Error; err != nil {
			t.Fatal("Failed to retrieve admin user")
		}
		if !checkAdmin.IsAdmin {
			t.Error("Admin flag should still be true")
		}
	})

	t.Run("allows admin to keep their admin status (no-op)", func(t *testing.T) {
		tmpDir := t.TempDir()
		db, err := InitDB(tmpDir)
		if err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}

		// Create admin user
		admin := User{
			DiscordID: "admin678",
			Username:  "admin",
			IsAdmin:   true,
		}
		if createErr := db.DB.Create(&admin).Error; createErr != nil {
			t.Fatalf("Failed to create admin user: %v", createErr)
		}

		// Setup test request
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		// Setup route with middleware that sets UserID
		r.PUT("/api/admin/users/:id/admin", func(ctx *gin.Context) {
			ctx.Set("UserID", admin.ID)
			HandleToggleUserAdmin(db)(ctx)
		})

		// Set admin flag to true (already true, should be allowed)
		requestBody, err := json.Marshal(map[string]interface{}{
			"is_admin": true,
		})
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		c.Request = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/users/%d/admin", admin.ID), bytes.NewBuffer(requestBody))
		c.Request.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify admin flag is still true
		var checkAdmin User
		db.DB.First(&checkAdmin, admin.ID)
		if !checkAdmin.IsAdmin {
			t.Error("Admin flag should still be true")
		}
	})

	t.Run("allows admin to grant admin to other users", func(t *testing.T) {
		tmpDir := t.TempDir()
		db, err := InitDB(tmpDir)
		if err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}

		// Create admin user
		admin := User{
			DiscordID: "admin901",
			Username:  "admin",
			IsAdmin:   true,
		}
		if createErr := db.DB.Create(&admin).Error; createErr != nil {
			t.Fatalf("Failed to create admin user: %v", createErr)
		}

		// Create another user to promote
		otherUser := User{
			DiscordID: "user012",
			Username:  "regularuser",
			IsAdmin:   false,
		}
		if createErr := db.DB.Create(&otherUser).Error; createErr != nil {
			t.Fatalf("Failed to create test user: %v", createErr)
		}

		// Setup test request
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		// Setup route with middleware that sets UserID
		r.PUT("/api/admin/users/:id/admin", func(ctx *gin.Context) {
			ctx.Set("UserID", admin.ID)
			HandleToggleUserAdmin(db)(ctx)
		})

		// Grant admin to the other user (should succeed)
		requestBody, err := json.Marshal(map[string]interface{}{
			"is_admin": true,
		})
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		c.Request = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/users/%d/admin", otherUser.ID), bytes.NewBuffer(requestBody))
		c.Request.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify user has admin privileges
		var promotedUser User
		db.DB.First(&promotedUser, otherUser.ID)
		if !promotedUser.IsAdmin {
			t.Error("User should have admin privileges")
		}
	})

	t.Run("allows admin to revoke admin from other users", func(t *testing.T) {
		tmpDir := t.TempDir()
		db, err := InitDB(tmpDir)
		if err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}

		// Create admin user
		admin := User{
			DiscordID: "admin234",
			Username:  "admin",
			IsAdmin:   true,
		}
		if createErr := db.DB.Create(&admin).Error; createErr != nil {
			t.Fatalf("Failed to create admin user: %v", createErr)
		}

		// Create another admin user to demote
		otherAdmin := User{
			DiscordID: "admin567",
			Username:  "otheradmin",
			IsAdmin:   true,
		}
		if createErr := db.DB.Create(&otherAdmin).Error; createErr != nil {
			t.Fatalf("Failed to create test admin: %v", createErr)
		}

		// Setup test request
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		// Setup route with middleware that sets UserID
		r.PUT("/api/admin/users/:id/admin", func(ctx *gin.Context) {
			ctx.Set("UserID", admin.ID)
			HandleToggleUserAdmin(db)(ctx)
		})

		// Revoke admin from the other user (should succeed)
		requestBody, err := json.Marshal(map[string]interface{}{
			"is_admin": false,
		})
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		c.Request = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/users/%d/admin", otherAdmin.ID), bytes.NewBuffer(requestBody))
		c.Request.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, c.Request)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify user no longer has admin privileges
		var demotedUser User
		db.DB.First(&demotedUser, otherAdmin.ID)
		if demotedUser.IsAdmin {
			t.Error("User should no longer have admin privileges")
		}
	})
}
