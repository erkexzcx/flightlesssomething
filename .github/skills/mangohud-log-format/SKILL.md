---
name: mangohud-log-format
description: "MangoHud CSV log file format specification — file structure, system info header, data columns (FPS, frametime, CPU/GPU metrics), elapsed time, log_versioning variant, summary files, file naming conventions, and configuration options for generating logs. Use when: parsing MangoHud CSV files, writing MangoHud CSV exporters, validating benchmark upload data, understanding column data types/units/ranges, or debugging MangoHud log parsing issues."
---

# MangoHud Log File Format Specification

Based on MangoHud source code (`src/logging.cpp`, `src/logging.h`, `src/overlay.cpp`, `src/overlay_params.h`).

Source repository: https://github.com/flightlessmango/MangoHud

---

## File Overview

MangoHud generates CSV log files containing per-frame hardware telemetry data captured during gaming sessions. Each file represents a single logging session (one "run") for one application.

- **Extension:** `.csv`
- **Encoding:** UTF-8, LF line endings
- **Delimiter:** Comma (`,`)
- **No quoting** of fields — values are written raw with no CSV escaping

---

## Standard Format (Default)

The standard log file has exactly **3 header lines** followed by data rows:

```
Line 1: System info field names (fixed string)
Line 2: System info values (comma-separated)
Line 3: Data column headers (fixed string)
Line 4+: Data rows (one per sample)
```

### Line 1 — Format Identifier

```
os,cpu,gpu,ram,kernel,driver,cpuscheduler
```

This line is **always identical** and serves as the file format identifier. Parsers detect MangoHud CSV files by checking if the first line equals exactly this string (after trimming trailing commas/whitespace).

### Line 2 — System Info Values

Seven comma-separated values corresponding to the header fields:

| Field | Source | Description | Example |
|-------|--------|-------------|---------|
| `os` | `/etc/os-release` PRETTY_NAME | Operating system name | `Steam Runtime 3 (sniper)` |
| `cpu` | `/proc/cpuinfo` model name | CPU model string (parenthetical text stripped) | `AMD Ryzen 7 9800X3D 8-Core Processor` |
| `gpu` | PCI ID lookup + driver info | GPU name with driver in parens | `AMD Radeon RX 9070 XT (RADV GFX1201)` |
| `ram` | `/proc/meminfo` MemTotal | Total system RAM in **kilobytes** (raw integer) | `65438132` |
| `kernel` | `uname -r` | Linux kernel version | `6.17.8-2-cachyos` |
| `driver` | OpenGL version query | Graphics driver version | (often empty — detection largely disabled) |
| `cpuscheduler` | sysfs `scaling_governor` | CPU frequency scaling governor | `performance` |

**Critical notes:**
- The `ram` field is in **kilobytes** — raw value from `/proc/meminfo`. To get human-readable: `65438132` KB = ~62.4 GiB
- The `driver` field is frequently **empty**, producing consecutive commas (e.g., `,,performance`)
- Fields may contain spaces, parentheses, and special characters but should not contain raw commas
- Empty fields produce consecutive commas

**Example:**
```
Steam Runtime 3 (sniper),AMD Ryzen 7 9800X3D 8-Core Processor,AMD Radeon RX 9070 XT (RADV GFX1201),65438132,6.17.8-2-cachyos,,performance
```

### Line 3 — Data Column Headers

**Current format (MangoHud ~0.8.x+, 16 columns):**
```
fps,frametime,cpu_load,cpu_power,gpu_load,cpu_temp,gpu_temp,gpu_core_clock,gpu_mem_clock,gpu_vram_used,gpu_power,ram_used,swap_used,process_rss,cpu_mhz,elapsed
```

**Older format (MangoHud ~0.7.x and earlier, 15 columns):**
```
fps,frametime,cpu_load,cpu_power,gpu_load,cpu_temp,gpu_temp,gpu_core_clock,gpu_mem_clock,gpu_vram_used,gpu_power,ram_used,swap_used,process_rss,elapsed
```

The difference is the `cpu_mhz` column, added in a later version. **Parsers must handle both by reading the header dynamically** rather than assuming fixed column positions.

### Lines 4+ — Data Rows

One row per sample. Example (16-column format):
```
238.813,4.18737,21.6521,0,84,54,51,3121,1358,4.93264,255,10.734,0,0,3500,333365637
```

---

## Log Versioning Format (Extended)

When `log_versioning` is enabled in MangoHud config, additional lines are prepended:

```
v1
0.8.2
---------------------SYSTEM INFO---------------------
os,cpu,gpu,ram,kernel,driver,cpuscheduler
<system info values>
--------------------FRAME METRICS--------------------
fps,frametime,cpu_load,...
<data rows>
```

**Extra lines before standard content:**
1. Version tag: `v1`
2. MangoHud version string (e.g., `0.8.2`)
3. Section separator: `---------------------SYSTEM INFO---------------------`
4. (Then standard Line 1–2)
5. Section separator: `--------------------FRAME METRICS--------------------`
6. (Then standard Line 3+)

**Detection:** If the first line is `v1` instead of the system info header, it's the versioned format. Skip to the standard content.

**Note:** This format is documented as "not supported on flightlessmango.com (yet)" in MangoHud's own config comments.

---

## Data Characteristics

### Sampling Behavior

- Hardware metrics (CPU/GPU load, temps, clocks, memory) update on a **separate thread** at `fps_sampling_period` rate (default 500ms)
- FPS and frametime are per-frame
- Multiple consecutive rows often have **identical hardware metrics** with different FPS/frametime — this is expected, not a bug
- The `elapsed` column always increases monotonically

### Zero Values

Zero values typically mean the metric sensor is **unavailable**, not that the actual value is zero:

| Field | When zero means unavailable |
|-------|-----------------------------|
| `cpu_power` | No RAPL access or zenpower/zenergy driver |
| `gpu_power` | GPU power sensor not available |
| `process_rss` | Process memory tracking unavailable |
| `cpu_mhz` | CPU frequency monitoring unavailable |
| `swap_used` | No swap configured (or genuinely 0 usage) |

### Edge Cases

- FPS can be extremely high (thousands) during loading screens or menus
- Frametime can be very small (<0.1ms) or very large (>1000ms during stalls)
- Temperature values are integers (truncated, not rounded)
- First few rows after logging starts may have stale/default hardware metrics
- The first `elapsed` value is typically 300–500ms, not 0, due to initialization delay

---

## Parsing Guidelines

1. **Detect format by first line:** If equals `os,cpu,gpu,ram,kernel,driver,cpuscheduler` → standard format. If equals `v1` → versioned format, skip extra headers.
2. **Trim trailing commas/whitespace** from the first line before comparison.
3. **Parse column headers dynamically:** Read the header name row and map column names to indices. Never hardcode positions.
4. **Handle variable column counts:** Older logs have 15 data columns (no `cpu_mhz`), newer have 16.
5. **Parse all data values as float64** for simplicity — integer columns will just have no decimal part. The `elapsed` column is a large integer (nanoseconds) that fits in float64 without precision loss for typical session durations.
6. **RAM in system info is in KB:** Convert to human-readable (e.g., `65438132` KB → ~62.4 GiB).
7. **Ignore non-numeric values gracefully:** If a field can't be parsed as a number, skip it rather than failing.
