# API Documentation

REST API for managing benchmark data.

## Base URL

```
http://your-server:5000
```

## Authentication

The API supports two authentication methods:

### API Tokens (Recommended for Programmatic Access)
API tokens are the recommended way to authenticate programmatically. They provide secure, revocable access without exposing credentials.

**Token Format**: Include token in `Authorization` header as `Bearer <token>`

```bash
curl http://localhost:5000/api/benchmarks \
  -H "Authorization: Bearer your_api_token_here"
```

**Benefits**:
- Designed for automated tools and scripts
- Can be revoked individually without affecting other tokens
- Track last usage per token
- Set descriptive names for organization
- Maximum 10 tokens per user

See [API Token Management](#api-token-management) for creating and managing tokens.

### Session Authentication (Web UI)
Users authenticate via Discord OAuth flow for web interface access. Sessions are stored in cookies.

Admin users can also use username/password authentication for web access.

## Endpoints

### Health Check

**GET** `/health`

Check server status.

```bash
curl http://localhost:5000/health
```

Response:
```json
{"status": "ok"}
```

---

### Authentication

**GET** `/auth/login` - Initiate Discord OAuth login

**GET** `/auth/login/callback` - Discord OAuth callback

**POST** `/auth/admin/login` - Admin login

Request:
```json
{"username": "admin", "password": "password"}
```

**POST** `/auth/logout` - Logout current user

---

### List Benchmarks

**GET** `/api/benchmarks`

List all benchmarks with pagination and search.

Query parameters:
- `page` (integer) - Page number (default: 1)
- `per_page` (integer) - Results per page (default: 10, max: 100)
- `search` (string) - Search in title/description
- `user_id` (integer) - Filter by user ID

Example:
```bash
curl "http://localhost:5000/api/benchmarks?page=1&per_page=20"
```

Response:
```json
{
  "benchmarks": [
    {
      "id": 1,
      "user_id": 1,
      "title": "Cyberpunk 2077",
      "description": "Ultra settings",
      "created_at": "2025-11-23T16:30:00Z",
      "updated_at": "2025-11-23T16:30:00Z",
      "user": {
        "id": 1,
        "username": "Player1",
        "avatar": "https://cdn.discordapp.com/..."
      }
    }
  ],
  "total": 50,
  "page": 1,
  "per_page": 20,
  "total_pages": 3
}
```

---

### Get Benchmark

**GET** `/api/benchmarks/:id`

Get benchmark details.

Example:
```bash
curl http://localhost:5000/api/benchmarks/1
```

---

### Get Benchmark Data

**GET** `/api/benchmarks/:id/data`

Download benchmark data (compressed binary).

Example:
```bash
curl http://localhost:5000/api/benchmarks/1/data -o benchmark.dat
```

---

### Download Benchmark (ZIP)

**GET** `/api/benchmarks/:id/download`

Download benchmark as ZIP with CSV files (MangoHud format).

Example:
```bash
curl http://localhost:5000/api/benchmarks/1/download -o benchmark.zip
```

---

### Create Benchmark

**POST** `/api/benchmarks`

**Authentication required** - See [API Token Management](#api-token-management) for creating tokens.

Upload benchmark with CSV/HML files.

Request: `multipart/form-data`
- `title` (string, required) - Game name
- `description` (string, optional) - Additional notes
- `files` (file[], required) - Benchmark CSV/HML files

Example:
```bash
curl -X POST http://localhost:5000/api/benchmarks \
  -H "Authorization: Bearer your_api_token_here" \
  -F "title=Cyberpunk 2077" \
  -F "description=Ultra settings" \
  -F "files=@benchmark1.csv" \
  -F "files=@benchmark2.csv"
```

Supported formats:
- MangoHud CSV
- MSI Afterburner HML

---

### Update Benchmark

**PUT** `/api/benchmarks/:id`

**Authentication required** (owner or admin) - See [API Token Management](#api-token-management) for creating tokens.

Update benchmark metadata and run labels.

Request:
```json
{
  "title": "Updated Title",
  "description": "Updated description",
  "labels": {
    "0": "Run 1 label",
    "1": "Run 2 label"
  }
}
```

Example:
```bash
curl -X PUT http://localhost:5000/api/benchmarks/1 \
  -H "Authorization: Bearer your_api_token_here" \
  -H "Content-Type: application/json" \
  -d '{"title":"New Title","description":"New description"}'
```

---

### Delete Benchmark

**DELETE** `/api/benchmarks/:id`

**Authentication required** (owner or admin) - See [API Token Management](#api-token-management) for creating tokens.

Delete benchmark and all associated data.

Example:
```bash
curl -X DELETE http://localhost:5000/api/benchmarks/1 \
  -H "Authorization: Bearer your_api_token_here"
```

---

### List Users (Admin)

**GET** `/api/admin/users`

**Admin authentication required** - See [API Token Management](#api-token-management) for creating tokens.

List all registered users.

Example:
```bash
curl http://localhost:5000/api/admin/users \
  -H "Authorization: Bearer your_admin_api_token_here"
```

---

### Delete User (Admin)

**DELETE** `/api/admin/users/:id`

**Admin authentication required** - See [API Token Management](#api-token-management) for creating tokens.

Delete user and optionally their benchmarks.

Query parameters:
- `delete_data` (boolean) - Also delete user's benchmarks (default: false)

Example:
```bash
# Delete user only
curl -X DELETE http://localhost:5000/api/admin/users/1 \
  -H "Authorization: Bearer your_admin_api_token_here"

# Delete user and data
curl -X DELETE "http://localhost:5000/api/admin/users/1?delete_data=true" \
  -H "Authorization: Bearer your_admin_api_token_here"
```

---

## API Token Management

API tokens provide secure, programmatic access to the API. Tokens must be created through the web interface after authenticating.

### List API Tokens

**GET** `/api/tokens`

**Authentication required** (session-based)

List all API tokens for the current user.

Example:
```bash
# Must be authenticated via web session
curl http://localhost:5000/api/tokens -b cookies.txt
```

Response:
```json
[
  {
    "id": 1,
    "name": "CI/CD Pipeline",
    "token": "abcdef1234567890...",
    "created_at": "2025-11-20T10:00:00Z",
    "last_used_at": "2025-11-27T08:30:00Z"
  }
]
```

---

### Create API Token

**POST** `/api/tokens`

**Authentication required** (session-based)

Create a new API token. Maximum 10 tokens per user.

Request:
```json
{
  "name": "Automation Script"
}
```

Example:
```bash
# Must be authenticated via web session
curl -X POST http://localhost:5000/api/tokens \
  -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{"name":"Automation Script"}'
```

Response:
```json
{
  "id": 1,
  "name": "Automation Script",
  "token": "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
  "created_at": "2025-11-27T08:35:00Z",
  "last_used_at": null
}
```

**Important**: Save the token value immediately. It cannot be retrieved later. The token shown above is an example placeholder only.

---

### Delete API Token

**DELETE** `/api/tokens/:id`

**Authentication required** (session-based)

Delete an API token. This immediately revokes access for that token.

Example:
```bash
# Must be authenticated via web session
curl -X DELETE http://localhost:5000/api/tokens/1 -b cookies.txt
```

Response:
```json
{"message": "token deleted"}
```

---

## Rate Limiting

### Benchmark Uploads
- **Limit**: 5 uploads per 10 minutes per user
- **Response**: HTTP 429 with `retry_after_secs`

### Admin Login
- **Limit**: 3 failed attempts in 10 minutes (global)
- **Response**: HTTP 429 with `retry_after_secs`

---

## Error Responses

```json
{"error": "Error message"}
```

HTTP status codes:
- 200 - Success
- 201 - Created
- 400 - Bad Request
- 401 - Unauthorized
- 403 - Forbidden
- 404 - Not Found
- 429 - Rate Limit Exceeded
- 500 - Internal Server Error

---

## Complete Workflow Example

### Using API Tokens (Recommended)

```bash
# 1. Create API token via web interface first
# - Login to web UI (Discord OAuth or admin login)
# - Navigate to Settings/Profile
# - Create a new API token and copy it

# 2. Upload benchmark
curl -X POST http://localhost:5000/api/benchmarks \
  -H "Authorization: Bearer your_api_token_here" \
  -F "title=Cyberpunk 2077" \
  -F "files=@benchmark.csv"

# 3. Get benchmark (ID from response)
curl http://localhost:5000/api/benchmarks/1

# 4. Download as ZIP
curl http://localhost:5000/api/benchmarks/1/download -o benchmark.zip

# 5. Update benchmark
curl -X PUT http://localhost:5000/api/benchmarks/1 \
  -H "Authorization: Bearer your_api_token_here" \
  -H "Content-Type: application/json" \
  -d '{"description":"Updated description"}'

# 6. Delete benchmark
curl -X DELETE http://localhost:5000/api/benchmarks/1 \
  -H "Authorization: Bearer your_api_token_here"
```

### Using Session Authentication (Web UI)

For interactive web UI usage, session cookies are used automatically:

```bash
# 1. Admin login
curl -X POST http://localhost:5000/auth/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' \
  -c cookies.txt

# 2. Create API token for programmatic access
curl -X POST http://localhost:5000/api/tokens \
  -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{"name":"Automation Script"}'

# 3. Logout
curl -X POST http://localhost:5000/auth/logout -b cookies.txt
```
