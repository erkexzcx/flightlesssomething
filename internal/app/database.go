package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DBInstance wraps the database connection
type DBInstance struct {
	DB *gorm.DB
}

// InitDB initializes the database connection and handles schema migrations
//
// Migration strategy (future-proof):
// 1. Format 1 (v0.20 and earlier): database.db file with old schema
//    - Detected by: database.db exists, flightlesssomething.db doesn't exist
//    - Action: Migrate from database.db to flightlesssomething.db, delete database.db on success
//
// 2. Format 2 (intermediate): flightlesssomething.db with old schema (no schema_versions table)
//    - Detected by: flightlesssomething.db exists, no schema_versions table, has ai_summary column
//    - Action: In-place schema upgrade, add schema_versions table with version 1
//
// 3. Format 3+ (current and future): flightlesssomething.db with schema_versions table
//    - Detected by: schema_versions table exists with version number
//    - Action: Check version number and apply incremental migrations as needed
//    - Current version: 1
//    - Future versions: Add migration logic for versions 2, 3, etc. in switch/case
func InitDB(dataDir string) (*DBInstance, error) {
	dbPath := filepath.Join(dataDir, "flightlesssomething.db")
	
	// Check if we need to migrate from old database.db file (Format 1)
	oldDBPath := filepath.Join(dataDir, "database.db")
	needsOldFileMigration := false
	if _, err := os.Stat(oldDBPath); err == nil {
		// Old database.db exists, check if new database doesn't exist or is empty
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			needsOldFileMigration = true
			log.Printf("Found old database.db, will migrate to flightlesssomething.db")
		}
	}
	
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Detect schema version (Format 2 vs Format 3+)
	version, err := detectSchemaVersion(db)
	if err != nil {
		return nil, fmt.Errorf("failed to detect schema version: %w", err)
	}

	// Handle old database.db file migration if needed (Format 1 → Format 3)
	//nolint:gocritic // if-else chain is more readable here than switch for different migration paths
	if needsOldFileMigration {
		log.Println("Migrating from database.db to flightlesssomething.db...")
		if err := migrateFromOldDatabaseFile(db, dataDir, oldDBPath); err != nil {
			return nil, fmt.Errorf("failed to migrate from old database file: %w", err)
		}
		// Set current schema version after successful migration
		if err := setSchemaVersion(db, currentSchemaVersion); err != nil {
			return nil, fmt.Errorf("failed to set schema version: %w", err)
		}
		// Delete old database file after successful migration
		if err := os.Remove(oldDBPath); err != nil {
			log.Printf("Warning: failed to remove old database.db file: %v", err)
			log.Printf("You can safely delete %s manually", oldDBPath)
		} else {
			log.Printf("Successfully removed old database.db file")
		}
	} else if version == 0 {
		// Handle in-place schema migration (Format 2 → Format 3)
		// This is for databases that are already flightlesssomething.db but have old schema
		if err := migrateFromOldSchema(db, dataDir); err != nil {
			return nil, fmt.Errorf("failed to migrate from old schema: %w", err)
		}
		// Set current schema version after successful migration
		if err := setSchemaVersion(db, currentSchemaVersion); err != nil {
			return nil, fmt.Errorf("failed to set schema version: %w", err)
		}
	} else if version > currentSchemaVersion {
		// Database is from a newer version of the application - this shouldn't happen
		return nil, fmt.Errorf("database version %d is newer than supported version %d - please upgrade the application", version, currentSchemaVersion)
	}

	// Auto-migrate the schema BEFORE running data migrations
	// This ensures columns exist before migration code tries to use them
	if err := db.AutoMigrate(&User{}, &Benchmark{}, &AuditLog{}, &APIToken{}, &SchemaVersion{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Run data migrations after schema is updated
	if version > 0 && version < currentSchemaVersion {
		// Handle incremental migrations for future versions (Format 3+ → newer Format 3+)
		log.Printf("Database is at version %d, current version is %d. Running data migrations...", version, currentSchemaVersion)
		
		if version == 1 {
			log.Println("Populating new fields for version 2...")
			if err := migrateFromV1ToV2(db); err != nil {
				return nil, fmt.Errorf("failed to migrate from v1 to v2: %w", err)
			}
			// Update version to 2 after successful migration
			if err := setSchemaVersion(db, 2); err != nil {
				return nil, fmt.Errorf("failed to set schema version to 2: %w", err)
			}
			log.Println("Successfully migrated to version 2")
		}
		// Future migrations would go here as additional else-if blocks:
		// else if version == 2 {
		//     if err := migrateFromV2ToV3(db); err != nil {
		//         return nil, fmt.Errorf("failed to migrate from v2 to v3: %w", err)
		//     }
		//     if err := setSchemaVersion(db, 3); err != nil {
		//         return nil, fmt.Errorf("failed to set schema version to 3: %w", err)
		//     }
		// }
	}

	// Ensure schema version is set for new databases
	if !needsOldFileMigration && version == currentSchemaVersion {
		// For brand new databases, set the version
		var count int64
		db.Model(&SchemaVersion{}).Count(&count)
		if count == 0 {
			if err := setSchemaVersion(db, currentSchemaVersion); err != nil {
				return nil, fmt.Errorf("failed to set initial schema version: %w", err)
			}
		}
	}

	return &DBInstance{DB: db}, nil
}

// EnsureSystemAdmin ensures there is exactly one system admin account
// and updates it with the current credentials from config
func EnsureSystemAdmin(db *DBInstance, username, password string) error {
	// First, check if there's already a system admin with discord_id = "admin"
	var systemAdmin User
	result := db.DB.Where("discord_id = ?", "admin").First(&systemAdmin)

	if result.Error == nil {
		// System admin exists, update username if changed
		updated := false
		if systemAdmin.Username != username {
			systemAdmin.Username = username
			updated = true
		}
		if !systemAdmin.IsAdmin {
			systemAdmin.IsAdmin = true
			updated = true
		}
		if updated {
			if err := db.DB.Save(&systemAdmin).Error; err != nil {
				return fmt.Errorf("failed to update system admin: %w", err)
			}
		}
	} else {
		// System admin doesn't exist, create it
		systemAdmin = User{
			DiscordID: "admin",
			Username:  username,
			IsAdmin:   true,
		}
		if err := db.DB.Create(&systemAdmin).Error; err != nil {
			return fmt.Errorf("failed to create system admin: %w", err)
		}
	}

	return nil
}
