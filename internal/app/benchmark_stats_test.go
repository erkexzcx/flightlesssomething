package app

import (
	"math"
	"testing"
)

func approxEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

func TestPercentileLinear(t *testing.T) {
	tests := []struct {
		name     string
		sorted   []float64
		p        float64
		expected float64
	}{
		{"empty", nil, 50, 0},
		{"single element", []float64{42}, 50, 42},
		{"two elements p0", []float64{10, 20}, 0, 10},
		{"two elements p50", []float64{10, 20}, 50, 15},
		{"two elements p100", []float64{10, 20}, 100, 20},
		{"five elements p1", []float64{1, 2, 3, 4, 5}, 1, 1.04},
		{"five elements p50", []float64{1, 2, 3, 4, 5}, 50, 3},
		{"five elements p97", []float64{1, 2, 3, 4, 5}, 97, 4.88},
		{"five elements p99", []float64{1, 2, 3, 4, 5}, 99, 4.96},
		{"interpolation", []float64{10, 20, 30}, 25, 15},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := percentileLinear(tc.sorted, tc.p)
			if !approxEqual(result, tc.expected, 0.01) {
				t.Errorf("percentileLinear(%v, %v) = %v, want %v", tc.sorted, tc.p, result, tc.expected)
			}
		})
	}
}

func TestPercentileMangoHud(t *testing.T) {
	tests := []struct {
		name     string
		sorted   []float64
		p        float64
		expected float64
	}{
		{"empty", nil, 50, 0},
		{"single element", []float64{42}, 50, 42},
		{"ten elements p1", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 1, 2},
		{"ten elements p97", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 97, 10},
		{"three elements p99", []float64{5, 10, 15}, 99, 15},
		{"three elements p1", []float64{5, 10, 15}, 1, 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := percentileMangoHud(tc.sorted, tc.p)
			if !approxEqual(result, tc.expected, 0.01) {
				t.Errorf("percentileMangoHud(%v, %v) = %v, want %v", tc.sorted, tc.p, result, tc.expected)
			}
		})
	}
}

func TestPercentileLinearVsMangoHud(t *testing.T) {
	sorted := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	linearP50 := percentileLinear(sorted, 50)
	mangoP50 := percentileMangoHud(sorted, 50)
	// They should differ (different algorithms)
	if linearP50 == mangoP50 {
		t.Logf("Both methods returned %v for p50, which may be coincidental", linearP50)
	}
}

func TestComputeDensityData(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := computeDensityData(nil, 0, 100)
		if len(result) != 0 {
			t.Errorf("expected empty density, got %v", result)
		}
	})

	t.Run("all outside range", func(t *testing.T) {
		values := []float64{1, 2, 3}
		result := computeDensityData(values, 10, 20)
		if len(result) != 0 {
			t.Errorf("expected empty density, got %v", result)
		}
	})

	t.Run("at boundaries", func(t *testing.T) {
		values := []float64{10, 10, 20, 20}
		result := computeDensityData(values, 10, 20)
		if len(result) != 2 {
			t.Fatalf("expected 2 density entries, got %d", len(result))
		}
		if result[0][0] != 10 || result[0][1] != 2 {
			t.Errorf("expected [10, 2], got %v", result[0])
		}
		if result[1][0] != 20 || result[1][1] != 2 {
			t.Errorf("expected [20, 2], got %v", result[1])
		}
	})

	t.Run("rounding", func(t *testing.T) {
		values := []float64{10.3, 10.7, 10.5}
		result := computeDensityData(values, 10, 11)
		if len(result) != 2 {
			t.Fatalf("expected 2 density entries, got %d", len(result))
		}
		// 10.3 rounds to 10, 10.5 and 10.7 round to 11
		if result[0][0] != 10 || result[0][1] != 1 {
			t.Errorf("expected [10, 1], got %v", result[0])
		}
		if result[1][0] != 11 || result[1][1] != 2 {
			t.Errorf("expected [11, 2], got %v", result[1])
		}
	})

	t.Run("sorted output", func(t *testing.T) {
		values := []float64{30, 10, 20}
		result := computeDensityData(values, 0, 50)
		for i := 1; i < len(result); i++ {
			if result[i][0] < result[i-1][0] {
				t.Errorf("density not sorted: %v", result)
			}
		}
	})
}

func TestComputeMetricStatsForMethod(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		result := computeMetricStatsForMethod(nil, "linear")
		if result != nil {
			t.Errorf("expected nil for empty data, got %v", result)
		}
	})

	t.Run("single element", func(t *testing.T) {
		result := computeMetricStatsForMethod([]float64{42}, "linear")
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Min != 42 || result.Max != 42 || result.Avg != 42 {
			t.Errorf("single element stats wrong: min=%v max=%v avg=%v", result.Min, result.Max, result.Avg)
		}
		if result.Variance != 0 {
			t.Errorf("single element variance should be 0, got %v", result.Variance)
		}
		if result.Count != 1 {
			t.Errorf("expected count=1, got %v", result.Count)
		}
	})

	t.Run("known values linear", func(t *testing.T) {
		data := []float64{10, 20, 30, 40, 50}
		result := computeMetricStatsForMethod(data, "linear")
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Min != 10 || result.Max != 50 {
			t.Errorf("min/max wrong: %v/%v", result.Min, result.Max)
		}
		if result.Avg != 30 {
			t.Errorf("avg wrong: %v", result.Avg)
		}
		if result.Count != 5 {
			t.Errorf("count wrong: %v", result.Count)
		}
		// Verify extended percentiles are populated and ordered
		if result.P01 > result.P05 || result.P05 > result.P10 || result.P10 > result.P25 {
			t.Errorf("lower percentiles not in order: p01=%v p05=%v p10=%v p25=%v", result.P01, result.P05, result.P10, result.P25)
		}
		if result.P25 > result.Median || result.Median > result.P75 || result.P75 > result.P90 {
			t.Errorf("mid percentiles not in order: p25=%v median=%v p75=%v p90=%v", result.P25, result.Median, result.P75, result.P90)
		}
		if result.P90 > result.P95 || result.P95 > result.P97 || result.P97 > result.P99 {
			t.Errorf("upper percentiles not in order: p90=%v p95=%v p97=%v p99=%v", result.P90, result.P95, result.P97, result.P99)
		}
		// IQR = P75 - P25
		expectedIQR := result.P75 - result.P25
		if !approxEqual(result.IQR, expectedIQR, 0.01) {
			t.Errorf("IQR wrong: got %v, want %v", result.IQR, expectedIQR)
		}
	})

	t.Run("mangohud method", func(t *testing.T) {
		data := []float64{10, 20, 30, 40, 50}
		resultLinear := computeMetricStatsForMethod(data, "linear")
		resultMango := computeMetricStatsForMethod(data, "mangohud")
		if resultLinear == nil || resultMango == nil {
			t.Fatal("expected non-nil results")
		}
		// Min, Max, Avg, Variance, StdDev should be the same
		if resultLinear.Min != resultMango.Min || resultLinear.Max != resultMango.Max {
			t.Errorf("min/max should be identical between methods")
		}
		if resultLinear.Avg != resultMango.Avg {
			t.Errorf("avg should be identical between methods")
		}
	})

	t.Run("rounding applied", func(t *testing.T) {
		data := []float64{1.005, 2.005, 3.005}
		result := computeMetricStatsForMethod(data, "linear")
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		// Verify all stats are rounded to 2 decimal places
		check := func(name string, val float64) {
			t.Helper()
			rounded := math.Round(val*100) / 100
			if val != rounded {
				t.Errorf("%s not rounded to 2 decimals: %v", name, val)
			}
		}
		check("Min", result.Min)
		check("Max", result.Max)
		check("Avg", result.Avg)
		check("Median", result.Median)
		check("P01", result.P01)
		check("P05", result.P05)
		check("P10", result.P10)
		check("P25", result.P25)
		check("P75", result.P75)
		check("P90", result.P90)
		check("P95", result.P95)
		check("P97", result.P97)
		check("P99", result.P99)
		check("IQR", result.IQR)
		check("StdDev", result.StdDev)
		check("Variance", result.Variance)
	})
}

func TestComputeFPSFromFrametimeForMethod(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		result := computeFPSFromFrametimeForMethod(nil, "linear")
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("zero frametime handling", func(t *testing.T) {
		ft := []float64{0, 0, 0}
		result := computeFPSFromFrametimeForMethod(ft, "linear")
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		// All zeros should not panic, FPS should be 0
		if result.Avg != 0 || result.Min != 0 || result.Max != 0 {
			t.Errorf("zero frametime should yield zero FPS")
		}
	})

	t.Run("constant frametime", func(t *testing.T) {
		ft := make([]float64, 100)
		for i := range ft {
			ft[i] = 16.67 // ~60 FPS
		}
		result := computeFPSFromFrametimeForMethod(ft, "linear")
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		expectedFPS := math.Round(1000/16.67*100) / 100
		if !approxEqual(result.Avg, expectedFPS, 0.1) {
			t.Errorf("avg FPS: got %v, want ~%v", result.Avg, expectedFPS)
		}
		if result.Count != 100 {
			t.Errorf("count: got %v, want 100", result.Count)
		}
	})

	t.Run("inverted percentiles", func(t *testing.T) {
		// Slower frametime = lower FPS
		ft := make([]float64, 1000)
		for i := range ft {
			ft[i] = float64(10 + i) // 10ms to 1009ms
		}
		result := computeFPSFromFrametimeForMethod(ft, "linear")
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		// P97 FPS should be higher than P01 FPS
		if result.P97 <= result.P01 {
			t.Errorf("p97 (%v) should be > p01 (%v) for FPS", result.P97, result.P01)
		}
		// All extended percentiles should be ordered
		if result.P01 > result.P05 || result.P05 > result.P10 || result.P10 > result.P25 {
			t.Errorf("lower FPS percentiles not in order: p01=%v p05=%v p10=%v p25=%v", result.P01, result.P05, result.P10, result.P25)
		}
		if result.P75 > result.P90 || result.P90 > result.P95 || result.P95 > result.P97 || result.P97 > result.P99 {
			t.Errorf("upper FPS percentiles not in order: p75=%v p90=%v p95=%v p97=%v p99=%v", result.P75, result.P90, result.P95, result.P97, result.P99)
		}
		// IQR should be positive
		if result.IQR <= 0 {
			t.Errorf("IQR should be > 0, got %v", result.IQR)
		}
		// Min FPS should come from max frametime
		if result.Min <= 0 {
			t.Errorf("min FPS should be > 0, got %v", result.Min)
		}
	})

	t.Run("mangohud method", func(t *testing.T) {
		ft := make([]float64, 100)
		for i := range ft {
			ft[i] = float64(10 + i)
		}
		resultLinear := computeFPSFromFrametimeForMethod(ft, "linear")
		resultMango := computeFPSFromFrametimeForMethod(ft, "mangohud")
		if resultLinear == nil || resultMango == nil {
			t.Fatal("expected non-nil results")
		}
		// Min, Max, Avg should be the same (not affected by percentile method)
		if resultLinear.Min != resultMango.Min {
			t.Errorf("min should match: linear=%v mangohud=%v", resultLinear.Min, resultMango.Min)
		}
	})

	t.Run("density uses FPS values", func(t *testing.T) {
		ft := []float64{10, 10, 10, 20, 20} // 100, 100, 100, 50, 50 FPS
		result := computeFPSFromFrametimeForMethod(ft, "linear")
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result.Density) == 0 {
			t.Error("expected non-empty density")
		}
		// Density values should be FPS-like (50, 100), not frametime-like (10, 20)
		for _, d := range result.Density {
			if d[0] < 40 {
				t.Errorf("density value %d looks like frametime, expected FPS", d[0])
			}
		}
	})
}

func TestDownsampleLTTB(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := downsampleLTTB(nil, 10)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		result := downsampleLTTB([][2]float64{}, 10)
		if result != nil {
			t.Errorf("expected nil for empty, got %v", result)
		}
	})

	t.Run("below threshold", func(t *testing.T) {
		data := [][2]float64{{0, 1}, {1, 2}, {2, 3}}
		result := downsampleLTTB(data, 10)
		if len(result) != 3 {
			t.Errorf("expected 3 points, got %d", len(result))
		}
	})

	t.Run("threshold 2 or less", func(t *testing.T) {
		data := [][2]float64{{0, 1}, {1, 2}, {2, 3}, {3, 4}, {4, 5}}
		result := downsampleLTTB(data, 2)
		if len(result) != 5 {
			t.Errorf("threshold <= 2 should return all points, got %d", len(result))
		}
	})

	t.Run("preserves first and last", func(t *testing.T) {
		data := make([][2]float64, 100)
		for i := range data {
			data[i] = [2]float64{float64(i), float64(i * i)}
		}
		result := downsampleLTTB(data, 20)
		if len(result) != 20 {
			t.Errorf("expected 20 points, got %d", len(result))
		}
		if result[0] != data[0] {
			t.Errorf("first point not preserved: got %v, want %v", result[0], data[0])
		}
		if result[len(result)-1] != data[len(data)-1] {
			t.Errorf("last point not preserved: got %v, want %v", result[len(result)-1], data[len(data)-1])
		}
	})

	t.Run("reduces data size", func(t *testing.T) {
		data := make([][2]float64, 5000)
		for i := range data {
			data[i] = [2]float64{float64(i), math.Sin(float64(i) / 100)}
		}
		result := downsampleLTTB(data, 200)
		if len(result) != 200 {
			t.Errorf("expected 200 points, got %d", len(result))
		}
	})
}

func TestBuildSeriesData(t *testing.T) {
	data := []float64{10.5, 20.3, 30.7}
	result := buildSeriesData(data)
	if len(result) != 3 {
		t.Fatalf("expected 3 points, got %d", len(result))
	}
	for i, pt := range result {
		if pt[0] != float64(i) {
			t.Errorf("point %d: index=%v, want %v", i, pt[0], float64(i))
		}
		if pt[1] != data[i] {
			t.Errorf("point %d: value=%v, want %v", i, pt[1], data[i])
		}
	}
}

func TestComputePreCalculatedRuns(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		result := ComputePreCalculatedRuns(nil)
		if len(result) != 0 {
			t.Errorf("expected 0 results, got %d", len(result))
		}
	})

	t.Run("single run no data", func(t *testing.T) {
		runs := []*BenchmarkData{{Label: "empty"}}
		result := ComputePreCalculatedRuns(runs)
		if len(result) != 1 {
			t.Fatalf("expected 1 result, got %d", len(result))
		}
		if result[0].Label != "empty" {
			t.Errorf("label: got %v, want empty", result[0].Label)
		}
		if len(result[0].Stats) != 0 {
			t.Errorf("expected no stats for empty run, got %d", len(result[0].Stats))
		}
	})

	t.Run("run with only FPS", func(t *testing.T) {
		runs := []*BenchmarkData{{
			Label:   "fps-only",
			DataFPS: []float64{60, 61, 59, 60, 62},
		}}
		result := ComputePreCalculatedRuns(runs)
		if len(result) != 1 {
			t.Fatalf("expected 1 result, got %d", len(result))
		}
		r := result[0]
		if r.TotalDataPoints != 5 {
			t.Errorf("total data points: got %d, want 5", r.TotalDataPoints)
		}
		if _, ok := r.Stats["FPS"]; !ok {
			t.Error("expected FPS stats")
		}
		if _, ok := r.StatsMangoHud["FPS"]; !ok {
			t.Error("expected FPS mangohud stats")
		}
		if _, ok := r.Series["FPS"]; !ok {
			t.Error("expected FPS series")
		}
	})

	t.Run("run with only frametime", func(t *testing.T) {
		ft := make([]float64, 50)
		for i := range ft {
			ft[i] = 16.67
		}
		runs := []*BenchmarkData{{
			Label:         "ft-only",
			DataFrameTime: ft,
		}}
		result := ComputePreCalculatedRuns(runs)
		r := result[0]
		if _, ok := r.Stats["FPS"]; !ok {
			t.Error("expected FPS stats computed from frametime")
		}
		if _, ok := r.Stats["FrameTime"]; !ok {
			t.Error("expected FrameTime stats")
		}
		// No FPS series since no raw FPS data
		if _, ok := r.Series["FPS"]; ok {
			t.Error("should not have FPS series without raw FPS data")
		}
	})

	t.Run("run with both FPS and frametime", func(t *testing.T) {
		ft := make([]float64, 50)
		fps := make([]float64, 50)
		for i := range ft {
			ft[i] = 16.67
			fps[i] = 1000 / 16.67
		}
		runs := []*BenchmarkData{{
			Label:         "both",
			DataFrameTime: ft,
			DataFPS:       fps,
		}}
		result := ComputePreCalculatedRuns(runs)
		r := result[0]
		if _, ok := r.Stats["FPS"]; !ok {
			t.Error("expected FPS stats")
		}
		if _, ok := r.Series["FPS"]; !ok {
			t.Error("expected FPS series from raw FPS data")
		}
		if _, ok := r.Series["FrameTime"]; !ok {
			t.Error("expected FrameTime series")
		}
	})

	t.Run("run with all metrics", func(t *testing.T) {
		data := make([]float64, 100)
		for i := range data {
			data[i] = float64(i)
		}
		runs := []*BenchmarkData{{
			Label:            "full",
			SpecOS:           "Linux",
			SpecCPU:          "Ryzen 9",
			SpecGPU:          "RTX 4090",
			SpecRAM:          "32GB",
			SpecLinuxKernel:   "6.5",
			SpecLinuxScheduler: "EEVDF",
			DataFPS:          data,
			DataFrameTime:    data,
			DataCPULoad:      data,
			DataGPULoad:      data,
			DataCPUTemp:      data,
			DataCPUPower:     data,
			DataGPUTemp:      data,
			DataGPUCoreClock: data,
			DataGPUMemClock:  data,
			DataGPUVRAMUsed:  data,
			DataGPUPower:     data,
			DataRAMUsed:      data,
			DataSwapUsed:     data,
		}}
		result := ComputePreCalculatedRuns(runs)
		r := result[0]
		if r.SpecOS != "Linux" || r.SpecCPU != "Ryzen 9" {
			t.Error("spec fields not copied")
		}

		expectedMetrics := []string{"FPS", "FrameTime", "CPULoad", "GPULoad", "CPUTemp", "CPUPower", "GPUTemp", "GPUCoreClock", "GPUMemClock", "GPUVRAMUsed", "GPUPower", "RAMUsed", "SwapUsed"}
		for _, key := range expectedMetrics {
			if _, ok := r.Stats[key]; !ok {
				t.Errorf("missing Stats[%s]", key)
			}
			if _, ok := r.StatsMangoHud[key]; !ok {
				t.Errorf("missing StatsMangoHud[%s]", key)
			}
		}
	})

	t.Run("multiple runs", func(t *testing.T) {
		runs := []*BenchmarkData{
			{Label: "run1", DataFPS: []float64{60, 61}},
			{Label: "run2", DataFPS: []float64{120, 121, 122}},
		}
		result := ComputePreCalculatedRuns(runs)
		if len(result) != 2 {
			t.Fatalf("expected 2 results, got %d", len(result))
		}
		if result[0].Label != "run1" || result[1].Label != "run2" {
			t.Error("labels not preserved")
		}
		if result[0].TotalDataPoints != 2 || result[1].TotalDataPoints != 3 {
			t.Error("data point counts wrong")
		}
	})
}

func TestPreCalculatedRunToMCPSummary(t *testing.T) {
	// Build a pre-calculated run with some data
	fpsStats := &MetricStats{
		Min: 50, Max: 120, Avg: 90, Median: 91,
		P01: 55, P05: 60, P10: 65, P25: 75, P75: 105, P90: 110, P95: 113, P97: 115, P99: 118,
		IQR: 30, StdDev: 10, Variance: 100,
		Count: 500, Density: [][2]int{{55, 1}, {90, 10}, {115, 1}},
	}
	series := make([][2]float64, 100)
	for i := range series {
		series[i] = [2]float64{float64(i), float64(60 + i)}
	}

	run := &PreCalculatedRun{
		Label:           "test-run",
		SpecOS:          "Linux",
		SpecCPU:         "Ryzen 9",
		SpecGPU:         "RTX 4090",
		SpecRAM:         "32GB",
		TotalDataPoints: 500,
		Series:          map[string][][2]float64{"FPS": series},
		Stats:           map[string]*MetricStats{"FPS": fpsStats},
		StatsMangoHud:   map[string]*MetricStats{"FPS": fpsStats},
	}

	t.Run("no data points", func(t *testing.T) {
		summary := PreCalculatedRunToMCPSummary(run, 0)
		if summary.Label != "test-run" {
			t.Errorf("label: got %v, want test-run", summary.Label)
		}
		fps, ok := summary.Metrics["fps"]
		if !ok {
			t.Fatal("expected fps metric")
		}
		if fps.Min != 50 || fps.Max != 120 {
			t.Errorf("fps stats wrong: min=%v max=%v", fps.Min, fps.Max)
		}
		if fps.P05 != 60 || fps.P10 != 65 || fps.P25 != 75 || fps.P75 != 105 || fps.P90 != 110 || fps.P95 != 113 || fps.P99 != 118 || fps.IQR != 30 {
			t.Error("extended percentile fields not mapped correctly to MCP summary")
		}
		if fps.Data != nil {
			t.Error("expected no data when maxPoints=0")
		}
	})

	t.Run("maxPoints greater than series", func(t *testing.T) {
		summary := PreCalculatedRunToMCPSummary(run, 200)
		fps := summary.Metrics["fps"]
		if fps == nil {
			t.Fatal("expected fps metric")
		}
		if len(fps.Data) != 100 {
			t.Errorf("expected 100 data points (series length), got %d", len(fps.Data))
		}
	})

	t.Run("maxPoints less than series triggers redownsample", func(t *testing.T) {
		summary := PreCalculatedRunToMCPSummary(run, 20)
		fps := summary.Metrics["fps"]
		if fps == nil {
			t.Fatal("expected fps metric")
		}
		if len(fps.Data) != 20 {
			t.Errorf("expected 20 data points, got %d", len(fps.Data))
		}
	})

	t.Run("key mapping", func(t *testing.T) {
		fullRun := &PreCalculatedRun{
			TotalDataPoints: 10,
			Series:          make(map[string][][2]float64),
			Stats: map[string]*MetricStats{
				"FPS":          {Count: 10},
				"FrameTime":    {Count: 10},
				"CPULoad":      {Count: 10},
				"GPULoad":      {Count: 10},
				"CPUTemp":      {Count: 10},
				"CPUPower":     {Count: 10},
				"GPUTemp":      {Count: 10},
				"GPUCoreClock": {Count: 10},
				"GPUMemClock":  {Count: 10},
				"GPUVRAMUsed":  {Count: 10},
				"GPUPower":     {Count: 10},
				"RAMUsed":      {Count: 10},
				"SwapUsed":     {Count: 10},
			},
			StatsMangoHud: make(map[string]*MetricStats),
		}
		summary := PreCalculatedRunToMCPSummary(fullRun, 0)

		expectedKeys := []string{"fps", "frame_time", "cpu_load", "gpu_load", "cpu_temp", "cpu_power", "gpu_temp", "gpu_core_clock", "gpu_mem_clock", "gpu_vram_used", "gpu_power", "ram_used", "swap_used"}
		for _, key := range expectedKeys {
			if _, ok := summary.Metrics[key]; !ok {
				t.Errorf("missing metric key %s in MCP summary", key)
			}
		}
	})

	t.Run("missing metric key ignored", func(t *testing.T) {
		badRun := &PreCalculatedRun{
			TotalDataPoints: 10,
			Series:          make(map[string][][2]float64),
			Stats:           map[string]*MetricStats{"UnknownMetric": {Count: 5}},
			StatsMangoHud:   make(map[string]*MetricStats),
		}
		summary := PreCalculatedRunToMCPSummary(badRun, 0)
		if len(summary.Metrics) != 0 {
			t.Errorf("expected no metrics for unknown keys, got %d", len(summary.Metrics))
		}
	})

	t.Run("data values rounded", func(t *testing.T) {
		smallSeries := [][2]float64{{0, 10.123}, {1, 20.456}, {2, 30.789}}
		roundRun := &PreCalculatedRun{
			TotalDataPoints: 3,
			Series:          map[string][][2]float64{"FPS": smallSeries},
			Stats:           map[string]*MetricStats{"FPS": {Count: 3}},
			StatsMangoHud:   make(map[string]*MetricStats),
		}
		summary := PreCalculatedRunToMCPSummary(roundRun, 10)
		fps := summary.Metrics["fps"]
		if fps == nil {
			t.Fatal("expected fps metric")
		}
		for i, v := range fps.Data {
			rounded := math.Round(v*100) / 100
			if v != rounded {
				t.Errorf("data[%d] not rounded: %v", i, v)
			}
		}
	})
}
