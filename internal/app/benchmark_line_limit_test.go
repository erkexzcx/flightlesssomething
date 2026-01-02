package app

import (
	"testing"
)

func TestCountTotalDataLines(t *testing.T) {
	t.Run("empty_benchmark_data", func(t *testing.T) {
		data := []*BenchmarkData{}
		count := CountTotalDataLines(data)
		if count != 0 {
			t.Errorf("Expected 0 lines for empty data, got %d", count)
		}
	})

	t.Run("single_run_with_data", func(t *testing.T) {
		data := []*BenchmarkData{
			{
				Label:   "Run 1",
				DataFPS: []float64{60.0, 61.0, 62.0},
			},
		}
		count := CountTotalDataLines(data)
		if count != 3 {
			t.Errorf("Expected 3 lines, got %d", count)
		}
	})

	t.Run("multiple_runs", func(t *testing.T) {
		data := []*BenchmarkData{
			{
				Label:   "Run 1",
				DataFPS: []float64{60.0, 61.0, 62.0},
			},
			{
				Label:   "Run 2",
				DataFPS: []float64{70.0, 71.0, 72.0, 73.0},
			},
			{
				Label:   "Run 3",
				DataFPS: []float64{80.0, 81.0},
			},
		}
		count := CountTotalDataLines(data)
		// 3 + 4 + 2 = 9
		if count != 9 {
			t.Errorf("Expected 9 lines, got %d", count)
		}
	})

	t.Run("different_array_lengths", func(t *testing.T) {
		data := []*BenchmarkData{
			{
				Label:        "Run 1",
				DataFPS:      []float64{60.0, 61.0, 62.0},
				DataCPULoad:  []float64{50.0, 51.0, 52.0, 53.0, 54.0},
				DataFrameTime: []float64{16.0, 17.0},
			},
		}
		count := CountTotalDataLines(data)
		// Should use the maximum length (5 from DataCPULoad)
		if count != 5 {
			t.Errorf("Expected 5 lines (max of arrays), got %d", count)
		}
	})

	t.Run("run_with_no_data", func(t *testing.T) {
		data := []*BenchmarkData{
			{
				Label: "Run 1",
			},
		}
		count := CountTotalDataLines(data)
		if count != 0 {
			t.Errorf("Expected 0 lines for run with no data, got %d", count)
		}
	})
}

func TestValidatePerRunDataLines(t *testing.T) {
	t.Run("within_limit", func(t *testing.T) {
		data := []*BenchmarkData{
			{
				Label:   "Run 1",
				DataFPS: make([]float64, 1000),
			},
			{
				Label:   "Run 2",
				DataFPS: make([]float64, 5000),
			},
		}
		err := ValidatePerRunDataLines(data)
		if err != nil {
			t.Errorf("Expected no error for data within limit, got: %v", err)
		}
	})

	t.Run("exactly_at_limit", func(t *testing.T) {
		data := []*BenchmarkData{
			{
				Label:   "Run 1",
				DataFPS: make([]float64, maxPerRunDataLines),
			},
		}
		err := ValidatePerRunDataLines(data)
		if err != nil {
			t.Errorf("Expected no error for data exactly at limit, got: %v", err)
		}
	})

	t.Run("exceeds_limit_first_run", func(t *testing.T) {
		data := []*BenchmarkData{
			{
				Label:   "Large Run",
				DataFPS: make([]float64, maxPerRunDataLines+1),
			},
		}
		err := ValidatePerRunDataLines(data)
		if err == nil {
			t.Error("Expected error for data exceeding limit")
		}
		expectedMsg := "Large Run has 500001 data points, which exceeds the maximum allowed 500000 per run"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("exceeds_limit_second_run", func(t *testing.T) {
		data := []*BenchmarkData{
			{
				Label:   "Run 1",
				DataFPS: make([]float64, 1000),
			},
			{
				Label:   "Huge Run",
				DataFPS: make([]float64, maxPerRunDataLines+100),
			},
		}
		err := ValidatePerRunDataLines(data)
		if err == nil {
			t.Error("Expected error for second run exceeding limit")
		}
		expectedMsg := "Huge Run has 500100 data points, which exceeds the maximum allowed 500000 per run"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("exceeds_limit_no_label", func(t *testing.T) {
		data := []*BenchmarkData{
			{
				Label:   "",
				DataFPS: make([]float64, maxPerRunDataLines+1),
			},
		}
		err := ValidatePerRunDataLines(data)
		if err == nil {
			t.Error("Expected error for data exceeding limit")
		}
		expectedMsg := "run #1 has 500001 data points, which exceeds the maximum allowed 500000 per run"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("different_array_lengths_within_limit", func(t *testing.T) {
		data := []*BenchmarkData{
			{
				Label:         "Run 1",
				DataFPS:       make([]float64, 100000),
				DataCPULoad:   make([]float64, 200000),
				DataFrameTime: make([]float64, 150000),
			},
		}
		err := ValidatePerRunDataLines(data)
		if err != nil {
			t.Errorf("Expected no error for data within limit, got: %v", err)
		}
	})

	t.Run("different_array_lengths_exceeds_limit", func(t *testing.T) {
		data := []*BenchmarkData{
			{
				Label:         "Run 1",
				DataFPS:       make([]float64, 100000),
				DataCPULoad:   make([]float64, maxPerRunDataLines+1000),
				DataFrameTime: make([]float64, 150000),
			},
		}
		err := ValidatePerRunDataLines(data)
		if err == nil {
			t.Error("Expected error for data exceeding limit")
		}
	})

	t.Run("empty_data", func(t *testing.T) {
		data := []*BenchmarkData{}
		err := ValidatePerRunDataLines(data)
		if err != nil {
			t.Errorf("Expected no error for empty data, got: %v", err)
		}
	})

	t.Run("run_with_no_data_arrays", func(t *testing.T) {
		data := []*BenchmarkData{
			{
				Label: "Empty Run",
			},
		}
		err := ValidatePerRunDataLines(data)
		if err != nil {
			t.Errorf("Expected no error for run with no data arrays, got: %v", err)
		}
	})
}
