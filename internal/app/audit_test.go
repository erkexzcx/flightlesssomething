package app

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAuditLog(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		t.Fatalf("Failed to create data dir: %v", err)
	}
	if err := InitAuditLog(dataDir); err != nil {
		t.Fatalf("Failed to initialize audit log: %v", err)
	}

	logsDir := filepath.Join(tmpDir, "logs")

	t.Run("creates audit log file and writes entry", func(t *testing.T) {
		LogBenchmarkCreated(1, 42, "Test Benchmark")

		// Verify file exists
		logPath := filepath.Join(logsDir, "audit.json")
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Fatal("Expected audit log file to be created")
		}

		// Read and verify contents
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read audit log file: %v", err)
		}

		scanner := bufio.NewScanner(bytes.NewReader(content))
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

		logPath := filepath.Join(logsDir, "audit.json")
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read audit log file: %v", err)
		}

		lineCount := 0
		scanner := bufio.NewScanner(bytes.NewReader(content))
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
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		t.Fatalf("Failed to create data dir: %v", err)
	}
	if err := InitAuditLog(dataDir); err != nil {
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
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read audit log file: %v", err)
	}

	expectedActions := []string{
		"Benchmark Created", "Benchmark Updated", "Benchmark Deleted",
		"Admin Granted", "Admin Revoked",
		"User Banned", "User Unbanned",
		"User Deleted", "User Benchmarks Deleted",
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
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

func TestAuditLogRotation(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		t.Fatalf("Failed to create data dir: %v", err)
	}
	if err := InitAuditLog(dataDir); err != nil {
		t.Fatalf("Failed to initialize audit log: %v", err)
	}

	logsDir := filepath.Join(tmpDir, "logs")
	logPath := filepath.Join(logsDir, "audit.json")

	// Write enough data to exceed the rotation threshold
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("Failed to create audit log file: %v", err)
	}
	bigLine := strings.Repeat("x", 1024) + "\n"
	for written := 0; written < auditLogMaxSize+1; written += len(bigLine) {
		if _, writeErr := f.WriteString(bigLine); writeErr != nil {
			t.Fatalf("Failed to write test data: %v", writeErr)
		}
	}
	if closeErr := f.Close(); closeErr != nil {
		t.Fatalf("Failed to close test file: %v", closeErr)
	}

	// Writing a new entry should trigger rotation
	LogBenchmarkCreated(1, 1, "trigger rotation")

	// Verify rotated file exists
	matches, err := filepath.Glob(filepath.Join(logsDir, "audit-*.json.gz"))
	if err != nil {
		t.Fatalf("Failed to glob rotated files: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("Expected 1 rotated file, got %d", len(matches))
	}

	// Verify the rotated file is valid gzip
	gzData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("Failed to read rotated file: %v", err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(gzData))
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	buf := make([]byte, 1024)
	if _, readErr := gz.Read(buf); readErr != nil && !errors.Is(readErr, io.EOF) {
		t.Fatalf("Failed to read from rotated gzip file: %v", readErr)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("Failed to close gzip reader: %v", err)
	}

	// Verify the current log file only has the new entry
	currentContent, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read current log file: %v", err)
	}

	lineCount := 0
	scanner := bufio.NewScanner(bytes.NewReader(currentContent))
	for scanner.Scan() {
		lineCount++
	}
	if lineCount != 1 {
		t.Errorf("Expected 1 line in current log after rotation, got %d", lineCount)
	}
}

func TestAuditLogRotationCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		t.Fatalf("Failed to create data dir: %v", err)
	}
	if err := InitAuditLog(dataDir); err != nil {
		t.Fatalf("Failed to initialize audit log: %v", err)
	}

	logsDir := filepath.Join(tmpDir, "logs")

	// Create more than auditLogMaxFiles rotated files
	for i := 0; i < auditLogMaxFiles+3; i++ {
		name := filepath.Join(logsDir, fmt.Sprintf("audit-20250101-%06d.json.gz", i))
		if err := os.WriteFile(name, []byte("test"), 0o600); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Verify we have more than the max
	matches, err := filepath.Glob(filepath.Join(logsDir, "audit-*.json.gz"))
	if err != nil {
		t.Fatalf("Failed to glob rotated files: %v", err)
	}
	if len(matches) != auditLogMaxFiles+3 {
		t.Fatalf("Expected %d files before cleanup, got %d", auditLogMaxFiles+3, len(matches))
	}

	// Run cleanup
	auditLogMu.Lock()
	cleanupRotatedAuditLogs()
	auditLogMu.Unlock()

	// Verify we're down to the max
	matches, err = filepath.Glob(filepath.Join(logsDir, "audit-*.json.gz"))
	if err != nil {
		t.Fatalf("Failed to glob rotated files after cleanup: %v", err)
	}
	if len(matches) != auditLogMaxFiles {
		t.Errorf("Expected %d files after cleanup, got %d", auditLogMaxFiles, len(matches))
	}
}
