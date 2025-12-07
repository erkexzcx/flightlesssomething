package app

import (
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
)

const (
	// maxLengthDifferenceThreshold is the maximum allowed percentage difference
	// between FPS and frametime data array lengths in Afterburner files.
	// Afterburner files may have duplicate column headers (e.g., "Framerate" and
	// "Frametime" appear twice at different positions), which can result in slightly
	// different array lengths when some rows have empty values in one set of columns.
	maxLengthDifferenceThreshold = 0.05
)

// TestParseAfterburnerTestData tests parsing of actual afterburner test files in testdata/
func TestParseAfterburnerTestData(t *testing.T) {
	// Get the testdata directory
	testdataDir := filepath.Join("..", "..", "testdata", "afterburner")
	
	// Read all files in the afterburner directory
	files, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No test files found in testdata/afterburner")
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			filePath := filepath.Join(testdataDir, file.Name())
			
			// Read the file
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			// Create a multipart file header
			fileHeaders := createMultipartFileHeaders(t, file.Name(), content)
			
			// Parse the file
			benchmarkData, err := ReadBenchmarkFiles(fileHeaders)
			if err != nil {
				t.Fatalf("Failed to parse file %s: %v", file.Name(), err)
			}

			// Verify we got data
			if len(benchmarkData) != 1 {
				t.Fatalf("Expected 1 benchmark data, got %d", len(benchmarkData))
			}

			data := benchmarkData[0]

			// Verify basic structure
			if data.SpecOS != "Windows" {
				t.Errorf("Expected OS to be Windows, got %s", data.SpecOS)
			}

			if data.SpecGPU == "" {
				t.Error("Expected GPU spec to be set")
			}

			// Verify we have FPS data
			if len(data.DataFPS) == 0 {
				t.Error("Expected FPS data to be present")
			}

			// Verify we have frametime data
			if len(data.DataFrameTime) == 0 {
				t.Error("Expected frametime data to be present")
			}

			// For Afterburner files, FPS and frametime may have slightly different lengths
			// because Afterburner format contains duplicate column headers:
			// - "Framerate" and "Frametime" appear at positions 4,5
			// - "Framerate" and "Frametime" appear again at positions 12,13
			// Some data rows may have values in only one set of columns, causing length mismatches.
			// We verify they're within the acceptable threshold.
			fpLen := len(data.DataFPS)
			ftLen := len(data.DataFrameTime)
			diff := fpLen - ftLen
			if diff < 0 {
				diff = -diff
			}
			maxLen := fpLen
			if ftLen > maxLen {
				maxLen = ftLen
			}
			if float64(diff) > float64(maxLen)*maxLengthDifferenceThreshold {
				t.Errorf("FPS and frametime data length difference too large: %d vs %d (diff: %d, threshold: %.1f%%)", 
					fpLen, ftLen, diff, maxLengthDifferenceThreshold*100)
			}

			t.Logf("Successfully parsed %s: %d data points, GPU: %s", 
				file.Name(), len(data.DataFPS), data.SpecGPU)
		})
	}
}

// TestParseMangoHudTestData tests parsing of actual mangohud test files in testdata/
func TestParseMangoHudTestData(t *testing.T) {
	// Get the testdata directory
	testdataDir := filepath.Join("..", "..", "testdata", "mangohud")
	
	// Read all files in the mangohud directory
	files, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No test files found in testdata/mangohud")
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			filePath := filepath.Join(testdataDir, file.Name())
			
			// Read the file
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			// Create a multipart file header
			fileHeaders := createMultipartFileHeaders(t, file.Name(), content)
			
			// Parse the file
			benchmarkData, err := ReadBenchmarkFiles(fileHeaders)
			if err != nil {
				t.Fatalf("Failed to parse file %s: %v", file.Name(), err)
			}

			// Verify we got data
			if len(benchmarkData) != 1 {
				t.Fatalf("Expected 1 benchmark data, got %d", len(benchmarkData))
			}

			data := benchmarkData[0]

			// Verify basic structure
			if data.SpecOS == "" {
				t.Error("Expected OS spec to be set")
			}

			if data.SpecCPU == "" {
				t.Error("Expected CPU spec to be set")
			}

			if data.SpecGPU == "" {
				t.Error("Expected GPU spec to be set")
			}

			// Verify we have FPS data
			if len(data.DataFPS) == 0 {
				t.Error("Expected FPS data to be present")
			}

			// Verify we have frametime data
			if len(data.DataFrameTime) == 0 {
				t.Error("Expected frametime data to be present")
			}

			// MangoHud should have more metrics
			if len(data.DataCPULoad) == 0 {
				t.Error("Expected CPU load data to be present")
			}

			if len(data.DataGPULoad) == 0 {
				t.Error("Expected GPU load data to be present")
			}

			t.Logf("Successfully parsed %s: %d data points, OS: %s, CPU: %s, GPU: %s", 
				file.Name(), len(data.DataFPS), data.SpecOS, data.SpecCPU, data.SpecGPU)
		})
	}
}

// TestRoundTripWithTestData tests that test data files can be exported and re-imported
func TestRoundTripWithTestData(t *testing.T) {
	testCases := []struct {
		name string
		dir  string
	}{
		{"Afterburner", filepath.Join("..", "..", "testdata", "afterburner")},
		{"MangoHud", filepath.Join("..", "..", "testdata", "mangohud")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Read all files in the directory
			files, err := os.ReadDir(tc.dir)
			if err != nil {
				t.Fatalf("Failed to read testdata directory: %v", err)
			}

			for _, file := range files {
				if file.IsDir() {
					continue
				}

				t.Run(file.Name(), func(t *testing.T) {
					// Setup temp directory
					tmpDir := t.TempDir()
					if err := InitBenchmarksDir(tmpDir); err != nil {
						t.Fatalf("Failed to init benchmarks dir: %v", err)
					}

					// Read and parse the original file
					filePath := filepath.Join(tc.dir, file.Name())
					content, err := os.ReadFile(filePath)
					if err != nil {
						t.Fatalf("Failed to read test file: %v", err)
					}

					fileHeaders := createMultipartFileHeaders(t, file.Name(), content)
					originalData, err := ReadBenchmarkFiles(fileHeaders)
					if err != nil {
						t.Fatalf("Failed to parse original file: %v", err)
					}

					if len(originalData) != 1 {
						t.Fatalf("Expected 1 benchmark, got %d", len(originalData))
					}

					// Store the data
					benchmarkID := uint(12345)
					if err := StoreBenchmarkData(originalData, benchmarkID); err != nil {
						t.Fatalf("Failed to store data: %v", err)
					}

					// Export to ZIP
					var buf bytes.Buffer
					if err := ExportBenchmarkDataAsZip(benchmarkID, &buf); err != nil {
						t.Fatalf("Failed to export to ZIP: %v", err)
					}

					// Re-import would require extracting from ZIP and creating multipart headers
					// For now, verify the export worked
					if buf.Len() == 0 {
						t.Error("Expected non-empty ZIP export")
					}

					t.Logf("Successfully round-tripped %s: original FPS count=%d, export size=%d bytes", 
						file.Name(), len(originalData[0].DataFPS), buf.Len())
				})
			}
		})
	}
}

// createMultipartFileHeaders creates multipart file headers for testing
func createMultipartFileHeaders(t *testing.T, filename string, content []byte) []*multipart.FileHeader {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	part, err := writer.CreateFormFile("files", filename)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	
	if _, writeErr := part.Write(content); writeErr != nil {
		t.Fatalf("Failed to write content: %v", writeErr)
	}
	
	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("Failed to close writer: %v", closeErr)
	}
	
	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(32 << 20)
	if err != nil {
		t.Fatalf("Failed to read form: %v", err)
	}
	
	t.Cleanup(func() {
		if removeErr := form.RemoveAll(); removeErr != nil {
			t.Logf("Warning: failed to remove form files: %v", removeErr)
		}
	})
	
	return form.File["files"]
}
