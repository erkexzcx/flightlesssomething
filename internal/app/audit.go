package app

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreateAuditLog creates a new audit log entry
func CreateAuditLog(db *DBInstance, userID uint, action, description, targetType string, targetID uint) error {
	log := AuditLog{
		UserID:      userID,
		Action:      action,
		Description: description,
		TargetType:  targetType,
		TargetID:    targetID,
	}
	return db.DB.Create(&log).Error
}

// HandleListAuditLogs returns a paginated list of audit logs (admin only)
func HandleListAuditLogs(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
		if err != nil || page < 1 {
			page = 1
		}
		perPage, err := strconv.Atoi(c.DefaultQuery("per_page", "50"))
		if err != nil || perPage < 1 || perPage > 100 {
			perPage = 50
		}

		// Build query with optional filters
		query := db.DB.Model(&AuditLog{}).Preload("User")

		// Filter by user ID
		if userIDStr := c.Query("user_id"); userIDStr != "" {
			if userID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
				query = query.Where("user_id = ?", userID)
			}
		}

		// Filter by action
		if action := c.Query("action"); action != "" {
			query = query.Where("action LIKE ?", "%"+action+"%")
		}

		// Filter by target type
		if targetType := c.Query("target_type"); targetType != "" {
			query = query.Where("target_type = ?", targetType)
		}

		// Get total count with filters applied
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		// Get audit logs
		var logs []AuditLog
		offset := (page - 1) * perPage
		if err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&logs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		// Calculate total pages
		totalPages := int((total + int64(perPage) - 1) / int64(perPage))

		c.JSON(http.StatusOK, gin.H{
			"logs":        logs,
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
		})
	}
}

// LogBenchmarkCreated logs when a benchmark is created
func LogBenchmarkCreated(db *DBInstance, userID, benchmarkID uint, title string) {
	if err := CreateAuditLog(db, userID, "Benchmark Created",
		fmt.Sprintf("Created benchmark #%d: %s", benchmarkID, title),
		"benchmark", benchmarkID); err != nil {
		fmt.Printf("Warning: failed to create audit log: %v\n", err)
	}
}

// LogBenchmarkUpdated logs when a benchmark is updated
func LogBenchmarkUpdated(db *DBInstance, userID, benchmarkID uint, title string) {
	if err := CreateAuditLog(db, userID, "Benchmark Updated",
		fmt.Sprintf("Updated benchmark #%d: %s", benchmarkID, title),
		"benchmark", benchmarkID); err != nil {
		fmt.Printf("Warning: failed to create audit log: %v\n", err)
	}
}

// LogBenchmarkDeleted logs when a benchmark is deleted
func LogBenchmarkDeleted(db *DBInstance, userID, benchmarkID uint, title string) {
	if err := CreateAuditLog(db, userID, "Benchmark Deleted",
		fmt.Sprintf("Deleted benchmark #%d: %s", benchmarkID, title),
		"benchmark", benchmarkID); err != nil {
		fmt.Printf("Warning: failed to create audit log: %v\n", err)
	}
}

// LogUserAdminGranted logs when a user is granted admin privileges
func LogUserAdminGranted(db *DBInstance, adminUserID, targetUserID uint, targetUsername string) {
	if err := CreateAuditLog(db, adminUserID, "Admin Granted",
		fmt.Sprintf("Granted admin privileges to user: %s", targetUsername),
		"user", targetUserID); err != nil {
		fmt.Printf("Warning: failed to create audit log: %v\n", err)
	}
}

// LogUserAdminRevoked logs when admin privileges are revoked from a user
func LogUserAdminRevoked(db *DBInstance, adminUserID, targetUserID uint, targetUsername string) {
	if err := CreateAuditLog(db, adminUserID, "Admin Revoked",
		fmt.Sprintf("Revoked admin privileges from user: %s", targetUsername),
		"user", targetUserID); err != nil {
		fmt.Printf("Warning: failed to create audit log: %v\n", err)
	}
}

// LogUserBanned logs when a user is banned
func LogUserBanned(db *DBInstance, adminUserID, targetUserID uint, targetUsername string) {
	if err := CreateAuditLog(db, adminUserID, "User Banned",
		fmt.Sprintf("Banned user: %s", targetUsername),
		"user", targetUserID); err != nil {
		fmt.Printf("Warning: failed to create audit log: %v\n", err)
	}
}

// LogUserUnbanned logs when a user is unbanned
func LogUserUnbanned(db *DBInstance, adminUserID, targetUserID uint, targetUsername string) {
	if err := CreateAuditLog(db, adminUserID, "User Unbanned",
		fmt.Sprintf("Unbanned user: %s", targetUsername),
		"user", targetUserID); err != nil {
		fmt.Printf("Warning: failed to create audit log: %v\n", err)
	}
}

// LogUserDeleted logs when a user is deleted
func LogUserDeleted(db *DBInstance, adminUserID, targetUserID uint, targetUsername string) {
	if err := CreateAuditLog(db, adminUserID, "User Deleted",
		fmt.Sprintf("Deleted user: %s", targetUsername),
		"user", targetUserID); err != nil {
		fmt.Printf("Warning: failed to create audit log: %v\n", err)
	}
}

// LogUserBenchmarksDeleted logs when all benchmarks for a user are deleted
func LogUserBenchmarksDeleted(db *DBInstance, adminUserID, targetUserID uint, targetUsername string) {
	if err := CreateAuditLog(db, adminUserID, "User Benchmarks Deleted",
		fmt.Sprintf("Deleted all benchmarks for user: %s", targetUsername),
		"user", targetUserID); err != nil {
		fmt.Printf("Warning: failed to create audit log: %v\n", err)
	}
}
