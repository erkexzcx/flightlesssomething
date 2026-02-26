package app

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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
	jsonrpcInternalError  = -32603
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
	Title           string `json:"title,omitempty"`
	ReadOnlyHint    *bool  `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool  `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool  `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool  `json:"openWorldHint,omitempty"`
}

type mcpIcon struct {
	Src      string `json:"src"`
	MIMEType string `json:"mimeType,omitempty"`
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
	P97      float64   `json:"p97"`
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

func (s *mcpServer) defineTools() []mcpTool {
	boolPtr := func(b bool) *bool { return &b }

	// Font Awesome 6 Free solid SVG icons via jsDelivr CDN
	faIcon := func(name string) []mcpIcon {
		return []mcpIcon{{
			Src:      "https://cdn.jsdelivr.net/npm/@fortawesome/fontawesome-free@6.7.2/svgs/solid/" + name + ".svg",
			MIMEType: "image/svg+xml",
		}}
	}

	return []mcpTool{
		{
			Name:        "list_benchmarks",
			Title:       "Browse Benchmarks",
			Description: "Search and list gaming benchmarks with pagination, search, and sorting. Returns benchmark metadata including title, description, user, run count, and timestamps.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page":      map[string]interface{}{"type": "integer", "description": "Page number (default: 1)"},
					"per_page":  map[string]interface{}{"type": "integer", "description": "Results per page, 1-100 (default: 10)"},
					"search":    map[string]interface{}{"type": "string", "description": "Search keywords (space-separated, AND logic). Searches title, description, username, run names, and specifications."},
					"user_id":   map[string]interface{}{"type": "integer", "description": "Filter by user ID"},
					"sort_by":   map[string]interface{}{"type": "string", "enum": []string{"title", "created_at", "updated_at"}, "description": "Sort field (default: created_at)"},
					"sort_order": map[string]interface{}{"type": "string", "enum": []string{"asc", "desc"}, "description": "Sort order (default: desc)"},
				},
			},
			Icons:       faIcon("magnifying-glass"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessPublic,
		},
		{
			Name:        "get_benchmark",
			Title:       "View Benchmark",
			Description: "Get detailed information about a specific benchmark including title, description, user, run count, run labels, and timestamps.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "integer", "description": "Benchmark ID"},
				},
			},
			Icons:       faIcon("eye"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessPublic,
		},
		{
			Name:        "get_benchmark_data",
			Title:       "Benchmark Statistics",
			Description: "Get computed statistics for all benchmark runs. Returns per-metric stats matching the web UI: min, max, avg, median, p01, p97, std_dev, variance, count. FPS stats are correctly derived from frametime data. Raw data points are omitted by default; set max_points > 0 to include downsampled time series.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id":         map[string]interface{}{"type": "integer", "description": "Benchmark ID"},
					"max_points": map[string]interface{}{"type": "integer", "description": "Include downsampled raw data points (default: 0 = stats only). Set 1-5000 for time series data alongside stats."},
				},
			},
			Icons:       faIcon("chart-simple"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessPublic,
		},
		{
			Name:        "get_benchmark_run",
			Title:       "Run Statistics",
			Description: "Get computed statistics for a specific run within a benchmark. Same stats as get_benchmark_data but for a single run. Raw data points omitted by default.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "run_index"},
				"properties": map[string]interface{}{
					"id":         map[string]interface{}{"type": "integer", "description": "Benchmark ID"},
					"run_index":  map[string]interface{}{"type": "integer", "description": "Run index (0-based)"},
					"max_points": map[string]interface{}{"type": "integer", "description": "Include downsampled raw data points (default: 0 = stats only). Set 1-5000 for time series data alongside stats."},
				},
			},
			Icons:       faIcon("chart-line"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessPublic,
		},
		{
			Name:        "get_current_user",
			Title:       "My Profile",
			Description: "Get the currently authenticated user's information including user ID, username, and admin status. Requires authentication via API token. Use the returned user ID with list_benchmarks to find your own benchmarks.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			Icons:       faIcon("circle-user"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAuth,
		},
		{
			Name:        "update_benchmark",
			Title:       "Edit Benchmark",
			Description: "Update benchmark metadata (title, description) and/or run labels. Requires authentication via API token. Only the benchmark owner or an admin can update.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id":          map[string]interface{}{"type": "integer", "description": "Benchmark ID"},
					"title":       map[string]interface{}{"type": "string", "description": "New title (max 100 chars)"},
					"description": map[string]interface{}{"type": "string", "description": "New description (max 5000 chars)"},
					"labels":      map[string]interface{}{"type": "object", "description": "Map of run index (as string) to new label, e.g. {\"0\": \"Run A\", \"1\": \"Run B\"}", "additionalProperties": map[string]interface{}{"type": "string"}},
				},
			},
			Icons:       faIcon("pen"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAuth,
		},
		{
			Name:        "delete_benchmark",
			Title:       "Delete Benchmark",
			Description: "Delete a benchmark and all its data. Requires authentication via API token. Only the benchmark owner or an admin can delete.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "integer", "description": "Benchmark ID"},
				},
			},
			Icons:       faIcon("trash-can"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(true), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAuth,
		},
		{
			Name:        "delete_benchmark_run",
			Title:       "Delete Run",
			Description: "Delete a specific run from a benchmark. Cannot delete the last remaining run. Requires authentication via API token. Only the benchmark owner or an admin can delete.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "run_index"},
				"properties": map[string]interface{}{
					"id":        map[string]interface{}{"type": "integer", "description": "Benchmark ID"},
					"run_index": map[string]interface{}{"type": "integer", "description": "Run index (0-based)"},
				},
			},
			Icons:       faIcon("circle-minus"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(true), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAuth,
		},
		{
			Name:        "list_api_tokens",
			Title:       "My API Tokens",
			Description: "List all API tokens for the currently authenticated user. Requires authentication via API token.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			Icons:       faIcon("key"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAuth,
		},
		{
			Name:        "create_api_token",
			Title:       "Create API Token",
			Description: "Create a new API token for the currently authenticated user. Maximum 10 tokens per user. Requires authentication via API token.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"name"},
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Token name (1-100 chars)"},
				},
			},
			Icons:       faIcon("circle-plus"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAuth,
		},
		{
			Name:        "delete_api_token",
			Title:       "Delete API Token",
			Description: "Delete an API token belonging to the currently authenticated user. Requires authentication via API token.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"token_id"},
				"properties": map[string]interface{}{
					"token_id": map[string]interface{}{"type": "integer", "description": "Token ID to delete"},
				},
			},
			Icons:       faIcon("circle-xmark"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(true), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAuth,
		},
		{
			Name:        "list_users",
			Title:       "Manage Users",
			Description: "List all users with pagination and optional search. Admin only. Requires authentication via API token with admin privileges.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page":     map[string]interface{}{"type": "integer", "description": "Page number (default: 1)"},
					"per_page": map[string]interface{}{"type": "integer", "description": "Results per page, 1-100 (default: 10)"},
					"search":   map[string]interface{}{"type": "string", "description": "Search by username or Discord ID"},
				},
			},
			Icons:       faIcon("users"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAdmin,
		},
		{
			Name:        "list_audit_logs",
			Title:       "Audit Logs",
			Description: "List audit logs with pagination and optional filters. Admin only. Requires authentication via API token with admin privileges.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page":        map[string]interface{}{"type": "integer", "description": "Page number (default: 1)"},
					"per_page":    map[string]interface{}{"type": "integer", "description": "Results per page, 1-100 (default: 50)"},
					"user_id":     map[string]interface{}{"type": "integer", "description": "Filter by user ID who performed the action"},
					"action":      map[string]interface{}{"type": "string", "description": "Filter by action (partial match)"},
					"target_type": map[string]interface{}{"type": "string", "description": "Filter by target type (e.g. 'user', 'benchmark')"},
				},
			},
			Icons:       faIcon("clipboard-list"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(true), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAdmin,
		},
		{
			Name:        "delete_user",
			Title:       "Delete User",
			Description: "Delete a user account. Admin only. Cannot delete your own account. Optionally delete all user data (benchmarks). Requires authentication via API token with admin privileges.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"user_id"},
				"properties": map[string]interface{}{
					"user_id":     map[string]interface{}{"type": "integer", "description": "User ID to delete"},
					"delete_data": map[string]interface{}{"type": "boolean", "description": "Also delete all benchmark data files (default: false)"},
				},
			},
			Icons:       faIcon("user-xmark"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(true), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAdmin,
		},
		{
			Name:        "delete_user_benchmarks",
			Title:       "Delete User's Benchmarks",
			Description: "Delete all benchmarks belonging to a user. Admin only. Requires authentication via API token with admin privileges.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"user_id"},
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{"type": "integer", "description": "User ID whose benchmarks to delete"},
				},
			},
			Icons:       faIcon("folder-minus"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(true), OpenWorldHint: boolPtr(false)},
			accessLevel: toolAccessAdmin,
		},
		{
			Name:        "ban_user",
			Title:       "Ban / Unban User",
			Description: "Ban or unban a user. Admin only. Cannot ban your own account. Requires authentication via API token with admin privileges.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"user_id", "banned"},
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{"type": "integer", "description": "User ID to ban/unban"},
					"banned":  map[string]interface{}{"type": "boolean", "description": "true to ban, false to unban"},
				},
			},
			Icons:       faIcon("ban"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
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
				},
			},
			Icons:       faIcon("shield-halved"),
			Annotations: &mcpToolAnnotations{ReadOnlyHint: boolPtr(false), DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(false)},
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
			resp = server.handleInitialize(&req)
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

func (s *mcpServer) handleInitialize(req *jsonrpcRequest) jsonrpcResponse {
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
			Instructions: "FlightlessSomething is a gaming benchmark storage service. " +
				"You can browse, search, and analyze benchmarks using the provided tools. " +
				"To create benchmarks, add runs, or download raw benchmark data, use curl with the REST API instead of MCP tools " +
				"(these operations involve large CSV files unsuitable for MCP). " +
				"Use 'Authorization: Bearer <token>' header for authentication. " +
				"REST API endpoints for benchmark data operations:\n" +
				"- Create benchmark: curl -X POST /api/benchmarks -H 'Authorization: Bearer <token>' -F 'title=...' -F 'files=@file.csv'\n" +
				"- Add runs: curl -X POST /api/benchmarks/<id>/runs -H 'Authorization: Bearer <token>' -F 'files=@file.csv'\n" +
				"- Download benchmark: curl /api/benchmarks/<id>/download -o benchmark.zip",
		},
	}
}

func (s *mcpServer) handleToolsList(c *gin.Context, req *jsonrpcRequest) jsonrpcResponse {
	_, isAdmin, authenticated := s.authenticateFromHeader(c)

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
	userID, isAdmin, authenticated := s.authenticateFromHeader(c)

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
		result, toolErr = s.toolUpdateBenchmark(params.Arguments, userID, isAdmin)
	case "delete_benchmark":
		result, toolErr = s.toolDeleteBenchmark(params.Arguments, userID, isAdmin)
	case "delete_benchmark_run":
		result, toolErr = s.toolDeleteBenchmarkRun(params.Arguments, userID, isAdmin)
	case "get_current_user":
		result, toolErr = s.toolGetCurrentUser(userID)
	case "list_api_tokens":
		result, toolErr = s.toolListAPITokens(userID)
	case "create_api_token":
		result, toolErr = s.toolCreateAPIToken(params.Arguments, userID)
	case "delete_api_token":
		result, toolErr = s.toolDeleteAPIToken(params.Arguments, userID)
	case "list_users":
		result, toolErr = s.toolListUsers(params.Arguments)
	case "list_audit_logs":
		result, toolErr = s.toolListAuditLogs(params.Arguments)
	case "delete_user":
		result, toolErr = s.toolDeleteUser(params.Arguments, userID)
	case "delete_user_benchmarks":
		result, toolErr = s.toolDeleteUserBenchmarks(params.Arguments, userID)
	case "ban_user":
		result, toolErr = s.toolBanUser(params.Arguments, userID)
	case "toggle_user_admin":
		result, toolErr = s.toolToggleUserAdmin(params.Arguments, userID)
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

	return jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: result}},
		},
	}
}

// authenticateFromHeader checks the Authorization header for an API token
// Returns (userID, isAdmin, authenticated)
func (s *mcpServer) authenticateFromHeader(c *gin.Context) (uint, bool, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return 0, false, false
	}

	const prefix = "Bearer "
	if len(authHeader) <= len(prefix) || authHeader[:len(prefix)] != prefix {
		return 0, false, false
	}

	token := authHeader[len(prefix):]

	var apiToken APIToken
	if err := s.db.DB.Preload("User").Where("token = ?", token).First(&apiToken).Error; err != nil {
		return 0, false, false
	}

	// Check if user is banned
	if apiToken.User.IsBanned {
		return 0, false, false
	}

	// Update last used timestamp
	now := time.Now()
	apiToken.LastUsedAt = &now
	s.db.DB.Save(&apiToken)
	s.db.DB.Model(&User{}).Where("id = ?", apiToken.UserID).Update("last_api_activity_at", now)

	return apiToken.UserID, apiToken.User.IsAdmin, true
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

// --- Tool implementations ---

func (s *mcpServer) toolListBenchmarks(args json.RawMessage) (string, error) {
	var params struct {
		Page      int    `json:"page"`
		PerPage   int    `json:"per_page"`
		Search    string `json:"search"`
		UserID    int    `json:"user_id"`
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
	// Default is 0 (stats only) — no switch case needed for == 0

	// Verify benchmark exists
	var benchmark Benchmark
	if err := s.db.DB.First(&benchmark, params.ID).Error; err != nil {
		return "", fmt.Errorf("benchmark not found")
	}

	benchmarkData, err := RetrieveBenchmarkData(uint(params.ID))
	if err != nil {
		return "", fmt.Errorf("failed to retrieve benchmark data: %w", err)
	}

	summaries := make([]*BenchmarkDataSummary, len(benchmarkData))
	for i, run := range benchmarkData {
		summaries[i] = summarizeBenchmarkData(run, maxPoints)
	}

	// Trigger GC to reclaim memory from loaded benchmark data
	runtime.GC()

	data, err := json.Marshal(summaries)
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

	run, err := RetrieveBenchmarkRun(uint(params.ID), params.RunIndex)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve run: %w", err)
	}

	summary := summarizeBenchmarkData(run, maxPoints)

	runtime.GC()

	data, err := json.Marshal(summary)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolUpdateBenchmark(args json.RawMessage, userID uint, isAdmin bool) (string, error) {
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

	// Validate title length
	if params.Title != "" {
		if len(params.Title) > 100 {
			return "", fmt.Errorf("title must be at most 100 characters")
		}
		benchmark.Title = params.Title
	}
	if params.Description != "" {
		if len(params.Description) > 5000 {
			return "", fmt.Errorf("description must be at most 5000 characters")
		}
		benchmark.Description = params.Description
	}

	// Update labels if provided
	if len(params.Labels) > 0 {
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

	LogBenchmarkUpdated(s.db, userID, benchmark.ID, benchmark.Title)

	data, err := json.Marshal(benchmark)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolDeleteBenchmark(args json.RawMessage, userID uint, isAdmin bool) (string, error) {
	var params struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.ID <= 0 {
		return "", fmt.Errorf("id is required")
	}

	// Check if user is banned (admins can still delete)
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

	title := benchmark.Title

	if err := DeleteBenchmarkData(benchmark.ID); err != nil {
		fmt.Printf("Warning: failed to delete benchmark data file: %v\n", err)
	}

	if err := s.db.DB.Delete(&benchmark).Error; err != nil {
		return "", fmt.Errorf("failed to delete benchmark: %w", err)
	}

	LogBenchmarkDeleted(s.db, userID, benchmark.ID, title)

	result := map[string]interface{}{
		"message": "benchmark deleted",
		"id":      params.ID,
		"title":   title,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolDeleteBenchmarkRun(args json.RawMessage, userID uint, isAdmin bool) (string, error) {
	var params struct {
		ID       int `json:"id"`
		RunIndex int `json:"run_index"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.ID <= 0 {
		return "", fmt.Errorf("id is required")
	}

	// Check if user is banned (admins can still delete)
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

	benchmarkData, err := RetrieveBenchmarkData(uint(params.ID))
	if err != nil {
		return "", fmt.Errorf("failed to retrieve benchmark data: %w", err)
	}

	if params.RunIndex < 0 || params.RunIndex >= len(benchmarkData) {
		return "", fmt.Errorf("run index out of range")
	}

	if len(benchmarkData) == 1 {
		return "", fmt.Errorf("cannot delete the last run - delete the entire benchmark instead")
	}

	benchmarkData = append(benchmarkData[:params.RunIndex], benchmarkData[params.RunIndex+1:]...)

	if err := StoreBenchmarkData(benchmarkData, uint(params.ID)); err != nil {
		return "", fmt.Errorf("failed to update benchmark data: %w", err)
	}

	runNames, specifications := ExtractSearchableMetadata(benchmarkData)
	benchmark.RunNames = runNames
	benchmark.Specifications = specifications

	if err := s.db.DB.Save(&benchmark).Error; err != nil {
		return "", fmt.Errorf("failed to update benchmark: %w", err)
	}

	LogBenchmarkUpdated(s.db, userID, benchmark.ID, benchmark.Title)

	runtime.GC()

	return `{"message":"run deleted successfully"}`, nil
}

func (s *mcpServer) toolGetCurrentUser(userID uint) (string, error) {
	var user User
	if err := s.db.DB.First(&user, userID).Error; err != nil {
		return "", fmt.Errorf("user not found")
	}

	result := map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
		"is_admin": user.IsAdmin,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolListAPITokens(userID uint) (string, error) {
	var tokens []APIToken
	if err := s.db.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&tokens).Error; err != nil {
		return "", fmt.Errorf("failed to retrieve tokens: %w", err)
	}

	data, err := json.Marshal(tokens)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolCreateAPIToken(args json.RawMessage, userID uint) (string, error) {
	var params struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.Name == "" || len(params.Name) > 100 {
		return "", fmt.Errorf("name is required and must be at most 100 characters")
	}

	// Check token limit
	var count int64
	if err := s.db.DB.Model(&APIToken{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return "", fmt.Errorf("failed to check token count: %w", err)
	}
	if count >= maxTokensPerUser {
		return "", fmt.Errorf("maximum number of tokens reached (%d)", maxTokensPerUser)
	}

	token, err := generateAPIToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	apiToken := APIToken{
		UserID: userID,
		Token:  token,
		Name:   params.Name,
	}
	if createErr := s.db.DB.Create(&apiToken).Error; createErr != nil {
		return "", fmt.Errorf("failed to create token: %w", createErr)
	}

	data, err := json.Marshal(apiToken)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolDeleteAPIToken(args json.RawMessage, userID uint) (string, error) {
	var params struct {
		TokenID int `json:"token_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if params.TokenID <= 0 {
		return "", fmt.Errorf("token_id is required")
	}

	var token APIToken
	if err := s.db.DB.Where("id = ? AND user_id = ?", params.TokenID, userID).First(&token).Error; err != nil {
		return "", fmt.Errorf("token not found")
	}

	if err := s.db.DB.Delete(&token).Error; err != nil {
		return "", fmt.Errorf("failed to delete token: %w", err)
	}

	result := map[string]interface{}{
		"message":  "token deleted",
		"token_id": params.TokenID,
	}
	data, err := json.Marshal(result)
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

func (s *mcpServer) toolListAuditLogs(args json.RawMessage) (string, error) {
	var params struct {
		Page       int    `json:"page"`
		PerPage    int    `json:"per_page"`
		UserID     int    `json:"user_id"`
		Action     string `json:"action"`
		TargetType string `json:"target_type"`
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
		params.PerPage = 50
	}

	query := s.db.DB.Model(&AuditLog{}).Preload("User")

	if params.UserID > 0 {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.Action != "" {
		query = query.Where("action LIKE ?", "%"+params.Action+"%")
	}
	if params.TargetType != "" {
		query = query.Where("target_type = ?", params.TargetType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	var logs []AuditLog
	offset := (params.Page - 1) * params.PerPage
	if err := query.Order("created_at DESC").Offset(offset).Limit(params.PerPage).Find(&logs).Error; err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	totalPages := int((total + int64(params.PerPage) - 1) / int64(params.PerPage))

	result := map[string]interface{}{
		"logs":        logs,
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

func (s *mcpServer) toolDeleteUser(args json.RawMessage, adminUserID uint) (string, error) {
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

	LogUserDeleted(s.db, adminUserID, user.ID, username)

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

func (s *mcpServer) toolDeleteUserBenchmarks(args json.RawMessage, adminUserID uint) (string, error) {
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

	LogUserBenchmarksDeleted(s.db, adminUserID, user.ID, user.Username)

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

func (s *mcpServer) toolBanUser(args json.RawMessage, adminUserID uint) (string, error) {
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
		LogUserBanned(s.db, adminUserID, user.ID, user.Username)
	} else {
		LogUserUnbanned(s.db, adminUserID, user.ID, user.Username)
	}

	data, err := json.Marshal(user)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

func (s *mcpServer) toolToggleUserAdmin(args json.RawMessage, adminUserID uint) (string, error) {
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
		LogUserAdminGranted(s.db, adminUserID, user.ID, user.Username)
	} else {
		LogUserAdminRevoked(s.db, adminUserID, user.ID, user.Username)
	}

	data, err := json.Marshal(user)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

// --- Statistics computation (matches web frontend statsCalculations.js) ---

// summarizeBenchmarkData computes per-metric statistics for a benchmark run.
// FPS stats are derived from frametime data (matching the web frontend behavior).
// If maxPoints > 0, downsampled raw data is also included.
func summarizeBenchmarkData(run *BenchmarkData, maxPoints int) *BenchmarkDataSummary {
	totalPoints := len(run.DataFPS)
	if totalPoints == 0 {
		totalPoints = len(run.DataFrameTime)
	}

	actualMax := maxPoints
	if maxPoints > 0 && totalPoints <= maxPoints {
		actualMax = totalPoints
	}
	if maxPoints <= 0 {
		actualMax = 0
	}

	summary := &BenchmarkDataSummary{
		Label:              run.Label,
		SpecOS:             run.SpecOS,
		SpecCPU:            run.SpecCPU,
		SpecGPU:            run.SpecGPU,
		SpecRAM:            run.SpecRAM,
		SpecLinuxKernel:    run.SpecLinuxKernel,
		SpecLinuxScheduler: run.SpecLinuxScheduler,
		TotalDataPoints:    totalPoints,
		Metrics:            make(map[string]*MetricSummary),
	}
	if actualMax > 0 {
		summary.DownsampledTo = actualMax
	}

	addMetric := func(name string, data []float64) {
		if len(data) == 0 {
			return
		}
		summary.Metrics[name] = computeMetricSummary(data, actualMax)
	}

	// FPS: compute stats from frametime (matching web frontend calculateFPSStatsFromFrametime)
	if len(run.DataFrameTime) > 0 {
		summary.Metrics["fps"] = computeFPSFromFrametime(run.DataFrameTime, run.DataFPS, actualMax)
	} else if len(run.DataFPS) > 0 {
		addMetric("fps", run.DataFPS)
	}

	addMetric("frame_time", run.DataFrameTime)
	addMetric("cpu_load", run.DataCPULoad)
	addMetric("gpu_load", run.DataGPULoad)
	addMetric("cpu_temp", run.DataCPUTemp)
	addMetric("cpu_power", run.DataCPUPower)
	addMetric("gpu_temp", run.DataGPUTemp)
	addMetric("gpu_core_clock", run.DataGPUCoreClock)
	addMetric("gpu_mem_clock", run.DataGPUMemClock)
	addMetric("gpu_vram_used", run.DataGPUVRAMUsed)
	addMetric("gpu_power", run.DataGPUPower)
	addMetric("ram_used", run.DataRAMUsed)
	addMetric("swap_used", run.DataSwapUsed)

	return summary
}

// computeFPSFromFrametime calculates FPS statistics from frametime data,
// matching the web frontend's calculateFPSStatsFromFrametime function.
// FPS percentiles are inverted from frametime: low frametime = high FPS.
func computeFPSFromFrametime(frametimeData, fpsData []float64, maxPoints int) *MetricSummary {
	n := len(frametimeData)
	if n == 0 {
		return nil
	}

	// Sort frametime for percentile calculations
	sortedFT := make([]float64, n)
	copy(sortedFT, frametimeData)
	sort.Float64s(sortedFT)

	// FPS percentiles from frametime (inverted relationship):
	// 3rd percentile frametime (fast) → 97th percentile FPS
	// 99th percentile frametime (slow) → 1st percentile FPS
	ftP03 := percentile(sortedFT, 3)
	ftP99 := percentile(sortedFT, 99)

	var fpsP97, fpsP01 float64
	if ftP03 > 0 {
		fpsP97 = 1000 / ftP03
	}
	if ftP99 > 0 {
		fpsP01 = 1000 / ftP99
	}

	// Average FPS from average frametime
	var ftSum float64
	for _, v := range frametimeData {
		ftSum += v
	}
	avgFT := ftSum / float64(n)
	var avgFPS float64
	if avgFT > 0 {
		avgFPS = 1000 / avgFT
	}

	// Min/Max FPS from max/min frametime
	minFT := sortedFT[0]
	maxFT := sortedFT[n-1]
	var maxFPS, minFPS float64
	if minFT > 0 {
		maxFPS = 1000 / minFT
	}
	if maxFT > 0 {
		minFPS = 1000 / maxFT
	}

	// Convert all frametime to FPS for stddev/variance/median
	fpsValues := make([]float64, n)
	for i, ft := range frametimeData {
		if ft > 0 {
			fpsValues[i] = 1000 / ft
		}
	}

	// Sample variance (n-1 divisor, matches Excel/frontend)
	fpsMean := func() float64 {
		var s float64
		for _, v := range fpsValues {
			s += v
		}
		return s / float64(n)
	}()
	var sumSq float64
	for _, v := range fpsValues {
		diff := v - fpsMean
		sumSq += diff * diff
	}
	var variance float64
	if n > 1 {
		variance = sumSq / float64(n-1)
	}
	stdDev := math.Sqrt(variance)

	// Median from FPS values
	sortedFPS := make([]float64, n)
	copy(sortedFPS, fpsValues)
	sort.Float64s(sortedFPS)
	median := percentile(sortedFPS, 50)

	summary := &MetricSummary{
		Min:      math.Round(minFPS*100) / 100,
		Max:      math.Round(maxFPS*100) / 100,
		Avg:      math.Round(avgFPS*100) / 100,
		Median:   math.Round(median*100) / 100,
		P01:      math.Round(fpsP01*100) / 100,
		P97:      math.Round(fpsP97*100) / 100,
		StdDev:   math.Round(stdDev*100) / 100,
		Variance: math.Round(variance*100) / 100,
		Count:    n,
	}

	// Optionally include downsampled FPS data points
	if maxPoints > 0 && len(fpsData) > 0 {
		if len(fpsData) <= maxPoints {
			summary.Data = make([]float64, len(fpsData))
			for i, v := range fpsData {
				summary.Data[i] = math.Round(v*100) / 100
			}
		} else {
			summary.Data = downsampleSlice(fpsData, maxPoints)
		}
	}

	return summary
}

// computeMetricSummary computes statistics for a generic metric.
// Uses sample variance (n-1 divisor) and linear interpolation for percentiles,
// matching the web frontend's calculateStats function.
func computeMetricSummary(data []float64, maxPoints int) *MetricSummary {
	n := len(data)
	if n == 0 {
		return nil
	}

	// Min, max, avg
	var sum float64
	minVal := data[0]
	maxVal := data[0]
	for _, v := range data {
		sum += v
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	avg := sum / float64(n)

	// Sample variance (n-1), matching Excel/LibreOffice/frontend
	var sumSq float64
	for _, v := range data {
		diff := v - avg
		sumSq += diff * diff
	}
	var variance float64
	if n > 1 {
		variance = sumSq / float64(n-1)
	}
	stdDev := math.Sqrt(variance)

	// Sort a copy for percentile calculations
	sorted := make([]float64, n)
	copy(sorted, data)
	sort.Float64s(sorted)

	median := percentile(sorted, 50)
	p01 := percentile(sorted, 1)
	p97 := percentile(sorted, 97)

	summary := &MetricSummary{
		Min:      math.Round(minVal*100) / 100,
		Max:      math.Round(maxVal*100) / 100,
		Avg:      math.Round(avg*100) / 100,
		Median:   math.Round(median*100) / 100,
		P01:      math.Round(p01*100) / 100,
		P97:      math.Round(p97*100) / 100,
		StdDev:   math.Round(stdDev*100) / 100,
		Variance: math.Round(variance*100) / 100,
		Count:    n,
	}

	// Optionally include downsampled data points
	if maxPoints > 0 {
		if n <= maxPoints {
			summary.Data = make([]float64, n)
			for i, v := range data {
				summary.Data[i] = math.Round(v*100) / 100
			}
		} else {
			summary.Data = downsampleSlice(data, maxPoints)
		}
	}

	return summary
}

func downsampleSlice(data []float64, targetPoints int) []float64 {
	n := len(data)
	if n <= targetPoints || targetPoints <= 0 {
		result := make([]float64, n)
		for i, v := range data {
			result[i] = math.Round(v*100) / 100
		}
		return result
	}

	result := make([]float64, targetPoints)
	step := float64(n-1) / float64(targetPoints-1)
	for i := 0; i < targetPoints; i++ {
		idx := int(math.Round(step * float64(i)))
		if idx >= n {
			idx = n - 1
		}
		result[i] = math.Round(data[idx]*100) / 100
	}
	return result
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	rank := (p / 100) * float64(len(sorted)-1)
	lower := int(math.Floor(rank))
	upper := int(math.Ceil(rank))

	if lower == upper || upper >= len(sorted) {
		return sorted[lower]
	}

	// Linear interpolation
	frac := rank - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}
