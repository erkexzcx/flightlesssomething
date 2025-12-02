package app

import (
	"testing"
)

func TestBenchmarkDataTrimming(t *testing.T) {
	// Create test data with 100 samples
	testData := &BenchmarkData{
		Label:              "Test Run",
		DataFPS:            make([]float64, 100),
		DataFrameTime:      make([]float64, 100),
		DataCPULoad:        make([]float64, 100),
		DataGPULoad:        make([]float64, 100),
		DataCPUTemp:        make([]float64, 100),
		DataCPUPower:       make([]float64, 100),
		DataGPUTemp:        make([]float64, 100),
		DataGPUCoreClock:   make([]float64, 100),
		DataGPUMemClock:    make([]float64, 100),
		DataGPUVRAMUsed:    make([]float64, 100),
		DataGPUPower:       make([]float64, 100),
		DataRAMUsed:        make([]float64, 100),
		DataSwapUsed:       make([]float64, 100),
	}

	// Fill with test values
	for i := 0; i < 100; i++ {
		testData.DataFPS[i] = float64(i)
		testData.DataFrameTime[i] = float64(i) * 0.5
		testData.DataCPULoad[i] = float64(i) * 0.3
		testData.DataGPULoad[i] = float64(i) * 0.4
	}

	t.Run("no trimming when TrimStart and TrimEnd are both 0", func(t *testing.T) {
		testData.TrimStart = 0
		testData.TrimEnd = 0

		trimmed := testData.GetTrimmedData()
		
		if len(trimmed.DataFPS) != 100 {
			t.Errorf("Expected DataFPS length 100, got %d", len(trimmed.DataFPS))
		}
		if len(trimmed.DataFrameTime) != 100 {
			t.Errorf("Expected DataFrameTime length 100, got %d", len(trimmed.DataFrameTime))
		}
	})

	t.Run("trims data correctly with valid range", func(t *testing.T) {
		testData.TrimStart = 10
		testData.TrimEnd = 89

		trimmed := testData.GetTrimmedData()
		
		// Should have 80 samples (from index 10 to 89 inclusive)
		if len(trimmed.DataFPS) != 80 {
			t.Errorf("Expected DataFPS length 80, got %d", len(trimmed.DataFPS))
		}
		
		// Check first value is correct (should be index 10)
		if trimmed.DataFPS[0] != 10.0 {
			t.Errorf("Expected first FPS value 10.0, got %f", trimmed.DataFPS[0])
		}
		
		// Check last value is correct (should be index 89)
		if trimmed.DataFPS[79] != 89.0 {
			t.Errorf("Expected last FPS value 89.0, got %f", trimmed.DataFPS[79])
		}

		// Check other arrays are trimmed too
		if len(trimmed.DataFrameTime) != 80 {
			t.Errorf("Expected DataFrameTime length 80, got %d", len(trimmed.DataFrameTime))
		}
		if len(trimmed.DataCPULoad) != 80 {
			t.Errorf("Expected DataCPULoad length 80, got %d", len(trimmed.DataCPULoad))
		}
	})

	t.Run("handles edge case when TrimStart equals TrimEnd", func(t *testing.T) {
		testData.TrimStart = 50
		testData.TrimEnd = 50

		trimmed := testData.GetTrimmedData()
		
		// Should have 1 sample
		if len(trimmed.DataFPS) != 1 {
			t.Errorf("Expected DataFPS length 1, got %d", len(trimmed.DataFPS))
		}
		if trimmed.DataFPS[0] != 50.0 {
			t.Errorf("Expected FPS value 50.0, got %f", trimmed.DataFPS[0])
		}
	})

	t.Run("handles TrimEnd beyond array length", func(t *testing.T) {
		testData.TrimStart = 80
		testData.TrimEnd = 150 // Beyond array length

		trimmed := testData.GetTrimmedData()
		
		// Should trim to the end of array (80 to 99 = 20 samples)
		if len(trimmed.DataFPS) != 20 {
			t.Errorf("Expected DataFPS length 20, got %d", len(trimmed.DataFPS))
		}
		if trimmed.DataFPS[0] != 80.0 {
			t.Errorf("Expected first FPS value 80.0, got %f", trimmed.DataFPS[0])
		}
		if trimmed.DataFPS[19] != 99.0 {
			t.Errorf("Expected last FPS value 99.0, got %f", trimmed.DataFPS[19])
		}
	})

	t.Run("handles TrimEnd of 0 as full length", func(t *testing.T) {
		testData.TrimStart = 50
		testData.TrimEnd = 0 // Should be treated as array length - 1

		trimmed := testData.GetTrimmedData()
		
		// Should trim from 50 to end (50 samples)
		if len(trimmed.DataFPS) != 50 {
			t.Errorf("Expected DataFPS length 50, got %d", len(trimmed.DataFPS))
		}
		if trimmed.DataFPS[0] != 50.0 {
			t.Errorf("Expected first FPS value 50.0, got %f", trimmed.DataFPS[0])
		}
	})

	t.Run("handles negative TrimStart", func(t *testing.T) {
		testData.TrimStart = -10 // Should be clamped to 0
		testData.TrimEnd = 20

		trimmed := testData.GetTrimmedData()
		
		// Should trim from 0 to 20 (21 samples)
		if len(trimmed.DataFPS) != 21 {
			t.Errorf("Expected DataFPS length 21, got %d", len(trimmed.DataFPS))
		}
		if trimmed.DataFPS[0] != 0.0 {
			t.Errorf("Expected first FPS value 0.0, got %f", trimmed.DataFPS[0])
		}
	})

	t.Run("handles invalid range (TrimStart > TrimEnd)", func(t *testing.T) {
		testData.TrimStart = 80
		testData.TrimEnd = 20 // Invalid: start > end

		trimmed := testData.GetTrimmedData()
		
		// Should return empty array
		if len(trimmed.DataFPS) != 0 {
			t.Errorf("Expected DataFPS length 0 for invalid range, got %d", len(trimmed.DataFPS))
		}
	})

	t.Run("GetDataLength returns correct length", func(t *testing.T) {
		length := testData.GetDataLength()
		if length != 100 {
			t.Errorf("Expected data length 100, got %d", length)
		}
	})

	t.Run("GetDataLength with empty arrays", func(t *testing.T) {
		emptyData := &BenchmarkData{
			Label: "Empty Run",
		}
		length := emptyData.GetDataLength()
		if length != 0 {
			t.Errorf("Expected data length 0 for empty data, got %d", length)
		}
	})

	t.Run("GetDataLength with arrays of different lengths", func(t *testing.T) {
		mixedData := &BenchmarkData{
			Label:         "Mixed Run",
			DataFPS:       make([]float64, 50),
			DataFrameTime: make([]float64, 100),
			DataCPULoad:   make([]float64, 75),
		}
		length := mixedData.GetDataLength()
		if length != 100 {
			t.Errorf("Expected data length 100 (max of all arrays), got %d", length)
		}
	})
}
