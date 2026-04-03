---
name: mangohud-log-columns
description: "MangoHud CSV log data column specifications — all 16 data columns with C++ types, output formats, units, value ranges, data sources, and update frequencies. Use when: implementing MangoHud CSV parsers, validating column data types, understanding what each metric measures, debugging unexpected values (zeros, duplicates, spikes), or mapping MangoHud columns to internal data structures."
---

# MangoHud Log Data Columns Reference

Based on the `logData` struct in `src/logging.h`, `writeToFile()` / `writeFileHeaders()` in `src/logging.cpp`, and metric population in `src/overlay.cpp`.

Source repository: https://github.com/flightlessmango/MangoHud

---

## Column Summary Table

| # | Column | C++ Type | Unit | Typical Range | Update Rate |
|---|--------|----------|------|---------------|-------------|
| 1 | `fps` | `double` | frames/sec | 1–1000+ | per frame |
| 2 | `frametime` | `float` | ms | 0.1–1000+ | per frame |
| 3 | `cpu_load` | `float` | % | 0–100 | ~500ms |
| 4 | `cpu_power` | `float` | watts | 0–500 | ~500ms |
| 5 | `gpu_load` | `int` | % | 0–100 | ~500ms |
| 6 | `cpu_temp` | `int` | °C | 0–110 | ~500ms |
| 7 | `gpu_temp` | `int` | °C | 0–110 | ~500ms |
| 8 | `gpu_core_clock` | `int` | MHz | 0–3500 | ~500ms |
| 9 | `gpu_mem_clock` | `int` | MHz | 0–3000 | ~500ms |
| 10 | `gpu_vram_used` | `float` | GiB | 0–24 | ~500ms |
| 11 | `gpu_power` | `int` | watts | 0–600 | ~500ms |
| 12 | `ram_used` | `float` | GiB | 0–256 | ~500ms |
| 13 | `swap_used` | `float` | GiB | 0–swap size | ~500ms |
| 14 | `process_rss` | `float` | GiB | 0–RAM size | ~500ms |
| 15 | `cpu_mhz` | `int` | MHz | 0–7000 | ~500ms |
| 16 | `elapsed` | `int64` | nanoseconds | 0–hours×3.6e12 | per frame |

**Note:** Column 15 (`cpu_mhz`) was added in a later MangoHud version. Older logs have 15 columns with `elapsed` at position 15.

---

## Detailed Column Descriptions

### fps (Column 1)

- **Source:** `fps = double(1000 / frametime_ms)` — calculated from frametime each frame
- **C++ type:** `double` (64-bit float)
- **Unit:** Frames per second
- **Precision:** 3–6 decimal places (e.g., `238.813`)
- **Notes:** Instantaneous FPS for that specific frame, NOT averaged. Derived from frametime: `fps = 1000.0 / frametime`.

### frametime (Column 2)

- **Source:** `frametime_ns / 1000000.f` — nanosecond frame timing converted to milliseconds
- **C++ type:** `float` (32-bit)
- **Unit:** Milliseconds (ms)
- **Precision:** 5–6 significant digits (e.g., `4.18737`)
- **Notes:** Raw frame-to-frame time measured as delta between consecutive Vulkan/OpenGL present calls. This is the **primary metric** — FPS is derived from it. Can spike >1000ms during stalls or loading.
- **Relationship:** `fps = 1000.0 / frametime`

### cpu_load (Column 3)

- **Source:** `cpuStats.GetCPUDataTotal().percent` — from `/proc/stat` delta calculation
- **C++ type:** `float`
- **Unit:** Percent (0–100)
- **Precision:** 1–4 decimal places (e.g., `21.6521`)
- **Notes:** Total CPU utilization across all cores. Updated at `fps_sampling_period` rate (default 500ms), NOT per frame. Multiple consecutive rows will have identical values.

### cpu_power (Column 4)

- **Source:** `cpuStats.GetCPUDataTotal().power` — from Intel RAPL or AMD zenpower/zenergy
- **C++ type:** `float`
- **Unit:** Watts (W)
- **Notes:** Frequently `0` when kernel power monitoring is unavailable. Requires readable `/sys/class/powercap/intel-rapl:0/energy_uj` for Intel, or zenpower3/zenergy driver for AMD Ryzen.

### gpu_load (Column 5)

- **Source:** `gpus->active_gpu()->metrics.load` — from GPU driver (NVML, amdgpu, i915, xe, fdinfo)
- **C++ type:** `int`
- **Unit:** Percent (0–100)
- **Notes:** GPU utilization. Source varies by vendor/driver. On Intel, may show per-process usage rather than system-wide (known Intel limitation).

### cpu_temp (Column 6)

- **Source:** `cpuStats.GetCPUDataTotal().temp` — from hwmon sysfs
- **C++ type:** `int` (truncated, not rounded)
- **Unit:** Degrees Celsius (°C)
- **Notes:** May be `0` if no suitable temperature sensor found.

### gpu_temp (Column 7)

- **Source:** `gpus->active_gpu()->metrics.temp` — from GPU driver/hwmon
- **C++ type:** `int` (truncated, not rounded)
- **Unit:** Degrees Celsius (°C)
- **Notes:** Edge temperature. Junction temp (`gpu_junction_temp`) and memory temp (`gpu_mem_temp`) are HUD-only — they are **NOT logged** to CSV.

### gpu_core_clock (Column 8)

- **Source:** `gpus->active_gpu()->metrics.CoreClock`
- **C++ type:** `int`
- **Unit:** MHz
- **Notes:** Current GPU core/shader clock frequency.

### gpu_mem_clock (Column 9)

- **Source:** `gpus->active_gpu()->metrics.MemClock`
- **C++ type:** `int`
- **Unit:** MHz
- **Notes:** Current GPU memory clock frequency. Not available on all GPU vendors.

### gpu_vram_used (Column 10)

- **Source:** `gpus->active_gpu()->metrics.sys_vram_used`
- **C++ type:** `float`
- **Unit:** GiB (binary gigabytes, 1 GiB = 1024³ bytes)
- **Precision:** 5–6 significant digits (e.g., `4.93264`)
- **Notes:** System-wide VRAM usage, not per-process. Not available on Intel GPUs.

### gpu_power (Column 11)

- **Source:** `gpus->active_gpu()->metrics.powerUsage`
- **C++ type:** `int`
- **Unit:** Watts (W)
- **Notes:** GPU board power draw. Available on NVIDIA, AMD, and Intel discrete GPUs. May be `0` if sensor unavailable.

### ram_used (Column 12)

- **Source:** `memused` — from `/proc/meminfo` (MemTotal - MemAvailable)
- **C++ type:** `float`
- **Unit:** GiB
- **Precision:** 4–5 significant digits (e.g., `10.734`)
- **Notes:** System-wide RAM usage, not per-process.

### swap_used (Column 13)

- **Source:** `swapused` — from `/proc/meminfo` (SwapTotal - SwapFree)
- **C++ type:** `float`
- **Unit:** GiB
- **Notes:** `0` if no swap configured or no swap in use.

### process_rss (Column 14)

- **Source:** `proc_mem_resident / float((2 << 29))` — from `/proc/self/statm`
- **C++ type:** `float`
- **Unit:** GiB (resident_bytes / 2^30)
- **Notes:** Resident set size of the game process. May be `0` if unavailable. Divisor `(2 << 29)` = `2^30` = 1 GiB.

### cpu_mhz (Column 15, newer versions only)

- **Source:** `cpuStats.GetCPUDataTotal().cpu_mhz` — from sysfs `scaling_cur_freq`
- **C++ type:** `int`
- **Unit:** MHz
- **Notes:** Added in a later MangoHud version. **Not present in older logs.** When absent, `elapsed` shifts to position 15.

### elapsed (Column 15 or 16)

- **Source:** `std::chrono::duration_cast<std::chrono::nanoseconds>(logArray.back().previous).count()` — monotonic clock delta
- **C++ type:** `int64` (large integer)
- **Unit:** Nanoseconds since logging started
- **Notes:** Always monotonically increasing. First value is typically 300–500ms (not 0) due to initialization. To convert: `elapsed_ns / 1_000_000_000 = seconds`.
- **Example:** `333365637` = ~0.333 seconds since logging started

---

## Update Frequency Groups

Not all columns update at the same rate. This explains the pattern of repeated identical values across consecutive rows:

| Rate | Columns |
|------|---------|
| **Per frame** | `fps`, `frametime`, `elapsed` |
| **Per fps_sampling_period** (default 500ms) | All hardware metrics: `cpu_load`, `cpu_power`, `cpu_temp`, `cpu_mhz`, `gpu_load`, `gpu_temp`, `gpu_core_clock`, `gpu_mem_clock`, `gpu_vram_used`, `gpu_power`, `ram_used`, `swap_used`, `process_rss` |

When `log_interval=0` (log every frame, the default), at 60 FPS you get ~30 rows between hardware metric updates. All 30 rows will have identical values for columns like `gpu_load` but different `fps`/`frametime`/`elapsed`.

---

## Type Mapping for Parsers

For parser implementations, these are the recommended type mappings:

| Parse as float64 (decimal values in CSV) | Parse as int (or float64) — integer values in CSV |
|------------------------------------------|---------------------------------------------------|
| `fps`, `frametime` | `gpu_load`, `cpu_temp`, `gpu_temp` |
| `cpu_load`, `cpu_power` | `gpu_core_clock`, `gpu_mem_clock`, `gpu_power` |
| `gpu_vram_used`, `ram_used` | `cpu_mhz`, `elapsed` |
| `swap_used`, `process_rss` | |

**Pragmatic approach:** Parse everything as `float64`. Integer columns will have no decimal part. This simplifies code and handles all MangoHud versions uniformly.

---

## GPU Vendor Metric Availability

Not all metrics are available on all GPU vendors/drivers:

| Metric | NVIDIA | AMD | Intel Discrete | Intel Integrated |
|--------|--------|-----|----------------|------------------|
| gpu_load | Yes | Yes | Yes | Yes |
| gpu_temp | Yes | Yes | Yes (i915 6.13+) | No |
| gpu_core_clock | Yes | Yes | Yes | Yes |
| gpu_mem_clock | Yes | Yes | No | No |
| gpu_vram_used | Yes | Yes | No (system-wide) | No |
| gpu_power | Yes | Yes | Yes | No |

When a metric is unavailable, the logged value will be `0`.
