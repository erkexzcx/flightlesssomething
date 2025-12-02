package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBenchmarkTrimIntegration(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create test user
	user := &User{
		DiscordID: "test_user",
		Username:  "Test User",
	}
	if err := db.DB.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create test benchmark with data
	testData := []*BenchmarkData{
		{
			Label:              "Test Run 1",
			SpecOS:             "Test OS",
			SpecCPU:            "Test CPU",
			SpecGPU:            "Test GPU",
			DataFPS:            make([]float64, 100),
			DataFrameTime:      make([]float64, 100),
			DataCPULoad:        make([]float64, 100),
			DataGPULoad:        make([]float64, 100),
		},
	}

	// Fill with test values - simulate loading screen spike at the beginning
	for i := 0; i < 100; i++ {
		if i < 10 {
			// Loading screen with high FPS
			testData[0].DataFPS[i] = 200.0 + float64(i)
		} else {
			// Normal gameplay with lower FPS
			testData[0].DataFPS[i] = 60.0 + float64(i%10)
		}
		testData[0].DataFrameTime[i] = 1000.0 / testData[0].DataFPS[i]
		testData[0].DataCPULoad[i] = 50.0
		testData[0].DataGPULoad[i] = 80.0
	}

	benchmark := &Benchmark{
		UserID:      user.ID,
		Title:       "Test Benchmark",
		Description: "Test benchmark for trimming",
	}
	if err := db.DB.Create(benchmark).Error; err != nil {
		t.Fatalf("Failed to create benchmark: %v", err)
	}

	// Store benchmark data
	if err := StoreBenchmarkData(testData, benchmark.ID); err != nil {
		t.Fatalf("Failed to store benchmark data: %v", err)
	}

	// Setup routes
	router.GET("/api/benchmarks/:id/data", HandleGetBenchmarkData(db))
	router.PATCH("/api/benchmarks/:id", func(c *gin.Context) {
		// Mock authentication
		c.Set("UserID", user.ID)
		c.Set("IsAdmin", false)
		HandleUpdateBenchmark(db)(c)
	})

	t.Run("retrieve original data without trim", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/benchmarks/1/data", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var data []*BenchmarkData
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(data) != 1 {
			t.Fatalf("Expected 1 run, got %d", len(data))
		}

		if len(data[0].DataFPS) != 100 {
			t.Fatalf("Expected 100 FPS samples, got %d", len(data[0].DataFPS))
		}

		// Calculate average - should be skewed by loading screen
		var sum float64
		for _, fps := range data[0].DataFPS {
			sum += fps
		}
		avgBefore := sum / float64(len(data[0].DataFPS))
		t.Logf("Average FPS before trim: %.2f", avgBefore)

		// Should be higher than 60 due to loading screen spikes
		if avgBefore < 70 {
			t.Errorf("Expected average FPS > 70 (due to loading screen), got %.2f", avgBefore)
		}
	})

	t.Run("apply trim to exclude loading screen", func(t *testing.T) {
		// Update benchmark to trim first 10 samples (loading screen)
		updateReq := map[string]interface{}{
			"trims": map[int]map[string]int{
				0: {
					"trim_start": 10,
					"trim_end":   99,
				},
			},
		}

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest(http.MethodPatch, "/api/benchmarks/1", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("retrieve trimmed data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/benchmarks/1/data", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var data []*BenchmarkData
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(data) != 1 {
			t.Fatalf("Expected 1 run, got %d", len(data))
		}

		// Data should still have 100 samples (not trimmed in API response)
		if len(data[0].DataFPS) != 100 {
			t.Fatalf("Expected 100 FPS samples, got %d", len(data[0].DataFPS))
		}

		// But trim parameters should be set
		if data[0].TrimStart != 10 {
			t.Errorf("Expected TrimStart=10, got %d", data[0].TrimStart)
		}
		if data[0].TrimEnd != 99 {
			t.Errorf("Expected TrimEnd=99, got %d", data[0].TrimEnd)
		}

		// Apply trimming manually to verify
		trimmed := data[0].GetTrimmedData()
		if len(trimmed.DataFPS) != 90 {
			t.Fatalf("Expected 90 trimmed FPS samples, got %d", len(trimmed.DataFPS))
		}

		// Calculate average after trim
		var sum float64
		for _, fps := range trimmed.DataFPS {
			sum += fps
		}
		avgAfter := sum / float64(len(trimmed.DataFPS))
		t.Logf("Average FPS after trim: %.2f", avgAfter)

		// Should be much closer to 60-69 range now
		if avgAfter < 60 || avgAfter > 70 {
			t.Errorf("Expected average FPS between 60-70 (gameplay only), got %.2f", avgAfter)
		}
	})

	t.Run("reset trim to default", func(t *testing.T) {
		// Reset by sending both as 0
		updateReq := map[string]interface{}{
			"trims": map[int]map[string]int{
				0: {
					"trim_start": 0,
					"trim_end":   0,
				},
			},
		}

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest(http.MethodPatch, "/api/benchmarks/1", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify trim was reset
		req = httptest.NewRequest(http.MethodGet, "/api/benchmarks/1/data", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var data []*BenchmarkData
		json.Unmarshal(w.Body.Bytes(), &data)

		if data[0].TrimStart != 0 || data[0].TrimEnd != 0 {
			t.Errorf("Expected trim to be reset to 0,0, got %d,%d", data[0].TrimStart, data[0].TrimEnd)
		}
	})
}
