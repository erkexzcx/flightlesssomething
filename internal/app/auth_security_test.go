package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// setupAuthTestRouter creates a test router with session support and RequireAuthOrToken middleware
func setupAuthTestRouter(db *DBInstance) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("test_session", store))
	return r
}

// loginSession performs a session login by setting session values directly
func loginSession(router *gin.Engine, userID uint, username string, isAdmin bool) *http.Cookie {
	// Create a temporary route that sets session values (simulates login)
	loginPath := "/test-login"
	router.GET(loginPath, func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("UserID", userID)
		session.Set("Username", username)
		session.Set("IsAdmin", isAdmin)
		if err := session.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "session save failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "logged in"})
	})

	req := httptest.NewRequest(http.MethodGet, loginPath, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Extract session cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "test_session" {
			return cookie
		}
	}
	return nil
}

// TestBannedUserRejectedViaAPIToken verifies that a banned user cannot access
// protected REST API endpoints using an API token.
func TestBannedUserRejectedViaAPIToken(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	router := setupAuthTestRouter(db)
	authorized := router.Group("/api")
	authorized.Use(RequireAuthOrToken(db))
	authorized.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetUint("UserID")})
	})

	// Create a user and token
	user := createTestUser(db, "bannedtokenuser", false)
	token := &APIToken{UserID: user.ID, Token: "banned-user-token-abcdef1234567890ab", Name: "Test Token"}
	db.DB.Create(token)

	t.Run("token_auth_works_for_active_user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token.Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for active user, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Ban the user
	user.IsBanned = true
	db.DB.Save(user)

	t.Run("token_auth_rejected_for_banned_user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token.Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403 for banned user, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		if !strings.Contains(fmt.Sprintf("%v", response["error"]), "banned") {
			t.Errorf("Expected banned error message, got: %v", response["error"])
		}
	})
}

// TestBannedUserRejectedViaSession verifies that a banned user cannot access
// protected REST API endpoints using a stale session.
func TestBannedUserRejectedViaSession(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	router := setupAuthTestRouter(db)
	authorized := router.Group("/api")
	authorized.Use(RequireAuthOrToken(db))
	authorized.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetUint("UserID")})
	})

	// Create a user and log in
	user := createTestUser(db, "bannedsessionuser", false)
	sessionCookie := loginSession(router, user.ID, user.Username, false)
	if sessionCookie == nil {
		t.Fatal("Failed to obtain session cookie")
	}

	t.Run("session_auth_works_for_active_user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.AddCookie(sessionCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for active user, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Ban the user after they logged in (simulates admin banning while user has active session)
	user.IsBanned = true
	db.DB.Save(user)

	t.Run("session_auth_rejected_for_banned_user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.AddCookie(sessionCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403 for banned user with stale session, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		if !strings.Contains(fmt.Sprintf("%v", response["error"]), "banned") {
			t.Errorf("Expected banned error message, got: %v", response["error"])
		}
	})
}

// TestDeletedUserRejectedViaSession verifies that a deleted user cannot access
// protected endpoints with a stale session.
func TestDeletedUserRejectedViaSession(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	router := setupAuthTestRouter(db)
	authorized := router.Group("/api")
	authorized.Use(RequireAuthOrToken(db))
	authorized.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetUint("UserID")})
	})

	// Create a user and log in
	user := createTestUser(db, "deletedsessionuser", false)
	sessionCookie := loginSession(router, user.ID, user.Username, false)
	if sessionCookie == nil {
		t.Fatal("Failed to obtain session cookie")
	}

	// Delete the user from DB after login
	db.DB.Delete(user)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for deleted user with stale session, got %d: %s", w.Code, w.Body.String())
	}
}

// TestDemotedAdminRejectedViaSession verifies that an admin whose privileges were
// revoked cannot access admin routes with a stale session.
func TestDemotedAdminRejectedViaSession(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	router := setupAuthTestRouter(db)
	admin := router.Group("/api/admin")
	admin.Use(RequireAuthOrToken(db), RequireAdmin())
	admin.GET("/users", HandleListUsers(db))

	// Create an admin user and log in with admin privileges
	user := createTestUser(db, "demotedadmin", true)
	sessionCookie := loginSession(router, user.ID, user.Username, true)
	if sessionCookie == nil {
		t.Fatal("Failed to obtain session cookie")
	}

	t.Run("admin_access_works_before_demotion", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
		req.AddCookie(sessionCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for admin, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Revoke admin privileges (simulates another admin demoting this user)
	user.IsAdmin = false
	db.DB.Save(user)

	t.Run("admin_access_rejected_after_demotion", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
		req.AddCookie(sessionCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403 for demoted admin with stale session, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// TestDemotedAdminRejectedViaAPIToken verifies that an admin whose privileges were
// revoked cannot access admin routes via API token.
func TestDemotedAdminRejectedViaAPIToken(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	router := setupAuthTestRouter(db)
	admin := router.Group("/api/admin")
	admin.Use(RequireAuthOrToken(db), RequireAdmin())
	admin.GET("/users", HandleListUsers(db))

	// Create an admin user with token
	user := createTestUser(db, "demotedadmintoken", true)
	token := &APIToken{UserID: user.ID, Token: "demoted-admin-token-abcdef12345678", Name: "Admin Token"}
	db.DB.Create(token)

	t.Run("admin_token_access_works_before_demotion", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+token.Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for admin token, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Revoke admin privileges
	user.IsAdmin = false
	db.DB.Save(user)

	t.Run("admin_token_access_rejected_after_demotion", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+token.Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403 for demoted admin token, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// TestNonAdminCannotAccessAdminRoutes verifies that regular users cannot access
// admin endpoints via any auth method.
func TestNonAdminCannotAccessAdminRoutes(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	router := setupAuthTestRouter(db)
	admin := router.Group("/api/admin")
	admin.Use(RequireAuthOrToken(db), RequireAdmin())
	admin.GET("/users", HandleListUsers(db))
	admin.DELETE("/users/:id", HandleDeleteUser(db))
	admin.PUT("/users/:id/ban", HandleBanUser(db))
	admin.PUT("/users/:id/admin", HandleToggleUserAdmin(db))
	admin.GET("/logs", HandleListAuditLogs(db))

	user := createTestUser(db, "regularuser", false)
	token := &APIToken{UserID: user.ID, Token: "regular-user-token-abcdef1234567890", Name: "User Token"}
	db.DB.Create(token)

	routes := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/api/admin/users", ""},
		{"DELETE", "/api/admin/users/1", ""},
		{"PUT", "/api/admin/users/1/ban", `{"banned":true}`},
		{"PUT", "/api/admin/users/1/admin", `{"is_admin":true}`},
		{"GET", "/api/admin/logs", ""},
	}

	for _, route := range routes {
		t.Run(fmt.Sprintf("non_admin_token_%s_%s", route.method, route.path), func(t *testing.T) {
			var body *bytes.Reader
			if route.body != "" {
				body = bytes.NewReader([]byte(route.body))
			} else {
				body = bytes.NewReader(nil)
			}
			req := httptest.NewRequest(route.method, route.path, body)
			req.Header.Set("Authorization", "Bearer "+token.Token)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Errorf("Expected 403 for non-admin on %s %s, got %d: %s", route.method, route.path, w.Code, w.Body.String())
			}
		})
	}
}

// TestUnauthenticatedCannotAccessProtectedRoutes verifies that unauthenticated
// requests are rejected on protected endpoints.
func TestUnauthenticatedCannotAccessProtectedRoutes(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	router := setupAuthTestRouter(db)
	authorized := router.Group("/api")
	authorized.Use(RequireAuthOrToken(db))
	authorized.POST("/benchmarks", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "created"})
	})
	authorized.GET("/tokens", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"tokens": []string{}})
	})

	admin := router.Group("/api/admin")
	admin.Use(RequireAuthOrToken(db), RequireAdmin())
	admin.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": []string{}})
	})

	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/benchmarks"},
		{"GET", "/api/tokens"},
		{"GET", "/api/admin/users"},
	}

	for _, route := range routes {
		t.Run(fmt.Sprintf("unauthenticated_%s_%s", route.method, route.path), func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected 401 for unauthenticated on %s %s, got %d: %s", route.method, route.path, w.Code, w.Body.String())
			}
		})
	}
}

// TestSessionContextReflectsCurrentDBState verifies that session-based auth
// sets context values from the database, not from stale session values.
func TestSessionContextReflectsCurrentDBState(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	router := setupAuthTestRouter(db)
	authorized := router.Group("/api")
	authorized.Use(RequireAuthOrToken(db))
	authorized.GET("/whoami", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":  c.GetUint("UserID"),
			"username": c.GetString("Username"),
			"is_admin": c.GetBool("IsAdmin"),
		})
	})

	// Create user as non-admin
	user := createTestUser(db, "contextuser", false)
	// Log in with non-admin session
	sessionCookie := loginSession(router, user.ID, user.Username, false)
	if sessionCookie == nil {
		t.Fatal("Failed to obtain session cookie")
	}

	// Promote user to admin in DB (after session was created)
	user.IsAdmin = true
	db.DB.Save(user)

	// The context should reflect the current DB state (admin=true), not the stale session value (admin=false)
	req := httptest.NewRequest(http.MethodGet, "/api/whoami", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	isAdmin, ok := response["is_admin"].(bool)
	if !ok || !isAdmin {
		t.Errorf("Expected is_admin=true from DB, got %v (context should reflect DB, not stale session)", response["is_admin"])
	}
}

// TestBenchmarkOwnershipEnforcement verifies that users can only modify their own
// benchmarks (unless they are admins).
func TestBenchmarkOwnershipEnforcement(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	router := setupAuthTestRouter(db)
	authorized := router.Group("/api")
	authorized.Use(RequireAuthOrToken(db))
	authorized.PUT("/benchmarks/:id", HandleUpdateBenchmark(db))
	authorized.DELETE("/benchmarks/:id", HandleDeleteBenchmark(db))

	// Create two users
	owner := createTestUser(db, "benchowner", false)
	other := createTestUser(db, "benchother", false)

	// Create a benchmark owned by the first user
	benchmark := &Benchmark{UserID: owner.ID, Title: "Owner Benchmark", Description: "Test"}
	db.DB.Create(benchmark)

	ownerToken := &APIToken{UserID: owner.ID, Token: "owner-token-abcdef12345678901234", Name: "Owner Token"}
	otherToken := &APIToken{UserID: other.ID, Token: "other-token-abcdef12345678901234", Name: "Other Token"}
	db.DB.Create(ownerToken)
	db.DB.Create(otherToken)

	t.Run("owner_can_update_own_benchmark", func(t *testing.T) {
		body := `{"title":"Updated Title"}`
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/benchmarks/%d", benchmark.ID), strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+ownerToken.Token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for owner update, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("other_user_cannot_update_benchmark", func(t *testing.T) {
		body := `{"title":"Hacked Title"}`
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/benchmarks/%d", benchmark.ID), strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+otherToken.Token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403 for non-owner update, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("other_user_cannot_delete_benchmark", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/benchmarks/%d", benchmark.ID), nil)
		req.Header.Set("Authorization", "Bearer "+otherToken.Token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403 for non-owner delete, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// TestAdminCanAccessOtherUsersBenchmarks verifies that admins can modify
// benchmarks owned by other users.
func TestAdminCanAccessOtherUsersBenchmarks(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	router := setupAuthTestRouter(db)
	authorized := router.Group("/api")
	authorized.Use(RequireAuthOrToken(db))
	authorized.PUT("/benchmarks/:id", HandleUpdateBenchmark(db))

	// Create a regular user and an admin
	owner := createTestUser(db, "benchowneradm", false)
	admin := createTestUser(db, "benchadmin", true)

	// Create a benchmark owned by the regular user
	benchmark := &Benchmark{UserID: owner.ID, Title: "Owned Benchmark", Description: "Test"}
	db.DB.Create(benchmark)

	adminToken := &APIToken{UserID: admin.ID, Token: "admin-bench-token-abcdef1234567890", Name: "Admin Token"}
	db.DB.Create(adminToken)

	body := `{"title":"Admin Updated Title"}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/benchmarks/%d", benchmark.ID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken.Token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for admin update, got %d: %s", w.Code, w.Body.String())
	}
}

// TestMCPBannedUserRejectedForAuthTools verifies that banned users cannot access
// authenticated MCP tools.
func TestMCPBannedUserRejectedForAuthTools(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpbannedauth", false)
	user.IsBanned = true
	db.DB.Save(user)

	token := &APIToken{UserID: user.ID, Token: "banned-mcp-auth-token-abcdef12345", Name: "Banned Token"}
	db.DB.Create(token)

	authTools := []string{"list_api_tokens", "create_api_token"}
	for _, tool := range authTools {
		t.Run(tool, func(t *testing.T) {
			var body string
			if tool == "create_api_token" {
				body = fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"%s","arguments":{"name":"test"}}}`, tool)
			} else {
				body = fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"%s","arguments":{}}}`, tool)
			}
			w := mcpRequest(t, router, body, token.Token)

			_, result := parseMCPToolResult(t, w)
			if !result.IsError {
				t.Errorf("Expected error for banned user on %s", tool)
			}
			if !strings.Contains(result.Content[0].Text, "authentication required") {
				t.Errorf("Expected authentication error for banned user on %s, got: %s", tool, result.Content[0].Text)
			}
		})
	}
}

// TestMCPBannedUserCanStillAccessPublicTools verifies that banned users can still
// access public (unauthenticated) MCP tools.
func TestMCPBannedUserCanStillAccessPublicTools(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	// Public tool should work without auth, even with a banned user's token
	user := createTestUser(db, "mcpbannedpub", false)
	user.IsBanned = true
	db.DB.Save(user)

	token := &APIToken{UserID: user.ID, Token: "banned-mcp-pub-token-abcdef12345a", Name: "Banned Token"}
	db.DB.Create(token)

	// list_benchmarks is public - should work (token is just ignored for public tools)
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_benchmarks","arguments":{}}}`
	w := mcpRequest(t, router, body, "")

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Errorf("Expected success for public tool, got error: %s", result.Content[0].Text)
	}
}

// TestMCPNonAdminCannotAccessAdminTools verifies that non-admin users cannot
// access admin MCP tools even with valid tokens.
func TestMCPNonAdminCannotAccessAdminTools(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpnonadmintools", false)
	token := &APIToken{UserID: user.ID, Token: "nonadmin-tools-token-abcdef123456", Name: "Non-Admin Token"}
	db.DB.Create(token)

	adminTools := []string{"list_users", "list_audit_logs", "delete_user", "delete_user_benchmarks", "ban_user", "toggle_user_admin"}
	for _, tool := range adminTools {
		t.Run(tool, func(t *testing.T) {
			body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"%s","arguments":{"user_id":1}}}`, tool)
			w := mcpRequest(t, router, body, token.Token)

			_, result := parseMCPToolResult(t, w)
			if !result.IsError {
				t.Errorf("Expected error for non-admin on %s", tool)
			}
			if !strings.Contains(result.Content[0].Text, "admin privileges required") {
				t.Errorf("Expected admin privileges error for %s, got: %s", tool, result.Content[0].Text)
			}
		})
	}
}

// TestMCPUnauthenticatedCannotAccessAuthTools verifies that unauthenticated
// requests are rejected for auth-required MCP tools.
func TestMCPUnauthenticatedCannotAccessAuthTools(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	authTools := []string{"update_benchmark", "delete_benchmark", "delete_benchmark_run", "list_api_tokens", "create_api_token", "delete_api_token"}
	for _, tool := range authTools {
		t.Run(tool, func(t *testing.T) {
			body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"%s","arguments":{"id":1,"token_id":1,"name":"test","run_index":0}}}`, tool)
			w := mcpRequest(t, router, body, "") // No token

			_, result := parseMCPToolResult(t, w)
			if !result.IsError {
				t.Errorf("Expected error for unauthenticated on %s", tool)
			}
			if !strings.Contains(result.Content[0].Text, "authentication required") {
				t.Errorf("Expected authentication required error for %s, got: %s", tool, result.Content[0].Text)
			}
		})
	}
}

// TestMCPToolsListFilteredByAuthLevel verifies that the tools/list response
// only shows tools appropriate for the caller's authentication level.
func TestMCPToolsListFilteredByAuthLevel(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcptoolslist", false)
	userToken := &APIToken{UserID: user.ID, Token: "toolslist-user-token-abcdef1234567", Name: "User Token"}
	db.DB.Create(userToken)

	admin := createTestUser(db, "mcptoolslistadmin", true)
	adminToken := &APIToken{UserID: admin.ID, Token: "toolslist-admin-token-abcdef123456", Name: "Admin Token"}
	db.DB.Create(adminToken)

	adminOnlyTools := []string{"list_users", "list_audit_logs", "delete_user", "delete_user_benchmarks", "ban_user", "toggle_user_admin"}

	t.Run("anonymous_sees_only_public_tools", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
		w := mcpRequest(t, router, body, "")

		var resp jsonrpcResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		resultBytes, err := json.Marshal(resp.Result)
		if err != nil {
			t.Fatalf("Failed to marshal result: %v", err)
		}
		var toolsList mcpToolsListResult
		if err := json.Unmarshal(resultBytes, &toolsList); err != nil {
			t.Fatalf("Failed to unmarshal tools list: %v", err)
		}

		for _, tool := range toolsList.Tools {
			for _, adminTool := range adminOnlyTools {
				if tool.Name == adminTool {
					t.Errorf("Anonymous user should not see admin tool: %s", tool.Name)
				}
			}
			// Auth tools should also be hidden from anonymous
			authTools := []string{"update_benchmark", "delete_benchmark", "list_api_tokens", "create_api_token"}
			for _, authTool := range authTools {
				if tool.Name == authTool {
					t.Errorf("Anonymous user should not see auth tool: %s", tool.Name)
				}
			}
		}
	})

	t.Run("regular_user_sees_public_and_auth_tools_but_not_admin", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
		w := mcpRequest(t, router, body, userToken.Token)

		var resp jsonrpcResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		resultBytes, err := json.Marshal(resp.Result)
		if err != nil {
			t.Fatalf("Failed to marshal result: %v", err)
		}
		var toolsList mcpToolsListResult
		if err := json.Unmarshal(resultBytes, &toolsList); err != nil {
			t.Fatalf("Failed to unmarshal tools list: %v", err)
		}

		for _, tool := range toolsList.Tools {
			for _, adminTool := range adminOnlyTools {
				if tool.Name == adminTool {
					t.Errorf("Non-admin user should not see admin tool: %s", tool.Name)
				}
			}
		}
	})

	t.Run("admin_user_sees_all_tools", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
		w := mcpRequest(t, router, body, adminToken.Token)

		var resp jsonrpcResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		resultBytes, err := json.Marshal(resp.Result)
		if err != nil {
			t.Fatalf("Failed to marshal result: %v", err)
		}
		var toolsList mcpToolsListResult
		if err := json.Unmarshal(resultBytes, &toolsList); err != nil {
			t.Fatalf("Failed to unmarshal tools list: %v", err)
		}

		// Admin should see all admin tools
		toolNames := make(map[string]bool)
		for _, tool := range toolsList.Tools {
			toolNames[tool.Name] = true
		}
		for _, adminTool := range adminOnlyTools {
			if !toolNames[adminTool] {
				t.Errorf("Admin user should see admin tool: %s", adminTool)
			}
		}
	})
}

// TestAPITokenCannotDeleteOtherUsersToken verifies that a user cannot delete
// another user's API token.
func TestAPITokenCannotDeleteOtherUsersToken(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	router := setupAuthTestRouter(db)
	authorized := router.Group("/api")
	authorized.Use(RequireAuthOrToken(db))
	authorized.DELETE("/tokens/:id", HandleDeleteAPIToken(db))

	user1 := createTestUser(db, "tokenowner1", false)
	user2 := createTestUser(db, "tokenowner2", false)

	token1 := &APIToken{UserID: user1.ID, Token: "user1-auth-token-abcdef1234567890ab", Name: "User1 Token"}
	token2 := &APIToken{UserID: user2.ID, Token: "user2-auth-token-abcdef1234567890ab", Name: "User2 Token"}
	db.DB.Create(token1)
	db.DB.Create(token2)

	// User2 tries to delete User1's token
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/tokens/%d", token1.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token2.Token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 when trying to delete other user's token, got %d: %s", w.Code, w.Body.String())
	}

	// Verify token1 still exists
	var count int64
	db.DB.Model(&APIToken{}).Where("id = ?", token1.ID).Count(&count)
	if count != 1 {
		t.Error("Token should not have been deleted")
	}
}
