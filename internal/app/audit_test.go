package app

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAuditLog(t *testing.T) {
	tmpDir := t.TempDir()
	if err := InitAuditLog(tmpDir); err != nil {
		t.Fatalf("Failed to initialize audit log: %v", err)
	}

	t.Run("creates audit log file and writes entry", func(t *testing.T) {
		LogBenchmarkCreated(1, 42, "Test Benchmark")

		// Verify file exists
		logPath := filepath.Join(tmpDir, "logs", "audit.json")
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Fatal("Expected audit log file to be created")
		}

		// Read and verify contents
		f, err := os.Open(logPath)
		if err != nil {
			t.Fatalf("Failed to open audit log file: %v", err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		if !scanner.Scan() {
			t.Fatal("Expected at least one line in audit log file")
		}

		var entry AuditLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal audit log entry: %v", err)
		}

		if entry.UserID != 1 {
			t.Errorf("Expected UserID 1, got %d", entry.UserID)
		}
		if entry.Action != "Benchmark Created" {
			t.Errorf("Expected Action 'Benchmark Created', got %s", entry.Action)
		}
		if entry.TargetType != "benchmark" {
			t.Errorf("Expected TargetType 'benchmark', got %s", entry.TargetType)
		}
		if entry.TargetID != 42 {
			t.Errorf("Expected TargetID 42, got %d", entry.TargetID)
		}
		if entry.Timestamp == "" {
			t.Error("Expected Timestamp to be set")
		}
	})

	t.Run("appends multiple entries", func(t *testing.T) {
		LogBenchmarkUpdated(2, 42, "Updated Benchmark")
		LogBenchmarkDeleted(3, 42, "Deleted Benchmark")

		logPath := filepath.Join(tmpDir, "logs", "audit.json")
		f, err := os.Open(logPath)
		if err != nil {
			t.Fatalf("Failed to open audit log file: %v", err)
		}
		defer f.Close()

		lineCount := 0
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			lineCount++
			var entry AuditLogEntry
			if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
				t.Fatalf("Failed to unmarshal line %d: %v", lineCount, err)
			}
		}

		// 1 from first subtest + 2 from this subtest
		if lineCount != 3 {
			t.Errorf("Expected 3 log entries, got %d", lineCount)
		}
	})
}

func TestAllLogFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	if err := InitAuditLog(tmpDir); err != nil {
		t.Fatalf("Failed to initialize audit log: %v", err)
	}

	// Call all log functions
	LogBenchmarkCreated(1, 1, "bench")
	LogBenchmarkUpdated(1, 1, "bench")
	LogBenchmarkDeleted(1, 1, "bench")
	LogUserAdminGranted(1, 2, "user2")
	LogUserAdminRevoked(1, 2, "user2")
	LogUserBanned(1, 2, "user2")
	LogUserUnbanned(1, 2, "user2")
	LogUserDeleted(1, 2, "user2")
	LogUserBenchmarksDeleted(1, 2, "user2")

	logPath := filepath.Join(tmpDir, "logs", "audit.json")
	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Failed to open audit log file: %v", err)
	}
	defer f.Close()

	expectedActions := []string{
		"Benchmark Created", "Benchmark Updated", "Benchmark Deleted",
		"Admin Granted", "Admin Revoked",
		"User Banned", "User Unbanned",
		"User Deleted", "User Benchmarks Deleted",
	}

	scanner := bufio.NewScanner(f)
	i := 0
	for scanner.Scan() {
		var entry AuditLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal entry %d: %v", i, err)
		}
		if i < len(expectedActions) && entry.Action != expectedActions[i] {
			t.Errorf("Entry %d: expected action %q, got %q", i, expectedActions[i], entry.Action)
		}
		i++
	}

	if i != len(expectedActions) {
		t.Errorf("Expected %d entries, got %d", len(expectedActions), i)
	}
}
