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
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/klauspost/compress/zstd"
)

type CSVFile struct {
	Filename string

	FPSPointsArray    string
	FrameTimeArray    string
	CPULoadArray      string
	GPULoadArray      string
	CPUTempArray      string
	GPUTempArray      string
	GPUCoreClockArray string
	GPUMemClockArray  string
	GPUVRAMUsedArray  string
	GPUPowerArray     string
	RAMUsedArray      string
	SwapUsedArray     string
}

type CSVSpecs struct {
	MaxPoints int

	Distro    string
	Kernel    string
	GPU       string
	CPU       string
	RAM       string
	Scheduler string
}

// readCSVFiles reads multiple CSV files and returns a slice of CSVFile pointers and the maximum number of FPS records found in any file
func readCSVFiles(files []*multipart.FileHeader) ([]*CSVFile, *CSVSpecs, error) {
	csvFiles := make([]*CSVFile, 0)
	csvSpecs := &CSVSpecs{}

	var linesCount int

	for _, fileHeader := range files {
		csvFile := CSVFile{}

		file, err := fileHeader.Open()
		if err != nil {
			return nil, nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)

		// Set file name (without extension)
		csvFile.Filename = strings.TrimSuffix(fileHeader.Filename, ".csv")

		// First line should contain this: os,cpu,gpu,ram,kernel,driver,cpuscheduler
		if !scanner.Scan() {
			return nil, nil, errors.New("invalid CSV file (err 1)")
		}
		record := strings.Split(strings.TrimRight(scanner.Text(), ","), ",")
		if len(record) != 7 {
			return nil, nil, errors.New("invalid CSV file (err 2)")
		}

		// Second line should contain values
		if !scanner.Scan() {
			return nil, nil, errors.New("invalid CSV file (err 3)")
		}
		record = strings.Split(scanner.Text(), ",")

		for i, v := range record {
			switch i {
			case 0:
				csvSpecs.Distro = truncateString(strings.TrimSpace(v))
			case 1:
				csvSpecs.CPU = truncateString(strings.TrimSpace(v))
			case 2:
				csvSpecs.GPU = truncateString(strings.TrimSpace(v))
			case 3:
				kilobytes := new(big.Int)
				_, ok := kilobytes.SetString(strings.TrimSpace(v), 10)
				if !ok {
					return nil, nil, errors.New("failed to convert RAM to big.Int")
				}
				bytes := new(big.Int).Mul(kilobytes, big.NewInt(1024))
				csvSpecs.RAM = humanize.Bytes(bytes.Uint64())
			case 4:
				csvSpecs.Kernel = truncateString(strings.TrimSpace(v))
			case 6:
				csvSpecs.Scheduler = truncateString(strings.TrimSpace(v))
			}
		}

		// 3rd line contain headers for benchmark data: fps,frametime,cpu_load,gpu_load,cpu_temp,gpu_temp,gpu_core_clock,gpu_mem_clock,gpu_vram_used,gpu_power,ram_used,swap_used,process_rss,elapsed
		if !scanner.Scan() {
			return nil, nil, errors.New("invalid CSV file (err 5)")
		}
		record = strings.Split(strings.TrimRight(scanner.Text(), ","), ",")
		if len(record) != 14 {
			return nil, nil, errors.New("invalid CSV file (err 6)")
		}

		fpsPoints := make([]string, 0, linesCount)
		frametimePoints := make([]string, 0, linesCount)
		cpuLoadPoints := make([]string, 0, linesCount)
		gpuLoadPoints := make([]string, 0, linesCount)
		cpuTempPoints := make([]string, 0, linesCount)
		gpuTempPoints := make([]string, 0, linesCount)
		gpuCoreClockPoints := make([]string, 0, linesCount)
		gpuMemClockPoints := make([]string, 0, linesCount)
		gpuVRAMUsedPoints := make([]string, 0, linesCount)
		gpuPowerPoints := make([]string, 0, linesCount)
		RAMUsedPoints := make([]string, 0, linesCount)
		SWAPUsedPoints := make([]string, 0, linesCount)

		var counter uint

		for scanner.Scan() {
			record = strings.Split(scanner.Text(), ",")
			if len(record) != 14 {
				return nil, nil, errors.New("invalid CSV file (err 7)")
			}
			fpsPoints = append(fpsPoints, record[0])
			frametimePoints = append(frametimePoints, record[1])
			cpuLoadPoints = append(cpuLoadPoints, record[2])
			gpuLoadPoints = append(gpuLoadPoints, record[3])
			cpuTempPoints = append(cpuTempPoints, record[4])
			gpuTempPoints = append(gpuTempPoints, record[5])
			gpuCoreClockPoints = append(gpuCoreClockPoints, record[6])
			gpuMemClockPoints = append(gpuMemClockPoints, record[7])
			gpuVRAMUsedPoints = append(gpuVRAMUsedPoints, record[8])
			gpuPowerPoints = append(gpuPowerPoints, record[9])
			RAMUsedPoints = append(RAMUsedPoints, record[10])
			SWAPUsedPoints = append(SWAPUsedPoints, record[11])

			counter++
			if counter == 100000 {
				return nil, nil, errors.New("too large CSV file")
			}
		}

		// More efficient buffer allocation
		linesCount = len(fpsPoints)

		if err := scanner.Err(); err != nil {
			log.Println("error (4) parsing CSV:", err)
			return nil, nil, err
		}

		if len(fpsPoints) == 0 {
			return nil, nil, errors.New("invalid CSV file (err 8)")
		}

		if len(fpsPoints) > csvSpecs.MaxPoints {
			csvSpecs.MaxPoints = len(fpsPoints)
		}

		csvFile.FPSPointsArray = strings.Join(fpsPoints, ",")
		csvFile.FrameTimeArray = strings.Join(frametimePoints, ",")
		csvFile.CPULoadArray = strings.Join(cpuLoadPoints, ",")
		csvFile.GPULoadArray = strings.Join(gpuLoadPoints, ",")
		csvFile.CPUTempArray = strings.Join(cpuTempPoints, ",")
		csvFile.GPUTempArray = strings.Join(gpuTempPoints, ",")
		csvFile.GPUCoreClockArray = strings.Join(gpuCoreClockPoints, ",")
		csvFile.GPUMemClockArray = strings.Join(gpuMemClockPoints, ",")
		csvFile.GPUVRAMUsedArray = strings.Join(gpuVRAMUsedPoints, ",")
		csvFile.GPUPowerArray = strings.Join(gpuPowerPoints, ",")
		csvFile.RAMUsedArray = strings.Join(RAMUsedPoints, ",")
		csvFile.SwapUsedArray = strings.Join(SWAPUsedPoints, ",")

		csvFiles = append(csvFiles, &csvFile)
	}

	return csvFiles, csvSpecs, nil
}

// truncateString truncates the input string to a maximum of 100 characters and appends "..." if it exceeds that length.
func truncateString(s string) string {
	const maxLength = 100
	if len(s) > maxLength {
		return s[:maxLength] + "..."
	}
	return s
}

func storeBenchmarkData(csvFiles []*CSVFile, benchmarkID uint) error {
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

func retrieveBenchmarkData(benchmarkID uint) (csvFiles []*CSVFile, err error) {
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
