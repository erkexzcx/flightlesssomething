package app

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	// currentSchemaVersion is the current schema version
	// Version history:
	// - 0: Old schema (Format 1 and Format 2) - no schema_versions table, has ai_summary column
	// - 1: Current schema (Format 3) - has schema_versions table, removed ai_summary column
	// - 2: Added RunNames and Specifications fields to Benchmark for enhanced search
	// Future versions should increment this and add migration logic in InitDB
	currentSchemaVersion = 2
	// Maximum description length in new schema
	maxDescriptionLength = 5000
)

// SchemaVersion stores the current schema version in the database
type SchemaVersion struct {
	gorm.Model
	Version int `gorm:"uniqueIndex"`
}

// OldUser represents a user from the old flightlesssomething project
// TableName returns "users" to match the old database table structure
type OldUser struct {
	gorm.Model
	DiscordID string `gorm:"size:20"`
	Username  string `gorm:"size:32"`
}

func (OldUser) TableName() string {
	return "users" // Intentionally matches current User table for in-place migration
}

// OldBenchmark represents a benchmark from the old project
// TableName returns "benchmarks" to match the old database table structure
type OldBenchmark struct {
	gorm.Model
	UserID      uint
	Title       string `gorm:"size:100"`
	Description string `gorm:"size:500"`
	AiSummary   string
}

func (OldBenchmark) TableName() string {
	return "benchmarks" // Intentionally matches current Benchmark table for in-place migration
}

// detectSchemaVersion detects the schema version of the database
// Returns:
//   - 0 if this is an old database (Format 2: no schema_versions table, old structure)
//   - currentSchemaVersion if this is a current database (Format 3+: has schema_versions table)
//   - error if unable to determine
//
// Detection logic:
// 1. If schema_versions table exists → read version from it (Format 3+)
// 2. If users/benchmarks tables exist with ai_summary column → version 0 (Format 2)
// 3. If no tables exist → new database, return currentSchemaVersion
//
// Note: Format 1 (database.db) is detected at file level before calling this function
func detectSchemaVersion(db *gorm.DB) (int, error) {
	// Check if schema_versions table exists
	if db.Migrator().HasTable(&SchemaVersion{}) {
		var version SchemaVersion
		if err := db.Order("version DESC").First(&version).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Table exists but empty - this shouldn't happen, treat as current
				return currentSchemaVersion, nil
			}
			return 0, fmt.Errorf("failed to read schema version: %w", err)
		}
		return version.Version, nil
	}

	// Check if this is an old database by looking for the old schema structure
	// Old databases have users and benchmarks tables but no schema_versions
	if db.Migrator().HasTable("users") && db.Migrator().HasTable("benchmarks") {
		// Check if it has old schema characteristics (e.g., ai_summary column)
		if db.Migrator().HasColumn(&OldBenchmark{}, "ai_summary") {
			log.Println("Detected old database schema (version 0)")
			return 0, nil
		}
	}

	// This is a new database - no tables yet
	return currentSchemaVersion, nil
}

// setSchemaVersion sets the schema version in the database
func setSchemaVersion(db *gorm.DB, version int) error {
	// Ensure the table exists
	if err := db.AutoMigrate(&SchemaVersion{}); err != nil {
		return fmt.Errorf("failed to migrate schema_versions table: %w", err)
	}

	// Insert or update the version
	schemaVersion := SchemaVersion{Version: version}
	result := db.Where("version = ?", version).FirstOrCreate(&schemaVersion)
	if result.Error != nil {
		return fmt.Errorf("failed to set schema version: %w", result.Error)
	}

	return nil
}

// migrateFromOldDatabaseFile migrates data from the old database.db file to the new flightlesssomething.db
func migrateFromOldDatabaseFile(newDB *gorm.DB, dataDir, oldDBPath string) error {
	log.Printf("Starting migration from %s...", oldDBPath)
	
	// Open the old database
	oldDB, err := gorm.Open(sqlite.Open(oldDBPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open old database: %w", err)
	}
	
	// Close old database connection when done
	sqlDB, err := oldDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}
	defer func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			log.Printf("Warning: failed to close old database: %v", closeErr)
		}
	}()
	
	// Migrate users
	log.Println("Migrating users from old database...")
	var oldUsers []OldUser
	if err := oldDB.Find(&oldUsers).Error; err != nil {
		return fmt.Errorf("failed to fetch old users: %w", err)
	}
	log.Printf("Found %d users to migrate", len(oldUsers))

	// Create new users table with proper schema
	if err := newDB.AutoMigrate(&User{}); err != nil {
		return fmt.Errorf("failed to migrate users table: %w", err)
	}

	// Migrate each user
	for _, oldUser := range oldUsers {
		log.Printf("  Migrating user: %s (ID: %d, Discord: %s)", oldUser.Username, oldUser.ID, oldUser.DiscordID)

		newUser := User{
			Model: gorm.Model{
				ID:        oldUser.ID, // Preserve original ID to maintain benchmark relationships
				CreatedAt: oldUser.CreatedAt,
				UpdatedAt: oldUser.UpdatedAt,
			},
			DiscordID: oldUser.DiscordID,
			Username:  oldUser.Username,
			IsAdmin:   false,
			IsBanned:  false,
		}
		if err := newDB.Create(&newUser).Error; err != nil {
			return fmt.Errorf("failed to create user %s: %w", oldUser.Username, err)
		}
		log.Printf("    Migrated successfully")
	}

	// Migrate benchmarks
	log.Println("Migrating benchmarks from old database...")
	var oldBenchmarks []OldBenchmark
	if err := oldDB.Find(&oldBenchmarks).Error; err != nil {
		return fmt.Errorf("failed to fetch old benchmarks: %w", err)
	}
	log.Printf("Found %d benchmarks to migrate", len(oldBenchmarks))

	// Create new benchmarks table with proper schema
	if err := newDB.AutoMigrate(&Benchmark{}); err != nil {
		return fmt.Errorf("failed to migrate benchmarks table: %w", err)
	}

	successCount := 0
	errorCount := 0
	benchmarksDir := filepath.Join(dataDir, "benchmarks")

	for i := range oldBenchmarks {
		oldBenchmark := &oldBenchmarks[i]
		log.Printf("  [%d/%d] Migrating benchmark: %s (ID: %d)", i+1, len(oldBenchmarks), oldBenchmark.Title, oldBenchmark.ID)

		// Truncate description if too long
		description := oldBenchmark.Description
		if len(description) > maxDescriptionLength {
			description = description[:maxDescriptionLength]
		}

		// Verify the user exists
		var userExists bool
		if err := newDB.Model(&User{}).Select("count(*) > 0").Where("id = ?", oldBenchmark.UserID).Find(&userExists).Error; err != nil {
			log.Printf("    ERROR: Database error checking user ID %d: %v", oldBenchmark.UserID, err)
			errorCount++
			continue
		}
		if !userExists {
			log.Printf("    WARNING: User ID %d not found, skipping", oldBenchmark.UserID)
			errorCount++
			continue
		}

		newBenchmark := Benchmark{
			Model: gorm.Model{
				ID:        oldBenchmark.ID, // Preserve original ID to maintain data file associations
				CreatedAt: oldBenchmark.CreatedAt,
				UpdatedAt: oldBenchmark.UpdatedAt,
			},
			UserID:      oldBenchmark.UserID, // Use original user ID (preserved from migration)
			Title:       oldBenchmark.Title,
			Description: description,
		}
		if err := newDB.Create(&newBenchmark).Error; err != nil {
			log.Printf("    ERROR: Failed to create benchmark: %v", err)
			errorCount++
			continue
		}

		// Verify benchmark data file exists
		dataFile := filepath.Join(benchmarksDir, fmt.Sprintf("%d.bin", oldBenchmark.ID))
		if _, err := os.Stat(dataFile); os.IsNotExist(err) {
			log.Printf("    WARNING: Data file not found: %s", dataFile)
			errorCount++
			continue
		}

		// Read and validate the data
		benchmarkData, err := readBenchmarkDataForMigration(dataFile)
		if err != nil {
			log.Printf("    ERROR: Failed to read data file: %v", err)
			errorCount++
			continue
		}

		// Create metadata file if it doesn't exist
		metaPath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.meta", oldBenchmark.ID))
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			if err := createMetadataFileForMigration(dataDir, oldBenchmark.ID, benchmarkData); err != nil {
				log.Printf("    ERROR: Failed to create metadata file: %v", err)
				errorCount++
				continue
			}
		}

		// Extract and populate searchable metadata (v2 schema)
		runNames, specifications := ExtractSearchableMetadata(benchmarkData)
		// Use UpdateColumns to avoid updating the UpdatedAt timestamp during migration
		if err := newDB.Model(&newBenchmark).UpdateColumns(map[string]interface{}{
			"run_names":      runNames,
			"specifications": specifications,
		}).Error; err != nil {
			log.Printf("    WARNING: Failed to update searchable metadata: %v", err)
		}

		log.Printf("    Successfully migrated (%d runs)", len(benchmarkData))
		successCount++
	}

	log.Println("\n=== Migration Summary ===")
	log.Printf("Users migrated: %d", len(oldUsers))
	log.Printf("Benchmarks attempted: %d", len(oldBenchmarks))
	log.Printf("Benchmarks succeeded: %d", successCount)
	log.Printf("Benchmarks failed: %d", errorCount)
	log.Println("=========================")

	if errorCount > 0 {
		log.Printf("WARNING: %d benchmarks failed to migrate, but migration will continue", errorCount)
	}

	log.Println("Migration from old database file completed successfully!")
	return nil
}

// migrateFromOldSchema migrates the database from old schema (version 0) to current
func migrateFromOldSchema(db *gorm.DB, dataDir string) error {
	log.Println("Starting migration from old schema to current schema...")

	// Get old benchmarks directory path
	oldBenchmarksDir := filepath.Join(dataDir, "benchmarks")

	// Migrate users
	log.Println("Migrating users from old schema...")
	var oldUsers []OldUser
	if err := db.Find(&oldUsers).Error; err != nil {
		return fmt.Errorf("failed to fetch old users: %w", err)
	}
	log.Printf("Found %d users to migrate", len(oldUsers))

	// Create new users table with proper schema
	if err := db.AutoMigrate(&User{}); err != nil {
		return fmt.Errorf("failed to migrate users table: %w", err)
	}

	// Migrate each user
	for _, oldUser := range oldUsers {
		log.Printf("  Migrating user: %s (ID: %d, Discord: %s)", oldUser.Username, oldUser.ID, oldUser.DiscordID)

		// Check if user already exists (in case of re-run)
		var existingUser User
		result := db.Where("id = ?", oldUser.ID).First(&existingUser)
		if result.Error == nil {
			log.Printf("    User already exists, skipping")
			continue
		}

		newUser := User{
			Model: gorm.Model{
				ID:        oldUser.ID, // Preserve original ID to maintain benchmark relationships
				CreatedAt: oldUser.CreatedAt,
				UpdatedAt: oldUser.UpdatedAt,
			},
			DiscordID: oldUser.DiscordID,
			Username:  oldUser.Username,
			IsAdmin:   false,
			IsBanned:  false,
		}
		if err := db.Create(&newUser).Error; err != nil {
			return fmt.Errorf("failed to create user %s: %w", oldUser.Username, err)
		}
		log.Printf("    Migrated successfully")
	}

	// Migrate benchmarks
	log.Println("Migrating benchmarks from old schema...")
	var oldBenchmarks []OldBenchmark
	if err := db.Find(&oldBenchmarks).Error; err != nil {
		return fmt.Errorf("failed to fetch old benchmarks: %w", err)
	}
	log.Printf("Found %d benchmarks to migrate", len(oldBenchmarks))

	// Create new benchmarks table with proper schema
	if err := db.AutoMigrate(&Benchmark{}); err != nil {
		return fmt.Errorf("failed to migrate benchmarks table: %w", err)
	}

	successCount := 0
	errorCount := 0

	for i := range oldBenchmarks {
		oldBenchmark := &oldBenchmarks[i]
		log.Printf("  [%d/%d] Migrating benchmark: %s (ID: %d)", i+1, len(oldBenchmarks), oldBenchmark.Title, oldBenchmark.ID)

		// Check if benchmark already exists (in case of re-run)
		var existingBenchmark Benchmark
		result := db.Where("id = ?", oldBenchmark.ID).First(&existingBenchmark)
		if result.Error == nil {
			log.Printf("    Benchmark already exists, skipping")
			successCount++
			continue
		}

		// Truncate description if too long
		description := oldBenchmark.Description
		if len(description) > maxDescriptionLength {
			description = description[:maxDescriptionLength]
		}

		// Verify the user exists
		var userExists bool
		if err := db.Model(&User{}).Select("count(*) > 0").Where("id = ?", oldBenchmark.UserID).Find(&userExists).Error; err != nil {
			log.Printf("    ERROR: Database error checking user ID %d: %v", oldBenchmark.UserID, err)
			errorCount++
			continue
		}
		if !userExists {
			log.Printf("    WARNING: User ID %d not found, skipping", oldBenchmark.UserID)
			errorCount++
			continue
		}

		newBenchmark := Benchmark{
			Model: gorm.Model{
				ID:        oldBenchmark.ID, // Preserve original ID to maintain data file associations
				CreatedAt: oldBenchmark.CreatedAt,
				UpdatedAt: oldBenchmark.UpdatedAt,
			},
			UserID:      oldBenchmark.UserID, // Use original user ID (preserved from migration)
			Title:       oldBenchmark.Title,
			Description: description,
		}
		if err := db.Create(&newBenchmark).Error; err != nil {
			log.Printf("    ERROR: Failed to create benchmark: %v", err)
			errorCount++
			continue
		}

		// Verify benchmark data file exists
		dataFile := filepath.Join(oldBenchmarksDir, fmt.Sprintf("%d.bin", oldBenchmark.ID))
		if _, err := os.Stat(dataFile); os.IsNotExist(err) {
			log.Printf("    WARNING: Data file not found: %s", dataFile)
			errorCount++
			continue
		}

		// Read and validate the data
		benchmarkData, err := readBenchmarkDataForMigration(dataFile)
		if err != nil {
			log.Printf("    ERROR: Failed to read data file: %v", err)
			errorCount++
			continue
		}

		// Create metadata file if it doesn't exist
		metaPath := filepath.Join(oldBenchmarksDir, fmt.Sprintf("%d.meta", oldBenchmark.ID))
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			if err := createMetadataFileForMigration(dataDir, oldBenchmark.ID, benchmarkData); err != nil {
				log.Printf("    ERROR: Failed to create metadata file: %v", err)
				errorCount++
				continue
			}
		}

		// Extract and populate searchable metadata (v2 schema)
		runNames, specifications := ExtractSearchableMetadata(benchmarkData)
		// Use UpdateColumns to avoid updating the UpdatedAt timestamp during migration
		if err := db.Model(&newBenchmark).UpdateColumns(map[string]interface{}{
			"run_names":      runNames,
			"specifications": specifications,
		}).Error; err != nil {
			log.Printf("    WARNING: Failed to update searchable metadata: %v", err)
		}

		log.Printf("    Successfully migrated (%d runs)", len(benchmarkData))
		successCount++
	}

	log.Println("\n=== Migration Summary ===")
	log.Printf("Users migrated: %d", len(oldUsers))
	log.Printf("Benchmarks attempted: %d", len(oldBenchmarks))
	log.Printf("Benchmarks succeeded: %d", successCount)
	log.Printf("Benchmarks failed: %d", errorCount)
	log.Println("=========================")

	if errorCount > 0 {
		log.Printf("WARNING: %d benchmarks failed to migrate, but migration will continue", errorCount)
	}

	log.Println("Migration from old schema completed successfully!")
	return nil
}

// readBenchmarkDataForMigration reads benchmark data from a file
func readBenchmarkDataForMigration(filePath string) ([]*BenchmarkData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Printf("Warning: failed to close file %s: %v", filePath, cerr)
		}
	}()

	zstdDecoder, err := zstd.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer zstdDecoder.Close()

	var buffer bytes.Buffer
	_, err = buffer.ReadFrom(zstdDecoder)
	if err != nil {
		return nil, err
	}

	var benchmarkData []*BenchmarkData
	gobDecoder := gob.NewDecoder(&buffer)
	err = gobDecoder.Decode(&benchmarkData)
	return benchmarkData, err
}

// createMetadataFileForMigration creates a metadata file for a benchmark
func createMetadataFileForMigration(dataDir string, benchmarkID uint, benchmarkData []*BenchmarkData) error {
	labels := make([]string, len(benchmarkData))
	for i, data := range benchmarkData {
		labels[i] = data.Label
	}

	metadata := BenchmarkMetadata{
		RunCount:  len(benchmarkData),
		RunLabels: labels,
	}

	metaPath := filepath.Join(dataDir, "benchmarks", fmt.Sprintf("%d.meta", benchmarkID))
	metaFile, err := os.Create(metaPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := metaFile.Close(); cerr != nil {
			log.Printf("Warning: failed to close metadata file %s: %v", metaPath, cerr)
		}
	}()

	gobEncoder := gob.NewEncoder(metaFile)
	return gobEncoder.Encode(metadata)
}

// migrateFromV1ToV2 migrates from schema version 1 to version 2
// This migration populates the RunNames and Specifications fields for all existing benchmarks
func migrateFromV1ToV2(db *gorm.DB) error {
	log.Println("Populating RunNames and Specifications for existing benchmarks...")
	
	// Get all benchmarks
	var benchmarks []Benchmark
	if err := db.Find(&benchmarks).Error; err != nil {
		return fmt.Errorf("failed to fetch benchmarks: %w", err)
	}
	log.Printf("Found %d benchmarks to update", len(benchmarks))
	
	successCount := 0
	errorCount := 0
	
	for i := range benchmarks {
		benchmark := &benchmarks[i]
		log.Printf("  [%d/%d] Updating benchmark: %s (ID: %d)", i+1, len(benchmarks), benchmark.Title, benchmark.ID)
		
		// Read benchmark data
		benchmarkData, err := RetrieveBenchmarkData(benchmark.ID)
		if err != nil {
			log.Printf("    WARNING: Failed to read data file: %v", err)
			errorCount++
			continue
		}
		
		// Extract searchable metadata
		runNames, specifications := ExtractSearchableMetadata(benchmarkData)
		
		// Update benchmark record using UpdateColumns to preserve UpdatedAt timestamp
		if err := db.Model(benchmark).UpdateColumns(map[string]interface{}{
			"run_names":      runNames,
			"specifications": specifications,
		}).Error; err != nil {
			log.Printf("    ERROR: Failed to update benchmark: %v", err)
			errorCount++
			continue
		}
		
		log.Printf("    Successfully updated (runs: %d, specs fields: %d chars)", len(benchmarkData), len(specifications))
		successCount++
	}
	
	log.Println("\n=== Migration Summary (v1 → v2) ===")
	log.Printf("Benchmarks updated: %d", successCount)
	log.Printf("Benchmarks failed: %d", errorCount)
	log.Println("=====================================")
	
	if errorCount > 0 {
		log.Printf("WARNING: %d benchmarks failed to update, but migration will continue", errorCount)
	}
	
	log.Println("Migration from v1 to v2 completed successfully!")
	return nil
}

