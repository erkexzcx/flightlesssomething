package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

// createTestUser creates a test user in the database
func createTestUser(db *DBInstance, username string, isAdmin bool) *User {
	user := &User{
		Username:  username,
		DiscordID: "test-" + username,
		IsAdmin:   isAdmin,
	}
	db.DB.Create(user)
	return user
}

func TestHandleListBenchmarks(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	router := setupTestRouter()
	router.GET("/api/benchmarks", HandleListBenchmarks(db))

	t.Run("returns empty list when no benchmarks", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/benchmarks", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		benchmarks, ok := response["benchmarks"].([]interface{})
		if !ok {
			t.Fatal("Expected benchmarks array in response")
		}
		if len(benchmarks) != 0 {
			t.Errorf("Expected empty benchmarks list, got %d items", len(benchmarks))
		}
	})

	t.Run("returns benchmarks when they exist", func(t *testing.T) {
		user := createTestUser(db, "testuser", false)
		benchmark := &Benchmark{
			Title:       "Test Benchmark",
			Description: "Test Description",
			UserID:      user.ID,
		}
		db.DB.Create(benchmark)

		req, err := http.NewRequest("GET", "/api/benchmarks", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		benchmarks, ok := response["benchmarks"].([]interface{})
		if !ok {
			t.Fatal("Expected benchmarks array in response")
		}
		if len(benchmarks) != 1 {
			t.Errorf("Expected 1 benchmark, got %d", len(benchmarks))
		}

		firstBenchmark, ok := benchmarks[0].(map[string]interface{})
		if !ok {
			t.Fatal("Expected benchmark to be map")
		}
		if firstBenchmark["Title"] != "Test Benchmark" {
			t.Errorf("Expected 'Test Benchmark', got %v", firstBenchmark["Title"])
		}
	})

	t.Run("pagination works correctly", func(t *testing.T) {
		// Create multiple benchmarks
		user := createTestUser(db, "paguser", false)
		for i := 1; i <= 15; i++ {
			benchmark := &Benchmark{
				Title:       "Benchmark " + strconv.Itoa(i),
				Description: "Description",
				UserID:      user.ID,
			}
			db.DB.Create(benchmark)
		}

		// Test first page
		req, err := http.NewRequest("GET", "/api/benchmarks?page=1&per_page=10", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response map[string]interface{}
		if unmarshalErr := json.Unmarshal(w.Body.Bytes(), &response); unmarshalErr != nil {
			t.Fatalf("Failed to unmarshal response: %v", unmarshalErr)
		}

		benchmarks, ok := response["benchmarks"].([]interface{})
		if !ok {
			t.Fatalf("Failed to parse benchmarks from response")
		}
		if len(benchmarks) != 10 {
			t.Errorf("Expected 10 benchmarks on first page, got %d", len(benchmarks))
		}

		// Test second page
		req, err = http.NewRequest("GET", "/api/benchmarks?page=2&per_page=10", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		benchmarks, ok = response["benchmarks"].([]interface{})
		if !ok {
			t.Fatalf("Failed to parse benchmarks from response")
		}
		if len(benchmarks) < 5 {
			t.Errorf("Expected at least 5 benchmarks on second page, got %d", len(benchmarks))
		}
	})
}

func TestHandleGetBenchmark(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	router := setupTestRouter()
	router.GET("/api/benchmarks/:id", HandleGetBenchmark(db))

	user := createTestUser(db, "getuser", false)
	benchmark := &Benchmark{
		Title:       "Get Test Benchmark",
		Description: "Get Test Description",
		UserID:      user.ID,
	}
	db.DB.Create(benchmark)

	t.Run("returns benchmark by id", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/benchmarks/"+strconv.FormatUint(uint64(benchmark.ID), 10), nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response Benchmark
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Title != "Get Test Benchmark" {
			t.Errorf("Expected 'Get Test Benchmark', got %s", response.Title)
		}
		if response.Description != "Get Test Description" {
			t.Errorf("Expected 'Get Test Description', got %s", response.Description)
		}
	})

	t.Run("returns 404 for non-existent benchmark", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/benchmarks/99999", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("returns 404 for invalid id", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/benchmarks/invalid", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

func TestHandleDeleteBenchmarkRun(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Initialize benchmarks directory for tests
	err := InitBenchmarksDir(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to initialize benchmarks directory: %v", err)
	}

	t.Run("deletes a run successfully", func(t *testing.T) {
		user := createTestUser(db, "testuser", false)
		benchmark := &Benchmark{
			Title:       "Test Benchmark",
			Description: "Test Description",
			UserID:      user.ID,
		}
		db.DB.Create(benchmark)

		// Create test data with multiple runs
		testData := []*BenchmarkData{
			{
				Label:   "Run 1",
				DataFPS: []float64{60.0, 61.0},
			},
			{
				Label:   "Run 2",
				DataFPS: []float64{70.0, 71.0},
			},
			{
				Label:   "Run 3",
				DataFPS: []float64{80.0, 81.0},
			},
		}
		err := StoreBenchmarkData(testData, benchmark.ID)
		if err != nil {
			t.Fatalf("Failed to store test data: %v", err)
		}

		router := setupTestRouter()
		router.DELETE("/api/benchmarks/:id/runs/:run_index", func(c *gin.Context) {
			c.Set("UserID", user.ID)
			c.Set("IsAdmin", false)
			HandleDeleteBenchmarkRun(db)(c)
		})

		// Delete run at index 1 (middle run)
		req, err := http.NewRequest("DELETE", "/api/benchmarks/"+strconv.Itoa(int(benchmark.ID))+"/runs/1", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify the run was deleted
		remainingData, err := RetrieveBenchmarkData(benchmark.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve data: %v", err)
		}
		if len(remainingData) != 2 {
			t.Errorf("Expected 2 runs, got %d", len(remainingData))
		}
		if remainingData[0].Label != "Run 1" || remainingData[1].Label != "Run 3" {
			t.Errorf("Unexpected remaining runs")
		}
	})

	t.Run("cannot delete last run", func(t *testing.T) {
		user := createTestUser(db, "testuser2", false)
		benchmark := &Benchmark{
			Title:       "Test Benchmark 2",
			Description: "Test Description",
			UserID:      user.ID,
		}
		db.DB.Create(benchmark)

		// Create test data with single run
		testData := []*BenchmarkData{
			{
				Label:   "Run 1",
				DataFPS: []float64{60.0, 61.0},
			},
		}
		err := StoreBenchmarkData(testData, benchmark.ID)
		if err != nil {
			t.Fatalf("Failed to store test data: %v", err)
		}

		router := setupTestRouter()
		router.DELETE("/api/benchmarks/:id/runs/:run_index", func(c *gin.Context) {
			c.Set("UserID", user.ID)
			c.Set("IsAdmin", false)
			HandleDeleteBenchmarkRun(db)(c)
		})

		req, err := http.NewRequest("DELETE", "/api/benchmarks/"+strconv.Itoa(int(benchmark.ID))+"/runs/0", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns 404 for non-existent benchmark", func(t *testing.T) {
		user := createTestUser(db, "testuser3", false)

		router := setupTestRouter()
		router.DELETE("/api/benchmarks/:id/runs/:run_index", func(c *gin.Context) {
			c.Set("UserID", user.ID)
			c.Set("IsAdmin", false)
			HandleDeleteBenchmarkRun(db)(c)
		})

		req, err := http.NewRequest("DELETE", "/api/benchmarks/99999/runs/0", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("returns 400 for invalid run index", func(t *testing.T) {
		user := createTestUser(db, "testuser4", false)
		benchmark := &Benchmark{
			Title:       "Test Benchmark 3",
			Description: "Test Description",
			UserID:      user.ID,
		}
		db.DB.Create(benchmark)

		testData := []*BenchmarkData{
			{
				Label:   "Run 1",
				DataFPS: []float64{60.0},
			},
		}
		err := StoreBenchmarkData(testData, benchmark.ID)
		if err != nil {
			t.Fatalf("Failed to store test data: %v", err)
		}

		router := setupTestRouter()
		router.DELETE("/api/benchmarks/:id/runs/:run_index", func(c *gin.Context) {
			c.Set("UserID", user.ID)
			c.Set("IsAdmin", false)
			HandleDeleteBenchmarkRun(db)(c)
		})

		req, err := http.NewRequest("DELETE", "/api/benchmarks/"+strconv.Itoa(int(benchmark.ID))+"/runs/10", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}
