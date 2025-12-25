package app

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
)

// HandleListUsers returns a list of all users (admin only)
func HandleListUsers(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
		if err != nil || page < 1 {
			page = 1
		}
		perPage, err := strconv.Atoi(c.DefaultQuery("per_page", "10"))
		if err != nil || perPage < 1 || perPage > 100 {
			perPage = 10
		}

		// Build query with optional search filter
		query := db.DB.Model(&User{})
		if search := c.Query("search"); search != "" {
			query = query.Where("username LIKE ? OR discord_id LIKE ?", "%"+search+"%", "%"+search+"%")
		}

		// Get total count with filter applied
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		// Get users with benchmark count
		var users []User
		offset := (page - 1) * perPage

		// First get all users with filter
		if err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		// Then count benchmarks for each user
		for i := range users {
			var count int64
			db.DB.Model(&Benchmark{}).Where("user_id = ?", users[i].ID).Count(&count)
			users[i].BenchmarkCount = int(count)

			// Count API tokens for each user
			var tokenCount int64
			db.DB.Model(&APIToken{}).Where("user_id = ?", users[i].ID).Count(&tokenCount)
			users[i].APITokenCount = int(tokenCount)
		}

		// Sort by benchmark count (top uploaders first)
		sort.Slice(users, func(i, j int) bool {
			return users[i].BenchmarkCount > users[j].BenchmarkCount
		})

		// Calculate total pages
		totalPages := int((total + int64(perPage) - 1) / int64(perPage))

		c.JSON(http.StatusOK, gin.H{
			"users":       users,
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
		})
	}
}

// HandleDeleteUser deletes a user and optionally their data (admin only)
func HandleDeleteUser(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		deleteData := c.Query("delete_data") == "true"

		var user User
		if err := db.DB.First(&user, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Prevent self-deletion
		if adminUserID, exists := c.Get("UserID"); exists {
			if uid, ok := adminUserID.(uint); ok && user.ID == uid {
				c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete your own account"})
				return
			}
		}

		// Store username for audit log
		username := user.Username

		if deleteData {
			// Get all user's benchmarks to delete their data files
			var benchmarks []Benchmark
			if err := db.DB.Where("user_id = ?", user.ID).Find(&benchmarks).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find user benchmarks"})
				return
			}

			// Delete all benchmark data files
			for i := range benchmarks {
				if err := DeleteBenchmarkData(benchmarks[i].ID); err != nil {
					// Log but continue
					fmt.Printf("Warning: failed to delete data for benchmark %d\n", benchmarks[i].ID)
				}
			}
		}

		// Delete user (cascade will handle benchmarks)
		if err := db.DB.Delete(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
			return
		}

		// Log the action
		if adminUserID, exists := c.Get("UserID"); exists {
			if uid, ok := adminUserID.(uint); ok {
				LogUserDeleted(db, uid, user.ID, username)
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
	}
}

// HandleDeleteUserBenchmarks deletes all benchmarks for a user (admin only)
func HandleDeleteUserBenchmarks(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var user User
		if err := db.DB.First(&user, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Get all user's benchmarks
		var benchmarks []Benchmark
		if err := db.DB.Where("user_id = ?", user.ID).Find(&benchmarks).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find user benchmarks"})
			return
		}

		// Delete all benchmark data files
		for i := range benchmarks {
			if err := DeleteBenchmarkData(benchmarks[i].ID); err != nil {
				fmt.Printf("Warning: failed to delete data for benchmark %d\n", benchmarks[i].ID)
			}
		}

		// Delete all benchmarks from database
		if err := db.DB.Where("user_id = ?", user.ID).Delete(&Benchmark{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete benchmarks"})
			return
		}

		// Log the action
		if adminUserID, exists := c.Get("UserID"); exists {
			if uid, ok := adminUserID.(uint); ok {
				LogUserBenchmarksDeleted(db, uid, user.ID, user.Username)
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "all user benchmarks deleted"})
	}
}

// HandleBanUser bans or unbans a user (admin only)
func HandleBanUser(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req struct {
			Banned bool `json:"banned"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		var user User
		if err := db.DB.First(&user, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Prevent self-ban
		if adminUserID, exists := c.Get("UserID"); exists {
			if uid, ok := adminUserID.(uint); ok && user.ID == uid && req.Banned {
				c.JSON(http.StatusBadRequest, gin.H{"error": "cannot ban your own account"})
				return
			}
		}

		user.IsBanned = req.Banned
		if err := db.DB.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}

		// Log the action
		if adminUserID, exists := c.Get("UserID"); exists {
			if uid, ok := adminUserID.(uint); ok {
				if req.Banned {
					LogUserBanned(db, uid, user.ID, user.Username)
				} else {
					LogUserUnbanned(db, uid, user.ID, user.Username)
				}
			}
		}

		c.JSON(http.StatusOK, user)
	}
}

// HandleToggleUserAdmin toggles admin status for a user (admin only)
func HandleToggleUserAdmin(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req struct {
			IsAdmin bool `json:"is_admin"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		var user User
		if err := db.DB.First(&user, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Prevent self-demotion from admin
		if adminUserID, exists := c.Get("UserID"); exists {
			if uid, ok := adminUserID.(uint); ok && user.ID == uid && !req.IsAdmin {
				c.JSON(http.StatusBadRequest, gin.H{"error": "cannot revoke your own admin privileges"})
				return
			}
		}

		user.IsAdmin = req.IsAdmin
		if err := db.DB.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}

		// Log the action
		if adminUserID, exists := c.Get("UserID"); exists {
			if uid, ok := adminUserID.(uint); ok {
				if req.IsAdmin {
					LogUserAdminGranted(db, uid, user.ID, user.Username)
				} else {
					LogUserAdminRevoked(db, uid, user.ID, user.Username)
				}
			}
		}

		c.JSON(http.StatusOK, user)
	}
}

// HandleRepopulateSearchMetadata re-populates RunNames and Specifications for all benchmarks (admin only)
func HandleRepopulateSearchMetadata(db *DBInstance) gin.HandlerFunc {
return func(c *gin.Context) {
// Get all benchmarks
var benchmarks []Benchmark
if err := db.DB.Find(&benchmarks).Error; err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch benchmarks"})
return
}

successCount := 0
errorCount := 0
var errors []string

for i := range benchmarks {
benchmark := &benchmarks[i]

// Read benchmark data
benchmarkData, err := RetrieveBenchmarkData(benchmark.ID)
if err != nil {
errorCount++
errors = append(errors, fmt.Sprintf("Benchmark %d: failed to read data - %v", benchmark.ID, err))
continue
}

// Extract searchable metadata
runNames, specifications := ExtractSearchableMetadata(benchmarkData)
benchmark.RunNames = runNames
benchmark.Specifications = specifications

// Update benchmark record
if err := db.DB.Save(benchmark).Error; err != nil {
errorCount++
errors = append(errors, fmt.Sprintf("Benchmark %d: failed to save - %v", benchmark.ID, err))
continue
}

successCount++
}

c.JSON(http.StatusOK, gin.H{
"total":        len(benchmarks),
"updated":      successCount,
"errors":       errorCount,
"error_details": errors,
})
}
}
