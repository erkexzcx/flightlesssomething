---
name: mangohud-log-generation
description: "MangoHud log file generation — configuration options (output_folder, log_interval, log_duration, autostart_log), file naming conventions, keybinds for toggling logging, summary file format and statistics, log_versioning option, upload to flightlessmango.com, and benchmark percentile configuration. Use when: configuring MangoHud for benchmark logging, understanding log timing/intervals, parsing summary CSV files, or troubleshooting why logs aren't generated."
---

# MangoHud Log Generation Reference

Based on MangoHud source code (`src/logging.cpp`, `src/logging.h`, `src/fps_metrics.h`) and config (`data/MangoHud.conf`).

Source repository: https://github.com/flightlessmango/MangoHud

---

## Configuration Options

These MangoHud config options control log file generation. Set them in `~/.config/MangoHud/MangoHud.conf` or via `MANGOHUD_CONFIG` environment variable.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `output_folder` | string | `$HOME` | Directory for log files. **Required** — logging won't work without a valid path |
| `output_file` | string | (auto-generated) | Override the full output file path/name |
| `log_interval` | int (ms) | `0` | Sampling interval in milliseconds. `0` = log every rendered frame |
| `log_duration` | int (sec) | (none) | Auto-stop logging after N seconds |
| `autostart_log` | int (sec) | (none) | Auto-start logging N seconds after MangoHud initializes |
| `toggle_logging` | keybind | `Shift_L+F2` | Keybind to manually start/stop logging |
| `log_versioning` | bool | off | Prepend version info and section headers to the log |
| `benchmark_percentiles` | string | `97,AVG,1,0.1` | Comma-separated percentiles for the summary file |
| `permit_upload` | int | `0` | Set to `1` to allow uploading logs to flightlessmango.com |
| `upload_logs` | bool | off | Automatically upload logs after logging stops |
| `upload_log` | keybind | `Shift_L+F3` | Keybind to upload last log |

### Minimal Config for Logging

```ini
output_folder=/home/user/mangologs
```

Then press `Shift_L+F2` during gameplay to start/stop logging.

### Auto-Logging Config

```ini
output_folder=/home/user/mangologs
autostart_log=5
log_duration=60
```

This starts logging 5 seconds after the game launches and stops after 60 seconds — useful for reproducible benchmarks.

---

## File Naming

Generated log files follow this pattern:

```
{program_name}_{YYYY-MM-DD_HH-MM-SS}.csv
```

- **program_name**: Detected automatically:
  - For Wine/Proton games: the Wine executable name (via `get_wine_exe_name()`)
  - For native Linux games: the process name (via `get_program_name()`)
- **Timestamp**: Local time when `start_logging()` is called, formatted as `%Y-%m-%d_%H-%M-%S`

**Examples:**
```
TheWitcher3_2025-03-15_14-30-45.csv
cs2_2025-03-15_15-00-12.csv
```

Files are written to the `output_folder` directory:
```
/home/user/mangologs/TheWitcher3_2025-03-15_14-30-45.csv
```

---

## Logging Modes

### Mode 1: Per-Frame Logging (`log_interval=0`, default)

Every rendered frame produces one data row. This gives the highest resolution:
- At 60 FPS → ~60 rows per second
- At 240 FPS → ~240 rows per second
- At 1000 FPS (loading screen) → ~1000 rows per second

Data is written immediately after each frame via `Logger::try_log()`, which is called from `update_hud_info_with_frametime()`.

### Mode 2: Interval Logging (`log_interval=N`)

A background thread samples data every N milliseconds. A dedicated logging thread runs:

```cpp
void Logger::logging(){
    wait_until_data_valid();
    while (is_active()){
        try_log();
        this_thread::sleep_for(std::chrono::milliseconds(log_interval));
    }
}
```

With `log_interval=100`, you get ~10 samples per second regardless of FPS.

### Writing Process

Data is written incrementally — each sample is appended and flushed immediately:

```cpp
void Logger::writeToFile(){
    if (!output_file){
        output_file.open(m_log_files.back(), ios::out | ios::app);
        writeFileHeaders(output_file);  // writes lines 1-3 on first call
    }
    // write one data row
    output_file << logArray.back().fps << ","
                << logArray.back().frametime << ","
                // ... all columns ...
                << elapsed_ns << "\n";
    output_file.flush();
}
```

This means:
- The file is valid CSV from the very first write
- The file grows incrementally during the session
- A crash/kill will preserve all data written up to that point
- The file is opened with `ios::out | ios::app` on first write only

---

## Summary File

When logging stops (`stop_logging()`), MangoHud automatically generates a companion summary CSV.

### Filename

```
{original_name_without_.csv}_summary.csv
```

**Example:** `TheWitcher3_2025-03-15_14-30-45_summary.csv`

### Format

The summary file is a single-header, single-row CSV:

```csv
0.1% Min FPS,1% Min FPS,97% Percentile FPS,Average FPS,GPU Load,CPU Load,Average Frame Time,Average GPU Temp,Average CPU Temp,Average VRAM Used,Average RAM Used,Average Swap Used,Peak GPU Load,Peak CPU Load,Peak GPU Temp,Peak CPU Temp,Peak VRAM Used,Peak RAM Used,Peak Swap Used
<values>
```

### Summary Columns

| Column | Calculation |
|--------|-------------|
| `0.1% Min FPS` | 0.1th percentile frametime converted to FPS |
| `1% Min FPS` | 1st percentile frametime converted to FPS |
| `97% Percentile FPS` | 97th percentile frametime converted to FPS |
| `Average FPS` | `1000 / (sum_of_frametimes / count)` |
| `GPU Load` | Average `gpu_load` across all samples |
| `CPU Load` | Average `cpu_load` across all samples |
| `Average Frame Time` | `sum_of_frametimes / count` (ms) |
| `Average GPU Temp` | Average `gpu_temp` |
| `Average CPU Temp` | Average `cpu_temp` |
| `Average VRAM Used` | Average `gpu_vram_used` |
| `Average RAM Used` | Average `ram_used` |
| `Average Swap Used` | Average `swap_used` |
| `Peak GPU Load` | Maximum `gpu_load` |
| `Peak CPU Load` | Maximum `cpu_load` |
| `Peak GPU Temp` | Maximum `gpu_temp` |
| `Peak CPU Temp` | Maximum `cpu_temp` |
| `Peak VRAM Used` | Maximum `gpu_vram_used` |
| `Peak RAM Used` | Maximum `ram_used` |
| `Peak Swap Used` | Maximum `swap_used` |

### Percentile Calculation

Percentiles are calculated from **frametimes** (NOT FPS values), sorted in descending order, then converted to FPS:

```cpp
std::sort(sorted_values.begin(), sorted_values.end(), std::greater<float>());
// For percentile P (0-1):
uint64_t idx = P * sorted_values.size() - 1;
fps_value = 1000.f / sorted_values[idx];
```

The "AVG" metric is: `1000.0 / (sum_of_frametimes / count)`

The default percentiles (`benchmark_percentiles=97,AVG,1,0.1`) produce the standard header columns. Custom percentiles change the first N columns of the summary.

---

## On-Screen Benchmark Results

After logging stops, MangoHud displays benchmark results on-screen for ~12 seconds showing:
- "Logging Finished"
- Duration
- Configured percentile values (from `benchmark_percentiles`)
- A frametime plot (line graph or histogram depending on `histogram` config)

This uses the same `benchmark.percentile_data` calculated by `Logger::calculate_benchmark_data()` using the `fpsMetrics` class.

---

## Upload to FlightlessMango.com

When uploading is triggered (via keybind or `upload_logs`), MangoHud executes curl:

```bash
curl --include --request POST https://flightlessmango.com/logs \
  -F 'log[game_id]=26506' \
  -F 'log[user_id]=176' \
  -F 'attachment=true' \
  -A 'mangohud' \
  -F 'log[uploads][]=@/path/to/logfile.csv'
```

Multiple log files from a session can be batch-uploaded. The response contains a redirect URL which MangoHud opens via `xdg-open`.

**Note:** This uploads to flightlessmango.com (the original site), NOT to FlightlessSomething instances. FlightlessSomething has its own upload API.

---

## Logging Lifecycle

1. **Start:** `Logger::start_logging()` — records start time, generates filename, optionally spawns background thread (if `log_interval > 0`)
2. **First write:** `writeToFile()` — opens file, writes 3 header lines, writes first data row
3. **Ongoing:** Each `try_log()` call appends one data row to the file
4. **Auto-stop:** If `log_duration` is set and elapsed time exceeds it, `stop_logging()` is called automatically
5. **Stop:** `Logger::stop_logging()` — stops the logging flag, joins background thread (if any), calls `calculate_benchmark_data()`, closes the output file, writes summary CSV
6. **Multiple sessions:** Starting logging again creates a new file with a new timestamp. All files from a session are tracked in `m_log_files` vector for batch upload.
