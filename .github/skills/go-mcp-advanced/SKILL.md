---
name: go-mcp-advanced
description: "Go MCP SDK advanced features — OAuth/auth (server bearer tokens, client OAuth flows, enterprise SSO), sampling, elicitation, progress notifications, logging to clients, distributed/stateless servers, security (DNS rebinding, CSRF, cross-origin), debugging (MCPGODEBUG, LoggingTransport, MCP Inspector), and known API rough edges. Use when: implementing auth, sampling, elicitation, progress, logging, distributed deployments, or debugging MCP issues."
---

# Go MCP SDK — Advanced Features

Continuation of the `go-mcp-sdk` skill. Covers authentication, server-to-client interactions, distributed deployments, security, and debugging.

---

## Authentication — Server Side

### Bearer Token Verification

The `auth` package provides HTTP middleware for token verification:

```go
import "github.com/modelcontextprotocol/go-sdk/auth"

verifier := func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
    // Validate the token (e.g., JWT verification, database lookup)
    if !isValid(token) {
        return nil, auth.ErrInvalidToken
    }
    return &auth.TokenInfo{
        Scopes:     []string{"read", "write"},
        Expiration: time.Now().Add(time.Hour),
        UserID:     "user-123",  // Used for session hijacking prevention
        Extra:      map[string]any{"role": "admin"},
    }, nil
}

mcpHandler := mcp.NewStreamableHTTPHandler(getServer, opts)

// Wrap with auth middleware
protected := auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{
    ResourceMetadataURL: "https://example.com/.well-known/oauth-protected-resource",
    Scopes:              []string{"mcp:read"},
})(mcpHandler)

http.Handle("/mcp", protected)
```

### Accessing Token Info in Handlers

Token info flows through to MCP handlers via `req.Extra.TokenInfo`:

```go
mcp.AddTool(server, &mcp.Tool{Name: "secure_tool"}, func(ctx context.Context, req *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
    tokenInfo := req.Extra.TokenInfo  // *auth.TokenInfo or nil
    if tokenInfo != nil {
        userID := tokenInfo.UserID
        scopes := tokenInfo.Scopes
        // Check permissions...
    }
    return nil, nil, nil
})
```

You can also extract from context directly:
```go
tokenInfo := auth.TokenInfoFromContext(ctx)
```

### Protected Resource Metadata (RFC 9728)

Serve OAuth discovery metadata:

```go
import "github.com/modelcontextprotocol/go-sdk/oauthex"

metadataHandler := auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
    Resource:               "https://api.example.com/mcp",
    AuthorizationServers:   []string{"https://auth.example.com"},
    ScopesSupported:        []string{"mcp:read", "mcp:write"},
    BearerMethodsSupported: []string{"header"},
})
http.Handle("/.well-known/oauth-protected-resource", metadataHandler)
```

### Auth Error Types

- `auth.ErrInvalidToken` — token verification failed (→ 401)
- `auth.ErrOAuth` — OAuth protocol error

---

## Authentication — Client Side

### OAuth Handler Interface

```go
type OAuthHandler interface {
    TokenSource(ctx context.Context) (oauth2.TokenSource, error)
    Authorize(ctx context.Context, req *http.Request, resp *http.Response) error
}
```

Set on `StreamableClientTransport`:

```go
transport := &mcp.StreamableClientTransport{
    Endpoint:    "https://api.example.com/mcp",
    OAuthHandler: myOAuthHandler,
}
```

The transport automatically:
1. Calls `TokenSource()` before each request to set the `Authorization` header
2. On 401/403 responses, calls `Authorize()` then retries the request once

### Built-in Authorization Code Handler

```go
import "github.com/modelcontextprotocol/go-sdk/auth"

oauthHandler := auth.NewAuthorizationCodeHandler(auth.AuthorizationCodeHandlerOptions{
    // ... OAuth configuration
})

transport := &mcp.StreamableClientTransport{
    Endpoint:     "https://api.example.com/mcp",
    OAuthHandler: oauthHandler,
}
```

Supports:
- **Client ID Metadata** (draft-parecki-oauth-client-id-metadata-document)
- **Pre-registered clients** (known client_id + client_secret)
- **Dynamic Client Registration** (RFC 7591)
- Automatic token refresh
- Step-up authentication (re-auth on 403)

### Enterprise Auth (SEP-990)

For OIDC Login → Token Exchange → JWT Bearer Grant flows:

```go
import "github.com/modelcontextprotocol/go-sdk/extauth"

handler := extauth.EnterpriseHandler{
    // OIDC and token exchange configuration
}
transport := &mcp.StreamableClientTransport{
    Endpoint:     "https://api.example.com/mcp",
    OAuthHandler: &handler,
}
```

---

## Sampling (Server → Client LLM Requests)

Servers can request LLM completions from the client. The client must set a handler:

### Client Side (Handler)

```go
client := mcp.NewClient(impl, &mcp.ClientOptions{
    // Basic sampling (single content block response)
    CreateMessageHandler: func(ctx context.Context, req *mcp.CreateMessageRequest) (*mcp.CreateMessageResult, error) {
        // req.Params.Messages — conversation messages
        // req.Params.MaxTokens — requested max tokens
        // req.Params.ModelPreferences — model selection hints
        // req.Params.SystemPrompt — optional system prompt

        // Call your LLM here...
        return &mcp.CreateMessageResult{
            Content: &mcp.TextContent{Text: "LLM response"},
            Model:   "claude-3-5-sonnet",
            Role:    "assistant",
        }, nil
    },

    // OR: Sampling with tools support (parallel tool calls, array content)
    CreateMessageWithToolsHandler: func(ctx context.Context, req *mcp.CreateMessageWithToolsRequest) (*mcp.CreateMessageWithToolsResult, error) {
        // req.Params.Tools — available tools for the model
        // req.Params.ToolChoice — how model should use tools ("auto"/"required"/"none")
        // req.Params.Messages — supports ToolUseContent and ToolResultContent
        return &mcp.CreateMessageWithToolsResult{
            Content: []mcp.Content{
                &mcp.ToolUseContent{ID: "call_1", Name: "search", Input: map[string]any{"q": "test"}},
            },
            Model:      "claude-3-5-sonnet",
            Role:       "assistant",
            StopReason: "toolUse",
        }, nil
    },
    // NOTE: Cannot set both CreateMessageHandler and CreateMessageWithToolsHandler (panics)
})
```

### Server Side (Calling)

```go
// In a tool handler or elsewhere with access to the session:
result, err := req.Session.CreateMessage(ctx, &mcp.CreateMessageParams{
    Messages: []*mcp.SamplingMessage{
        {Role: "user", Content: &mcp.TextContent{Text: "Summarize this document"}},
    },
    MaxTokens: 1000,
    ModelPreferences: &mcp.ModelPreferences{
        Hints: []mcp.ModelHint{{Name: "claude"}},
        CostPriority:         0.3,
        SpeedPriority:        0.5,
        IntelligencePriority: 0.8,
    },
    SystemPrompt: "You are a helpful assistant.",
})

// With tools:
result, err := req.Session.CreateMessageWithTools(ctx, &mcp.CreateMessageWithToolsParams{
    Messages: []*mcp.SamplingMessageV2{ /* ... */ },
    MaxTokens: 1000,
    Tools:      []*mcp.Tool{{Name: "search", Description: "web search"}},
    ToolChoice: &mcp.ToolChoice{Mode: "auto"},
})
```

### Capabilities

Setting `CreateMessageHandler` → auto-advertises `sampling` capability.
Setting `CreateMessageWithToolsHandler` → auto-advertises `sampling` with `tools` support.

Override:
```go
Capabilities: &mcp.ClientCapabilities{
    Sampling: &mcp.SamplingCapabilities{
        Tools:   &mcp.SamplingToolsCapabilities{},
        Context: &mcp.SamplingContextCapabilities{},
    },
},
```

---

## Elicitation (Server → Client User Input)

Servers can request information from users through the client.

### Client Side (Handler)

```go
client := mcp.NewClient(impl, &mcp.ClientOptions{
    ElicitationHandler: func(ctx context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
        // req.Params.Message — what to ask the user
        // req.Params.RequestedSchema — JSON Schema for expected response
        // Present a form/dialog to the user...
        return &mcp.ElicitResult{
            Action:  "accept",  // "accept", "decline", or "cancel"
            Content: map[string]any{"name": "Alice", "age": 30},
        }, nil
    },
})
```

### Server Side (Calling)

```go
// Form-based elicitation
result, err := req.Session.Elicit(ctx, &mcp.ElicitParams{
    Message:         "Please provide your configuration",
    RequestedSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "name": map[string]any{"type": "string", "description": "Your name"},
            "age":  map[string]any{"type": "integer"},
        },
        "required": []string{"name"},
    },
})
if result.Action == "accept" {
    name := result.Content["name"].(string)
}
```

### URL Elicitation

For redirect-based flows (e.g., OAuth consent pages):

```go
// Server returns this error to trigger URL elicitation
return nil, zero, mcp.URLElicitationRequiredError([]*mcp.ElicitParams{
    {
        Message:       "Please authorize access",
        ElicitationID: "auth-flow-1",
        // URL provided via the error mechanism
    },
})
```

### Capabilities

```go
// Form + URL elicitation
Capabilities: &mcp.ClientCapabilities{
    Elicitation: &mcp.ElicitationCapabilities{
        Form: &mcp.FormElicitationCapabilities{},
        URL:  &mcp.URLElicitationCapabilities{},
    },
},
```

---

## Progress Notifications

### Server → Client (during tool execution)

```go
mcp.AddTool(server, &mcp.Tool{Name: "process"}, func(ctx context.Context, req *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
    token := req.Params.GetProgressToken()
    if token != nil {
        for i := 0; i < 100; i += 10 {
            req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
                ProgressToken: token,
                Progress:      float64(i),
                Total:         100,
                Message:       fmt.Sprintf("Processing %d%%", i),
            })
        }
    }
    return nil, Output{Done: true}, nil
})
```

### Client → Server

```go
// Listen for progress on the client
client := mcp.NewClient(impl, &mcp.ClientOptions{
    ProgressNotificationHandler: func(ctx context.Context, req *mcp.ProgressNotificationClientRequest) {
        fmt.Printf("Progress: %.0f/%.0f - %s\n",
            req.Params.Progress, req.Params.Total, req.Params.Message)
    },
})

// Set a progress token on the request
params := &mcp.CallToolParams{Name: "process", Arguments: map[string]any{}}
params.SetProgressToken("my-token-123")
result, err := session.CallTool(ctx, params)
```

### Client → Server Progress (reverse direction)

Clients can also send progress for server-initiated requests:

```go
session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
    ProgressToken: token,
    Progress:      50,
    Total:         100,
})
```

---

## Logging (Server → Client)

### Low-Level API

```go
// In a handler with access to the session:
req.Session.Log(ctx, &mcp.LoggingMessageParams{
    Level: "info",
    Data:  "Processing request",
})
```

### slog Integration (Recommended)

```go
mcp.AddTool(server, &mcp.Tool{Name: "process"}, func(ctx context.Context, req *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
    logger := slog.New(mcp.NewLoggingHandler(req.Session, &mcp.LoggingHandlerOptions{
        MinInterval: 100 * time.Millisecond,  // rate limit log messages
    }))
    logger.Info("starting processing", "input", in.Query)
    logger.Warn("slow query detected", "duration", elapsed)
    return nil, result, nil
})
```

### Client-Side Log Reception

```go
client := mcp.NewClient(impl, &mcp.ClientOptions{
    LoggingMessageHandler: func(ctx context.Context, req *mcp.LoggingMessageRequest) {
        fmt.Printf("[%s] %v\n", req.Params.Level, req.Params.Data)
    },
})

// Set the logging level on the server
session.SetLoggingLevel(ctx, &mcp.SetLoggingLevelParams{Level: "warning"})
```

### Logging Levels

Maps to syslog severities (RFC 5424):

| MCP Level | slog Equivalent |
|-----------|----------------|
| `debug` | `slog.LevelDebug` |
| `info` | `slog.LevelInfo` |
| `warning` | `slog.LevelWarn` |
| `error` | `slog.LevelError` |
| `critical` | `slog.LevelError + 4` |
| `alert` | `slog.LevelError + 8` |
| `emergency` | `slog.LevelError + 12` |

---

## Distributed / Stateless Servers

For load-balanced deployments where requests can hit any server instance:

```go
cache := mcp.NewSchemaCache()  // shared across requests, avoids repeated reflection

handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
    // Create a fresh server per request
    s := mcp.NewServer(&mcp.Implementation{Name: "distributed"}, &mcp.ServerOptions{
        SchemaCache: cache,
    })
    mcp.AddTool(s, myTool, myHandler)
    return s
}, &mcp.StreamableHTTPOptions{
    Stateless: true,  // no session validation
})

// Use behind a reverse proxy / load balancer
http.Handle("/mcp", handler)
```

### Fork-and-Exec Pattern

For process-per-request isolation (see `examples/server/distributed/`):

```go
handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
    // Fork a worker process and proxy the request
    return newWorkerServer()
}, &mcp.StreamableHTTPOptions{Stateless: true})
```

### Custom Session IDs

For distributed systems that need deterministic session IDs:

```go
&mcp.ServerOptions{
    GetSessionID: func() string {
        return generateGloballyUniqueID()  // must be unique across all instances
    },
}
```

Return empty string to omit the `Mcp-Session-Id` header entirely.

---

## Security

### DNS Rebinding Protection

Enabled by default. Requests arriving via localhost that have a non-localhost `Host` header are rejected with 403.

Disable (dangerous, only if you understand the implications):
```go
&mcp.StreamableHTTPOptions{
    DisableLocalhostProtection: true,
}
```

Or via environment variable: `MCPGODEBUG=disablelocalhostprotection`

### Cross-Origin (CSRF) Protection

Enabled by default in v1.4.1+. Customize or disable:
```go
&mcp.StreamableHTTPOptions{
    CrossOriginProtection: /* custom config */,
}
```

Or disable via: `MCPGODEBUG=disablecrossoriginprotection`

### Session Hijacking Prevention

When `TokenInfo.UserID` is set by the token verifier, the transport ensures all requests for a given session come from the same user.

### HTTPS Enforcement

The auth package enforces HTTPS for OAuth flows. Use appropriate TLS configuration in production.

---

## Debugging

### MCPGODEBUG Environment Variable

Controls backward-compatible behavior changes:

| Value | Version | Effect |
|-------|---------|--------|
| `disablecrossoriginprotection` | v1.4.1 | Disable CSRF/cross-origin checks |
| `jsonescaping` | v1.4.0 | Use Go's standard JSON escaping |
| `disablelocalhostprotection` | v1.4.0 | Disable DNS rebinding checks |

Multiple values: `MCPGODEBUG=jsonescaping,disablelocalhostprotection`

Options are removed after 2 minor versions.

### LoggingTransport

Wraps any transport to log all JSON-RPC traffic:

```go
transport := &mcp.LoggingTransport{
    Transport: &mcp.StdioTransport{},
    Writer:    os.Stderr,  // or a file
}
```

### MCP Inspector

Use the official [MCP Inspector](https://github.com/modelcontextprotocol/inspector) to interactively test servers:

```bash
npx @anthropic/mcp-inspector my-server-binary
```

### HTTP Traffic Inspection

For streamable HTTP servers, use standard HTTP debugging tools:
- `curl` for manual requests
- Wireshark/tcpdump for traffic capture
- HTTP middleware for request/response logging:

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        slog.Info("request", "method", r.Method, "path", r.URL.Path,
            "session", r.Header.Get("Mcp-Session-Id"))
        next.ServeHTTP(w, r)
    })
}
```

---

## Extensions

Both `ServerCapabilities` and `ClientCapabilities` support custom extensions:

```go
caps := &mcp.ServerCapabilities{}
caps.AddExtension("myvendor/custom-feature", map[string]any{
    "version": "1.0",
    "enabled": true,
})
```

---

## Known API Rough Edges (v1.x)

These are known issues that cannot be fixed without breaking backward compatibility:

1. **`ClientCapabilities.Roots` should be a pointer** — Use `RootsV2` instead:
   ```go
   // Wrong (can't distinguish "no roots" from "roots without listChanged")
   Capabilities: &mcp.ClientCapabilities{Roots: struct{ListChanged bool}{}},
   // Correct
   Capabilities: &mcp.ClientCapabilities{RootsV2: &mcp.RootCapabilities{ListChanged: true}},
   ```

2. **Default capabilities should be empty** — For historical reasons, servers default to `{"logging":{}}` and clients default to `{"roots":{"listChanged":true}}`. Set explicit `Capabilities` to override.

3. **`ProgressNotificationParams` naming** — Should be `ProgressParams` per naming conventions, but cannot rename.

4. **`EventStore.Open` is unnecessary** — The `Open` method exists but doesn't provide useful functionality.

---

## Complete Example: HTTP Server with Auth + Tools

```go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "time"

    "github.com/modelcontextprotocol/go-sdk/auth"
    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/modelcontextprotocol/go-sdk/oauthex"
)

type TimeInput struct {
    Timezone string `json:"timezone" jsonschema:"IANA timezone name,default=UTC"`
}

type TimeOutput struct {
    Time     string `json:"time"`
    Timezone string `json:"timezone"`
}

func main() {
    server := mcp.NewServer(&mcp.Implementation{
        Name:    "time-server",
        Title:   "Time Server",
        Version: "v1.0.0",
    }, &mcp.ServerOptions{
        Instructions: "Use the get_time tool to get the current time in any timezone.",
        Logger:       slog.Default(),
    })

    mcp.AddTool(server, &mcp.Tool{
        Name:        "get_time",
        Description: "Get the current time in a timezone",
        Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
    }, func(ctx context.Context, req *mcp.CallToolRequest, in TimeInput) (*mcp.CallToolResult, TimeOutput, error) {
        loc, err := time.LoadLocation(in.Timezone)
        if err != nil {
            return nil, TimeOutput{}, err
        }
        now := time.Now().In(loc)
        return nil, TimeOutput{Time: now.Format(time.RFC3339), Timezone: in.Timezone}, nil
    })

    // Token verifier
    verifier := func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
        if token == "valid-token" {
            return &auth.TokenInfo{UserID: "user-1"}, nil
        }
        return nil, auth.ErrInvalidToken
    }

    // MCP handler
    mcpHandler := mcp.NewStreamableHTTPHandler(
        func(r *http.Request) *mcp.Server { return server },
        &mcp.StreamableHTTPOptions{Logger: slog.Default()},
    )

    // Protected resource metadata
    http.Handle("/.well-known/oauth-protected-resource",
        auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
            Resource:             "https://example.com/mcp",
            AuthorizationServers: []string{"https://auth.example.com"},
        }))

    // MCP endpoint with auth
    http.Handle("/mcp", auth.RequireBearerToken(verifier, nil)(mcpHandler))

    slog.Info("starting server", "addr", ":8080")
    http.ListenAndServe(":8080", nil)
}
```

---

## Package Summary

| Package | Import Path | Purpose |
|---------|-------------|---------|
| `mcp` | `github.com/modelcontextprotocol/go-sdk/mcp` | Primary API: Server, Client, Transport, Tools, Prompts, Resources |
| `auth` | `github.com/modelcontextprotocol/go-sdk/auth` | Server-side Bearer token middleware, client-side OAuthHandler interface |
| `jsonrpc` | `github.com/modelcontextprotocol/go-sdk/jsonrpc` | Low-level JSON-RPC 2.0 types for custom transports |
| `oauthex` | `github.com/modelcontextprotocol/go-sdk/oauthex` | OAuth extensions (ProtectedResourceMetadata, etc.) |
| `extauth` | `github.com/modelcontextprotocol/go-sdk/extauth` | Enterprise auth (OIDC + token exchange, SEP-990) |
