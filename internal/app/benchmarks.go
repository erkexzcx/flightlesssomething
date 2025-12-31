package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// HandleListBenchmarks returns a list of benchmarks
func HandleListBenchmarks(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
		if err != nil || page < 1 {
			page = 1
		}
		perPage, err := strconv.Atoi(c.DefaultQuery("per_page", "10"))
		if err != nil || perPage < 1 || perPage > 100 {
			perPage = 10
		}

		var benchmarks []Benchmark
		query := db.DB.Preload("User")

		// Optional filters
		if userID := c.Query("user_id"); userID != "" {
			query = query.Where("user_id = ?", userID)
		}
		if search := c.Query("search"); search != "" {
			// Get search fields from query parameter (comma-separated)
			// Default to title,description to match frontend defaults
			searchFieldsParam := c.DefaultQuery("search_fields", "title,description")
			searchFields := strings.Split(searchFieldsParam, ",")
			
			// Build a map of enabled search fields - validate against allowlist
			validFields := map[string]bool{
				"title":          true,
				"description":    true,
				"user":           true,
				"run_name":       true,
				"specifications": true,
			}
			enabledFields := make(map[string]bool)
			for _, field := range searchFields {
				field = strings.TrimSpace(field)
				// Only add valid fields to prevent SQL injection
				if validFields[field] {
					enabledFields[field] = true
				}
			}
			
			// Split search query into keywords
			keywords := strings.Fields(search)
			
			// For each keyword, search in enabled fields
			// All keywords must match (AND logic), but each keyword can match any enabled field (OR logic)
			for _, keyword := range keywords {
				keyword = strings.TrimSpace(keyword)
				if keyword != "" {
					// Build OR conditions for this keyword across enabled fields
					var orConditions []string
					var orValues []interface{}
					
					if enabledFields["title"] {
						orConditions = append(orConditions, "benchmarks.title LIKE ?")
						orValues = append(orValues, "%"+keyword+"%")
					}
					if enabledFields["description"] {
						orConditions = append(orConditions, "benchmarks.description LIKE ?")
						orValues = append(orValues, "%"+keyword+"%")
					}
					if enabledFields["user"] {
						orConditions = append(orConditions, "EXISTS (SELECT 1 FROM users WHERE users.id = benchmarks.user_id AND users.username LIKE ?)")
						orValues = append(orValues, "%"+keyword+"%")
					}
					if enabledFields["run_name"] {
						orConditions = append(orConditions, "benchmarks.run_names LIKE ?")
						orValues = append(orValues, "%"+keyword+"%")
					}
					if enabledFields["specifications"] {
						orConditions = append(orConditions, "benchmarks.specifications LIKE ?")
						orValues = append(orValues, "%"+keyword+"%")
					}
					
					// Only apply search if at least one field is enabled
					if len(orConditions) > 0 {
						orClause := strings.Join(orConditions, " OR ")
						query = query.Where(orClause, orValues...)
					}
				}
			}
		}

		// Sorting
		sortBy := c.DefaultQuery("sort_by", "created_at")
		sortOrder := c.DefaultQuery("sort_order", "desc")

		// Validate sort_by to prevent SQL injection
		// Use explicit column names to avoid any possibility of injection
		var orderClause string
		switch sortBy {
		case "title":
			orderClause = "title"
		case "created_at":
			orderClause = "created_at"
		case "updated_at":
			orderClause = "updated_at"
		default:
			orderClause = "created_at"
		}

		// Validate sort_order - only allow specific values
		var ascending bool
		if sortOrder == "asc" {
			ascending = true
		} else {
			ascending = false
		}

		// Use GORM's safe order method with validated inputs
		if ascending {
			query = query.Order(orderClause + " ASC")
		} else {
			query = query.Order(orderClause + " DESC")
		}

		// Get total count
		var total int64
		if err := query.Model(&Benchmark{}).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		// Get paginated results
		offset := (page - 1) * perPage
		if err := query.Offset(offset).Limit(perPage).Find(&benchmarks).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		// Populate run count and labels for each benchmark concurrently
		// Thread safety: Each goroutine writes to a different index in the slice.
		// In Go, writing to different indices of a slice is safe without synchronization.
		var wg sync.WaitGroup
		for i := range benchmarks {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				count, labels, err := GetBenchmarkRunCount(benchmarks[idx].ID)
				if err == nil {
					benchmarks[idx].RunCount = count
					benchmarks[idx].RunLabels = labels
				}
			}(i)
		}
		wg.Wait()

		// Calculate total pages
		totalPages := int((total + int64(perPage) - 1) / int64(perPage))

		c.JSON(http.StatusOK, gin.H{
			"benchmarks":  benchmarks,
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
		})
	}
}

// HandleGetBenchmark returns a single benchmark
func HandleGetBenchmark(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var benchmark Benchmark
		if err := db.DB.Preload("User").First(&benchmark, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "benchmark not found"})
			return
		}

		c.JSON(http.StatusOK, benchmark)
	}
}

// HandleGetBenchmarkData returns the data for a benchmark
func HandleGetBenchmarkData(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		benchmarkID, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid benchmark ID"})
			return
		}

		// Verify benchmark exists
		var benchmark Benchmark
		if dbErr := db.DB.First(&benchmark, benchmarkID).Error; dbErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "benchmark not found"})
			return
		}

		// Check if data size exceeds the limit for browser loading
		if benchmark.DataSizeBytes > maxDataSizeBytes {
			sizeMB := float64(benchmark.DataSizeBytes) / (1024 * 1024)
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("Benchmark data is too large to load in browser (%.1f MB). Please use the download button to get the data as a ZIP file instead.", sizeMB),
				"data_size_bytes": benchmark.DataSizeBytes,
			})
			return
		}

		data, err := RetrieveBenchmarkData(uint(benchmarkID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve benchmark data"})
			return
		}

		c.JSON(http.StatusOK, data)
	}
}

// HandleCreateBenchmark creates a new benchmark
func HandleCreateBenchmark(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("UserID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID type"})
			return
		}

		// Check if user is banned
		var user User
		if err := db.DB.First(&user, uid).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
			return
		}
		if user.IsBanned {
			c.JSON(http.StatusForbidden, gin.H{"error": "your account has been banned"})
			return
		}

		// Check rate limiting for benchmark uploads (skip for admins)
		if !user.IsAdmin {
			limiter := GetBenchmarkUploadLimiter()
			userKey := fmt.Sprintf("user_%d", uid)
			if !limiter.Allow(userKey) {
				remaining := limiter.GetRemainingTime(userKey)
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":            "rate limit exceeded: maximum 5 benchmarks per 10 minutes",
					"retry_after_secs": int(remaining.Seconds()),
				})
				return
			}
		}

		var req struct {
			Title       string `form:"title" binding:"required,max=100"`
			Description string `form:"description" binding:"max=5000"`
		}

		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}

		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no files uploaded"})
			return
		}

		files := form.File["files"]
		if len(files) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no files uploaded"})
			return
		}

		benchmarkData, err := ReadBenchmarkFiles(files)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse files: " + err.Error()})
			return
		}

		// Create benchmark record
		benchmark := Benchmark{
			UserID:      uid,
			Title:       req.Title,
			Description: req.Description,
		}

		if err := db.DB.Create(&benchmark).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create benchmark"})
			return
		}

		// Store benchmark data
		dataSizeBytes, err := StoreBenchmarkData(benchmarkData, benchmark.ID)
		if err != nil {
			db.DB.Delete(&benchmark)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store benchmark data"})
			return
		}

		// Extract and update searchable metadata
		runNames, specifications := ExtractSearchableMetadata(benchmarkData)
		benchmark.RunNames = runNames
		benchmark.Specifications = specifications
		benchmark.DataSizeBytes = dataSizeBytes
		if err := db.DB.Save(&benchmark).Error; err != nil {
			// Log error but don't fail - this is just for search optimization
			fmt.Printf("Warning: failed to update searchable metadata for benchmark %d (%s): %v\n", benchmark.ID, benchmark.Title, err)
		}

		// Reload benchmark with User to return complete data
		if err := db.DB.Preload("User").First(&benchmark, benchmark.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load benchmark"})
			return
		}

		// Log benchmark creation
		LogBenchmarkCreated(db, uid, benchmark.ID, benchmark.Title)

		c.JSON(http.StatusCreated, benchmark)
	}
}

// HandleUpdateBenchmark updates an existing benchmark
func HandleUpdateBenchmark(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID, _ := c.Get("UserID")
		isAdmin, _ := c.Get("IsAdmin")

		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID type"})
			return
		}

		adminFlag := false
		if isAdmin != nil {
			if af, ok := isAdmin.(bool); ok {
				adminFlag = af
			}
		}

		// Check if user is banned (admins can still update)
		if !adminFlag {
			var user User
			if err := db.DB.First(&user, uid).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
				return
			}
			if user.IsBanned {
				c.JSON(http.StatusForbidden, gin.H{"error": "your account has been banned"})
				return
			}
		}

		var benchmark Benchmark
		if err := db.DB.First(&benchmark, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "benchmark not found"})
			return
		}

		// Check ownership or admin
		if benchmark.UserID != uid && !adminFlag {
			c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
			return
		}

		var req struct {
			Title       string         `json:"title" binding:"max=100"`
			Description string         `json:"description" binding:"max=5000"`
			Labels      map[int]string `json:"labels"` // Map of index to new label
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		if req.Title != "" {
			benchmark.Title = req.Title
		}
		if req.Description != "" {
			benchmark.Description = req.Description
		}

		// Update labels if provided
		if len(req.Labels) > 0 {
			benchmarkID, err := strconv.ParseUint(id, 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid benchmark ID"})
				return
			}
			benchmarkData, err := RetrieveBenchmarkData(uint(benchmarkID))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve benchmark data"})
				return
			}

			// Update labels
			for idx, newLabel := range req.Labels {
				if idx >= 0 && idx < len(benchmarkData) {
					benchmarkData[idx].Label = newLabel
				}
			}

			// Store updated data
			dataSizeBytes, err := StoreBenchmarkData(benchmarkData, uint(benchmarkID))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update labels"})
				return
			}

			// Update searchable metadata after label changes
			runNames, specifications := ExtractSearchableMetadata(benchmarkData)
			benchmark.RunNames = runNames
			benchmark.Specifications = specifications
			benchmark.DataSizeBytes = dataSizeBytes
		}

		if err := db.DB.Save(&benchmark).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update benchmark"})
			return
		}

		// Reload benchmark with User to return complete data
		if err := db.DB.Preload("User").First(&benchmark, benchmark.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load benchmark"})
			return
		}

		// Log benchmark update
		LogBenchmarkUpdated(db, uid, benchmark.ID, benchmark.Title)

		c.JSON(http.StatusOK, benchmark)
	}
}

// HandleDeleteBenchmark deletes a benchmark
func HandleDeleteBenchmark(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID, _ := c.Get("UserID")
		isAdmin, _ := c.Get("IsAdmin")

		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID type"})
			return
		}

		adminFlag := false
		if isAdmin != nil {
			if af, ok := isAdmin.(bool); ok {
				adminFlag = af
			}
		}

		// Check if user is banned (admins can still delete)
		if !adminFlag {
			var user User
			if err := db.DB.First(&user, uid).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
				return
			}
			if user.IsBanned {
				c.JSON(http.StatusForbidden, gin.H{"error": "your account has been banned"})
				return
			}
		}

		var benchmark Benchmark
		if err := db.DB.First(&benchmark, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "benchmark not found"})
			return
		}

		// Check ownership or admin
		if benchmark.UserID != uid && !adminFlag {
			c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
			return
		}

		// Store title for audit log
		title := benchmark.Title

		// Delete data file
		if err := DeleteBenchmarkData(benchmark.ID); err != nil {
			// Log error but continue with database deletion
			fmt.Printf("Warning: failed to delete benchmark data file: %v\n", err)
		}

		if err := db.DB.Delete(&benchmark).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete benchmark"})
			return
		}

		// Log benchmark deletion
		LogBenchmarkDeleted(db, uid, benchmark.ID, title)

		c.JSON(http.StatusOK, gin.H{"message": "benchmark deleted"})
	}
}

// HandleDownloadBenchmarkData downloads benchmark data as a ZIP file containing CSV files
func HandleDownloadBenchmarkData(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		benchmarkID, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid benchmark ID"})
			return
		}

		// Verify benchmark exists
		var benchmark Benchmark
		if err := db.DB.First(&benchmark, benchmarkID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "benchmark not found"})
			return
		}

		// Set headers for ZIP download
		c.Header("Content-Type", "application/zip")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"benchmark_%d.zip\"", benchmarkID))

		// Export data as ZIP
		if err := ExportBenchmarkDataAsZip(uint(benchmarkID), c.Writer); err != nil {
			// If we've already started writing, we can't change the status code
			// Log the error instead
			fmt.Printf("Error exporting benchmark data: %v\n", err)
			return
		}
	}
}

// HandleDeleteBenchmarkRun deletes a specific run from a benchmark
func HandleDeleteBenchmarkRun(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		runIndex := c.Param("run_index")
		userID, _ := c.Get("UserID")
		isAdmin, _ := c.Get("IsAdmin")

		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID type"})
			return
		}

		adminFlag := false
		if isAdmin != nil {
			if af, ok := isAdmin.(bool); ok {
				adminFlag = af
			}
		}

		// Check if user is banned (admins can still delete)
		if !adminFlag {
			var user User
			if err := db.DB.First(&user, uid).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
				return
			}
			if user.IsBanned {
				c.JSON(http.StatusForbidden, gin.H{"error": "your account has been banned"})
				return
			}
		}

		// Parse benchmark ID
		benchmarkID, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid benchmark ID"})
			return
		}

		// Parse run index
		idx, err := strconv.Atoi(runIndex)
		if err != nil || idx < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run index"})
			return
		}

		// Verify benchmark exists and check ownership
		var benchmark Benchmark
		if dbErr := db.DB.First(&benchmark, benchmarkID).Error; dbErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "benchmark not found"})
			return
		}

		// Check ownership or admin
		if benchmark.UserID != uid && !adminFlag {
			c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
			return
		}

		// Retrieve benchmark data
		benchmarkData, err := RetrieveBenchmarkData(uint(benchmarkID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve benchmark data"})
			return
		}

		// Validate run index
		if idx >= len(benchmarkData) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "run index out of range"})
			return
		}

		// Cannot delete the last run
		if len(benchmarkData) == 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete the last run - delete the entire benchmark instead"})
			return
		}

		// Remove the run at the specified index
		benchmarkData = append(benchmarkData[:idx], benchmarkData[idx+1:]...)

		// Store updated data
		dataSizeBytes, err := StoreBenchmarkData(benchmarkData, uint(benchmarkID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update benchmark data"})
			return
		}

		// Update searchable metadata after deleting run
		runNames, specifications := ExtractSearchableMetadata(benchmarkData)
		benchmark.RunNames = runNames
		benchmark.Specifications = specifications
		benchmark.DataSizeBytes = dataSizeBytes

		// Update the benchmark's UpdatedAt timestamp
		if err := db.DB.Save(&benchmark).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update benchmark"})
			return
		}

		// Log benchmark update
		LogBenchmarkUpdated(db, uid, benchmark.ID, benchmark.Title)

		c.JSON(http.StatusOK, gin.H{"message": "run deleted successfully"})
	}
}

// HandleAddBenchmarkRuns adds new runs to an existing benchmark
func HandleAddBenchmarkRuns(db *DBInstance) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID, _ := c.Get("UserID")
		isAdmin, _ := c.Get("IsAdmin")

		uid, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID type"})
			return
		}

		adminFlag := false
		if isAdmin != nil {
			if af, ok := isAdmin.(bool); ok {
				adminFlag = af
			}
		}

		// Check if user is banned (admins can still add)
		if !adminFlag {
			var user User
			if err := db.DB.First(&user, uid).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
				return
			}
			if user.IsBanned {
				c.JSON(http.StatusForbidden, gin.H{"error": "your account has been banned"})
				return
			}
		}

		// Parse benchmark ID
		benchmarkID, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid benchmark ID"})
			return
		}

		// Verify benchmark exists and check ownership
		var benchmark Benchmark
		if dbErr := db.DB.First(&benchmark, benchmarkID).Error; dbErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "benchmark not found"})
			return
		}

		// Check ownership or admin
		if benchmark.UserID != uid && !adminFlag {
			c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
			return
		}

		// Get uploaded files
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no files uploaded"})
			return
		}

		files := form.File["files"]
		if len(files) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no files uploaded"})
			return
		}

		// Parse new benchmark files
		newBenchmarkData, err := ReadBenchmarkFiles(files)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse files: " + err.Error()})
			return
		}

		// Retrieve existing benchmark data
		existingData, err := RetrieveBenchmarkData(uint(benchmarkID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve existing benchmark data"})
			return
		}

		// Append new runs to existing data
		existingData = append(existingData, newBenchmarkData...)

		// Store combined data
		dataSizeBytes, err := StoreBenchmarkData(existingData, uint(benchmarkID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store benchmark data"})
			return
		}

		// Update searchable metadata after adding runs
		runNames, specifications := ExtractSearchableMetadata(existingData)
		benchmark.RunNames = runNames
		benchmark.Specifications = specifications
		benchmark.DataSizeBytes = dataSizeBytes

		// Update the benchmark's UpdatedAt timestamp
		if err := db.DB.Save(&benchmark).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update benchmark"})
			return
		}

		// Log benchmark update
		LogBenchmarkUpdated(db, uid, benchmark.ID, benchmark.Title)

		c.JSON(http.StatusOK, gin.H{
			"message":         "runs added successfully",
			"runs_added":      len(newBenchmarkData),
			"total_run_count": len(existingData),
		})
	}
}
