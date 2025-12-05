# API Documentation

REST API for managing benchmark data.

## Base URL

```
http://your-server:5000
```

## Authentication

The API uses API tokens for programmatic access. API tokens provide secure, revocable access without exposing credentials.

**Token Format**: Include token in `Authorization` header as `Bearer <token>`

```bash
curl http://localhost:5000/api/benchmarks \
  -H "Authorization: Bearer your_api_token_here"
```

**Token Management**:
1. Log in to the web interface (Discord OAuth or admin credentials)
2. Navigate to API Tokens page
3. Create a new token with a descriptive name
4. Copy the token immediately - you can reveal it later by clicking the eye icon

**Benefits**:
- Designed for automated tools and scripts
- Can be revoked individually without affecting other tokens
- Track last usage per token
- Set descriptive names for organization
- Maximum 10 tokens per user

---

## User Endpoints

These endpoints are available for all authenticated users.

### Health Check

**GET** `/health`

Check server status (no authentication required).

```bash
curl http://localhost:5000/health
```

Response:
```json
{"status": "ok"}
```

---

### List Benchmarks

**GET** `/api/benchmarks`

List all benchmarks with pagination, search, and sorting (no authentication required).

Query parameters:
- `page` (integer) - Page number (default: 1)
- `per_page` (integer) - Results per page (default: 10, max: 100)
- `search` (string) - Search in title/description
- `user_id` (integer) - Filter by user ID
- `sort_by` (string) - Sort field: `title`, `created_at`, or `updated_at` (default: `created_at`)
- `sort_order` (string) - Sort order: `asc` or `desc` (default: `desc`)

Example:
```bash
curl "http://localhost:5000/api/benchmarks?page=1&per_page=20&sort_by=title&sort_order=asc"
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

Get benchmark details (no authentication required).

Example:
```bash
curl http://localhost:5000/api/benchmarks/1
```

---

### Get Benchmark Data

**GET** `/api/benchmarks/:id/data`

Download benchmark data (compressed binary, no authentication required).

Example:
```bash
curl http://localhost:5000/api/benchmarks/1/data -o benchmark.dat
```

---

### Download Benchmark (ZIP)

**GET** `/api/benchmarks/:id/download`

Download benchmark as ZIP with CSV files in MangoHud format (no authentication required).

Example:
```bash
curl http://localhost:5000/api/benchmarks/1/download -o benchmark.zip
```

---

### Create Benchmark

**POST** `/api/benchmarks`

**Authentication required** - Use API token.

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

**Rate Limiting**: 5 uploads per 10 minutes per user (admins exempt)

---

### Update Benchmark

**PUT** `/api/benchmarks/:id`

**Authentication required** - Use API token. Only owner or admin can update.

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

**Authentication required** - Use API token. Only owner or admin can delete.

Delete benchmark and all associated data.

Example:
```bash
curl -X DELETE http://localhost:5000/api/benchmarks/1 \
  -H "Authorization: Bearer your_api_token_here"
```

---

### Delete Benchmark Run

**DELETE** `/api/benchmarks/:id/runs/:run_index`

**Authentication required** - Use API token. Only owner or admin can delete.

Delete a specific run from a benchmark. Cannot delete the last remaining run.

Example:
```bash
curl -X DELETE http://localhost:5000/api/benchmarks/1/runs/0 \
  -H "Authorization: Bearer your_api_token_here"
```

---

### Add Benchmark Runs

**POST** `/api/benchmarks/:id/runs`

**Authentication required** - Use API token. Only owner or admin can add runs.

Add new runs to an existing benchmark.

Request: `multipart/form-data`
- `files` (file[], required) - Benchmark CSV/HML files

Example:
```bash
curl -X POST http://localhost:5000/api/benchmarks/1/runs \
  -H "Authorization: Bearer your_api_token_here" \
  -F "files=@benchmark3.csv" \
  -F "files=@benchmark4.csv"
```

---

## Admin Endpoints

These endpoints are restricted to admin users only.

### List Users

**GET** `/api/admin/users`

**Admin authentication required** - Use admin API token.

List all registered users with statistics.

Query parameters:
- `page` (integer) - Page number (default: 1)
- `per_page` (integer) - Results per page (default: 10, max: 100)
- `search` (string) - Search in username

Example:
```bash
curl http://localhost:5000/api/admin/users \
  -H "Authorization: Bearer your_admin_api_token_here"
```

---

### Delete User

**DELETE** `/api/admin/users/:id`

**Admin authentication required** - Use admin API token.

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

### Delete User's Benchmarks

**DELETE** `/api/admin/users/:id/benchmarks`

**Admin authentication required** - Use admin API token.

Delete all benchmarks for a specific user.

Example:
```bash
curl -X DELETE http://localhost:5000/api/admin/users/1/benchmarks \
  -H "Authorization: Bearer your_admin_api_token_here"
```

---

### Ban/Unban User

**PUT** `/api/admin/users/:id/ban`

**Admin authentication required** - Use admin API token.

Ban or unban a user. Banned users cannot log in or create/modify content.

Request:
```json
{
  "banned": true
}
```

Example:
```bash
# Ban user
curl -X PUT http://localhost:5000/api/admin/users/1/ban \
  -H "Authorization: Bearer your_admin_api_token_here" \
  -H "Content-Type: application/json" \
  -d '{"banned":true}'

# Unban user
curl -X PUT http://localhost:5000/api/admin/users/1/ban \
  -H "Authorization: Bearer your_admin_api_token_here" \
  -H "Content-Type: application/json" \
  -d '{"banned":false}'
```

---

### Grant/Revoke Admin

**PUT** `/api/admin/users/:id/admin`

**Admin authentication required** - Use admin API token.

Grant or revoke admin privileges for a user.

Request:
```json
{
  "is_admin": true
}
```

Example:
```bash
# Grant admin privileges
curl -X PUT http://localhost:5000/api/admin/users/1/admin \
  -H "Authorization: Bearer your_admin_api_token_here" \
  -H "Content-Type: application/json" \
  -d '{"is_admin":true}'

# Revoke admin privileges
curl -X PUT http://localhost:5000/api/admin/users/1/admin \
  -H "Authorization: Bearer your_admin_api_token_here" \
  -H "Content-Type: application/json" \
  -d '{"is_admin":false}'
```

---

### View Audit Logs

**GET** `/api/admin/logs`

**Admin authentication required** - Use admin API token.

View audit logs for admin actions.

Query parameters:
- `page` (integer) - Page number (default: 1)
- `per_page` (integer) - Results per page (default: 50, max: 100)
- `user_id` (integer) - Filter by user ID
- `action` (string) - Filter by action type
- `target_type` (string) - Filter by target type

Example:
```bash
curl http://localhost:5000/api/admin/logs \
  -H "Authorization: Bearer your_admin_api_token_here"
```

---

## Rate Limiting

### Benchmark Uploads
- **Limit**: 5 uploads per 10 minutes per user
- **Response**: HTTP 429 with `retry_after_secs`
- **Admin Exemption**: Admins are exempt from rate limiting

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

```bash
# 1. Create API token via web interface first
# - Login to web UI (Discord OAuth or admin login)
# - Navigate to API Tokens page
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
