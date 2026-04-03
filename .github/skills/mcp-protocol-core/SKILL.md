---
name: mcp-protocol-core
description: "Model Context Protocol (MCP) 2025-11-25 core specification — architecture (host/client/server), JSON-RPC 2.0 base protocol, lifecycle (initialization, capability negotiation, shutdown), error codes, _meta fields, icons, utilities (ping, cancellation, progress, tasks, pagination). Use when: implementing MCP servers or clients, designing MCP protocol messages, debugging MCP lifecycle issues, understanding MCP architecture."
---

# MCP Core Protocol Specification (2025-11-25)

Reference: https://modelcontextprotocol.io/specification/2025-11-25
TypeScript schema: https://github.com/modelcontextprotocol/specification/blob/main/schema/2025-11-25/schema.ts

---

## Architecture

MCP follows a **client-host-server** architecture. Built on JSON-RPC 2.0, it provides a stateful session protocol focused on context exchange and sampling coordination.

### Core Components

**Host**: The container/coordinator process (e.g., an IDE or chat app).
- Creates and manages multiple client instances
- Controls client connection permissions and lifecycle
- Enforces security policies and consent requirements
- Handles user authorization decisions
- Coordinates AI/LLM integration and sampling
- Manages context aggregation across clients

**Client**: Created by the host, maintains an isolated 1:1 connection with one server.
- Establishes one stateful session per server
- Handles protocol negotiation and capability exchange
- Routes protocol messages bidirectionally
- Manages subscriptions and notifications
- Maintains security boundaries between servers

**Server**: Provides specialized context and capabilities.
- Exposes resources, tools, and prompts via MCP primitives
- Operates independently with focused responsibilities
- Requests sampling through client interfaces
- Must respect security constraints
- Can be local processes or remote services

### Design Principles

1. **Servers should be extremely easy to build** — hosts handle complex orchestration
2. **Servers should be highly composable** — focused functionality, combined seamlessly
3. **Servers should not see the whole conversation** — receive only necessary context; host enforces isolation
4. **Features can be added progressively** — core protocol is minimal; additional capabilities negotiated as needed; backwards compatibility maintained

---

## Base Protocol — JSON-RPC 2.0 Messages

All messages MUST follow JSON-RPC 2.0 specification. Three message types:

### Requests

```json
{
  "jsonrpc": "2.0",
  "id": "string | number",
  "method": "string",
  "params": { ... }
}
```

- MUST include a string or integer `id` (NOT null)
- ID MUST NOT have been previously used by the requestor in the same session
- Sent client→server or server→client

### Responses

**Result (success):**
```json
{
  "jsonrpc": "2.0",
  "id": "string | number",
  "result": { ... }
}
```

**Error (failure):**
```json
{
  "jsonrpc": "2.0",
  "id": "string | number",
  "error": {
    "code": -32602,
    "message": "Human-readable error",
    "data": { ... }
  }
}
```

- MUST include same ID as the request
- Error codes MUST be integers

### Notifications

```json
{
  "jsonrpc": "2.0",
  "method": "string",
  "params": { ... }
}
```

- MUST NOT include an `id`
- One-way message; receiver MUST NOT send a response

---

## Standard JSON-RPC Error Codes

| Code | Meaning | Usage |
|------|---------|-------|
| -32700 | Parse error | Invalid JSON |
| -32600 | Invalid request | Not a valid JSON-RPC request |
| -32601 | Method not found | Unknown method / capability not supported |
| -32602 | Invalid params | Invalid method parameters, unknown tool, invalid cursor |
| -32603 | Internal error | Server-side errors |
| -32002 | Resource not found | Resource URI not found |
| -32042 | URL elicitation required | Request needs URL mode elicitation completion |
| -1 | User rejected | User rejected sampling/elicitation request |

---

## JSON Schema Usage

MCP uses JSON Schema for validation throughout the protocol.

- **Default dialect**: JSON Schema 2020-12 (when no `$schema` field present)
- Schemas MAY include `$schema` to specify a different dialect
- Implementations MUST support at least 2020-12
- Implementations MUST handle unsupported dialects gracefully with appropriate errors
- Schemas MUST be valid according to their declared or default dialect

---

## General Fields

### `_meta`

Reserved property for attaching metadata to interactions.

**Key name format**: `[prefix/]name`
- Prefix: optional, series of labels separated by dots followed by `/`
  - SHOULD use reverse DNS notation (e.g., `com.example/`)
  - Prefixes where second label is `modelcontextprotocol` or `mcp` are reserved (e.g., `io.modelcontextprotocol/`, `dev.mcp/`)
  - `com.example.mcp/` is NOT reserved (second label is `example`)
- Name: alphanumeric, may contain hyphens, underscores, dots

### `icons`

Standardized visual identifiers for servers, tools, resources, prompts, and implementations.

```json
{
  "icons": [
    {
      "src": "https://example.com/icon.png",
      "mimeType": "image/png",
      "sizes": ["48x48"],
      "theme": "light"
    }
  ]
}
```

- `src`: URI (HTTPS or data: URI) — required
- `mimeType`: optional MIME type
- `sizes`: optional array (e.g., `["48x48"]`, `["any"]` for SVG)
- `theme`: optional `"light"` or `"dark"`

**Required MIME support**: `image/png`, `image/jpeg`
**Recommended**: `image/svg+xml`, `image/webp`

**Security**: Treat icon metadata as untrusted. MUST reject unsafe URI schemes (`javascript:`, `file:`, `ftp:`, `ws:`). Fetch without credentials. Verify same-origin. Guard against SVG with embedded scripts. Validate MIME types via magic bytes. Set limits on image size/dimensions/frames.

Attachable to: `Implementation`, `Tool`, `Prompt`, `Resource`.

---

## Lifecycle

Three phases: **Initialization → Operation → Shutdown**

### Initialization

MUST be the first interaction. Client sends `initialize` request:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-11-25",
    "capabilities": {
      "roots": { "listChanged": true },
      "sampling": {},
      "elicitation": { "form": {}, "url": {} },
      "tasks": {
        "requests": {
          "elicitation": { "create": {} },
          "sampling": { "createMessage": {} }
        }
      }
    },
    "clientInfo": {
      "name": "ExampleClient",
      "title": "Display Name",
      "version": "1.0.0",
      "description": "An example MCP client",
      "icons": [{ "src": "https://example.com/icon.png", "mimeType": "image/png", "sizes": ["48x48"] }],
      "websiteUrl": "https://example.com"
    }
  }
}
```

Server responds with its capabilities:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-11-25",
    "capabilities": {
      "logging": {},
      "prompts": { "listChanged": true },
      "resources": { "subscribe": true, "listChanged": true },
      "tools": { "listChanged": true },
      "completions": {},
      "tasks": {
        "list": {},
        "cancel": {},
        "requests": { "tools": { "call": {} } }
      }
    },
    "serverInfo": {
      "name": "ExampleServer",
      "title": "Display Name",
      "version": "1.0.0"
    },
    "instructions": "Optional instructions for the client"
  }
}
```

After successful initialization, client MUST send:
```json
{ "jsonrpc": "2.0", "method": "notifications/initialized" }
```

- Client SHOULD NOT send requests (except pings) before server responds to `initialize`
- Server SHOULD NOT send requests (except pings and logging) before receiving `initialized`

### Version Negotiation

- Client sends protocol version it supports (SHOULD be latest)
- If server supports it, responds with same version
- Otherwise, server responds with another version it supports (SHOULD be latest it supports)
- If client doesn't support server's version, it SHOULD disconnect
- For HTTP: client MUST include `MCP-Protocol-Version: <version>` header on all subsequent requests

### Capability Negotiation

| Category | Capability | Description |
|----------|-----------|-------------|
| Client | `roots` | Can provide filesystem roots |
| Client | `sampling` | Supports LLM sampling requests |
| Client | `elicitation` | Supports server elicitation requests |
| Client | `tasks` | Supports task-augmented client requests |
| Client | `experimental` | Non-standard experimental features |
| Server | `prompts` | Offers prompt templates |
| Server | `resources` | Provides readable resources |
| Server | `tools` | Exposes callable tools |
| Server | `logging` | Emits structured log messages |
| Server | `completions` | Supports argument autocompletion |
| Server | `tasks` | Supports task-augmented server requests |
| Server | `experimental` | Non-standard experimental features |

Sub-capabilities:
- `listChanged`: support for list change notifications (prompts, resources, tools)
- `subscribe`: support for subscribing to individual resource changes (resources only)

### Operation Phase

Both parties MUST:
- Respect the negotiated protocol version
- Only use capabilities that were successfully negotiated

### Shutdown

**stdio**: Client closes stdin to server → waits for exit → SIGTERM → SIGKILL. Server MAY initiate by closing stdout and exiting.

**HTTP**: Close the associated HTTP connection(s).

---

## Timeouts

- Implementations SHOULD establish timeouts for all sent requests
- When timeout expires: issue cancellation notification, stop waiting for response
- SDKs SHOULD allow per-request timeout configuration
- MAY reset timeout clock on receiving progress notifications
- SHOULD always enforce maximum timeout regardless of progress notifications

---

## Utilities

### Ping

Verify connection liveness. Either side can send:

```json
{ "jsonrpc": "2.0", "id": "123", "method": "ping" }
```

Receiver MUST respond promptly:
```json
{ "jsonrpc": "2.0", "id": "123", "result": {} }
```

- SHOULD periodically issue pings for connection health
- Frequency SHOULD be configurable
- Timeout → may consider connection stale, terminate, reconnect
- Avoid excessive pinging

### Cancellation

Either side can cancel an in-progress request:

```json
{
  "jsonrpc": "2.0",
  "method": "notifications/cancelled",
  "params": {
    "requestId": "123",
    "reason": "User requested cancellation"
  }
}
```

Rules:
- MUST only reference requests previously issued in the same direction and believed in-progress
- `initialize` request MUST NOT be cancelled
- For task-augmented requests, use `tasks/cancel` instead
- Receivers SHOULD stop processing, free resources, not send response for cancelled request
- Receivers MAY ignore if request unknown or already completed
- Sender SHOULD ignore any response that arrives after sending cancellation
- Handle race conditions gracefully (cancellation may arrive after completion)
- Log cancellation reasons for debugging

### Progress

Track long-running operations via notifications.

Request includes progress token in `_meta`:
```json
{
  "method": "some_method",
  "params": { "_meta": { "progressToken": "abc123" } }
}
```

Receiver sends progress notifications:
```json
{
  "jsonrpc": "2.0",
  "method": "notifications/progress",
  "params": {
    "progressToken": "abc123",
    "progress": 50,
    "total": 100,
    "message": "Processing..."
  }
}
```

- `progressToken`: string or integer, MUST be unique across active requests
- `progress` MUST increase with each notification; MAY be floating point
- `total` MAY be omitted if unknown; MAY be floating point
- `message` SHOULD provide relevant human-readable progress info
- For task-augmented requests, `progressToken` remains valid throughout task lifetime until terminal status
- Receivers MAY choose not to send progress notifications at all
- Implement rate limiting to prevent flooding

### Tasks (Experimental — introduced 2025-11-25)

Durable state machines for expensive/long-running operations with polling and deferred result retrieval. Either clients or servers can be requestors/receivers.

**Creating a task** — include `task` field in request params:
```json
{
  "method": "tools/call",
  "params": {
    "name": "long_operation",
    "arguments": { ... },
    "task": { "ttl": 60000 }
  }
}
```

Response: `CreateTaskResult` (NOT the operation result):
```json
{
  "result": {
    "task": {
      "taskId": "786512e2-9e0d-44bd-8f29-789f320fe840",
      "status": "working",
      "statusMessage": "In progress",
      "createdAt": "2025-11-25T10:30:00Z",
      "lastUpdatedAt": "2025-11-25T10:30:00Z",
      "ttl": 60000,
      "pollInterval": 5000
    }
  }
}
```

**Task Status Lifecycle**:
```
working → input_required | completed | failed | cancelled
input_required → working | completed | failed | cancelled
completed, failed, cancelled → (terminal, no transitions)
```

**Key Operations**:

| Method | Direction | Purpose |
|--------|-----------|---------|
| `tasks/get` | requestor→receiver | Poll task status (respect `pollInterval`) |
| `tasks/result` | requestor→receiver | Retrieve final result (blocks until terminal) |
| `tasks/list` | requestor→receiver | List tasks (paginated) |
| `tasks/cancel` | requestor→receiver | Cancel a task |
| `notifications/tasks/status` | receiver→requestor | Optional status change notification |

**Capability Declaration**:
- Server: `tasks.list`, `tasks.cancel`, `tasks.requests.tools.call`
- Client: `tasks.list`, `tasks.cancel`, `tasks.requests.sampling.createMessage`, `tasks.requests.elicitation.create`

**Tool-Level Negotiation** via `execution.taskSupport`:
- `"forbidden"` (default): MUST NOT use task augmentation
- `"optional"`: MAY use task augmentation or normal request
- `"required"`: MUST use task augmentation; server returns `-32601` if client doesn't

**`input_required` Status**: Receiver needs input from requestor. Requestor should preemptively call `tasks/result`. Receiver includes `io.modelcontextprotocol/related-task` in associated requests.

**Related Task Metadata**: All task-related messages MUST include in `_meta`:
```json
{ "io.modelcontextprotocol/related-task": { "taskId": "..." } }
```
Exception: `tasks/get`, `tasks/list`, `tasks/cancel` use the `taskId` parameter directly.

**TTL & Resource Management**:
- `createdAt` (ISO 8601) MUST be included in all task responses
- `lastUpdatedAt` (ISO 8601) MUST be included
- Receivers MAY override requested TTL; MUST include actual TTL (or null) in responses
- After TTL expires, receivers MAY delete task and results
- Task IDs: string, generated by receiver, MUST be unique

**Security**:
- When auth context available, receivers MUST bind tasks to auth context
- Without auth context, use cryptographically secure task IDs with sufficient entropy
- Receivers MUST reject access to tasks not belonging to requestor's auth context
- Implement rate limiting on task operations
- Enforce limits on concurrent tasks per requestor

### Pagination

Cursor-based pagination for list operations.

```json
// Request
{ "method": "resources/list", "params": { "cursor": "opaque-cursor" } }

// Response
{ "result": { "resources": [...], "nextCursor": "next-opaque-cursor" } }
```

- Cursors are opaque strings — clients MUST NOT parse/modify them
- Missing `nextCursor` = end of results
- Page size determined by server; clients MUST NOT assume fixed page size
- Supported: `resources/list`, `resources/templates/list`, `prompts/list`, `tools/list`, `tasks/list`
- Invalid cursors → error `-32602`
- Cursors SHOULD be stable; MUST NOT be persisted across sessions

---

## Security & Trust Principles

1. **User Consent and Control** — explicit consent for all data access and operations; clear UIs for review/authorization
2. **Data Privacy** — explicit consent before exposing user data to servers; don't transmit resource data without consent
3. **Tool Safety** — treat tool annotations as untrusted unless from trusted server; require explicit consent before invocation
4. **LLM Sampling Controls** — explicit approval for sampling requests; user controls prompts sent and results visible; protocol limits server visibility into prompts

### Implementation Guidelines

- Build robust consent and authorization flows
- Provide clear documentation of security implications
- Implement appropriate access controls and data protections
- Follow security best practices
- Consider privacy implications in feature designs

---

## Error Handling

Handle these error cases:
- Protocol version mismatch → return error with supported versions
- Failure to negotiate required capabilities → disconnect
- Request timeouts → cancellation notification
- Invalid cancellation notifications → silently ignore
- Invalid cursors → `-32602`
- Unknown methods → `-32601`
