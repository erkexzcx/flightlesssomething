# Performance and Memory Optimization Guide

This document describes the performance optimizations implemented in FlightlessSomething and how to tune them for your deployment.

## Memory Optimizations

### Overview

The application has been optimized to minimize RAM usage when handling large benchmark datasets. Key optimizations include:

1. **Streaming JSON responses** - Benchmark data is streamed directly to HTTP response writer
2. **Explicit garbage collection hints** - GC is triggered after serving large data
3. **Streaming data processing** - No intermediate buffers during compression/decompression
4. **Concurrent compression/decompression** - Better CPU utilization with zstd
5. **Optimized CSV export** - Buffered writing with reused allocations
6. **Garbage collector tuning** - Configurable GC aggressiveness for memory-constrained environments

### Compiler Optimizations

The following compiler optimizations are enabled by default:

- **`-ldflags="-w -s"`** - Strip debug info and symbol table (reduces binary size ~30%)
- **`-trimpath`** - Remove absolute paths for reproducible builds
- **Better compression** - Uses zstd's `SpeedDefault` level for ~20% better compression than `SpeedFastest`

Binary size: ~25MB (down from ~36MB without optimizations)

### Runtime Configuration

#### GOGC - Garbage Collection Frequency

`GOGC` controls how often the Go garbage collector runs. Lower values trigger GC more frequently, reducing memory usage at a slight CPU cost.

**Default:** 50 (more aggressive than Go's default of 100)

**Recommended settings:**
- **Low RAM (< 1GB):** `GOGC=25` - Very aggressive GC, minimal memory
- **Medium RAM (1-4GB):** `GOGC=50` - Balanced (default)
- **High RAM (> 4GB):** `GOGC=100` - Go's default, less frequent GC

**Usage:**
```bash
# Command line
GOGC=50 ./server [options]

# Docker Compose
environment:
  - GOGC=50

# Dockerfile ENV
ENV GOGC=50
```

#### GOMEMLIMIT - Soft Memory Limit

`GOMEMLIMIT` sets a soft memory limit for the Go runtime (Go 1.19+). The GC will try to keep total memory usage below this limit.

**Recommended settings:**
- Set to 80-90% of available container/system memory
- Example: For a 1GB container, use `GOMEMLIMIT=800MiB`

**Usage:**
```bash
# Command line
GOMEMLIMIT=512MiB ./server [options]

# Docker Compose
environment:
  - GOMEMLIMIT=800MiB

# Dockerfile ENV
ENV GOMEMLIMIT=800MiB
```

**Supported units:** `B`, `KiB`, `MiB`, `GiB`

### Compression Improvements

#### Storage
- **Algorithm:** zstd with `SpeedDefault` compression level
- **Concurrency:** 2 threads for compression
- **Streaming:** Direct gob → zstd → file (no intermediate buffer)

#### Retrieval
- **Concurrency:** 2 threads for decompression
- **Streaming:** Direct file → zstd → gob (no intermediate buffer)

### Memory Usage Patterns

**Before streaming optimizations:**
- Viewing benchmark: Load → Decompress → Buffer → Decode → JSON marshal → Send (2x data in memory)
- Exporting to ZIP: Load → Decompress → Buffer → Decode → CSV conversion → ZIP → Send

**After streaming optimizations:**
- Viewing benchmark: Load → Stream decompress → Decode → **Stream JSON encode** → Send (1x data in memory)
- Exporting to ZIP: Load → Stream decompress → Decode → Buffered CSV → ZIP → Send + **GC hint**

**Key improvements:**
- **GET /api/benchmarks/:id/data** - Uses `StreamBenchmarkDataAsJSON()` to stream JSON directly to response writer
- **Automatic GC hints** - `runtime.GC()` is called after serving large data to trigger garbage collection sooner
- **No duplicate copies** - Data is encoded directly to the response, avoiding intermediate JSON buffers
- **Immediate cleanup** - Data becomes eligible for garbage collection as soon as streaming completes

### Performance Tips

1. **For memory-constrained servers (< 2GB RAM):**
   ```bash
   GOGC=25 GOMEMLIMIT=400MiB ./server [options]
   ```

2. **For high-performance servers (> 4GB RAM):**
   ```bash
   GOGC=100 ./server [options]
   ```

3. **For containerized deployments:**
   - Set `GOMEMLIMIT` to 80% of container memory limit
   - Monitor memory usage with `docker stats`
   - Adjust `GOGC` based on observed patterns

4. **For very large benchmarks (> 100 runs or > 100K data points):**
   - Consider splitting into multiple benchmark uploads
   - Monitor server memory usage during operations

### Monitoring Memory Usage

Monitor the application's memory usage:

```bash
# Docker
docker stats <container-name>

# System (if running directly)
ps aux | grep server
top -p $(pgrep server)

# Go runtime metrics (enable pprof endpoint in production)
curl http://localhost:5000/debug/pprof/heap
```

### Real-World Performance Impact

**Observed memory usage (docker stats):**

Before optimization:
- Application start: ~6 MiB
- Loading large benchmark in browser: ~280 MiB (47x increase!)
- Issue: Data stayed in memory until next GC cycle

After optimization:
- Application start: ~6 MiB
- Loading large benchmark in browser: ~20-40 MiB (much lower peak)
- Data is streamed and eligible for immediate GC cleanup

**Key benefits:**
- Lower peak memory usage during benchmark viewing
- Faster return to baseline memory after requests
- Better handling of concurrent requests
- More predictable memory footprint

### Troubleshooting

**Symptom:** High memory usage despite optimizations
- **Check:** Are you working with very large benchmarks?
- **Solution:** Consider splitting large benchmarks into smaller uploads

**Symptom:** Slow GC pauses affecting response times
- **Check:** Is `GOGC` set too low (< 25)?
- **Solution:** Increase `GOGC` to 50-75 for better balance

**Symptom:** Out of memory errors in containers
- **Check:** Is `GOMEMLIMIT` configured?
- **Solution:** Set `GOMEMLIMIT` to 80% of container limit

**Symptom:** CPU usage spikes during benchmark operations
- **Check:** This is normal - zstd uses concurrent decompression
- **Solution:** No action needed - compression happens in background

## Benchmark Data Limits

To prevent abuse and ensure reasonable memory usage:

- **Maximum data points per benchmark:** 1,000,000 total across all runs
- **Maximum file upload size:** Configurable via reverse proxy (nginx, etc.)
- **Rate limiting:** 5 benchmark uploads per 10 minutes (per user)

## Future Optimizations

Potential future improvements:

1. **Incremental data format** - Replace gob encoding with a streaming-friendly format (JSON Lines, Protocol Buffers)
2. **Client-side filtering** - Allow filtering runs by label/index
3. **Lazy loading** - Load metadata first, fetch full data on demand per-run
4. **Compression level tuning** - Per-benchmark compression based on size
5. **Chunk-based streaming** - Stream individual runs instead of entire benchmark at once

## See Also

- [Testing Guide](testing.md) - Performance and load testing
- [Deployment Guide](deployment.md) - Production deployment recommendations
- [API Documentation](api.md) - API reference
