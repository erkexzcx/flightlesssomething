package flightlesssomething

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/gob"
	"errors"
	"fmt"
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

type BenchmarkData struct {
	Label string

	// Specs
	SpecOS             string
	SpecGPU            string
	SpecCPU            string
	SpecRAM            string
	SpecLinuxKernel    string
	SpecLinuxScheduler string

	// Data
	DataFPS          []float64
	DataFrameTime    []float64
	DataCPULoad      []float64
	DataGPULoad      []float64
	DataCPUTemp      []float64
	DataGPUTemp      []float64
	DataCPUPower     []float64
	DataGPUCoreClock []float64
	DataGPUMemClock  []float64
	DataGPUVRAMUsed  []float64
	DataGPUPower     []float64
	DataRAMUsed      []float64
	DataSwapUsed     []float64
}

const (
	FileTypeUnknown = iota
	FileTypeMangoHud
	FileTypeAfterburner
)

func parseHeader(scanner *bufio.Scanner) (map[string]int, error) {
	if !scanner.Scan() {
		return nil, errors.New("failed to read file (header)")
	}
	line := strings.TrimRight(scanner.Text(), ", ")
	line = strings.TrimSpace(line)

	headerMap := make(map[string]int)
	for i, field := range strings.Split(line, ",") {
		headerMap[strings.TrimSpace(field)] = i
	}
	return headerMap, nil
}

func parseData(scanner *bufio.Scanner, headerMap map[string]int, benchmarkData *BenchmarkData, isAfterburner bool) error {
	var counter uint
	for scanner.Scan() {
		record := strings.Split(scanner.Text(), ",")
		if len(record) < len(headerMap) {
			return errors.New("failed to read file (data)")
		}

		// Trim all values
		for i := 0; i < len(record); i++ {
			record[i] = strings.TrimSpace(record[i])
		}

		for key, index := range headerMap {
			// Skip timestamp fields for Afterburner format
			if isAfterburner && (index == 0 || index == 1) {
				continue
			}

			val, err := strconv.ParseFloat(record[index], 64)
			if err != nil {
				return fmt.Errorf("failed to parse %s value '%s': %v", key, record[index], err)
			}

			switch key {
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
					val = math.Round(val/2*100000) / 100000 // divide by 2 and round to 5 decimal places
				}
				benchmarkData.DataGPUMemClock = append(benchmarkData.DataGPUMemClock, val)
			case "gpu_vram_used", "Memory usage":
				if isAfterburner {
					val = math.Round(val/1024*100000) / 100000 // divide by 1024 and round to 5 decimal places
				}
				benchmarkData.DataGPUVRAMUsed = append(benchmarkData.DataGPUVRAMUsed, val)
			case "gpu_power", "Power":
				benchmarkData.DataGPUPower = append(benchmarkData.DataGPUPower, val)
			case "ram_used", "RAM usage":
				if isAfterburner {
					val = math.Round(val/1024*100000) / 100000 // divide by 1024 and round to 5 decimal places
				}
				benchmarkData.DataRAMUsed = append(benchmarkData.DataRAMUsed, val)
			case "swap_used":
				benchmarkData.DataSwapUsed = append(benchmarkData.DataSwapUsed, val)
			}
		}

		counter++
		if counter == 100000 {
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
		len(benchmarkData.DataGPUTemp) == 0 &&
		len(benchmarkData.DataCPUPower) == 0 &&
		len(benchmarkData.DataGPUCoreClock) == 0 &&
		len(benchmarkData.DataGPUMemClock) == 0 &&
		len(benchmarkData.DataGPUVRAMUsed) == 0 &&
		len(benchmarkData.DataGPUPower) == 0 &&
		len(benchmarkData.DataRAMUsed) == 0 &&
		len(benchmarkData.DataSwapUsed) == 0 {
		return errors.New("empty file")
	}

	return nil
}

func readBenchmarkFile(scanner *bufio.Scanner, fileType int) (*BenchmarkData, error) {
	benchmarkData := &BenchmarkData{}

	// Second line should contain specs
	if !scanner.Scan() {
		return nil, errors.New("failed to read file (err 1)")
	}
	record := strings.Split(scanner.Text(), ",")
	switch fileType {
	case FileTypeAfterburner:
		if len(record) < 3 {
			return nil, errors.New("failed to read file (err 2)")
		}
		benchmarkData.SpecOS = "Windows" // Hardcode
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
					// Contains number that represents kilobytes
					bytes := new(big.Int).Mul(kilobytes, big.NewInt(1024))
					benchmarkData.SpecRAM = humanize.Bytes(bytes.Uint64())
				} else {
					// Contains humanized (or invalid) value, so no conversion needed
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
		// Skip len(headerMap) amount of lines as this is not needed
		for i := 0; i < len(headerMap); i++ {
			if !scanner.Scan() {
				return nil, errors.New("failed to read file (err 3)")
			}
		}
	}

	err = parseData(scanner, headerMap, benchmarkData, fileType == FileTypeAfterburner)
	if err != nil {
		return nil, err
	}

	return benchmarkData, nil
}

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

func readBenchmarkFiles(files []*multipart.FileHeader) ([]*BenchmarkData, error) {
	benchmarkDatas := make([]*BenchmarkData, 0)

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return nil, err
		}

		defer file.Close()
		scanner := bufio.NewScanner(file)

		// FirstLine identifies file format
		if !scanner.Scan() {
			return nil, errors.New("failed to read file (err 1)")
		}
		firstLine := scanner.Text()
		firstLine = strings.TrimRight(firstLine, ", ")
		firstLine = strings.TrimSpace(firstLine)

		fileType := detectFileType(firstLine)
		if fileType == FileTypeUnknown {
			return nil, errors.New("unsupported file format")
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
		benchmarkDatas = append(benchmarkDatas, benchmarkData)
	}

	return benchmarkDatas, nil
}

// truncateString truncates the input string to a maximum of 100 characters and appends "..." if it exceeds that length.
func truncateString(s string) string {
	const maxLength = 100
	if len(s) > maxLength {
		return s[:maxLength] + "..."
	}
	return s
}

func storeBenchmarkData(benchmarkData []*BenchmarkData, benchmarkID uint) error {
	// Store to disk
	filePath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.bin", benchmarkID))
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Convert to []byte
	var buffer bytes.Buffer
	gobEncoder := gob.NewEncoder(&buffer)
	err = gobEncoder.Encode(benchmarkData)
	if err != nil {
		return err
	}

	// Compress and write to file
	zstdEncoder, err := zstd.NewWriter(file, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		return err
	}
	defer zstdEncoder.Close()
	_, err = zstdEncoder.Write(buffer.Bytes())
	return err
}

func retrieveBenchmarkData(benchmarkID uint) (benchmarkData []*BenchmarkData, err error) {
	filePath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.bin", benchmarkID))
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decompress and read from file
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

	// Decode
	gobDecoder := gob.NewDecoder(&buffer)
	err = gobDecoder.Decode(&benchmarkData)
	return benchmarkData, err
}

func deleteBenchmarkData(benchmarkID uint) error {
	filePath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.bin", benchmarkID))
	return os.Remove(filePath)
}

func createZipFromBenchmarkData(benchmarkData []*BenchmarkData) (*bytes.Buffer, error) {
	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, data := range benchmarkData {
		// Create a new CSV file in the zip archive.
		fileName := fmt.Sprintf("%s.csv", data.Label)
		fileWriter, err := zipWriter.Create(fileName)
		if err != nil {
			return nil, fmt.Errorf("could not create file in zip: %v", err)
		}

		// Create a CSV writer.
		csvWriter := csv.NewWriter(fileWriter)

		// Write the header.
		header := []string{"os", "cpu", "gpu", "ram", "kernel", "driver", "cpuscheduler"}
		specs := []string{data.SpecOS, data.SpecCPU, data.SpecGPU, data.SpecRAM, data.SpecLinuxKernel, "", data.SpecLinuxScheduler}
		csvWriter.Write(header)
		csvWriter.Write(specs)

		// Dynamically build the data header and rows based on available data.
		dataHeader := []string{}
		dataRows := [][]string{}

		addColumn := func(headerName string, data []float64) {
			if len(data) > 0 {
				dataHeader = append(dataHeader, headerName)
				for i := 0; i < len(data); i++ {
					if len(dataRows) <= i {
						dataRows = append(dataRows, make([]string, len(dataHeader)-1))
					}
					dataRows[i] = append(dataRows[i], formatFloatOrZero(data, i))
				}
			}
		}

		addColumn("fps", data.DataFPS)
		addColumn("frametime", data.DataFrameTime)
		addColumn("cpu_load", data.DataCPULoad)
		addColumn("gpu_load", data.DataGPULoad)
		addColumn("cpu_temp", data.DataCPUTemp)
		addColumn("cpu_power", data.DataCPUPower)
		addColumn("gpu_temp", data.DataGPUTemp)
		addColumn("gpu_core_clock", data.DataGPUCoreClock)
		addColumn("gpu_mem_clock", data.DataGPUMemClock)
		addColumn("gpu_vram_used", data.DataGPUVRAMUsed)
		addColumn("gpu_power", data.DataGPUPower)
		addColumn("ram_used", data.DataRAMUsed)
		addColumn("swap_used", data.DataSwapUsed)

		// Write the data header.
		csvWriter.Write(dataHeader)

		// Write the data rows.
		for _, row := range dataRows {
			csvWriter.Write(row)
		}

		// Make sure to flush the writer.
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return nil, fmt.Errorf("could not write CSV: %v", err)
		}
	}

	// Close the zip writer to flush the buffer.
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("could not close zip writer: %v", err)
	}

	return buf, nil
}

// Helper function to format float or return "0.0000" if index is out of range.
func formatFloatOrZero(data []float64, index int) string {
	if index < len(data) {
		return strconv.FormatFloat(data[index], 'f', 4, 64)
	}
	return "0.0000"
}
