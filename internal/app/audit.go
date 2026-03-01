package app

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	// auditLogMaxSize is the maximum size of the audit log file before rotation (10 MB)
	auditLogMaxSize = 10 * 1024 * 1024
	// auditLogMaxFiles is the maximum number of rotated (compressed) log files to keep
	auditLogMaxFiles = 10
)

var (
	auditLogPath string
	auditLogsDir string
	auditLogMu   sync.Mutex
)

// AuditLogEntry represents a single audit log entry written to the JSON log file
type AuditLogEntry struct {
	Timestamp   string `json:"timestamp"`
	UserID      uint   `json:"user_id"`
	Action      string `json:"action"`
	Description string `json:"description"`
	TargetType  string `json:"target_type"`
	TargetID    uint   `json:"target_id"`
}

// InitAuditLog initializes the audit log directory and file path
func InitAuditLog(dataDir string) error {
	auditLogsDir = filepath.Join(dataDir, "logs")
	auditLogPath = filepath.Join(auditLogsDir, "audit.json")
	return os.MkdirAll(auditLogsDir, 0o750)
}

// rotateAuditLog compresses the current log file and removes old rotated files.
// Must be called with auditLogMu held.
func rotateAuditLog() {
	// Use microseconds in the timestamp to avoid collisions if rotation happens multiple times per second
	rotatedName := fmt.Sprintf("audit-%s.json.gz", time.Now().UTC().Format("20060102-150405.000000"))
	rotatedPath := filepath.Join(auditLogsDir, rotatedName)

	src, err := os.Open(auditLogPath)
	if err != nil {
		fmt.Printf("Warning: failed to open audit log for rotation: %v\n", err)
		return
	}

	dst, err := os.OpenFile(rotatedPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o640)
	if err != nil {
		if cerr := src.Close(); cerr != nil {
			fmt.Printf("Warning: failed to close audit log source: %v\n", cerr)
		}
		fmt.Printf("Warning: failed to create rotated audit log: %v\n", err)
		return
	}

	gz := gzip.NewWriter(dst)
	if _, err := io.Copy(gz, src); err != nil {
		fmt.Printf("Warning: failed to compress audit log: %v\n", err)
		if cerr := gz.Close(); cerr != nil {
			fmt.Printf("Warning: failed to close gzip writer: %v\n", cerr)
		}
		if cerr := dst.Close(); cerr != nil {
			fmt.Printf("Warning: failed to close rotated file: %v\n", cerr)
		}
		if cerr := src.Close(); cerr != nil {
			fmt.Printf("Warning: failed to close audit log source: %v\n", cerr)
		}
		// Clean up the partially written rotated file
		if cerr := os.Remove(rotatedPath); cerr != nil {
			fmt.Printf("Warning: failed to remove partial rotated file: %v\n", cerr)
		}
		return
	}

	if err := gz.Close(); err != nil {
		fmt.Printf("Warning: failed to close gzip writer: %v\n", err)
	}
	if err := dst.Close(); err != nil {
		fmt.Printf("Warning: failed to close rotated file: %v\n", err)
	}
	if err := src.Close(); err != nil {
		fmt.Printf("Warning: failed to close audit log source: %v\n", err)
	}

	// Truncate the current log file
	if err := os.Truncate(auditLogPath, 0); err != nil {
		fmt.Printf("Warning: failed to truncate audit log after rotation: %v\n", err)
	}

	// Clean up old rotated files
	cleanupRotatedAuditLogs()
}

// cleanupRotatedAuditLogs removes the oldest rotated files if there are more than auditLogMaxFiles.
// Must be called with auditLogMu held.
func cleanupRotatedAuditLogs() {
	matches, err := filepath.Glob(filepath.Join(auditLogsDir, "audit-*.json.gz"))
	if err != nil {
		fmt.Printf("Warning: failed to list rotated audit logs: %v\n", err)
		return
	}

	if len(matches) <= auditLogMaxFiles {
		return
	}

	// Sort ascending by filename (timestamps sort naturally)
	sort.Strings(matches)

	// Remove the oldest files
	for _, f := range matches[:len(matches)-auditLogMaxFiles] {
		if err := os.Remove(f); err != nil {
			fmt.Printf("Warning: failed to remove old rotated audit log %s: %v\n", f, err)
		}
	}
}

// writeAuditLog appends a JSON log entry to the audit log file, rotating if needed
func writeAuditLog(userID uint, action, description, targetType string, targetID uint) {
	entry := AuditLogEntry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		UserID:      userID,
		Action:      action,
		Description: description,
		TargetType:  targetType,
		TargetID:    targetID,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Printf("Warning: failed to marshal audit log entry: %v\n", err)
		return
	}
	data = append(data, '\n')

	auditLogMu.Lock()
	defer auditLogMu.Unlock()

	// Check if rotation is needed
	if info, err := os.Stat(auditLogPath); err == nil && info.Size() >= auditLogMaxSize {
		rotateAuditLog()
	}

	f, err := os.OpenFile(auditLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		fmt.Printf("Warning: failed to open audit log file: %v\n", err)
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			fmt.Printf("Warning: failed to close audit log file: %v\n", cerr)
		}
	}()

	if _, err := f.Write(data); err != nil {
		fmt.Printf("Warning: failed to write audit log entry: %v\n", err)
	}
}

// LogBenchmarkCreated logs when a benchmark is created
func LogBenchmarkCreated(userID, benchmarkID uint, title string) {
	writeAuditLog(userID, "Benchmark Created",
		fmt.Sprintf("Created benchmark #%d: %s", benchmarkID, title),
		"benchmark", benchmarkID)
}

// LogBenchmarkUpdated logs when a benchmark is updated
func LogBenchmarkUpdated(userID, benchmarkID uint, title string) {
	writeAuditLog(userID, "Benchmark Updated",
		fmt.Sprintf("Updated benchmark #%d: %s", benchmarkID, title),
		"benchmark", benchmarkID)
}

// LogBenchmarkDeleted logs when a benchmark is deleted
func LogBenchmarkDeleted(userID, benchmarkID uint, title string) {
	writeAuditLog(userID, "Benchmark Deleted",
		fmt.Sprintf("Deleted benchmark #%d: %s", benchmarkID, title),
		"benchmark", benchmarkID)
}

// LogUserAdminGranted logs when a user is granted admin privileges
func LogUserAdminGranted(adminUserID, targetUserID uint, targetUsername string) {
	writeAuditLog(adminUserID, "Admin Granted",
		fmt.Sprintf("Granted admin privileges to user: %s", targetUsername),
		"user", targetUserID)
}

// LogUserAdminRevoked logs when admin privileges are revoked from a user
func LogUserAdminRevoked(adminUserID, targetUserID uint, targetUsername string) {
	writeAuditLog(adminUserID, "Admin Revoked",
		fmt.Sprintf("Revoked admin privileges from user: %s", targetUsername),
		"user", targetUserID)
}

// LogUserBanned logs when a user is banned
func LogUserBanned(adminUserID, targetUserID uint, targetUsername string) {
	writeAuditLog(adminUserID, "User Banned",
		fmt.Sprintf("Banned user: %s", targetUsername),
		"user", targetUserID)
}

// LogUserUnbanned logs when a user is unbanned
func LogUserUnbanned(adminUserID, targetUserID uint, targetUsername string) {
	writeAuditLog(adminUserID, "User Unbanned",
		fmt.Sprintf("Unbanned user: %s", targetUsername),
		"user", targetUserID)
}

// LogUserDeleted logs when a user is deleted
func LogUserDeleted(adminUserID, targetUserID uint, targetUsername string) {
	writeAuditLog(adminUserID, "User Deleted",
		fmt.Sprintf("Deleted user: %s", targetUsername),
		"user", targetUserID)
}

// LogUserBenchmarksDeleted logs when all benchmarks for a user are deleted
func LogUserBenchmarksDeleted(adminUserID, targetUserID uint, targetUsername string) {
	writeAuditLog(adminUserID, "User Benchmarks Deleted",
		fmt.Sprintf("Deleted all benchmarks for user: %s", targetUsername),
		"user", targetUserID)
}
