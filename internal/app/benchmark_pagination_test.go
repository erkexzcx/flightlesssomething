package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleGetBenchmarkData_Pagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Initialize benchmarks directory using the same temp dir
	tempDir := t.TempDir()
	if err := InitBenchmarksDir(tempDir); err != nil {
		t.Fatalf("Failed to initialize benchmarks directory: %v", err)
	}

	// Create a test user
	testUser := User{
		DiscordID: "test123",
		Username:  "testuser",
	}
	if err := db.DB.Create(&testUser).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create a test benchmark
	benchmark := Benchmark{
		UserID:      testUser.ID,
		Title:       "Test Benchmark",
		Description: "Test Description",
	}
	if err := db.DB.Create(&benchmark).Error; err != nil {
		t.Fatalf("Failed to create benchmark: %v", err)
	}

	// Create test data with multiple runs
	numRuns := 20
	testData := make([]*BenchmarkData, numRuns)
	for i := 0; i < numRuns; i++ {
		testData[i] = &BenchmarkData{
			Label:   "Run " + string(rune('A'+i)),
			SpecOS:  "TestOS",
			SpecCPU: "TestCPU",
			SpecGPU: "TestGPU",
			DataFPS: []float64{60.0, 61.0, 62.0},
		}
	}

	// Store benchmark data
	if err := StoreBenchmarkData(testData, benchmark.ID); err != nil {
		t.Fatalf("Failed to store benchmark data: %v", err)
	}

	t.Run("returns all runs without pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/benchmarks/1/data", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result []*BenchmarkData
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(result) != numRuns {
			t.Errorf("Expected %d runs, got %d", numRuns, len(result))
		}
	})

	t.Run("returns paginated runs with limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/benchmarks/1/data?run_offset=0&run_limit=10", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		runs, ok := result["runs"].([]interface{})
		if !ok {
			t.Fatalf("Expected 'runs' field in paginated response")
		}

		if len(runs) != 10 {
			t.Errorf("Expected 10 runs, got %d", len(runs))
		}

		totalRuns, ok := result["total_runs"].(float64)
		if !ok || int(totalRuns) != numRuns {
			t.Errorf("Expected total_runs=%d, got %v", numRuns, result["total_runs"])
		}

		offset, ok := result["offset"].(float64)
		if !ok || int(offset) != 0 {
			t.Errorf("Expected offset=0, got %v", result["offset"])
		}

		limit, ok := result["limit"].(float64)
		if !ok || int(limit) != 10 {
			t.Errorf("Expected limit=10, got %v", result["limit"])
		}
	})

	t.Run("returns correct runs with offset", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/benchmarks/1/data?run_offset=10&run_limit=5", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		runs, ok := result["runs"].([]interface{})
		if !ok {
			t.Fatalf("Expected 'runs' field in paginated response")
		}

		if len(runs) != 5 {
			t.Errorf("Expected 5 runs, got %d", len(runs))
		}

		offset, ok := result["offset"].(float64)
		if !ok || int(offset) != 10 {
			t.Errorf("Expected offset=10, got %v", result["offset"])
		}
	})

	t.Run("handles offset beyond total runs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/benchmarks/1/data?run_offset=100&run_limit=10", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		runs, ok := result["runs"].([]interface{})
		if !ok {
			t.Fatalf("Expected 'runs' field in paginated response")
		}

		if len(runs) != 0 {
			t.Errorf("Expected 0 runs for out-of-bounds offset, got %d", len(runs))
		}
	})

	t.Run("handles limit larger than remaining runs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/benchmarks/1/data?run_offset=15&run_limit=100", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		runs, ok := result["runs"].([]interface{})
		if !ok {
			t.Fatalf("Expected 'runs' field in paginated response")
		}

		// Should return only the remaining 5 runs (20 total - 15 offset)
		expectedRuns := numRuns - 15
		if len(runs) != expectedRuns {
			t.Errorf("Expected %d runs, got %d", expectedRuns, len(runs))
		}
	})

	t.Run("returns error for invalid offset", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/benchmarks/1/data?run_offset=invalid", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns error for negative offset", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/benchmarks/1/data?run_offset=-5", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns error for invalid limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/benchmarks/1/data?run_limit=invalid", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns error for zero limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/benchmarks/1/data?run_limit=0", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}
