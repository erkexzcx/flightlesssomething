package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCreateAuditLog(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	user := createTestUser(db, "audituser", false)

	t.Run("creates audit log entry", func(t *testing.T) {
		err := CreateAuditLog(db, user.ID, "CREATE", "Created a test benchmark", "benchmark", 1)
		if err != nil {
			t.Errorf("Failed to create audit log: %v", err)
		}

		// Verify the log was created
		var log AuditLog
		result := db.DB.First(&log)
		if result.Error != nil {
			t.Errorf("Failed to retrieve audit log: %v", result.Error)
		}

		if log.UserID != user.ID {
			t.Errorf("Expected UserID %d, got %d", user.ID, log.UserID)
		}
		if log.Action != "CREATE" {
			t.Errorf("Expected Action 'CREATE', got %s", log.Action)
		}
		if log.Description != "Created a test benchmark" {
			t.Errorf("Expected specific description, got %s", log.Description)
		}
		if log.TargetType != "benchmark" {
			t.Errorf("Expected TargetType 'benchmark', got %s", log.TargetType)
		}
		if log.TargetID != 1 {
			t.Errorf("Expected TargetID 1, got %d", log.TargetID)
		}
	})

}

func TestHandleListAuditLogs(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	adminUser := createTestUser(db, "adminaudit", true)
	regularUser := createTestUser(db, "regularaudit", false)

	// Create some audit logs
	if err := CreateAuditLog(db, regularUser.ID, "CREATE", "Created benchmark 1", "benchmark", 1); err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}
	if err := CreateAuditLog(db, regularUser.ID, "UPDATE", "Updated benchmark 1", "benchmark", 1); err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}
	if err := CreateAuditLog(db, adminUser.ID, "DELETE", "Deleted benchmark 2", "benchmark", 2); err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	router := setupTestRouter()

	// Setup admin route
	router.GET("/api/audit-logs", func(c *gin.Context) {
		c.Set("UserID", adminUser.ID)
		c.Set("Username", adminUser.Username)
		c.Set("IsAdmin", adminUser.IsAdmin)
		c.Next()
	}, HandleListAuditLogs(db))

	t.Run("admin can list audit logs", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/audit-logs", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		logs, ok := response["logs"].([]interface{})
		if !ok {
			t.Fatal("Expected logs array in response")
		}
		if len(logs) < 3 {
			t.Errorf("Expected at least 3 logs, got %d", len(logs))
		}
	})

	t.Run("filters by user_id", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/audit-logs?user_id="+strconv.FormatUint(uint64(regularUser.ID), 10), nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		logs, ok := response["logs"].([]interface{})
		if !ok {
			t.Fatal("Expected logs array in response")
		}
		// All logs should be from regularUser
		for _, logEntry := range logs {
			log, ok := logEntry.(map[string]interface{})
			if !ok {
				t.Fatal("Expected log entry to be map")
			}
			userIDFloat, ok := log["UserID"].(float64)
			if !ok {
				t.Fatal("Expected UserID to be float64")
			}
			userID := uint(userIDFloat)
			if userID != regularUser.ID {
				t.Errorf("Expected UserID %d, got %d", regularUser.ID, userID)
			}
		}
	})

	t.Run("filters by action", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/audit-logs?action=CREATE", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		logs, ok := response["logs"].([]interface{})
		if !ok {
			t.Fatal("Expected logs array in response")
		}
		// All logs should have CREATE action
		for _, logEntry := range logs {
			log, ok := logEntry.(map[string]interface{})
			if !ok {
				t.Fatal("Expected log entry to be map")
			}
			action, ok := log["Action"].(string)
			if !ok {
				t.Fatal("Expected Action to be string")
			}
			if action != "CREATE" {
				t.Errorf("Expected Action 'CREATE', got %s", action)
			}
		}
	})

	t.Run("pagination works", func(t *testing.T) {
		// Create more logs for pagination test
		for i := 0; i < 60; i++ {
			if err := CreateAuditLog(db, regularUser.ID, "TEST", "Test log "+strconv.Itoa(i), "test", uint(i)); err != nil {
				t.Fatalf("Failed to create audit log: %v", err)
			}
		}

		// Request first page
		req, err := http.NewRequest("GET", "/api/audit-logs?page=1&per_page=10", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		logs, ok := response["logs"].([]interface{})
		if !ok {
			t.Fatal("Expected logs array in response")
		}
		if len(logs) != 10 {
			t.Errorf("Expected 10 logs on first page, got %d", len(logs))
		}

		totalFloat, ok := response["total"].(float64)
		if !ok {
			t.Fatal("Expected total to be float64")
		}
		total := int64(totalFloat)
		if total < 60 {
			t.Errorf("Expected at least 60 total logs, got %d", total)
		}
	})
}
