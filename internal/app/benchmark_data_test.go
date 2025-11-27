package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectFileType(t *testing.T) {
	tests := []struct {
		name      string
		firstLine string
		want      int
	}{
		{
			name:      "MangoHud format",
			firstLine: "os,cpu,gpu,ram,kernel,driver,cpuscheduler",
			want:      FileTypeMangoHud,
		},
		{
			name:      "Afterburner format",
			firstLine: "Test, Hardware monitoring log v1.0",
			want:      FileTypeAfterburner,
		},
		{
			name:      "Unknown format",
			firstLine: "unknown,format,here",
			want:      FileTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectFileType(tt.firstLine)
			if got != tt.want {
				t.Errorf("detectFileType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "short string",
			input: "short",
			want:  "short",
		},
		{
			name:  "exact 100 chars",
			input: strings.Repeat("a", 100),
			want:  strings.Repeat("a", 100),
		},
		{
			name:  "over 100 chars",
			input: strings.Repeat("a", 150),
			want:  strings.Repeat("a", 100) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input)
			if got != tt.want {
				t.Errorf("truncateString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStorageOperations(t *testing.T) {
	// Setup temporary directory
	tmpDir := t.TempDir()
	if err := InitBenchmarksDir(tmpDir); err != nil {
		t.Fatalf("Failed to initialize benchmarks dir: %v", err)
	}

	// Create test data
	testData := []*BenchmarkData{
		{
			Label:   "test1",
			SpecOS:  "Linux",
			SpecCPU: "Test CPU",
			SpecGPU: "Test GPU",
			DataFPS: []float64{60.0, 59.5, 61.2},
		},
	}

	benchmarkID := uint(1)

	// Test storage
	if err := StoreBenchmarkData(testData, benchmarkID); err != nil {
		t.Fatalf("Failed to store benchmark data: %v", err)
	}

	// Test retrieval
	retrieved, err := RetrieveBenchmarkData(benchmarkID)
	if err != nil {
		t.Fatalf("Failed to retrieve benchmark data: %v", err)
	}

	if len(retrieved) != len(testData) {
		t.Errorf("Retrieved data length = %d, want %d", len(retrieved), len(testData))
	}

	if retrieved[0].Label != testData[0].Label {
		t.Errorf("Retrieved label = %s, want %s", retrieved[0].Label, testData[0].Label)
	}

	// Test deletion
	if deleteErr := DeleteBenchmarkData(benchmarkID); deleteErr != nil {
		t.Fatalf("Failed to delete benchmark data: %v", deleteErr)
	}

	// Verify deletion
	_, err = RetrieveBenchmarkData(benchmarkID)
	if err == nil {
		t.Error("Expected error when retrieving deleted data, got nil")
	}
}

func TestMetadataOperations(t *testing.T) {
	// Setup temporary directory
	tmpDir := t.TempDir()
	if err := InitBenchmarksDir(tmpDir); err != nil {
		t.Fatalf("Failed to initialize benchmarks dir: %v", err)
	}

	// Create test data with multiple runs
	testData := []*BenchmarkData{
		{
			Label:   "Run 1",
			SpecOS:  "Linux",
			SpecCPU: "Test CPU 1",
			DataFPS: []float64{60.0, 59.5, 61.2},
		},
		{
			Label:   "Run 2",
			SpecOS:  "Windows",
			SpecCPU: "Test CPU 2",
			DataFPS: []float64{55.0, 54.5, 56.2},
		},
		{
			Label:   "Run 3",
			SpecOS:  "macOS",
			SpecCPU: "Test CPU 3",
			DataFPS: []float64{50.0, 49.5, 51.2},
		},
	}

	benchmarkID := uint(100)

	// Store benchmark data (should also create metadata)
	if err := StoreBenchmarkData(testData, benchmarkID); err != nil {
		t.Fatalf("Failed to store benchmark data: %v", err)
	}

	// Test GetBenchmarkRunCount (should read from metadata file)
	count, labels, err := GetBenchmarkRunCount(benchmarkID)
	if err != nil {
		t.Fatalf("Failed to get run count: %v", err)
	}

	// Verify count
	if count != 3 {
		t.Errorf("Run count = %d, want 3", count)
	}

	// Verify labels
	expectedLabels := []string{"Run 1", "Run 2", "Run 3"}
	if len(labels) != len(expectedLabels) {
		t.Errorf("Labels length = %d, want %d", len(labels), len(expectedLabels))
	}
	for i, label := range labels {
		if label != expectedLabels[i] {
			t.Errorf("Label[%d] = %s, want %s", i, label, expectedLabels[i])
		}
	}

	// Verify that metadata file exists
	metaPath := filepath.Join(tmpDir, "benchmarks", "100.meta")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Error("Metadata file does not exist")
	}

	// Test deletion (should also delete metadata)
	if err := DeleteBenchmarkData(benchmarkID); err != nil {
		t.Fatalf("Failed to delete benchmark data: %v", err)
	}

	// Verify both files are deleted
	dataPath := filepath.Join(tmpDir, "benchmarks", "100.bin")
	if _, err := os.Stat(dataPath); !os.IsNotExist(err) {
		t.Error("Data file still exists after deletion")
	}
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Error("Metadata file still exists after deletion")
	}
}

func TestMetadataBackwardCompatibility(t *testing.T) {
	// Setup temporary directory
	tmpDir := t.TempDir()
	if err := InitBenchmarksDir(tmpDir); err != nil {
		t.Fatalf("Failed to initialize benchmarks dir: %v", err)
	}

	// Create test data
	testData := []*BenchmarkData{
		{
			Label:   "Legacy Run",
			SpecOS:  "Linux",
			DataFPS: []float64{60.0},
		},
	}

	benchmarkID := uint(200)

	// Store data with metadata first
	if err := StoreBenchmarkData(testData, benchmarkID); err != nil {
		t.Fatalf("Failed to store: %v", err)
	}

	// Delete the metadata file to simulate legacy data
	metaPath := filepath.Join(tmpDir, "benchmarks", "200.meta")
	if err := os.Remove(metaPath); err != nil {
		t.Fatalf("Failed to remove metadata: %v", err)
	}

	// Test GetBenchmarkRunCount should fall back to loading full data
	count, labels, err := GetBenchmarkRunCount(benchmarkID)
	if err != nil {
		t.Fatalf("Failed to get run count from legacy data: %v", err)
	}

	if count != 1 {
		t.Errorf("Run count = %d, want 1", count)
	}

	if len(labels) != 1 || labels[0] != "Legacy Run" {
		t.Errorf("Labels = %v, want [Legacy Run]", labels)
	}

	// Verify that metadata file was created by the fallback
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Error("Metadata file was not created by fallback mechanism")
	}
}

func TestReadBenchmarkFiles(t *testing.T) {
	// Create a test CSV file
	tmpFile := filepath.Join(t.TempDir(), "test.csv")
	content := `os,cpu,gpu,ram,kernel,driver,cpuscheduler
Linux,Test CPU,Test GPU,16000000,5.10.0,,performance
fps,frametime,cpu_load,gpu_load
60.0,16.67,50.0,80.0
59.5,16.81,51.0,81.0
61.2,16.34,49.0,79.0`

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// This test would require creating a multipart.FileHeader which is complex
	// In a real scenario, you'd use httptest to create proper file uploads
	t.Skip("Skipping file parsing test - requires complex multipart setup")
}
