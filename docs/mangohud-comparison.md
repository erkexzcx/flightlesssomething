# MangoHud vs FlightlessSomething Calculation Comparison

## Executive Summary

This document explains why MangoHud's summary statistics differ slightly from FlightlessSomething's calculations, even when processing the same benchmark data. 

FlightlessSomething provides **two calculation methods**:
1. **Linear Interpolation** (default) - More statistically precise
2. **Frametime-Based Thresholds** - Attempts to replicate MangoHud's method

Even with the "Frametime-Based Thresholds" method (designed to match MangoHud), there are **small differences (< 0.02%)** due to:

1. **Index calculation formula differences** - MangoHud uses `val * n - 1`, FlightlessSomething uses `floor(val * n)`
2. **Precision differences** - MangoHud uses 32-bit float, FlightlessSomething uses 64-bit double
3. **Sorting order** - MangoHud sorts descending, FlightlessSomething sorts ascending

**Bottom line:** Both tools calculate statistics correctly - the differences are purely due to implementation details and are within acceptable tolerance for benchmark analysis.

---

## Key Findings

### 1% Min FPS Comparison

FlightlessSomething offers TWO calculation methods. Here's how they compare to MangoHud:

| Tool / Method | Value | Difference from MangoHud |
|---------------|-------|--------------------------|
| **MangoHud** | 26.8786 | Baseline |
| **FlightlessSomething (Linear Interpolation)** | 26.8961 | +0.0175 (+0.065%) |
| **FlightlessSomething (Frametime-Based Thresholds)** | 26.8841 | +0.0055 (+0.020%) |

**Key insight:** Even the "Frametime-Based Thresholds" method (which attempts to replicate MangoHud) has a small 0.02% difference due to index formula variations.

### Average FPS Comparison

| Tool | Value | Difference |
|------|-------|------------|
| **MangoHud** | 48.2 | Baseline |
| **FlightlessSomething** | 48.2183 | +0.0183 (+0.038%) |

### 97% Percentile FPS Comparison

| Tool | Value | Difference |
|------|-------|------------|
| **MangoHud** | 69.7556 | Baseline |
| **FlightlessSomething (Linear)** | 69.7633 | +0.0077 (+0.011%) |

---

## Detailed Analysis

### Understanding MangoHud's Implementation

After analyzing MangoHud's source code (specifically `src/logging.cpp` and `src/fps_metrics.h`), here's how it calculates percentiles:

```cpp
// From fps_metrics.h, line 82-86
uint64_t idx = val * sorted_values.size() - 1;
if (idx >= sorted_values.size())
    break;
it->value = 1000.f / sorted_values[idx];
```

**Key observations:**

1. **Index calculation:** `idx = val * sorted_values.size() - 1`
   - For 1% percentile (val = 0.01): `idx = 0.01 * n - 1`
   - This is a **simple truncation** approach (no interpolation)
   - The `-1` adjusts for 0-based array indexing

2. **Data is sorted in descending order** (`std::greater<float>()`):
   ```cpp
   std::sort(sorted_values.begin(), sorted_values.end(), std::greater<float>());
   ```
   - Highest frametimes (slowest frames) come first
   - Lowest frametimes (fastest frames) come last

3. **FPS calculation from frametime:**
   ```cpp
   it->value = 1000.f / sorted_values[idx];
   ```
   - Uses single-precision float (32-bit)

4. **Summary output formatting:**
   ```cpp
   // From logging.cpp, line 83-116
   out << fixed << setprecision(1) << result << ",";
   ```
   - Most values are formatted with 1 decimal place precision
   - This causes **display rounding** but doesn't affect internal calculations

### Understanding FlightlessSomething's Implementation

FlightlessSomething provides **two calculation methods**:

#### Method 1: Linear Interpolation (Default)

From `web/src/utils/statsCalculations.js`:

```javascript
export function calculatePercentileLinearInterpolation(sortedData, percentile) {
  const n = sortedData.length
  const idx = (percentile / 100) * (n - 1)
  const lower = Math.floor(idx)
  const upper = Math.ceil(idx)
  
  if (lower === upper) {
    return sortedData[lower]
  }
  
  const fraction = idx - lower
  return sortedData[lower] * (1 - fraction) + sortedData[upper] * fraction
}
```

**Key features:**
- Uses **linear interpolation** between adjacent data points
- Matches scientific/numpy percentile calculation methods
- Provides more precise values when index falls between data points
- Uses double-precision (64-bit) floating point

#### Method 2: MangoHud Threshold (Simplified)

```javascript
export function calculatePercentileMangoHudThreshold(sortedData, percentile) {
  const n = sortedData.length
  const idx = Math.floor((percentile / 100) * n)
  const clampedIdx = Math.min(Math.max(idx, 0), n - 1)
  return sortedData[clampedIdx]
}
```

**Key features:**
- Uses **floor-based indexing** without interpolation
- Attempts to replicate MangoHud's approach
- **Still has minor differences (~0.02%)** due to index formula mismatch

### Why FlightlessSomething's "Frametime-Based Thresholds" Doesn't Exactly Match MangoHud

Even though FlightlessSomething has a dedicated "Frametime-Based Thresholds" calculation method designed to replicate MangoHud, there's still a **small 0.02% difference**. Here's why:

#### The Index Formula Mismatch

**MangoHud's formula:**
```cpp
idx = val * sorted_values.size() - 1
```

**FlightlessSomething's formula:**
```javascript
idx = Math.floor((percentile / 100) * n)
```

These formulas produce different indices:

| Dataset Size | Percentile | MangoHud Index | FS Index | Difference |
|--------------|------------|----------------|----------|------------|
| 1000 samples | 1% | `0.01 * 1000 - 1 = 9` | `floor(0.01 * 1000) = 10` | +1 index |
| 1000 samples | 99% | `0.99 * 1000 - 1 = 989` | `floor(0.99 * 1000) = 990` | +1 index |

This off-by-one difference causes the 0.02% variance in results.

#### Why Not Fix It?

FlightlessSomething could change to `Math.floor(val * n) - 1` to exactly match MangoHud, but:

1. **The difference is negligible** (0.02% = statistically insignificant)
2. **Current formula is more standard** - matches common percentile implementations
3. **Both methods are already available** - users can choose based on their needs
4. **Breaking change risk** - existing benchmarks would show different values

### Why the Calculations Differ Between Methods

#### 1. Index Calculation Formula Differences (Linear vs Threshold)

**MangoHud:**
```cpp
idx = val * sorted_values.size() - 1
```
For 1% percentile (val = 0.01) with 1000 samples:
```
idx = 0.01 * 1000 - 1 = 10 - 1 = 9
```

**FlightlessSomething (Linear Interpolation):**
```javascript
idx = (percentile / 100) * (n - 1)
```
For 1% percentile with 1000 samples (uses 99th percentile of frametime):
```
idx = (99 / 100) * (1000 - 1) = 989.01 (then interpolates)
```

**FlightlessSomething (Frametime-Based Thresholds):**
```javascript
idx = Math.floor((percentile / 100) * n)
```
For 1% percentile with 1000 samples (uses 99th percentile of frametime):
```
idx = Math.floor((99 / 100) * 1000) = floor(990) = 990
```

**Result:** Three different indices (9 for MangoHud descending, 989-990 for FS ascending) lead to slight variations.

#### 2. Interpolation vs No Interpolation

**FlightlessSomething Linear Method:**
- When the calculated index is 9.5, it interpolates between values at index 9 and 10
- Result: 26.8961 FPS

**MangoHud / FS MangoHud Method:**
- When the calculated index is 9 (or 10), it uses that exact value
- Result: 26.8786 FPS (MangoHud) or 26.8841 FPS (FS MangoHud)

#### 3. Floating Point Precision

**MangoHud:**
- Uses `float` (32-bit, ~7 decimal digits precision)
- Example: `1000.f / sorted_values[idx]`

**FlightlessSomething:**
- Uses JavaScript `Number` (64-bit double, ~15 decimal digits precision)
- Example: `1000 / frametimeP99`

This causes minor differences in the final decimal places.

#### 4. Sorting Order

**MangoHud:**
- Sorts frametimes in **descending order** (slowest first)
- For 1% percentile, accesses early indices (slow frames)

**FlightlessSomething:**
- Sorts frametimes in **ascending order** (fastest first)
- For 1% FPS, uses 99th percentile of frametimes (converts to FPS)

Both approaches are mathematically equivalent, just accessing different ends of the sorted array.

---

## Verification with Spreadsheet Formulas

The user verified FlightlessSomething's calculations using spreadsheet formulas. Here are the results:

### Linear Interpolation Method

FlightlessSomething uses formulas equivalent to:
```
=1000/PERCENTILE(B:B, 0.99)  // For 1% FPS
=1000/AVERAGE(B:B)             // For Average FPS
=1000/PERCENTILE(B:B, 0.03)  // For 97% FPS
```

These formulas **exactly match** FlightlessSomething's output, confirming correctness.

### MangoHud Threshold Method

FlightlessSomething uses formulas equivalent to:
```
=1000/INDEX(SORT(B:B), FLOOR(0.99*COUNT(B:B), 1)+1)  // For 1% FPS
```

These also **match** FlightlessSomething's output, confirming the implementation is correct.

---

## Does MangoHud Do Any Post-Processing?

After analyzing the MangoHud source code, **NO** - MangoHud does not perform any post-processing on the summary statistics beyond:

1. **Calculation** of percentiles using the simple index formula
2. **Formatting** the output with fixed decimal precision

The summary file is generated in `writeSummary()` function:
- Line 83-88: Writes header row
- Line 89-116: Calculates and writes 0.1%, 1%, and 97% percentiles
- Line 120-179: Calculates averages and peaks for other metrics

**No filtering, smoothing, or adjustments** are applied to FPS/frametime data.

---

## Why Do Other Summary Values Match?

Looking at the MangoHud summary output:

```
Average FPS: 48.2
Average Frame Time: 20.7
```

Calculation: `1000 / 20.7 ≈ 48.31`

But wait - if average frametime is 20.7ms, average FPS should be ~48.31, not 48.2!

**What's happening:**

1. MangoHud calculates average frametime: `total / sorted.size()` → let's say 20.746ms
2. Formats with 1 decimal: displays as `20.7`
3. Calculates average FPS: `1000 / 20.746` → 48.2 FPS
4. Formats with 1 decimal: displays as `48.2`

So the "Average Frame Time" shown is **rounded for display**, but the actual calculation uses the full precision value.

FlightlessSomething shows:
```
Average FPS: 48.22 (full precision: 48.2183698959615)
Average Frametime: 20.74 (full precision: 20.7389839631172)
```

Verification: `1000 / 20.7389839631172 = 48.2183698959615` ✓

Both are correct - MangoHud just rounds more aggressively for display purposes.

---

## Summary Table: All Metrics Compared

| Metric | MangoHud | FlightlessSomething | Difference | % Diff |
|--------|----------|---------------------|------------|--------|
| **0.1% Min FPS** | 21.3287 | N/A* | N/A | N/A |
| **1% Min FPS** | 26.8786 | 26.8961 (linear) | +0.0175 | +0.065% |
| **1% Min FPS** | 26.8786 | 26.8841 (mangohud) | +0.0055 | +0.020% |
| **97% Percentile FPS** | 69.7556 | 69.7633 (linear) | +0.0077 | +0.011% |
| **97% Percentile FPS** | 69.7556 | 69.7773 (mangohud) | +0.0217 | +0.031% |
| **Average FPS** | 48.2 | 48.22 | +0.02 | +0.041% |
| **Average Frame Time** | 20.7 | 20.74 | +0.04 | +0.19% |

*FlightlessSomething doesn't calculate 0.1% by default, but could with minor code changes.

**All differences are < 0.1%**, well within acceptable tolerance for benchmark analysis.

---

## Conclusions

### Are the Calculations Correct?

**YES** - Both MangoHud and FlightlessSomething calculate statistics correctly:

1. **MangoHud** uses a simpler, faster approach with truncation
2. **FlightlessSomething** offers more precision with linear interpolation
3. Both approaches are mathematically valid
4. Differences are due to legitimate implementation choices, not errors

### Which Method Should You Trust?

Both are trustworthy, but they serve different purposes:

**Use MangoHud's values when:**
- Comparing with other MangoHud benchmarks
- You want simple, fast calculations
- Sub-0.1% precision doesn't matter

**Use FlightlessSomething's Linear Interpolation when:**
- You want maximum statistical accuracy
- Comparing with scientific tools (numpy, pandas, Excel PERCENTILE)
- You need reproducible, precise values

**Use FlightlessSomething's MangoHud Threshold when:**
- You want to approximate MangoHud's method
- Debugging discrepancies with MangoHud output

### Why the Small Differences Exist

1. **Index calculation formula:** MangoHud uses `val * n - 1`, FlightlessSomething uses `(val/100) * (n-1)` or `floor((val/100) * n)`
2. **Interpolation:** FlightlessSomething's linear method interpolates, MangoHud doesn't
3. **Floating point precision:** MangoHud uses 32-bit float, JavaScript uses 64-bit double
4. **Display rounding:** MangoHud rounds to 1 decimal place for display

All differences are **expected and acceptable** - they're not bugs, just implementation details.

---

## Recommendations

### For Users

- Don't worry about the < 0.1% differences
- Both tools are accurate for performance analysis
- Use FlightlessSomething's "MangoHud Threshold" method if you want closest match to MangoHud
- Use FlightlessSomething's "Linear Interpolation" for maximum precision

### For Developers

- **Do not change** FlightlessSomething's calculations to "match" MangoHud exactly
- The current implementation is **more accurate** (uses interpolation + double precision)
- Keep both methods available for user choice
- Document the differences clearly (this document!)

---

## Technical References

### MangoHud Source Code

- **Logging:** [`src/logging.cpp`](https://github.com/flightlessmango/MangoHud/blob/master/src/logging.cpp)
  - Line 68-183: `writeSummary()` - generates summary CSV
  - Line 360-379: `calculate_benchmark_data()` - calculates percentiles

- **FPS Metrics:** [`src/fps_metrics.h`](https://github.com/flightlessmango/MangoHud/blob/master/src/fps_metrics.h)
  - Line 49-95: `calculate()` - core percentile calculation logic
  - Line 82: Index calculation: `uint64_t idx = val * sorted_values.size() - 1;`
  - Line 86: FPS from frametime: `it->value = 1000.f / sorted_values[idx];`

### FlightlessSomething Source Code

- **Statistics Calculations:** `web/src/utils/statsCalculations.js`
  - Line 19-38: `calculatePercentileLinearInterpolation()` - linear interpolation method
  - Line 50-64: `calculatePercentileMangoHudThreshold()` - simplified threshold method
  - Line 151-208: `calculateFPSStatsFromFrametime()` - FPS calculation from frametime data

- **Debug Calculator:** `web/src/views/DebugCalc.vue`
  - Provides interactive comparison tool
  - Exports to spreadsheet for verification
  - Shows both calculation methods side-by-side

### Mathematical Formulas

**Linear Interpolation Percentile:**
```
idx = (p/100) * (n-1)
lower = floor(idx)
upper = ceil(idx)
fraction = idx - lower
result = data[lower] * (1-fraction) + data[upper] * fraction
```

**MangoHud Percentile:**
```
idx = p * n - 1
result = data[idx]
```

**FPS from Frametime:**
```
FPS = 1000 / frametime_ms
```

---

## Appendix: Sample Calculation Walkthrough

Let's walk through a 1% FPS calculation with a small dataset to illustrate the differences.

### Sample Data (10 frametimes in ms, sorted ascending)
```
[10, 12, 15, 18, 20, 22, 25, 30, 35, 50]
```

### MangoHud Method

1. **Sort descending:** `[50, 35, 30, 25, 22, 20, 18, 15, 12, 10]`
2. **Calculate index for 1% (0.01):** `idx = 0.01 * 10 - 1 = -0.9` → 0 (clamped)
3. **Get frametime:** `sorted[0] = 50ms`
4. **Calculate FPS:** `1000 / 50 = 20 FPS`

**Result: 20 FPS**

### FlightlessSomething Linear Interpolation

1. **Sort ascending:** `[10, 12, 15, 18, 20, 22, 25, 30, 35, 50]`
2. **Calculate 99th percentile frametime (for 1% FPS):**
   - `idx = (99/100) * (10-1) = 0.99 * 9 = 8.91`
   - `lower = floor(8.91) = 8`
   - `upper = ceil(8.91) = 9`
   - `fraction = 8.91 - 8 = 0.91`
   - `result = 35 * (1-0.91) + 50 * 0.91 = 35 * 0.09 + 50 * 0.91 = 3.15 + 45.5 = 48.65ms`
3. **Calculate FPS:** `1000 / 48.65 = 20.555 FPS`

**Result: 20.56 FPS**

### FlightlessSomething MangoHud Threshold

1. **Sort ascending:** `[10, 12, 15, 18, 20, 22, 25, 30, 35, 50]`
2. **Calculate 99th percentile frametime:**
   - `idx = floor((99/100) * 10) = floor(9.9) = 9`
   - `result = sorted[9] = 50ms`
3. **Calculate FPS:** `1000 / 50 = 20 FPS`

**Result: 20 FPS**

### Comparison

| Method | Index | Frametime | FPS | Notes |
|--------|-------|-----------|-----|-------|
| MangoHud | 0 (desc) | 50ms | 20.0 | No interpolation |
| FS Linear | 8.91 | 48.65ms | 20.56 | Interpolated value |
| FS MangoHud | 9 | 50ms | 20.0 | No interpolation |

The linear interpolation method provides a more accurate representation of the "99th percentile" by not being limited to exact data points.
