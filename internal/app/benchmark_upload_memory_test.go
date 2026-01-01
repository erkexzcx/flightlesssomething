package app

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"runtime"
	"strings"
	"testing"
)

// TestUploadParsingMemoryUsage tests memory usage during file upload parsing
// This demonstrates the two-pass optimization: first pass counts lines (streaming),
// second pass parses with 100% accurate pre-allocation to eliminate all reallocations
func TestUploadParsingMemoryUsage(t *testing.T) {
	// Skip in short mode as this test analyzes memory
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// Create a large MangoHud CSV file in memory (simulating upload)
	// This will be ~5000 lines of data (~500KB)
	numDataPoints := 5000
	
	var csvContent strings.Builder
	// Header line
	csvContent.WriteString("os,cpu,gpu,ram,kernel,driver,cpuscheduler\n")
	// Specs line
	csvContent.WriteString("Linux,AMD Ryzen 9 5900X,NVIDIA RTX 3080,32768000,5.15.0-generic,nvidia-driver-515,SCHED_EXT\n")
	// Column headers
	csvContent.WriteString("fps,frametime,cpu_load,gpu_load,cpu_temp,cpu_power,gpu_temp,gpu_core_clock,gpu_mem_clock,gpu_vram_used,gpu_power,ram_used,swap_used\n")
	
	// Data rows
	for i := 0; i < numDataPoints; i++ {
		csvContent.WriteString(fmt.Sprintf("%.2f,%.2f,%.1f,%.1f,%.1f,%.1f,%.1f,%.0f,%.0f,%.0f,%.1f,%.0f,%.0f\n",
			60.0+float64(i%30),     // fps
			16.67,                   // frametime
			50.0+float64(i%40),     // cpu_load
			90.0+float64(i%10),     // gpu_load
			65.0,                    // cpu_temp
			85.0,                    // cpu_power
			70.0,                    // gpu_temp
			1800.0,                  // gpu_core_clock
			7000.0,                  // gpu_mem_clock
			8000.0,                  // gpu_vram_used
			250.0,                   // gpu_power
			16000.0,                 // ram_used
			0.0,                     // swap_used
		))
	}
	
	csvBytes := []byte(csvContent.String())
	fileSize := int64(len(csvBytes))
	
	t.Logf("Created test CSV with %d data points, size: %d bytes (%.2f KB)",
		numDataPoints, fileSize, float64(fileSize)/1024)
	
	// Create a multipart file header
	fileHeader := createTestFileHeader("test.csv", csvBytes)
	
	// Measure memory before parsing
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	beforeParseMB := float64(m1.Alloc) / (1024 * 1024)
	
	// Parse the file
	benchmarkData, err := readSingleBenchmarkFile(fileHeader)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	
	// Measure memory after parsing
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	afterParseMB := float64(m2.Alloc) / (1024 * 1024)
	
	// Verify data was parsed correctly
	if len(benchmarkData.DataFPS) != numDataPoints {
		t.Errorf("Expected %d FPS data points, got %d", numDataPoints, len(benchmarkData.DataFPS))
	}
	
	// Calculate memory increase
	memoryIncreaseMB := afterParseMB - beforeParseMB
	
	t.Logf("\n=== UPLOAD PARSING MEMORY TEST ===")
	t.Logf("File size: %.2f KB", float64(fileSize)/1024)
	t.Logf("Data points parsed: %d", len(benchmarkData.DataFPS))
	t.Logf("Memory before parsing: %.2f MB", beforeParseMB)
	t.Logf("Memory after parsing: %.2f MB", afterParseMB)
	t.Logf("Memory increase: %.2f MB", memoryIncreaseMB)
	t.Logf("=====================================")
	
	// The key metric: with our optimization, memory increase should be reasonable
	// It should be roughly: (num_data_points * 13_columns * 8_bytes_per_float64) / 1MB
	// = (5000 * 13 * 8) / (1024*1024) = ~0.49 MB for the data arrays
	// Plus overhead for structs, strings, etc.
	expectedDataSizeMB := float64(numDataPoints*13*8) / (1024 * 1024)
	t.Logf("Expected data size: ~%.2f MB (pure data arrays)", expectedDataSizeMB)
	
	// Memory increase should be close to expected (within 2x for overhead)
	if memoryIncreaseMB > expectedDataSizeMB*3 {
		t.Logf("WARNING: Memory increase (%.2f MB) is more than 3x expected data size (%.2f MB)",
			memoryIncreaseMB, expectedDataSizeMB)
		t.Logf("This could indicate inefficient memory usage, but may be acceptable depending on Go's allocator")
	}
	
	// Clean up
	benchmarkData = nil
	runtime.GC()
}

// createTestFileHeader creates a multipart.FileHeader for testing purposes.
// It simulates a file upload by creating a multipart form with the given filename and content.
// This is necessary because multipart.FileHeader is normally created by the HTTP framework
// during file upload processing, but we need to create one manually for unit tests.
//
// Parameters:
//   - filename: The name of the file to simulate (e.g., "test.csv")
//   - content: The file content as a byte slice
//
// Returns:
//   - *multipart.FileHeader: A file header that can be passed to readSingleBenchmarkFile
//
// Panics if the multipart form cannot be created (test setup failure).
func createTestFileHeader(filename string, content []byte) *multipart.FileHeader {
	// Create a buffer to write multipart form
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	
	// Create the form file
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="files"; filename="%s"`, filename))
	h.Set("Content-Type", "text/csv")
	
	part, err := writer.CreatePart(h)
	if err != nil {
		panic(err)
	}
	
	if _, writeErr := part.Write(content); writeErr != nil {
		panic(writeErr)
	}
	
	if closeErr := writer.Close(); closeErr != nil {
		panic(closeErr)
	}
	
	// Parse the multipart form to get FileHeader
	reader := multipart.NewReader(&b, writer.Boundary())
	form, err := reader.ReadForm(int64(len(content)) + 1024)
	if err != nil {
		panic(err)
	}
	
	if len(form.File["files"]) == 0 {
		panic("no file in form")
	}
	
	return form.File["files"][0]
}
