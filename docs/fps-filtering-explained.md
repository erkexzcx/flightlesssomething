# 97% FPS Filtering - Technical Explanation

## Problem Statement

### What is 97% FPS?
97% FPS (97th percentile) represents the FPS value that 97% of frames achieve or exceed. In other words, only the worst 3% of frames fall below this value. It's used to measure **high-end performance** during gameplay.

### Why was it inflated?

The previous implementation calculated 97% FPS using **all frametimes**, including:
- **Loading screens**: Often run at 200-500+ FPS (game is just loading assets, not rendering complex scenes)
- **Menu screens**: Often uncapped or very high FPS (simple 2D UI rendering)
- **Cutscenes**: Variable FPS, sometimes higher than gameplay
- **Paused moments**: Very high FPS when nothing is moving

This caused two major issues:

#### Issue 1: Capped Runs (e.g., 60 FPS V-Sync)
When a game is capped at 60 FPS during gameplay, but loading screens run uncapped:

**Before the fix:**
```
Gameplay frames: 1000 frames at ~60 FPS
Loading frames: 100 frames at 200+ FPS

Calculation uses ALL frames:
- Converts to frametimes
- Sorts frametimes
- Takes 3rd percentile (fastest frames)
- Result: 97% FPS = 85 FPS ❌ (inflated by loading screens)
```

**This is wrong!** You can't have 97% FPS of 85 when the game is capped at 60.

#### Issue 2: Uncapped Runs
In uncapped gameplay with menu/cutscene spikes:

**Before the fix:**
```
Gameplay frames: 1000 frames at 80-100 FPS
Menu frames: 50 frames at 400+ FPS

Result: 97% FPS = 150+ FPS ❌ (inflated by menu spikes)
```

**This is misleading!** The actual gameplay performance is 80-100 FPS, not 150+.

## The Solution

### Step 1: Auto-detect Capped Runs

Check if the run is capped at a common FPS value:
```javascript
function detectFPSCap(fpsData) {
  const commonCaps = [30, 60, 120, 144, 165, 240]
  
  for (const cap of commonCaps) {
    const countNearCap = fpsData.filter(fps => Math.abs(fps - cap) < 1).length
    const percentageNearCap = countNearCap / fpsData.length
    
    // If >80% of frames are near this cap, it's capped
    if (percentageNearCap > 0.8) {
      return cap
    }
  }
  
  return null // Not capped
}
```

### Step 2: Filter Extreme Frames

Remove non-gameplay frames using smart thresholds:

```javascript
function filterExtremeFrames(fpsData) {
  const detectedCap = detectFPSCap(fpsData)
  
  let threshold
  if (detectedCap !== null) {
    // Capped run: Remove frames > cap × 1.5
    // Example: 60 FPS cap → remove frames > 90 FPS
    threshold = detectedCap * 1.5
  } else {
    // Uncapped run: Remove frames > median × 3
    // Example: median 90 FPS → remove frames > 270 FPS
    const median = calculateMedian(fpsData)
    threshold = median * 3
  }
  
  return fpsData.filter(fps => fps <= threshold)
}
```

### Step 3: Calculate Percentile on Filtered Data

Apply filtering only for high percentiles (≥90%) to preserve low percentile accuracy:

```javascript
function calculatePercentileFPS(fpsData, percentile) {
  // Apply filtering only for high percentiles
  let processedFpsData = fpsData
  if (percentile >= 90) {
    processedFpsData = filterExtremeFrames(fpsData)
  }
  
  // Convert to frametimes and calculate percentile
  // ... (rest of the calculation)
}
```

## Why This Works

### For Capped Runs (60 FPS example)

**Before:**
```
1000 frames at 60 FPS
100 frames at 250 FPS (loading)

97% FPS = ~85 FPS ❌
```

**After (with threshold of 90 FPS):**
```
1000 frames at 60 FPS (kept)
100 frames at 250 FPS (filtered out)

97% FPS = ~59 FPS ✅ (realistic!)
```

### For Uncapped Runs (90 FPS median example)

**Before:**
```
1000 frames at 80-100 FPS
50 frames at 500 FPS (menus)

97% FPS = ~150 FPS ❌
```

**After (with threshold of 270 FPS):**
```
1000 frames at 80-100 FPS (kept)
50 frames at 500 FPS (filtered out)

97% FPS = ~95 FPS ✅ (reflects actual gameplay)
```

## Important Notes

### 1% Low FPS is NOT Affected

The filtering is only applied to percentiles ≥90%. This means:
- **1% low FPS** (1st percentile) uses ALL frames, including drops
- **Average FPS** was already calculated correctly (harmonic mean)
- **97% FPS** (97th percentile) now uses filtered frames

This is intentional because:
- Low percentiles should reflect **worst performance** (including all drops)
- High percentiles should reflect **typical good performance** (excluding non-gameplay)

### Why Median × 3 for Uncapped Runs?

The factor of 3 is chosen because:
- **2× would be too strict**: Normal gameplay variance (e.g., 70-110 FPS) would be cut
- **4× would be too loose**: Some menu spikes might slip through
- **3× is the sweet spot**: Removes extreme outliers while keeping normal variance

### Why Cap × 1.5 for Capped Runs?

The factor of 1.5 is chosen because:
- **Accounts for minor cap escapes**: Some games allow 1-2 frames above cap
- **Removes loading screen spikes**: Loading screens often 2-5× the cap
- **Example**: 60 FPS cap → threshold 90 FPS (removes 100+ FPS loading screens)

## Testing

Comprehensive unit tests cover:
- ✅ Capped run detection (60, 120, 144 FPS)
- ✅ Uncapped run handling
- ✅ Filtering thresholds (cap×1.5, median×3)
- ✅ 97% FPS realistic values
- ✅ 1% low FPS unaffected
- ✅ Real-world scenarios (loading screens, menus, cutscenes)
- ✅ Edge cases (empty data, all outliers)

## Benefits

1. **Prevents unrealistic 97% FPS** in capped runs (no more 85 FPS when capped at 60)
2. **Accurately reflects gameplay performance** by excluding loading/menu spikes
3. **Works automatically** for both capped and uncapped runs
4. **Preserves low percentile accuracy** (1% low FPS unchanged)
5. **Mirrors real-world expectations** (gamers care about actual gameplay FPS)

## Comparison with Other Tools

This implementation mirrors MangoHud's approach but adds intelligent filtering:

- **MangoHud**: Calculates percentiles on all frames (same issue)
- **FlightlessSomething (before)**: Same as MangoHud
- **FlightlessSomething (after)**: Adds filtering to remove non-gameplay frames ✅

## References

- Original issue: [#issue-number]
- Similar discussion: [MangoHud percentile calculations]
- FPS measurement standards: [PC gaming performance metrics]
