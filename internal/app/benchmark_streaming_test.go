package app

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"time"
)

// TestStreamingMemoryUsage tests that streaming JSON encoding uses minimal memory
func TestStreamingMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming memory test in short mode")
	}
	
	// Create temp directory
	tmpDir := t.TempDir()
	if err := InitBenchmarksDir(tmpDir); err != nil {
		t.Fatalf("Failed to init benchmarks dir: %v", err)
	}
	
	// Create a large benchmark - similar to user's scenario
	numRuns := 100
	pointsPerRun := 10000
	
	t.Logf("Creating large benchmark: %d runs × %d points = %.1fM data points",
		numRuns, pointsPerRun, float64(numRuns*pointsPerRun)/1e6)
	
	benchmarkData := make([]*BenchmarkData, numRuns)
	for i := 0; i < numRuns; i++ {
		data := &BenchmarkData{
			Label:   fmt.Sprintf("LargeRun_%d", i),
			SpecOS:  "Linux",
			SpecCPU: "AMD Ryzen 9 5900X",
			SpecGPU: "NVIDIA RTX 3080",
			SpecRAM: "32 GB",
		}
		
		// Create large data arrays (8 arrays × 10k points × 8 bytes = 640KB per run)
		data.DataFPS = make([]float64, pointsPerRun)
		data.DataFrameTime = make([]float64, pointsPerRun)
		data.DataCPULoad = make([]float64, pointsPerRun)
		data.DataGPULoad = make([]float64, pointsPerRun)
		data.DataCPUTemp = make([]float64, pointsPerRun)
		data.DataGPUTemp = make([]float64, pointsPerRun)
		data.DataGPUPower = make([]float64, pointsPerRun)
		data.DataRAMUsed = make([]float64, pointsPerRun)
		
		// Fill with data
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
	
	// Store using new v2 format
	benchmarkID := uint(1234)
	if err := StoreBenchmarkData(benchmarkData, benchmarkID); err != nil {
		t.Fatalf("Failed to store benchmark: %v", err)
	}
	
	// Clear from memory and trigger GC
	benchmarkData = nil //nolint:ineffassign // Intentional to help GC reclaim memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	baselineMB := float64(m1.Alloc) / (1024 * 1024)
	t.Logf("Baseline memory before streaming: %.2f MB", baselineMB)
	
	// Create a response writer that discards data (simulates sending to client)
	// We use a custom writer that counts bytes but doesn't store them
	discard := &discardWriter{bytesWritten: 0}
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{} // This will be ignored, we'll override Write
	
	// Override to use discard writer
	responseWriter := &customResponseWriter{
		ResponseWriter: w,
		writer:         discard,
	}
	
	// Stream the benchmark data as JSON
	t.Logf("Streaming benchmark data to client (v2 format)...")
	startStream := time.Now()
	
	if err := StreamBenchmarkDataAsJSON(benchmarkID, responseWriter); err != nil {
		t.Fatalf("Failed to stream benchmark: %v", err)
	}
	
	streamDuration := time.Since(startStream)
	t.Logf("Streaming completed in %v", streamDuration)
	t.Logf("Total bytes streamed: %.2f MB", float64(discard.bytesWritten)/(1024*1024))
	
	// Measure memory during streaming
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	duringStreamMB := float64(m2.Alloc) / (1024 * 1024)
	t.Logf("Memory during/after streaming: %.2f MB", duringStreamMB)
	
	// Force GC and wait
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var m3 runtime.MemStats
	runtime.ReadMemStats(&m3)
	afterGCMB := float64(m3.Alloc) / (1024 * 1024)
	t.Logf("Memory after GC: %.2f MB", afterGCMB)
	
	// Calculate memory increase
	memoryIncrease := duringStreamMB - baselineMB
	t.Log("\n=== STREAMING TEST SUMMARY ===")
	t.Logf("Baseline: %.2f MB", baselineMB)
	t.Logf("During stream: %.2f MB", duringStreamMB)
	t.Logf("After GC: %.2f MB", afterGCMB)
	t.Logf("Memory increase during streaming: %.2f MB", memoryIncrease)
	
	// This is the key assertion: memory increase should be minimal
	// With v2 streaming, we only hold ~1 run in memory at a time
	// Expected increase: < 30MB (vs user's 200-400MB spike)
	if memoryIncrease > 50 {
		t.Errorf("Memory increase during streaming (%.2f MB) exceeds expected limit (50 MB)", memoryIncrease)
		t.Errorf("This suggests streaming is not working efficiently")
	} else {
		t.Logf("✓ Memory increase is acceptable: %.2f MB (much better than user's reported 200-400MB)", memoryIncrease)
	}
	
	// Verify JSON was actually streamed (rough check)
	if discard.bytesWritten < 1000000 {
		t.Errorf("Expected to stream at least 1MB of JSON, got %d bytes", discard.bytesWritten)
	}
	
	// Clean up test files
	_ = os.Remove(fmt.Sprintf("%s/%d.bin", tmpDir, benchmarkID))   //nolint:errcheck // Test cleanup, errors not critical
	_ = os.Remove(fmt.Sprintf("%s/%d.meta", tmpDir, benchmarkID)) //nolint:errcheck // Test cleanup, errors not critical
}

// discardWriter is a writer that counts bytes but doesn't store them
type discardWriter struct {
	bytesWritten int64
}

func (d *discardWriter) Write(p []byte) (n int, err error) {
	d.bytesWritten += int64(len(p))
	return len(p), nil
}

// customResponseWriter wraps httptest.ResponseRecorder to use our custom writer
type customResponseWriter struct {
	http.ResponseWriter
	writer io.Writer
	header http.Header
	statusCode int
}

func (c *customResponseWriter) Header() http.Header {
	if c.header == nil {
		c.header = make(http.Header)
	}
	return c.header
}

func (c *customResponseWriter) Write(data []byte) (int, error) {
	return c.writer.Write(data)
}

func (c *customResponseWriter) WriteHeader(statusCode int) {
	c.statusCode = statusCode
}
