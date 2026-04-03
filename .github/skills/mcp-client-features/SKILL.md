---
name: mcp-client-features
description: "MCP 2025-11-25 client features — sampling (createMessage, model preferences, hints, priorities, tool use in sampling, multi-turn tool loops, toolChoice modes, content constraints), roots (filesystem boundaries, list, notifications), elicitation (form mode with JSON Schema, URL mode for sensitive data, response actions, completion notifications, URLElicitationRequiredError). Use when: implementing MCP client capabilities, handling sampling requests, managing roots, processing elicitation."
---

# MCP Client Features (2025-11-25)

Clients provide three capabilities to servers:

| Feature | Purpose |
|---------|---------|
| **Sampling** | Server-initiated LLM interactions (completions/generations) |
| **Roots** | Filesystem boundaries for server operations |
| **Elicitation** | Server-initiated requests for user information |

---

## Sampling

Allows servers to request LLM completions from clients without needing their own API keys. Enables agentic behaviors with nested LLM calls inside other MCP operations.

### Capabilities

```json
// Basic sampling
{ "capabilities": { "sampling": {} } }

// With tool use support
{ "capabilities": { "sampling": { "tools": {} } } }

// With context inclusion support (soft-deprecated)
{ "capabilities": { "sampling": { "context": {} } } }
```

The `includeContext` parameter values `"thisServer"` and `"allServers"` are soft-deprecated. Servers SHOULD avoid using them unless client declares `sampling.context` capability.

### Creating Messages (`sampling/createMessage`)

Server→Client request:
```json
{
  "method": "sampling/createMessage",
  "params": {
    "messages": [
      { "role": "user", "content": { "type": "text", "text": "What is the capital of France?" } }
    ],
    "modelPreferences": {
      "hints": [{ "name": "claude-3-sonnet" }, { "name": "claude" }],
      "intelligencePriority": 0.8,
      "speedPriority": 0.5,
      "costPriority": 0.3
    },
    "systemPrompt": "You are a helpful assistant.",
    "maxTokens": 100
  }
}
```

Client→Server response:
```json
{
  "result": {
    "role": "assistant",
    "content": { "type": "text", "text": "The capital of France is Paris." },
    "model": "claude-3-sonnet-20240307",
    "stopReason": "endTurn"
  }
}
```

### Model Preferences

Servers cannot request models by name (client may use different providers). Instead, a preference system with priorities and hints:

**Capability Priorities** (0-1 normalized):
- `costPriority`: prefer cheaper models
- `speedPriority`: prefer faster models
- `intelligencePriority`: prefer more capable models

**Model Hints**: Substrings matching model names flexibly.
- Multiple hints evaluated in order of preference
- Clients MAY map hints to equivalent models from different providers
- Hints are advisory — client makes final selection

```json
{
  "hints": [{ "name": "claude-3-sonnet" }, { "name": "claude" }],
  "costPriority": 0.3,
  "speedPriority": 0.8,
  "intelligencePriority": 0.5
}
```

### Sampling with Tools

Servers can request tool use during sampling by providing `tools` and optional `toolChoice`. Client MUST declare `sampling.tools` capability.

```json
{
  "method": "sampling/createMessage",
  "params": {
    "messages": [{ "role": "user", "content": { "type": "text", "text": "Weather in Paris and London?" } }],
    "tools": [{
      "name": "get_weather",
      "description": "Get current weather for a city",
      "inputSchema": { "type": "object", "properties": { "city": { "type": "string" } }, "required": ["city"] }
    }],
    "toolChoice": { "mode": "auto" },
    "maxTokens": 1000
  }
}
```

Response with tool use (stopReason: "toolUse"):
```json
{
  "result": {
    "role": "assistant",
    "content": [
      { "type": "tool_use", "id": "call_abc123", "name": "get_weather", "input": { "city": "Paris" } },
      { "type": "tool_use", "id": "call_def456", "name": "get_weather", "input": { "city": "London" } }
    ],
    "model": "claude-3-sonnet-20240307",
    "stopReason": "toolUse"
  }
}
```

### Multi-Turn Tool Loop

1. Server receives tool use response
2. Executes requested tools
3. Sends new sampling request with tool results appended
4. Receives LLM response (may contain more tool uses)
5. Repeats (server may cap iterations; pass `toolChoice: {mode: "none"}` on last iteration)

Tool results in follow-up request:
```json
{
  "messages": [
    { "role": "user", "content": { "type": "text", "text": "Weather in Paris and London?" } },
    { "role": "assistant", "content": [
      { "type": "tool_use", "id": "call_abc123", "name": "get_weather", "input": { "city": "Paris" } },
      { "type": "tool_use", "id": "call_def456", "name": "get_weather", "input": { "city": "London" } }
    ]},
    { "role": "user", "content": [
      { "type": "tool_result", "toolUseId": "call_abc123", "content": [{ "type": "text", "text": "18°C, partly cloudy" }] },
      { "type": "tool_result", "toolUseId": "call_def456", "content": [{ "type": "text", "text": "15°C, rainy" }] }
    ]}
  ]
}
```

### Tool Choice Modes

- `{ "mode": "auto" }`: Model decides whether to use tools (default)
- `{ "mode": "required" }`: Model MUST use at least one tool
- `{ "mode": "none" }`: Model MUST NOT use any tools

### Message Content Constraints

**Tool Result Messages**: When a user message contains tool results, it MUST contain ONLY tool results. No mixing with text/image/audio.

**Tool Use and Result Balance**: Every assistant message with `ToolUseContent` MUST be followed by a user message consisting entirely of matching `ToolResultContent` blocks before any other message. Each tool use (by `id`) must be matched by a corresponding tool result (by `toolUseId`).

### Cross-API Compatibility

- Two roles only: `"user"` and `"assistant"`
- Tool use requests in assistant role; tool results in user role
- Parallel tool use supported (multiple ToolUseContent in one message)
- Compatible with Claude, OpenAI, and Gemini APIs

### Message Content Types

- **Text**: `{ "type": "text", "text": "..." }`
- **Image**: `{ "type": "image", "data": "base64...", "mimeType": "image/jpeg" }`
- **Audio**: `{ "type": "audio", "data": "base64...", "mimeType": "audio/wav" }`

### Error Handling

- User rejected sampling: `-1`
- Tool result missing: `-32602`
- Tool results mixed with other content: `-32602`

### Security

- SHOULD always have human in the loop to deny sampling requests
- Applications SHOULD provide UI to review requests, view/edit prompts, review responses
- Implement user approval controls; validate message content; respect model preference hints
- Implement rate limiting; handle sensitive data appropriately
- When tools used: ensure tool use/result balance; implement iteration limits for tool loops

---

## Roots

Filesystem boundaries that define where servers can operate. Servers request the list from clients.

### Capabilities

```json
{ "capabilities": { "roots": { "listChanged": true } } }
```

### Protocol Messages

**List Roots** (`roots/list`, server→client request):
```json
// Response
{
  "result": {
    "roots": [
      { "uri": "file:///home/user/projects/myproject", "name": "My Project" },
      { "uri": "file:///home/user/repos/backend", "name": "Backend Repository" }
    ]
  }
}
```

**Root List Changed** (`notifications/roots/list_changed`, client→server):
```json
{ "method": "notifications/roots/list_changed" }
```

### Root Data Type

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `uri` | string | Yes | MUST be a `file://` URI |
| `name` | string | No | Human-readable display name |

### Security

Clients MUST: Only expose roots with appropriate permissions; validate URIs to prevent path traversal; implement access controls; monitor root accessibility; prompt users for consent before exposing.

Servers SHOULD: Handle unavailable roots gracefully; respect root boundaries; validate paths against provided roots; cache root info appropriately.

### Error Handling

- Client doesn't support roots: `-32601`
- Internal errors: `-32603`

---

## Elicitation

Allows servers to request additional information from users through the client. Supports two modes:
- **Form mode**: Structured data collection with JSON Schema validation
- **URL mode**: External URL navigation for sensitive interactions

### Capabilities

```json
{
  "capabilities": {
    "elicitation": {
      "form": {},
      "url": {}
    }
  }
}
```

- Empty `"elicitation": {}` = form mode only (backwards compatibility)
- MUST support at least one mode
- Servers MUST NOT send requests with unsupported modes

### Form Mode Elicitation

Server→Client request:
```json
{
  "method": "elicitation/create",
  "params": {
    "mode": "form",
    "message": "Please provide your GitHub username",
    "requestedSchema": {
      "type": "object",
      "properties": {
        "name": { "type": "string" },
        "email": { "type": "string", "format": "email" },
        "age": { "type": "number", "minimum": 18 }
      },
      "required": ["name", "email"]
    }
  }
}
```

**Requested Schema** — restricted to flat objects with primitive properties only:

1. **String Schema**: `type: "string"`, supports `minLength`, `maxLength`, `pattern`, `format` (email, uri, date, date-time), `default`
2. **Number Schema**: `type: "number"` or `"integer"`, supports `minimum`, `maximum`, `default`
3. **Boolean Schema**: `type: "boolean"`, supports `default`
4. **Enum Schema**:
   - Single-select without titles: `{ "type": "string", "enum": ["Red", "Green", "Blue"] }`
   - Single-select with titles: `{ "type": "string", "oneOf": [{ "const": "#FF0000", "title": "Red" }, ...] }`
   - Multi-select without titles: `{ "type": "array", "items": { "type": "string", "enum": [...] }, "minItems": 1, "maxItems": 2 }`
   - Multi-select with titles: `{ "type": "array", "items": { "anyOf": [{ "const": "...", "title": "..." }, ...] } }`

All types support optional `default` values. Complex nested structures and arrays of objects are intentionally NOT supported.

Servers MUST NOT use form mode for sensitive info (passwords, API keys, tokens, payment credentials).

### URL Mode Elicitation (New in 2025-11-25)

For sensitive interactions that MUST NOT pass through the MCP client:

```json
{
  "method": "elicitation/create",
  "params": {
    "mode": "url",
    "elicitationId": "550e8400-e29b-41d4-a716-446655440000",
    "url": "https://mcp.example.com/ui/set_api_key",
    "message": "Please provide your API key to continue."
  }
}
```

- `url`: MUST be a valid URL (HTTPS for production)
- `elicitationId`: unique identifier
- Response `action: "accept"` = user consented to navigate (NOT that interaction is complete)
- Interaction occurs out-of-band; server sends optional completion notification

**Completion Notification** (server→client):
```json
{ "method": "notifications/elicitation/complete", "params": { "elicitationId": "..." } }
```

**URL Elicitation Required Error** (`-32042`): Server returns when request can't proceed without URL elicitation:
```json
{
  "error": {
    "code": -32042,
    "message": "This request requires more information.",
    "data": {
      "elicitations": [{
        "mode": "url",
        "elicitationId": "...",
        "url": "https://mcp.example.com/connect?...",
        "message": "Authorization required."
      }]
    }
  }
}
```

URL mode is NOT for authorizing the MCP client's access to the MCP server (that's handled by MCP authorization). It's for when the server needs to obtain sensitive information or third-party authorization on behalf of the user.

### Response Actions

Three-action model for all elicitation modes:

| Action | Meaning | Content |
|--------|---------|---------|
| `"accept"` | User submitted/approved | Form: data matching schema; URL: omitted |
| `"decline"` | User explicitly declined | Typically omitted |
| `"cancel"` | User dismissed without choice | Typically omitted |

### URL Mode for Third-Party OAuth

When MCP servers need credentials for external services:
1. Server generates authorization URL (acting as OAuth client to third-party)
2. Server sends URL mode elicitation to client
3. User completes OAuth flow directly with third-party
4. Third-party redirects back to MCP server
5. MCP server stores tokens bound to user identity

Critical requirements:
- Third-party credentials MUST NOT transit through MCP client
- MCP server MUST NOT use client's token for third-party (forbidden token passthrough)
- User MUST authorize MCP server directly
- MCP server manages third-party tokens (must be stateful)

### Security

**Safe URL Handling** — Servers:
- MUST NOT include sensitive user info in elicitation URLs
- MUST NOT provide pre-authenticated URLs
- SHOULD use HTTPS for production

**Safe URL Handling** — Clients:
- MUST NOT auto-fetch or pre-fetch the URL
- MUST NOT open URL without explicit user consent
- MUST show full URL for examination before consent
- MUST open URL securely (no client/LLM inspection of content — e.g., SFSafariViewController, not WKWebView)
- SHOULD highlight the domain; warn for suspicious/Punycode URIs

**Identifying the User**: Servers MUST NOT rely on client-provided user identification. Use authorization-based identity.

**Phishing Prevention**: Server MUST verify the user who opens a URL elicitation is the same user who the elicitation was generated for. Typically done via MCP authorization server session cookies.

### Statefulness

Servers implementing elicitation MUST securely associate state with individual users:
- State MUST NOT be associated with session IDs alone
- State storage MUST be protected against unauthorized access
- For remote servers, user identity MUST be derived from MCP authorization credentials

### Error Handling

- Unsupported elicitation mode: `-32602`
- URL elicitation required: `-32042`
