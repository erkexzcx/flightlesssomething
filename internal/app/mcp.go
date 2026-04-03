package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/itchyny/gojq"
)

// JSON-RPC 2.0 types
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSON-RPC error codes
const (
	jsonrpcParseError     = -32700
	jsonrpcInvalidRequest = -32600
	jsonrpcMethodNotFound = -32601
	jsonrpcInvalidParams  = -32602
)

// MCP protocol types
type mcpInitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    mcpCapabilities `json:"capabilities"`
	ServerInfo      mcpServerInfo   `json:"serverInfo"`
	Instructions    string          `json:"instructions,omitempty"`
}

type mcpCapabilities struct {
	Tools *mcpToolsCapability `json:"tools,omitempty"`
}

type mcpToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type mcpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool access levels for filtering tools/list by caller's auth level
const (
	toolAccessPublic = "public" // No auth required
	toolAccessAuth   = "auth"   // Requires authentication
	toolAccessAdmin  = "admin"  // Requires admin privileges
)

type mcpToolAnnotations struct {
	ReadOnlyHint    *bool `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool `json:"openWorldHint,omitempty"`
}

type mcpIcon struct {
	Src      string   `json:"src"`
	MIMEType string   `json:"mimeType,omitempty"`
	Sizes    []string `json:"sizes,omitempty"`
	Theme    string   `json:"theme,omitempty"`
}

type mcpTool struct {
	Name        string              `json:"name"`
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description"`
	InputSchema interface{}         `json:"inputSchema"`
	Icons       []mcpIcon           `json:"icons,omitempty"`
	Annotations *mcpToolAnnotations `json:"annotations,omitempty"`
	accessLevel string              // internal: "public", "auth", "admin" — not serialized
}

type mcpToolsListResult struct {
	Tools []mcpTool `json:"tools"`
}

type mcpToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type mcpToolResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Data downsampling types
const (
	maxMaxPoints = 5000
)

// MetricSummary holds computed statistics for a metric, matching the web frontend calculations.
// Stats are always provided. Raw data points are optional (opt-in via max_points > 0).
type MetricSummary struct {
	Min      float64   `json:"min"`
	Max      float64   `json:"max"`
	Avg      float64   `json:"avg"`
	Median   float64   `json:"median"`
	P01      float64   `json:"p01"`
	P05      float64   `json:"p05"`
	P10      float64   `json:"p10"`
	P25      float64   `json:"p25"`
	P75      float64   `json:"p75"`
	P90      float64   `json:"p90"`
	P95      float64   `json:"p95"`
	P97      float64   `json:"p97"`
	P99      float64   `json:"p99"`
	IQR      float64   `json:"iqr"`
	StdDev   float64   `json:"std_dev"`
	Variance float64   `json:"variance"`
	Count    int       `json:"count"`
	Data     []float64 `json:"data,omitempty"`
}

// BenchmarkDataSummary holds computed stats per metric for a benchmark run.
// This is the primary response format — stats are always computed from full data.
type BenchmarkDataSummary struct {
	Label              string                    `json:"label"`
	SpecOS             string                    `json:"spec_os"`
	SpecCPU            string                    `json:"spec_cpu"`
	SpecGPU            string                    `json:"spec_gpu"`
	SpecRAM            string                    `json:"spec_ram"`
	SpecLinuxKernel    string                    `json:"spec_linux_kernel,omitempty"`
	SpecLinuxScheduler string                    `json:"spec_linux_scheduler,omitempty"`
	TotalDataPoints    int                       `json:"total_data_points"`
	DownsampledTo      int                       `json:"downsampled_to,omitempty"`
	Metrics            map[string]*MetricSummary `json:"metrics"`
}

// mcpServer holds the MCP server state
type mcpServer struct {
	db      *DBInstance
	version string
	tools   []mcpTool
}

func newMCPServer(db *DBInstance, version string) *mcpServer {
	s := &mcpServer{db: db, version: version}
	s.tools = s.defineTools()
	return s
}

// jqProperty is the common JSON schema property for the optional jq filter parameter.
// It is added to every tool's InputSchema.
var jqProperty = map[string]interface{}{
	"type":        "string",
	"description": "Optional jq expression to filter/transform the JSON result server-side. Reduces response size and avoids unnecessary context usage. Example: '.benchmarks[] | {id, title}' or '.runs[0].metrics.fps | {avg, p01, p99}'",
}

func (s *mcpServer) defineTools() []mcpTool {
	boolPtr := func(b bool) *bool { return &b }

	// Font Awesome 6 Free solid SVG icons via jsDelivr CDN
	faIcon := func(name string) []mcpIcon {
		return []mcpIcon{{
			Src:      "https://cdn.jsdelivr.net/npm/@fortawesome/fontawesome-free@6.7.2/svgs/solid/" + name + ".svg",
			MIMEType: "image/svg+xml",
			Sizes:    []string{"any"},
		}}
	}

	return []mcpTool{
		{
			Name:        "list_benchmarks",
			Title:       "List Benchmarks",
			Description: "Search and list gaming benchmarks with pagination, search, and sorting. Returns benchmark metadata including title, description (markdown), user, run count, and timestamps. After listing, use get_benchmark_data to retrieve statistics for analysis.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page":       map[string]interface{}{"type": "integer", "description": "Page number (default: 1)"},
					"per_page":   map[string]interface{}{"type": "integer", "description": "Results per page, 1-100 (default: 10)"},
					"search":     map[string]interface{}{"type": "string", "description": "Search keywords (space-separated, AND logic). Searches title, description, username, run names, and specifications."},
					"user_id":    map[string]interface{}{"type": "integer", "description": "Filter by user ID"},
					"username":   map[string]interface{}{"type": "string", "description": "Filter by exact username (case-insensitive). Use this instead of user_id when you know the username but not the ID."},
					"sort_by":    map[string]interface{}{"type": "string", "enum": []string{"title", "created_at", "updated_at"}, "description": "Sort field (default: created_at)"},
					"sort_order": map[string]interface{}{"type": "string", "enum": []string{"asc", "desc"}, "description": "Sort order (default: desc)"},
					"jq":         jqProperty,
				},
			},
			Icons:       faIcon("magnifying-glass"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessPublic,
		},
		{
			Name:        "get_benchmark",
			Title:       "Get Benchmark Details",
			Description: "Get detailed information about a specific benchmark including title, description (markdown formatted), user, run count, run labels, and timestamps. Note: get_benchmark_data also includes this metadata alongside statistics, so prefer get_benchmark_data when you need both metadata and stats.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "integer", "description": "Benchmark ID"},
					"jq": jqProperty,
				},
			},
			Icons:       faIcon("eye"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessPublic,
		},
		{
			Name:        "get_benchmark_data",
			Title:       "Get Benchmark Statistics",
			Description: "Get benchmark metadata and computed statistics for all runs in a single call. Returns the benchmark info (title, description, user, timestamps) alongside per-metric stats: min, max, avg, median, p01, p05, p10, p25, p75, p90, p95, p97, p99, iqr, std_dev, variance, count. FPS stats are correctly derived from frametime data. Raw data points are omitted by default; set max_points > 0 to include downsampled time series. This is the primary tool for benchmark analysis — no need to call get_benchmark separately.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id":         map[string]interface{}{"type": "integer", "description": "Benchmark ID"},
					"max_points": map[string]interface{}{"type": "integer", "description": "Include downsampled raw data points (default: 0 = stats only). Set 1-5000 for time series data alongside stats."},
					"jq":         jqProperty,
				},
			},
			Icons:       faIcon("chart-simple"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessPublic,
		},
		{
			Name:        "get_benchmark_run",
			Title:       "Get Run Statistics",
			Description: "Get computed statistics for a specific run within a benchmark. Same stats as get_benchmark_data but for a single run. Raw data points omitted by default.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "run_index"},
				"properties": map[string]interface{}{
					"id":         map[string]interface{}{"type": "integer", "description": "Benchmark ID"},
					"run_index":  map[string]interface{}{"type": "integer", "description": "Run index (0-based)"},
					"max_points": map[string]interface{}{"type": "integer", "description": "Include downsampled raw data points (default: 0 = stats only). Set 1-5000 for time series data alongside stats."},
					"jq":         jqProperty,
				},
			},
			Icons:       faIcon("chart-line"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessPublic,
		},
		{
			Name:        "update_benchmark",
			Title:       "Update Benchmark Metadata",
			Description: "Update benchmark metadata (title, description) and/or run labels. Description supports markdown formatting. Requires authentication via API token. Only the benchmark owner or an admin can update.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id":          map[string]interface{}{"type": "integer", "description": "Benchmark ID"},
					"title":       map[string]interface{}{"type": "string", "description": "New title (max 100 chars)"},
					"description": map[string]interface{}{"type": "string", "description": "New description in markdown format (max 5000 chars)"},
					"labels":      map[string]interface{}{"type": "object", "description": "Map of run index (as string) to new label, e.g. {\"0\": \"Run A\", \"1\": \"Run B\"}", "additionalProperties": map[string]interface{}{"type": "string"}},
					"jq":          jqProperty,
				},
			},
			Icons:       faIcon("pen"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(false), IdempotentHint: boolPtr(true), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAuth,
		},
		{
			Name:        "list_users",
			Title:       "List Users",
			Description: "List all users with pagination and optional search. Admin only. Requires authentication via API token with admin privileges.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page":     map[string]interface{}{"type": "integer", "description": "Page number (default: 1)"},
					"per_page": map[string]interface{}{"type": "integer", "description": "Results per page, 1-100 (default: 10)"},
					"search":   map[string]interface{}{"type": "string", "description": "Search by username or Discord ID"},
					"jq":       jqProperty,
				},
			},
			Icons:       faIcon("users"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAdmin,
		},
		{
			Name:        "delete_user",
			Title:       "Delete User Account",
			Description: "Delete a user account. Admin only. Cannot delete your own account. Optionally delete all user data (benchmarks). Requires authentication via API token with admin privileges.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"user_id"},
				"properties": map[string]interface{}{
					"user_id":     map[string]interface{}{"type": "integer", "description": "User ID to delete"},
					"delete_data": map[string]interface{}{"type": "boolean", "description": "Also delete all benchmark data files (default: false)"},
					"jq":          jqProperty,
				},
			},
			Icons:       faIcon("user-xmark"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(true), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAdmin,
		},
		{
			Name:        "delete_user_benchmarks",
			Title:       "Delete User Benchmarks",
			Description: "Delete all benchmarks belonging to a user. Admin only. Requires authentication via API token with admin privileges.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"user_id"},
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{"type": "integer", "description": "User ID whose benchmarks to delete"},
					"jq":      jqProperty,
				},
			},
			Icons:       faIcon("folder-minus"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(true), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAdmin,
		},
		{
			Name:        "ban_user",
			Title:       "Ban or Unban User",
			Description: "Ban or unban a user. Admin only. Cannot ban your own account. Requires authentication via API token with admin privileges.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"user_id", "banned"},
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{"type": "integer", "description": "User ID to ban/unban"},
					"banned":  map[string]interface{}{"type": "boolean", "description": "true to ban, false to unban"},
					"jq":      jqProperty,
				},
			},
			Icons:       faIcon("ban"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(false), IdempotentHint: boolPtr(true), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAdmin,
		},
		{
			Name:        "toggle_user_admin",
			Title:       "Toggle Admin Privileges",
			Description: "Grant or revoke admin privileges for a user. Admin only. Cannot revoke your own admin privileges. Requires authentication via API token with admin privileges.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"user_id", "is_admin"},
				"properties": map[string]interface{}{
					"user_id":  map[string]interface{}{"type": "integer", "description": "User ID to modify"},
					"is_admin": map[string]interface{}{"type": "boolean", "description": "true to grant admin, false to revoke"},
					"jq":       jqProperty,
				},
			},
			Icons:       faIcon("shield-halved"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(false), IdempotentHint: boolPtr(true), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAdmin,
		},
	}
}

// HandleMCP handles MCP JSON-RPC requests via POST
func HandleMCP(db *DBInstance, version string) gin.HandlerFunc {
	server := newMCPServer(db, version)
	return func(c *gin.Context) {
		// Validate Content-Type
		contentType := c.GetHeader("Content-Type")
		if !strings.HasPrefix(contentType, "application/json") {
			c.JSON(http.StatusBadRequest, jsonrpcResponse{
				JSONRPC: "2.0",
				Error:   &jsonrpcError{Code: jsonrpcParseError, Message: "Content-Type must be application/json"},
			})
			return
		}

		// Parse JSON-RPC request
		var req jsonrpcRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusOK, jsonrpcResponse{
				JSONRPC: "2.0",
				Error:   &jsonrpcError{Code: jsonrpcParseError, Message: "failed to parse request"},
			})
			return
		}

		if req.JSONRPC != "2.0" {
			c.JSON(http.StatusOK, jsonrpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &jsonrpcError{Code: jsonrpcInvalidRequest, Message: "jsonrpc must be \"2.0\""},
			})
			return
		}

		// Handle notifications (no ID) - respond with 202 Accepted
		if req.ID == nil || string(req.ID) == "null" {
			if req.Method == "notifications/initialized" {
				c.Status(http.StatusAccepted)
				return
			}
			c.Status(http.StatusAccepted)
			return
		}

		// Handle methods
		var resp jsonrpcResponse
		switch req.Method {
		case "initialize":
			resp = server.handleInitialize(c, &req)
		case "tools/list":
			resp = server.handleToolsList(c, &req)
		case "tools/call":
			resp = server.handleToolsCall(c, &req)
		case "ping":
			resp = jsonrpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]interface{}{}}
		default:
			resp = jsonrpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &jsonrpcError{Code: jsonrpcMethodNotFound, Message: fmt.Sprintf("method not found: %s", req.Method)},
			}
		}

		c.JSON(http.StatusOK, resp)
	}
}

// MCPCors returns a middleware that adds CORS headers for the MCP endpoint.
// This is required for browser-based MCP clients (e.g. MCP Inspector) that
// send preflight OPTIONS requests from a different origin.
// Wildcard origin is acceptable because MCP endpoints use Bearer token
// authentication, and browser-based clients run on arbitrary localhost ports.
func MCPCors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// HandleMCPGet handles GET requests for server-sent events (SSE stream)
func HandleMCPGet(c *gin.Context) {
	// For stateless implementation, GET is not used for SSE
	c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "SSE not supported, use POST for JSON-RPC requests"})
}

// HandleMCPDelete handles session termination
func HandleMCPDelete(c *gin.Context) {
	// Stateless server - no sessions to terminate
	c.JSON(http.StatusOK, gin.H{"message": "session terminated"})
}

func (s *mcpServer) handleInitialize(c *gin.Context, req *jsonrpcRequest) jsonrpcResponse {
	// Build base URL from request
	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	baseURL := scheme + "://" + c.Request.Host

	// Build instructions with context
	instructions := `FlightlessSomething is a gaming benchmark storage service. You can browse, search, and analyze benchmarks using the provided tools. Benchmark descriptions are markdown formatted. When asked about a benchmark, use get_benchmark_data to retrieve its metadata and performance statistics in a single call — do not call get_benchmark separately unless you only need metadata without statistics. The list_benchmarks tool supports filtering by username directly, so you don't need to resolve user IDs first. This MCP server does not support benchmark data upload, download, or deletion operations — these involve large CSV file transfers which are not suitable for the MCP protocol. Use the web UI at ` + baseURL + ` for uploading, downloading, or deleting benchmarks. API tokens can be managed via the web UI at ` + baseURL + `/api-tokens. All tools support an optional "jq" parameter for server-side filtering and transformation of results, reducing response size.

Server base URL: ` + baseURL

	// Add authenticated user context or anonymous mode notice
	userID, _, isAdmin, authenticated := s.authenticateFromHeader(c)
	if authenticated {
		var user User
		if err := s.db.DB.First(&user, userID).Error; err == nil {
			instructions += fmt.Sprintf("\n\nAuthenticated user context:\n- User ID: %d\n- Username: %s\n- Admin: %v", user.ID, user.Username, isAdmin)
		}
	} else {
		instructions += "\n\nAnonymous mode: No API token provided. Only read-only operations (browsing and viewing benchmarks) are available. To access write operations, provide an API token via the Authorization header."
	}

	return jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: mcpInitializeResult{
			ProtocolVersion: "2025-11-25",
			Capabilities: mcpCapabilities{
				Tools: &mcpToolsCapability{ListChanged: false},
			},
			ServerInfo: mcpServerInfo{
				Name:    "FlightlessSomething",
				Version: s.version,
			},
			Instructions: instructions,
		},
	}
}

func (s *mcpServer) handleToolsList(c *gin.Context, req *jsonrpcRequest) jsonrpcResponse {
	_, _, isAdmin, authenticated := s.authenticateFromHeader(c)

	filtered := make([]mcpTool, 0, len(s.tools))
	for _, tool := range s.tools {
		switch tool.accessLevel {
		case toolAccessAdmin:
			if !isAdmin {
				continue
			}
		case toolAccessAuth:
			if !authenticated {
				continue
			}
		}
		filtered = append(filtered, tool)
	}

	return jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  mcpToolsListResult{Tools: filtered},
	}
}

func (s *mcpServer) handleToolsCall(c *gin.Context, req *jsonrpcRequest) jsonrpcResponse {
	var params mcpToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonrpcError{Code: jsonrpcInvalidParams, Message: "invalid tool call params"},
		}
	}

	// Determine authentication state
	userID, username, isAdmin, authenticated := s.authenticateFromHeader(c)

	// Check access level from tool definition (defense-in-depth: enforced even if tools/list was bypassed)
	toolLevel := s.getToolAccessLevel(params.Name)
	if toolLevel == toolAccessAuth && !authenticated {
		return s.toolError(req.ID, "authentication required: provide API token via Authorization: Bearer <token> header")
	}
	if toolLevel == toolAccessAdmin && !authenticated {
		return s.toolError(req.ID, "authentication required: provide API token via Authorization: Bearer <token> header")
	}
	if toolLevel == toolAccessAdmin && !isAdmin {
		return s.toolError(req.ID, "admin privileges required")
	}

	// Execute tool
	var result string
	var toolErr error

	switch params.Name {
	case "list_benchmarks":
		result, toolErr = s.toolListBenchmarks(params.Arguments)
	case "get_benchmark":
		result, toolErr = s.toolGetBenchmark(params.Arguments)
	case "get_benchmark_data":
		result, toolErr = s.toolGetBenchmarkData(params.Arguments)
	case "get_benchmark_run":
		result, toolErr = s.toolGetBenchmarkRun(params.Arguments)
	case "update_benchmark":
		result, toolErr = s.toolUpdateBenchmark(params.Arguments, userID, username, isAdmin)
	case "list_users":
		result, toolErr = s.toolListUsers(params.Arguments)
	case "delete_user":
		result, toolErr = s.toolDeleteUser(params.Arguments, userID, username)
	case "delete_user_benchmarks":
		result, toolErr = s.toolDeleteUserBenchmarks(params.Arguments, userID, username)
	case "ban_user":
		result, toolErr = s.toolBanUser(params.Arguments, userID, username)
	case "toggle_user_admin":
		result, toolErr = s.toolToggleUserAdmin(params.Arguments, userID, username)
	default:
		return jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonrpcError{Code: jsonrpcMethodNotFound, Message: fmt.Sprintf("unknown tool: %s", params.Name)},
		}
	}

	if toolErr != nil {
		return s.toolError(req.ID, toolErr.Error())
	}

	// Apply jq filter if provided
	result, toolErr = applyJQFilter(params.Arguments, result)
	if toolErr != nil {
		return s.toolError(req.ID, toolErr.Error())
	}

	return jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: result}},
		},
	}
}

// authenticateFromHeader checks the Authorization header for an API token
// Returns (userID, username, isAdmin, authenticated)
func (s *mcpServer) authenticateFromHeader(c *gin.Context) (uint, string, bool, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return 0, "", false, false
	}

	const prefix = "Bearer "
	if len(authHeader) <= len(prefix) || authHeader[:len(prefix)] != prefix {
		return 0, "", false, false
	}

	token := authHeader[len(prefix):]

	var apiToken APIToken
	if err := s.db.DB.Preload("User").Where("token = ?", token).First(&apiToken).Error; err != nil {
		return 0, "", false, false
	}

	// Check if user is banned
	if apiToken.User.IsBanned {
		return 0, "", false, false
	}

	// Update last used timestamp
	now := time.Now()
	apiToken.LastUsedAt = &now
	s.db.DB.Save(&apiToken)
	s.db.DB.Model(&User{}).Where("id = ?", apiToken.UserID).Update("last_api_activity_at", now)

	return apiToken.UserID, apiToken.User.Username, apiToken.User.IsAdmin, true
}

func (s *mcpServer) toolError(id json.RawMessage, msg string) jsonrpcResponse {
	return jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: msg}},
			IsError: true,
		},
	}
}

// getToolAccessLevel returns the access level for a tool by name.
func (s *mcpServer) getToolAccessLevel(name string) string {
	for _, tool := range s.tools {
		if tool.Name == name {
			return tool.accessLevel
		}
	}
	// Unknown tools default to admin level to fail safely
	return toolAccessAdmin
}

// applyJQFilter applies an optional jq filter from the tool arguments to the JSON result string.
// If no "jq" argument is provided, the result is returned unchanged.
func applyJQFilter(args json.RawMessage, result string) (string, error) {
	if args == nil {
		return result, nil
	}

	var jqArgs struct {
		JQ string `json:"jq"`
	}
	if err := json.Unmarshal(args, &jqArgs); err != nil {
		return result, nil //nolint:nilerr // args may not contain jq field; ignore unmarshal errors
	}
	if jqArgs.JQ == "" {
		return result, nil
	}

	query, err := gojq.Parse(jqArgs.JQ)
	if err != nil {
		return "", fmt.Errorf("jq parse error: %w", err)
	}

	var input interface{}
	if parseErr := json.Unmarshal([]byte(result), &input); parseErr != nil {
		return "", fmt.Errorf("jq: failed to parse tool result as JSON: %w", parseErr)
	}

	iter := query.Run(input)
	var outputs []interface{}
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if jqErr, isErr := v.(error); isErr {
			return "", fmt.Errorf("jq evaluation error: %w", jqErr)
		}
		outputs = append(outputs, v)
	}

	// If single result, return it directly; otherwise return as array
	var out interface{}
	switch len(outputs) {
	case 0:
		out = nil
	case 1:
		out = outputs[0]
	default:
		out = outputs
	}

	data, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("jq: failed to marshal filtered result: %w", err)
	}
	return string(data), nil
}

// --- Tool implementations ---

func (s *mcpServer) toolListBenchmarks(args json.RawMessage) (string, error) {
	var params struct {
		Page      int    `json:"page"`
		PerPage   int    `json:"per_page"`
		Search    string `json:"search"`
		UserID    int    `json:"user_id"`
		Username  string `json:"username"`
		SortBy    string `json:"sort_by"`
		SortOrder string `json:"sort_order"`
	}
	if args != nil {
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}

	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 || params.PerPage > 100 {
		params.PerPage = 10
	}

	query := s.db.DB.Preload("User")

	if params.UserID > 0 {
		query = query.Where("user_id = ?", params.UserID)
	}

	// Resolve username to user_id if provided (case-insensitive exact match)
	if params.Username != "" && params.UserID <= 0 {
		var user User
		if err := s.db.DB.Where("username = ? COLLATE NOCASE", params.Username).First(&user).Error; err != nil {
			return "", fmt.Errorf("user not found: %s", params.Username)
		}
		query = query.Where("user_id = ?", user.ID)
	}

	if params.Search != "" {
		keywords := strings.Fields(params.Search)
		for _, keyword := range keywords {
			keyword = strings.TrimSpace(keyword)
			if keyword != "" {
				orClause := "benchmarks.title LIKE ? OR benchmarks.description LIKE ? OR EXISTS (SELECT 1 FROM users WHERE users.id = benchmarks.user_id AND users.username LIKE ?) OR benchmarks.run_names LIKE ? OR benchmarks.specifications LIKE ?"
				query = query.Where(orClause, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
			}
		}
	}

	// Sorting
	var orderClause string
	switch params.SortBy {
	case "title":
		orderClause = "title"
	case "updated_at":
		orderClause = "updated_at"
	default:
		orderClause = "created_at"
	}
	if params.SortOrder == "asc" {
		orderClause += " ASC"
	} else {
		orderClause += " DESC"
	}
	query = query.Order(orderClause)

	var total int64
	if err := query.Model(&Benchmark{}).Count(&total).Error; err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	var benchmarks []Benchmark
	offset := (params.Page - 1) * params.PerPage
	if err := query.Offset(offset).Limit(params.PerPage).Find(&benchmarks).Error; err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	// Populate run count and labels
	var wg sync.WaitGroup
	for i := range benchmarks {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			count, labels, err := GetBenchmarkRunCount(benchmarks[idx].ID)
			if err == nil {
				benchmarks[idx].RunCount = count
				benchmarks[idx].RunLabels = labels
			}
		}(i)
	}
	wg.Wait()

	totalPages := int((total + int64(params.PerPage) - 1) / int64(params.PerPage))

	result := map[string]interface{}{
		"benchmarks":  benchmarks,
		"page":        params.Page,
		"per_page":    params.PerPage,
		"total":       total,
		"total_pages": totalPages,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolGetBenchmark(args json.RawMessage) (string, error) {
	var params struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.ID <= 0 {
		return "", fmt.Errorf("id is required")
	}

	var benchmark Benchmark
	if err := s.db.DB.Preload("User").First(&benchmark, params.ID).Error; err != nil {
		return "", fmt.Errorf("benchmark not found")
	}

	count, labels, err := GetBenchmarkRunCount(benchmark.ID)
	if err == nil {
		benchmark.RunCount = count
		benchmark.RunLabels = labels
	}

	data, err := json.Marshal(benchmark)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolGetBenchmarkData(args json.RawMessage) (string, error) {
	var params struct {
		ID        int `json:"id"`
		MaxPoints int `json:"max_points"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.ID <= 0 {
		return "", fmt.Errorf("id is required")
	}

	maxPoints := params.MaxPoints
	switch {
	case maxPoints < 0:
		maxPoints = 0 // stats only
	case maxPoints > maxMaxPoints:
		maxPoints = maxMaxPoints
	}

	// Verify benchmark exists and load metadata
	var benchmark Benchmark
	if err := s.db.DB.Preload("User").First(&benchmark, params.ID).Error; err != nil {
		return "", fmt.Errorf("benchmark not found")
	}

	count, labels, err := GetBenchmarkRunCount(benchmark.ID)
	if err == nil {
		benchmark.RunCount = count
		benchmark.RunLabels = labels
	}

	// Use pre-calculated stats
	preCalc, err := RetrievePreCalculatedStats(uint(params.ID))
	if err != nil {
		return "", fmt.Errorf("pre-calculated stats not available")
	}

	summaries := make([]*BenchmarkDataSummary, len(preCalc))
	for i, run := range preCalc {
		summaries[i] = PreCalculatedRunToMCPSummary(run, maxPoints)
	}

	result := map[string]interface{}{
		"benchmark": benchmark,
		"runs":      summaries,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolGetBenchmarkRun(args json.RawMessage) (string, error) {
	var params struct {
		ID        int `json:"id"`
		RunIndex  int `json:"run_index"`
		MaxPoints int `json:"max_points"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.ID <= 0 {
		return "", fmt.Errorf("id is required")
	}

	maxPoints := params.MaxPoints
	switch {
	case maxPoints < 0:
		maxPoints = 0
	case maxPoints > maxMaxPoints:
		maxPoints = maxMaxPoints
	}

	// Verify benchmark exists
	var benchmark Benchmark
	if err := s.db.DB.First(&benchmark, params.ID).Error; err != nil {
		return "", fmt.Errorf("benchmark not found")
	}

	// Use pre-calculated stats for the single run
	run, err := RetrievePreCalculatedStatsRun(uint(params.ID), params.RunIndex)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve run: %w", err)
	}

	summary := PreCalculatedRunToMCPSummary(run, maxPoints)

	data, err := json.Marshal(summary)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolUpdateBenchmark(args json.RawMessage, userID uint, username string, isAdmin bool) (string, error) {
	var params struct {
		ID          int               `json:"id"`
		Title       string            `json:"title"`
		Description string            `json:"description"`
		Labels      map[string]string `json:"labels"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.ID <= 0 {
		return "", fmt.Errorf("id is required")
	}

	// Check if user is banned (admins can still update)
	if !isAdmin {
		var user User
		if err := s.db.DB.First(&user, userID).Error; err != nil {
			return "", fmt.Errorf("user not found")
		}
		if user.IsBanned {
			return "", fmt.Errorf("your account has been banned")
		}
	}

	var benchmark Benchmark
	if err := s.db.DB.First(&benchmark, params.ID).Error; err != nil {
		return "", fmt.Errorf("benchmark not found")
	}

	// Check ownership or admin
	if benchmark.UserID != userID && !isAdmin {
		return "", fmt.Errorf("not authorized")
	}

	// Track what fields were changed for audit logging
	var changes []string

	// Validate title length
	if params.Title != "" {
		if len(params.Title) > 100 {
			return "", fmt.Errorf("title must be at most 100 characters")
		}
		benchmark.Title = params.Title
		changes = append(changes, "title")
	}
	if params.Description != "" {
		if len(params.Description) > 5000 {
			return "", fmt.Errorf("description must be at most 5000 characters")
		}
		benchmark.Description = params.Description
		changes = append(changes, "description")
	}

	// Update labels if provided
	if len(params.Labels) > 0 {
		changes = append(changes, "labels")
		benchmarkData, err := RetrieveBenchmarkData(uint(params.ID))
		if err != nil {
			return "", fmt.Errorf("failed to retrieve benchmark data: %w", err)
		}

		for idxStr, newLabel := range params.Labels {
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				continue
			}
			if idx >= 0 && idx < len(benchmarkData) {
				benchmarkData[idx].Label = newLabel
			}
		}

		if err := StoreBenchmarkData(benchmarkData, uint(params.ID)); err != nil {
			return "", fmt.Errorf("failed to update labels: %w", err)
		}

		runNames, specifications := ExtractSearchableMetadata(benchmarkData)
		benchmark.RunNames = runNames
		benchmark.Specifications = specifications

		runtime.GC()
	}

	if err := s.db.DB.Save(&benchmark).Error; err != nil {
		return "", fmt.Errorf("failed to update benchmark: %w", err)
	}

	// Reload with user data
	if err := s.db.DB.Preload("User").First(&benchmark, benchmark.ID).Error; err != nil {
		return "", fmt.Errorf("failed to load benchmark: %w", err)
	}

	LogBenchmarkUpdated(userID, username, benchmark.ID, benchmark.Title, changes)

	data, err := json.Marshal(benchmark)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolListUsers(args json.RawMessage) (string, error) {
	var params struct {
		Page    int    `json:"page"`
		PerPage int    `json:"per_page"`
		Search  string `json:"search"`
	}
	if args != nil {
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}

	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 || params.PerPage > 100 {
		params.PerPage = 10
	}

	query := s.db.DB.Model(&User{})
	if params.Search != "" {
		query = query.Where("username LIKE ? OR discord_id LIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	var users []User
	offset := (params.Page - 1) * params.PerPage
	if err := query.Order("created_at DESC").Offset(offset).Limit(params.PerPage).Find(&users).Error; err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	for i := range users {
		var benchCount int64
		s.db.DB.Model(&Benchmark{}).Where("user_id = ?", users[i].ID).Count(&benchCount)
		users[i].BenchmarkCount = int(benchCount)

		var tokenCount int64
		s.db.DB.Model(&APIToken{}).Where("user_id = ?", users[i].ID).Count(&tokenCount)
		users[i].APITokenCount = int(tokenCount)
	}

	totalPages := int((total + int64(params.PerPage) - 1) / int64(params.PerPage))

	result := map[string]interface{}{
		"users":       users,
		"page":        params.Page,
		"per_page":    params.PerPage,
		"total":       total,
		"total_pages": totalPages,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolDeleteUser(args json.RawMessage, adminUserID uint, adminUsername string) (string, error) {
	var params struct {
		UserID     int  `json:"user_id"`
		DeleteData bool `json:"delete_data"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.UserID <= 0 {
		return "", fmt.Errorf("user_id is required")
	}

	var user User
	if err := s.db.DB.First(&user, params.UserID).Error; err != nil {
		return "", fmt.Errorf("user not found")
	}

	// Prevent self-deletion
	if user.ID == adminUserID {
		return "", fmt.Errorf("cannot delete your own account")
	}

	username := user.Username

	if params.DeleteData {
		var benchmarks []Benchmark
		if err := s.db.DB.Where("user_id = ?", user.ID).Find(&benchmarks).Error; err != nil {
			return "", fmt.Errorf("failed to find user benchmarks: %w", err)
		}
		for i := range benchmarks {
			if delErr := DeleteBenchmarkData(benchmarks[i].ID); delErr != nil {
				fmt.Printf("Warning: failed to delete data for benchmark %d\n", benchmarks[i].ID)
			}
		}
	}

	if err := s.db.DB.Delete(&user).Error; err != nil {
		return "", fmt.Errorf("failed to delete user: %w", err)
	}

	LogUserDeleted(adminUserID, adminUsername, user.ID, username)

	result := map[string]interface{}{
		"message":  "user deleted",
		"user_id":  params.UserID,
		"username": username,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolDeleteUserBenchmarks(args json.RawMessage, adminUserID uint, adminUsername string) (string, error) {
	var params struct {
		UserID int `json:"user_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.UserID <= 0 {
		return "", fmt.Errorf("user_id is required")
	}

	var user User
	if err := s.db.DB.First(&user, params.UserID).Error; err != nil {
		return "", fmt.Errorf("user not found")
	}

	var benchmarks []Benchmark
	if err := s.db.DB.Where("user_id = ?", user.ID).Find(&benchmarks).Error; err != nil {
		return "", fmt.Errorf("failed to find user benchmarks: %w", err)
	}

	for i := range benchmarks {
		if delErr := DeleteBenchmarkData(benchmarks[i].ID); delErr != nil {
			fmt.Printf("Warning: failed to delete data for benchmark %d\n", benchmarks[i].ID)
		}
	}

	if err := s.db.DB.Where("user_id = ?", user.ID).Delete(&Benchmark{}).Error; err != nil {
		return "", fmt.Errorf("failed to delete benchmarks: %w", err)
	}

	LogUserBenchmarksDeleted(adminUserID, adminUsername, user.ID, user.Username, len(benchmarks))

	result := map[string]interface{}{
		"message":  "all user benchmarks deleted",
		"user_id":  params.UserID,
		"username": user.Username,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolBanUser(args json.RawMessage, adminUserID uint, adminUsername string) (string, error) {
	var params struct {
		UserID int  `json:"user_id"`
		Banned bool `json:"banned"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.UserID <= 0 {
		return "", fmt.Errorf("user_id is required")
	}

	var user User
	if err := s.db.DB.First(&user, params.UserID).Error; err != nil {
		return "", fmt.Errorf("user not found")
	}

	// Prevent self-ban
	if user.ID == adminUserID && params.Banned {
		return "", fmt.Errorf("cannot ban your own account")
	}

	user.IsBanned = params.Banned
	if err := s.db.DB.Save(&user).Error; err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	if params.Banned {
		LogUserBanned(adminUserID, adminUsername, user.ID, user.Username)
	} else {
		LogUserUnbanned(adminUserID, adminUsername, user.ID, user.Username)
	}

	data, err := json.Marshal(user)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolToggleUserAdmin(args json.RawMessage, adminUserID uint, adminUsername string) (string, error) {
	var params struct {
		UserID  int  `json:"user_id"`
		IsAdmin bool `json:"is_admin"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.UserID <= 0 {
		return "", fmt.Errorf("user_id is required")
	}

	var user User
	if err := s.db.DB.First(&user, params.UserID).Error; err != nil {
		return "", fmt.Errorf("user not found")
	}

	// Prevent self-demotion
	if user.ID == adminUserID && !params.IsAdmin {
		return "", fmt.Errorf("cannot revoke your own admin privileges")
	}

	user.IsAdmin = params.IsAdmin
	if err := s.db.DB.Save(&user).Error; err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	if params.IsAdmin {
		LogUserAdminGranted(adminUserID, adminUsername, user.ID, user.Username)
	} else {
		LogUserAdminRevoked(adminUserID, adminUsername, user.ID, user.Username)
	}

	data, err := json.Marshal(user)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}
