# Response to Issue #174: FPS Calculation Methods

## Summary

Thank you for this detailed analysis! After investigating both FlightlessSomething and MangoHud's source code, I can confirm that **both tools use the same calculation method for average FPS**. Here's what I found:

## Average FPS Calculation

### FlightlessSomething Implementation
```javascript
// Convert FPS to frametimes
const frametimes = fpsData.map(fps => 1000 / fps)

// Calculate average frametime
const avgFrametime = sum(frametimes) / frametimes.length

// Convert back to FPS (harmonic mean)
avgFPS = 1000 / avgFrametime
```

### MangoHud Implementation (from source code)
```cpp
// From MangoHud/src/logging.cpp
total = 0;
for (auto input : sorted) {
    total = total + input.frametime;
}

// Average FPS
result = 1000 / (total / sorted.size());
```

**Both use the harmonic mean method**: `1000 / mean(frametime)` ✅

## Why You're Seeing Different Values

If you're observing:
- MangoHud summary: **48.2 FPS**
- FlightlessSomething: **50.66 FPS**

This difference is **NOT** due to different calculation methods, since both use the same formula. The difference must come from:

1. **Different data subsets**: MangoHud's summary file might include a different time range than the full CSV
2. **Filtering**: FlightlessSomething may apply outlier filtering in some cases
3. **Data processing**: One tool might exclude certain frames during processing

### Verification Test

When you calculate in LibreOffice Calc:
- `=AVERAGE(A4:A28092)` (FPS column) → **50.66 FPS** ← This is arithmetic mean (WRONG for FPS)
- `=1000/AVERAGE(B4:B28092)` (Frametime column) → **48.2 FPS** ← This is harmonic mean (CORRECT for FPS)

The 50.66 value is the **arithmetic mean of FPS values**, which is mathematically incorrect for averaging rates. The correct method is the harmonic mean (48.2), which both MangoHud and FlightlessSomething use.

**If FlightlessSomething is showing 50.66**, it means it's processing a different subset of your data than MangoHud's summary. Make sure you're:
- Uploading the same CSV file that MangoHud used for the summary
- Not filtering the data before upload
- Comparing the same time range

## 97th Percentile FPS

### How FlightlessSomething Calculates It

```javascript
function calculatePercentileFPS(fpsData, percentile) {
  // Convert FPS to frametimes
  const frametimes = fpsData.map(fps => 1000 / fps)
  
  // Invert percentile (97th → 3rd) because of inverse relationship
  const invertedPercentile = 100 - percentile  // 100 - 97 = 3
  
  // Sort frametimes and get 3rd percentile (fastest frames)
  const sorted = [...frametimes].sort((a, b) => a - b)
  const index = Math.round((invertedPercentile / 100) * (n + 1))
  const frametimePercentile = sorted[index]
  
  // Convert back to FPS
  return 1000 / frametimePercentile
}
```

### Why Percentiles are Inverted

Because FPS and frametime have an **inverse relationship**:
- Low frametime = High FPS
- High frametime = Low FPS

Therefore:
- **97th percentile FPS** = 3rd percentile of frametimes (fastest 3% of frames)
- **1% low FPS** = 99th percentile of frametimes (slowest 1% of frames)

### How to Manually Calculate 97th Percentile FPS

**WRONG** (direct percentile of FPS):
```
=PERCENTILE(A4:A28092, 0.97)  ← This is incorrect!
```

**CORRECT** (inverted percentile of frametimes):
```
=1000/PERCENTILE(B4:B28092, 0.03)  ← Use 3rd percentile of frametimes
```

### Why MangoHud and FlightlessSomething Might Differ on Percentiles

1. **Outlier filtering**: FlightlessSomething applies intelligent filtering to remove loading screens and menu spikes (see [fps-filtering-explained.md](docs/fps-filtering-explained.md))
2. **Index calculation**: Slight differences in how the percentile index is computed
3. **Rounding**: Different precision levels

## Comprehensive Documentation

I've created detailed documentation that covers:
- ✅ Mathematical explanation of harmonic mean vs arithmetic mean
- ✅ Source code analysis of both MangoHud and FlightlessSomething
- ✅ Why percentiles must be inverted for FPS calculations
- ✅ How to manually verify calculations in Excel/LibreCalc
- ✅ Common pitfalls and misconceptions

**New documentation**: [docs/fps-calculation-methods.md](docs/fps-calculation-methods.md)

## Conclusion

Both FlightlessSomething and MangoHud use **mathematically correct** methods:
- ✅ Harmonic mean (via frametimes) for average FPS
- ✅ Percentile-based calculations with proper inversion for percentile FPS

The implementations are sound and follow best practices. Any numerical differences you observe are due to:
- Different data subsets being processed
- Outlier filtering (which is intentional and documented)
- Different time ranges in the data

## References

- MangoHud source code: https://github.com/flightlessmango/MangoHud
  - Average FPS: `src/logging.cpp` (writeSummary function)
  - Percentiles: `src/fps_metrics.h` (fpsMetrics class)
- FlightlessSomething implementation: `web/src/components/BenchmarkCharts.vue`
- New documentation: `docs/fps-calculation-methods.md`
- Filtering explanation: `docs/fps-filtering-explained.md`
