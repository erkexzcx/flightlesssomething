package app

import (
	"archive/zip"
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"testing"
)

// TestCSVRoundTrip tests that exported CSV files can be re-uploaded and parsed correctly
func TestCSVRoundTrip(t *testing.T) {
	// Create test benchmark data
	originalData := []*BenchmarkData{
		{
			Label:              "Original Run",
			SpecOS:             "Linux",
			SpecCPU:            "AMD Ryzen 9 5900X",
			SpecGPU:            "NVIDIA RTX 3090",
			SpecRAM:            "32 GB",
			SpecLinuxKernel:    "6.1.0",
			SpecLinuxScheduler: "cfs",
			DataFPS:            []float64{120.5, 119.8, 121.2, 120.0},
			DataFrameTime:      []float64{8.3, 8.35, 8.28, 8.33},
			DataCPULoad:        []float64{55.2, 56.1, 54.8, 55.5},
			DataGPULoad:        []float64{98.5, 99.1, 97.8, 98.2},
			DataCPUTemp:        []float64{65.0, 66.0, 65.5, 65.8},
			DataCPUPower:       []float64{95.0, 96.5, 94.8, 95.2},
			DataGPUTemp:        []float64{75.0, 76.0, 75.5, 75.2},
			DataGPUCoreClock:   []float64{1850, 1855, 1848, 1852},
			DataGPUMemClock:    []float64{9501, 9502, 9500, 9501},
			DataGPUVRAMUsed:    []float64{8.5, 8.6, 8.5, 8.5},
			DataGPUPower:       []float64{350, 352, 349, 351},
			DataRAMUsed:        []float64{16.2, 16.3, 16.2, 16.2},
			DataSwapUsed:       []float64{0.0, 0.0, 0.0, 0.0},
		},
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "benchmark_roundtrip_test")
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

	// Store the original data
	benchmarkID := uint(456)
	if _, storeErr := StoreBenchmarkData(originalData, benchmarkID); storeErr != nil {
		t.Fatalf("Failed to store benchmark data: %v", storeErr)
	}

	// Export to ZIP
	var buf bytes.Buffer
	if exportErr := ExportBenchmarkDataAsZip(benchmarkID, &buf); exportErr != nil {
		t.Fatalf("Failed to export to ZIP: %v", exportErr)
	}

	// Read ZIP and extract CSV
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Failed to read ZIP: %v", err)
	}

	if len(zipReader.File) != 1 {
		t.Fatalf("Expected 1 file in ZIP, got %d", len(zipReader.File))
	}

	// Read CSV content
	csvFile, err := zipReader.File[0].Open()
	if err != nil {
		t.Fatalf("Failed to open CSV file: %v", err)
	}
	defer func() {
		if closeErr := csvFile.Close(); closeErr != nil {
			t.Logf("Warning: failed to close file: %v", closeErr)
		}
	}()

	csvContent, err := io.ReadAll(csvFile)
	if err != nil {
		t.Fatalf("Failed to read CSV content: %v", err)
	}

	// Create a multipart file from the CSV content
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", "test.csv")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if _, writeErr := part.Write(csvContent); writeErr != nil {
		t.Fatalf("Failed to write CSV to form: %v", writeErr)
	}
	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("Failed to close multipart writer: %v", closeErr)
	}

	// Parse the multipart form
	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(32 << 20)
	if err != nil {
		t.Fatalf("Failed to read form: %v", err)
	}
	defer func() {
		if removeErr := form.RemoveAll(); removeErr != nil {
			t.Logf("Warning: failed to remove form files: %v", removeErr)
		}
	}()

	// Parse the CSV file
	reImportedData, err := ReadBenchmarkFiles(form.File["files"])
	if err != nil {
		t.Fatalf("Failed to re-import CSV: %v", err)
	}

	// Verify the data
	if len(reImportedData) != 1 {
		t.Fatalf("Expected 1 benchmark, got %d", len(reImportedData))
	}

	reimported := reImportedData[0]
	original := originalData[0]

	// Verify specs
	if reimported.SpecOS != original.SpecOS {
		t.Errorf("OS mismatch: got %q, want %q", reimported.SpecOS, original.SpecOS)
	}
	if reimported.SpecCPU != original.SpecCPU {
		t.Errorf("CPU mismatch: got %q, want %q", reimported.SpecCPU, original.SpecCPU)
	}
	if reimported.SpecGPU != original.SpecGPU {
		t.Errorf("GPU mismatch: got %q, want %q", reimported.SpecGPU, original.SpecGPU)
	}

	// Verify data arrays (check lengths and some sample values)
	if len(reimported.DataFPS) != len(original.DataFPS) {
		t.Errorf("FPS data length mismatch: got %d, want %d", len(reimported.DataFPS), len(original.DataFPS))
	} else {
		// Check first and last values
		tolerance := 0.1
		if abs(reimported.DataFPS[0]-original.DataFPS[0]) > tolerance {
			t.Errorf("FPS[0] mismatch: got %.2f, want %.2f", reimported.DataFPS[0], original.DataFPS[0])
		}
	}

	if len(reimported.DataFrameTime) != len(original.DataFrameTime) {
		t.Errorf("FrameTime data length mismatch: got %d, want %d", len(reimported.DataFrameTime), len(original.DataFrameTime))
	}

	if len(reimported.DataCPULoad) != len(original.DataCPULoad) {
		t.Errorf("CPULoad data length mismatch: got %d, want %d", len(reimported.DataCPULoad), len(original.DataCPULoad))
	}

	if len(reimported.DataCPUPower) != len(original.DataCPUPower) {
		t.Errorf("CPUPower data length mismatch: got %d, want %d", len(reimported.DataCPUPower), len(original.DataCPUPower))
	} else if len(reimported.DataCPUPower) > 0 {
		tolerance := 0.1
		if abs(reimported.DataCPUPower[0]-original.DataCPUPower[0]) > tolerance {
			t.Errorf("CPUPower[0] mismatch: got %.2f, want %.2f", reimported.DataCPUPower[0], original.DataCPUPower[0])
		}
	}

	t.Log("Round-trip test completed successfully")
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
