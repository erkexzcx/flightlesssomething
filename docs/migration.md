# Migration Guide

Guide for migrating data from the old [flightlesssomething](https://github.com/erkexzcx/flightlesssomething) project.

**Note:** As of the current version, migration from the old schema happens automatically on server startup. You no longer need a separate migration tool.

## What Gets Migrated

- **Users**: All user accounts with Discord IDs and usernames
- **Benchmarks**: Metadata (title, description, timestamps)
- **Benchmark Data**: Binary files with performance metrics (already in place)
- **Metadata Files**: New `.meta` files generated automatically during migration

## Schema Changes

### Old Project
- Database: `database.db`
- User fields: DiscordID, Username
- Benchmark fields: UserID, Title, Description, AiSummary

### New Project
- Database: `flightlesssomething.db`
- User fields: DiscordID, Username, IsAdmin, IsBanned, activity timestamps
- Benchmark fields: UserID, Title, Description (no AiSummary)
- New tables: APIToken, AuditLog, SchemaVersion

### Migration Notes
- Users migrated with `IsAdmin=false`, `IsBanned=false`
- Description limit increased: 500 → 5000 chars
- AiSummary field discarded (not used)
- Schema version tracking added to prevent re-migration
- Migration is idempotent and safe to re-run

## Prerequisites

1. Stop the old server
2. Backup your old data directory
3. Access to old project's data directory

## Automatic Migration

The new application automatically detects and migrates old databases on startup.

### Migration Steps

1. **Prepare Your Data**
   
   Copy your old data directory or rename it to use with the new server:
   ```bash
   # Example: Use existing data directory
   cp -r /path/to/old/data /path/to/data-for-new-server
   ```

2. **Start the New Server**
   
   Simply start the new server pointing to your old data directory:
   ```bash
   ./server \
     -bind="0.0.0.0:5000" \
     -data-dir="/path/to/data-for-new-server" \
     -session-secret="your-secret" \
     -discord-client-id="your-id" \
     -discord-client-secret="your-secret" \
     -discord-redirect-url="http://localhost:5000/auth/login/callback" \
     -admin-username="admin" \
     -admin-password="admin"
   ```

3. **Watch the Logs**
   
   The server will detect the old database file and automatically migrate:
   ```
   2024/11/29 10:00:00 Found old database.db, will migrate to flightlesssomething.db
   2024/11/29 10:00:00 Migrating from database.db to flightlesssomething.db...
   2024/11/29 10:00:00 Starting migration from /path/to/database.db...
   2024/11/29 10:00:00 Migrating users from old database...
   2024/11/29 10:00:00 Found 42 users to migrate
   2024/11/29 10:00:00   Migrating user: JohnDoe (ID: 1, Discord: 123456789)
   2024/11/29 10:00:00     Migrated successfully
   ...
   2024/11/29 10:00:01 Migrating benchmarks from old database...
   2024/11/29 10:00:01 Found 987 benchmarks to migrate
   2024/11/29 10:00:01   [1/987] Migrating benchmark: Cyberpunk 2077 (ID: 1)
   2024/11/29 10:00:01     Successfully migrated (3 runs)
   ...
   === Migration Summary ===
   Users migrated: 42
   Benchmarks attempted: 987
   Benchmarks succeeded: 987
   Benchmarks failed: 0
   =========================
   Migration from old database file completed successfully!
   ```

4. **Verify Migration**
   
   After migration, you'll see both database files:
   ```bash
   ls -la /path/to/data/
   # database.db              <- Old database (kept as backup)
   # flightlesssomething.db   <- New database with migrated data
   # benchmarks/              <- Benchmark data files (unchanged)
   ```
   
   Check your data:
   ```bash
   # Count migrated data in new database
   sqlite3 /path/to/data/flightlesssomething.db \
     "SELECT COUNT(*) FROM users;"
   sqlite3 /path/to/data/flightlesssomething.db \
     "SELECT COUNT(*) FROM benchmarks;"
   ```

## How It Works

1. **File Detection**: On startup, checks if `database.db` exists in the data directory
2. **File Migration**: If `database.db` found and `flightlesssomething.db` doesn't exist, migrates from old file
3. **Schema Detection**: Otherwise, checks for `schema_versions` table in `flightlesssomething.db`
4. **Schema Migration**: If old schema detected (no version table + `ai_summary` column), performs in-place upgrade
5. **Version Tracking**: Sets schema version to prevent re-migration
6. **Normal Startup**: After migration, server starts normally

### Migration Paths

**Path 1: File-based migration (v0.20 or earlier)**
```
database.db exists + flightlesssomething.db missing
→ Migrate from database.db to flightlesssomething.db
→ Set schema version to 1
```

**Path 2: In-place schema migration**
```
flightlesssomething.db exists with old schema (no schema_versions table)
→ Upgrade schema in place
→ Set schema version to 1
```

**Path 3: New database**
```
No existing database
→ Create flightlesssomething.db with current schema
→ Set schema version to 1
```

## Post-Migration

### Set Admin Users

After migration, set admin status manually if needed:

```bash
sqlite3 /path/to/data-for-new-server/flightlesssomething.db
UPDATE users SET is_admin=1 WHERE discord_id='YOUR_DISCORD_ID';
.quit
```

Or use the admin panel in the web UI (log in with the system admin account first).

### System Admin Account

The system creates an admin account with `discord_id='admin'` on first startup. This is separate from Discord users and can be used to manage the application.

## Safety

- Migration **never modifies** old data structure (only adds columns)
- Schema version prevents accidental re-migration
- Migration is idempotent - safe to run multiple times
- Original IDs are preserved for users and benchmarks
- Existing benchmark data files remain in place

## Troubleshooting

### Migration Doesn't Start

If you see no migration logs, the database might already be in the new format. Check:
```bash
sqlite3 /path/to/data/flightlesssomething.db ".schema schema_versions"
```

If the table exists, migration already happened.

### "Failed to read data file"

Some benchmark data files may be corrupted. The migration will:
- Log the error
- Skip the corrupted benchmark
- Continue with remaining benchmarks
- Report failures in the summary

Check the logs for specific benchmark IDs that failed.

### "User ID not found"

A benchmark references a non-existent user (data inconsistency). The migration will:
- Log a warning
- Skip the benchmark
- Continue with remaining benchmarks

This is rare but can happen with data corruption.

### Disk Space

You don't need extra disk space since the migration happens in-place. However, it's always recommended to have a backup of your data directory before migration.

### Re-running Migration

If migration fails partway through, you can safely restart the server. The migration logic:
- Checks if each user/benchmark already exists
- Skips already-migrated items
- Completes only the remaining items

## Need Help?

If issues occur:
1. Check error messages in server logs
2. Verify old data structure matches expected format
3. Ensure proper file permissions
4. Check available disk space
5. Open a GitHub issue with error details and relevant log output
