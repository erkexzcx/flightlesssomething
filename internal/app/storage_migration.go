package app

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	
	"github.com/klauspost/compress/zstd"
)

// MigrateBenchmarkStorageToV2 migrates all benchmark data files from V1 to V2 format
// This function scans all .bin files and re-encodes them using the new streaming-friendly format
// Migration is done in-place without backups for efficiency
func MigrateBenchmarkStorageToV2(dataDir string) error {
	log.Println("=== Starting Benchmark Storage Format Migration (V1 → V2) ===")
	
	benchmarksDirPath := filepath.Join(dataDir, "benchmarks")
	if _, err := os.Stat(benchmarksDirPath); os.IsNotExist(err) {
		log.Println("No benchmarks directory found - nothing to migrate")
		return nil
	}
	
	// Find all .bin files
	files, err := filepath.Glob(filepath.Join(benchmarksDirPath, "*.bin"))
	if err != nil {
		return fmt.Errorf("failed to list benchmark files: %w", err)
	}
	
	if len(files) == 0 {
		log.Println("No benchmark files found - nothing to migrate")
		return nil
	}
	
	log.Printf("Found %d benchmark file(s) to check\n", len(files))
	
	successCount := 0
	skipCount := 0
	errorCount := 0
	
	for _, filePath := range files {
		// Extract benchmark ID from filename
		basename := filepath.Base(filePath)
		idStr := strings.TrimSuffix(basename, ".bin")
		
		var benchmarkID uint
		if _, err := fmt.Sscanf(idStr, "%d", &benchmarkID); err != nil {
			log.Printf("Skipping file with invalid name: %s", basename)
			skipCount++
			continue
		}
		
		// Check if already in V2 format
		isV2, err := isBenchmarkFormatV2(benchmarkID)
		if err != nil {
			log.Printf("Benchmark %d: ERROR - Failed to check format: %v", benchmarkID, err)
			errorCount++
			continue
		}
		
		if isV2 {
			log.Printf("Benchmark %d: Already in V2 format - skipped", benchmarkID)
			skipCount++
			continue
		}
		
		// Load data using legacy reader (loads all into memory)
		benchmarkData, err := retrieveBenchmarkDataLegacy(benchmarkID)
		if err != nil {
			log.Printf("Benchmark %d: ERROR - Failed to load V1 data: %v", benchmarkID, err)
			errorCount++
			continue
		}
		
		runCount := len(benchmarkData)
		
		// Store in new V2 format (in-place, overwrites old file)
		if err := StoreBenchmarkData(benchmarkData, benchmarkID); err != nil {
			log.Printf("Benchmark %d: ERROR - Failed to save V2: %v", benchmarkID, err)
			errorCount++
			continue
		}
		
		// Note: StoreBenchmarkData already calls storeBenchmarkMetadata internally
		// which now includes JSON size calculation, so metadata is already generated
		
		log.Printf("Benchmark %d: ✓ Migrated to V2 format (%d runs)", benchmarkID, runCount)
		
		// Clear loaded data to help GC
		benchmarkData = nil //nolint:ineffassign // Intentional to help GC reclaim memory
		runtime.GC()
		
		successCount++
	}
	
	log.Println("\n=== Storage Migration Summary ===")
	log.Printf("Total files found: %d", len(files))
	log.Printf("Successfully migrated: %d", successCount)
	log.Printf("Already V2 (skipped): %d", skipCount)
	log.Printf("Failed: %d", errorCount)
	log.Println("==================================")
	
	if errorCount > 0 {
		return fmt.Errorf("storage migration completed with %d errors", errorCount)
	}
	
	log.Println("Storage format migration completed successfully!")
	return nil
}

// isBenchmarkFormatV2 checks if a benchmark file is in V2 format
func isBenchmarkFormatV2(benchmarkID uint) (bool, error) {
	filePath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.bin", benchmarkID))
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = file.Close() //nolint:errcheck // Defer close, error not critical in read-only operation
	}()
	
	// Set up decompression
	zstdDecoder, err := zstd.NewReader(file, zstd.WithDecoderConcurrency(2))
	if err != nil {
		return false, err
	}
	defer zstdDecoder.Close()
	
	// Try to read header
	gobDecoder := gob.NewDecoder(zstdDecoder)
	var header fileHeader
	if err := gobDecoder.Decode(&header); err != nil {
		// If header decode fails, it's V1 format
		return false, nil
	}
	
	// Check if version matches V2
	return header.Version == storageFormatVersion, nil
}

// MigratePreCalculateStats pre-calculates statistics for all existing benchmarks
// This creates .stats files from existing .bin raw data files
func MigratePreCalculateStats(dataDir string) error {
	log.Println("=== Starting Pre-Calculate Stats Migration (V3 → V4) ===")

	benchmarksDirPath := filepath.Join(dataDir, "benchmarks")
	if _, err := os.Stat(benchmarksDirPath); os.IsNotExist(err) {
		log.Println("No benchmarks directory found - nothing to migrate")
		return nil
	}

	// Find all .bin files
	files, err := filepath.Glob(filepath.Join(benchmarksDirPath, "*.bin"))
	if err != nil {
		return fmt.Errorf("failed to list benchmark files: %w", err)
	}

	if len(files) == 0 {
		log.Println("No benchmark files found - nothing to migrate")
		return nil
	}

	log.Printf("Found %d benchmark file(s) to process\n", len(files))

	successCount := 0
	skipCount := 0
	errorCount := 0

	for _, filePath := range files {
		basename := filepath.Base(filePath)
		idStr := strings.TrimSuffix(basename, ".bin")

		var benchmarkID uint
		if _, err := fmt.Sscanf(idStr, "%d", &benchmarkID); err != nil {
			log.Printf("Skipping file with invalid name: %s", basename)
			skipCount++
			continue
		}

		// Check if .stats file already exists
		statsPath := filepath.Join(benchmarksDirPath, fmt.Sprintf("%d.stats", benchmarkID))
		if _, err := os.Stat(statsPath); err == nil {
			log.Printf("Benchmark %d: Stats file already exists - skipped", benchmarkID)
			skipCount++
			continue
		}

		// Load raw data
		benchmarkData, err := RetrieveBenchmarkData(benchmarkID)
		if err != nil {
			log.Printf("Benchmark %d: ERROR - Failed to load data: %v", benchmarkID, err)
			errorCount++
			continue
		}

		// Compute pre-calculated stats
		preCalc := ComputePreCalculatedRuns(benchmarkData)

		// Store stats
		if err := StorePreCalculatedStats(preCalc, benchmarkID); err != nil {
			log.Printf("Benchmark %d: ERROR - Failed to save stats: %v", benchmarkID, err)
			errorCount++
			continue
		}

		log.Printf("Benchmark %d: ✓ Pre-calculated stats generated (%d runs)", benchmarkID, len(benchmarkData))

		// Clear loaded data to help GC
		runtime.GC()

		successCount++
	}

	log.Println("\n=== Pre-Calculate Stats Migration Summary ===")
	log.Printf("Total files found: %d", len(files))
	log.Printf("Successfully processed: %d", successCount)
	log.Printf("Already exists (skipped): %d", skipCount)
	log.Printf("Failed: %d", errorCount)
	log.Println("===============================================")

	if errorCount > 0 {
		return fmt.Errorf("stats migration completed with %d errors", errorCount)
	}

	log.Println("Pre-calculate stats migration completed successfully!")
	return nil
}
