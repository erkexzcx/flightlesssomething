---
name: mcp-server-features
description: "MCP 2025-11-25 server features — resources (list, read, templates, subscriptions, annotations), tools (list, call, inputSchema, outputSchema, annotations, structured content, tool names), prompts (list, get, arguments, messages), completion (autocomplete for prompt/resource arguments), logging (setLevel, notifications, syslog levels). Use when: implementing MCP server capabilities, adding tools/resources/prompts, handling tool calls, resource subscriptions."
---

# MCP Server Features (2025-11-25)

Servers provide three core primitives to clients, plus utility features:

| Primitive | Control | Description | Example |
|-----------|---------|-------------|---------|
| **Prompts** | User-controlled | Interactive templates invoked by user choice | Slash commands, menu options |
| **Resources** | Application-controlled | Contextual data attached and managed by the client | File contents, git history |
| **Tools** | Model-controlled | Functions exposed to the LLM to take actions | API POST requests, file writing |

---

## Resources

Resources allow servers to share data (files, database schemas, application info) identified by URIs (RFC 3986).

### Capabilities

```json
{
  "capabilities": {
    "resources": {
      "subscribe": true,
      "listChanged": true
    }
  }
}
```

Both `subscribe` and `listChanged` are optional — servers can support neither, either, or both. An empty `"resources": {}` declares resource support with no sub-features.

### Protocol Messages

**List Resources** (`resources/list`) — supports pagination:
```json
// Request
{ "method": "resources/list", "params": { "cursor": "optional-cursor" } }

// Response
{
  "result": {
    "resources": [
      {
        "uri": "file:///project/src/main.rs",
        "name": "main.rs",
        "title": "Rust Application Main File",
        "description": "Primary application entry point",
        "mimeType": "text/x-rust",
        "size": 1024,
        "icons": [{ "src": "https://example.com/icon.png", "mimeType": "image/png", "sizes": ["48x48"] }]
      }
    ],
    "nextCursor": "next-page-cursor"
  }
}
```

**Read Resource** (`resources/read`):
```json
// Request
{ "method": "resources/read", "params": { "uri": "file:///project/src/main.rs" } }

// Response — text content
{
  "result": {
    "contents": [{
      "uri": "file:///project/src/main.rs",
      "mimeType": "text/x-rust",
      "text": "fn main() {\n    println!(\"Hello world!\");\n}"
    }]
  }
}

// Response — binary content
{
  "result": {
    "contents": [{
      "uri": "file:///example.png",
      "mimeType": "image/png",
      "blob": "base64-encoded-data"
    }]
  }
}
```

**List Resource Templates** (`resources/templates/list`) — supports pagination:
```json
{
  "result": {
    "resourceTemplates": [{
      "uriTemplate": "file:///{path}",
      "name": "Project Files",
      "title": "Project Files",
      "description": "Access files in the project directory",
      "mimeType": "application/octet-stream"
    }]
  }
}
```

Templates use URI templates (RFC 6570). Arguments can be auto-completed via the completion API.

**List Changed Notification** (server→client, requires `listChanged`):
```json
{ "method": "notifications/resources/list_changed" }
```

**Subscriptions** (requires `subscribe`):
```json
// Subscribe
{ "method": "resources/subscribe", "params": { "uri": "file:///project/src/main.rs" } }

// Unsubscribe
{ "method": "resources/unsubscribe", "params": { "uri": "file:///project/src/main.rs" } }

// Update notification (server→client)
{ "method": "notifications/resources/updated", "params": { "uri": "file:///project/src/main.rs" } }
```

### Resource Data Types

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `uri` | string | Yes | Unique URI identifier |
| `name` | string | Yes | Resource name |
| `title` | string | No | Human-readable display name |
| `description` | string | No | Description |
| `icons` | Icon[] | No | Display icons |
| `mimeType` | string | No | MIME type |
| `size` | number | No | Size in bytes |

### Annotations

Resources, resource templates, and content blocks support optional annotations:

```json
{
  "annotations": {
    "audience": ["user", "assistant"],
    "priority": 0.8,
    "lastModified": "2025-01-12T15:00:58Z"
  }
}
```

- `audience`: array of `"user"` and/or `"assistant"` — who the content is for
- `priority`: 0.0 (least important) to 1.0 (most important/required)
- `lastModified`: ISO 8601 timestamp

### Common URI Schemes

- `https://` — web resources the client can fetch directly; servers SHOULD use only when client can fetch on its own
- `file://` — filesystem-like resources (don't need to map to actual files); MAY use XDG MIME types like `inode/directory`
- `git://` — git version control integration
- Custom schemes — MUST conform to RFC 3986

### Error Handling

- Resource not found: `-32002`
- Internal errors: `-32603`

### Security

- Validate all resource URIs
- Implement access controls for sensitive resources
- Properly encode binary data
- Check resource permissions before operations

---

## Tools

Tools enable models to interact with external systems (databases, APIs, computations). Each tool has a unique name and metadata describing its schema.

### Capabilities

```json
{ "capabilities": { "tools": { "listChanged": true } } }
```

### Protocol Messages

**List Tools** (`tools/list`) — supports pagination:
```json
{
  "result": {
    "tools": [{
      "name": "get_weather",
      "title": "Weather Information Provider",
      "description": "Get current weather information for a location",
      "inputSchema": {
        "type": "object",
        "properties": {
          "location": { "type": "string", "description": "City name or zip code" }
        },
        "required": ["location"]
      },
      "outputSchema": {
        "type": "object",
        "properties": {
          "temperature": { "type": "number" },
          "conditions": { "type": "string" }
        },
        "required": ["temperature", "conditions"]
      },
      "icons": [{ "src": "https://example.com/weather.png", "mimeType": "image/png", "sizes": ["48x48"] }],
      "annotations": { "readOnlyHint": true, "openWorldHint": true },
      "execution": { "taskSupport": "optional" }
    }],
    "nextCursor": "next-page-cursor"
  }
}
```

**Call Tool** (`tools/call`):
```json
// Request
{ "method": "tools/call", "params": { "name": "get_weather", "arguments": { "location": "New York" } } }

// Response
{
  "result": {
    "content": [{ "type": "text", "text": "Temperature: 72°F\nConditions: Partly cloudy" }],
    "isError": false
  }
}
```

**List Changed Notification** (server→client, requires `listChanged`):
```json
{ "method": "notifications/tools/list_changed" }
```

### Tool Definition Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique identifier (1-128 chars recommended) |
| `title` | string | No | Human-readable display name |
| `description` | string | Yes | Human-readable description |
| `icons` | Icon[] | No | Display icons |
| `inputSchema` | JSON Schema | Yes | Expected parameters schema |
| `outputSchema` | JSON Schema | No | Expected output structure schema |
| `annotations` | object | No | Behavior hints (untrusted from untrusted servers) |
| `execution` | object | No | Execution properties (e.g., `taskSupport`) |

### Tool Names

- SHOULD be 1-128 characters
- Case-sensitive
- Allowed characters: A-Z, a-z, 0-9, `_`, `-`, `.`
- SHOULD NOT contain spaces, commas, or special characters
- MUST be unique within a server
- Examples: `getUser`, `DATA_EXPORT_v2`, `admin.tools.list`

### Input Schema

- Follows JSON Schema usage guidelines (default 2020-12)
- MUST be a valid JSON Schema object (not null)
- For tools with no parameters:
  - Recommended: `{ "type": "object", "additionalProperties": false }`
  - Alternative: `{ "type": "object" }`

### Output Schema

If provided:
- Servers MUST provide `structuredContent` in results conforming to the schema
- Clients SHOULD validate structured results against this schema
- For backwards compatibility, tools with `outputSchema` SHOULD also return serialized JSON in a `TextContent` block

### Tool Result Content Types

**Text Content:**
```json
{ "type": "text", "text": "Result text" }
```

**Image Content:**
```json
{ "type": "image", "data": "base64-encoded-data", "mimeType": "image/png" }
```

**Audio Content:**
```json
{ "type": "audio", "data": "base64-encoded-audio-data", "mimeType": "audio/wav" }
```

**Resource Links** (URIs that can be subscribed to or fetched):
```json
{
  "type": "resource_link",
  "uri": "file:///project/src/main.rs",
  "name": "main.rs",
  "description": "Primary entry point",
  "mimeType": "text/x-rust"
}
```

Resource links returned by tools are NOT guaranteed to appear in `resources/list`.

**Embedded Resources** (inline resource data):
```json
{
  "type": "resource",
  "resource": {
    "uri": "file:///project/src/main.rs",
    "mimeType": "text/x-rust",
    "text": "fn main() { ... }"
  }
}
```

**Structured Content** (JSON in `structuredContent` field):
```json
{
  "result": {
    "content": [{ "type": "text", "text": "{\"temperature\": 22.5}" }],
    "structuredContent": { "temperature": 22.5, "conditions": "Partly cloudy" }
  }
}
```

All content types support optional annotations (audience, priority, lastModified).

### Tool Annotations

Behavior hints for clients (MUST be considered untrusted from untrusted servers):

| Annotation | Type | Default | Description |
|------------|------|---------|-------------|
| `readOnlyHint` | boolean | false | Tool doesn't modify state |
| `destructiveHint` | boolean | true | Tool may perform destructive operations |
| `idempotentHint` | boolean | false | Repeated identical calls have same effect |
| `openWorldHint` | boolean | true | Tool interacts with external entities |

### Error Handling — Two Mechanisms

1. **Protocol Errors** (JSON-RPC errors): Unknown tools, malformed requests, server errors
   - Models are less likely to self-correct from these
2. **Tool Execution Errors** (`isError: true` in result): API failures, validation errors, business logic errors
   - Contain actionable feedback for model self-correction
   - Clients SHOULD provide these to models

```json
// Protocol error
{ "error": { "code": -32602, "message": "Unknown tool: invalid_tool_name" } }

// Tool execution error
{ "result": { "content": [{ "type": "text", "text": "Invalid date: must be in the future" }], "isError": true } }
```

Input validation errors (e.g., date in wrong format) SHOULD be tool execution errors, not protocol errors.

### Security

Servers MUST: validate all tool inputs, implement access controls, rate limit invocations, sanitize outputs.
Clients SHOULD: prompt for user confirmation, show tool inputs before calling, validate results, implement timeouts, log usage.

---

## Prompts

Prompt templates allow servers to provide structured messages/instructions for interacting with language models.

### Capabilities

```json
{ "capabilities": { "prompts": { "listChanged": true } } }
```

### Protocol Messages

**List Prompts** (`prompts/list`) — supports pagination:
```json
{
  "result": {
    "prompts": [{
      "name": "code_review",
      "title": "Request Code Review",
      "description": "Analyze code quality and suggest improvements",
      "arguments": [
        { "name": "code", "description": "The code to review", "required": true }
      ],
      "icons": [{ "src": "https://example.com/review.svg", "mimeType": "image/svg+xml", "sizes": ["any"] }]
    }],
    "nextCursor": "next-page-cursor"
  }
}
```

**Get Prompt** (`prompts/get`) — arguments can be auto-completed via completion API:
```json
// Request
{
  "method": "prompts/get",
  "params": { "name": "code_review", "arguments": { "code": "def hello():\n    print('world')" } }
}

// Response
{
  "result": {
    "description": "Code review prompt",
    "messages": [
      {
        "role": "user",
        "content": {
          "type": "text",
          "text": "Please review this Python code:\ndef hello():\n    print('world')"
        }
      }
    ]
  }
}
```

**List Changed Notification**:
```json
{ "method": "notifications/prompts/list_changed" }
```

### Prompt Data Types

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique identifier |
| `title` | string | No | Human-readable display name |
| `description` | string | No | Human-readable description |
| `icons` | Icon[] | No | Display icons |
| `arguments` | Argument[] | No | List of arguments |

### PromptMessage Content Types

Messages have a `role` ("user" or "assistant") and content:

- **Text**: `{ "type": "text", "text": "..." }`
- **Image**: `{ "type": "image", "data": "base64...", "mimeType": "image/png" }` — base64-encoded, valid MIME required
- **Audio**: `{ "type": "audio", "data": "base64...", "mimeType": "audio/wav" }` — base64-encoded, valid MIME required
- **Embedded Resource**: `{ "type": "resource", "resource": { "uri": "...", "mimeType": "...", "text": "..." } }`

All content types support optional annotations.

### Error Handling

- Invalid prompt name: `-32602`
- Missing required arguments: `-32602`
- Internal errors: `-32603`

### Security

MUST validate all prompt inputs and outputs to prevent injection attacks or unauthorized resource access.

---

## Completion (Autocompletion)

Servers offer autocompletion suggestions for prompt arguments and resource template arguments.

### Capabilities

```json
{ "capabilities": { "completions": {} } }
```

### Protocol Messages

**Complete** (`completion/complete`):
```json
// Request — prompt argument
{
  "method": "completion/complete",
  "params": {
    "ref": { "type": "ref/prompt", "name": "code_review" },
    "argument": { "name": "language", "value": "py" }
  }
}

// Request — with context from previous completions
{
  "method": "completion/complete",
  "params": {
    "ref": { "type": "ref/prompt", "name": "code_review" },
    "argument": { "name": "framework", "value": "fla" },
    "context": { "arguments": { "language": "python" } }
  }
}

// Response
{ "result": { "completion": { "values": ["python", "pytorch", "pyside"], "total": 10, "hasMore": true } } }
```

### Reference Types

| Type | Usage | Example |
|------|-------|---------|
| `ref/prompt` | Prompt by name | `{ "type": "ref/prompt", "name": "code_review" }` |
| `ref/resource` | Resource URI template | `{ "type": "ref/resource", "uri": "file:///{path}" }` |

### Completion Results

- Maximum 100 items per response
- `total`: optional total number of available matches
- `hasMore`: boolean indicating additional results exist
- Values ranked by relevance
- Servers SHOULD implement fuzzy matching where appropriate
- Clients SHOULD debounce rapid requests and cache results

---

## Logging

Structured log messages from server to client.

### Capabilities

```json
{ "capabilities": { "logging": {} } }
```

### Log Levels (RFC 5424 syslog severity)

| Level | Description | Example |
|-------|-------------|---------|
| `debug` | Detailed debugging | Function entry/exit |
| `info` | General informational | Operation progress |
| `notice` | Normal but significant | Configuration changes |
| `warning` | Warning conditions | Deprecated feature usage |
| `error` | Error conditions | Operation failures |
| `critical` | Critical conditions | Component failures |
| `alert` | Action required immediately | Data corruption |
| `emergency` | System unusable | Complete system failure |

### Protocol Messages

**Set Log Level** (`logging/setLevel`, client→server):
```json
{ "method": "logging/setLevel", "params": { "level": "info" } }
```

**Log Notification** (`notifications/message`, server→client):
```json
{
  "method": "notifications/message",
  "params": {
    "level": "error",
    "logger": "database",
    "data": { "error": "Connection failed", "details": { "host": "localhost", "port": 5432 } }
  }
}
```

- Server sends only messages at or above the configured level
- `logger`: optional name for the log source
- `data`: arbitrary JSON-serializable data

### Security

Log messages MUST NOT contain: credentials/secrets, personal identifying information, internal system details that could aid attacks. Rate limit messages. Validate all data fields.
