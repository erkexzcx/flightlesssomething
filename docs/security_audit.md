# Security Audit Report: API Endpoints & Permissions

**Date:** 2025-11-27  
**Auditor:** Security Analysis Agent  
**Repository:** erkexzcx/flightlesssomething

## Executive Summary

This document provides a comprehensive security audit of all HTTP endpoints in the FlightlessSomething application. Each endpoint has been analyzed for proper authentication and authorization controls. The application implements a well-structured permission system with three access levels: **Anonymous**, **Authenticated Users**, and **Admins**.

### Overall Security Posture: ✅ STRONG

The application demonstrates solid security practices:
- Clear separation between public, authenticated, and admin endpoints
- Proper middleware enforcement (RequireAuth, RequireAdmin, RequireAuthOrToken)
- Ban status checking for sensitive operations
- Rate limiting on critical endpoints
- Audit logging for admin actions
- API token authentication for programmatic access

---

## Authentication Methods

The application supports three authentication methods:

1. **Session-based (Web)** - Discord OAuth login via `/auth/login` and `/auth/login/callback`
2. **Admin Credentials** - Username/password login via `/auth/admin/login`
3. **API Tokens** - Bearer token authentication via `Authorization: Bearer <token>` header

---

## Endpoint Analysis

### 1. Public Endpoints (Anonymous Access)

These endpoints are accessible to anyone without authentication:

#### `GET /health`
- **Purpose:** Health check endpoint
- **Who can access:** Anonymous (anyone)
- **What it does:** Returns server status and version
- **What it cannot do:** N/A - Read-only health check
- **Security verdict:** ✅ Safe - No sensitive data exposed

#### `GET /auth/login`
- **Purpose:** Initiates Discord OAuth flow
- **Who can access:** Anonymous (anyone)
- **What it does:** 
  - Redirects to Discord OAuth if not logged in
  - Returns "already logged in" message if authenticated
- **What it cannot do:** Cannot bypass authentication or access protected resources
- **Security verdict:** ✅ Safe - Standard OAuth flow

#### `GET /auth/login/callback`
- **Purpose:** Handles Discord OAuth callback
- **Who can access:** Anonymous (OAuth redirect)
- **What it does:**
  - Validates OAuth state token
  - Creates or updates user account
  - Establishes authenticated session
  - **CRITICAL CHECK:** Prevents banned users from logging in (line 141-144 in auth.go)
- **What it cannot do:** Cannot grant admin privileges (handled separately)
- **Security verdict:** ✅ Safe - Includes ban check, proper state validation

#### `POST /auth/admin/login`
- **Purpose:** Admin login with username/password
- **Who can access:** Anonymous (login endpoint)
- **What it does:**
  - Validates admin credentials against config
  - Creates/retrieves admin user with admin flag
  - **Rate Limiting:** 3 failed attempts locks for 10 minutes (global)
- **What it cannot do:** Cannot be bypassed without correct credentials
- **Security verdict:** ✅ Safe - Rate limited, credentials validated against config

#### `POST /auth/logout`
- **Purpose:** Logs out current user
- **Who can access:** Anonymous (no auth required)
- **What it does:** Clears session and cookie
- **What it cannot do:** N/A
- **Security verdict:** ✅ Safe - No sensitive operations

#### `GET /api/auth/me`
- **Purpose:** Returns current user info
- **Who can access:** Anonymous
- **What it does:**
  - Returns 401 if not authenticated
  - Returns user_id, username, is_admin if authenticated
- **What it cannot do:** Cannot access other users' data
- **Security verdict:** ✅ Safe - Only returns own session data

#### `GET /api/benchmarks`
- **Purpose:** List all benchmarks with pagination and filtering
- **Who can access:** Anonymous (anyone)
- **What it does:**
  - Returns paginated list of all benchmarks
  - Supports filtering by user_id and search term
  - Includes public metadata and user info
- **What it cannot do:** Cannot modify benchmarks or access admin-only data
- **Security verdict:** ✅ Safe - Public read-only access to benchmark listings

#### `GET /api/benchmarks/:id`
- **Purpose:** Get single benchmark details
- **Who can access:** Anonymous (anyone)
- **What it does:** Returns benchmark metadata including title, description, user info
- **What it cannot do:** Cannot access the actual benchmark data (separate endpoint)
- **Security verdict:** ✅ Safe - Public metadata only

#### `GET /api/benchmarks/:id/data`
- **Purpose:** Get benchmark performance data
- **Who can access:** Anonymous (anyone)
- **What it does:** Returns full benchmark data arrays (FPS, CPU, GPU, etc.)
- **What it cannot do:** Cannot modify data
- **Security verdict:** ✅ Safe - Public read access, verify benchmark exists before returning

#### `GET /api/benchmarks/:id/download`
- **Purpose:** Download benchmark data as ZIP
- **Who can access:** Anonymous (anyone)
- **What it does:** Exports benchmark data to ZIP with CSV files
- **What it cannot do:** Cannot modify data
- **Security verdict:** ✅ Safe - Public read access, proper file handling

---

### 2. Authenticated Endpoints (Require Login or API Token)

These endpoints require authentication via session OR API token (middleware: `RequireAuthOrToken`):

#### `POST /api/benchmarks`
- **Purpose:** Create new benchmark
- **Who can access:** Authenticated users
- **What it does:**
  - Validates user is not banned
  - **Rate Limiting:** 5 benchmarks per 10 minutes (per user, admins exempt)
  - Accepts multipart form with files, title, description
  - Creates benchmark record and stores data
  - Logs creation in audit log
- **What it cannot do:**
  - Banned users cannot create benchmarks
  - Non-admins are rate limited
  - Cannot create benchmarks for other users
- **Security verdict:** ✅ Safe - Ban check, rate limiting, user ownership enforced

#### `PUT /api/benchmarks/:id`
- **Purpose:** Update existing benchmark
- **Who can access:** Authenticated users (owner or admin)
- **What it does:**
  - **Ownership Check:** Only owner or admin can update (line 261-264 in benchmarks.go)
  - Banned users cannot update (admins exempt)
  - Updates title, description, and/or run labels
  - Logs update in audit log
- **What it cannot do:**
  - Regular users cannot update others' benchmarks
  - Banned users (non-admin) cannot update
- **Security verdict:** ✅ Safe - Proper ownership validation

#### `DELETE /api/benchmarks/:id`
- **Purpose:** Delete benchmark
- **Who can access:** Authenticated users (owner or admin)
- **What it does:**
  - **Ownership Check:** Only owner or admin can delete (line 369-372 in benchmarks.go)
  - Banned users cannot delete (admins exempt)
  - Deletes data file and database record
  - Logs deletion in audit log
- **What it cannot do:**
  - Regular users cannot delete others' benchmarks
  - Banned users (non-admin) cannot delete
- **Security verdict:** ✅ Safe - Proper ownership validation

#### `DELETE /api/benchmarks/:id/runs/:run_index`
- **Purpose:** Delete specific run from benchmark
- **Who can access:** Authenticated users (owner or admin)
- **What it does:**
  - **Ownership Check:** Only owner or admin can delete run (line 482-485 in benchmarks.go)
  - Banned users cannot delete (admins exempt)
  - Prevents deletion of last run
  - Updates benchmark data
  - Logs update in audit log
- **What it cannot do:**
  - Regular users cannot delete runs from others' benchmarks
  - Cannot delete the last remaining run
- **Security verdict:** ✅ Safe - Proper ownership validation, prevents invalid state

#### `POST /api/benchmarks/:id/runs`
- **Purpose:** Add new runs to existing benchmark
- **Who can access:** Authenticated users (owner or admin)
- **What it does:**
  - **Ownership Check:** Only owner or admin can add runs (line 576-579 in benchmarks.go)
  - Banned users cannot add (admins exempt)
  - Accepts multipart form with benchmark files
  - Appends new runs to existing data
  - Logs update in audit log
- **What it cannot do:**
  - Regular users cannot add runs to others' benchmarks
- **Security verdict:** ✅ Safe - Proper ownership validation

#### `GET /api/tokens`
- **Purpose:** List user's API tokens
- **Who can access:** Authenticated users
- **What it does:**
  - Returns only tokens belonging to current user (line 22 in api_tokens.go)
  - Ordered by creation date
- **What it cannot do:**
  - Cannot see other users' tokens
  - Cannot access tokens without authentication
- **Security verdict:** ✅ Safe - User isolation enforced

#### `POST /api/tokens`
- **Purpose:** Create new API token
- **Who can access:** Authenticated users
- **What it does:**
  - Creates API token for current user
  - Enforces max 10 tokens per user (line 52-55 in api_tokens.go)
  - Generates cryptographically random token
- **What it cannot do:**
  - Cannot create tokens for other users
  - Cannot exceed 10 token limit
- **Security verdict:** ✅ Safe - User isolation, rate limiting

#### `DELETE /api/tokens/:id`
- **Purpose:** Delete API token
- **Who can access:** Authenticated users
- **What it does:**
  - **Ownership Check:** Verifies token belongs to current user (line 87-90 in api_tokens.go)
  - Deletes only user's own tokens
- **What it cannot do:**
  - Cannot delete other users' tokens
- **Security verdict:** ✅ Safe - Proper ownership validation

---

### 3. Admin-Only Endpoints

These endpoints require **both** authentication AND admin privileges (middleware: `RequireAuth()` + `RequireAdmin()`):

#### `GET /api/admin/users`
- **Purpose:** List all users with statistics
- **Who can access:** Admins only
- **What it does:**
  - Returns paginated list of all users
  - Includes benchmark count and API token count per user
  - Supports search filter
  - Sorted by benchmark count
- **What it cannot do:**
  - **Regular users CANNOT access this** (403 Forbidden)
  - Anonymous users CANNOT access this (401 Unauthorized)
- **Security verdict:** ✅ Safe - Properly restricted to admins

#### `DELETE /api/admin/users/:id`
- **Purpose:** Delete user account
- **Who can access:** Admins only
- **What it does:**
  - Deletes user account
  - Optional query param `delete_data=true` also deletes all benchmark files
  - Cascade deletes benchmarks and tokens via GORM
  - Logs deletion in audit log
- **What it cannot do:**
  - **Regular users CANNOT delete users** (403 Forbidden)
  - Anonymous users CANNOT access this (401 Unauthorized)
- **Security verdict:** ✅ Safe - Properly restricted to admins

#### `DELETE /api/admin/users/:id/benchmarks`
- **Purpose:** Delete all benchmarks for a user
- **Who can access:** Admins only
- **What it does:**
  - Deletes all benchmark files and database records for specified user
  - Logs action in audit log
- **What it cannot do:**
  - **Regular users CANNOT delete others' benchmarks** (403 Forbidden)
  - Anonymous users CANNOT access this (401 Unauthorized)
- **Security verdict:** ✅ Safe - Properly restricted to admins

#### `PUT /api/admin/users/:id/ban`
- **Purpose:** Ban or unban a user
- **Who can access:** Admins only
- **What it does:**
  - Sets user's `is_banned` flag
  - Logs ban/unban action in audit log
  - **Effect:** Banned users cannot login (checked in callback) or perform operations
- **What it cannot do:**
  - **Regular users CANNOT ban users** (403 Forbidden)
  - Anonymous users CANNOT access this (401 Unauthorized)
- **Security verdict:** ✅ Safe - Properly restricted to admins

#### `PUT /api/admin/users/:id/admin`
- **Purpose:** Grant or revoke admin privileges
- **Who can access:** Admins only
- **What it does:**
  - Sets user's `is_admin` flag
  - Logs admin grant/revoke action in audit log
  - **CRITICAL:** Only existing admins can make other users admins
- **What it cannot do:**
  - **Regular users CANNOT grant admin privileges** (403 Forbidden)
  - Anonymous users CANNOT access this (401 Unauthorized)
- **Security verdict:** ✅ Safe - Properly restricted to admins, prevents privilege escalation

#### `GET /api/admin/logs`
- **Purpose:** View audit logs
- **Who can access:** Admins only
- **What it does:**
  - Returns paginated audit logs
  - Supports filtering by user_id, action, target_type
  - Includes user information for each log entry
- **What it cannot do:**
  - **Regular users CANNOT view audit logs** (403 Forbidden)
  - Anonymous users CANNOT access this (401 Unauthorized)
- **Security verdict:** ✅ Safe - Properly restricted to admins

---

## Middleware Security Analysis

### RequireAuth()
- **Location:** `auth.go:272-288`
- **Purpose:** Ensures user is authenticated via session
- **Validation:**
  - ✅ Checks session for UserID
  - ✅ Returns 401 if not authenticated
  - ✅ Sets user context for downstream handlers
- **Security verdict:** ✅ Properly implemented

### RequireAdmin()
- **Location:** `auth.go:291-310`
- **Purpose:** Ensures user has admin privileges
- **Validation:**
  - ✅ Checks session for IsAdmin flag
  - ✅ Returns 403 if not admin
  - ✅ Verifies boolean type safety
- **Security verdict:** ✅ Properly implemented

### RequireAuthOrToken()
- **Location:** `api_tokens.go:111-184`
- **Purpose:** Accepts either session or API token authentication
- **Validation:**
  - ✅ First checks session authentication
  - ✅ Falls back to Bearer token validation
  - ✅ Validates token exists in database
  - ✅ Updates last used timestamps
  - ✅ Returns 401 if neither method succeeds
  - ✅ Sets user context including IsAdmin flag from token user
- **Security verdict:** ✅ Properly implemented

---

## Security Vulnerabilities & Recommendations

### ✅ No Critical Vulnerabilities Found

The application demonstrates strong security practices:

1. **Proper Authorization Checks**
   - All sensitive endpoints verify ownership or admin status
   - No endpoints allow users to escalate their own privileges
   - Admin endpoints are properly gated with RequireAdmin middleware

2. **Ban System Works Correctly**
   - Banned users blocked at login (auth.go:141-144)
   - Banned users blocked from creating/modifying content
   - Admins can still operate when banned (intentional override)

3. **Rate Limiting**
   - Admin login: 3 attempts per 10 minutes (global)
   - Benchmark uploads: 5 per 10 minutes per user (admins exempt)

4. **Audit Logging**
   - All admin actions are logged
   - Logs track user deletions, bans, admin grants, etc.

### 🟡 Minor Recommendations

#### 1. Self-Admin Revocation Protection
**Location:** `admin.go:210-248` (HandleToggleUserAdmin)

**Issue:** An admin could accidentally revoke their own admin privileges.

**Current State:** The endpoint allows an admin to set `is_admin=false` on their own account.

**Recommendation:** Add a check to prevent admins from revoking their own admin status:

```go
// Prevent self-demotion
currentUserID := c.GetUint("UserID")
if user.ID == currentUserID && !req.IsAdmin {
    c.JSON(http.StatusBadRequest, gin.H{"error": "cannot revoke your own admin privileges"})
    return
}
```

#### 2. Self-Ban Protection
**Location:** `admin.go:169-207` (HandleBanUser)

**Issue:** An admin could accidentally ban themselves.

**Recommendation:** Add a check to prevent admins from banning their own account:

```go
// Prevent self-ban
currentUserID := c.GetUint("UserID")
if user.ID == currentUserID && req.Banned {
    c.JSON(http.StatusBadRequest, gin.H{"error": "cannot ban your own account"})
    return
}
```

#### 3. Self-Deletion Protection
**Location:** `admin.go:78-124` (HandleDeleteUser)

**Issue:** An admin could accidentally delete their own account.

**Recommendation:** Add a check to prevent admins from deleting their own account:

```go
// Prevent self-deletion
currentUserID := c.GetUint("UserID")
if user.ID == currentUserID {
    c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete your own account"})
    return
}
```

#### 4. API Token Admin Inheritance
**Location:** `api_tokens.go:178-180`

**Current Behavior:** API tokens inherit the `IsAdmin` flag from the user who created them.

**Observation:** This is correct behavior - tokens should carry the privileges of their owner. However, if an admin is demoted, their existing tokens retain admin privileges until the next API call refreshes the data from the database.

**Recommendation:** Consider documenting this behavior or implementing token invalidation when admin status changes.

---

## Permission Matrix

| Endpoint | Anonymous | User (Owner) | User (Non-Owner) | Admin | Banned User |
|----------|-----------|--------------|------------------|-------|-------------|
| **Public Endpoints** |
| `GET /health` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GET /auth/login` | ✅ | ✅ | ✅ | ✅ | ❌ Ban Check |
| `GET /auth/login/callback` | ✅ | ✅ | ✅ | ✅ | ❌ Ban Check |
| `POST /auth/admin/login` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `POST /auth/logout` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GET /api/auth/me` | ❌ 401 | ✅ | ✅ | ✅ | ✅ |
| `GET /api/benchmarks` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GET /api/benchmarks/:id` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GET /api/benchmarks/:id/data` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GET /api/benchmarks/:id/download` | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Authenticated Endpoints** |
| `POST /api/benchmarks` | ❌ 401 | ✅ | ✅ | ✅ | ❌ 403 |
| `PUT /api/benchmarks/:id` | ❌ 401 | ✅ Owner | ❌ 403 | ✅ | ❌ 403 |
| `DELETE /api/benchmarks/:id` | ❌ 401 | ✅ Owner | ❌ 403 | ✅ | ❌ 403 |
| `DELETE /api/benchmarks/:id/runs/:idx` | ❌ 401 | ✅ Owner | ❌ 403 | ✅ | ❌ 403 |
| `POST /api/benchmarks/:id/runs` | ❌ 401 | ✅ Owner | ❌ 403 | ✅ | ❌ 403 |
| `GET /api/tokens` | ❌ 401 | ✅ Own Only | ✅ Own Only | ✅ Own Only | ✅ Own Only |
| `POST /api/tokens` | ❌ 401 | ✅ | ✅ | ✅ | ✅ |
| `DELETE /api/tokens/:id` | ❌ 401 | ✅ Own Only | ✅ Own Only | ✅ Own Only | ✅ Own Only |
| **Admin Endpoints** |
| `GET /api/admin/users` | ❌ 401 | ❌ 403 | ❌ 403 | ✅ | ❌ 403 |
| `DELETE /api/admin/users/:id` | ❌ 401 | ❌ 403 | ❌ 403 | ✅ | ❌ 403 |
| `DELETE /api/admin/users/:id/benchmarks` | ❌ 401 | ❌ 403 | ❌ 403 | ✅ | ❌ 403 |
| `PUT /api/admin/users/:id/ban` | ❌ 401 | ❌ 403 | ❌ 403 | ✅ | ❌ 403 |
| `PUT /api/admin/users/:id/admin` | ❌ 401 | ❌ 403 | ❌ 403 | ✅ | ❌ 403 |
| `GET /api/admin/logs` | ❌ 401 | ❌ 403 | ❌ 403 | ✅ | ❌ 403 |

**Legend:**
- ✅ = Allowed
- ❌ = Denied (with HTTP status code)
- "Own Only" = Can only access their own resources
- "Owner" = Can only access if they own the resource

---

## Conclusion

The FlightlessSomething application has a **well-implemented security model** with proper separation of concerns between anonymous, authenticated, and admin access levels. All endpoints have appropriate authorization checks, and there are no critical security vulnerabilities that would allow privilege escalation or unauthorized data access.

The three minor recommendations (self-admin revocation, self-ban, self-deletion protection) are defensive measures to prevent accidental misuse rather than exploitable vulnerabilities.

### Summary of User Capabilities

**Anonymous Users:**
- Can view all benchmarks and their data (read-only)
- Can log in via Discord OAuth or admin credentials
- Cannot create, modify, or delete any content

**Authenticated Users:**
- All anonymous capabilities
- Can create benchmarks (rate limited: 5 per 10 minutes)
- Can modify/delete ONLY their own benchmarks
- Can manage ONLY their own API tokens (max 10)
- Cannot access or modify other users' content
- Cannot access admin functions

**Admins:**
- All authenticated user capabilities
- Can modify/delete ANY user's benchmarks
- Can view all users and their statistics
- Can ban/unban users
- Can grant/revoke admin privileges
- Can delete users and their data
- Can view audit logs
- Exempt from rate limiting on benchmark uploads
- Can perform operations even when banned (intentional override)

**Banned Users:**
- Cannot log in via Discord OAuth (blocked in callback)
- Cannot create or modify content
- Existing sessions are not terminated (until next auth check)
- Admins with banned flag can still operate (intentional)
