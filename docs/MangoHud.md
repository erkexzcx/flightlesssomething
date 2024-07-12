# MangoHud instructions

1. Install MangoHud overlay package to your Linux distribution appropriately. More information can be found [here](https://wiki.archlinux.org/title/MangoHud).
2. (Optional) install `Goverlay` application which allows you to configure MangoHud overlay with graphical UI.
3. Edit `~/.config/MangoHud/MangoHud.conf` with the following contents (read the in-code comments):

```ini
legacy_layout=false

background_alpha=0.6
round_corners=0
background_alpha=0.6
background_color=000000

font_size=24
text_color=FFFFFF
position=top-left
toggle_hud=Shift_R+F12
pci_dev=0:0b:00.0
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

# Update to your preferred logs location here:
output_folder=/home/user/mangohud_logs

# Set this to maximum log duration (in seconds). It will autostop after this duration, which is useful if
# you know the duration of your benchmark, otherwise set to something large, like 9999...
log_duration=90

# If your application starts right into the benchmark - setting this to e.g. '10' gives game 10s to load. If you
# don't want it to autostart logging the data - leave this set to '0'.
autostart_log=0

# Set this to interval of how frequently logs are collected (in miliseconds). '100' is what I use, 50 provides
# more data and is suitable for short benchmarks, while 200-500 is suitable for (very) long benchmarks.
# 
# NOTE - If you are comparing Linux and Windows, then make sure this value is identical in both!
# 
log_interval=100

toggle_logging=Shift_L+F2
```

When you start the game, overlay should be visible. Pressing `SHIFT+F2` starts the logging and either it ends due to `log_duration` value, or can be manually stopped by pressing `SHIFT+F2` again. Note that there is an indication in overlay, where it shows big red dot, indicating that recording in progress.

After recording is done, you might end up with (or without) `*-summary.csv` file. This file can be deleted. Then there is `<game>-<timestamp>.csv` file - rename it to a label that you want to see in the website. Something like `Linux` or `something else` (with or without `.csv` extension) will work.
