package flightlesssomething

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
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
	DataGPUCoreClock []float64
	DataGPUMemClock  []float64
	DataGPUVRAMUsed  []float64
	DataGPUPower     []float64
	DataRAMUsed      []float64
	DataSwapUsed     []float64
}

// readBenchmarkFiles reads the uploaded benchmark files and returns a slice of BenchmarkData.
func readBenchmarkFiles(files []*multipart.FileHeader) ([]*BenchmarkData, error) {
	csvFiles := make([]*BenchmarkData, 0)
	linesCount := 0

	for _, fileHeader := range files {
		csvFile := BenchmarkData{}

		file, err := fileHeader.Open()
		if err != nil {
			return nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)

		// Label is filename without extension
		csvFile.Label = strings.TrimSuffix(fileHeader.Filename, ".csv")
		csvFile.Label = strings.TrimSuffix(csvFile.Label, ".htm")

		// First line should contain this: os,cpu,gpu,ram,kernel,driver,cpuscheduler
		if !scanner.Scan() {
			return nil, errors.New("invalid CSV file (err 1)")
		}
		record := strings.Split(strings.TrimRight(scanner.Text(), ","), ",")
		if len(record) != 7 {
			return nil, errors.New("invalid CSV file (err 2)")
		}

		// Second line should contain values
		if !scanner.Scan() {
			return nil, errors.New("invalid CSV file (err 3)")
		}
		record = strings.Split(scanner.Text(), ",")

		for i, v := range record {
			switch i {
			case 0:
				csvFile.SpecOS = truncateString(strings.TrimSpace(v))
			case 1:
				csvFile.SpecCPU = truncateString(strings.TrimSpace(v))
			case 2:
				csvFile.SpecGPU = truncateString(strings.TrimSpace(v))
			case 3:
				kilobytes := new(big.Int)
				_, ok := kilobytes.SetString(strings.TrimSpace(v), 10)
				if !ok {
					return nil, errors.New("failed to convert RAM to big.Int")
				}
				bytes := new(big.Int).Mul(kilobytes, big.NewInt(1024))
				csvFile.SpecRAM = humanize.Bytes(bytes.Uint64())
			case 4:
				csvFile.SpecLinuxKernel = truncateString(strings.TrimSpace(v))
			case 6:
				csvFile.SpecLinuxScheduler = truncateString(strings.TrimSpace(v))
			}
		}

		// 3rd line contain headers for benchmark data: fps,frametime,cpu_load,gpu_load,cpu_temp,gpu_temp,gpu_core_clock,gpu_mem_clock,gpu_vram_used,gpu_power,ram_used,swap_used,process_rss,elapsed
		if !scanner.Scan() {
			return nil, errors.New("invalid CSV file (err 5)")
		}
		record = strings.Split(strings.TrimRight(scanner.Text(), ","), ",")
		if len(record) != 14 {
			return nil, errors.New("invalid CSV file (err 6)")
		}

		// Preallocate slices. First file will be inefficient, but later files will contain
		// value of linesCount that would help to optimize preallocation.
		csvFile.DataFPS = make([]float64, 0, linesCount)
		csvFile.DataFrameTime = make([]float64, 0, linesCount)
		csvFile.DataCPULoad = make([]float64, 0, linesCount)
		csvFile.DataGPULoad = make([]float64, 0, linesCount)
		csvFile.DataCPUTemp = make([]float64, 0, linesCount)
		csvFile.DataGPUTemp = make([]float64, 0, linesCount)
		csvFile.DataGPUCoreClock = make([]float64, 0, linesCount)
		csvFile.DataGPUMemClock = make([]float64, 0, linesCount)
		csvFile.DataGPUVRAMUsed = make([]float64, 0, linesCount)
		csvFile.DataGPUPower = make([]float64, 0, linesCount)
		csvFile.DataRAMUsed = make([]float64, 0, linesCount)
		csvFile.DataSwapUsed = make([]float64, 0, linesCount)

		var counter uint

		for scanner.Scan() {
			record = strings.Split(scanner.Text(), ",")
			if len(record) != 14 {
				return nil, errors.New("invalid CSV file (err 7)")
			}

			val, err := strconv.ParseFloat(record[0], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse FPS value '%s': %v", record[0], err)
			}
			csvFile.DataFPS = append(csvFile.DataFPS, val)

			val, err = strconv.ParseFloat(record[1], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse frametime value '%s': %v", record[1], err)
			}
			csvFile.DataFrameTime = append(csvFile.DataFrameTime, val)

			val, err = strconv.ParseFloat(record[2], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse CPU load value '%s': %v", record[2], err)
			}
			csvFile.DataCPULoad = append(csvFile.DataCPULoad, val)

			val, err = strconv.ParseFloat(record[3], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse GPU load value '%s': %v", record[3], err)
			}
			csvFile.DataGPULoad = append(csvFile.DataGPULoad, val)

			val, err = strconv.ParseFloat(record[4], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse CPU temp value '%s': %v", record[4], err)
			}
			csvFile.DataCPUTemp = append(csvFile.DataCPUTemp, val)

			val, err = strconv.ParseFloat(record[5], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse GPU temp value '%s': %v", record[5], err)
			}
			csvFile.DataGPUTemp = append(csvFile.DataGPUTemp, val)

			val, err = strconv.ParseFloat(record[6], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse GPU core clock value '%s': %v", record[6], err)
			}
			csvFile.DataGPUCoreClock = append(csvFile.DataGPUCoreClock, val)

			val, err = strconv.ParseFloat(record[7], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse GPU mem clock value '%s': %v", record[7], err)
			}
			csvFile.DataGPUMemClock = append(csvFile.DataGPUMemClock, val)

			val, err = strconv.ParseFloat(record[8], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse GPU VRAM used value '%s': %v", record[8], err)
			}
			csvFile.DataGPUVRAMUsed = append(csvFile.DataGPUVRAMUsed, val)

			val, err = strconv.ParseFloat(record[9], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse GPU power value '%s': %v", record[9], err)
			}
			csvFile.DataGPUPower = append(csvFile.DataGPUPower, val)

			val, err = strconv.ParseFloat(record[10], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse RAM used value '%s': %v", record[10], err)
			}
			csvFile.DataRAMUsed = append(csvFile.DataRAMUsed, val)

			val, err = strconv.ParseFloat(record[11], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SWAP used value '%s': %v", record[11], err)
			}
			csvFile.DataSwapUsed = append(csvFile.DataSwapUsed, val)

			counter++
			if counter == 100000 {
				return nil, errors.New("CSV file cannot have more than 100000 data lines")
			}
		}

		// Next file would be more efficient to preallocate slices
		if linesCount < len(csvFile.DataFPS) {
			linesCount = len(csvFile.DataFPS)
		}

		if err := scanner.Err(); err != nil {
			log.Println("error (4) parsing CSV:", err)
			return nil, err
		}

		if len(csvFile.DataFPS) == 0 &&
			len(csvFile.DataFrameTime) == 0 &&
			len(csvFile.DataCPULoad) == 0 &&
			len(csvFile.DataGPULoad) == 0 &&
			len(csvFile.DataCPUTemp) == 0 &&
			len(csvFile.DataGPUTemp) == 0 &&
			len(csvFile.DataGPUCoreClock) == 0 &&
			len(csvFile.DataGPUMemClock) == 0 &&
			len(csvFile.DataGPUVRAMUsed) == 0 &&
			len(csvFile.DataGPUPower) == 0 &&
			len(csvFile.DataRAMUsed) == 0 &&
			len(csvFile.DataSwapUsed) == 0 {
			return nil, errors.New("empty CSV file (err 8)")
		}

		csvFiles = append(csvFiles, &csvFile)
	}

	return csvFiles, nil
}

// truncateString truncates the input string to a maximum of 100 characters and appends "..." if it exceeds that length.
func truncateString(s string) string {
	const maxLength = 100
	if len(s) > maxLength {
		return s[:maxLength] + "..."
	}
	return s
}

func storeBenchmarkData(csvFiles []*BenchmarkData, benchmarkID uint) error {
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
	err = gobEncoder.Encode(csvFiles)
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

func retrieveBenchmarkData(benchmarkID uint) (csvFiles []*BenchmarkData, err error) {
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
	err = gobDecoder.Decode(&csvFiles)
	return csvFiles, err
}

func deleteBenchmarkData(benchmarkID uint) error {
	filePath := filepath.Join(benchmarksDir, fmt.Sprintf("%d.bin", benchmarkID))
	return os.Remove(filePath)
}
