# Benchmark Guide

How to capture and upload gaming benchmark data to FlightlessSomething.

## Overview

FlightlessSomething accepts benchmark data from two popular FPS monitoring tools:
- **MangoHud** (Linux)
- **MSI Afterburner** (Windows)

Both tools capture frame time data during gameplay, which FlightlessSomething visualizes and analyzes.

## Linux - MangoHud

### Installation

Install MangoHud using your distribution's package manager:

```bash
# Arch Linux
sudo pacman -S mangohud

# Ubuntu/Debian
sudo apt install mangohud

# Fedora
sudo dnf install mangohud
```

### Configuration

Create or edit `~/.config/MangoHud/MangoHud.conf`:

```ini
# Enable logging
output_folder=/home/YOUR_USERNAME/mangohud_logs
log_duration=0 # In seconds (0=indefinitel until stopped)
log_interval=100
toggle_logging=Shift_L+F2
```

**Important settings:**
- `log_interval=100` - Sample every 100ms (matches Windows setup)
- `log_duration=0` - Log indefinitely until manually stopped
- `output_folder` - Where CSV files are saved
- `toggle_logging` - Shortcut to start/stop recording

### Capturing Benchmarks

Start game with MangoHud logging enabled in Steam:

```bash
mangohud %command%
```

**Controls:**
- Press `Shift_L+F2` to start/stop logging
- CSV files saved to configured `output_folder`
- Each run creates a new timestamped CSV file

### Finding Your Files

MangoHud saves CSV files to the configured output folder:

```bash
ls ~/mangohud_logs/
# Example: MangoHud_2025-11-26_183045.csv
```

## Windows - MSI Afterburner + RTSS

### Installation

1. Download and install MSI Afterburner.
2. Afterburner includes RivaTuner Statistics Server (RTSS)
3. Start Afterburner which starts RTSS

### MSI Afterburner Configuration

Open MSI Afterburner settings:

1. Go to **Settings** → **Monitoring** tab
2. Select metrics to monitor (Framerate, Frametime, GPU temp, etc.)
3. Check **"Log history to file"** for each metric
4. Set **Hardware polling period** to **100 ms**

**Important MSI Afterburner settings:**
- **Hardware polling period: 100 ms** - Matches MangoHud sampling rate
- Enable logging for at least: Framerate, Frametime

### RTSS Configuration

Open RivaTuner Statistics Server settings:

⚠️ **Critical Settings**:

1. **Framerate averaging: 0** (OFF)
   - Disables frame averaging for accurate data
2. **Refresh period: 100 ms**
   - Matches MangoHud and Afterburner polling rate

### Capturing Benchmarks

1. Start your game
2. Press `F11` (default) to start logging in MSI Afterburner
3. Play your benchmark sequence
4. Press `F11` again to stop logging

### Finding Your Files

MSI Afterburner saves HML files to:

```
C:\Program Files (x86)\MSI Afterburner\HardwareMonitoring.hml
```

Or check your configured logging directory in Afterburner settings.

**Note:** Each logging session overwrites the previous HML file. Save/rename files immediately after each run.

## Uploading to FlightlessSomething

### Web UI

1. Log in to FlightlessSomething
2. Navigate to **"Create Benchmark"**
3. **Upload files:**
   - Click "Choose Files"
   - Select one or more CSV (MangoHud) or HML (Afterburner) files
   - You can upload multiple runs at once
4. **Customize labels:**
   - Each file gets a default label based on filename
   - Edit labels to describe each run (e.g., "High Settings", "Ray Tracing On")
5. **Add details:**
   - **Title** (required) - Game name or benchmark description
   - **Description** (optional) - Hardware specs, settings, notes (Markdown supported)
6. Click **"Upload Benchmark"**
