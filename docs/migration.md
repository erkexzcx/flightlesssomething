# Migration Guide

Guide for migrating data from the old [flightlesssomething](https://github.com/erkexzcx/flightlesssomething) project.

## What Gets Migrated

- **Users**: All user accounts with Discord IDs and usernames
- **Benchmarks**: Metadata (title, description, timestamps)
- **Benchmark Data**: Binary files with performance metrics
- **Metadata Files**: New `.meta` files for optimization

## Schema Changes

### Old Project
- Database: `database.db`
- User fields: DiscordID, Username
- Benchmark fields: UserID, Title, Description, AiSummary

### New Project
- Database: `flightlesssomething.db`
- User fields: DiscordID, Username, IsAdmin, IsBanned, activity timestamps
- Benchmark fields: UserID, Title, Description (no AiSummary)
- New tables: APIToken, AuditLog

### Migration Notes
- Users migrated with `IsAdmin=false`, `IsBanned=false`
- Description limit increased: 500 → 5000 chars
- AiSummary field discarded (not used)
- Data files copied and validated
- New metadata files generated

## Prerequisites

1. Stop both old and new servers
2. Access to old project's data directory
3. Go 1.21+ installed

## Migration Steps

### 1. Build Migration Tool

```bash
cd flightlesssomething
go build -o migrate ./cmd/migrate
```

### 2. Dry Run (Preview)

Preview without making changes:

```bash
./migrate \
  -old-data-dir=/path/to/old/data \
  -new-data-dir=/path/to/new/data \
  -dry-run
```

Shows:
- Number of users to migrate
- Number of benchmarks to migrate
- Validation status

### 3. Run Migration

Perform actual migration:

```bash
./migrate \
  -old-data-dir=/path/to/old/data \
  -new-data-dir=/path/to/new/data
```

The tool will:
1. Create new data directory
2. Initialize new database
3. Migrate all users
4. Migrate all benchmarks with data
5. Generate metadata files
6. Display summary

### 4. Verify Migration

Check results:

```bash
# Check files
ls -la /path/to/new/data/
ls -la /path/to/new/data/benchmarks/

# Count migrated data
sqlite3 /path/to/new/data/flightlesssomething.db \
  "SELECT COUNT(*) FROM users;"
sqlite3 /path/to/new/data/flightlesssomething.db \
  "SELECT COUNT(*) FROM benchmarks;"
```

### 5. Start New Server

```bash
./server \
  -bind="0.0.0.0:5000" \
  -data-dir="/path/to/new/data" \
  -session-secret="your-secret" \
  -discord-client-id="your-id" \
  -discord-client-secret="your-secret" \
  -discord-redirect-url="http://localhost:5000/auth/login/callback" \
  -admin-username="admin" \
  -admin-password="admin"
```

## Example Output

```
2024/11/26 11:00:00 Starting migration from /old/data to /new/data
2024/11/26 11:00:00 Opening old database...
2024/11/26 11:00:00 Opening new database...
2024/11/26 11:00:00 Running database migrations...
2024/11/26 11:00:00 Migrating users...
2024/11/26 11:00:00 Found 42 users to migrate
  Migrating user: JohnDoe (ID: 1, Discord: 123456789)
    Created with new ID: 1
  ...
2024/11/26 11:00:01 Migrating benchmarks...
2024/11/26 11:00:01 Found 987 benchmarks to migrate
  [1/987] Migrating benchmark: Cyberpunk 2077 (ID: 1)
    Successfully migrated (new ID: 1, 3 runs)
  ...

=== Migration Summary ===
Users migrated: 42
Benchmarks attempted: 987
Benchmarks succeeded: 987
Benchmarks failed: 0
=========================

Migration completed successfully!
```

## Post-Migration

### Set Admin Users

After migration, set admin status manually:

```bash
sqlite3 /path/to/new/data/flightlesssomething.db
UPDATE users SET is_admin=1 WHERE discord_id='YOUR_DISCORD_ID';
.quit
```

Or use admin panel in web UI.

### System Admin Account

The system creates an admin account with `discord_id='admin'` on first startup. This is separate from Discord users.

## Safety

- Migration tool **never modifies** old data directory
- All changes made only to new data directory
- Safe to run multiple times (creates duplicates)
- Always use `-dry-run` first

## Troubleshooting

### "Old database not found"
- Verify old data directory path
- Check `database.db` exists

### "Failed to read data file"
- Some data files may be corrupted
- Tool skips corrupted files and continues
- Check error messages for specific IDs

### "User ID not found"
- Benchmark references non-existent user
- Rare but can happen with data inconsistencies
- Benchmark will be skipped

### Disk Space
- Need roughly double space (original + copy)
- Check available disk space first

## Need Help?

If issues occur:
1. Check error messages in output
2. Verify old data structure matches expected format
3. Ensure proper file permissions
4. Open GitHub issue with error details
