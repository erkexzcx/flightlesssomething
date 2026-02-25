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
	r := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("test_session", store))
	r.POST("/mcp", HandleMCP(db, "test"))
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

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
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
	if result.ProtocolVersion != "2025-03-26" {
		t.Errorf("Expected protocol version 2025-03-26, got %s", result.ProtocolVersion)
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

	body := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
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

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}
	var result mcpToolsListResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	expectedTools := []string{
		"list_benchmarks", "get_benchmark", "get_benchmark_data",
		"get_benchmark_run", "update_benchmark", "delete_benchmark",
		"delete_benchmark_run", "get_current_user", "list_api_tokens",
		"create_api_token", "delete_api_token", "list_users",
		"list_audit_logs",
	}
	if len(result.Tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(result.Tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}
	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("Missing tool: %s", name)
		}
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

	body := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"delete_benchmark","arguments":{"id":` + idStr(b.ID) + `}}}`
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

	body := `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"delete_benchmark","arguments":{"id":` + idStr(b.ID) + `}}}`
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

func TestDownsampleSlice(t *testing.T) {
	small := []float64{1.0, 2.0, 3.0}
	result := downsampleSlice(small, 10)
	if len(result) != 3 {
		t.Errorf("Expected 3 points, got %d", len(result))
	}

	large := make([]float64, 1000)
	for i := range large {
		large[i] = float64(i)
	}
	result = downsampleSlice(large, 10)
	if len(result) != 10 {
		t.Errorf("Expected 10 points, got %d", len(result))
	}
	if result[0] != 0 {
		t.Errorf("Expected first point 0, got %f", result[0])
	}
	if result[9] != 999 {
		t.Errorf("Expected last point 999, got %f", result[9])
	}
}

func TestPercentile(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	p50 := percentile(data, 50)
	if p50 != 5.5 {
		t.Errorf("Expected p50=5.5, got %f", p50)
	}

	p0 := percentile(data, 0)
	if p0 != 1 {
		t.Errorf("Expected p0=1, got %f", p0)
	}

	p100 := percentile(data, 100)
	if p100 != 10 {
		t.Errorf("Expected p100=10, got %f", p100)
	}
}

func TestComputeMetricSummary(t *testing.T) {
	data := []float64{10, 20, 30, 40, 50}

	summary := computeMetricSummary(data, 3)
	if summary.Min != 10 {
		t.Errorf("Expected min=10, got %f", summary.Min)
	}
	if summary.Max != 50 {
		t.Errorf("Expected max=50, got %f", summary.Max)
	}
	if summary.Avg != 30 {
		t.Errorf("Expected avg=30, got %f", summary.Avg)
	}
	if summary.Count != 5 {
		t.Errorf("Expected count=5, got %d", summary.Count)
	}
	if summary.Variance == 0 {
		t.Error("Expected non-zero variance")
	}
	if summary.StdDev == 0 {
		t.Error("Expected non-zero std_dev")
	}
	// Sample variance (n-1): sum((x-30)^2) / 4 = (400+100+0+100+400)/4 = 250
	if summary.Variance != 250 {
		t.Errorf("Expected variance=250 (sample), got %f", summary.Variance)
	}

	statsOnly := computeMetricSummary(data, 0)
	if statsOnly.Data != nil {
		t.Error("Expected no data with max_points=0")
	}
	if statsOnly.Variance != 250 {
		t.Error("Stats should still be computed even with max_points=0")
	}
}

func TestSummarizeBenchmarkData(t *testing.T) {
	run := &BenchmarkData{
		Label:         "Test Run",
		SpecOS:        "Linux",
		SpecCPU:       "AMD Ryzen 7",
		SpecGPU:       "RTX 3080",
		SpecRAM:       "32GB",
		DataFPS:       []float64{60, 120, 90, 144, 30},
		DataFrameTime: []float64{16.67, 8.33, 11.11, 6.94, 33.33},
	}

	// Default: stats only (maxPoints=0)
	summary := summarizeBenchmarkData(run, 0)
	if summary.Label != "Test Run" {
		t.Errorf("Expected label 'Test Run', got '%s'", summary.Label)
	}
	if summary.TotalDataPoints != 5 {
		t.Errorf("Expected 5 total points, got %d", summary.TotalDataPoints)
	}
	if summary.Metrics["fps"] == nil {
		t.Fatal("Expected fps metric in summary")
	}
	if summary.Metrics["fps"].Count != 5 {
		t.Errorf("Expected count 5, got %d", summary.Metrics["fps"].Count)
	}
	// FPS stats should be derived from frametime
	if summary.Metrics["fps"].Data != nil {
		t.Error("Expected no data points with maxPoints=0")
	}
	if summary.Metrics["fps"].Variance == 0 {
		t.Error("Expected non-zero FPS variance")
	}
	if summary.Metrics["frame_time"] == nil {
		t.Fatal("Expected frame_time metric in summary")
	}
	if summary.Metrics["frame_time"].Variance == 0 {
		t.Error("Expected non-zero frametime variance")
	}
}

func TestComputeFPSFromFrametime(t *testing.T) {
	// 10ms frametime = 100 FPS, 20ms = 50 FPS
	ft := []float64{10, 20, 10, 20, 10}
	fps := []float64{100, 50, 100, 50, 100}

	summary := computeFPSFromFrametime(ft, fps, 0)
	if summary == nil {
		t.Fatal("Expected non-nil summary")
	}
	// Min FPS should come from max frametime (20ms → 50 FPS)
	if summary.Min != 50 {
		t.Errorf("Expected min FPS=50, got %f", summary.Min)
	}
	// Max FPS should come from min frametime (10ms → 100 FPS)
	if summary.Max != 100 {
		t.Errorf("Expected max FPS=100, got %f", summary.Max)
	}
	if summary.Variance == 0 {
		t.Error("Expected non-zero variance")
	}
	if summary.Data != nil {
		t.Error("Expected no data with maxPoints=0")
	}
}

func idStr(id uint) string {
	return fmt.Sprintf("%d", id)
}

func TestMCPGetCurrentUser(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpcurrentuser", false)
	apiToken := &APIToken{UserID: user.ID, Token: "current-user-token-abcdef123456", Name: "Current User Token"}
	db.DB.Create(apiToken)

	body := `{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"get_current_user","arguments":{}}}`
	w := mcpRequest(t, router, body, apiToken.Token)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "mcpcurrentuser") {
		t.Error("Expected username in result")
	}
	if !strings.Contains(result.Content[0].Text, fmt.Sprintf("\"user_id\":%d", user.ID)) {
		t.Error("Expected user_id in result")
	}
}

func TestMCPGetCurrentUserRequiresAuth(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	body := `{"jsonrpc":"2.0","id":11,"method":"tools/call","params":{"name":"get_current_user","arguments":{}}}`
	w := mcpRequest(t, router, body, "")

	_, result := parseMCPToolResult(t, w)
	if !result.IsError {
		t.Error("Expected error for unauthenticated get_current_user")
	}
	if !strings.Contains(result.Content[0].Text, "authentication required") {
		t.Error("Expected authentication error message")
	}
}

func TestMCPListAPITokens(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcptokenuser", false)
	apiToken := &APIToken{UserID: user.ID, Token: "list-tokens-abcdef1234567890ab", Name: "List Token"}
	db.DB.Create(apiToken)
	apiToken2 := &APIToken{UserID: user.ID, Token: "second-token-abcdef1234567890a", Name: "Second Token"}
	db.DB.Create(apiToken2)

	body := `{"jsonrpc":"2.0","id":12,"method":"tools/call","params":{"name":"list_api_tokens","arguments":{}}}`
	w := mcpRequest(t, router, body, apiToken.Token)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "List Token") {
		t.Error("Expected first token name in result")
	}
	if !strings.Contains(result.Content[0].Text, "Second Token") {
		t.Error("Expected second token name in result")
	}
}

func TestMCPCreateAPIToken(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpcreatetoken", false)
	apiToken := &APIToken{UserID: user.ID, Token: "create-token-abcdef1234567890a", Name: "Auth Token"}
	db.DB.Create(apiToken)

	body := `{"jsonrpc":"2.0","id":13,"method":"tools/call","params":{"name":"create_api_token","arguments":{"name":"New MCP Token"}}}`
	w := mcpRequest(t, router, body, apiToken.Token)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "New MCP Token") {
		t.Error("Expected new token name in result")
	}

	// Verify token was created in DB
	var count int64
	db.DB.Model(&APIToken{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 2 { // original + new
		t.Errorf("Expected 2 tokens, got %d", count)
	}
}

func TestMCPDeleteAPIToken(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpdeletetoken", false)
	apiToken := &APIToken{UserID: user.ID, Token: "delete-token-abcdef1234567890a", Name: "Auth Token"}
	db.DB.Create(apiToken)
	targetToken := &APIToken{UserID: user.ID, Token: "target-token-abcdef1234567890a", Name: "To Delete"}
	db.DB.Create(targetToken)

	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":14,"method":"tools/call","params":{"name":"delete_api_token","arguments":{"token_id":%d}}}`, targetToken.ID)
	w := mcpRequest(t, router, body, apiToken.Token)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "token deleted") {
		t.Error("Expected deletion confirmation")
	}

	// Verify token was deleted
	var count int64
	db.DB.Model(&APIToken{}).Where("id = ?", targetToken.ID).Count(&count)
	if count != 0 {
		t.Error("Expected token to be deleted")
	}
}

func TestMCPDeleteAPITokenOwnership(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user1 := createTestUser(db, "mcptokenowner1", false)
	user2 := createTestUser(db, "mcptokenowner2", false)

	token1 := &APIToken{UserID: user1.ID, Token: "owner1-token-abcdef1234567890a", Name: "User1 Token"}
	db.DB.Create(token1)
	token2 := &APIToken{UserID: user2.ID, Token: "owner2-token-abcdef1234567890a", Name: "User2 Token"}
	db.DB.Create(token2)

	// User1 tries to delete User2's token
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":15,"method":"tools/call","params":{"name":"delete_api_token","arguments":{"token_id":%d}}}`, token2.ID)
	w := mcpRequest(t, router, body, token1.Token)

	_, result := parseMCPToolResult(t, w)
	if !result.IsError {
		t.Error("Expected error for deleting another user's token")
	}
	if !strings.Contains(result.Content[0].Text, "token not found") {
		t.Error("Expected token not found error")
	}
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

func TestMCPListAuditLogsRequiresAdmin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	user := createTestUser(db, "mcpnonadmin2", false)
	apiToken := &APIToken{UserID: user.ID, Token: "nonadmin2-token-abcdef1234567", Name: "Non-Admin Token"}
	db.DB.Create(apiToken)

	body := `{"jsonrpc":"2.0","id":18,"method":"tools/call","params":{"name":"list_audit_logs","arguments":{}}}`
	w := mcpRequest(t, router, body, apiToken.Token)

	_, result := parseMCPToolResult(t, w)
	if !result.IsError {
		t.Error("Expected error for non-admin list_audit_logs")
	}
	if !strings.Contains(result.Content[0].Text, "admin privileges required") {
		t.Error("Expected admin privileges error")
	}
}

func TestMCPListAuditLogsAsAdmin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	router := setupMCPTestRouter(db)

	admin := createTestUser(db, "mcpadmin2", true)
	apiToken := &APIToken{UserID: admin.ID, Token: "admin2-token-abcdef1234567890", Name: "Admin Token"}
	db.DB.Create(apiToken)

	// Create a benchmark to generate an audit log entry
	b := &Benchmark{Title: "Audit Test Bench", UserID: admin.ID}
	db.DB.Create(b)
	LogBenchmarkCreated(db, admin.ID, b.ID, b.Title)

	body := `{"jsonrpc":"2.0","id":19,"method":"tools/call","params":{"name":"list_audit_logs","arguments":{"page":1,"per_page":10}}}`
	w := mcpRequest(t, router, body, apiToken.Token)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	_, result := parseMCPToolResult(t, w)
	if result.IsError {
		t.Fatalf("Unexpected error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "Benchmark Created") {
		t.Error("Expected audit log entry in result")
	}
}
