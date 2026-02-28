package app

import (
	"math"
	"sort"
)

const (
	maxDownsamplePoints = 2000 // Default LTTB downsample threshold for series data
)

// MetricStats holds pre-calculated statistics for a single metric.
// JSON tags match the frontend expectations (camelCase for WebUI consumption).
type MetricStats struct {
	Min      float64  `json:"min"`
	Max      float64  `json:"max"`
	Avg      float64  `json:"avg"`
	Median   float64  `json:"median"`
	P01      float64  `json:"p01"`
	P97      float64  `json:"p97"`
	StdDev   float64  `json:"stddev"`
	Variance float64  `json:"variance"`
	Count    int      `json:"count"`
	Density  [][2]int `json:"density"` // [[roundedValue, count], ...]
}

// PreCalculatedRun stores all pre-calculated data for a single benchmark run.
// JSON tags match the frontend expectations for direct consumption by the WebUI.
// For MCP, use PreCalculatedRunToMCPSummary to convert to the MCP format.
type PreCalculatedRun struct {
	Label              string `json:"label"`
	SpecOS             string `json:"specOS"`
	SpecCPU            string `json:"specCPU"`
	SpecGPU            string `json:"specGPU"`
	SpecRAM            string `json:"specRAM"`
	SpecLinuxKernel    string `json:"specLinuxKernel,omitempty"`
	SpecLinuxScheduler string `json:"specLinuxScheduler,omitempty"`
	TotalDataPoints    int    `json:"totalDataPoints"`

	// Downsampled series data for line charts (LTTB, max 2000 points)
	// metric key -> [[index, value], ...]
	Series map[string][][2]float64 `json:"series"`

	// Pre-calculated stats for Linear Interpolation method
	Stats map[string]*MetricStats `json:"stats"`

	// Pre-calculated stats for MangoHud threshold method
	StatsMangoHud map[string]*MetricStats `json:"statsMangoHud"`
}

// percentileLinear computes the p-th percentile using linear interpolation.
func percentileLinear(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	rank := (p / 100) * float64(len(sorted)-1)
	lower := int(math.Floor(rank))
	upper := int(math.Ceil(rank))

	if lower == upper || upper >= len(sorted) {
		return sorted[lower]
	}

	fraction := rank - float64(lower)
	return sorted[lower]*(1-fraction) + sorted[upper]*fraction
}

// percentileMangoHud computes the p-th percentile using MangoHud's floor-based method.
func percentileMangoHud(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}

	valMango := (100 - p) / 100
	idxDesc := int(math.Floor(valMango*float64(n) - 1))
	idx := n - 1 - idxDesc

	// Clamp index to valid range
	if idx < 0 {
		idx = 0
	}
	if idx > n-1 {
		idx = n - 1
	}

	return sorted[idx]
}

// percentileFunc returns the appropriate percentile function for the given method.
func percentileFunc(method string) func([]float64, float64) float64 {
	if method == "mangohud" {
		return percentileMangoHud
	}
	return percentileLinear
}

// computeDensityData computes a density histogram from values, filtering outliers outside p01-p97.
func computeDensityData(values []float64, p01, p97 float64) [][2]int {
	counts := make(map[int]int)
	for _, v := range values {
		if v >= p01 && v <= p97 {
			rounded := int(math.Round(v))
			counts[rounded]++
		}
	}

	density := make([][2]int, 0, len(counts))
	for val, count := range counts {
		density = append(density, [2]int{val, count})
	}
	sort.Slice(density, func(i, j int) bool {
		return density[i][0] < density[j][0]
	})
	return density
}

// computeMetricStatsForMethod computes statistics for a single metric using the specified method.
func computeMetricStatsForMethod(data []float64, method string) *MetricStats {
	n := len(data)
	if n == 0 {
		return nil
	}

	var sum float64
	minVal := data[0]
	maxVal := data[0]
	for _, v := range data {
		sum += v
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	avg := sum / float64(n)

	// Sample variance (n-1 divisor)
	var sumSq float64
	for _, v := range data {
		diff := v - avg
		sumSq += diff * diff
	}
	var variance float64
	if n > 1 {
		variance = sumSq / float64(n-1)
	}
	stdDev := math.Sqrt(variance)

	sorted := make([]float64, n)
	copy(sorted, data)
	sort.Float64s(sorted)

	pFunc := percentileFunc(method)
	median := pFunc(sorted, 50)
	p01 := pFunc(sorted, 1)
	p97 := pFunc(sorted, 97)

	density := computeDensityData(data, p01, p97)

	return &MetricStats{
		Min:      math.Round(minVal*100) / 100,
		Max:      math.Round(maxVal*100) / 100,
		Avg:      math.Round(avg*100) / 100,
		Median:   math.Round(median*100) / 100,
		P01:      math.Round(p01*100) / 100,
		P97:      math.Round(p97*100) / 100,
		StdDev:   math.Round(stdDev*100) / 100,
		Variance: math.Round(variance*100) / 100,
		Count:    n,
		Density:  density,
	}
}

// computeFPSFromFrametimeForMethod computes FPS statistics derived from frametime data.
// Percentiles are inverted: p03 frametime → p97 FPS, p99 frametime → p01 FPS.
func computeFPSFromFrametimeForMethod(frametimeData []float64, method string) *MetricStats {
	n := len(frametimeData)
	if n == 0 {
		return nil
	}

	sortedFT := make([]float64, n)
	copy(sortedFT, frametimeData)
	sort.Float64s(sortedFT)

	pFunc := percentileFunc(method)

	// Inverted percentiles: low frametime = high FPS
	ftP03 := pFunc(sortedFT, 3)
	ftP99 := pFunc(sortedFT, 99)

	var fpsP97, fpsP01 float64
	if ftP03 > 0 {
		fpsP97 = 1000 / ftP03
	}
	if ftP99 > 0 {
		fpsP01 = 1000 / ftP99
	}

	// Average FPS from average frametime
	var ftSum float64
	for _, v := range frametimeData {
		ftSum += v
	}
	avgFT := ftSum / float64(n)
	var avgFPS float64
	if avgFT > 0 {
		avgFPS = 1000 / avgFT
	}

	// Min/Max FPS from max/min frametime
	minFT := sortedFT[0]
	maxFT := sortedFT[n-1]
	var maxFPS, minFPS float64
	if minFT > 0 {
		maxFPS = 1000 / minFT
	}
	if maxFT > 0 {
		minFPS = 1000 / maxFT
	}

	// Convert all frametime to FPS for stddev/variance/median/density
	fpsValues := make([]float64, n)
	for i, ft := range frametimeData {
		if ft > 0 {
			fpsValues[i] = 1000 / ft
		}
	}

	fpsMean := func() float64 {
		var s float64
		for _, v := range fpsValues {
			s += v
		}
		return s / float64(n)
	}()
	var sumSq float64
	for _, v := range fpsValues {
		diff := v - fpsMean
		sumSq += diff * diff
	}
	var variance float64
	if n > 1 {
		variance = sumSq / float64(n-1)
	}
	stdDev := math.Sqrt(variance)

	sortedFPS := make([]float64, n)
	copy(sortedFPS, fpsValues)
	sort.Float64s(sortedFPS)
	medianFPS := pFunc(sortedFPS, 50)

	// Density uses converted FPS values
	density := computeDensityData(fpsValues, fpsP01, fpsP97)

	return &MetricStats{
		Min:      math.Round(minFPS*100) / 100,
		Max:      math.Round(maxFPS*100) / 100,
		Avg:      math.Round(avgFPS*100) / 100,
		Median:   math.Round(medianFPS*100) / 100,
		P01:      math.Round(fpsP01*100) / 100,
		P97:      math.Round(fpsP97*100) / 100,
		StdDev:   math.Round(stdDev*100) / 100,
		Variance: math.Round(variance*100) / 100,
		Count:    n,
		Density:  density,
	}
}

// downsampleLTTB performs Largest Triangle Three Buckets downsampling on indexed data.
func downsampleLTTB(data [][2]float64, threshold int) [][2]float64 {
	n := len(data)
	if n == 0 {
		return nil
	}
	if n <= threshold || threshold <= 2 {
		result := make([][2]float64, n)
		copy(result, data)
		return result
	}

	sampled := make([][2]float64, 0, threshold)
	bucketSize := float64(n-2) / float64(threshold-2)

	// Always add first point
	sampled = append(sampled, data[0])

	for i := 0; i < threshold-2; i++ {
		// Calculate average of next bucket
		avgRangeStart := int(math.Floor(float64(i+1)*bucketSize)) + 1
		avgRangeEnd := int(math.Floor(float64(i+2)*bucketSize)) + 1
		if avgRangeEnd > n {
			avgRangeEnd = n
		}

		var avgX, avgY float64
		validPoints := 0
		for j := avgRangeStart; j < avgRangeEnd; j++ {
			avgX += data[j][0]
			avgY += data[j][1]
			validPoints++
		}
		if validPoints == 0 {
			continue
		}
		avgX /= float64(validPoints)
		avgY /= float64(validPoints)

		// Current bucket range
		rangeStart := int(math.Floor(float64(i)*bucketSize)) + 1
		rangeEnd := int(math.Floor(float64(i+1)*bucketSize)) + 1
		if rangeEnd > n {
			rangeEnd = n
		}

		lastPoint := sampled[len(sampled)-1]
		pointAX := lastPoint[0]
		pointAY := lastPoint[1]

		maxArea := -1.0
		var maxAreaPoint [2]float64
		found := false

		for j := rangeStart; j < rangeEnd; j++ {
			area := math.Abs(
				(pointAX-avgX)*(data[j][1]-pointAY)-
					(pointAX-data[j][0])*(avgY-pointAY),
			) * 0.5

			if area > maxArea {
				maxArea = area
				maxAreaPoint = data[j]
				found = true
			}
		}

		if found {
			sampled = append(sampled, maxAreaPoint)
		}
	}

	// Always add last point
	sampled = append(sampled, data[n-1])

	return sampled
}

// metricKeyToSnake maps camelCase metric keys to snake_case for MCP compatibility.
var metricKeyToSnake = map[string]string{
	"FPS":          "fps",
	"FrameTime":    "frame_time",
	"CPULoad":      "cpu_load",
	"GPULoad":      "gpu_load",
	"CPUTemp":      "cpu_temp",
	"CPUPower":     "cpu_power",
	"GPUTemp":      "gpu_temp",
	"GPUCoreClock": "gpu_core_clock",
	"GPUMemClock":  "gpu_mem_clock",
	"GPUVRAMUsed":  "gpu_vram_used",
	"GPUPower":     "gpu_power",
	"RAMUsed":      "ram_used",
	"SwapUsed":     "swap_used",
}

// buildSeriesData creates indexed [index, value] pairs from a raw data slice.
func buildSeriesData(data []float64) [][2]float64 {
	points := make([][2]float64, len(data))
	for i, v := range data {
		points[i] = [2]float64{float64(i), v}
	}
	return points
}

// ComputePreCalculatedRuns computes pre-calculated data for all benchmark runs.
func ComputePreCalculatedRuns(runs []*BenchmarkData) []*PreCalculatedRun {
	results := make([]*PreCalculatedRun, len(runs))

	for i, run := range runs {
		results[i] = computePreCalculatedRun(run)
	}

	return results
}

// computePreCalculatedRun computes pre-calculated data for a single benchmark run.
func computePreCalculatedRun(run *BenchmarkData) *PreCalculatedRun {
	totalPoints := len(run.DataFPS)
	if totalPoints == 0 {
		totalPoints = len(run.DataFrameTime)
	}

	result := &PreCalculatedRun{
		Label:              run.Label,
		SpecOS:             run.SpecOS,
		SpecCPU:            run.SpecCPU,
		SpecGPU:            run.SpecGPU,
		SpecRAM:            run.SpecRAM,
		SpecLinuxKernel:    run.SpecLinuxKernel,
		SpecLinuxScheduler: run.SpecLinuxScheduler,
		TotalDataPoints:    totalPoints,
		Series:             make(map[string][][2]float64),
		Stats:              make(map[string]*MetricStats),
		StatsMangoHud:      make(map[string]*MetricStats),
	}

	type metricEntry struct {
		key  string
		data []float64
	}

	metrics := []metricEntry{
		{"FrameTime", run.DataFrameTime},
		{"CPULoad", run.DataCPULoad},
		{"GPULoad", run.DataGPULoad},
		{"CPUTemp", run.DataCPUTemp},
		{"CPUPower", run.DataCPUPower},
		{"GPUTemp", run.DataGPUTemp},
		{"GPUCoreClock", run.DataGPUCoreClock},
		{"GPUMemClock", run.DataGPUMemClock},
		{"GPUVRAMUsed", run.DataGPUVRAMUsed},
		{"GPUPower", run.DataGPUPower},
		{"RAMUsed", run.DataRAMUsed},
		{"SwapUsed", run.DataSwapUsed},
	}

	// Compute series + stats for each standard metric
	for _, m := range metrics {
		if len(m.data) == 0 {
			continue
		}

		// LTTB-downsampled series
		raw := buildSeriesData(m.data)
		result.Series[m.key] = downsampleLTTB(raw, maxDownsamplePoints)

		// Stats for both methods
		result.Stats[m.key] = computeMetricStatsForMethod(m.data, "linear")
		result.StatsMangoHud[m.key] = computeMetricStatsForMethod(m.data, "mangohud")
	}

	// FPS: compute from frametime when available, otherwise from raw FPS data
	if len(run.DataFrameTime) > 0 {
		result.Stats["FPS"] = computeFPSFromFrametimeForMethod(run.DataFrameTime, "linear")
		result.StatsMangoHud["FPS"] = computeFPSFromFrametimeForMethod(run.DataFrameTime, "mangohud")

		// Series uses raw FPS data if available
		if len(run.DataFPS) > 0 {
			raw := buildSeriesData(run.DataFPS)
			result.Series["FPS"] = downsampleLTTB(raw, maxDownsamplePoints)
		}
	} else if len(run.DataFPS) > 0 {
		raw := buildSeriesData(run.DataFPS)
		result.Series["FPS"] = downsampleLTTB(raw, maxDownsamplePoints)

		result.Stats["FPS"] = computeMetricStatsForMethod(run.DataFPS, "linear")
		result.StatsMangoHud["FPS"] = computeMetricStatsForMethod(run.DataFPS, "mangohud")
	}

	return result
}

// PreCalculatedRunToMCPSummary converts pre-calculated data to the MCP BenchmarkDataSummary format.
// Uses linear interpolation stats. If maxPoints > 0, includes downsampled data from the series.
func PreCalculatedRunToMCPSummary(run *PreCalculatedRun, maxPoints int) *BenchmarkDataSummary {
	summary := &BenchmarkDataSummary{
		Label:              run.Label,
		SpecOS:             run.SpecOS,
		SpecCPU:            run.SpecCPU,
		SpecGPU:            run.SpecGPU,
		SpecRAM:            run.SpecRAM,
		SpecLinuxKernel:    run.SpecLinuxKernel,
		SpecLinuxScheduler: run.SpecLinuxScheduler,
		TotalDataPoints:    run.TotalDataPoints,
		Metrics:            make(map[string]*MetricSummary),
	}

	for camelKey, stats := range run.Stats {
		snakeKey, ok := metricKeyToSnake[camelKey]
		if !ok {
			continue
		}

		ms := &MetricSummary{
			Min:      stats.Min,
			Max:      stats.Max,
			Avg:      stats.Avg,
			Median:   stats.Median,
			P01:      stats.P01,
			P97:      stats.P97,
			StdDev:   stats.StdDev,
			Variance: stats.Variance,
			Count:    stats.Count,
		}

		// Include downsampled data if requested
		if maxPoints > 0 {
			if series, exists := run.Series[camelKey]; exists && len(series) > 0 {
				target := maxPoints
				if len(series) < target {
					target = len(series)
				}
				summary.DownsampledTo = target

				// Extract values from [index, value] pairs, further downsample if needed
				if len(series) <= maxPoints {
					ms.Data = make([]float64, len(series))
					for i, pt := range series {
						ms.Data[i] = math.Round(pt[1]*100) / 100
					}
				} else {
					// Re-downsample the already LTTB'd data with simple linear sampling
					ms.Data = make([]float64, maxPoints)
					step := float64(len(series)-1) / float64(maxPoints-1)
					for i := 0; i < maxPoints; i++ {
						idx := int(math.Round(step * float64(i)))
						if idx >= len(series) {
							idx = len(series) - 1
						}
						ms.Data[i] = math.Round(series[idx][1]*100) / 100
					}
				}
			}
		}

		summary.Metrics[snakeKey] = ms
	}

	return summary
}
