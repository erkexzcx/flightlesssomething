package app

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestExportBenchmarkDataAsZip(t *testing.T) {
	// Create a test benchmark data
	testData := []*BenchmarkData{
		{
			Label:              "Test Run 1",
			SpecOS:             "Linux",
			SpecCPU:            "Intel i7",
			SpecGPU:            "NVIDIA RTX 3080",
			SpecRAM:            "16 GB",
			SpecLinuxKernel:    "5.15.0",
			SpecLinuxScheduler: "cfs",
			DataFPS:            []float64{60.5, 61.2, 59.8},
			DataFrameTime:      []float64{16.5, 16.3, 16.7},
			DataCPULoad:        []float64{45.2, 46.1, 44.8},
			DataGPULoad:        []float64{98.5, 99.1, 97.8},
		},
		{
			Label:         "Test Run 2",
			SpecOS:        "Windows",
			SpecGPU:       "AMD RX 6800",
			DataFPS:       []float64{55.1, 56.3},
			DataFrameTime: []float64{18.1, 17.8},
		},
	}

	// Create temp directory for benchmarks
	tempDir, err := os.MkdirTemp("", "benchmark_export_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Logf("Warning: failed to remove temp dir: %v", removeErr)
		}
	}()

	// Initialize benchmarks directory
	if initErr := InitBenchmarksDir(tempDir); initErr != nil {
		t.Fatalf("Failed to init benchmarks dir: %v", initErr)
	}

	// Store the benchmark data
	benchmarkID := uint(123)
	if storeErr := StoreBenchmarkData(testData, benchmarkID); storeErr != nil {
		t.Fatalf("Failed to store benchmark data: %v", storeErr)
	}

	// Export to ZIP
	var buf bytes.Buffer
	if exportErr := ExportBenchmarkDataAsZip(benchmarkID, &buf); exportErr != nil {
		t.Fatalf("Failed to export to ZIP: %v", exportErr)
	}

	// Verify ZIP contents
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Failed to read ZIP: %v", err)
	}

	// Should have 2 files
	if len(zipReader.File) != 2 {
		t.Errorf("Expected 2 files in ZIP, got %d", len(zipReader.File))
	}

	// Check first file
	if zipReader.File[0].Name != "Test_Run_1.csv" {
		t.Errorf("Expected filename 'Test_Run_1.csv', got '%s'", zipReader.File[0].Name)
	}

	// Read and verify CSV content
	file, err := zipReader.File[0].Open()
	if err != nil {
		t.Fatalf("Failed to open ZIP file: %v", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			t.Logf("Warning: failed to close file: %v", closeErr)
		}
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read file content: %v", err)
	}

	csvContent := string(content)

	// Check for MangoHud header
	if !strings.Contains(csvContent, "os,cpu,gpu,ram,kernel,driver,cpuscheduler") {
		t.Error("CSV missing MangoHud header")
	}

	// Check for specs
	if !strings.Contains(csvContent, "Linux") {
		t.Error("CSV missing OS spec")
	}
	if !strings.Contains(csvContent, "Intel i7") {
		t.Error("CSV missing CPU spec")
	}

	// Check for column headers
	if !strings.Contains(csvContent, "fps,frametime,cpu_load,gpu_load") {
		t.Error("CSV missing column headers")
	}

	// Check for data
	if !strings.Contains(csvContent, "60.5") {
		t.Error("CSV missing FPS data")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal_filename", "normal_filename"},
		{"file/with/slashes", "file_with_slashes"},
		{"file\\with\\backslashes", "file_with_backslashes"},
		{"file:with:colons", "file_with_colons"},
		{"file*with?special<chars>", "file_with_special_chars_"},
		{"  spaces  ", "spaces"},
		{"", "benchmark"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertRAMToKB(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"16 GB", "16777216"},
		{"8 GB", "8388608"},
		{"512 MB", "524288"},
		{"1024 KB", "1024"},
		{"12345", "12345"}, // Already a number
		{"invalid", ""},    // Invalid format
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertRAMToKB(tt.input)
			if result != tt.expected {
				t.Errorf("convertRAMToKB(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
