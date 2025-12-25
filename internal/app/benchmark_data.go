package app

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"mime/multipart"
	"os"
	"path/filepath"
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
	precisionFactor = 100000
	bytesToKB       = 1024
	maxDataLines    = 100000
	maxStringLength = 100
)

var benchmarksDir string

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
func parseData(scanner *bufio.Scanner, headerMap map[int]string, benchmarkData *BenchmarkData, isAfterburner bool) error {
	counter := 0

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
		if counter == maxDataLines {
			return errors.New("file cannot have more than 100000 data lines")
		}
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
func readBenchmarkFile(scanner *bufio.Scanner, fileType int) (*BenchmarkData, error) {
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

	err = parseData(scanner, headerMap, benchmarkData, fileType == FileTypeAfterburner)
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
	benchmarkDatas := make([]*BenchmarkData, 0)

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
	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log error but continue - this is cleanup
			fmt.Printf("Warning: failed to close file: %v\n", closeErr)
		}
	}()

	scanner := bufio.NewScanner(file)

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

	benchmarkData, err := readBenchmarkFile(scanner, fileType)
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

// truncateString truncates the input string to maxStringLength characters
func truncateString(s string) string {
	if len(s) > maxStringLength {
		return s[:maxStringLength] + "..."
	}
	return s
}

// StoreBenchmarkData stores benchmark data to disk
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

	var buffer bytes.Buffer
	gobEncoder := gob.NewEncoder(&buffer)
	err = gobEncoder.Encode(benchmarkData)
	if err != nil {
		return err
	}

	zstdEncoder, err := zstd.NewWriter(file, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		return err
	}

	_, err = zstdEncoder.Write(buffer.Bytes())
	if err != nil {
		return err
	}

	if err := zstdEncoder.Close(); err != nil {
		return fmt.Errorf("failed to close zstd encoder: %w", err)
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

	zstdDecoder, err := zstd.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer zstdDecoder.Close()

	var buffer bytes.Buffer
	_, err = buffer.ReadFrom(zstdDecoder)
	if err != nil {
		return nil, err
	}

	var benchmarkData []*BenchmarkData
	gobDecoder := gob.NewDecoder(&buffer)
	err = gobDecoder.Decode(&benchmarkData)
	return benchmarkData, err
}

// GetBenchmarkRunCount returns the count of runs and their labels for a benchmark
// This is optimized to read only metadata without loading the full benchmark data
func GetBenchmarkRunCount(benchmarkID uint) (int, []string, error) {
	metaPath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.meta", benchmarkID))
	metaFile, err := os.Open(metaPath)
	if err != nil {
		// Fallback: if metadata doesn't exist, load full data (backward compatibility)
		if os.IsNotExist(err) {
			benchmarkData, retrieveErr := RetrieveBenchmarkData(benchmarkID)
			if retrieveErr != nil {
				return 0, nil, retrieveErr
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

			return len(benchmarkData), labels, nil
		}
		return 0, nil, err
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
		return 0, nil, err
	}

	return metadata.RunCount, metadata.RunLabels, nil
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
func ExportBenchmarkDataAsZip(benchmarkID uint, writer io.Writer) error {
	// Retrieve the benchmark data
	benchmarkData, err := RetrieveBenchmarkData(benchmarkID)
	if err != nil {
		return err
	}

	if len(benchmarkData) == 0 {
		return errors.New("no benchmark data to export")
	}

	// Create a new ZIP writer
	zipWriter := zip.NewWriter(writer)
	defer func() {

		if err := zipWriter.Close(); err != nil {

			// Log error but continue - this is cleanup

			fmt.Printf("Warning: failed to close zipWriter: %v\n", err)

		}

	}()

	// Export each benchmark data as a CSV file
	for _, data := range benchmarkData {
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
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

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

	// Write data rows
	for i := 0; i < maxLen; i++ {
		row := make([]string, len(headers))
		for j, arr := range dataArrays {
			if i < len(arr) {
				row[j] = strconv.FormatFloat(arr[i], 'f', -1, 64)
			} else {
				row[j] = ""
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
