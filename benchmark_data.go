package flightlesssomething

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/gob"
	"errors"
	"fmt"
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

		var benchmarkData *BenchmarkData
		var suffix string
		switch firstLine {
		case "os,cpu,gpu,ram,kernel,driver,cpuscheduler": // MangoHud
			benchmarkData, err = readMangoHudFile(scanner)
			suffix = ".csv"
		case "PLACEHOLDER": // RivaTuner
			benchmarkData, err = readMangoHudFile(scanner)
			suffix = ".htm"
		default:
			return nil, errors.New("unsupported file format")
		}

		if err != nil {
			return nil, err
		}
		benchmarkData.Label = strings.TrimSuffix(fileHeader.Filename, suffix)
		benchmarkDatas = append(benchmarkDatas, benchmarkData)
	}

	return benchmarkDatas, nil
}

func readMangoHudFile(scanner *bufio.Scanner) (*BenchmarkData, error) {
	benchmarkData := &BenchmarkData{}

	// Second line should contain values
	if !scanner.Scan() {
		return nil, errors.New("failed to read file (err mh1)")
	}
	record := strings.Split(scanner.Text(), ",")

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
			if !ok {
				bytes := new(big.Int).Mul(kilobytes, big.NewInt(1024))
				benchmarkData.SpecRAM = humanize.Bytes(bytes.Uint64())
			} else {
				benchmarkData.SpecRAM = truncateString(strings.TrimSpace(v))
			}
		case 4:
			benchmarkData.SpecLinuxKernel = truncateString(strings.TrimSpace(v))
		case 6:
			benchmarkData.SpecLinuxScheduler = truncateString(strings.TrimSpace(v))
		}
	}

	// 3rd line contain headers for benchmark data
	if !scanner.Scan() {
		return nil, errors.New("failed to read file (err mh2)")
	}
	record = strings.Split(strings.TrimRight(scanner.Text(), ","), ",")
	if len(record) == 0 {
		return nil, errors.New("failed to read file (err mh3)")
	}

	benchmarkData.DataFPS = make([]float64, 0)
	benchmarkData.DataFrameTime = make([]float64, 0)
	benchmarkData.DataCPULoad = make([]float64, 0)
	benchmarkData.DataGPULoad = make([]float64, 0)
	benchmarkData.DataCPUTemp = make([]float64, 0)
	benchmarkData.DataGPUTemp = make([]float64, 0)
	benchmarkData.DataGPUCoreClock = make([]float64, 0)
	benchmarkData.DataGPUMemClock = make([]float64, 0)
	benchmarkData.DataGPUVRAMUsed = make([]float64, 0)
	benchmarkData.DataGPUPower = make([]float64, 0)
	benchmarkData.DataRAMUsed = make([]float64, 0)
	benchmarkData.DataSwapUsed = make([]float64, 0)

	var counter uint
	for scanner.Scan() {
		record = strings.Split(scanner.Text(), ",")
		if len(record) < 12 { // Ignore last 2 columns as they are not needed
			return nil, errors.New("failed to read file (err mh4)")
		}

		val, err := strconv.ParseFloat(record[0], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse FPS value '%s': %v", record[0], err)
		}
		benchmarkData.DataFPS = append(benchmarkData.DataFPS, val)

		val, err = strconv.ParseFloat(record[1], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse frametime value '%s': %v", record[1], err)
		}
		benchmarkData.DataFrameTime = append(benchmarkData.DataFrameTime, val)

		val, err = strconv.ParseFloat(record[2], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CPU load value '%s': %v", record[2], err)
		}
		benchmarkData.DataCPULoad = append(benchmarkData.DataCPULoad, val)

		val, err = strconv.ParseFloat(record[3], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GPU load value '%s': %v", record[3], err)
		}
		benchmarkData.DataGPULoad = append(benchmarkData.DataGPULoad, val)

		val, err = strconv.ParseFloat(record[4], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CPU temp value '%s': %v", record[4], err)
		}
		benchmarkData.DataCPUTemp = append(benchmarkData.DataCPUTemp, val)

		val, err = strconv.ParseFloat(record[5], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GPU temp value '%s': %v", record[5], err)
		}
		benchmarkData.DataGPUTemp = append(benchmarkData.DataGPUTemp, val)

		val, err = strconv.ParseFloat(record[6], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GPU core clock value '%s': %v", record[6], err)
		}
		benchmarkData.DataGPUCoreClock = append(benchmarkData.DataGPUCoreClock, val)

		val, err = strconv.ParseFloat(record[7], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GPU mem clock value '%s': %v", record[7], err)
		}
		benchmarkData.DataGPUMemClock = append(benchmarkData.DataGPUMemClock, val)

		val, err = strconv.ParseFloat(record[8], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GPU VRAM used value '%s': %v", record[8], err)
		}
		benchmarkData.DataGPUVRAMUsed = append(benchmarkData.DataGPUVRAMUsed, val)

		val, err = strconv.ParseFloat(record[9], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GPU power value '%s': %v", record[9], err)
		}
		benchmarkData.DataGPUPower = append(benchmarkData.DataGPUPower, val)

		val, err = strconv.ParseFloat(record[10], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RAM used value '%s': %v", record[10], err)
		}
		benchmarkData.DataRAMUsed = append(benchmarkData.DataRAMUsed, val)

		val, err = strconv.ParseFloat(record[11], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SWAP used value '%s': %v", record[11], err)
		}
		benchmarkData.DataSwapUsed = append(benchmarkData.DataSwapUsed, val)

		counter++
		if counter == 100000 {
			return nil, errors.New("CSV file cannot have more than 100000 data lines")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(benchmarkData.DataFPS) == 0 &&
		len(benchmarkData.DataFrameTime) == 0 &&
		len(benchmarkData.DataCPULoad) == 0 &&
		len(benchmarkData.DataGPULoad) == 0 &&
		len(benchmarkData.DataCPUTemp) == 0 &&
		len(benchmarkData.DataGPUTemp) == 0 &&
		len(benchmarkData.DataGPUCoreClock) == 0 &&
		len(benchmarkData.DataGPUMemClock) == 0 &&
		len(benchmarkData.DataGPUVRAMUsed) == 0 &&
		len(benchmarkData.DataGPUPower) == 0 &&
		len(benchmarkData.DataRAMUsed) == 0 &&
		len(benchmarkData.DataSwapUsed) == 0 {
		return nil, errors.New("empty CSV file")
	}

	return benchmarkData, nil
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
		csvWriter.Write(header)
		specs := []string{data.SpecOS, data.SpecCPU, data.SpecGPU, data.SpecRAM, data.SpecLinuxKernel, "", data.SpecLinuxScheduler}
		csvWriter.Write(specs)

		// Write the data header.
		dataHeader := []string{"fps", "frametime", "cpu_load", "gpu_load", "cpu_temp", "gpu_temp", "gpu_core_clock", "gpu_mem_clock", "gpu_vram_used", "gpu_power", "ram_used", "swap_used"}
		csvWriter.Write(dataHeader)

		// Write the data rows.
		for i := range data.DataFPS {
			row := []string{
				strconv.FormatFloat(data.DataFPS[i], 'f', 4, 64),
				strconv.FormatFloat(data.DataFrameTime[i], 'f', 4, 64),
				strconv.FormatFloat(data.DataCPULoad[i], 'f', 4, 64),
				strconv.FormatFloat(data.DataGPULoad[i], 'f', 4, 64),
				strconv.FormatFloat(data.DataCPUTemp[i], 'f', 4, 64),
				strconv.FormatFloat(data.DataGPUTemp[i], 'f', 4, 64),
				strconv.FormatFloat(data.DataGPUCoreClock[i], 'f', 4, 64),
				strconv.FormatFloat(data.DataGPUMemClock[i], 'f', 4, 64),
				strconv.FormatFloat(data.DataGPUVRAMUsed[i], 'f', 4, 64),
				strconv.FormatFloat(data.DataGPUPower[i], 'f', 4, 64),
				strconv.FormatFloat(data.DataRAMUsed[i], 'f', 4, 64),
				strconv.FormatFloat(data.DataSwapUsed[i], 'f', 4, 64),
			}
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
