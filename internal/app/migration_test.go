package app

import (
	"os"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestSchemaVersionDetection tests detection of schema versions
func TestSchemaVersionDetection(t *testing.T) {
	t.Run("detects new database", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}

		version, err := detectSchemaVersion(db)
		if err != nil {
			t.Fatalf("Failed to detect schema version: %v", err)
		}

		if version != currentSchemaVersion {
			t.Errorf("Expected version %d for new database, got %d", currentSchemaVersion, version)
		}
	})

	t.Run("detects old database", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}

		// Create old schema
		if migrateErr := db.AutoMigrate(&OldUser{}, &OldBenchmark{}); migrateErr != nil {
			t.Fatalf("Failed to create old schema: %v", migrateErr)
		}

		version, err := detectSchemaVersion(db)
		if err != nil {
			t.Fatalf("Failed to detect schema version: %v", err)
		}

		if version != 0 {
			t.Errorf("Expected version 0 for old database, got %d", version)
		}
	})

	t.Run("detects current database with version", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}

		// Create current schema with version
		if migrateErr := db.AutoMigrate(&User{}, &Benchmark{}, &SchemaVersion{}); migrateErr != nil {
			t.Fatalf("Failed to create current schema: %v", migrateErr)
		}
		if versionErr := setSchemaVersion(db, 1); versionErr != nil {
			t.Fatalf("Failed to set schema version: %v", versionErr)
		}

		version, err := detectSchemaVersion(db)
		if err != nil {
			t.Fatalf("Failed to detect schema version: %v", err)
		}

		if version != 1 {
			t.Errorf("Expected version 1, got %d", version)
		}
	})
}

// TestMigrationFromOldSchema tests the migration process
func TestMigrationFromOldSchema(t *testing.T) {
	t.Run("migrates old schema successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "flightlesssomething.db")
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}

		// Create old schema
		if err := db.AutoMigrate(&OldUser{}, &OldBenchmark{}); err != nil {
			t.Fatalf("Failed to create old schema: %v", err)
		}

		// Create benchmarks directory
		benchmarksDir := filepath.Join(tmpDir, "benchmarks")
		if err := os.MkdirAll(benchmarksDir, 0o755); err != nil {
			t.Fatalf("Failed to create benchmarks directory: %v", err)
		}

		// Add test users
		oldUsers := []OldUser{
			{DiscordID: "123456", Username: "testuser1"},
			{DiscordID: "789012", Username: "testuser2"},
		}
		for i := range oldUsers {
			if err := db.Create(&oldUsers[i]).Error; err != nil {
				t.Fatalf("Failed to create old user: %v", err)
			}
		}

		// Add test benchmarks
		oldBenchmarks := []OldBenchmark{
			{UserID: oldUsers[0].ID, Title: "Test Benchmark 1", Description: "Description 1"},
			{UserID: oldUsers[1].ID, Title: "Test Benchmark 2", Description: "Description 2"},
		}
		for i := range oldBenchmarks {
			if err := db.Create(&oldBenchmarks[i]).Error; err != nil {
				t.Fatalf("Failed to create old benchmark: %v", err)
			}
		}

		// Create benchmark data files
		for _, oldBenchmark := range oldBenchmarks {
			// Skip creating files for this test - we're not testing data file migration in detail
			_ = oldBenchmark
		}

		// Run migration
		if err := migrateFromOldSchema(db, tmpDir); err != nil {
			t.Fatalf("Migration failed: %v", err)
		}

		// Verify users migrated
		var newUsers []User
		if err := db.Find(&newUsers).Error; err != nil {
			t.Fatalf("Failed to query new users: %v", err)
		}
		if len(newUsers) != len(oldUsers) {
			t.Errorf("Expected %d users, got %d", len(oldUsers), len(newUsers))
		}

		// Verify user data
		for i, newUser := range newUsers {
			if newUser.DiscordID != oldUsers[i].DiscordID {
				t.Errorf("User %d: expected DiscordID %s, got %s", i, oldUsers[i].DiscordID, newUser.DiscordID)
			}
			if newUser.Username != oldUsers[i].Username {
				t.Errorf("User %d: expected Username %s, got %s", i, oldUsers[i].Username, newUser.Username)
			}
			if newUser.IsAdmin {
				t.Errorf("User %d: expected IsAdmin false, got true", i)
			}
			if newUser.IsBanned {
				t.Errorf("User %d: expected IsBanned false, got true", i)
			}
		}

		// Verify benchmarks migrated
		var newBenchmarks []Benchmark
		if err := db.Find(&newBenchmarks).Error; err != nil {
			t.Fatalf("Failed to query new benchmarks: %v", err)
		}
		if len(newBenchmarks) != len(oldBenchmarks) {
			t.Errorf("Expected %d benchmarks, got %d", len(oldBenchmarks), len(newBenchmarks))
		}

		// Verify benchmark data
		for i, newBenchmark := range newBenchmarks {
			if newBenchmark.Title != oldBenchmarks[i].Title {
				t.Errorf("Benchmark %d: expected Title %s, got %s", i, oldBenchmarks[i].Title, newBenchmark.Title)
			}
			if newBenchmark.Description != oldBenchmarks[i].Description {
				t.Errorf("Benchmark %d: expected Description %s, got %s", i, oldBenchmarks[i].Description, newBenchmark.Description)
			}
		}
	})

	t.Run("handles re-run gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "flightlesssomething.db")
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}

		// Create old schema
		if err := db.AutoMigrate(&OldUser{}, &OldBenchmark{}); err != nil {
			t.Fatalf("Failed to create old schema: %v", err)
		}

		// Create benchmarks directory
		benchmarksDir := filepath.Join(tmpDir, "benchmarks")
		if err := os.MkdirAll(benchmarksDir, 0o755); err != nil {
			t.Fatalf("Failed to create benchmarks directory: %v", err)
		}

		// Add test user
		oldUser := OldUser{DiscordID: "123456", Username: "testuser"}
		if err := db.Create(&oldUser).Error; err != nil {
			t.Fatalf("Failed to create old user: %v", err)
		}

		// Run migration first time
		if err := migrateFromOldSchema(db, tmpDir); err != nil {
			t.Fatalf("First migration failed: %v", err)
		}

		// Run migration second time (should be idempotent)
		if err := migrateFromOldSchema(db, tmpDir); err != nil {
			t.Fatalf("Second migration failed: %v", err)
		}

		// Verify still only one user
		var count int64
		db.Model(&User{}).Count(&count)
		if count != 1 {
			t.Errorf("Expected 1 user after re-run, got %d", count)
		}
	})
}

// TestInitDBWithMigration tests that InitDB handles migration correctly
func TestInitDBWithMigration(t *testing.T) {
	t.Run("new database initializes with schema version", func(t *testing.T) {
		tmpDir := t.TempDir()
		db, err := InitDB(tmpDir)
		if err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}
		defer cleanupTestDB(t, db)

		// Check schema version is set
		var version SchemaVersion
		if err := db.DB.First(&version).Error; err != nil {
			t.Fatalf("Failed to read schema version: %v", err)
		}

		if version.Version != currentSchemaVersion {
			t.Errorf("Expected schema version %d, got %d", currentSchemaVersion, version.Version)
		}
	})

	t.Run("old database migrates on init", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "flightlesssomething.db")

		// Create old schema manually
		oldDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		if migrateErr := oldDB.AutoMigrate(&OldUser{}, &OldBenchmark{}); migrateErr != nil {
			t.Fatalf("Failed to create old schema: %v", migrateErr)
		}

		// Create benchmarks directory
		benchmarksDir := filepath.Join(tmpDir, "benchmarks")
		if mkdirErr := os.MkdirAll(benchmarksDir, 0o755); mkdirErr != nil {
			t.Fatalf("Failed to create benchmarks directory: %v", mkdirErr)
		}

		// Add test user
		oldUser := OldUser{DiscordID: "123456", Username: "testuser"}
		if createErr := oldDB.Create(&oldUser).Error; createErr != nil {
			t.Fatalf("Failed to create old user: %v", createErr)
		}

		// Close old DB
		sqlDB, dbErr := oldDB.DB()
		if dbErr != nil {
			t.Fatalf("Failed to get sql.DB: %v", dbErr)
		}
		if sqlDB != nil {
			if closeErr := sqlDB.Close(); closeErr != nil {
				t.Fatalf("Failed to close database: %v", closeErr)
			}
		}

		// Now initialize with InitDB (should trigger migration)
		db, err := InitDB(tmpDir)
		if err != nil {
			t.Fatalf("Failed to initialize database with migration: %v", err)
		}
		defer cleanupTestDB(t, db)

		// Verify user was migrated
		var users []User
		if err := db.DB.Find(&users).Error; err != nil {
			t.Fatalf("Failed to query users: %v", err)
		}
		if len(users) != 1 {
			t.Errorf("Expected 1 user, got %d", len(users))
		}
		if len(users) > 0 && users[0].DiscordID != "123456" {
			t.Errorf("Expected DiscordID 123456, got %s", users[0].DiscordID)
		}

		// Verify schema version is set
		var version SchemaVersion
		if err := db.DB.First(&version).Error; err != nil {
			t.Fatalf("Failed to read schema version: %v", err)
		}
		if version.Version != currentSchemaVersion {
			t.Errorf("Expected schema version %d, got %d", currentSchemaVersion, version.Version)
		}
	})

	t.Run("migrates from database.db file", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create old database.db manually
		oldDBPath := filepath.Join(tmpDir, "database.db")
		oldDB, err := gorm.Open(sqlite.Open(oldDBPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("Failed to open old database: %v", err)
		}
		if migrateErr := oldDB.AutoMigrate(&OldUser{}, &OldBenchmark{}); migrateErr != nil {
			t.Fatalf("Failed to create old schema: %v", migrateErr)
		}

		// Create benchmarks directory
		benchmarksDir := filepath.Join(tmpDir, "benchmarks")
		if mkdirErr := os.MkdirAll(benchmarksDir, 0o755); mkdirErr != nil {
			t.Fatalf("Failed to create benchmarks directory: %v", mkdirErr)
		}

		// Add test user to old database
		oldUser := OldUser{DiscordID: "999888", Username: "olddbuser"}
		if createErr := oldDB.Create(&oldUser).Error; createErr != nil {
			t.Fatalf("Failed to create old user: %v", createErr)
		}

		// Close old DB
		sqlDB, dbErr := oldDB.DB()
		if dbErr != nil {
			t.Fatalf("Failed to get sql.DB: %v", dbErr)
		}
		if sqlDB != nil {
			if closeErr := sqlDB.Close(); closeErr != nil {
				t.Fatalf("Failed to close database: %v", closeErr)
			}
		}

		// Now initialize with InitDB - should detect and migrate from database.db
		db, err := InitDB(tmpDir)
		if err != nil {
			t.Fatalf("Failed to initialize database with old file migration: %v", err)
		}
		defer cleanupTestDB(t, db)

		// Verify flightlesssomething.db was created
		newDBPath := filepath.Join(tmpDir, "flightlesssomething.db")
		if _, err := os.Stat(newDBPath); os.IsNotExist(err) {
			t.Fatalf("New database file not created")
		}

		// Verify user was migrated
		var users []User
		if err := db.DB.Find(&users).Error; err != nil {
			t.Fatalf("Failed to query users: %v", err)
		}
		// Should have 1 user: the migrated one (system admin is created by EnsureSystemAdmin, not InitDB)
		if len(users) != 1 {
			t.Fatalf("Expected 1 user (migrated), got %d", len(users))
		}
		
		// Verify it's the migrated user
		if users[0].DiscordID != "999888" {
			t.Errorf("Expected DiscordID 999888, got %s", users[0].DiscordID)
		}
		if users[0].Username != "olddbuser" {
			t.Errorf("Expected Username olddbuser, got %s", users[0].Username)
		}

		// Verify schema version is set
		var version SchemaVersion
		if err := db.DB.First(&version).Error; err != nil {
			t.Fatalf("Failed to read schema version: %v", err)
		}
		if version.Version != currentSchemaVersion {
			t.Errorf("Expected schema version %d, got %d", currentSchemaVersion, version.Version)
		}
	})
}
