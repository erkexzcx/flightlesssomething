package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	auditLogPath string
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
	logsDir := filepath.Join(dataDir, "logs")
	auditLogPath = filepath.Join(logsDir, "audit.json")
	return os.MkdirAll(logsDir, 0o750)
}

// writeAuditLog appends a JSON log entry to the audit log file
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
