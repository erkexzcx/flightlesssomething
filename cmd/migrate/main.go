package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/klauspost/compress/zstd"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	// Directory permissions for data directories
	dirPerm = 0o755
	// Maximum description length in new schema
	maxDescriptionLength = 5000
)

// OldUser represents a user from the old flightlesssomething project
type OldUser struct {
	gorm.Model
	DiscordID string `gorm:"size:20"`
	Username  string `gorm:"size:32"`
}

func (OldUser) TableName() string {
	return "users"
}

type OldBenchmark struct {
	gorm.Model
	UserID      uint
	Title       string `gorm:"size:100"`
	Description string `gorm:"size:500"`
	AiSummary   string
}

func (OldBenchmark) TableName() string {
	return "benchmarks"
}

// NewUser represents a user in the new flightlesssomething project
type NewUser struct {
	gorm.Model
	DiscordID         string     `gorm:"size:20;uniqueIndex"`
	Username          string     `gorm:"size:32"`
	IsAdmin           bool       `gorm:"default:false"`
	IsBanned          bool       `gorm:"default:false"`
	LastWebActivityAt *time.Time `gorm:"default:null"`
	LastAPIActivityAt *time.Time `gorm:"default:null"`
}

func (NewUser) TableName() string {
	return "users"
}

type NewBenchmark struct {
	gorm.Model
	UserID      uint
	Title       string `gorm:"size:100"`
	Description string `gorm:"size:5000"`
}

func (NewBenchmark) TableName() string {
	return "benchmarks"
}

// BenchmarkData structure (same in both projects)
type BenchmarkData struct {
	Label string

	// System specs
	SpecOS             string
	SpecCPU            string
	SpecGPU            string
	SpecRAM            string
	SpecLinuxKernel    string
	SpecLinuxScheduler string

	// Performance data arrays
	DataFPS          []float64
	DataFrameTime    []float64
	DataCPULoad      []float64
	DataGPULoad      []float64
	DataCPUTemp      []float64
	DataGPUTemp      []float64
	DataGPUCoreClock []float64
	DataGPUMemClock  []float64
	DataGPUVRAMUsed  []float64
	DataGPUPower     []float64
	DataRAMUsed      []float64
	DataSwapUsed     []float64
}

// BenchmarkMetadata for the new project
type BenchmarkMetadata struct {
	RunCount  int
	RunLabels []string
}

func main() {
	// Parse flags
	oldDataDir := flag.String("old-data-dir", "", "Path to old project data directory (required)")
	newDataDir := flag.String("new-data-dir", "", "Path to new project data directory (required)")
	dryRun := flag.Bool("dry-run", false, "Run without making changes (preview mode)")
	flag.Parse()

	if *oldDataDir == "" || *newDataDir == "" {
		flag.Usage()
		log.Fatal("Both -old-data-dir and -new-data-dir are required")
	}

	log.Printf("Starting migration from %s to %s", *oldDataDir, *newDataDir)
	if *dryRun {
		log.Println("DRY RUN MODE - No changes will be made")
	}

	// Validate old data directory
	oldDBPath := filepath.Join(*oldDataDir, "database.db")
	if _, err := os.Stat(oldDBPath); os.IsNotExist(err) {
		log.Fatalf("Old database not found at %s", oldDBPath)
	}
	oldBenchmarksDir := filepath.Join(*oldDataDir, "benchmarks")
	if _, err := os.Stat(oldBenchmarksDir); os.IsNotExist(err) {
		log.Fatalf("Old benchmarks directory not found at %s", oldBenchmarksDir)
	}

	// Create new data directory if it doesn't exist
	if !*dryRun {
		if err := os.MkdirAll(*newDataDir, dirPerm); err != nil {
			log.Fatalf("Failed to create new data directory: %v", err)
		}
		newBenchmarksDir := filepath.Join(*newDataDir, "benchmarks")
		if err := os.MkdirAll(newBenchmarksDir, dirPerm); err != nil {
			log.Fatalf("Failed to create new benchmarks directory: %v", err)
		}
	}

	// Open old database
	log.Println("Opening old database...")
	oldDB, err := gorm.Open(sqlite.Open(oldDBPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to open old database: %v", err)
	}

	// Open or create new database
	var newDB *gorm.DB
	if !*dryRun {
		log.Println("Opening new database...")
		newDBPath := filepath.Join(*newDataDir, "flightlesssomething.db")
		newDB, err = gorm.Open(sqlite.Open(newDBPath), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to open new database: %v", err)
		}

		// Run migrations for new database
		log.Println("Running database migrations...")
		if err := newDB.AutoMigrate(&NewUser{}, &NewBenchmark{}); err != nil {
			log.Fatalf("Failed to migrate new database: %v", err)
		}
	}

	// Migrate users
	log.Println("Migrating users...")
	var oldUsers []OldUser
	if err := oldDB.Find(&oldUsers).Error; err != nil {
		log.Fatalf("Failed to fetch old users: %v", err)
	}
	log.Printf("Found %d users to migrate", len(oldUsers))

	for _, oldUser := range oldUsers {
		log.Printf("  Migrating user: %s (ID: %d, Discord: %s)", oldUser.Username, oldUser.ID, oldUser.DiscordID)

		if !*dryRun {
			newUser := NewUser{
				Model: gorm.Model{
					ID:        oldUser.ID, // Preserve original ID
					CreatedAt: oldUser.CreatedAt,
					UpdatedAt: oldUser.UpdatedAt,
				},
				DiscordID: oldUser.DiscordID,
				Username:  oldUser.Username,
				IsAdmin:   false,
				IsBanned:  false,
			}
			if err := newDB.Create(&newUser).Error; err != nil {
				log.Fatalf("Failed to create user %s: %v", oldUser.Username, err)
			}
			log.Printf("    Created with ID: %d", newUser.ID)
		}
	}

	// Migrate benchmarks
	log.Println("Migrating benchmarks...")
	var oldBenchmarks []OldBenchmark
	if err := oldDB.Find(&oldBenchmarks).Error; err != nil {
		log.Fatalf("Failed to fetch old benchmarks: %v", err)
	}
	log.Printf("Found %d benchmarks to migrate", len(oldBenchmarks))

	successCount := 0
	errorCount := 0

	for i := range oldBenchmarks {
		oldBenchmark := &oldBenchmarks[i]
		log.Printf("  [%d/%d] Migrating benchmark: %s (ID: %d)", i+1, len(oldBenchmarks), oldBenchmark.Title, oldBenchmark.ID)

		// Truncate description if too long (new limit is maxDescriptionLength, old was 500)
		description := oldBenchmark.Description
		if len(description) > maxDescriptionLength {
			description = description[:maxDescriptionLength]
		}

		if !*dryRun {
			// Verify the user exists in the new database (since we preserve user IDs)
			var userExists bool
			if err := newDB.Model(&NewUser{}).Select("count(*) > 0").Where("id = ?", oldBenchmark.UserID).Find(&userExists).Error; err != nil {
				log.Printf("    ERROR: Database error checking user ID %d: %v", oldBenchmark.UserID, err)
				errorCount++
				continue
			}
			if !userExists {
				log.Printf("    WARNING: User ID %d not found in new database, skipping", oldBenchmark.UserID)
				errorCount++
				continue
			}

			newBenchmark := NewBenchmark{
				Model: gorm.Model{
					ID:        oldBenchmark.ID, // Preserve original ID
					CreatedAt: oldBenchmark.CreatedAt,
					UpdatedAt: oldBenchmark.UpdatedAt,
				},
				UserID:      oldBenchmark.UserID, // Use original user ID (preserved)
				Title:       oldBenchmark.Title,
				Description: description,
			}
			if err := newDB.Create(&newBenchmark).Error; err != nil {
				log.Printf("    ERROR: Failed to create benchmark: %v", err)
				errorCount++
				continue
			}

			// Copy benchmark data file (using old ID since we preserve IDs)
			oldDataFile := filepath.Join(oldBenchmarksDir, fmt.Sprintf("%d.bin", oldBenchmark.ID))
			newDataFile := filepath.Join(*newDataDir, "benchmarks", fmt.Sprintf("%d.bin", oldBenchmark.ID))

			if _, err := os.Stat(oldDataFile); os.IsNotExist(err) {
				log.Printf("    WARNING: Data file not found: %s", oldDataFile)
				errorCount++
				continue
			}

			// Read and re-write the data to ensure compatibility
			benchmarkData, err := readBenchmarkData(oldDataFile)
			if err != nil {
				log.Printf("    ERROR: Failed to read data file: %v", err)
				errorCount++
				continue
			}

			// Write to new location
			if err := writeBenchmarkData(newDataFile, benchmarkData); err != nil {
				log.Printf("    ERROR: Failed to write data file: %v", err)
				errorCount++
				continue
			}

			// Create metadata file for new system
			if err := createMetadataFile(*newDataDir, oldBenchmark.ID, benchmarkData); err != nil {
				log.Printf("    ERROR: Failed to create metadata file: %v", err)
				errorCount++
				continue
			}

			log.Printf("    Successfully migrated (ID: %d, %d runs)", oldBenchmark.ID, len(benchmarkData))
			successCount++
		}
	}

	log.Println("\n=== Migration Summary ===")
	log.Printf("Users migrated: %d", len(oldUsers))
	log.Printf("Benchmarks attempted: %d", len(oldBenchmarks))
	if !*dryRun {
		log.Printf("Benchmarks succeeded: %d", successCount)
		log.Printf("Benchmarks failed: %d", errorCount)
	}
	log.Println("=========================")

	if *dryRun {
		log.Println("\nDRY RUN completed - no changes were made")
		log.Println("Remove -dry-run flag to perform actual migration")
	} else {
		log.Println("\nMigration completed successfully!")
	}
}

func readBenchmarkData(filePath string) ([]*BenchmarkData, error) {
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

func writeBenchmarkData(filePath string, benchmarkData []*BenchmarkData) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Printf("Warning: failed to close file %s: %v", filePath, cerr)
		}
	}()

	var buffer bytes.Buffer
	gobEncoder := gob.NewEncoder(&buffer)
	err = gobEncoder.Encode(benchmarkData)
	if err != nil {
		return err
	}

	zstdEncoder, err := zstd.NewWriter(file, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		return err
	}
	defer func() {
		if cerr := zstdEncoder.Close(); cerr != nil {
			log.Printf("Warning: failed to close zstd encoder for %s: %v", filePath, cerr)
		}
	}()

	_, err = zstdEncoder.Write(buffer.Bytes())
	return err
}

func createMetadataFile(dataDir string, benchmarkID uint, benchmarkData []*BenchmarkData) error {
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
