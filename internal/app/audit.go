package app

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// AuditLogEntry represents a single audit log entry written to the JSON log file.
// Fields are structured for log shipper compatibility (e.g. Loki, Elasticsearch, Splunk).
type AuditLogEntry struct {
	Timestamp   string                 `json:"timestamp"`
	UserID      uint                   `json:"user_id"`
	Username    string                 `json:"username"`
	Action      string                 `json:"action"`
	Description string                 `json:"description"`
	TargetType  string                 `json:"target_type"`
	TargetID    uint                   `json:"target_id"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// GetUsernameFromContext extracts the username from a gin context, returning "unknown" if not available.
func GetUsernameFromContext(c interface{ Get(any) (any, bool) }) string {
	if val, exists := c.Get("Username"); exists {
		if s, ok := val.(string); ok && s != "" {
			return s
		}
	}
	return "unknown"
}

// InitAuditLog initializes the audit log directory and file path.
// The logs directory is created alongside (as a sibling of) the data directory.
// For example, if dataDir is /data, logs go to /logs/audit.json.
// Docker deployments must mount the logs directory separately.
func InitAuditLog(dataDir string) error {
	auditLogsDir = filepath.Join(filepath.Dir(dataDir), "logs")
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

	dst, err := os.OpenFile(rotatedPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
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
func writeAuditLog(userID uint, username, action, description, targetType string, targetID uint, details map[string]interface{}) {
	entry := AuditLogEntry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		UserID:      userID,
		Username:    username,
		Action:      action,
		Description: description,
		TargetType:  targetType,
		TargetID:    targetID,
		Details:     details,
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
	if info, statErr := os.Stat(auditLogPath); statErr == nil && info.Size() >= auditLogMaxSize {
		rotateAuditLog()
	}

	f, err := os.OpenFile(auditLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
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
func LogBenchmarkCreated(userID uint, username string, benchmarkID uint, title string, runCount int) {
	writeAuditLog(userID, username, "benchmark_created",
		fmt.Sprintf("User %s (ID %d) created benchmark #%d: %s with %d run(s)", username, userID, benchmarkID, title, runCount),
		"benchmark", benchmarkID, map[string]interface{}{
			"benchmark_title": title,
			"run_count":       runCount,
		})
}

// LogBenchmarkUpdated logs when a benchmark's metadata is updated (title, description, labels)
func LogBenchmarkUpdated(userID uint, username string, benchmarkID uint, title string, changes []string) {
	writeAuditLog(userID, username, "benchmark_updated",
		fmt.Sprintf("User %s (ID %d) updated benchmark #%d: %s (changed: %s)", username, userID, benchmarkID, title, strings.Join(changes, ", ")),
		"benchmark", benchmarkID, map[string]interface{}{
			"benchmark_title": title,
			"changed_fields":  changes,
		})
}

// LogBenchmarkRunsAdded logs when new runs are added to an existing benchmark
func LogBenchmarkRunsAdded(userID uint, username string, benchmarkID uint, title string, runsAdded, totalRuns int) {
	writeAuditLog(userID, username, "benchmark_runs_added",
		fmt.Sprintf("User %s (ID %d) added %d run(s) to benchmark #%d: %s (total runs: %d)", username, userID, runsAdded, benchmarkID, title, totalRuns),
		"benchmark", benchmarkID, map[string]interface{}{
			"benchmark_title": title,
			"runs_added":      runsAdded,
			"total_runs":      totalRuns,
		})
}

// LogBenchmarkRunDeleted logs when a specific run is deleted from a benchmark
func LogBenchmarkRunDeleted(userID uint, username string, benchmarkID uint, title string, runIndex int, runLabel string) {
	writeAuditLog(userID, username, "benchmark_run_deleted",
		fmt.Sprintf("User %s (ID %d) deleted run %d (%s) from benchmark #%d: %s", username, userID, runIndex, runLabel, benchmarkID, title),
		"benchmark", benchmarkID, map[string]interface{}{
			"benchmark_title": title,
			"run_index":       runIndex,
			"run_label":       runLabel,
		})
}

// LogBenchmarkDeleted logs when a benchmark is deleted
func LogBenchmarkDeleted(userID uint, username string, benchmarkID uint, title string) {
	writeAuditLog(userID, username, "benchmark_deleted",
		fmt.Sprintf("User %s (ID %d) deleted benchmark #%d: %s", username, userID, benchmarkID, title),
		"benchmark", benchmarkID, map[string]interface{}{
			"benchmark_title": title,
		})
}

// LogUserAdminGranted logs when a user is granted admin privileges
func LogUserAdminGranted(adminUserID uint, adminUsername string, targetUserID uint, targetUsername string) {
	writeAuditLog(adminUserID, adminUsername, "user_admin_granted",
		fmt.Sprintf("Admin %s (ID %d) granted admin privileges to user %s (ID %d)", adminUsername, adminUserID, targetUsername, targetUserID),
		"user", targetUserID, map[string]interface{}{
			"target_username": targetUsername,
		})
}

// LogUserAdminRevoked logs when admin privileges are revoked from a user
func LogUserAdminRevoked(adminUserID uint, adminUsername string, targetUserID uint, targetUsername string) {
	writeAuditLog(adminUserID, adminUsername, "user_admin_revoked",
		fmt.Sprintf("Admin %s (ID %d) revoked admin privileges from user %s (ID %d)", adminUsername, adminUserID, targetUsername, targetUserID),
		"user", targetUserID, map[string]interface{}{
			"target_username": targetUsername,
		})
}

// LogUserBanned logs when a user is banned
func LogUserBanned(adminUserID uint, adminUsername string, targetUserID uint, targetUsername string) {
	writeAuditLog(adminUserID, adminUsername, "user_banned",
		fmt.Sprintf("Admin %s (ID %d) banned user %s (ID %d)", adminUsername, adminUserID, targetUsername, targetUserID),
		"user", targetUserID, map[string]interface{}{
			"target_username": targetUsername,
		})
}

// LogUserUnbanned logs when a user is unbanned
func LogUserUnbanned(adminUserID uint, adminUsername string, targetUserID uint, targetUsername string) {
	writeAuditLog(adminUserID, adminUsername, "user_unbanned",
		fmt.Sprintf("Admin %s (ID %d) unbanned user %s (ID %d)", adminUsername, adminUserID, targetUsername, targetUserID),
		"user", targetUserID, map[string]interface{}{
			"target_username": targetUsername,
		})
}

// LogUserDeleted logs when a user is deleted
func LogUserDeleted(adminUserID uint, adminUsername string, targetUserID uint, targetUsername string) {
	writeAuditLog(adminUserID, adminUsername, "user_deleted",
		fmt.Sprintf("Admin %s (ID %d) deleted user %s (ID %d)", adminUsername, adminUserID, targetUsername, targetUserID),
		"user", targetUserID, map[string]interface{}{
			"target_username": targetUsername,
		})
}

// LogUserBenchmarksDeleted logs when all benchmarks for a user are deleted
func LogUserBenchmarksDeleted(adminUserID uint, adminUsername string, targetUserID uint, targetUsername string, benchmarkCount int) {
	writeAuditLog(adminUserID, adminUsername, "user_benchmarks_deleted",
		fmt.Sprintf("Admin %s (ID %d) deleted all %d benchmark(s) for user %s (ID %d)", adminUsername, adminUserID, benchmarkCount, targetUsername, targetUserID),
		"user", targetUserID, map[string]interface{}{
			"target_username": targetUsername,
			"benchmark_count": benchmarkCount,
		})
}
