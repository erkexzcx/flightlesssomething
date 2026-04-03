---
name: mcp-transport-auth
description: "MCP 2025-11-25 transports and authorization — stdio transport (subprocess model, stdin/stdout JSON-RPC, stderr logging, shutdown sequence), Streamable HTTP transport (single endpoint, POST/GET/DELETE, SSE streaming, session management with MCP-Session-Id, resumability via Last-Event-ID, multiple connections, backwards compatibility), OAuth 2.1 authorization (Protected Resource Metadata RFC 9728, AS discovery, client registration approaches, PKCE S256, resource parameter RFC 8707, scope challenge handling, security constraints). Use when: implementing MCP transports, configuring HTTP endpoints, implementing OAuth for MCP, handling session management."
---

# MCP Transports & Authorization (2025-11-25)

MCP defines two standard transports and an OAuth 2.1-based authorization framework.

---

## stdio Transport

Server runs as a subprocess of the client. Communication via standard I/O streams.

### Architecture

- Client launches server as a child process
- Client sends JSON-RPC messages on server's **stdin**
- Server writes JSON-RPC messages on **stdout**
- Server MAY write UTF-8 logging/diagnostics on **stderr** (client SHOULD capture/forward but NOT send protocol messages on stderr)

### Message Format

- Messages are newline-delimited JSON
- Each JSON-RPC message MUST NOT contain embedded newlines
- Messages delimited by newlines

### Shutdown Sequence

1. Client closes stdin to server subprocess
2. Server exits gracefully after finishing in-progress work
3. Client sends `SIGTERM` if server doesn't exit within reasonable time
4. Client sends `SIGKILL` if server still doesn't exit (last resort)

### Security

- Server inherits client's environment (env vars, working directory)
- Suitable for local (same-machine) communication only
- Client SHOULD validate server process identity

---

## Streamable HTTP Transport

Client-server communication over HTTPS using a single endpoint. Supports request/response and streaming via Server-Sent Events (SSE).

### Endpoint

A single HTTP endpoint handles all MCP communication (e.g., `https://example.com/mcp`).

### HTTP Methods

**POST** — Client sends JSON-RPC messages:
- Request body: Single JSON-RPC message (request, notification, or batch)
- `Accept` header: MUST include both `application/json` and `text/event-stream`
- `Content-Type` header: `application/json`
- Server responds with: `application/json` (single response) OR `text/event-stream` (SSE stream)

For notifications/responses-only: server responds with HTTP 202 Accepted (no body).

**GET** — Client opens server-initiated SSE stream:
- `Accept` header: `text/event-stream`
- Server opens long-lived SSE connection for server-initiated messages (requests, notifications)
- GET is OPTIONAL; server may return 405 if it doesn't initiate messages
- Only valid AFTER initialization is complete

**DELETE** — Client terminates session:
- Server MUST respond with HTTP 405 if it doesn't support sessions

### Server-Sent Events (SSE)

SSE streams carry JSON-RPC messages as events:
```
event: message
data: {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-11-25",...}}

event: message
data: {"jsonrpc":"2.0","method":"notifications/progress","params":{...}}
```

Each SSE event has:
- `event: message`
- `data:` containing a single JSON-RPC message
- Optional `id:` for resumability

Batched requests: server MAY send responses for individual requests in a batch as separate SSE events on the same stream.

### Session Management

**MCP-Session-Id Header**:
- Server MAY assign a session ID in the `initialize` response via `MCP-Session-Id` header
- If assigned, client MUST include this header in ALL subsequent requests
- Server returns HTTP **400 Bad Request** if client sends unknown/invalid session ID
- Session IDs MUST be unpredictable, cryptographically secure (UUID v4, random tokens)
- Server MUST NOT use session ID as sole authentication mechanism

**Session Lifecycle**:
1. Client sends `initialize` POST → server returns `MCP-Session-Id`
2. Client includes `MCP-Session-Id` on all subsequent requests
3. Client sends HTTP DELETE to terminate session
4. Server invalidates session ID

**Without Sessions**: If server does not send `MCP-Session-Id`, client MUST NOT include the header. Server treats each request independently.

### Resumability

SSE reconnection with event IDs:
- Server MAY attach IDs to SSE events: `id: event-123`
- On disconnect, client MAY reconnect with `Last-Event-ID` header
- Server SHOULD replay missed events if possible
- Server MAY respond with HTTP 204 No Content if replay not possible (client should cancel the pending request and re-send)

### Multiple Connections

- Client MAY open multiple SSE connections (GET and/or POST) simultaneously
- Server MUST NOT assume requests arrive on any particular connection
- Server MUST NOT broadcast messages to all connections; each SSE event targets one logical client

### Protocol Version Header

- Server MAY include `MCP-Protocol-Version` header in responses (set to negotiated version string, e.g., `2025-11-25`)

### DNS Rebinding Protection

- Server MUST validate `Origin` header on all requests
- Server MUST reject requests with unexpected `Origin` with HTTP 403
- Servers bound to localhost MUST check origin to prevent DNS rebinding attacks

### Backwards Compatibility

The 2025-11-25 Streamable HTTP transport is backwards compatible with the deprecated HTTP+SSE transport (2024-11-05). Clients MAY attempt new transport first with `MCP-Protocol-Version` header, then fall back to old transport on failure.

Old HTTP+SSE transport used two endpoints:
- SSE endpoint for server→client streaming
- Separate POST endpoint for client→server messages
- Not supported in 2025-11-25 as a first-class transport

---

## Authorization

### Overview

OAuth 2.1-based authorization framework. The MCP server acts as an OAuth 2.0 **resource server** (protected resource). The client acts as an OAuth **client**. An external authorization server (AS) issues tokens.

Authorization is REQUIRED for Streamable HTTP transport. NOT applicable to stdio transport (uses OS-level security).

### Discovery Flow

1. Client sends request without credentials
2. Server responds with **HTTP 401** and `WWW-Authenticate` header:
   ```
   WWW-Authenticate: Bearer resource_metadata="https://mcp.example.com/.well-known/oauth-protected-resource"
   ```
3. Client fetches Protected Resource Metadata (RFC 9728)
4. Client discovers authorization server(s) from metadata
5. Client fetches AS metadata
6. Client registers (if needed) and performs OAuth flow
7. Client retries original request with Bearer token

### Protected Resource Metadata (RFC 9728)

Resource metadata URL from `WWW-Authenticate` header, or fallback:
```
https://mcp.example.com/.well-known/oauth-protected-resource/mcp-endpoint-path
```

Response contains:
```json
{
  "resource": "https://mcp.example.com/mcp",
  "authorization_servers": ["https://auth.example.com"],
  "scopes_supported": ["mcp:read", "mcp:write"],
  "bearer_methods_supported": ["header"]
}
```

### Authorization Server Metadata Discovery

Priority ordered:
1. OAuth 2.0 Authorization Server Metadata (RFC 8414): `https://auth.example.com/.well-known/oauth-authorization-server`
2. OpenID Connect Discovery: `https://auth.example.com/.well-known/openid-configuration`

AS metadata includes standard OAuth fields: `issuer`, `authorization_endpoint`, `token_endpoint`, `registration_endpoint`, `response_types_supported`, `grant_types_supported`, `code_challenge_methods_supported`, etc.

### Client Registration

Three approaches (in recommended order):

**1. Client ID Metadata Document (Recommended)**
- Client publishes metadata at a URL: `https://myapp.example.com/.well-known/oauth-client`
- `client_id` = the metadata URL
- AS fetches and validates client metadata from this URL
- No pre-registration or dynamic registration needed
- Client metadata document contains: `client_id`, `client_name`, `redirect_uris`, `grant_types`, `response_types`, `token_endpoint_auth_method`, `scope`, `contacts`, `logo_uri`

**2. Pre-Registration**
- AS provides client credentials out-of-band
- Traditional approach; requires manual setup per AS

**3. Dynamic Client Registration (RFC 7591)**
- Client sends registration request to `registration_endpoint`
- AS returns `client_id` and `client_secret`
- Automated but not all AS support it

### Authorization Flow (PKCE Required)

1. **Generate PKCE**: Create `code_verifier` (43-128 chars, URL-safe), compute `code_challenge` = BASE64URL(SHA256(code_verifier))
2. **Authorization Request**: Redirect user to AS with:
   - `response_type=code`
   - `client_id=...`
   - `redirect_uri=...`
   - `code_challenge=...` + `code_challenge_method=S256` (MUST use S256)
   - `scope=...` (from WWW-Authenticate or scopes_supported)
   - `resource=...` (RFC 8707 resource parameter = MCP server URL)
3. **User Authenticates**: User logs in and consents at AS
4. **Callback**: AS redirects to `redirect_uri` with `code=...`
5. **Token Exchange**: Client sends to token endpoint:
   - `grant_type=authorization_code`
   - `code=...`
   - `redirect_uri=...`
   - `code_verifier=...` (proves PKCE)
   - `resource=...` (MUST match authorization request)
6. **Receive Tokens**: AS responds with `access_token`, optional `refresh_token`, `token_type`, `expires_in`, `scope`

### Scope Selection Strategy

1. Use scope from `WWW-Authenticate` header (server's explicit challenge)
2. If not available, use `scopes_supported` from Protected Resource Metadata
3. Clients SHOULD request minimum necessary scopes

### Access Token Usage

Include on EVERY request to MCP server:
```
Authorization: Bearer eyJhbGciOiJS...
```

Server validates token on each request. On token expiry, server responds with HTTP 401. Client uses refresh token to get new access token.

### Scope Challenge Handling

When client has a valid token but insufficient scope:
1. Server responds with HTTP **403 Forbidden** and:
   ```
   WWW-Authenticate: Bearer error="insufficient_scope" scope="mcp:admin"
   ```
2. Client performs **step-up authorization**: new OAuth flow requesting the additional scope
3. Client retries request with new token

### Security Requirements

**Token Audience Binding**:
- Tokens MUST be bound to the specific MCP server (resource)
- Use `resource` parameter (RFC 8707) in both authorization and token requests
- MCP server validates token audience matches its own URL

**Token Theft Prevention**:
- Use short-lived access tokens
- Implement token rotation for refresh tokens
- Bind tokens to specific clients (PKCE accomplishes this)

**Communication Security**:
- MUST use HTTPS in production (TLS 1.2+)
- HTTP only for local development/testing
- Validate TLS certificates

**Authorization Code Protection**:
- PKCE MUST use S256 method (not plain)
- Authorization codes: single-use, short-lived (max 10 minutes recommended)
- Bind codes to specific client and redirect URI

**Open Redirection Prevention**:
- Validate `redirect_uri` exactly matches registered URI
- Do not allow partial/pattern matching

**Confused Deputy Problem**:
- Tokens obtained for one MCP server MUST NOT be used for another
- `resource` parameter prevents this
- Server MUST validate token's intended audience

**Token Passthrough Forbidden**:
- MCP server MUST NOT forward client's access token to third-party services
- If server needs third-party access, use separate credentials (e.g., URL elicitation for third-party OAuth)

---

## Transport Selection Guide

| Criteria | stdio | Streamable HTTP |
|----------|-------|-----------------|
| Deployment | Local process | Local or remote server |
| Security model | OS process isolation | OAuth 2.1 + HTTPS |
| Streaming | Bidirectional via stdin/stdout | SSE (server→client), POST (client→server) |
| Session management | Process lifetime | MCP-Session-Id header |
| Resumability | N/A | SSE event IDs + Last-Event-ID |
| Multiple clients | One client per process | Multiple sessions per server |
| Use case | CLI tools, IDE extensions | Web services, shared servers |

---

## Error Codes Reference (Transport/Auth)

| Code | Name | Context |
|------|------|---------|
| HTTP 400 | Bad Request | Invalid/expired session ID |
| HTTP 401 | Unauthorized | Missing/invalid Bearer token |
| HTTP 403 | Forbidden | Insufficient scope, invalid Origin |
| HTTP 405 | Method Not Allowed | GET/DELETE not supported by server |
| HTTP 202 | Accepted | Notification/response-only POST accepted |
| HTTP 204 | No Content | SSE replay not available |
