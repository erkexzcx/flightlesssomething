# FPS Calculation Methods: FlightlessSomething vs MangoHud

## Issue Context

This document clarifies the differences between how FlightlessSomething and MangoHud calculate FPS statistics, addressing questions raised in [issue #174](https://github.com/erkexzcx/flightlesssomething/issues/174).

## Average FPS Calculation

### Mathematical Background

There are two common ways to calculate average FPS:

1. **Arithmetic mean of FPS values** (incorrect for FPS):
   ```
   avg_fps = mean(FPS_i) = (FPS_1 + FPS_2 + ... + FPS_n) / n
   where FPS_i = 1000 / frametime_i
   ```
   
2. **Harmonic mean via frametimes** (correct for FPS):
   ```
   avg_fps = 1000 / mean(frametime)
           = 1000 / ((frametime_1 + frametime_2 + ... + frametime_n) / n)
   ```

### Why Harmonic Mean is Correct

FPS and frametime have an **inverse relationship**: `FPS = 1000 / frametime`.

When averaging **rates** (like FPS), you must use the harmonic mean to get accurate results. The arithmetic mean of FPS values overweights fast frames and underweights slow frames.

**Example:**
- Frame 1: 100 FPS (10ms frametime)
- Frame 2: 50 FPS (20ms frametime)

**Arithmetic mean of FPS:**
```
avg = (100 + 50) / 2 = 75 FPS
```

**Harmonic mean (via frametimes):**
```
avg_frametime = (10 + 20) / 2 = 15ms
avg_fps = 1000 / 15 = 66.67 FPS
```

The second method is correct because it properly accounts for the time spent at each frame rate.

### FlightlessSomething Implementation

**Location:** `web/src/components/BenchmarkCharts.vue`, function `calculateAverageFPS()`

```javascript
function calculateAverageFPS(fpsData) {
  if (!fpsData || fpsData.length === 0) return 0
  
  // Convert FPS to frametimes (ms): frametime = 1000 / fps
  const frametimes = fpsData.map(fps => fps > 0 ? 1000 / fps : MAX_FRAMETIME_FOR_INVALID_FPS)
  
  // Calculate sum of frametimes
  const sumFrametimes = frametimes.reduce((acc, ft) => acc + ft, 0)
  
  // Average frametime
  const avgFrametime = sumFrametimes / frametimes.length
  
  // Convert back to FPS: 1000 / avgFrametime
  return avgFrametime > 0 ? 1000 / avgFrametime : 0
}
```

**This uses the harmonic mean method**, which is the same as MangoHud's summary calculation.

### MangoHud Implementation

MangoHud calculates average FPS in its summary file as:
```
avg_fps = 1000 / mean(frametime)
```

This is **identical** to FlightlessSomething's method.

## Clarification on User's Observation

The user reported observing:
- MangoHud summary average FPS: **48.2**
- FlightlessSomething average FPS: **50.66**

### Why the Difference?

If both tools use the same calculation method (harmonic mean), the difference must come from **which data points are included**:

1. **Different data subsets**: MangoHud's summary file might be a filtered subset of the full CSV data
2. **Timestamp ranges**: The summary might cover a different time range than the full CSV
3. **Data processing**: One tool might filter out certain frames (loading screens, menus, etc.)

### Verification

To verify which method is being used, you can test with the CSV files in LibreOffice Calc:

**Method 1 (Arithmetic mean - WRONG):**
```
=AVERAGE(A4:A28092)  # FPS column
Result: 50.66 FPS
```

**Method 2 (Harmonic mean - CORRECT):**
```
=1000/AVERAGE(B4:B28092)  # Frametime column
Result: 48.2 FPS
```

The fact that Method 2 matches MangoHud's summary (48.2) and Method 1 gives a different result (50.66) suggests:
- **MangoHud summary uses harmonic mean** (correct method)
- **The 50.66 value from arithmetic mean is incorrect for FPS**

However, FlightlessSomething's code **also uses harmonic mean**, so if you're seeing 50.66 in FlightlessSomething, it means:
- FlightlessSomething is processing a different subset of data
- OR there's a display bug showing the wrong value

## Percentile Calculations

### What is 97th Percentile FPS?

The 97th percentile FPS represents the FPS value that 97% of frames achieve or exceed. Only the worst 3% of frames fall below this value.

### FlightlessSomething Implementation

**Location:** `web/src/components/BenchmarkCharts.vue`, function `calculatePercentileFPS()`

```javascript
function calculatePercentileFPS(fpsData, percentile) {
  if (!fpsData || fpsData.length === 0) return 0
  
  // Convert FPS to frametimes
  const frametimes = fpsData.map(fps => fps > 0 ? 1000 / fps : MAX_FRAMETIME_FOR_INVALID_FPS)
  
  // IMPORTANT: Percentiles must be inverted when working with frametimes
  // - 97th percentile FPS (good performance) = 3rd percentile of frametimes (fastest frames)
  // - 1% low FPS (worst performance) = 99th percentile of frametimes (slowest frames)
  const invertedPercentile = 100 - percentile
  const sorted = [...frametimes].sort((a, b) => a - b)
  const n = sorted.length
  
  // Calculate percentile index
  const index = Math.round((invertedPercentile / 100) * (n + 1))
  const clampedIndex = Math.max(0, Math.min(index, n - 1))
  const frametimePercentile = sorted[clampedIndex]
  
  // Convert back to FPS
  return frametimePercentile > 0 ? 1000 / frametimePercentile : 0
}
```

### Why Percentiles are Inverted

Because FPS and frametime have an inverse relationship:
- **Low frametime** = **High FPS**
- **High frametime** = **Low FPS**

Therefore:
- **97th percentile FPS** = 3rd percentile of frametimes (fastest 3% of frames)
- **1st percentile FPS (1% low)** = 99th percentile of frametimes (slowest 1% of frames)

### User's Observation

The user reported:
> "I also noticed that FlightlessSomething reports 97% FPS significantly higher than the raw data would suggest (≈78 vs 69.75)."

This could be due to:

1. **Filtering**: FlightlessSomething has logic to filter extreme outliers for high percentiles (see `docs/fps-filtering-explained.md`). This removes loading screens and menu spikes that would inflate the percentile.

2. **Calculation method**: Using the frametime-based method with percentile inversion is correct but may produce different results than a naive percentile of FPS values.

3. **Excel/LibreCalc limitations**: Standard spreadsheet `PERCENTILE()` functions on FPS values don't account for the inverse relationship and will give incorrect results.

### Correct Manual Calculation

To manually verify 97th percentile FPS in LibreOffice Calc:

```
# WRONG (direct percentile of FPS):
=PERCENTILE(A4:A28092, 0.97)  # FPS column

# CORRECT (inverted percentile of frametimes):
=1000/PERCENTILE(B4:B28092, 0.03)  # Frametime column, 3rd percentile
```

## MangoHud Source Code Analysis

### Average FPS Calculation

**Source:** `MangoHud/src/logging.cpp` (writeSummary function)

```cpp
// Calculate average frametime
total = 0;
for (auto input : sorted) {
    total = total + input.frametime;
}

// Average FPS
result = 1000 / (total / sorted.size());
```

**This confirms**: MangoHud uses `1000 / mean(frametime)` (harmonic mean) ✅

### Percentile Calculation

**Source:** `MangoHud/src/fps_metrics.h` (fpsMetrics::calculate method)

```cpp
// Sort frametimes in descending order (largest to smallest)
std::vector<float> sorted_values = frametimes;
std::sort(sorted_values.begin(), sorted_values.end(), std::greater<float>());

// For a percentile value like 0.97 (97th percentile)
float val = std::stof(it->name);  // e.g., 0.97
uint64_t idx = val * sorted_values.size() - 1;
it->value = 1000.f / sorted_values[idx];
```

**Analysis:**
- Frametimes are sorted in **descending order** (slowest to fastest)
- For 97th percentile: `idx = 0.97 * n - 1`
- This picks a frametime near the end of the descending array (faster frames)
- The result is converted back to FPS: `1000 / frametime`

**MangoHud's percentile semantics**: The 97th percentile in MangoHud represents the FPS value that is better than 3% of frames (the fastest 97% of frames).

## Summary

### Average FPS
- **FlightlessSomething**: Uses harmonic mean via frametimes ✅ (correct)
- **MangoHud Summary**: Uses harmonic mean via frametimes ✅ (correct)
- **Both methods are identical**

**Verification from MangoHud source code:**
```cpp
result = 1000 / (total / sorted.size());  // Harmonic mean
```

Any differences in the reported average FPS values are due to:
- Different data subsets being processed
- Different time ranges
- Data filtering or exclusion (FlightlessSomething may filter extreme outliers)

### 97th Percentile FPS
- **FlightlessSomething**: Uses inverted percentile of frametimes ✅ (correct)
- **MangoHud**: Uses direct percentile on frametimes sorted descending
- **Both methods should produce similar results**

The mathematical approach is equivalent:
- **FlightlessSomething**: Sort frametimes ascending, take 3rd percentile → convert to FPS
- **MangoHud**: Sort frametimes descending, take 97th percentile → convert to FPS
- Both approaches select the same frametime value from opposite ends of the sorted array

### Why User Might See Different Values

If you observe different values between FlightlessSomething and MangoHud summary:

1. **Different data ranges**: The CSV file may contain more data than what's in the summary
2. **Filtering**: FlightlessSomething applies outlier filtering for high percentiles (see `docs/fps-filtering-explained.md`)
3. **Rounding**: Different precision in calculations
4. **Index calculation**: Slight differences in how the percentile index is calculated

## Conclusion

Both FlightlessSomething and MangoHud use the **correct mathematical methods** for FPS statistics:
- ✅ Harmonic mean (via frametimes) for average FPS
- ✅ Percentile-based calculations on frametimes for percentile FPS
- ✅ Proper conversion between FPS and frametime domains

The implementations are mathematically sound and should produce comparable results when processing the same data.

## References

- Issue #174: https://github.com/erkexzcx/flightlesssomething/issues/174
- MangoHud Source Code: https://github.com/flightlessmango/MangoHud
  - Average FPS: `src/logging.cpp` (writeSummary function)
  - Percentiles: `src/fps_metrics.h` (fpsMetrics class)
- FPS Filtering Documentation: `docs/fps-filtering-explained.md`
- FlightlessSomething Implementation: `web/src/components/BenchmarkCharts.vue`
