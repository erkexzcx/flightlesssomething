package app

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/klauspost/compress/zstd"
)

// MigrateBenchmarkStorageToV2 migrates all benchmark data files from V1 to V2 format
// This function scans all .bin files and re-encodes them using the new streaming-friendly format
func MigrateBenchmarkStorageToV2(dataDir string) error {
	log.Println("=== Starting Benchmark Storage Format Migration (V1 → V2) ===")
	
	benchmarksDir := filepath.Join(dataDir, "benchmarks")
	if _, err := os.Stat(benchmarksDir); os.IsNotExist(err) {
		log.Println("No benchmarks directory found - nothing to migrate")
		return nil
	}
	
	// Find all .bin files
	files, err := filepath.Glob(filepath.Join(benchmarksDir, "*.bin"))
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
		
		log.Printf("Processing benchmark %d...", benchmarkID)
		
		// Check if already in V2 format
		isV2, err := isBenchmarkFormatV2(benchmarkID)
		if err != nil {
			log.Printf("  ERROR: Failed to check format: %v", err)
			errorCount++
			continue
		}
		
		if isV2 {
			log.Printf("  Already in V2 format - skipping")
			skipCount++
			continue
		}
		
		// Load data using legacy reader
		log.Printf("  Loading V1 format data...")
		benchmarkData, err := retrieveBenchmarkDataLegacy(benchmarkID)
		if err != nil {
			log.Printf("  ERROR: Failed to load data: %v", err)
			errorCount++
			continue
		}
		
		// Create backup of original file
		backupPath := filePath + ".v1.bak"
		if err := os.Rename(filePath, backupPath); err != nil {
			log.Printf("  ERROR: Failed to create backup: %v", err)
			errorCount++
			continue
		}
		
		// Store in new V2 format
		log.Printf("  Converting to V2 format (%d runs)...", len(benchmarkData))
		if err := StoreBenchmarkData(benchmarkData, benchmarkID); err != nil {
			// Restore backup on failure
			log.Printf("  ERROR: Failed to save V2 format: %v", err)
			log.Printf("  Restoring backup...")
			if restoreErr := os.Rename(backupPath, filePath); restoreErr != nil {
				log.Printf("  CRITICAL: Failed to restore backup: %v", restoreErr)
			}
			errorCount++
			continue
		}
		
		// Verify the conversion worked
		log.Printf("  Verifying conversion...")
		verifyData, err := RetrieveBenchmarkData(benchmarkID)
		if err != nil {
			log.Printf("  ERROR: Verification failed: %v", err)
			log.Printf("  Restoring backup...")
			os.Remove(filePath)
			if restoreErr := os.Rename(backupPath, filePath); restoreErr != nil {
				log.Printf("  CRITICAL: Failed to restore backup: %v", restoreErr)
			}
			errorCount++
			continue
		}
		
		if len(verifyData) != len(benchmarkData) {
			log.Printf("  ERROR: Data mismatch after conversion (expected %d runs, got %d)", 
				len(benchmarkData), len(verifyData))
			log.Printf("  Restoring backup...")
			os.Remove(filePath)
			if restoreErr := os.Rename(backupPath, filePath); restoreErr != nil {
				log.Printf("  CRITICAL: Failed to restore backup: %v", restoreErr)
			}
			errorCount++
			continue
		}
		
		// Success! Remove backup
		if err := os.Remove(backupPath); err != nil {
			log.Printf("  WARNING: Failed to remove backup file: %v", err)
		}
		
		log.Printf("  ✓ Successfully migrated to V2 format")
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
	defer file.Close()
	
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
