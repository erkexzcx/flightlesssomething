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
   
   The server will detect the old schema and automatically migrate:
   ```
   2024/11/29 10:00:00 Detected old database schema (version 0)
   2024/11/29 10:00:00 Starting migration from old schema to current schema...
   2024/11/29 10:00:00 Migrating users from old schema...
   2024/11/29 10:00:00 Found 42 users to migrate
   2024/11/29 10:00:00   Migrating user: JohnDoe (ID: 1, Discord: 123456789)
   2024/11/29 10:00:00     Migrated successfully
   ...
   2024/11/29 10:00:01 Migrating benchmarks from old schema...
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
   Migration from old schema completed successfully!
   ```

4. **Verify Migration**
   
   Check your data:
   ```bash
   # Check files
   ls -la /path/to/data-for-new-server/
   ls -la /path/to/data-for-new-server/benchmarks/
   
   # Count migrated data
   sqlite3 /path/to/data-for-new-server/flightlesssomething.db \
     "SELECT COUNT(*) FROM users;"
   sqlite3 /path/to/data-for-new-server/flightlesssomething.db \
     "SELECT COUNT(*) FROM benchmarks;"
   ```

## How It Works

1. **Schema Detection**: On startup, the application checks for a `schema_versions` table
2. **Old Schema Detected**: If the table doesn't exist but old tables do, migration begins
3. **Data Migration**: Users and benchmarks are migrated with ID preservation
4. **Version Tracking**: A schema version is stored to prevent re-migration
5. **Normal Startup**: After migration, the server starts normally

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
