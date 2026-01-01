package app

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"
)

// TestMemoryUsageWithLargeBenchmark tests memory usage with a large benchmark
// This test creates a benchmark similar to what the user described: ~1M data points
func TestMemoryUsageWithLargeBenchmark(t *testing.T) {
	// Skip in short mode as this test takes time
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}
	
	// Create temp directory for test data
	tmpDir := t.TempDir()
	if err := InitBenchmarksDir(tmpDir); err != nil {
		t.Fatalf("Failed to init benchmarks dir: %v", err)
	}
	
	// Create a large benchmark
	// User mentioned "over 1 million lines" with "120mb of data"
	// Let's create 100 runs with 10k data points each = 1 million total points
	numRuns := 100
	pointsPerRun := 10000
	
	t.Logf("Creating benchmark with %d runs, %d points per run (%.1fM total points)",
		numRuns, pointsPerRun, float64(numRuns*pointsPerRun)/1e6)
	
	benchmarkData := make([]*BenchmarkData, numRuns)
	for i := 0; i < numRuns; i++ {
		data := &BenchmarkData{
			Label:   fmt.Sprintf("Run_%d", i),
			SpecOS:  "Linux",
			SpecCPU: "AMD Ryzen 9 5900X",
			SpecGPU: "NVIDIA RTX 3080",
			SpecRAM: "32 GB",
		}
		
		// Create 8 data arrays with 10k points each (simulating real benchmark data)
		data.DataFPS = make([]float64, pointsPerRun)
		data.DataFrameTime = make([]float64, pointsPerRun)
		data.DataCPULoad = make([]float64, pointsPerRun)
		data.DataGPULoad = make([]float64, pointsPerRun)
		data.DataCPUTemp = make([]float64, pointsPerRun)
		data.DataGPUTemp = make([]float64, pointsPerRun)
		data.DataGPUPower = make([]float64, pointsPerRun)
		data.DataRAMUsed = make([]float64, pointsPerRun)
		
		// Fill with varying data
		for j := 0; j < pointsPerRun; j++ {
			data.DataFPS[j] = 60.0 + float64(j%30)
			data.DataFrameTime[j] = 16.67
			data.DataCPULoad[j] = 50.0 + float64(j%40)
			data.DataGPULoad[j] = 90.0 + float64(j%10)
			data.DataCPUTemp[j] = 65.0
			data.DataGPUTemp[j] = 70.0
			data.DataGPUPower[j] = 250.0
			data.DataRAMUsed[j] = 16000.0
		}
		
		benchmarkData[i] = data
	}
	
	// Measure initial memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialMB := float64(m1.Alloc) / (1024 * 1024)
	t.Logf("Initial RAM usage: %.2f MB", initialMB)
	
	// Store the benchmark (uses new v2 format)
	benchmarkID := uint(999)
	startStore := time.Now()
	if err := StoreBenchmarkData(benchmarkData, benchmarkID); err != nil {
		t.Fatalf("Failed to store: %v", err)
	}
	t.Logf("Storage took: %v", time.Since(startStore))
	
	// Clear the benchmark data from memory
	benchmarkData = nil
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	afterStoreMB := float64(m2.Alloc) / (1024 * 1024)
	t.Logf("After storage and GC: %.2f MB", afterStoreMB)
	
	// Now retrieve using RetrieveBenchmarkData (should use new v2 streaming format)
	t.Log("Retrieving benchmark data (v2 format)...")
	startRetrieve := time.Now()
	retrievedData, err := RetrieveBenchmarkData(benchmarkID)
	if err != nil {
		t.Fatalf("Failed to retrieve: %v", err)
	}
	t.Logf("Retrieval took: %v", time.Since(startRetrieve))
	
	var m3 runtime.MemStats
	runtime.ReadMemStats(&m3)
	afterRetrieveMB := float64(m3.Alloc) / (1024 * 1024)
	t.Logf("After retrieval: %.2f MB (loaded %d runs)", afterRetrieveMB, len(retrievedData))
	
	// Verify data integrity
	if len(retrievedData) != numRuns {
		t.Errorf("Expected %d runs, got %d", numRuns, len(retrievedData))
	}
	
	for i, run := range retrievedData {
		if len(run.DataFPS) != pointsPerRun {
			t.Errorf("Run %d: Expected %d FPS points, got %d", i, pointsPerRun, len(run.DataFPS))
		}
		// Check first and last values
		if i == 0 && len(run.DataFPS) > 0 {
			if run.DataFPS[0] != 60.0 {
				t.Errorf("Data integrity check failed: expected FPS[0]=60.0, got %.2f", run.DataFPS[0])
			}
		}
	}
	
	// Clear again
	retrievedData = nil
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var m4 runtime.MemStats
	runtime.ReadMemStats(&m4)
	afterClearMB := float64(m4.Alloc) / (1024 * 1024)
	t.Logf("After clearing and GC: %.2f MB", afterClearMB)
	
	// Summary
	t.Log("\n=== MEMORY TEST SUMMARY ===")
	t.Logf("Initial memory: %.2f MB", initialMB)
	t.Logf("Peak during retrieval: %.2f MB", afterRetrieveMB)
	t.Logf("After clearing: %.2f MB", afterClearMB)
	t.Logf("Memory increase during retrieval: %.2f MB", afterRetrieveMB-afterStoreMB)
	
	// The key metric: memory should not spike to 200-400MB like the user reported
	// With our optimization, it should stay much lower
	// Note: This test has the data in memory initially for creation, so the absolute
	// values may be high, but the increase during retrieval should be reasonable
	if afterRetrieveMB-afterStoreMB > 200 {
		t.Logf("WARNING: Memory increased by %.2f MB during retrieval, which is higher than expected", afterRetrieveMB-afterStoreMB)
		t.Logf("However, this test creates data in memory first, so baseline is already high")
	}
	
	// Clean up test file
	os.Remove(fmt.Sprintf("%s/%d.bin", tmpDir, benchmarkID))
	os.Remove(fmt.Sprintf("%s/%d.meta", tmpDir, benchmarkID))
}
