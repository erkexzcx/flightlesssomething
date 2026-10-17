package app

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFrametimeDensityMigration(t *testing.T) {
	for _, corruptRaw := range []bool{false, true} {
		name := "regenerates legacy statistics"
		if corruptRaw {
			name = "failed migration keeps schema version"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := InitBenchmarksDir(dir); err != nil {
				t.Fatal(err)
			}
			db, err := InitDB(dir)
			if err != nil {
				t.Fatal(err)
			}
			if err = db.DB.Exec("DELETE FROM schema_versions").Error; err != nil {
				t.Fatal(err)
			}
			if err = setSchemaVersion(db.DB, 5); err != nil {
				t.Fatal(err)
			}
			cleanupTestDB(t, db)

			values := make([]float64, 0, 152)
			values = append(values, 0, 99)
			for range 50 {
				values = append(values, 3.14, 3.24, 3.34)
			}
			if err = StoreBenchmarkData([]*BenchmarkData{{Label: "precision", DataFrameTime: values}}, 1); err != nil {
				t.Fatal(err)
			}
			// Pin the old gob representation independently of the current MetricStats type.
			type legacyMetricStats struct {
				Density [][2]int
			}
			type legacyRun struct {
				Stats map[string]*legacyMetricStats
			}
			var encoded bytes.Buffer
			writer, err := zstd.NewWriter(&encoded)
			if err != nil {
				t.Fatal(err)
			}
			legacy := []*legacyRun{{Stats: map[string]*legacyMetricStats{"FrameTime": {Density: [][2]int{{3, 150}}}}}}
			if err = gob.NewEncoder(writer).Encode(legacy); err != nil {
				t.Fatal(err)
			}
			if err = writer.Close(); err != nil {
				t.Fatal(err)
			}
			statsPath := filepath.Join(dir, "benchmarks", "1.stats")
			if err = os.WriteFile(statsPath, encoded.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			rawPath := filepath.Join(dir, "benchmarks", "1.bin")
			if corruptRaw {
				if err = os.WriteFile(rawPath, []byte("invalid raw data"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			rawBefore, err := os.ReadFile(rawPath)
			if err != nil {
				t.Fatal(err)
			}

			migrated, err := InitDB(dir)
			if corruptRaw {
				if err == nil {
					cleanupTestDB(t, migrated)
					t.Fatal("corrupt raw data must fail migration")
				}
				rawDB, openErr := gorm.Open(sqlite.Open(filepath.Join(dir, "flightlesssomething.db")), &gorm.Config{})
				checkDB := &DBInstance{DB: rawDB}
				if openErr != nil {
					t.Fatal(openErr)
				}
				defer cleanupTestDB(t, checkDB)
				version, versionErr := detectSchemaVersion(checkDB.DB)
				if versionErr != nil || version != 5 {
					t.Fatalf("failed migration changed version: %d, %v", version, versionErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer cleanupTestDB(t, migrated)
			version, err := detectSchemaVersion(migrated.DB)
			if err != nil || version != 6 {
				t.Fatalf("expected schema 6, got %d, %v", version, err)
			}
			stats, err := RetrievePreCalculatedStats(1)
			if err != nil {
				t.Fatal(err)
			}
			for _, method := range []map[string]*MetricStats{stats[0].Stats, stats[0].StatsMangoHud} {
				density := method["FrameTime"].Density
				if len(density) != 3 || !approxEqual(density[0][0], 3.1, 1e-9) || density[0][1] != 50 {
					t.Errorf("legacy stats were not regenerated: %v", density)
				}
			}
			rawAfter, err := os.ReadFile(rawPath)
			if err != nil || !bytes.Equal(rawBefore, rawAfter) {
				t.Fatalf("migration modified raw benchmark data: %v", err)
			}
		})
	}
}

func TestPreCalculatedStatsFormat(t *testing.T) {
	for _, tc := range []struct {
		name        string
		header      *fileHeader
		omitPayload bool
		wantError   string
	}{
		{name: "fractional density", header: &fileHeader{Version: 2, RunCount: 1}},
		{name: "missing header", wantError: "failed to decode stats header"},
		{name: "unsupported version", header: &fileHeader{Version: 3, RunCount: 1}, wantError: "unsupported stats format version"},
		{name: "negative count", header: &fileHeader{Version: 2, RunCount: -1}, wantError: "invalid stats run count"},
		{name: "excessive count", header: &fileHeader{Version: 2, RunCount: maxRunsPerBenchmark + 1}, wantError: "invalid stats run count"},
		{name: "missing payload", header: &fileHeader{Version: 2, RunCount: 1}, omitPayload: true, wantError: "failed to decode stats:"},
		{name: "count mismatch", header: &fileHeader{Version: 2, RunCount: 2}, wantError: "does not match header"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := InitBenchmarksDir(dir); err != nil {
				t.Fatal(err)
			}
			var encoded bytes.Buffer
			writer, err := zstd.NewWriter(&encoded)
			if err != nil {
				t.Fatal(err)
			}
			encoder := gob.NewEncoder(writer)
			if tc.header != nil {
				if err = encoder.Encode(tc.header); err != nil {
					t.Fatal(err)
				}
				if !tc.omitPayload {
					payload := []*PreCalculatedRun{{Stats: map[string]*MetricStats{"FrameTime": {Density: [][2]float64{{3.2, 50}}}}}}
					if err = encoder.Encode(payload); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err = writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(dir, "benchmarks", "1.stats"), encoded.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			stats, err := RetrievePreCalculatedStats(1)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("expected %q, got %v", tc.wantError, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(stats) != 1 || len(stats[0].Stats["FrameTime"].Density) != 1 || stats[0].Stats["FrameTime"].Density[0] != [2]float64{3.2, 50} {
				t.Fatalf("fractional density did not survive decoding: %+v", stats)
			}
		})
	}
}
