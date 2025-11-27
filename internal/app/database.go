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

// InitDB initializes the database connection
func InitDB(dataDir string) (*DBInstance, error) {
	dbPath := filepath.Join(dataDir, "flightlesssomething.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&User{}, &Benchmark{}, &AuditLog{}, &APIToken{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
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
