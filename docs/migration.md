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
   Successfully removed old database.db file
   ```

4. **Verify Migration**
   
   After migration, only the new database file will remain:
   ```bash
   ls -la /path/to/data/
   # flightlesssomething.db   <- New database with migrated data
   # benchmarks/              <- Benchmark data files (unchanged)
   ```
   
   The old `database.db` file is automatically removed after successful migration.
   
   Check your data:
   ```bash
   # Count migrated data in new database
   sqlite3 /path/to/data/flightlesssomething.db \
     "SELECT COUNT(*) FROM users;"
   sqlite3 /path/to/data/flightlesssomething.db \
     "SELECT COUNT(*) FROM benchmarks;"
   ```

## How It Works

The migration system is designed to be future-proof with three distinct database formats:

### Format Detection

1. **Format 1 (v0.20 and earlier)**: `database.db` file with old schema
   - **Detected by**: File named `database.db` exists in data directory
   - **Action**: Migrate data from `database.db` to `flightlesssomething.db`, then delete `database.db`
   - **Result**: Clean transition to Format 3

2. **Format 2 (intermediate)**: `flightlesssomething.db` with old schema
   - **Detected by**: No `schema_versions` table but has `ai_summary` column in benchmarks
   - **Action**: In-place schema upgrade, add `schema_versions` table with version 1
   - **Result**: Upgrade to Format 3

3. **Format 3+ (current and future)**: `flightlesssomething.db` with version tracking
   - **Detected by**: `schema_versions` table exists with version number
   - **Action**: Apply incremental migrations if version < currentSchemaVersion
   - **Current version**: 3
   - **Version history**:
     - Version 1: Initial schema with version tracking
     - Version 2: Added RunNames and Specifications fields for enhanced search
     - Version 3: Migrated benchmark storage format from V1 to V2 (streaming-friendly)

### Migration Flow

```
database.db exists?
  ├─ Yes → Format 1 migration
  │   ├─ Migrate to flightlesssomething.db
  │   ├─ Set schema_versions to version 1
  │   └─ Delete database.db
  │
  └─ No → Check flightlesssomething.db
      ├─ schema_versions table exists?
      │   ├─ Yes → Format 3+
      │   │   ├─ Version = 1? → No migration needed
      │   │   └─ Version < 1? → Apply incremental migrations
      │   │
      │   └─ No → Check for old schema
      │       ├─ Has ai_summary column? → Format 2
      │       │   └─ In-place upgrade to Format 3
      │       │
      │       └─ No tables? → New database
      │           └─ Initialize with Format 3
```

### Safety Features

- Old `database.db` is only deleted after successful migration
- If deletion fails, a warning is logged (data is already migrated)
- Schema version tracking prevents re-migration
- Migration is idempotent - safe to re-run
- Original IDs preserved for relationships and file associations

## Storage Format Migration (V1 → V2)

Starting from version 3 of the schema, benchmark data files are automatically migrated from V1 to V2 format on first startup. This migration improves memory efficiency when viewing large benchmarks.

### What Gets Migrated

The migration converts benchmark data files (`.bin`) from:
- **V1 format**: Single gob-encoded array (requires loading entire dataset into memory)
- **V2 format**: Header + individually-encoded runs (enables true streaming)

### Automatic Migration Process

When you start the server with schema version < 3:

```
Database is at version 2, current version is 3. Running data migrations...
Migrating benchmark storage format from V1 to V2...
=== Starting Benchmark Storage Format Migration (V1 → V2) ===
Found 50 benchmark file(s) to check

Processing benchmark 1...
  Loading V1 format data...
  Converting to V2 format (100 runs)...
  Verifying conversion...
  ✓ Successfully migrated to V2 format

Processing benchmark 2...
  Already in V2 format - skipping

...

=== Storage Migration Summary ===
Total files found: 50
Successfully migrated: 35
Already V2 (skipped): 15
Failed: 0
==================================
Storage format migration completed successfully!
Successfully migrated to version 3
```

### Backup Files

**During migration, backup files are automatically created:**

Each benchmark data file is backed up as `<benchmark_id>.bin.v1.bak` before conversion. For example:
```
data/benchmarks/
├── 1.bin           # V2 format (migrated)
├── 1.bin.v1.bak    # V1 format (backup)
├── 2.bin           # V2 format (migrated)
├── 2.bin.v1.bak    # V1 format (backup)
└── ...
```

**Backup cleanup:**
- On **successful** migration: Backup file is automatically deleted
- On **failed** migration: Backup is automatically restored, conversion retried
- If backup deletion fails: Warning logged, but migration continues (backup remains on disk)

### Restoring from Backup

If you encounter issues after migration and need to restore the V1 format:

#### Option 1: Manual Restoration (Recommended)

1. **Stop the server** to prevent concurrent file access

2. **Identify backup files:**
   ```bash
   cd /path/to/data/benchmarks
   ls -lh *.v1.bak
   ```

3. **Restore specific benchmarks:**
   ```bash
   # For a specific benchmark (e.g., benchmark 42)
   mv 42.bin.v1.bak 42.bin
   ```

4. **Restore all benchmarks:**
   ```bash
   # Restore all V1 backups
   for f in *.v1.bak; do
       mv "$f" "${f%.v1.bak}"
   done
   ```

5. **Downgrade schema version** to prevent re-migration:
   ```bash
   sqlite3 /path/to/data/flightlesssomething.db
   UPDATE schema_versions SET version = 2 WHERE version = 3;
   .quit
   ```

6. **Restart the server** with the old version that supports V1 format

#### Option 2: Prevent Migration (Before It Happens)

If you want to skip the storage migration:

1. **Before starting the updated server**, manually set schema version to 3:
   ```bash
   sqlite3 /path/to/data/flightlesssomething.db
   UPDATE schema_versions SET version = 3;
   .quit
   ```

2. Start the server - migration will be skipped since version is already 3

3. Your V1 format files will continue to work (backward compatible)

**Note:** V1 files have higher memory usage but are still supported via fallback reader.

#### Option 3: Keep Both Versions

You can keep both V1 backups and V2 files:

1. **Don't delete backup files** - they remain harmless on disk
2. V2 files are used by default
3. Manually restore specific files if needed (see Option 1)

Backup files only consume disk space and don't affect server operation.

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

- Migration system is designed to be future-proof with clear format versioning
- Old `database.db` file is automatically removed after successful migration
- If file deletion fails, migration still succeeds (with warning logged)
- Schema version prevents accidental re-migration
- Migration is idempotent - checks for existing records before insertion
- Original IDs preserved for users and benchmarks
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

**Storage format migration:** Temporarily requires 2x disk space per benchmark file during conversion (original + V2 version), but backups are deleted after successful migration.

### Storage Format Migration Issues

**Problem:** Migration fails with "Failed to save V2 format"

**Solution:** 
1. Check disk space (need ~2x current benchmark data size temporarily)
2. Check file permissions (write access to benchmarks directory)
3. Check logs for specific error details
4. Backup will be automatically restored on failure

**Problem:** Want to revert to V1 format after migration

**Solution:** See "Restoring from Backup" section above. Backups are kept if deletion fails, or you can manually restore before successful migration completes.

**Problem:** Backup files remain after successful migration

**Cause:** Backup deletion failed (permissions or disk full)

**Solution:** These files are harmless and can be safely deleted:
```bash
cd /path/to/data/benchmarks
rm *.v1.bak
```

**Problem:** High memory usage after migration

**Cause:** Some V1 files might not have been migrated (check logs)

**Solution:** 
1. Check migration logs for which files were skipped
2. Manually verify format of remaining files
3. V1 files still work but use more memory (fallback reader)

### Re-running Migration

If migration fails partway through, you can safely restart the server. The migration logic:
- Checks if each user/benchmark already exists
- Skips already-migrated items
- Completes only the remaining items
- Skips V2 files (checks format before migrating)

## Need Help?

If issues occur:
1. Check error messages in server logs
2. Verify old data structure matches expected format
3. Ensure proper file permissions
4. Check available disk space
5. Open a GitHub issue with error details and relevant log output
