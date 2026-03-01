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
		LogBenchmarkCreated(1, "testuser", 42, "Test Benchmark", 3)

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
		if entry.Username != "testuser" {
			t.Errorf("Expected Username 'testuser', got %s", entry.Username)
		}
		if entry.Action != "benchmark_created" {
			t.Errorf("Expected Action 'benchmark_created', got %s", entry.Action)
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
		if entry.Details == nil {
			t.Error("Expected Details to be set")
		}
		if title, ok := entry.Details["benchmark_title"]; !ok || title != "Test Benchmark" {
			t.Errorf("Expected Details.benchmark_title 'Test Benchmark', got %v", title)
		}
		if !strings.Contains(entry.Description, "testuser") {
			t.Errorf("Expected Description to contain username, got %s", entry.Description)
		}
	})

	t.Run("appends multiple entries", func(t *testing.T) {
		LogBenchmarkUpdated(2, "user2", 42, "Updated Benchmark", []string{"title"})
		LogBenchmarkDeleted(3, "user3", 42, "Deleted Benchmark")

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

func TestInitAuditLogCreatesAlongsideDataDir(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		t.Fatalf("Failed to create data dir: %v", err)
	}
	if err := InitAuditLog(dataDir); err != nil {
		t.Fatalf("Failed to initialize audit log: %v", err)
	}

	// Verify the logs directory is alongside (sibling of) the data directory
	expectedLogsDir := filepath.Join(tmpDir, "logs")
	if auditLogsDir != expectedLogsDir {
		t.Errorf("Expected auditLogsDir %q, got %q", expectedLogsDir, auditLogsDir)
	}
	expectedLogPath := filepath.Join(tmpDir, "logs", "audit.json")
	if auditLogPath != expectedLogPath {
		t.Errorf("Expected auditLogPath %q, got %q", expectedLogPath, auditLogPath)
	}

	// Verify logs directory was actually created
	info, err := os.Stat(expectedLogsDir)
	if err != nil {
		t.Fatalf("Logs directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Expected logs path to be a directory")
	}

	// Verify logs dir is NOT inside data dir
	rel, relErr := filepath.Rel(dataDir, auditLogsDir)
	if relErr != nil {
		t.Fatalf("Failed to compute relative path: %v", relErr)
	}
	if !strings.HasPrefix(rel, "..") {
		t.Errorf("Logs dir %q should not be inside data dir %q (rel=%q)", auditLogsDir, dataDir, rel)
	}

	// Write a log entry and verify the file appears in the expected location
	LogBenchmarkCreated(1, "testuser", 1, "test", 1)
	if _, err := os.Stat(expectedLogPath); os.IsNotExist(err) {
		t.Error("Audit log file was not created alongside data directory")
	}
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
	LogBenchmarkCreated(1, "creator", 1, "bench", 2)
	LogBenchmarkUpdated(1, "updater", 1, "bench", []string{"title", "description"})
	LogBenchmarkRunsAdded(1, "adder", 1, "bench", 3, 5)
	LogBenchmarkRunDeleted(1, "deleter", 1, "bench", 0, "run-0")
	LogBenchmarkDeleted(1, "deleter", 1, "bench")
	LogUserAdminGranted(1, "admin1", 2, "user2")
	LogUserAdminRevoked(1, "admin1", 2, "user2")
	LogUserBanned(1, "admin1", 2, "user2")
	LogUserUnbanned(1, "admin1", 2, "user2")
	LogUserDeleted(1, "admin1", 2, "user2")
	LogUserBenchmarksDeleted(1, "admin1", 2, "user2", 5)

	logPath := filepath.Join(tmpDir, "logs", "audit.json")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read audit log file: %v", err)
	}

	expectedActions := []string{
		"benchmark_created", "benchmark_updated", "benchmark_runs_added",
		"benchmark_run_deleted", "benchmark_deleted",
		"user_admin_granted", "user_admin_revoked",
		"user_banned", "user_unbanned",
		"user_deleted", "user_benchmarks_deleted",
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

func TestAuditLogStructuredFields(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		t.Fatalf("Failed to create data dir: %v", err)
	}
	if err := InitAuditLog(dataDir); err != nil {
		t.Fatalf("Failed to initialize audit log: %v", err)
	}

	t.Run("benchmark_created has username and details", func(t *testing.T) {
		LogBenchmarkCreated(1, "alice", 10, "GPU Test", 3)

		logPath := filepath.Join(tmpDir, "logs", "audit.json")
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read audit log file: %v", err)
		}

		scanner := bufio.NewScanner(bytes.NewReader(content))
		if !scanner.Scan() {
			t.Fatal("Expected at least one line")
		}

		var entry AuditLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if entry.Username != "alice" {
			t.Errorf("Expected username 'alice', got %q", entry.Username)
		}
		if entry.Details == nil {
			t.Fatal("Expected details to be non-nil")
		}
		if entry.Details["benchmark_title"] != "GPU Test" {
			t.Errorf("Expected benchmark_title 'GPU Test', got %v", entry.Details["benchmark_title"])
		}
		// JSON numbers are decoded as float64
		if runCount, ok := entry.Details["run_count"].(float64); !ok || runCount != 3 {
			t.Errorf("Expected run_count 3, got %v", entry.Details["run_count"])
		}
		if !strings.Contains(entry.Description, "alice") {
			t.Errorf("Expected description to contain 'alice', got %q", entry.Description)
		}
		if !strings.Contains(entry.Description, "3 run(s)") {
			t.Errorf("Expected description to contain '3 run(s)', got %q", entry.Description)
		}
	})

	t.Run("benchmark_updated tracks changed fields", func(t *testing.T) {
		LogBenchmarkUpdated(2, "bob", 10, "GPU Test v2", []string{"title", "labels"})

		logPath := filepath.Join(tmpDir, "logs", "audit.json")
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read audit log file: %v", err)
		}

		// Read the second line (last entry)
		scanner := bufio.NewScanner(bytes.NewReader(content))
		var lastLine []byte
		for scanner.Scan() {
			lastLine = make([]byte, len(scanner.Bytes()))
			copy(lastLine, scanner.Bytes())
		}

		var entry AuditLogEntry
		if err := json.Unmarshal(lastLine, &entry); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if entry.Action != "benchmark_updated" {
			t.Errorf("Expected action 'benchmark_updated', got %q", entry.Action)
		}
		if !strings.Contains(entry.Description, "title, labels") {
			t.Errorf("Expected description to contain changed fields, got %q", entry.Description)
		}
		if entry.Details == nil {
			t.Fatal("Expected details to be non-nil")
		}
		changedFields, ok := entry.Details["changed_fields"].([]interface{})
		if !ok {
			t.Fatalf("Expected changed_fields to be array, got %T", entry.Details["changed_fields"])
		}
		if len(changedFields) != 2 {
			t.Errorf("Expected 2 changed fields, got %d", len(changedFields))
		}
	})

	t.Run("benchmark_run_deleted has run details", func(t *testing.T) {
		LogBenchmarkRunDeleted(3, "charlie", 10, "GPU Test", 2, "run-at-60fps")

		logPath := filepath.Join(tmpDir, "logs", "audit.json")
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read audit log file: %v", err)
		}

		scanner := bufio.NewScanner(bytes.NewReader(content))
		var lastLine []byte
		for scanner.Scan() {
			lastLine = make([]byte, len(scanner.Bytes()))
			copy(lastLine, scanner.Bytes())
		}

		var entry AuditLogEntry
		if err := json.Unmarshal(lastLine, &entry); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if entry.Action != "benchmark_run_deleted" {
			t.Errorf("Expected action 'benchmark_run_deleted', got %q", entry.Action)
		}
		if !strings.Contains(entry.Description, "run-at-60fps") {
			t.Errorf("Expected description to contain run label, got %q", entry.Description)
		}
		if entry.Details["run_label"] != "run-at-60fps" {
			t.Errorf("Expected run_label 'run-at-60fps', got %v", entry.Details["run_label"])
		}
	})

	t.Run("benchmark_runs_added has count details", func(t *testing.T) {
		LogBenchmarkRunsAdded(4, "dave", 10, "GPU Test", 2, 5)

		logPath := filepath.Join(tmpDir, "logs", "audit.json")
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read audit log file: %v", err)
		}

		scanner := bufio.NewScanner(bytes.NewReader(content))
		var lastLine []byte
		for scanner.Scan() {
			lastLine = make([]byte, len(scanner.Bytes()))
			copy(lastLine, scanner.Bytes())
		}

		var entry AuditLogEntry
		if err := json.Unmarshal(lastLine, &entry); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if entry.Action != "benchmark_runs_added" {
			t.Errorf("Expected action 'benchmark_runs_added', got %q", entry.Action)
		}
		if runsAdded, ok := entry.Details["runs_added"].(float64); !ok || runsAdded != 2 {
			t.Errorf("Expected runs_added 2, got %v", entry.Details["runs_added"])
		}
		if totalRuns, ok := entry.Details["total_runs"].(float64); !ok || totalRuns != 5 {
			t.Errorf("Expected total_runs 5, got %v", entry.Details["total_runs"])
		}
	})

	t.Run("user_benchmarks_deleted has benchmark count", func(t *testing.T) {
		LogUserBenchmarksDeleted(5, "admin", 10, "targetuser", 7)

		logPath := filepath.Join(tmpDir, "logs", "audit.json")
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read audit log file: %v", err)
		}

		scanner := bufio.NewScanner(bytes.NewReader(content))
		var lastLine []byte
		for scanner.Scan() {
			lastLine = make([]byte, len(scanner.Bytes()))
			copy(lastLine, scanner.Bytes())
		}

		var entry AuditLogEntry
		if err := json.Unmarshal(lastLine, &entry); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if entry.Action != "user_benchmarks_deleted" {
			t.Errorf("Expected action 'user_benchmarks_deleted', got %q", entry.Action)
		}
		if benchmarkCount, ok := entry.Details["benchmark_count"].(float64); !ok || benchmarkCount != 7 {
			t.Errorf("Expected benchmark_count 7, got %v", entry.Details["benchmark_count"])
		}
		if !strings.Contains(entry.Description, "7 benchmark(s)") {
			t.Errorf("Expected description to contain benchmark count, got %q", entry.Description)
		}
	})
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
	LogBenchmarkCreated(1, "testuser", 1, "trigger rotation", 1)

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
	if closeErr := gz.Close(); closeErr != nil {
		t.Fatalf("Failed to close gzip reader: %v", closeErr)
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
