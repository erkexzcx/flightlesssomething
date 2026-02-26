# Benchmark Capture Guide

How to capture and upload gaming benchmark data to FlightlessSomething.

## Supported Formats

FlightlessSomething accepts benchmark data from two FPS monitoring tools:

- **MangoHud** (Linux) — `.csv` files
- **MSI Afterburner** (Windows) — `.hml` files

## Linux — MangoHud

### 1. Install MangoHud

Install MangoHud using your distribution's package manager. More information can be found on the [Arch Wiki](https://wiki.archlinux.org/title/MangoHud).

Optionally, install **GOverlay** — a graphical application for configuring the MangoHud overlay.

### 2. Configure MangoHud

Edit `~/.config/MangoHud/MangoHud.conf` with the following contents (read the comments and update accordingly):

```ini
legacy_layout=false

background_alpha=0.6
round_corners=0
background_color=000000

font_size=24
text_color=FFFFFF
position=top-left
toggle_hud=Shift_R+F12
table_columns=3
gpu_text=GPU
gpu_stats
gpu_temp
cpu_text=CPU
cpu_stats
core_load
core_bars
cpu_temp
io_stats
io_read
io_write
vram
vram_color=AD64C1
ram
ram_color=C26693
fps
gpu_name
frame_timing
frametime_color=00FF00
fps_limit_method=late
toggle_fps_limit=Shift_L+F1
fps_limit=0

# Update to your preferred logs location:
output_folder=/home/user/mangohud_logs

# Maximum log duration in seconds. Logging will auto-stop after this.
# Set to something large (e.g. 9999) if you don't know the benchmark duration.
log_duration=90

# Seconds to wait before auto-starting the log. Useful to give the game time to load.
# Set to 0 to disable auto-start (you will start logging manually with the toggle key).
autostart_log=0

# How frequently data is collected, in milliseconds:
#   100  — good default for most benchmarks
#   50   — more data points, suitable for short benchmarks
#   200+ — suitable for very long benchmarks
#   0    — capture every frame for maximum detail
#
# IMPORTANT: If comparing Linux vs Windows results, use the same interval on both!
log_interval=100

toggle_logging=Shift_L+F2
```

### 3. Capture a Benchmark

1. Start your game with MangoHud enabled (e.g. in Steam, set launch options to `mangohud %command%`).
2. The MangoHud overlay should be visible in-game.
3. Press **Shift_L+F2** (left Shift + F2) to start logging. A red dot will appear in the overlay to indicate recording is in progress.
4. Run your benchmark or play through the test scene.
5. Press **Shift_L+F2** again to stop logging, or wait for `log_duration` to expire.

### 4. Find Your Files

MangoHud saves CSV files to the configured `output_folder`:

```
~/mangohud_logs/
├── GameName-2025-01-15_183045.csv        ← upload this one
└── GameName-2025-01-15_183045-summary.csv ← can be deleted
```

The `*-summary.csv` file (if present) can be deleted — it is not used by FlightlessSomething.

## Windows — MSI Afterburner + RTSS

### 1. Install MSI Afterburner

Download and install [MSI Afterburner](https://www.msi.com/Landing/afterburner/graphics-cards). The installer also includes **RivaTuner Statistics Server (RTSS)**.

### 2. Configure MSI Afterburner

Open MSI Afterburner and go to **Settings** → **Monitoring** tab.

1. Set **Hardware polling period** to **100 ms** (ensure this matches your MangoHud `log_interval` if comparing Linux vs Windows!).
2. Disable all monitoring graphs, then enable the following:
   - GPU temperature
   - GPU usage
   - Memory usage
   - Core clock
   - Memory clock
   - Power
   - CPU temperature
   - CPU usage
   - RAM usage
   - Framerate
   - Frametime
3. *(Optional)* Click each metric and check **"Show in On-Screen Display"** to see them in the overlay.
4. Check **"Log history to file"** and select a location for the log file (e.g. Desktop or Downloads). This step is needed to configure the file path.
5. Check **"Recreate existing log files"** so each session creates a fresh file.
6. Now **uncheck** "Log history to file" — this prevents Afterburner from automatically recording when a game starts. You will control recording manually with the shortcuts configured below.
7. Set **"Begin logging"** and **"End logging"** shortcuts (suggestion: **Shift+F2** and **Shift+F3**).
8. Close MSI Afterburner settings.

### 3. Capture a Benchmark

1. Ensure both MSI Afterburner and RivaTuner Statistics Server are running (check the system tray).
2. Start your game. The overlay should appear within 5–30 seconds.
3. When ready to benchmark, press the **Begin logging** shortcut (e.g. **Shift+F2**).
4. Run your benchmark or play through the test scene.
5. Press the **End logging** shortcut (e.g. **Shift+F3**) to stop recording.

> **Note:** There is no visual indication in the overlay that recording is in progress, unlike MangoHud.

### 4. Find Your Files

MSI Afterburner saves `.hml` files to the location you configured. The default location is:

```
C:\Program Files (x86)\MSI Afterburner\HardwareMonitoring.hml
```

> **Note:** Each logging session overwrites the previous file. Rename or move your `.hml` file immediately after each recording.

## Uploading to FlightlessSomething

1. Log in to FlightlessSomething using your Discord account.
2. Navigate to **Create Benchmark**.
3. **Upload files** — select one or more `.csv` (MangoHud) or `.hml` (Afterburner) files.
4. **Edit labels** — each file gets a default label based on its filename. Edit the labels to describe each run (e.g. `Linux BORE`, `Windows Default`, `Ray Tracing On`).
5. **Add details:**
   - **Title** (required, max 100 characters) — game name or benchmark description.
   - **Description** (optional, max 5,000 characters) — hardware specs, settings, notes. Markdown is supported.
6. Click **Upload Benchmark**.

### File Naming Tips

Rename your benchmark files **before uploading** to use meaningful labels, since the filename (without extension) becomes the default run label. For example:

- `Linux.csv` → label will be **Linux**
- `Windows RT On.hml` → label will be **Windows RT On**

You can also edit labels directly in the upload form.

## Limits

| Limit | Value |
|-------|-------|
| Maximum total data lines across all runs | 1,000,000 |
| Maximum data lines per single run | 500,000 |
| Maximum benchmark title length | 100 characters |
| Maximum description length | 5,000 characters |

## Captured Metrics

FlightlessSomething extracts the following metrics from your benchmark files:

| Metric | MangoHud Column | Afterburner Column |
|--------|----------------|--------------------|
| FPS | `fps` | `Framerate` |
| Frame Time | `frametime` | `Frametime` |
| CPU Load | `cpu_load` | `CPU usage` |
| GPU Load | `gpu_load` | `GPU usage` |
| CPU Temp | `cpu_temp` | `CPU temperature` |
| CPU Power | `cpu_power` | — |
| GPU Temp | `gpu_temp` | `GPU temperature` |
| GPU Core Clock | `gpu_core_clock` | `Core clock` |
| GPU Memory Clock | `gpu_mem_clock` | `Memory clock` |
| GPU VRAM Used | `gpu_vram_used` | `Memory usage` |
| GPU Power | `gpu_power` | `Power` |
| RAM Used | `ram_used` | `RAM usage` |
| Swap Used | `swap_used` | — |

Not all metrics are required — FlightlessSomething will display charts only for metrics present in your files.

MangoHud additionally captures system specs (OS, CPU, GPU, RAM, kernel, CPU scheduler) from the file header. Afterburner captures the GPU name.
