package app

import (
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

func setupMCPTestRouter(db *DBInstance) *gin.Engine {
	gin.SetMode(gin.TestMode)
	InitRateLimiters()
	r := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("test_session", store))
	mcp := r.Group("/mcp")
	mcp.Use(MCPCors())
	mcp.OPTIONS("", func(c *gin.Context) {})
	mcp.POST("", HandleMCP(db, "test"))
	mcp.GET("", HandleMCPGet)
	mcp.DELETE("", HandleMCPDelete)
	return r
}

func mcpRequest(t *testing.T, router *gin.Engine, body string, token string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("POST", "/mcp", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// parseMCPToolResult parses the JSON-RPC response and extracts the tool result
func parseMCPToolResult(t *testing.T, w *httptest.ResponseRecorder) (jsonrpcResponse, mcpToolResult) {
	t.Helper()
	var resp jsonrpcResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}
	var result mcpToolResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal tool result: %v", err)
	}
	return resp, result
}

func TestMCPInitialize(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	w := mcpRequest(t, router, body, "")

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp jsonrpcResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %s", resp.Error.Message)
	}

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}
	var result mcpInitializeResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	if result.ProtocolVersion != "2025-11-25" {
		t.Errorf("Expected protocol version 2025-11-25, got %s", result.ProtocolVersion)
	}
	if result.ServerInfo.Name != "FlightlessSomething" {
		t.Errorf("Expected server name FlightlessSomething, got %s", result.ServerInfo.Name)
	}
	if result.Capabilities.Tools == nil {
		t.Error("Expected tools capability to be set")
	}
}

func TestMCPToolsList(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	// Helper to parse tools/list response
	parseToolsList := func(t *testing.T, w *httptest.ResponseRecorder) []string {
		t.Helper()
		var resp jsonrpcResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if resp.Error != nil {
			t.Fatalf("Unexpected error: %s", resp.Error.Message)
		}
		resultBytes, err := json.Marshal(resp.Result)
		if err != nil {
			t.Fatalf("Failed to marshal result: %v", err)
		}
		var result mcpToolsListResult
		if err := json.Unmarshal(resultBytes, &result); err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
		}
		names := make([]string, len(result.Tools))
		for i, tool := range result.Tools {
			names[i] = tool.Name
		}
		return names
	}

	body := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`

	// Anonymous: should only see public tools (4)
	t.Run("anonymous sees only public tools", func(t *testing.T) {
		w := mcpRequest(t, router, body, "")
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}
		names := parseToolsList(t, w)
		publicTools := []string{
			"list_benchmarks", "get_benchmark", "get_benchmark_data",
			"get_benchmark_run",
		}
		if len(names) != len(publicTools) {
			t.Errorf("Expected %d public tools, got %d: %v", len(publicTools), len(names), names)
		}
		nameSet := make(map[string]bool)
		for _, n := range names {
			nameSet[n] = true
		}
		for _, n := range publicTools {
			if !nameSet[n] {
				t.Errorf("Missing public tool: %s", n)
			}
		}
		// Admin tools must NOT be visible
		for _, n := range names {
			if n == "list_users" || n == "delete_user" || n == "ban_user" {
				t.Errorf("Anonymous should not see admin tool: %s", n)
			}
		}
		// Removed data tools must NOT be visible
		for _, n := range names {
			if n == "create_benchmark" || n == "add_benchmark_runs" || n == "download_benchmark" {
				t.Errorf("Removed data tool should not be visible: %s", n)
			}
		}
	})

	// Authenticated regular user: should see public + auth tools (5)
	t.Run("regular user sees public and auth tools", func(t *testing.T) {
		user := createTestUser(db, "mcptoolslistuser", false)
		apiToken := &APIToken{UserID: user.ID, Token: "toolslist-user-token-abcdef123", Name: "ToolsList Token"}
		db.DB.Create(apiToken)

		w := mcpRequest(t, router, body, apiToken.Token)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}
		names := parseToolsList(t, w)
		if len(names) != 5 {
			t.Errorf("Expected 5 tools for regular user, got %d: %v", len(names), names)
		}
		// Should include auth tools
		nameSet := make(map[string]bool)
		for _, n := range names {
			nameSet[n] = true
		}
		for _, required := range []string{
			"list_benchmarks", "get_benchmark", "get_benchmark_data", "get_benchmark_run",
			"update_benchmark",
		} {
			if !nameSet[required] {
				t.Errorf("Missing auth tool: %s", required)
			}
		}
		// Removed tools must NOT be visible
		for _, n := range names {
			if n == "get_current_user" || n == "delete_benchmark" || n == "delete_benchmark_run" ||
				n == "list_api_tokens" || n == "create_api_token" || n == "delete_api_token" {
				t.Errorf("Removed tool should not be visible: %s", n)
			}
		}
		// Admin tools must NOT be visible
		for _, n := range names {
			if n == "list_users" || n == "delete_user" || n == "ban_user" {
				t.Errorf("Regular user should not see admin tool: %s", n)
			}
		}
	})

	// Admin user: should see all tools (10)
	t.Run("admin sees all tools", func(t *testing.T) {
		admin := createTestUser(db, "mcptoolslistadmin", true)
		adminToken := &APIToken{UserID: admin.ID, Token: "toolslist-admin-token-abcdef12", Name: "ToolsList Admin"}
		db.DB.Create(adminToken)

		w := mcpRequest(t, router, body, adminToken.Token)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}
		names := parseToolsList(t, w)
		allTools := []string{
			"list_benchmarks", "get_benchmark", "get_benchmark_data",
			"get_benchmark_run",
			"update_benchmark",
			"list_users", "delete_user",
			"delete_user_benchmarks", "ban_user", "toggle_user_admin",
		}
		if len(names) != len(allTools) {
			t.Errorf("Expected %d tools for admin, got %d: %v", len(allTools), len(names), names)
		}
		nameSet := make(map[string]bool)
		for _, n := range names {
			nameSet[n] = true
		}
		for _, n := range allTools {
			if !nameSet[n] {
				t.Errorf("Admin missing tool: %s", n)
			}
		}
	})
}

func TestMCPToolAnnotations(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	server := newMCPServer(db, "test")

	// Build a map of tool name -> tool for easy lookup
	toolMap := make(map[string]mcpTool)
	for _, tool := range server.tools {
		toolMap[tool.Name] = tool
	}

	boolVal := func(b *bool) bool {
		if b == nil {
			return false
		}
		return *b
	}

	type expectedAnnotations struct {
		readOnly    bool
		destructive bool
		idempotent  bool
		openWorld   bool
	}

	tests := map[string]expectedAnnotations{
		// Read-only public tools
		"list_benchmarks":    {readOnly: true, destructive: false, idempotent: false, openWorld: false},
		"get_benchmark":      {readOnly: true, destructive: false, idempotent: false, openWorld: false},
		"get_benchmark_data": {readOnly: true, destructive: false, idempotent: false, openWorld: false},
		"get_benchmark_run":  {readOnly: true, destructive: false, idempotent: false, openWorld: false},

		// Auth tools - write operations
		"update_benchmark": {readOnly: false, destructive: false, idempotent: true, openWorld: false},

		// Admin tools
		"list_users":             {readOnly: true, destructive: false, idempotent: false, openWorld: false},
		"delete_user":            {readOnly: false, destructive: true, idempotent: false, openWorld: false},
		"delete_user_benchmarks": {readOnly: false, destructive: true, idempotent: false, openWorld: false},
		"ban_user":               {readOnly: false, destructive: false, idempotent: true, openWorld: false},
		"toggle_user_admin":      {readOnly: false, destructive: false, idempotent: true, openWorld: false},
	}

	// Verify all tools are covered
	if len(tests) != len(server.tools) {
		t.Errorf("Test covers %d tools but server defines %d tools", len(tests), len(server.tools))
	}

	for name, expected := range tests {
		t.Run(name, func(t *testing.T) {
			tool, ok := toolMap[name]
			if !ok {
				t.Fatalf("Tool %s not found in server tools", name)
			}
			if tool.Annotations == nil {
				t.Fatal("Annotations must not be nil")
			}
			if boolVal(tool.Annotations.ReadOnlyHint) != expected.readOnly {
				t.Errorf("ReadOnlyHint: got %v, want %v", boolVal(tool.Annotations.ReadOnlyHint), expected.readOnly)
			}
			if boolVal(tool.Annotations.DestructiveHint) != expected.destructive {
				t.Errorf("DestructiveHint: got %v, want %v", boolVal(tool.Annotations.DestructiveHint), expected.destructive)
			}
			if boolVal(tool.Annotations.IdempotentHint) != expected.idempotent {
				t.Errorf("IdempotentHint: got %v, want %v", boolVal(tool.Annotations.IdempotentHint), expected.idempotent)
			}
			if boolVal(tool.Annotations.OpenWorldHint) != expected.openWorld {
				t.Errorf("OpenWorldHint: got %v, want %v", boolVal(tool.Annotations.OpenWorldHint), expected.openWorld)
			}
		})
	}
}

func TestMCPToolIcons(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	server := newMCPServer(db, "test")

	for _, tool := range server.tools {
		t.Run(tool.Name, func(t *testing.T) {
			if len(tool.Icons) == 0 {
				t.Fatal("Tool must have at least one icon")
			}
			for i, icon := range tool.Icons {
				if icon.Src == "" {
					t.Errorf("Icon[%d].Src must not be empty", i)
				}
				if icon.MIMEType == "" {
					t.Errorf("Icon[%d].MIMEType should be set", i)
				}
				if len(icon.Sizes) == 0 {
					t.Errorf("Icon[%d].Sizes should be set", i)
				}
			}
		})
	}
}

func TestMCPNotification(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	body := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	w := mcpRequest(t, router, body, "")

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected 202, got %d", w.Code)
	}
}

func TestMCPPing(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	body := `{"jsonrpc":"2.0","id":1,"method":"ping"}`
	w := mcpRequest(t, router, body, "")

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp jsonrpcResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %s", resp.Error.Message)
	}
}

func TestMCPListBenchmarksAnonymous(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpuser", false)
	db.DB.Create(&Benchmark{Title: "Test Bench", Description: "Desc", UserID: user.ID})

	body := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_benchmarks","arguments":{"page":1,"per_page":10}}}`
	w := mcpRequest(t, router, body, "")

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	_, result := parseMCPToolResult(t, w)
	if len(result.Content) == 0 {
		t.Fatal("Expected content in result")
	}
	if result.Content[0].Type != "text" {
		t.Errorf("Expected text content, got %s", result.Content[0].Type)
	}
	if !strings.Contains(result.Content[0].Text, "Test Bench") {
		t.Error("Expected benchmark title in result")
	}
}

func TestMCPGetBenchmarkAnonymous(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpuser2", false)
	b := &Benchmark{Title: "Detail Test", Description: "Detailed", UserID: user.ID}
	db.DB.Create(b)

	body := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"get_benchmark","arguments":{"id":` + idStr(b.ID) + `}}}`
	w := mcpRequest(t, router, body, "")

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	_, result := parseMCPToolResult(t, w)
	if !strings.Contains(result.Content[0].Text, "Detail Test") {
		t.Error("Expected benchmark title in result")
	}
}

func TestMCPWriteToolRequiresAuth(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpuser3", false)
	b := &Benchmark{Title: "Auth Test", UserID: user.ID}
	db.DB.Create(b)

	body := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"update_benchmark","arguments":{"id":` + idStr(b.ID) + `,"title":"Hacked"}}}`
	w := mcpRequest(t, router, body, "")

	_, result := parseMCPToolResult(t, w)
	if !result.IsError {
		t.Error("Expected error for unauthenticated write operation")
	}
	if !strings.Contains(result.Content[0].Text, "authentication required") {
		t.Error("Expected authentication error message")
	}
}

func TestMCPWriteToolWithAuth(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpauthuser", false)
	apiToken := &APIToken{UserID: user.ID, Token: "test-mcp-token-abcdef1234567890", Name: "MCP Test"}
	db.DB.Create(apiToken)

	b := &Benchmark{Title: "Update Me", Description: "Old desc", UserID: user.ID}
	db.DB.Create(b)

	body := `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"update_benchmark","arguments":{"id":` + idStr(b.ID) + `,"title":"Updated Title"}}}`
	w := mcpRequest(t, router, body, apiToken.Token)

	resp, result := parseMCPToolResult(t, w)
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %s", resp.Error.Message)
	}
	if result.IsError {
		t.Fatalf("Unexpected tool error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "Updated Title") {
		t.Error("Expected updated title in result")
	}

	// Verify in DB
	var updated Benchmark
	db.DB.First(&updated, b.ID)
	if updated.Title != "Updated Title" {
		t.Errorf("Expected 'Updated Title', got '%s'", updated.Title)
	}
}

func TestMCPOwnershipCheck(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	owner := createTestUser(db, "mcpowner", false)
	other := createTestUser(db, "mcpother", false)

	apiToken := &APIToken{UserID: other.ID, Token: "other-token-abcdef1234567890abcd", Name: "Other Token"}
	db.DB.Create(apiToken)

	b := &Benchmark{Title: "Owner Only", UserID: owner.ID}
	db.DB.Create(b)

	body := `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"update_benchmark","arguments":{"id":` + idStr(b.ID) + `,"title":"Hacked"}}}`
	w := mcpRequest(t, router, body, apiToken.Token)

	_, result := parseMCPToolResult(t, w)
	if !result.IsError {
		t.Error("Expected error for non-owner update")
	}
	if !strings.Contains(result.Content[0].Text, "not authorized") {
		t.Error("Expected authorization error message")
	}
}

func TestMCPBannedUserRejected(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpbanned", false)
	user.IsBanned = true
	db.DB.Save(user)

	apiToken := &APIToken{UserID: user.ID, Token: "banned-token-abcdef12345678901", Name: "Banned Token"}
	db.DB.Create(apiToken)

	b := &Benchmark{Title: "Some Bench", UserID: user.ID}
	db.DB.Create(b)

	body := `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"update_benchmark","arguments":{"id":` + idStr(b.ID) + `,"title":"Hacked"}}}`
	w := mcpRequest(t, router, body, apiToken.Token)

	_, result := parseMCPToolResult(t, w)
	if !result.IsError {
		t.Error("Expected error for banned user")
	}
	if !strings.Contains(result.Content[0].Text, "authentication required") {
		t.Error("Expected authentication error for banned user")
	}
}

func TestMCPMethodNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	body := `{"jsonrpc":"2.0","id":9,"method":"unknown/method"}`
	w := mcpRequest(t, router, body, "")

	var resp jsonrpcResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("Expected error for unknown method")
	}
	if resp.Error.Code != jsonrpcMethodNotFound {
		t.Errorf("Expected error code %d, got %d", jsonrpcMethodNotFound, resp.Error.Code)
	}
}

func TestMCPInvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	w := mcpRequest(t, router, "not json", "")

	var resp jsonrpcResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("Expected error for invalid JSON")
	}
	if resp.Error.Code != jsonrpcParseError {
		t.Errorf("Expected error code %d, got %d", jsonrpcParseError, resp.Error.Code)
	}
}

func idStr(id uint) string {
	return fmt.Sprintf("%d", id)
}

func TestMCPInitializeWithAuthContext(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpinituser", false)
	apiToken := &APIToken{UserID: user.ID, Token: "init-context-token-abcdef123456", Name: "Init Token"}
	db.DB.Create(apiToken)

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	w := mcpRequest(t, router, body, apiToken.Token)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp jsonrpcResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}
	var result mcpInitializeResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Should contain user context
	if !strings.Contains(result.Instructions, "mcpinituser") {
		t.Error("Expected username in initialize instructions")
	}
	if !strings.Contains(result.Instructions, fmt.Sprintf("User ID: %d", user.ID)) {
		t.Error("Expected user ID in initialize instructions")
	}
	// Should contain base URL
	if !strings.Contains(result.Instructions, "Server base URL:") {
		t.Error("Expected base URL in initialize instructions")
	}
	// Should NOT contain anonymous mode
	if strings.Contains(result.Instructions, "Anonymous mode") {
		t.Error("Should not contain anonymous mode for authenticated user")
	}
	// Should NOT contain curl instructions (removed in favor of "not supported" message)
	if strings.Contains(result.Instructions, "curl") {
		t.Error("Should not contain curl instructions (removed)")
	}
	// Should contain "not supported" message for data operations
	if !strings.Contains(result.Instructions, "does not support") {
		t.Error("Expected 'does not support' message for data operations")
	}
}

func TestMCPInitializeAnonymous(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	w := mcpRequest(t, router, body, "")

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp jsonrpcResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}
	var result mcpInitializeResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Should contain anonymous mode notice
	if !strings.Contains(result.Instructions, "Anonymous mode") {
		t.Error("Expected anonymous mode notice in initialize instructions")
	}
	// Should contain base URL
	if !strings.Contains(result.Instructions, "Server base URL:") {
		t.Error("Expected base URL in initialize instructions")
	}
	// Should NOT contain user context
	if strings.Contains(result.Instructions, "Authenticated user context") {
		t.Error("Should not contain user context for anonymous user")
	}
}

func TestMCPListBenchmarksByUsername(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user1 := createTestUser(db, "benchuser1", false)
	user2 := createTestUser(db, "benchuser2", false)
	db.DB.Create(&Benchmark{Title: "User1 Bench", UserID: user1.ID})
	db.DB.Create(&Benchmark{Title: "User2 Bench", UserID: user2.ID})

	// Filter by username
	body := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_benchmarks","arguments":{"username":"benchuser1"}}}`
	w := mcpRequest(t, router, body, "")

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "User1 Bench") {
		t.Error("Expected user1's benchmark in result")
	}
	if strings.Contains(result.Content[0].Text, "User2 Bench") {
		t.Error("Should not contain user2's benchmark when filtering by user1")
	}

	// Case-insensitive username match
	body = `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_benchmarks","arguments":{"username":"BENCHUSER1"}}}`
	w = mcpRequest(t, router, body, "")

	_, result = parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error for case-insensitive match: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "User1 Bench") {
		t.Error("Expected case-insensitive match to find benchmarks")
	}

	// Non-existent username
	body = `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"list_benchmarks","arguments":{"username":"nonexistent"}}}`
	w = mcpRequest(t, router, body, "")

	_, result = parseMCPToolResult(t, w)
	if !result.IsError {
		t.Error("Expected error for non-existent username")
	}
	if !strings.Contains(result.Content[0].Text, "user not found") {
		t.Error("Expected user not found error")
	}
}

func TestMCPGetBenchmarkDataIncludesMetadata(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	if err := InitBenchmarksDir(t.TempDir()); err != nil {
		t.Fatalf("Failed to init benchmarks dir: %v", err)
	}
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpmetadata", false)
	benchID := mcpCreateBenchmarkHelper(t, db, user.ID)

	// Update benchmark title for verification
	db.DB.Model(&Benchmark{}).Where("id = ?", benchID).Update("title", "Metadata Test Bench")

	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":40,"method":"tools/call","params":{"name":"get_benchmark_data","arguments":{"id":%d}}}`, benchID)
	w := mcpRequest(t, router, body, "")

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}
	// Should contain benchmark metadata
	if !strings.Contains(result.Content[0].Text, "Metadata Test Bench") {
		t.Error("Expected benchmark title in result")
	}
	if !strings.Contains(result.Content[0].Text, "mcpmetadata") {
		t.Error("Expected username in result")
	}
	// Should contain runs data
	if !strings.Contains(result.Content[0].Text, "runs") {
		t.Error("Expected runs field in result")
	}
	// Should contain benchmark field
	if !strings.Contains(result.Content[0].Text, "benchmark") {
		t.Error("Expected benchmark field in result")
	}
}

func TestMCPRemovedToolsReturnError(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpremovedtools", false)
	apiToken := &APIToken{UserID: user.ID, Token: "removed-tools-token-abcdef1234", Name: "Removed Tools Token"}
	db.DB.Create(apiToken)

	removedTools := []string{
		"get_current_user", "delete_benchmark", "delete_benchmark_run",
		"list_api_tokens", "create_api_token", "delete_api_token",
	}
	for _, tool := range removedTools {
		body := fmt.Sprintf(`{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"%s","arguments":{}}}`, tool)
		w := mcpRequest(t, router, body, apiToken.Token)

		_, result := parseMCPToolResult(t, w)
		if !result.IsError {
			t.Errorf("Expected error for removed tool: %s", tool)
		}
	}
}

func TestMCPJQFilter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpjquser", false)
	db.DB.Create(&Benchmark{Title: "JQ Test Bench", Description: "Testing jq", UserID: user.ID})
	db.DB.Create(&Benchmark{Title: "Second Bench", Description: "Another", UserID: user.ID})

	t.Run("jq filters list_benchmarks results", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":12,"method":"tools/call","params":{"name":"list_benchmarks","arguments":{"jq":".total"}}}`
		w := mcpRequest(t, router, body, "")

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}

		_, result := parseMCPToolResult(t, w)
		if result.IsError {
			t.Fatalf("Unexpected error: %s", result.Content[0].Text)
		}
		if result.Content[0].Text != "2" {
			t.Errorf("Expected jq to extract total=2, got %s", result.Content[0].Text)
		}
	})

	t.Run("jq extracts specific fields", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":13,"method":"tools/call","params":{"name":"list_benchmarks","arguments":{"jq":"[.benchmarks[] | .title]"}}}`
		w := mcpRequest(t, router, body, "")

		_, result := parseMCPToolResult(t, w)
		if result.IsError {
			t.Fatalf("Unexpected error: %s", result.Content[0].Text)
		}
		if !strings.Contains(result.Content[0].Text, "JQ Test Bench") {
			t.Error("Expected 'JQ Test Bench' in filtered result")
		}
		if !strings.Contains(result.Content[0].Text, "Second Bench") {
			t.Error("Expected 'Second Bench' in filtered result")
		}
	})

	t.Run("jq invalid expression returns error", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":14,"method":"tools/call","params":{"name":"list_benchmarks","arguments":{"jq":"invalid[["}}}`
		w := mcpRequest(t, router, body, "")

		_, result := parseMCPToolResult(t, w)
		if !result.IsError {
			t.Error("Expected error for invalid jq expression")
		}
		if !strings.Contains(result.Content[0].Text, "jq parse error") {
			t.Error("Expected jq parse error message")
		}
	})

	t.Run("jq empty string is ignored", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":15,"method":"tools/call","params":{"name":"list_benchmarks","arguments":{"jq":""}}}`
		w := mcpRequest(t, router, body, "")

		_, result := parseMCPToolResult(t, w)
		if result.IsError {
			t.Fatalf("Unexpected error: %s", result.Content[0].Text)
		}
		// Should return full result since jq is empty
		if !strings.Contains(result.Content[0].Text, "benchmarks") {
			t.Error("Expected full result when jq is empty")
		}
	})

	t.Run("jq works with get_benchmark", func(t *testing.T) {
		var bench Benchmark
		db.DB.Where("title = ?", "JQ Test Bench").First(&bench)

		body := fmt.Sprintf(`{"jsonrpc":"2.0","id":16,"method":"tools/call","params":{"name":"get_benchmark","arguments":{"id":%d,"jq":".title"}}}`, bench.ID)
		w := mcpRequest(t, router, body, "")

		_, result := parseMCPToolResult(t, w)
		if result.IsError {
			t.Fatalf("Unexpected error: %s", result.Content[0].Text)
		}
		if result.Content[0].Text != `"JQ Test Bench"` {
			t.Errorf("Expected jq to extract title, got %s", result.Content[0].Text)
		}
	})

	t.Run("jq arithmetic expression", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":17,"method":"tools/call","params":{"name":"list_benchmarks","arguments":{"jq":".total * 2"}}}`
		w := mcpRequest(t, router, body, "")

		_, result := parseMCPToolResult(t, w)
		if result.IsError {
			t.Fatalf("Unexpected error: %s", result.Content[0].Text)
		}
		if result.Content[0].Text != "4" {
			t.Errorf("Expected jq arithmetic result 4, got %s", result.Content[0].Text)
		}
	})
}

func TestMCPListUsersRequiresAdmin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpnonadmin", false)
	apiToken := &APIToken{UserID: user.ID, Token: "nonadmin-token-abcdef12345678", Name: "Non-Admin Token"}
	db.DB.Create(apiToken)

	body := `{"jsonrpc":"2.0","id":16,"method":"tools/call","params":{"name":"list_users","arguments":{}}}`
	w := mcpRequest(t, router, body, apiToken.Token)

	_, result := parseMCPToolResult(t, w)
	if !result.IsError {
		t.Error("Expected error for non-admin list_users")
	}
	if !strings.Contains(result.Content[0].Text, "admin privileges required") {
		t.Error("Expected admin privileges error")
	}
}

func TestMCPListUsersAsAdmin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	admin := createTestUser(db, "mcpadminuser", true)
	apiToken := &APIToken{UserID: admin.ID, Token: "admin-token-abcdef12345678901", Name: "Admin Token"}
	db.DB.Create(apiToken)

	createTestUser(db, "regularuser1", false)

	body := `{"jsonrpc":"2.0","id":17,"method":"tools/call","params":{"name":"list_users","arguments":{"page":1,"per_page":10}}}`
	w := mcpRequest(t, router, body, apiToken.Token)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "mcpadminuser") {
		t.Error("Expected admin username in result")
	}
	if !strings.Contains(result.Content[0].Text, "regularuser1") {
		t.Error("Expected regular user in result")
	}
}

const testMangoHudCSV = `os,cpu,gpu,ram,kernel,driver,cpuscheduler
TestOS,TestCPU,TestGPU,16384,5.0.0,,none
fps,frametime,cpu_load,gpu_load,cpu_temp,gpu_temp,gpu_core_clock,gpu_mem_clock,gpu_vram_used,gpu_power,ram_used,swap_used
60,16.67,50,80,65,70,1500,900,4096,200,8192,0
120,8.33,55,85,67,72,1600,950,4100,210,8200,0
90,11.11,52,82,66,71,1550,920,4080,205,8150,0`

func TestMCPDeleteUser(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	admin := createTestUser(db, "mcpdeleteadmin", true)
	adminToken := &APIToken{UserID: admin.ID, Token: "deleteuser-admin-token-abcdef1", Name: "Admin Token"}
	db.DB.Create(adminToken)

	target := createTestUser(db, "mcpdeletetarget", false)

	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":26,"method":"tools/call","params":{"name":"delete_user","arguments":{"user_id":%d}}}`, target.ID)
	w := mcpRequest(t, router, body, adminToken.Token)

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "user deleted") {
		t.Error("Expected deletion confirmation")
	}

	// Verify user is deleted
	var count int64
	db.DB.Model(&User{}).Where("id = ?", target.ID).Count(&count)
	if count != 0 {
		t.Error("Expected user to be deleted")
	}
}

func TestMCPDeleteUserSelfProtection(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	admin := createTestUser(db, "mcpselfdelete", true)
	adminToken := &APIToken{UserID: admin.ID, Token: "selfdelete-admin-token-abcdef", Name: "Admin Token"}
	db.DB.Create(adminToken)

	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":27,"method":"tools/call","params":{"name":"delete_user","arguments":{"user_id":%d}}}`, admin.ID)
	w := mcpRequest(t, router, body, adminToken.Token)

	_, result := parseMCPToolResult(t, w)
	if !result.IsError {
		t.Error("Expected error for self-deletion")
	}
	if !strings.Contains(result.Content[0].Text, "cannot delete your own account") {
		t.Error("Expected self-deletion error message")
	}
}

func TestMCPDeleteUserBenchmarks(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	admin := createTestUser(db, "mcpdelbenAdmin", true)
	adminToken := &APIToken{UserID: admin.ID, Token: "delbench-admin-token-abcdef12", Name: "Admin Token"}
	db.DB.Create(adminToken)

	target := createTestUser(db, "mcpdelbenTarget", false)
	db.DB.Create(&Benchmark{Title: "Target Bench", UserID: target.ID})

	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":28,"method":"tools/call","params":{"name":"delete_user_benchmarks","arguments":{"user_id":%d}}}`, target.ID)
	w := mcpRequest(t, router, body, adminToken.Token)

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "all user benchmarks deleted") {
		t.Error("Expected deletion confirmation")
	}

	// Verify benchmarks are deleted
	var count int64
	db.DB.Model(&Benchmark{}).Where("user_id = ?", target.ID).Count(&count)
	if count != 0 {
		t.Error("Expected benchmarks to be deleted")
	}
}

func TestMCPBanUser(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	admin := createTestUser(db, "mcpbanadmin", true)
	adminToken := &APIToken{UserID: admin.ID, Token: "banuser-admin-token-abcdef123", Name: "Admin Token"}
	db.DB.Create(adminToken)

	target := createTestUser(db, "mcpbantarget", false)

	// Ban user
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":29,"method":"tools/call","params":{"name":"ban_user","arguments":{"user_id":%d,"banned":true}}}`, target.ID)
	w := mcpRequest(t, router, body, adminToken.Token)

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}

	// Verify user is banned
	var user User
	db.DB.First(&user, target.ID)
	if !user.IsBanned {
		t.Error("Expected user to be banned")
	}

	// Unban user
	body = fmt.Sprintf(`{"jsonrpc":"2.0","id":30,"method":"tools/call","params":{"name":"ban_user","arguments":{"user_id":%d,"banned":false}}}`, target.ID)
	w = mcpRequest(t, router, body, adminToken.Token)

	_, result = parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}

	db.DB.First(&user, target.ID)
	if user.IsBanned {
		t.Error("Expected user to be unbanned")
	}
}

func TestMCPBanUserSelfProtection(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	admin := createTestUser(db, "mcpselfban", true)
	adminToken := &APIToken{UserID: admin.ID, Token: "selfban-admin-token-abcdef123", Name: "Admin Token"}
	db.DB.Create(adminToken)

	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":31,"method":"tools/call","params":{"name":"ban_user","arguments":{"user_id":%d,"banned":true}}}`, admin.ID)
	w := mcpRequest(t, router, body, adminToken.Token)

	_, result := parseMCPToolResult(t, w)
	if !result.IsError {
		t.Error("Expected error for self-ban")
	}
	if !strings.Contains(result.Content[0].Text, "cannot ban your own account") {
		t.Error("Expected self-ban error message")
	}
}

func TestMCPToggleUserAdmin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	admin := createTestUser(db, "mcptoggleadmin", true)
	adminToken := &APIToken{UserID: admin.ID, Token: "toggle-admin-token-abcdef1234", Name: "Admin Token"}
	db.DB.Create(adminToken)

	target := createTestUser(db, "mcptoggletarget", false)

	// Grant admin
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":32,"method":"tools/call","params":{"name":"toggle_user_admin","arguments":{"user_id":%d,"is_admin":true}}}`, target.ID)
	w := mcpRequest(t, router, body, adminToken.Token)

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}

	var user User
	db.DB.First(&user, target.ID)
	if !user.IsAdmin {
		t.Error("Expected user to be admin")
	}

	// Revoke admin
	body = fmt.Sprintf(`{"jsonrpc":"2.0","id":33,"method":"tools/call","params":{"name":"toggle_user_admin","arguments":{"user_id":%d,"is_admin":false}}}`, target.ID)
	w = mcpRequest(t, router, body, adminToken.Token)

	_, result = parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}

	db.DB.First(&user, target.ID)
	if user.IsAdmin {
		t.Error("Expected user to not be admin")
	}
}

func TestMCPToggleUserAdminSelfProtection(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	admin := createTestUser(db, "mcpselftoggle", true)
	adminToken := &APIToken{UserID: admin.ID, Token: "selftoggle-admin-token-abcdef", Name: "Admin Token"}
	db.DB.Create(adminToken)

	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":34,"method":"tools/call","params":{"name":"toggle_user_admin","arguments":{"user_id":%d,"is_admin":false}}}`, admin.ID)
	w := mcpRequest(t, router, body, adminToken.Token)

	_, result := parseMCPToolResult(t, w)
	if !result.IsError {
		t.Error("Expected error for self-demotion")
	}
	if !strings.Contains(result.Content[0].Text, "cannot revoke your own admin privileges") {
		t.Error("Expected self-demotion error message")
	}
}

func TestMCPAdminToolsRequireAdmin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpnonadmin3", false)
	apiToken := &APIToken{UserID: user.ID, Token: "nonadmin3-token-abcdef1234567", Name: "Non-Admin Token"}
	db.DB.Create(apiToken)

	adminTools := []string{"delete_user", "delete_user_benchmarks", "ban_user", "toggle_user_admin"}
	for _, tool := range adminTools {
		body := fmt.Sprintf(`{"jsonrpc":"2.0","id":35,"method":"tools/call","params":{"name":"%s","arguments":{"user_id":1}}}`, tool)
		w := mcpRequest(t, router, body, apiToken.Token)

		_, result := parseMCPToolResult(t, w)
		if !result.IsError {
			t.Errorf("Expected error for non-admin %s", tool)
		}
		if !strings.Contains(result.Content[0].Text, "admin privileges required") {
			t.Errorf("Expected admin privileges error for %s", tool)
		}
	}
}

// mcpCreateBenchmarkHelper creates a benchmark directly via the storage API and returns its ID.
// This bypasses MCP tools (which don't include data upload) and creates benchmark data directly.
func mcpCreateBenchmarkHelper(t *testing.T, db *DBInstance, userID uint) int {
	t.Helper()
	data, err := ReadBenchmarkCSVContent(testMangoHudCSV, "Run 1")
	if err != nil {
		t.Fatalf("Failed to parse test CSV: %v", err)
	}

	benchmark := Benchmark{
		UserID: userID,
		Title:  "Helper Bench",
	}
	if err := db.DB.Create(&benchmark).Error; err != nil {
		t.Fatalf("Failed to create benchmark: %v", err)
	}

	benchmarkData := []*BenchmarkData{data}
	if err := StoreBenchmarkData(benchmarkData, benchmark.ID); err != nil {
		t.Fatalf("Failed to store benchmark data: %v", err)
	}

	// Also store pre-calculated stats (required by new pre-calculated API)
	preCalc := ComputePreCalculatedRuns(benchmarkData)
	if err := StorePreCalculatedStats(preCalc, benchmark.ID); err != nil {
		t.Fatalf("Failed to store pre-calculated stats: %v", err)
	}

	return int(benchmark.ID)
}

func TestMCPGetBenchmarkData(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	if err := InitBenchmarksDir(t.TempDir()); err != nil {
		t.Fatalf("Failed to init benchmarks dir: %v", err)
	}
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpgetdata", false)
	apiToken := &APIToken{UserID: user.ID, Token: "getdata-token-abcdef1234567890", Name: "GetData Token"}
	db.DB.Create(apiToken)

	benchID := mcpCreateBenchmarkHelper(t, db, user.ID)

	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":40,"method":"tools/call","params":{"name":"get_benchmark_data","arguments":{"id":%d,"max_points":100}}}`, benchID)
	w := mcpRequest(t, router, body, "")

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "fps") {
		t.Error("Expected fps metric in result")
	}
	if !strings.Contains(result.Content[0].Text, "frame_time") {
		t.Error("Expected frame_time metric in result")
	}
	if !strings.Contains(result.Content[0].Text, "total_data_points") {
		t.Error("Expected total_data_points in result")
	}
	// Verify extended percentile fields are present
	for _, field := range []string{`"p05"`, `"p10"`, `"p25"`, `"p75"`, `"p90"`, `"p95"`, `"p99"`, `"iqr"`} {
		if !strings.Contains(result.Content[0].Text, field) {
			t.Errorf("Expected %s in result", field)
		}
	}
}

func TestMCPGetBenchmarkRun(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	if err := InitBenchmarksDir(t.TempDir()); err != nil {
		t.Fatalf("Failed to init benchmarks dir: %v", err)
	}
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpgetrun", false)
	apiToken := &APIToken{UserID: user.ID, Token: "getrun-token-abcdef12345678901", Name: "GetRun Token"}
	db.DB.Create(apiToken)

	benchID := mcpCreateBenchmarkHelper(t, db, user.ID)

	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":41,"method":"tools/call","params":{"name":"get_benchmark_run","arguments":{"id":%d,"run_index":0,"max_points":50}}}`, benchID)
	w := mcpRequest(t, router, body, "")

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "Run 1") {
		t.Error("Expected run label in result")
	}
	if !strings.Contains(result.Content[0].Text, "fps") {
		t.Error("Expected fps metric in result")
	}
	// Verify extended percentile fields are present
	for _, field := range []string{`"p05"`, `"p10"`, `"p25"`, `"p75"`, `"p90"`, `"p95"`, `"p99"`, `"iqr"`} {
		if !strings.Contains(result.Content[0].Text, field) {
			t.Errorf("Expected %s in result", field)
		}
	}
}

func TestMCPJQFilterWithBenchmarkData(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	if err := InitBenchmarksDir(t.TempDir()); err != nil {
		t.Fatalf("Failed to init benchmarks dir: %v", err)
	}
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpjqdata", false)
	benchID := mcpCreateBenchmarkHelper(t, db, user.ID)

	t.Run("jq extracts fps stats from benchmark data", func(t *testing.T) {
		body := fmt.Sprintf(`{"jsonrpc":"2.0","id":50,"method":"tools/call","params":{"name":"get_benchmark_data","arguments":{"id":%d,"jq":".runs[0].metrics.fps | {avg, min, max}"}}}`, benchID)
		w := mcpRequest(t, router, body, "")

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		_, result := parseMCPToolResult(t, w)
		if result.IsError {
			t.Fatalf("Unexpected error: %s", result.Content[0].Text)
		}
		// Should contain only the filtered fields
		if !strings.Contains(result.Content[0].Text, "avg") {
			t.Error("Expected 'avg' in filtered result")
		}
		if !strings.Contains(result.Content[0].Text, "min") {
			t.Error("Expected 'min' in filtered result")
		}
		if !strings.Contains(result.Content[0].Text, "max") {
			t.Error("Expected 'max' in filtered result")
		}
		// Should NOT contain full benchmark metadata (jq filtered it out)
		if strings.Contains(result.Content[0].Text, "benchmark") {
			t.Error("Did not expect 'benchmark' key in jq-filtered result")
		}
	})

	t.Run("jq extracts benchmark title from data response", func(t *testing.T) {
		db.DB.Model(&Benchmark{}).Where("id = ?", benchID).Update("title", "Helper Bench")
		body := fmt.Sprintf(`{"jsonrpc":"2.0","id":51,"method":"tools/call","params":{"name":"get_benchmark_data","arguments":{"id":%d,"jq":".benchmark.title"}}}`, benchID)
		w := mcpRequest(t, router, body, "")

		_, result := parseMCPToolResult(t, w)
		if result.IsError {
			t.Fatalf("Unexpected error: %s", result.Content[0].Text)
		}
		if result.Content[0].Text != `"Helper Bench"` {
			t.Errorf("Expected '\"Helper Bench\"', got %s", result.Content[0].Text)
		}
	})

	t.Run("jq works with get_benchmark_run", func(t *testing.T) {
		body := fmt.Sprintf(`{"jsonrpc":"2.0","id":52,"method":"tools/call","params":{"name":"get_benchmark_run","arguments":{"id":%d,"run_index":0,"jq":".label"}}}`, benchID)
		w := mcpRequest(t, router, body, "")

		_, result := parseMCPToolResult(t, w)
		if result.IsError {
			t.Fatalf("Unexpected error: %s", result.Content[0].Text)
		}
		if result.Content[0].Text != `"Run 1"` {
			t.Errorf("Expected '\"Run 1\"', got %s", result.Content[0].Text)
		}
	})
}

func TestMCPToolsHaveJQParameter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	server := newMCPServer(db, "test")

	for _, tool := range server.tools {
		t.Run(tool.Name, func(t *testing.T) {
			schema, ok := tool.InputSchema.(map[string]interface{})
			if !ok {
				t.Fatal("InputSchema is not a map")
			}
			props, ok := schema["properties"].(map[string]interface{})
			if !ok {
				t.Fatal("properties is not a map")
			}
			jqProp, ok := props["jq"]
			if !ok {
				t.Fatal("Tool must have a 'jq' parameter")
			}
			jqMap, ok := jqProp.(map[string]interface{})
			if !ok {
				t.Fatal("jq property must be a map")
			}
			if jqMap["type"] != "string" {
				t.Errorf("jq type must be 'string', got %v", jqMap["type"])
			}
		})
	}
}

func TestMCPCorsHeaders(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	t.Run("OPTIONS preflight returns 204 with CORS headers", func(t *testing.T) {
		req, err := http.NewRequest("OPTIONS", "/mcp", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Origin", "http://localhost:6274")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected 204, got %d", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin '*', got '%s'", got)
		}
		if got := w.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, DELETE, OPTIONS" {
			t.Errorf("Expected Access-Control-Allow-Methods 'GET, POST, DELETE, OPTIONS', got '%s'", got)
		}
		if got := w.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization" {
			t.Errorf("Expected Access-Control-Allow-Headers 'Content-Type, Authorization', got '%s'", got)
		}
	})

	t.Run("POST response includes CORS headers", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"method":"ping"}`
		w := mcpRequest(t, router, body, "")

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin '*', got '%s'", got)
		}
	})

	t.Run("GET response includes CORS headers", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/mcp", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin '*', got '%s'", got)
		}
	})

	t.Run("DELETE response includes CORS headers", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", "/mcp", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin '*', got '%s'", got)
		}
	})
}
