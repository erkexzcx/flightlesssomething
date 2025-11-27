#!/bin/bash
# Test script for migration tool
# This creates a sample old database and tests migration

set -e

echo "=== Migration Tool Test ==="
echo ""

# Clean up previous test data
rm -rf /tmp/test-migration-old /tmp/test-migration-new
mkdir -p /tmp/test-migration-old/benchmarks

echo "Creating sample old database..."

# Create a minimal old database using SQLite directly
cat > /tmp/create_test_data.sql << 'EOF'
-- Create tables matching old schema
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    discord_id VARCHAR(20),
    username VARCHAR(32)
);

CREATE TABLE benchmarks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    user_id INTEGER,
    title VARCHAR(100),
    description VARCHAR(500),
    ai_summary TEXT
);

-- Insert test users
INSERT INTO users (id, created_at, updated_at, discord_id, username) 
VALUES 
    (1, '2024-01-01 10:00:00', '2024-01-01 10:00:00', '123456789', 'TestUser1'),
    (2, '2024-01-02 10:00:00', '2024-01-02 10:00:00', '987654321', 'TestUser2');

-- Insert test benchmarks
INSERT INTO benchmarks (id, created_at, updated_at, user_id, title, description, ai_summary)
VALUES
    (1, '2024-01-03 10:00:00', '2024-01-03 10:00:00', 1, 'Cyberpunk 2077', 'High settings', 'Good performance'),
    (2, '2024-01-04 10:00:00', '2024-01-04 10:00:00', 2, 'Red Dead Redemption 2', 'Ultra settings', 'Excellent');
EOF

sqlite3 /tmp/test-migration-old/database.db < /tmp/create_test_data.sql

# Create minimal benchmark data files using Go
mkdir -p /tmp/bench_data_creator
cat > /tmp/bench_data_creator/main.go << 'EOF'
package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
)

type BenchmarkData struct {
	Label              string
	SpecOS             string
	SpecCPU            string
	SpecGPU            string
	SpecRAM            string
	SpecLinuxKernel    string
	SpecLinuxScheduler string
	DataFPS            []float64
	DataFrameTime      []float64
	DataCPULoad        []float64
	DataGPULoad        []float64
	DataCPUTemp        []float64
	DataGPUTemp        []float64
	DataGPUCoreClock   []float64
	DataGPUMemClock    []float64
	DataGPUVRAMUsed    []float64
	DataGPUPower       []float64
	DataRAMUsed        []float64
	DataSwapUsed       []float64
}

func createFile(benchmarkID int, label string, fps float64) {
	data := []*BenchmarkData{
		{
			Label:         label,
			SpecOS:        "Linux",
			SpecCPU:       "AMD Ryzen 9",
			SpecGPU:       "NVIDIA RTX 3080",
			SpecRAM:       "32GB",
			SpecLinuxKernel: "6.1.0",
			SpecLinuxScheduler: "performance",
			DataFPS:       []float64{fps - 2, fps, fps + 2},
			DataFrameTime: []float64{16.6, 16.7, 16.5},
			DataCPULoad:   []float64{45.0, 50.0, 48.0},
			DataGPULoad:   []float64{95.0, 97.0, 96.0},
		},
	}

	var buffer bytes.Buffer
	gobEncoder := gob.NewEncoder(&buffer)
	if err := gobEncoder.Encode(data); err != nil {
		panic(err)
	}

	filePath := filepath.Join("/tmp/test-migration-old/benchmarks", fmt.Sprintf("%d.bin", benchmarkID))
	file, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	zstdEncoder, err := zstd.NewWriter(file, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		panic(err)
	}
	defer zstdEncoder.Close()

	if _, err := zstdEncoder.Write(buffer.Bytes()); err != nil {
		panic(err)
	}
}

func main() {
	createFile(1, "Run 1", 60.0)
	createFile(2, "Run 1", 55.0)
	fmt.Println("Benchmark data files created")
}
EOF

cat > /tmp/bench_data_creator/go.mod << 'GOMOD'
module benchdata

go 1.24

require github.com/klauspost/compress v1.18.1
GOMOD

(cd /tmp/bench_data_creator && go mod tidy && go run main.go)

echo "Test data created successfully!"
echo ""

# Build migration tool if not present
if [ ! -f ./migrate ]; then
    echo "Building migration tool..."
    go build -o migrate ./cmd/migrate
    echo ""
fi

# Show what was created
echo "Old database contents:"
sqlite3 /tmp/test-migration-old/database.db "SELECT COUNT(*) as users FROM users;"
sqlite3 /tmp/test-migration-old/database.db "SELECT COUNT(*) as benchmarks FROM benchmarks;"
ls -lh /tmp/test-migration-old/benchmarks/
echo ""

# Run migration in dry-run mode
echo "Running migration in DRY-RUN mode..."
./migrate -old-data-dir=/tmp/test-migration-old -new-data-dir=/tmp/test-migration-new -dry-run
echo ""

# Run actual migration
echo "Running actual migration..."
./migrate -old-data-dir=/tmp/test-migration-old -new-data-dir=/tmp/test-migration-new
echo ""

# Verify migration results
echo "Verifying migration results..."
echo ""
echo "New database contents:"
sqlite3 /tmp/test-migration-new/flightlesssomething.db "SELECT COUNT(*) as users FROM users;"
sqlite3 /tmp/test-migration-new/flightlesssomething.db "SELECT COUNT(*) as benchmarks FROM benchmarks;"
echo ""
echo "New benchmark files:"
ls -lh /tmp/test-migration-new/benchmarks/
echo ""

# Verify data integrity
echo "Checking migrated user data:"
sqlite3 /tmp/test-migration-new/flightlesssomething.db "SELECT id, discord_id, username, is_admin, is_banned FROM users;"
echo ""
echo "Checking migrated benchmark data:"
sqlite3 /tmp/test-migration-new/flightlesssomething.db "SELECT id, user_id, title, substr(description, 1, 50) as description FROM benchmarks;"
echo ""

echo "=== Migration Test Complete ==="
echo "Test data preserved in:"
echo "  Old: /tmp/test-migration-old/"
echo "  New: /tmp/test-migration-new/"
