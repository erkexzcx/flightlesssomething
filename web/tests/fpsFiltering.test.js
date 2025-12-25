#!/usr/bin/env node

/**
 * Unit tests for FPS filtering logic
 * 
 * Tests the logic that filters extreme frames before calculating 97% FPS
 * to prevent non-gameplay frames (loading, menus, cutscenes) from inflating the value.
 * 
 * Run with: node tests/fpsFiltering.test.js
 */

// Simple test runner
let testsPassed = 0;
let testsFailed = 0;

function test(description, fn) {
  try {
    fn();
    console.log(`✓ ${description}`);
    testsPassed++;
  } catch (error) {
    console.error(`✗ ${description}`);
    console.error(`  ${error.message}`);
    testsFailed++;
  }
}

function assertEquals(actual, expected, message) {
  if (actual !== expected) {
    throw new Error(message || `Expected "${expected}" but got "${actual}"`);
  }
}

function assertAlmostEquals(actual, expected, tolerance = 0.01, message) {
  if (Math.abs(actual - expected) > tolerance) {
    throw new Error(message || `Expected ${expected} (±${tolerance}) but got ${actual}`);
  }
}

function assertTrue(condition, message) {
  if (!condition) {
    throw new Error(message || 'Expected condition to be true');
  }
}

// Re-implement the filtering logic from BenchmarkCharts.vue
const MAX_FRAMETIME_FOR_INVALID_FPS = 1000000;

function calculateMedian(data) {
  if (!data || data.length === 0) return 0;
  const sorted = [...data].sort((a, b) => a - b);
  const n = sorted.length;
  if (n % 2 === 0) {
    return (sorted[n / 2 - 1] + sorted[n / 2]) / 2;
  } else {
    return sorted[Math.floor(n / 2)];
  }
}

function detectFPSCap(fpsData) {
  if (!fpsData || fpsData.length === 0) return null;
  
  const commonCaps = [30, 60, 120, 144, 165, 240];
  
  for (const cap of commonCaps) {
    const countNearCap = fpsData.filter(fps => Math.abs(fps - cap) < 1).length;
    const percentageNearCap = countNearCap / fpsData.length;
    
    if (percentageNearCap > 0.8) {
      return cap;
    }
  }
  
  return null;
}

function filterExtremeFrames(fpsData) {
  if (!fpsData || fpsData.length === 0) return [];
  
  const detectedCap = detectFPSCap(fpsData);
  
  let threshold;
  if (detectedCap !== null) {
    threshold = detectedCap * 1.5;
  } else {
    const median = calculateMedian(fpsData);
    threshold = median * 3;
  }
  
  return fpsData.filter(fps => fps <= threshold);
}

function calculatePercentileFPS(fpsData, percentile) {
  if (!fpsData || fpsData.length === 0) return 0;
  
  let processedFpsData = fpsData;
  if (percentile >= 90) {
    processedFpsData = filterExtremeFrames(fpsData);
    if (processedFpsData.length === 0) {
      processedFpsData = fpsData;
    }
  }
  
  const frametimes = processedFpsData.map(fps => fps > 0 ? 1000 / fps : MAX_FRAMETIME_FOR_INVALID_FPS);
  
  const invertedPercentile = 100 - percentile;
  const sorted = [...frametimes].sort((a, b) => a - b);
  const n = sorted.length;
  
  const index = Math.round((invertedPercentile / 100) * (n + 1));
  const clampedIndex = Math.max(0, Math.min(index, n - 1));
  const frametimePercentile = sorted[clampedIndex];
  
  return frametimePercentile > 0 ? 1000 / frametimePercentile : 0;
}

console.log('Running FPS filtering tests...\n');

// Test 1: Capped 60 FPS run detection
test('detects 60 FPS cap when >80% frames are near 60', () => {
  // Simulate a 60 FPS capped run with some outliers
  const fps60Capped = [
    ...Array(900).fill(59.5),  // 90% at ~60 FPS
    ...Array(100).fill(120)     // 10% outliers (loading screens)
  ];
  
  const detectedCap = detectFPSCap(fps60Capped);
  assertEquals(detectedCap, 60, `Expected cap of 60, got ${detectedCap}`);
});

// Test 2: No cap detected for uncapped runs
test('does not detect cap for uncapped run', () => {
  // Simulate an uncapped run with variable FPS
  const fpsUncapped = [
    ...Array(100).fill(85),
    ...Array(100).fill(90),
    ...Array(100).fill(95),
    ...Array(100).fill(100),
  ];
  
  const detectedCap = detectFPSCap(fpsUncapped);
  assertEquals(detectedCap, null, `Expected no cap, got ${detectedCap}`);
});

// Test 3: Filter extreme frames in capped run
test('filters frames > cap*1.5 in capped 60 FPS run', () => {
  const fps60Capped = [
    ...Array(900).fill(60),
    ...Array(100).fill(150)  // Extreme outliers > 90 (60*1.5)
  ];
  
  const filtered = filterExtremeFrames(fps60Capped);
  
  // All frames > 90 should be filtered out
  assertTrue(filtered.every(fps => fps <= 90), 'Some frames above threshold were not filtered');
  assertTrue(filtered.length === 900, `Expected 900 frames, got ${filtered.length}`);
});

// Test 4: Filter extreme frames in uncapped run
test('filters frames > median*3 in uncapped run', () => {
  const fpsUncapped = [
    ...Array(100).fill(90),   // median = 95
    ...Array(100).fill(95),
    ...Array(100).fill(100),
    ...Array(10).fill(500)    // Extreme spikes > 285 (95*3)
  ];
  
  const filtered = filterExtremeFrames(fpsUncapped);
  
  // All frames > 285 should be filtered out
  const median = calculateMedian(fpsUncapped);
  const threshold = median * 3;
  assertTrue(filtered.every(fps => fps <= threshold), 'Some frames above threshold were not filtered');
  assertTrue(filtered.length === 300, `Expected 300 frames, got ${filtered.length}`);
});

// Test 5: 97% FPS should be lower after filtering (capped run)
test('97% FPS is lower/realistic after filtering in capped run', () => {
  const fps60Capped = [
    ...Array(900).fill(59.5),
    ...Array(100).fill(200)  // Extreme outliers
  ];
  
  // Without filtering (simulate old behavior)
  const frametimes = fps60Capped.map(fps => 1000 / fps);
  const sorted = [...frametimes].sort((a, b) => a - b);
  const index = Math.round(0.03 * (sorted.length + 1));
  const oldFPS97 = 1000 / sorted[index];
  
  // With filtering (new behavior)
  const newFPS97 = calculatePercentileFPS(fps60Capped, 97);
  
  // New 97% FPS should be lower and closer to 60
  assertTrue(newFPS97 < oldFPS97, `New FPS (${newFPS97}) should be < old FPS (${oldFPS97})`);
  assertTrue(newFPS97 <= 65, `97% FPS (${newFPS97}) should be <= 65 for 60 FPS cap`);
});

// Test 6: 1% low FPS should not be affected by filtering
test('1% low FPS is NOT affected by filtering (percentile < 90)', () => {
  const fps60Capped = [
    ...Array(10).fill(30),   // Low FPS
    ...Array(890).fill(60),
    ...Array(100).fill(200)  // Extreme outliers
  ];
  
  const fps1Low = calculatePercentileFPS(fps60Capped, 1);
  
  // 1% low should reflect the actual low frames (around 30)
  assertAlmostEquals(fps1Low, 30, 2, `1% low FPS should be ~30, got ${fps1Low}`);
});

// Test 7: Empty data handling
test('handles empty FPS data gracefully', () => {
  const result = calculatePercentileFPS([], 97);
  assertEquals(result, 0, `Expected 0 for empty data, got ${result}`);
});

// Test 8: Median calculation
test('calculates median correctly for odd-length array', () => {
  const data = [1, 2, 3, 4, 5];
  const median = calculateMedian(data);
  assertEquals(median, 3, `Expected median 3, got ${median}`);
});

test('calculates median correctly for even-length array', () => {
  const data = [1, 2, 3, 4];
  const median = calculateMedian(data);
  assertEquals(median, 2.5, `Expected median 2.5, got ${median}`);
});

// Test 9: Real-world scenario - 60 FPS cap with loading screens
test('real-world: 60 FPS cap with loading screen spikes', () => {
  // Simulate a real gaming session:
  // - Mostly at 60 FPS (gameplay)
  // - Some dips to 45-55 FPS
  // - Loading screens at 300+ FPS
  const realWorldFPS = [
    ...Array(50).fill(45),   // Some frame drops
    ...Array(850).fill(60),  // Stable gameplay at cap
    ...Array(50).fill(55),   // Minor dips
    ...Array(50).fill(300)   // Loading screens
  ];
  
  const fps97 = calculatePercentileFPS(realWorldFPS, 97);
  
  // 97% FPS should be close to 60, not inflated by loading screens
  assertTrue(fps97 > 55 && fps97 < 65, `Expected 97% FPS between 55-65, got ${fps97}`);
});

// Test 10: Real-world scenario - uncapped run with menu spikes
test('real-world: uncapped run with menu/cutscene spikes', () => {
  // Simulate uncapped run:
  // - Gameplay at 80-100 FPS
  // - Menus/cutscenes at 400+ FPS
  const realWorldFPS = [
    ...Array(300).fill(80),
    ...Array(400).fill(90),
    ...Array(250).fill(100),
    ...Array(50).fill(500)   // Menu/cutscene spikes
  ];
  
  const fps97 = calculatePercentileFPS(realWorldFPS, 97);
  
  // 97% FPS should reflect gameplay, not menu spikes
  assertTrue(fps97 > 80 && fps97 < 150, `Expected 97% FPS between 80-150, got ${fps97}`);
});

// Print results
console.log('\n' + '='.repeat(50));
console.log(`Tests passed: ${testsPassed}`);
console.log(`Tests failed: ${testsFailed}`);
console.log('='.repeat(50));

if (testsFailed > 0) {
  process.exit(1);
} else {
  console.log('\n✓ All tests passed!');
}
