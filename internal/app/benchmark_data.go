package app

import (
	"archive/zip"
	"bufio"
	"encoding/csv"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/klauspost/compress/zstd"
)

const (
	FileTypeUnknown = iota
	FileTypeMangoHud
	FileTypeAfterburner

	// Data processing constants
	precisionFactor     = 100000
	bytesToKB           = 1024
	maxTotalDataLines   = 1000000 // Total limit across all runs in a benchmark
	maxPerRunDataLines  = 500000  // Maximum data lines per single run
	maxStringLength     = 100
	
	// Storage format version for backward compatibility
	storageFormatVersion = 2 // Version 2: Streaming-friendly format with individual run encoding
	
	// GC tuning constants for streaming operations
	// These control how often runtime.GC() is called during streaming to aggressively reclaim memory
	gcFrequencyStreaming = 10 // Trigger GC every N runs during JSON streaming (viewing benchmarks)
	gcFrequencyExport    = 5  // Trigger GC every N runs during ZIP export (more aggressive due to CSV overhead)
)

var benchmarksDir string

// fileHeader is written at the beginning of the benchmark data file
type fileHeader struct {
	Version  int // Storage format version
	RunCount int // Number of runs in this file
}

// InitBenchmarksDir initializes the directory for benchmark data
func InitBenchmarksDir(dataDir string) error {
	benchmarksDir = filepath.Join(dataDir, "benchmarks")
	return os.MkdirAll(benchmarksDir, 0o750)
}

// parseHeader parses the CSV header line and returns a map of column names to indices
func parseHeader(scanner *bufio.Scanner) (map[int]string, error) {
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read header line: %w", err)
		}
		return nil, errors.New("unexpected end of file while reading header line")
	}

	headerLine := scanner.Text()
	headers := strings.Split(headerLine, ",")

	headerMap := make(map[int]string)
	for i, header := range headers {
		headerMap[i] = strings.TrimSpace(header)
	}

	return headerMap, nil
}

// parseData parses the data lines from the CSV file
func parseData(scanner *bufio.Scanner, headerMap map[int]string, benchmarkData *BenchmarkData, isAfterburner bool, expectedLines int) error {
	counter := 0
	
	// Pre-allocate slices with EXACT capacity based on actual line count
	// Two-pass approach: first pass counts lines (streaming, no memory storage),
	// second pass parses with exact pre-allocation to achieve 100% accuracy.
	// This eliminates all reallocation overhead while keeping memory usage minimal.
	capacity := expectedLines
	if capacity < 0 {
		capacity = 0 // Safety check
	}
	
	benchmarkData.DataFPS = make([]float64, 0, capacity)
	benchmarkData.DataFrameTime = make([]float64, 0, capacity)
	benchmarkData.DataCPULoad = make([]float64, 0, capacity)
	benchmarkData.DataGPULoad = make([]float64, 0, capacity)
	benchmarkData.DataCPUTemp = make([]float64, 0, capacity)
	benchmarkData.DataCPUPower = make([]float64, 0, capacity)
	benchmarkData.DataGPUTemp = make([]float64, 0, capacity)
	benchmarkData.DataGPUCoreClock = make([]float64, 0, capacity)
	benchmarkData.DataGPUMemClock = make([]float64, 0, capacity)
	benchmarkData.DataGPUVRAMUsed = make([]float64, 0, capacity)
	benchmarkData.DataGPUPower = make([]float64, 0, capacity)
	benchmarkData.DataRAMUsed = make([]float64, 0, capacity)
	benchmarkData.DataSwapUsed = make([]float64, 0, capacity)

	for scanner.Scan() {
		line := scanner.Text()
		record := strings.Split(line, ",")

		for i, valStr := range record {
			colName := headerMap[i]
			val, err := strconv.ParseFloat(strings.TrimSpace(valStr), 64)
			if err != nil {
				continue
			}

			switch colName {
			case "fps", "Framerate":
				benchmarkData.DataFPS = append(benchmarkData.DataFPS, val)
			case "frametime", "Frametime":
				benchmarkData.DataFrameTime = append(benchmarkData.DataFrameTime, val)
			case "cpu_load", "CPU usage":
				benchmarkData.DataCPULoad = append(benchmarkData.DataCPULoad, val)
			case "gpu_load", "GPU usage":
				benchmarkData.DataGPULoad = append(benchmarkData.DataGPULoad, val)
			case "cpu_temp", "CPU temperature":
				benchmarkData.DataCPUTemp = append(benchmarkData.DataCPUTemp, val)
			case "cpu_power":
				benchmarkData.DataCPUPower = append(benchmarkData.DataCPUPower, val)
			case "gpu_temp", "GPU temperature":
				benchmarkData.DataGPUTemp = append(benchmarkData.DataGPUTemp, val)
			case "gpu_core_clock", "Core clock":
				benchmarkData.DataGPUCoreClock = append(benchmarkData.DataGPUCoreClock, val)
			case "gpu_mem_clock", "Memory clock":
				if isAfterburner {
					val = math.Round((val/2)*precisionFactor) / precisionFactor
				}
				benchmarkData.DataGPUMemClock = append(benchmarkData.DataGPUMemClock, val)
			case "gpu_vram_used", "Memory usage":
				if isAfterburner {
					val = math.Round(val/bytesToKB*precisionFactor) / precisionFactor
				}
				benchmarkData.DataGPUVRAMUsed = append(benchmarkData.DataGPUVRAMUsed, val)
			case "gpu_power", "Power":
				benchmarkData.DataGPUPower = append(benchmarkData.DataGPUPower, val)
			case "ram_used", "RAM usage":
				if isAfterburner {
					val = math.Round(val/bytesToKB*precisionFactor) / precisionFactor
				}
				benchmarkData.DataRAMUsed = append(benchmarkData.DataRAMUsed, val)
			case "swap_used":
				benchmarkData.DataSwapUsed = append(benchmarkData.DataSwapUsed, val)
			}
		}

		counter++
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if len(benchmarkData.DataFPS) == 0 &&
		len(benchmarkData.DataFrameTime) == 0 &&
		len(benchmarkData.DataCPULoad) == 0 &&
		len(benchmarkData.DataGPULoad) == 0 &&
		len(benchmarkData.DataCPUTemp) == 0 &&
		len(benchmarkData.DataCPUPower) == 0 &&
		len(benchmarkData.DataGPUTemp) == 0 &&
		len(benchmarkData.DataGPUCoreClock) == 0 &&
		len(benchmarkData.DataGPUMemClock) == 0 &&
		len(benchmarkData.DataGPUVRAMUsed) == 0 &&
		len(benchmarkData.DataGPUPower) == 0 &&
		len(benchmarkData.DataRAMUsed) == 0 &&
		len(benchmarkData.DataSwapUsed) == 0 {
		return errors.New("no valid benchmark data found in file (all data columns are empty)")
	}

	return nil
}

// readBenchmarkFile reads a single benchmark file
func readBenchmarkFile(scanner *bufio.Scanner, fileType, totalLines int) (*BenchmarkData, error) {
	benchmarkData := &BenchmarkData{}

	// Second line should contain specs
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read specs line: %w", err)
		}
		return nil, errors.New("unexpected end of file while reading specs line")
	}
	record := strings.Split(scanner.Text(), ",")
	switch fileType {
	case FileTypeAfterburner:
		if len(record) < 3 {
			return nil, errors.New("invalid specs line format")
		}
		benchmarkData.SpecOS = "Windows"
		benchmarkData.SpecGPU = truncateString(strings.TrimSpace(record[2]))
	case FileTypeMangoHud:
		for i, v := range record {
			switch i {
			case 0:
				benchmarkData.SpecOS = truncateString(strings.TrimSpace(v))
			case 1:
				benchmarkData.SpecCPU = truncateString(strings.TrimSpace(v))
			case 2:
				benchmarkData.SpecGPU = truncateString(strings.TrimSpace(v))
			case 3:
				kilobytes := new(big.Int)
				_, ok := kilobytes.SetString(strings.TrimSpace(v), 10)
				if ok {
					ramBytes := new(big.Int).Mul(kilobytes, big.NewInt(1024))
					benchmarkData.SpecRAM = humanize.Bytes(ramBytes.Uint64())
				} else {
					benchmarkData.SpecRAM = truncateString(strings.TrimSpace(v))
				}
			case 4:
				benchmarkData.SpecLinuxKernel = truncateString(strings.TrimSpace(v))
			case 6:
				benchmarkData.SpecLinuxScheduler = truncateString(strings.TrimSpace(v))
			}
		}
	}

	headerMap, err := parseHeader(scanner)
	if err != nil {
		return nil, err
	}

	if fileType == FileTypeAfterburner {
		// Skip len(headerMap) amount of lines
		for i := 0; i < len(headerMap); i++ {
			if !scanner.Scan() {
				if scanErr := scanner.Err(); scanErr != nil {
					return nil, fmt.Errorf("failed to skip afterburner header lines: %w", scanErr)
				}
				return nil, fmt.Errorf("unexpected end of file while skipping afterburner header lines (expected %d lines, got %d)", len(headerMap), i)
			}
		}
	}

	// Calculate expected data lines for pre-allocation
	// totalLines is EXACT count from first pass (streaming line count)
	// Subtract header lines to get data line count for array pre-allocation
	// Total lines - first line (format) - specs line - header line - afterburner extra headers
	dataLines := totalLines - 3 // format, specs, header
	if fileType == FileTypeAfterburner {
		dataLines -= len(headerMap) // additional header lines for afterburner
	}
	// Ensure we have a reasonable minimum for pre-allocation
	if dataLines < 0 {
		dataLines = 100 // Fallback minimum
	}
	
	err = parseData(scanner, headerMap, benchmarkData, fileType == FileTypeAfterburner, dataLines)
	if err != nil {
		return nil, err
	}

	return benchmarkData, nil
}

// detectFileType detects the type of benchmark file from the first line
func detectFileType(firstLine string) int {
	switch {
	case firstLine == "os,cpu,gpu,ram,kernel,driver,cpuscheduler":
		return FileTypeMangoHud
	case strings.Contains(firstLine, ", Hardware monitoring log v"):
		return FileTypeAfterburner
	default:
		return FileTypeUnknown
	}
}

// ReadBenchmarkFiles reads and parses multiple benchmark files
func ReadBenchmarkFiles(files []*multipart.FileHeader) ([]*BenchmarkData, error) {
	// Pre-allocate slice with exact capacity to avoid reallocations
	benchmarkDatas := make([]*BenchmarkData, 0, len(files))

	for _, fileHeader := range files {
		benchmarkData, err := readSingleBenchmarkFile(fileHeader)
		if err != nil {
			return nil, fmt.Errorf("file '%s': %w", fileHeader.Filename, err)
		}
		benchmarkDatas = append(benchmarkDatas, benchmarkData)
	}

	return benchmarkDatas, nil
}

func readSingleBenchmarkFile(fileHeader *multipart.FileHeader) (*BenchmarkData, error) {
	// PASS 1: Count total lines in the file for 100% accurate pre-allocation
	// This pass streams through the file without storing content, keeping memory usage minimal
	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	
	lineCount := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineCount++
	}
	if scanErr := scanner.Err(); scanErr != nil {
		_ = file.Close() //nolint:errcheck // Error from counting, will report scan error
		return nil, fmt.Errorf("failed to count lines: %w", scanErr)
	}
	
	// Close file after first pass
	if closeErr := file.Close(); closeErr != nil {
		return nil, fmt.Errorf("failed to close file after counting: %w", closeErr)
	}
	
	// PASS 2: Reopen file and parse with exact pre-allocation
	file, err = fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log error but continue - this is cleanup
			fmt.Printf("Warning: failed to close file: %v\n", closeErr)
		}
	}()
	
	reader := bufio.NewReader(file)
	scanner = bufio.NewScanner(reader)

	// First line identifies file format
	if !scanner.Scan() {
		if scanErr := scanner.Err(); scanErr != nil {
			return nil, fmt.Errorf("failed to read first line: %w", scanErr)
		}
		return nil, errors.New("file is empty or failed to read first line")
	}
	firstLine := scanner.Text()
	firstLine = strings.TrimRight(firstLine, ", ")
	firstLine = strings.TrimSpace(firstLine)

	fileType := detectFileType(firstLine)
	if fileType == FileTypeUnknown {
		return nil, fmt.Errorf("unsupported file format (expected MangoHud CSV or Afterburner HML, got: '%.50s...')", firstLine)
	}

	// Use exact line count for 100% accurate pre-allocation (no reallocation needed)
	benchmarkData, err := readBenchmarkFile(scanner, fileType, lineCount)
	if err != nil {
		return nil, err
	}

	var suffix string
	switch fileType {
	case FileTypeMangoHud:
		suffix = ".csv"
	case FileTypeAfterburner:
		suffix = ".hml"
	}

	benchmarkData.Label = strings.TrimSuffix(fileHeader.Filename, suffix)
	return benchmarkData, nil
}

// ReadBenchmarkCSVContent parses benchmark CSV content from a string (for MCP tool usage).
// The label parameter sets the run label. The content should be MangoHud CSV or Afterburner HML format.
func ReadBenchmarkCSVContent(content, label string) (*BenchmarkData, error) {
	// Count lines for pre-allocation
	lineCount := strings.Count(content, "\n") + 1

	reader := strings.NewReader(content)
	scanner := bufio.NewScanner(reader)

	// First line identifies file format
	if !scanner.Scan() {
		if scanErr := scanner.Err(); scanErr != nil {
			return nil, fmt.Errorf("failed to read first line: %w", scanErr)
		}
		return nil, errors.New("content is empty or failed to read first line")
	}
	firstLine := scanner.Text()
	firstLine = strings.TrimRight(firstLine, ", ")
	firstLine = strings.TrimSpace(firstLine)

	fileType := detectFileType(firstLine)
	if fileType == FileTypeUnknown {
		return nil, fmt.Errorf("unsupported file format (expected MangoHud CSV or Afterburner HML, got: '%.50s...')", firstLine)
	}

	benchmarkData, err := readBenchmarkFile(scanner, fileType, lineCount)
	if err != nil {
		return nil, err
	}

	benchmarkData.Label = label
	return benchmarkData, nil
}

// truncateString truncates the input string to maxStringLength characters
func truncateString(s string) string {
	if len(s) > maxStringLength {
		return s[:maxStringLength] + "..."
	}
	return s
}

// StoreBenchmarkData stores benchmark data to disk in streaming-friendly format
// Format: [zstd compressed: [header: version, run_count] [run1] [run2] ... [runN]]
// This allows streaming reads without loading entire dataset into memory
func StoreBenchmarkData(benchmarkData []*BenchmarkData, benchmarkID uint) error {
	// Store the full data
	filePath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.bin", benchmarkID))
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log error but continue - this is cleanup
			fmt.Printf("Warning: failed to close file: %v\n", closeErr)
		}
	}()

	// Use buffered writer to reduce syscalls and improve write performance
	// 256KB buffer is large enough for efficient I/O without excessive memory use
	bufWriter := bufio.NewWriterSize(file, 256*1024)
	
	// Use higher compression concurrency and better compression level for storage
	// SpeedDefault provides good balance between compression ratio and speed
	zstdEncoder, err := zstd.NewWriter(bufWriter, 
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderConcurrency(2))
	if err != nil {
		return err
	}

	gobEncoder := gob.NewEncoder(zstdEncoder)
	
	// Write file header with version and run count
	// Pre-allocated struct avoids allocation during encoding
	header := fileHeader{
		Version:  storageFormatVersion,
		RunCount: len(benchmarkData),
	}
	if err := gobEncoder.Encode(&header); err != nil {
		if closeErr := zstdEncoder.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close zstd encoder after header encode error: %v\n", closeErr)
		}
		return fmt.Errorf("failed to encode header: %w", err)
	}
	
	// Write each run separately (enables streaming reads)
	for i, run := range benchmarkData {
		if err := gobEncoder.Encode(run); err != nil {
			if closeErr := zstdEncoder.Close(); closeErr != nil {
				fmt.Printf("Warning: failed to close zstd encoder after encode error: %v\n", closeErr)
			}
			return fmt.Errorf("failed to encode run %d: %w", i, err)
		}
	}

	// Ensure all data is flushed before metadata write
	if err := zstdEncoder.Close(); err != nil {
		return fmt.Errorf("failed to close zstd encoder: %w", err)
	}
	
	// Flush buffered writer to disk
	if err := bufWriter.Flush(); err != nil {
		return fmt.Errorf("failed to flush buffer: %w", err)
	}

	// Store metadata separately for fast access
	return storeBenchmarkMetadata(benchmarkData, benchmarkID)
}

// storeBenchmarkMetadata stores lightweight metadata (run count and labels) separately
func storeBenchmarkMetadata(benchmarkData []*BenchmarkData, benchmarkID uint) error {
	labels := make([]string, len(benchmarkData))
	for i, data := range benchmarkData {
		labels[i] = data.Label
	}

	metadata := BenchmarkMetadata{
		RunCount:  len(benchmarkData),
		RunLabels: labels,
	}

	metaPath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.meta", benchmarkID))
	metaFile, err := os.Create(metaPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := metaFile.Close(); err != nil {
			// Log error but continue - this is cleanup
			fmt.Printf("Warning: failed to close metaFile: %v\n", err)
		}
	}()

	// Use gob encoding for metadata (no need for compression, it's tiny)
	gobEncoder := gob.NewEncoder(metaFile)
	return gobEncoder.Encode(metadata)
}

// RetrieveBenchmarkData retrieves benchmark data from disk
// Supports both old format (version 1: single array) and new format (version 2: streaming)
func RetrieveBenchmarkData(benchmarkID uint) ([]*BenchmarkData, error) {
	filePath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.bin", benchmarkID))
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {

		if closeErr := file.Close(); closeErr != nil {

			// Log error but continue - this is cleanup

			fmt.Printf("Warning: failed to close file: %v\n", closeErr)

		}

	}()

	// Use concurrent decompression for better performance with large files
	zstdDecoder, err := zstd.NewReader(file, zstd.WithDecoderConcurrency(2))
	if err != nil {
		return nil, err
	}
	defer zstdDecoder.Close()

	gobDecoder := gob.NewDecoder(zstdDecoder)
	
	// Try to read header first
	var header fileHeader
	if err := gobDecoder.Decode(&header); err != nil {
		// If header decode fails, this might be old format (version 1)
		// Reopen and try old format
		return retrieveBenchmarkDataLegacy(benchmarkID)
	}
	
	// Check version
	if header.Version == 1 {
		// Old format: single array (shouldn't happen as old files don't have headers, but handle it)
		return retrieveBenchmarkDataLegacy(benchmarkID)
	} else if header.Version != storageFormatVersion {
		return nil, fmt.Errorf("unsupported storage format version: %d", header.Version)
	}
	
	// New format (version 2): read runs individually
	benchmarkData := make([]*BenchmarkData, header.RunCount)
	for i := 0; i < header.RunCount; i++ {
		var run BenchmarkData
		if err := gobDecoder.Decode(&run); err != nil {
			return nil, fmt.Errorf("failed to decode run %d: %w", i, err)
		}
		benchmarkData[i] = &run
	}
	
	return benchmarkData, nil
}

// retrieveBenchmarkDataLegacy reads data in the old format (version 1: single array)
func retrieveBenchmarkDataLegacy(benchmarkID uint) ([]*BenchmarkData, error) {
	filePath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.bin", benchmarkID))
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close file: %v\n", closeErr)
		}
	}()

	zstdDecoder, err := zstd.NewReader(file, zstd.WithDecoderConcurrency(2))
	if err != nil {
		return nil, err
	}
	defer zstdDecoder.Close()

	// Stream directly to gob decoder to avoid intermediate buffer allocation
	var benchmarkData []*BenchmarkData
	gobDecoder := gob.NewDecoder(zstdDecoder)
	err = gobDecoder.Decode(&benchmarkData)
	return benchmarkData, err
}

// StreamBenchmarkDataAsJSON streams benchmark data directly to the writer as JSON
// This uses true streaming to minimize memory usage - loads and encodes one run at a time
func StreamBenchmarkDataAsJSON(benchmarkID uint, w http.ResponseWriter) error {
	// Set response headers
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	
	// Open the file and prepare for streaming reads
	filePath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.bin", benchmarkID))
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close file: %v\n", closeErr)
		}
	}()

	// Setup decompression
	zstdDecoder, err := zstd.NewReader(file, zstd.WithDecoderConcurrency(2))
	if err != nil {
		return err
	}
	defer zstdDecoder.Close()

	gobDecoder := gob.NewDecoder(zstdDecoder)
	
	// Read header
	var header fileHeader
	if err := gobDecoder.Decode(&header); err != nil {
		// Fall back to legacy format (loads all into memory)
		benchmarkData, legacyErr := retrieveBenchmarkDataLegacy(benchmarkID)
		if legacyErr != nil {
			return legacyErr
		}
		return json.NewEncoder(w).Encode(benchmarkData)
	}
	
	// Check version
	if header.Version != storageFormatVersion {
		// For unsupported versions, fall back to RetrieveBenchmarkData
		benchmarkData, err := RetrieveBenchmarkData(benchmarkID)
		if err != nil {
			return err
		}
		return json.NewEncoder(w).Encode(benchmarkData)
	}
	
	// Stream the JSON array manually to match json.NewEncoder output exactly
	// Format: [{...},{...}]\n (compact, no whitespace between elements)
	
	// Write opening bracket
	if _, err := w.Write([]byte("[")); err != nil {
		return err
	}
	
	// Stream each run individually
	for i := 0; i < header.RunCount; i++ {
		var run BenchmarkData
		if err := gobDecoder.Decode(&run); err != nil {
			return fmt.Errorf("failed to decode run %d: %w", i, err)
		}
		
		// Add comma separator before this element (except for first element)
		if i > 0 {
			if _, err := w.Write([]byte(",")); err != nil {
				return err
			}
		}
		
		// Encode this run to JSON (compact format, no newlines)
		jsonBytes, err := json.Marshal(&run)
		if err != nil {
			return fmt.Errorf("failed to encode run %d: %w", i, err)
		}
		
		if _, err := w.Write(jsonBytes); err != nil {
			return err
		}
		
		// Trigger GC periodically to release memory from encoded runs
		if (i+1)%gcFrequencyStreaming == 0 {
			runtime.GC()
		}
	}
	
	// Write closing bracket and final newline (to match json.Encoder)
	if _, err := w.Write([]byte("]\n")); err != nil {
		return err
	}
	
	return nil
}

// GetBenchmarkRunCount returns the count of runs and their labels for a benchmark
// This is optimized to read only metadata without loading the full benchmark data
func GetBenchmarkRunCount(benchmarkID uint) (int, []string, error) {
	count, labels, _, err := GetBenchmarkMetadata(benchmarkID)
	return count, labels, err
}

// GetBenchmarkMetadata returns the full metadata for a benchmark
// This is optimized to read only metadata without loading the full benchmark data
func GetBenchmarkMetadata(benchmarkID uint) (int, []string, *BenchmarkMetadata, error) {
	metaPath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.meta", benchmarkID))
	metaFile, err := os.Open(metaPath)
	if err != nil {
		// Fallback: if metadata doesn't exist, load full data (backward compatibility)
		if os.IsNotExist(err) {
			benchmarkData, retrieveErr := RetrieveBenchmarkData(benchmarkID)
			if retrieveErr != nil {
				return 0, nil, nil, retrieveErr
			}

			labels := make([]string, len(benchmarkData))
			for i, data := range benchmarkData {
				labels[i] = data.Label
			}

			// Try to create metadata file for future use
			if storeErr := storeBenchmarkMetadata(benchmarkData, benchmarkID); storeErr != nil {
				// Log but don't fail - this is just an optimization
				fmt.Printf("Warning: failed to store benchmark metadata: %v\n", storeErr)
			}

			return len(benchmarkData), labels, nil, nil
		}
		return 0, nil, nil, err
	}
	defer func() {
		if closeErr := metaFile.Close(); closeErr != nil {
			// Log error but continue - this is cleanup
			fmt.Printf("Warning: failed to close metaFile: %v\n", closeErr)
		}
	}()

	var metadata BenchmarkMetadata
	gobDecoder := gob.NewDecoder(metaFile)
	err = gobDecoder.Decode(&metadata)
	if err != nil {
		return 0, nil, nil, err
	}

	return metadata.RunCount, metadata.RunLabels, &metadata, nil
}

// ExtractSearchableMetadata extracts run names and specifications from benchmark data for searching.
// It combines run labels from all runs into a comma-separated string and collects unique
// specifications (OS, CPU, GPU, RAM, kernel, scheduler) across all runs into another
// comma-separated string sorted alphabetically for deterministic output.
//
// Parameters:
//   - benchmarkData: slice of BenchmarkData pointers containing run information
//
// Returns:
//   - runNames: comma-separated string of unique run labels (e.g., "run1, run2, run3")
//   - specifications: comma-separated string of unique specifications sorted alphabetically
func ExtractSearchableMetadata(benchmarkData []*BenchmarkData) (runNames, specifications string) {
	// Extract run names - use a set to deduplicate
	runLabelSet := make(map[string]bool)
	var runLabelsOrdered []string
	for _, data := range benchmarkData {
		if data.Label != "" && !runLabelSet[data.Label] {
			runLabelSet[data.Label] = true
			runLabelsOrdered = append(runLabelsOrdered, data.Label)
		}
	}
	runNames = strings.Join(runLabelsOrdered, ", ")
	
	// Extract specifications - collect unique values from all runs
	specSet := make(map[string]bool)
	for _, data := range benchmarkData {
		if data.SpecOS != "" {
			specSet[data.SpecOS] = true
		}
		if data.SpecCPU != "" {
			specSet[data.SpecCPU] = true
		}
		if data.SpecGPU != "" {
			specSet[data.SpecGPU] = true
		}
		if data.SpecRAM != "" {
			specSet[data.SpecRAM] = true
		}
		if data.SpecLinuxKernel != "" {
			specSet[data.SpecLinuxKernel] = true
		}
		if data.SpecLinuxScheduler != "" {
			specSet[data.SpecLinuxScheduler] = true
		}
	}
	
	// Convert set to slice and sort for deterministic output
	specs := make([]string, 0, len(specSet))
	for spec := range specSet {
		specs = append(specs, spec)
	}
	// Sort alphabetically for consistent ordering
	sort.Strings(specs)
	specifications = strings.Join(specs, ", ")
	
	return runNames, specifications
}

// getRunDataPointCount returns the number of data lines in a single benchmark run.
// It returns the maximum length among all data arrays, as that represents the number of data rows.
func getRunDataPointCount(data *BenchmarkData) int {
	dataArrays := [][]float64{
		data.DataFPS,
		data.DataFrameTime,
		data.DataCPULoad,
		data.DataGPULoad,
		data.DataCPUTemp,
		data.DataCPUPower,
		data.DataGPUTemp,
		data.DataGPUCoreClock,
		data.DataGPUMemClock,
		data.DataGPUVRAMUsed,
		data.DataGPUPower,
		data.DataRAMUsed,
		data.DataSwapUsed,
	}
	maxLen := 0
	for _, arr := range dataArrays {
		if len(arr) > maxLen {
			maxLen = len(arr)
		}
	}
	return maxLen
}

// CountTotalDataLines counts the total number of data lines across all benchmark runs.
// It returns the maximum length among all data arrays, as that represents the number of data rows.
func CountTotalDataLines(benchmarkData []*BenchmarkData) int {
	totalLines := 0
	for _, data := range benchmarkData {
		totalLines += getRunDataPointCount(data)
	}
	return totalLines
}

// ValidatePerRunDataLines validates that no single run exceeds the per-run data line limit.
// Returns an error if any run exceeds maxPerRunDataLines.
func ValidatePerRunDataLines(benchmarkData []*BenchmarkData) error {
	for i, data := range benchmarkData {
		runDataPoints := getRunDataPointCount(data)
		
		if runDataPoints > maxPerRunDataLines {
			runLabel := data.Label
			if runLabel == "" {
				runLabel = fmt.Sprintf("run #%d", i+1)
			}
			return fmt.Errorf("%s has %d data lines, which exceeds the maximum allowed %d per run", runLabel, runDataPoints, maxPerRunDataLines)
		}
	}
	return nil
}

// DeleteBenchmarkData deletes benchmark data file and metadata from disk
func DeleteBenchmarkData(benchmarkID uint) error {
	filePath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.bin", benchmarkID))
	metaPath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.meta", benchmarkID))

	// Delete the main data file
	err := os.Remove(filePath)

	// Try to delete metadata file, only ignore error if file doesn't exist
	if metaErr := os.Remove(metaPath); metaErr != nil && !os.IsNotExist(metaErr) {
		// Log non-existence errors (but don't fail the operation)
		fmt.Printf("Warning: failed to delete metadata file %s: %v\n", metaPath, metaErr)
	}

	return err
}

// ExportBenchmarkDataAsZip exports benchmark data as a ZIP file containing CSV files
// Uses streaming to minimize memory usage
func ExportBenchmarkDataAsZip(benchmarkID uint, writer io.Writer) error {
	// Open the data file
	filePath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.bin", benchmarkID))
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close file: %v\n", closeErr)
		}
		// Hint to GC after we're done
		runtime.GC()
	}()

	// Set up decompression
	zstdDecoder, err := zstd.NewReader(file, zstd.WithDecoderConcurrency(2))
	if err != nil {
		return err
	}
	defer zstdDecoder.Close()

	gobDecoder := gob.NewDecoder(zstdDecoder)
	
	// Try to read header to determine format version
	var header fileHeader
	var runCount int
	
	if err := gobDecoder.Decode(&header); err != nil {
		// Old format - fall back to loading entire dataset
		zstdDecoder.Close()
		_ = file.Close() //nolint:errcheck // Error not critical, falling back to legacy reader
		
		benchmarkData, err := retrieveBenchmarkDataLegacy(benchmarkID)
		if err != nil {
			return err
		}
		
		return exportBenchmarkDataLegacy(benchmarkData, writer)
	}
	
	// New format
	if header.Version != storageFormatVersion {
		return fmt.Errorf("unsupported storage format version: %d", header.Version)
	}
	
	runCount = header.RunCount

	if runCount == 0 {
		return errors.New("no benchmark data to export")
	}

	// Create a new ZIP writer
	zipWriter := zip.NewWriter(writer)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			fmt.Printf("Warning: failed to close zipWriter: %v\n", err)
		}
	}()

	// Export each benchmark run as a CSV file (one at a time - true streaming)
	for i := 0; i < runCount; i++ {
		var run BenchmarkData
		if err := gobDecoder.Decode(&run); err != nil {
			return fmt.Errorf("failed to decode run %d: %w", i, err)
		}
		
		// Create a safe filename from the label
		filename := sanitizeFilename(run.Label) + ".csv"

		// Create a file in the ZIP archive
		fileWriter, err := zipWriter.Create(filename)
		if err != nil {
			return err
		}

		// Write CSV content
		if err := writeBenchmarkDataAsCSV(&run, fileWriter); err != nil {
			return err
		}
		
		// Trigger GC periodically to aggressively reclaim memory
		if i%gcFrequencyExport == 0 && i > 0 {
			runtime.GC()
		}
	}

	return nil
}

// exportBenchmarkDataLegacy exports data in old format (for backward compatibility)
func exportBenchmarkDataLegacy(benchmarkData []*BenchmarkData, writer io.Writer) error {
	if len(benchmarkData) == 0 {
		return errors.New("no benchmark data to export")
	}

	// Create a new ZIP writer
	zipWriter := zip.NewWriter(writer)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			fmt.Printf("Warning: failed to close zipWriter: %v\n", err)
		}
	}()

	// Export each benchmark data as a CSV file
	for i, data := range benchmarkData {
		// Create a safe filename from the label
		filename := sanitizeFilename(data.Label) + ".csv"

		// Create a file in the ZIP archive
		fileWriter, err := zipWriter.Create(filename)
		if err != nil {
			return err
		}

		// Write CSV content
		if err := writeBenchmarkDataAsCSV(data, fileWriter); err != nil {
			return err
		}
		
		// Explicitly nil out the reference to allow GC to reclaim this run's memory
		benchmarkData[i] = nil
		
		// Trigger GC periodically to aggressively reclaim memory
		if i%gcFrequencyExport == 0 && i > 0 {
			runtime.GC()
		}
	}

	return nil
}

// sanitizeFilename removes or replaces characters that are not safe for filenames
func sanitizeFilename(filename string) string {
	// First trim whitespace
	filename = strings.TrimSpace(filename)

	// Replace problematic characters with underscores
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	sanitized := replacer.Replace(filename)

	// If empty after sanitization, use a default name
	if sanitized == "" {
		sanitized = "benchmark"
	}

	return sanitized
}

// writeBenchmarkDataAsCSV writes benchmark data to a writer in MangoHud CSV format
func writeBenchmarkDataAsCSV(data *BenchmarkData, writer io.Writer) error {
	// Use buffered writer for better performance
	bufWriter := bufio.NewWriterSize(writer, 64*1024) // 64KB buffer
	csvWriter := csv.NewWriter(bufWriter)
	defer func() {
		csvWriter.Flush()
		if err := bufWriter.Flush(); err != nil {
			// Log flush error in defer - write errors should have been caught earlier
			fmt.Printf("Warning: failed to flush buffer in defer: %v\n", err)
		}
	}()

	// Write the header line (MangoHud format)
	if err := csvWriter.Write([]string{"os", "cpu", "gpu", "ram", "kernel", "driver", "cpuscheduler"}); err != nil {
		return err
	}

	// Write the specs line
	specsLine := []string{
		data.SpecOS,
		data.SpecCPU,
		data.SpecGPU,
		convertRAMToKB(data.SpecRAM),
		data.SpecLinuxKernel,
		"", // driver (we don't store this separately)
		data.SpecLinuxScheduler,
	}
	if err := csvWriter.Write(specsLine); err != nil {
		return err
	}

	// Write the column headers
	headers := []string{"fps", "frametime", "cpu_load", "gpu_load", "cpu_temp", "cpu_power", "gpu_temp", "gpu_core_clock", "gpu_mem_clock", "gpu_vram_used", "gpu_power", "ram_used", "swap_used"}
	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	// Determine the maximum length of data arrays
	maxLen := 0
	dataArrays := [][]float64{
		data.DataFPS,
		data.DataFrameTime,
		data.DataCPULoad,
		data.DataGPULoad,
		data.DataCPUTemp,
		data.DataCPUPower,
		data.DataGPUTemp,
		data.DataGPUCoreClock,
		data.DataGPUMemClock,
		data.DataGPUVRAMUsed,
		data.DataGPUPower,
		data.DataRAMUsed,
		data.DataSwapUsed,
	}
	for _, arr := range dataArrays {
		if len(arr) > maxLen {
			maxLen = len(arr)
		}
	}

	// Pre-allocate row buffer to avoid repeated allocations
	// Reusing the same slice prevents allocations and also ensures proper clearing
	// of values when arrays have different lengths
	row := make([]string, len(headers))
	
	// Write data rows
	for i := 0; i < maxLen; i++ {
		for j, arr := range dataArrays {
			if i < len(arr) {
				row[j] = strconv.FormatFloat(arr[i], 'f', -1, 64)
			} else {
				row[j] = "" // Clear previous value for shorter arrays
			}
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// convertRAMToKB converts a human-readable RAM string back to kilobytes
func convertRAMToKB(ramStr string) string {
	if ramStr == "" {
		return ""
	}

	// Try to parse the human-readable format
	// This is a best-effort conversion for re-upload compatibility
	ramStr = strings.TrimSpace(ramStr)

	// If it's already a number, return it
	if _, err := strconv.ParseInt(ramStr, 10, 64); err == nil {
		return ramStr
	}

	// Parse human-readable format (e.g., "16 GB", "8.0 GB")
	parts := strings.Fields(ramStr)
	if len(parts) >= 2 {
		val, err := strconv.ParseFloat(parts[0], 64)
		if err == nil {
			unit := strings.ToUpper(parts[1])
			switch unit {
			case "GB":
				return strconv.FormatInt(int64(val*1024*1024), 10)
			case "MB":
				return strconv.FormatInt(int64(val*1024), 10)
			case "KB":
				return strconv.FormatInt(int64(val), 10)
			case "B":
				return strconv.FormatInt(int64(val/1024), 10)
			}
		}
	}

	// If we can't parse it, return empty string
	return ""
}

// RetrieveBenchmarkRun retrieves a single run from benchmark data
// runIndex is 0-based
func RetrieveBenchmarkRun(benchmarkID uint, runIndex int) (*BenchmarkData, error) {
filePath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.bin", benchmarkID))
file, err := os.Open(filePath)
if err != nil {
return nil, err
}
defer func() {
if closeErr := file.Close(); closeErr != nil {
fmt.Printf("Warning: failed to close file: %v\n", closeErr)
}
}()

zstdDecoder, err := zstd.NewReader(file, zstd.WithDecoderConcurrency(2))
if err != nil {
return nil, err
}
defer zstdDecoder.Close()

gobDecoder := gob.NewDecoder(zstdDecoder)

// Try to read header to determine format version
var header fileHeader
if err := gobDecoder.Decode(&header); err != nil {
// Old format - need to load all data
return nil, fmt.Errorf("old format not supported for single run retrieval")
}

// New format detected
if header.Version != storageFormatVersion {
return nil, fmt.Errorf("unsupported storage format version: %d", header.Version)
}

// Check if runIndex is valid
if runIndex < 0 || runIndex >= header.RunCount {
return nil, fmt.Errorf("invalid run index %d (total runs: %d)", runIndex, header.RunCount)
}

// Skip to the requested run
for i := 0; i < runIndex; i++ {
var skipRun BenchmarkData
if err := gobDecoder.Decode(&skipRun); err != nil {
return nil, fmt.Errorf("failed to skip run %d: %w", i, err)
}
}

// Decode the requested run
var run BenchmarkData
if err := gobDecoder.Decode(&run); err != nil {
return nil, fmt.Errorf("failed to decode run %d: %w", runIndex, err)
}

return &run, nil
}
