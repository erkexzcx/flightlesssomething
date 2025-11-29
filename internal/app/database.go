package app

import (
	"fmt"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DBInstance wraps the database connection
type DBInstance struct {
	DB *gorm.DB
}

// InitDB initializes the database connection and handles schema migrations
func InitDB(dataDir string) (*DBInstance, error) {
	dbPath := filepath.Join(dataDir, "flightlesssomething.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Detect schema version
	version, err := detectSchemaVersion(db)
	if err != nil {
		return nil, fmt.Errorf("failed to detect schema version: %w", err)
	}

	// Handle old schema migration if needed
	if version == 0 {
		if err := migrateFromOldSchema(db, dataDir); err != nil {
			return nil, fmt.Errorf("failed to migrate from old schema: %w", err)
		}
		// Set current schema version after successful migration
		if err := setSchemaVersion(db, currentSchemaVersion); err != nil {
			return nil, fmt.Errorf("failed to set schema version: %w", err)
		}
	}

	// Auto-migrate the schema (this is safe for both new and existing databases)
	if err := db.AutoMigrate(&User{}, &Benchmark{}, &AuditLog{}, &APIToken{}, &SchemaVersion{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Ensure schema version is set for new databases
	if version == currentSchemaVersion {
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
