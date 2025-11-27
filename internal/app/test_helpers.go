package app

import (
	"testing"
)

// setupTestDB creates a temporary test database
func setupTestDB(t *testing.T) *DBInstance {
	t.Helper()
	tmpDir := t.TempDir()
	db, err := InitDB(tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	return db
}

// cleanupTestDB cleans up the test database
func cleanupTestDB(t *testing.T, db *DBInstance) {
	t.Helper()
	// The temp directory is automatically cleaned up by t.TempDir()
	if db != nil && db.DB != nil {
		sqlDB, err := db.DB.DB()
		if err == nil {
			if closeErr := sqlDB.Close(); closeErr != nil {
				t.Logf("Warning: failed to close database: %v", closeErr)
			}
		}
	}
}
