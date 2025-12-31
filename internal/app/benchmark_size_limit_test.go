package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleGetBenchmarkData_SizeLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create a temporary directory for benchmark data
	tmpDir := t.TempDir()
	if err := InitBenchmarksDir(tmpDir); err != nil {
		t.Fatalf("Failed to initialize benchmarks directory: %v", err)
	}

	// Create a test user
	user := User{
		Username:  "testuser",
		DiscordID: "123456",
	}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create a small benchmark (should work)
	t.Run("small_benchmark_allowed", func(t *testing.T) {
		smallBenchmark := Benchmark{
			UserID:        user.ID,
			Title:         "Small Benchmark",
			Description:   "Should load fine",
			DataSizeBytes: 1024 * 1024, // 1MB - well under limit
		}
		if err := db.DB.Create(&smallBenchmark).Error; err != nil {
			t.Fatalf("Failed to create small benchmark: %v", err)
		}

		// Store small test data
		testData := []*BenchmarkData{
			{
				Label:   "Small Run",
				DataFPS: []float64{60.0, 61.0, 62.0},
			},
		}
		if _, err := StoreBenchmarkData(testData, smallBenchmark.ID); err != nil {
			t.Fatalf("Failed to store small benchmark data: %v", err)
		}

		router := setupTestRouter()
		router.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))

		req, err := http.NewRequest("GET", "/api/benchmarks/"+fmt.Sprint(smallBenchmark.ID)+"/data", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var data []*BenchmarkData
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(data) != 1 {
			t.Errorf("Expected 1 run, got %d", len(data))
		}
	})

	// Create a large benchmark (should be blocked)
	t.Run("large_benchmark_blocked", func(t *testing.T) {
		largeBenchmark := Benchmark{
			UserID:        user.ID,
			Title:         "Large Benchmark",
			Description:   "Should be too large",
			DataSizeBytes: 100 * 1024 * 1024, // 100MB - over the 80MB limit
		}
		if err := db.DB.Create(&largeBenchmark).Error; err != nil {
			t.Fatalf("Failed to create large benchmark: %v", err)
		}

		// Store some test data (size doesn't matter, we're testing the metadata check)
		testData := []*BenchmarkData{
			{
				Label:   "Large Run",
				DataFPS: []float64{60.0, 61.0, 62.0},
			},
		}
		if _, err := StoreBenchmarkData(testData, largeBenchmark.ID); err != nil {
			t.Fatalf("Failed to store large benchmark data: %v", err)
		}

		router := setupTestRouter()
		router.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))

		req, err := http.NewRequest("GET", "/api/benchmarks/"+fmt.Sprint(largeBenchmark.ID)+"/data", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413 (Request Entity Too Large), got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		errorMsg, ok := response["error"].(string)
		if !ok {
			t.Errorf("Expected error message in response")
		}

		// Check that error message mentions downloading
		if errorMsg == "" || len(errorMsg) < 10 {
			t.Errorf("Expected helpful error message, got: %s", errorMsg)
		}

		// Check that data_size_bytes is included
		if dataSizeBytes, ok := response["data_size_bytes"].(float64); !ok || dataSizeBytes != 100*1024*1024 {
			t.Errorf("Expected data_size_bytes to be 100MB, got %v", response["data_size_bytes"])
		}
	})

	// Create a benchmark right at the limit (should work)
	t.Run("at_limit_benchmark_allowed", func(t *testing.T) {
		atLimitBenchmark := Benchmark{
			UserID:        user.ID,
			Title:         "At Limit Benchmark",
			Description:   "Exactly at the limit",
			DataSizeBytes: maxDataSizeBytes, // Exactly 80MB
		}
		if err := db.DB.Create(&atLimitBenchmark).Error; err != nil {
			t.Fatalf("Failed to create at-limit benchmark: %v", err)
		}

		// Store test data
		testData := []*BenchmarkData{
			{
				Label:   "At Limit Run",
				DataFPS: []float64{60.0, 61.0, 62.0},
			},
		}
		if _, err := StoreBenchmarkData(testData, atLimitBenchmark.ID); err != nil {
			t.Fatalf("Failed to store at-limit benchmark data: %v", err)
		}

		router := setupTestRouter()
		router.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))

		req, err := http.NewRequest("GET", "/api/benchmarks/"+fmt.Sprint(atLimitBenchmark.ID)+"/data", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for benchmark at limit, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// Create a benchmark just over the limit (should be blocked)
	t.Run("just_over_limit_benchmark_blocked", func(t *testing.T) {
		overLimitBenchmark := Benchmark{
			UserID:        user.ID,
			Title:         "Just Over Limit Benchmark",
			Description:   "1 byte over the limit",
			DataSizeBytes: maxDataSizeBytes + 1, // 80MB + 1 byte
		}
		if err := db.DB.Create(&overLimitBenchmark).Error; err != nil {
			t.Fatalf("Failed to create over-limit benchmark: %v", err)
		}

		// Store test data
		testData := []*BenchmarkData{
			{
				Label:   "Over Limit Run",
				DataFPS: []float64{60.0, 61.0, 62.0},
			},
		}
		if _, err := StoreBenchmarkData(testData, overLimitBenchmark.ID); err != nil {
			t.Fatalf("Failed to store over-limit benchmark data: %v", err)
		}

		router := setupTestRouter()
		router.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))

		req, err := http.NewRequest("GET", "/api/benchmarks/"+fmt.Sprint(overLimitBenchmark.ID)+"/data", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413 for benchmark 1 byte over limit, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// Test backward compatibility: benchmark with DataSizeBytes = 0 (legacy)
	t.Run("legacy_benchmark_with_zero_size", func(t *testing.T) {
		legacyBenchmark := Benchmark{
			UserID:        user.ID,
			Title:         "Legacy Benchmark",
			Description:   "Legacy benchmark with DataSizeBytes = 0",
			DataSizeBytes: 0, // Not yet calculated
		}
		if err := db.DB.Create(&legacyBenchmark).Error; err != nil {
			t.Fatalf("Failed to create legacy benchmark: %v", err)
		}

		// Store small test data
		testData := []*BenchmarkData{
			{
				Label:   "Legacy Run",
				DataFPS: []float64{60.0, 61.0, 62.0},
			},
		}
		if _, err := StoreBenchmarkData(testData, legacyBenchmark.ID); err != nil {
			t.Fatalf("Failed to store legacy benchmark data: %v", err)
		}

		// Manually set DataSizeBytes back to 0 to simulate legacy data
		legacyBenchmark.DataSizeBytes = 0
		if err := db.DB.Save(&legacyBenchmark).Error; err != nil {
			t.Fatalf("Failed to reset DataSizeBytes: %v", err)
		}

		router := setupTestRouter()
		router.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))

		req, err := http.NewRequest("GET", "/api/benchmarks/"+fmt.Sprint(legacyBenchmark.ID)+"/data", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still work - size is calculated on-the-fly
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for legacy benchmark, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Verify data was returned
		var data []*BenchmarkData
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(data) != 1 {
			t.Errorf("Expected 1 run, got %d", len(data))
		}

		// Verify that DataSizeBytes was updated in the database
		var updatedBenchmark Benchmark
		if err := db.DB.First(&updatedBenchmark, legacyBenchmark.ID).Error; err != nil {
			t.Fatalf("Failed to fetch updated benchmark: %v", err)
		}

		if updatedBenchmark.DataSizeBytes == 0 {
			t.Error("Expected DataSizeBytes to be calculated and saved, but it's still 0")
		}
	})
}
