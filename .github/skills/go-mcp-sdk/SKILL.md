---
name: go-mcp-sdk
description: "Go MCP SDK (github.com/modelcontextprotocol/go-sdk) v1.4.1 — creating MCP servers and clients in Go: tools, prompts, resources, transports (stdio, streamable HTTP, SSE), middleware, pagination, testing. Use when: building MCP servers or clients in Go, adding tools/prompts/resources, choosing transports, writing MCP tests."
---

# Go MCP SDK — Core Usage

Official Go SDK for the Model Context Protocol. Import path: `github.com/modelcontextprotocol/go-sdk`.

- **Version**: v1.4.1 (supports MCP spec 2025-11-25)
- **Packages**: `mcp` (primary API), `auth` (server-side token verification + client-side OAuth), `jsonrpc` (custom transports), `oauthex` (OAuth extensions)
- **License**: Apache 2.0 (new contributions), MIT (existing)
- **Go requirement**: Go 1.24+

---

## Quick Start

### Minimal Stdio Server

```go
package main

import (
    "context"
    "log"

    "github.com/modelcontextprotocol/go-sdk/mcp"
)

type GreetArgs struct {
    Name string `json:"name" jsonschema:"the person to greet"`
}

func main() {
    server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v1.0.0"}, nil)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "greet",
        Description: "say hello",
    }, func(ctx context.Context, req *mcp.CallToolRequest, args GreetArgs) (*mcp.CallToolResult, any, error) {
        return &mcp.CallToolResult{
            Content: []mcp.Content{&mcp.TextContent{Text: "Hello, " + args.Name + "!"}},
        }, nil, nil
    })

    if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
        log.Fatal(err)
    }
}
```

### Minimal Client Connecting to a Stdio Server

```go
package main

import (
    "context"
    "log"
    "os/exec"

    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
    client := mcp.NewClient(&mcp.Implementation{Name: "my-client", Version: "v1.0.0"}, nil)
    transport := &mcp.CommandTransport{Command: exec.Command("./myserver")}

    session, err := client.Connect(context.Background(), transport, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer session.Close()

    result, err := session.CallTool(ctx, &mcp.CallToolParams{
        Name:      "greet",
        Arguments: map[string]any{"name": "World"},
    })
    if err != nil {
        log.Fatal(err)
    }
    // result.Content contains the response
}
```

---

## Architecture: Clients, Servers, and Sessions

```
Client                                                   Server
  ⇅                    (jsonrpc2)                          ⇅
ClientSession ⇄ Client Transport ⇄ Server Transport ⇄ ServerSession
```

- **`Client`** / **`Server`**: Long-lived objects that handle many concurrent connections.
- **`ClientSession`** / **`ServerSession`**: Created per connection via `Connect()`. Expose the API to interact with the peer.
- A single `Server` can serve many `ServerSession`s simultaneously (e.g., via `StreamableHTTPHandler`).
- `Server.Run()` is a convenience for single-session servers (stdio). It blocks until the client disconnects.

### Lifecycle

```go
// Server side
server := mcp.NewServer(&mcp.Implementation{Name: "myserver"}, opts)
// Add features...
session, err := server.Connect(ctx, transport, nil)  // or server.Run(ctx, transport)
session.Wait()  // wait for client to disconnect

// Client side
client := mcp.NewClient(&mcp.Implementation{Name: "myclient"}, opts)
session, err := client.Connect(ctx, transport, nil)
defer session.Close()
```

---

## Tools

### Generic `AddTool` (Recommended)

The top-level `mcp.AddTool` function provides automatic schema inference, input validation, output marshaling, and error handling:

```go
type CalculateInput struct {
    Expression string `json:"expression" jsonschema:"the math expression to evaluate"`
}

type CalculateOutput struct {
    Result float64 `json:"result"`
}

mcp.AddTool(server, &mcp.Tool{
    Name:        "calculate",
    Description: "Evaluate a math expression",
}, func(ctx context.Context, req *mcp.CallToolRequest, in CalculateInput) (*mcp.CallToolResult, CalculateOutput, error) {
    result := evaluate(in.Expression)
    return nil, CalculateOutput{Result: result}, nil
})
```

**Key behaviors of `mcp.AddTool`:**
- **Input schema**: Auto-inferred from the `In` type using `jsonschema` struct tags. Can be overridden by setting `Tool.InputSchema`.
- **Output schema**: Auto-inferred from the `Out` type. Use `any` as Out type to skip output schema.
- **Input validation**: Arguments are automatically unmarshaled and validated against the schema.
- **Error handling**: If the handler returns an error, `IsError` is set to true and error text is included in `Content`.
- **Structured output**: The `Out` value is automatically set as `StructuredContent` and a JSON text `Content` item.
- **`CallToolResult`**: Can be nil — the SDK populates it from the Out value. Or return a non-nil result to override Content/StructuredContent.

### Handler Signature

```go
type ToolHandlerFor[In, Out any] func(
    ctx context.Context,
    request *mcp.CallToolRequest,
    input In,
) (result *mcp.CallToolResult, output Out, _ error)
```

- `In` must be a struct or map (JSON Schema type "object").
- `Out` can be any type. Use `any` to skip structured output.
- Return `(nil, out, nil)` for the simplest case — SDK builds the result.
- Return `(result, out, nil)` to customize Content alongside structured output.
- Return `(nil, zero, err)` for errors — SDK sets `IsError: true`.

### Low-Level `Server.AddTool`

For full control (no auto-validation, no schema inference):

```go
server.AddTool(&mcp.Tool{
    Name:        "raw_tool",
    InputSchema: json.RawMessage(`{"type":"object","properties":{"x":{"type":"number"}}}`),
}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // req.Params.Arguments is json.RawMessage — unmarshal manually
    var args map[string]any
    json.Unmarshal(req.Params.Arguments, &args)
    return &mcp.CallToolResult{
        Content: []mcp.Content{&mcp.TextContent{Text: "done"}},
    }, nil
})
```

### Tool Annotations

```go
mcp.AddTool(server, &mcp.Tool{
    Name: "search",
    Annotations: &mcp.ToolAnnotations{
        ReadOnlyHint:    true,                // does not modify environment
        OpenWorldHint:   boolPtr(true),       // interacts with external entities
        IdempotentHint:  false,               // repeated calls may differ
        DestructiveHint: boolPtr(false),      // no destructive updates
        Title:           "Search the web",    // human-readable title
    },
}, handler)
```

### Tool Icons

```go
mcp.AddTool(server, &mcp.Tool{
    Name: "search",
    Icons: []mcp.Icon{{
        Source:   "https://example.com/icon.png",
        MIMEType: "image/png",
        Theme:    mcp.IconThemeDark,
    }},
}, handler)
```

### Dynamic Tool Management

```go
server.RemoveTools("old_tool")                    // remove by name
// All connected sessions receive notifications/tools/list_changed
```

### Resource Links in Tool Results

Tools can return resource links to reference server resources:

```go
return &mcp.CallToolResult{
    Content: []mcp.Content{
        &mcp.ResourceLink{
            URI:      "file:///path/to/result.txt",
            Name:     "result",
            MIMEType: "text/plain",
        },
    },
}, nil, nil
```

---

## Prompts

```go
server.AddPrompt(&mcp.Prompt{
    Name:        "code_review",
    Description: "Review code for issues",
    Arguments: []*mcp.PromptArgument{
        {Name: "code", Description: "The code to review", Required: true},
        {Name: "language", Description: "Programming language"},
    },
}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
    code := req.Params.Arguments["code"]
    return &mcp.GetPromptResult{
        Description: "Code review prompt",
        Messages: []*mcp.PromptMessage{
            {
                Role:    "user",
                Content: &mcp.TextContent{Text: "Review this code:\n" + code},
            },
        },
    }, nil
})
```

Client-side:
```go
// Paginated iterator (handles pagination automatically)
for prompt, err := range session.Prompts(ctx, nil) {
    if err != nil { break }
    fmt.Println(prompt.Name)
}

// Get a specific prompt
result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
    Name:      "code_review",
    Arguments: map[string]string{"code": "func main() {}", "language": "go"},
})
```

Dynamic management:
```go
server.RemovePrompts("old_prompt")
```

---

## Resources

### Static Resources

```go
server.AddResource(&mcp.Resource{
    URI:         "file:///config.json",
    Name:        "config",
    Description: "Application configuration",
    MIMEType:    "application/json",
}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
    data, _ := os.ReadFile("/config.json")
    return &mcp.ReadResourceResult{
        Contents: []*mcp.ResourceContents{{
            URI:      req.Params.URI,
            MIMEType: "application/json",
            Text:     string(data),
        }},
    }, nil
})
```

### Resource Templates (URI Templates per RFC 6570)

```go
server.AddResourceTemplate(&mcp.ResourceTemplate{
    URITemplate: "file:///users/{id}/profile",
    Name:        "user_profile",
    Description: "User profile by ID",
    MIMEType:    "application/json",
}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
    // req.Params.URI contains the resolved URI, e.g. "file:///users/123/profile"
    // Extract ID from URI and fetch data
    return &mcp.ReadResourceResult{
        Contents: []*mcp.ResourceContents{{
            URI:  req.Params.URI,
            Text: `{"name": "Alice"}`,
        }},
    }, nil
})
```

### Resource Not Found

```go
func handler(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
    if !exists(req.Params.URI) {
        return nil, mcp.ResourceNotFoundError(req.Params.URI)
    }
    // ...
}
```

### Resource Subscriptions

Enable via `ServerOptions.SubscribeHandler`:

```go
server := mcp.NewServer(impl, &mcp.ServerOptions{
    SubscribeHandler:   func(ctx context.Context, req *mcp.SubscribeRequest) error { return nil },
    UnsubscribeHandler: func(ctx context.Context, req *mcp.UnsubscribeRequest) error { return nil },
})

// Notify clients when a resource changes
server.ResourceUpdated(ctx, &mcp.ResourceUpdatedNotificationParams{URI: "file:///config.json"})
```

Client-side:
```go
session.Subscribe(ctx, &mcp.SubscribeParams{URI: "file:///config.json"})
session.Unsubscribe(ctx, &mcp.UnsubscribeParams{URI: "file:///config.json"})

// Paginated iterators
for resource, err := range session.Resources(ctx, nil) { /* ... */ }
for tmpl, err := range session.ResourceTemplates(ctx, nil) { /* ... */ }
result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "file:///config.json"})
```

Dynamic management:
```go
server.RemoveResources("file:///old")
server.RemoveResourceTemplates("file:///old/{id}")
```

---

## Transports

### Stdio Transport (Server Side)

For CLI tools invoked by a host process:

```go
server.Run(ctx, &mcp.StdioTransport{})
```

### Command Transport (Client Side)

Launches a server as a subprocess and communicates over stdin/stdout:

```go
transport := &mcp.CommandTransport{
    Command:           exec.Command("./myserver", "--flag"),
    TerminateDuration: 10 * time.Second,  // wait before SIGTERM (default 5s)
}
session, err := client.Connect(ctx, transport, nil)
```

### IO Transport

For custom I/O (e.g., pipes, network connections):

```go
transport := &mcp.IOTransport{
    Reader: myReadCloser,
    Writer: myWriteCloser,
}
```

### Streamable HTTP Transport (Server Side)

The recommended transport for HTTP-based servers. Implements `http.Handler`:

```go
handler := mcp.NewStreamableHTTPHandler(
    func(r *http.Request) *mcp.Server {
        return server  // can return different servers per request
    },
    &mcp.StreamableHTTPOptions{
        // All options are optional
        Logger:         slog.Default(),
        SessionTimeout: 30 * time.Minute,
    },
)
http.Handle("/mcp", handler)
http.ListenAndServe(":8080", nil)
```

### Streamable HTTP Transport (Client Side)

```go
transport := &mcp.StreamableClientTransport{
    Endpoint:   "http://localhost:8080/mcp",
    HTTPClient: http.DefaultClient,             // optional, defaults to http.DefaultClient
    MaxRetries: 5,                              // reconnect attempts (default 5, negative to disable)
}
session, err := client.Connect(ctx, transport, nil)
```

**`DisableStandaloneSSE`**: Set to `true` to skip the persistent SSE connection for server-initiated messages. Useful when you only need request-response communication.

### Legacy SSE Transport (Deprecated)

For compatibility with the 2024-11-05 spec:

```go
// Server
sseHandler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server { return server }, nil)
http.Handle("/sse", sseHandler)

// Client
transport := &mcp.SSEClientTransport{Endpoint: "http://localhost:8080/sse"}
```

### Logging Transport (Debugging)

Wraps any transport to log all JSON-RPC messages:

```go
transport := &mcp.LoggingTransport{
    Transport: &mcp.StdioTransport{},
    Writer:    os.Stderr,
}
server.Run(ctx, transport)
```

### In-Memory Transports (Testing)

```go
serverTransport, clientTransport := mcp.NewInMemoryTransports()

// Connect server first, then client
serverSession, err := server.Connect(ctx, serverTransport, nil)
clientSession, err := client.Connect(ctx, clientTransport, nil)
```

---

## StreamableHTTPOptions Reference

```go
type StreamableHTTPOptions struct {
    Stateless      bool              // No session validation; temporary sessions per request
    JSONResponse   bool              // Return application/json instead of text/event-stream
    Logger         *slog.Logger
    EventStore     EventStore        // Enables stream resumption (SDK provides MemoryEventStore)
    SessionTimeout time.Duration     // Auto-close idle sessions (zero = never)

    DisableLocalhostProtection bool  // Disable DNS rebinding protection (dangerous)
    CrossOriginProtection      *...  // Customize cross-origin protection
}
```

### Stateless Mode

For distributed/load-balanced deployments where any server can handle any request:

```go
handler := mcp.NewStreamableHTTPHandler(
    func(r *http.Request) *mcp.Server {
        return mcp.NewServer(impl, opts)  // new server per request
    },
    &mcp.StreamableHTTPOptions{Stateless: true},
)
```

In stateless mode:
- No `Mcp-Session-Id` validation
- Server-to-client requests are rejected (no persistent connection)
- Server-to-client notifications work only within the context of an incoming request

### Stream Resumability

```go
handler := mcp.NewStreamableHTTPHandler(getServer, &mcp.StreamableHTTPOptions{
    EventStore: &mcp.MemoryEventStore{},  // for testing; implement EventStore for production
})
```

---

## Middleware

Both `Server` and `Client` support receiving and sending middleware:

```go
// Receiving middleware — intercepts incoming messages
server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
    return func(ctx context.Context, method string, params any) (any, error) {
        slog.Info("received", "method", method)
        start := time.Now()
        result, err := next(ctx, method, params)
        slog.Info("handled", "method", method, "duration", time.Since(start))
        return result, err
    }
})

// Sending middleware — intercepts outgoing messages
server.AddSendingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
    return func(ctx context.Context, method string, params any) (any, error) {
        slog.Info("sending", "method", method)
        return next(ctx, method, params)
    }
})
```

### Rate Limiting via Middleware

```go
limiter := rate.NewLimiter(rate.Every(time.Second), 10)

server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
    return func(ctx context.Context, method string, params any) (any, error) {
        if !limiter.Allow() {
            return nil, &jsonrpc.Error{Code: -32000, Message: "rate limited"}
        }
        return next(ctx, method, params)
    }
})
```

### Per-Method Middleware

```go
server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
    return func(ctx context.Context, method string, params any) (any, error) {
        if method == "tools/call" {
            // special handling for tool calls
        }
        return next(ctx, method, params)
    }
})
```

### HTTP-Level Middleware (for StreamableHTTPHandler)

Since `StreamableHTTPHandler` implements `http.Handler`, wrap it with standard HTTP middleware:

```go
handler := mcp.NewStreamableHTTPHandler(getServer, opts)
http.Handle("/mcp", loggingMiddleware(authMiddleware(handler)))
```

---

## Server Options

```go
server := mcp.NewServer(&mcp.Implementation{
    Name:    "myserver",
    Title:   "My MCP Server",      // human-readable, for UI display
    Version: "v1.0.0",
    Icons:   []mcp.Icon{{Source: "https://example.com/icon.png"}},
}, &mcp.ServerOptions{
    Instructions: "Use the search tool to find information.",  // hint for LLMs
    Logger:       slog.Default(),
    PageSize:     100,                         // pagination size (default 1000)
    KeepAlive:    30 * time.Second,            // auto-ping interval
    Capabilities: &mcp.ServerCapabilities{},   // explicit capabilities (nil = defaults)

    // Handlers
    InitializedHandler: func(ctx context.Context, req *mcp.InitializedRequest) { /* ... */ },
    SubscribeHandler:   func(ctx context.Context, req *mcp.SubscribeRequest) error { return nil },
    UnsubscribeHandler: func(ctx context.Context, req *mcp.UnsubscribeRequest) error { return nil },
    CompletionHandler:  func(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) { /* ... */ },

    // For stateless/distributed deployments
    SchemaCache:  mcp.NewSchemaCache(),        // cache schemas across server instances
    GetSessionID: func() string { return "" }, // custom session ID generation
})
```

### Capabilities

Capabilities are **auto-inferred** when you add features:
- Adding tools → advertises `tools` capability with `listChanged: true`
- Adding prompts → advertises `prompts` capability with `listChanged: true`
- Adding resources → advertises `resources` capability with `listChanged: true`
- Setting `CompletionHandler` → advertises `completions` capability
- Default: `logging` capability enabled (set `Capabilities: &mcp.ServerCapabilities{}` to disable)

Override inferred capabilities:
```go
Capabilities: &mcp.ServerCapabilities{
    Tools:     &mcp.ToolCapabilities{ListChanged: false},  // disable list change notifications
    Prompts:   &mcp.PromptCapabilities{ListChanged: true},
    Resources: &mcp.ResourceCapabilities{Subscribe: true, ListChanged: true},
    Logging:   &mcp.LoggingCapabilities{},
},
```

---

## Client Options

```go
client := mcp.NewClient(&mcp.Implementation{Name: "myclient", Version: "v1.0.0"}, &mcp.ClientOptions{
    Logger:    slog.Default(),
    KeepAlive: 30 * time.Second,

    // Capabilities (default: roots with listChanged:true; set &ClientCapabilities{} to disable)
    Capabilities: &mcp.ClientCapabilities{
        RootsV2: &mcp.RootCapabilities{ListChanged: true},  // use RootsV2, not Roots
    },

    // Notification handlers
    ToolListChangedHandler:     func(ctx context.Context, req *mcp.ToolListChangedRequest) { /* ... */ },
    PromptListChangedHandler:   func(ctx context.Context, req *mcp.PromptListChangedRequest) { /* ... */ },
    ResourceListChangedHandler: func(ctx context.Context, req *mcp.ResourceListChangedRequest) { /* ... */ },
    ResourceUpdatedHandler:     func(ctx context.Context, req *mcp.ResourceUpdatedNotificationRequest) { /* ... */ },
    LoggingMessageHandler:      func(ctx context.Context, req *mcp.LoggingMessageRequest) { /* ... */ },
    ProgressNotificationHandler: func(ctx context.Context, req *mcp.ProgressNotificationClientRequest) { /* ... */ },
})
```

---

## Pagination

Server pagination is on by default (default page size: 1000, configurable via `ServerOptions.PageSize`).

### Client-Side Iterators (Recommended)

```go
for tool, err := range session.Tools(ctx, nil) {
    if err != nil { log.Fatal(err) }
    fmt.Println(tool.Name)
}

for prompt, err := range session.Prompts(ctx, nil) { /* ... */ }
for resource, err := range session.Resources(ctx, nil) { /* ... */ }
for tmpl, err := range session.ResourceTemplates(ctx, nil) { /* ... */ }
```

### Manual Pagination

```go
result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
for result.NextCursor != "" {
    result, err = session.ListTools(ctx, &mcp.ListToolsParams{Cursor: result.NextCursor})
}
```

---

## Ping and KeepAlive

```go
// Manual ping
err := session.Ping(ctx, nil)

// Auto-ping: set KeepAlive on ServerOptions or ClientOptions
// If peer fails to respond, session is automatically closed
```

---

## Cancellation

Context cancellation automatically sends `notifications/cancelled` to the peer:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
result, err := session.CallTool(ctx, params)  // cancellation propagated to server
```

---

## Content Types

The `Content` interface is implemented by:

| Type | Use Case |
|------|----------|
| `*mcp.TextContent` | Plain text |
| `*mcp.ImageContent` | Base64 image with MIME type |
| `*mcp.AudioContent` | Base64 audio with MIME type |
| `*mcp.ResourceLink` | Link to a server resource |
| `*mcp.EmbeddedResource` | Inline resource content |
| `*mcp.ToolUseContent` | Tool invocation request (sampling only) |
| `*mcp.ToolResultContent` | Tool invocation result (sampling only) |

---

## Testing

### In-Memory Transport

```go
func TestMyTool(t *testing.T) {
    server := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
    mcp.AddTool(server, &mcp.Tool{Name: "double"}, doubleHandler)

    client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
    st, ct := mcp.NewInMemoryTransports()

    // Connect server first, then client
    _, err := server.Connect(context.Background(), st, nil)
    if err != nil { t.Fatal(err) }

    session, err := client.Connect(context.Background(), ct, nil)
    if err != nil { t.Fatal(err) }
    defer session.Close()

    result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
        Name:      "double",
        Arguments: map[string]any{"number": 21},
    })
    if err != nil { t.Fatal(err) }
    // assert on result.Content, result.StructuredContent, etc.
}
```

### httptest for HTTP Servers

```go
func TestHTTPServer(t *testing.T) {
    server := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
    // add tools...

    handler := mcp.NewStreamableHTTPHandler(
        func(r *http.Request) *mcp.Server { return server },
        nil,
    )
    ts := httptest.NewServer(handler)
    defer ts.Close()

    client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
    session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: ts.URL}, nil)
    // ...
}
```

---

## JSON Schema Customization

### Struct Tags

```go
type Input struct {
    Query   string   `json:"query" jsonschema:"the search query"`
    Limit   int      `json:"limit" jsonschema:"max results,default=10"`
    Tags    []string `json:"tags,omitempty"`
}
```

The SDK uses `github.com/google/jsonschema-go` (draft 2020-12) for inference.

### Custom Type Schemas

```go
import "github.com/google/jsonschema-go"

schema := jsonschema.For[Input](&jsonschema.Options{
    TypeSchemas: map[reflect.Type]*jsonschema.Schema{
        reflect.TypeOf(MyCustomType{}): {Type: "string", Format: "date-time"},
    },
})
```

### Schema Cache (for Stateless Servers)

In stateless deployments, a new `Server` is created per request. `SchemaCache` avoids repeated reflection:

```go
cache := mcp.NewSchemaCache()

handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
    s := mcp.NewServer(impl, &mcp.ServerOptions{SchemaCache: cache})
    mcp.AddTool(s, tool, handler)  // schema computed once, cached
    return s
}, &mcp.StreamableHTTPOptions{Stateless: true})
```

---

## Roots (Client Feature)

Clients expose filesystem roots to servers:

```go
client.AddRoots(&mcp.Root{
    URI:  "file:///home/user/project",
    Name: "My Project",
})
client.RemoveRoots("file:///home/user/old-project")
// Connected servers receive notifications/roots/list_changed
```

Server-side:
```go
server := mcp.NewServer(impl, &mcp.ServerOptions{
    // Called when client sends roots/list_changed
    // (set via ClientOptions on the client side — handled automatically)
})

// In a tool handler, list roots from the client:
roots, err := req.Session.ListRoots(ctx, nil)
```

---

## Error Handling

### Tool Errors vs Protocol Errors

```go
// Tool error (visible to LLM, allows self-correction)
return &mcp.CallToolResult{
    IsError: true,
    Content: []mcp.Content{&mcp.TextContent{Text: "file not found"}},
}, nil, nil

// Or with generic AddTool, simply return an error:
return nil, zero, fmt.Errorf("file not found")
// SDK automatically sets IsError:true and includes error in Content

// Protocol error (for exceptional conditions like missing tool)
return nil, zero, &jsonrpc.Error{Code: -32602, Message: "invalid params"}
```

### CallToolResult.SetError / GetError

```go
result := &mcp.CallToolResult{}
result.SetError(fmt.Errorf("something went wrong"))
// Automatically sets IsError:true and Content with error text
```

### Resource Not Found

```go
return nil, mcp.ResourceNotFoundError("file:///missing.txt")
// Returns MCP error code -32002
```

---

## Completion (Autocompletion)

Server-side:
```go
server := mcp.NewServer(impl, &mcp.ServerOptions{
    CompletionHandler: func(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
        // req.Params.Ref — what's being completed (ref/prompt or ref/resource)
        // req.Params.Argument — the argument being completed
        // req.Params.Context — previously resolved variables
        return &mcp.CompleteResult{
            Completion: mcp.CompletionResultDetails{
                Values:  []string{"option1", "option2"},
                HasMore: false,
            },
        }, nil
    },
})
```

Client-side:
```go
result, err := session.Complete(ctx, &mcp.CompleteParams{
    Ref:      &mcp.CompleteReference{Type: "ref/prompt", Name: "my_prompt"},
    Argument: mcp.CompleteParamsArgument{Name: "language", Value: "go"},
})
```

---

## Server Sessions Iterator

Access all active sessions:

```go
for session := range server.Sessions() {
    fmt.Println("Session:", session.ID())
    // Send notifications, check state, etc.
}
```

---

## MCP Error Codes

| Code | Constant | Meaning |
|------|----------|---------|
| -32002 | `CodeResourceNotFound` | Requested resource not found |
| -32042 | `CodeURLElicitationRequired` | Server requires URL elicitation |

Standard JSON-RPC 2.0 error codes also apply (-32700, -32600, -32601, -32602, -32603).

---

## Version Compatibility

| SDK Version | MCP Spec | Notes |
|-------------|----------|-------|
| v1.4.0+ | 2025-11-25 | Full spec support, experimental client-side OAuth |
| v1.2.0–v1.3.1 | 2025-11-25 (partial) | |
| v1.0.0–v1.1.0 | 2025-06-18 | |
| v0.x | Various | Pre-stable API |
