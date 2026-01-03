#!/usr/bin/env node

/**
 * Unit tests for benchmarkDataProcessor.js
 * 
 * Tests that FPS statistics are correctly calculated from frametime data
 * rather than directly from FPS values.
 * 
 * Run with: node tests/benchmarkDataProcessor.test.js
 */

import { processRun } from '../src/utils/benchmarkDataProcessor.js';

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

function assertApprox(actual, expected, tolerance, message) {
  if (Math.abs(actual - expected) > tolerance) {
    throw new Error(message || `Expected ${expected} (±${tolerance}) but got ${actual}`);
  }
}

function assertGreaterThan(actual, expected, message) {
  if (actual <= expected) {
    throw new Error(message || `Expected ${actual} to be greater than ${expected}`);
  }
}

function assertLessThan(actual, expected, message) {
  if (actual >= expected) {
    throw new Error(message || `Expected ${actual} to be less than ${expected}`);
  }
}

console.log('Running benchmarkDataProcessor tests...\n');

// Test percentile calculation with linear interpolation
test('Percentile calculation uses linear interpolation', () => {
  // Create a simple dataset where we can verify interpolation
  // Using 10 data points: [10, 20, 30, 40, 50, 60, 70, 80, 90, 100]
  const values = [10, 20, 30, 40, 50, 60, 70, 80, 90, 100];
  const runData = {
    Label: 'Test Interpolation',
    DataFPS: [], // Not used
    DataFrameTime: [], // Not used
    DataCPULoad: values // Use CPU load to test percentile calculation
  };

  const processed = processRun(runData, 0);
  const stats = processed.stats.CPULoad;

  // For 10 values (indices 0-9):
  // 1st percentile: idx = 0.01 * 9 = 0.09
  // Should interpolate between index 0 (10) and index 1 (20)
  // fraction = 0.09, so: value[0] * (1 - 0.09) + value[1] * 0.09
  // Result: 10 * 0.91 + 20 * 0.09 = 9.1 + 1.8 = 10.9
  assertApprox(stats.p01, 10.9, 0.01, `1st percentile should be ~10.9, got ${stats.p01}`);

  // 99th percentile: idx = 0.99 * 9 = 8.91
  // Should interpolate between index 8 (90) and index 9 (100)
  // fraction = 0.91, so: value[8] * (1 - 0.91) + value[9] * 0.91
  // Result: 90 * 0.09 + 100 * 0.91 = 8.1 + 91 = 99.1
  assertApprox(stats.p99, 99.1, 0.01, `99th percentile should be ~99.1, got ${stats.p99}`);
});

// Test data: constant FPS of 60 (frametime should be 16.667ms)
test('FPS stats calculated from frametime - constant 60 FPS', () => {
  const runData = {
    Label: 'Test Run',
    DataFPS: Array(100).fill(60),
    DataFrameTime: Array(100).fill(16.667) // 1000/60 = 16.667ms
  };

  const processed = processRun(runData, 0);
  const fpsStats = processed.stats.FPS;

  // Average should be close to 60 FPS (1000 / 16.667)
  assertApprox(fpsStats.avg, 60, 0.1, `Average FPS should be ~60, got ${fpsStats.avg}`);
  
  // Min and max should also be close to 60
  assertApprox(fpsStats.min, 60, 0.1, `Min FPS should be ~60, got ${fpsStats.min}`);
  assertApprox(fpsStats.max, 60, 0.1, `Max FPS should be ~60, got ${fpsStats.max}`);
});

// Test data: varying frametimes
test('FPS stats calculated from frametime - varying frametimes', () => {
  // Create frametimes: 10ms, 20ms, 30ms (corresponds to 100, 50, 33.33 FPS)
  const frametimes = [10, 20, 30];
  const runData = {
    Label: 'Test Run Varying',
    DataFPS: frametimes.map(ft => 1000 / ft), // Calculate FPS from frametime for consistency
    DataFrameTime: frametimes
  };

  const processed = processRun(runData, 0);
  const fpsStats = processed.stats.FPS;

  // Average frametime = (10 + 20 + 30) / 3 = 20ms
  // Average FPS should be 1000 / 20 = 50 FPS
  assertApprox(fpsStats.avg, 50, 0.1, `Average FPS should be ~50, got ${fpsStats.avg}`);
  
  // Min frametime (10ms) = Max FPS (100)
  assertApprox(fpsStats.max, 100, 0.1, `Max FPS should be ~100, got ${fpsStats.max}`);
  
  // Max frametime (30ms) = Min FPS (33.33)
  assertApprox(fpsStats.min, 33.33, 0.1, `Min FPS should be ~33.33, got ${fpsStats.min}`);
});

// Test percentile calculation
test('FPS percentiles calculated correctly from frametime', () => {
  // Create 100 data points with varying frametimes
  // Sorted frametimes will range from 10ms to 100ms
  const frametimes = Array.from({ length: 100 }, (_, i) => 10 + i); // 10ms to 109ms
  const runData = {
    Label: 'Test Run Percentiles',
    DataFPS: frametimes.map(ft => 1000 / ft),
    DataFrameTime: frametimes
  };

  const processed = processRun(runData, 0);
  const fpsStats = processed.stats.FPS;

  // 1st percentile frametime (~11ms) should give ~90.9 FPS
  // 99th percentile frametime (~109ms) should give ~9.17 FPS
  // Note: These are approximate due to rounding in percentile calculation
  
  // p01 (1% low FPS) should be lower than average
  assertLessThan(fpsStats.p01, fpsStats.avg, 
    `p01 (${fpsStats.p01}) should be less than avg (${fpsStats.avg})`);
  
  // p99 (99th percentile FPS) should be higher than average
  assertGreaterThan(fpsStats.p99, fpsStats.avg, 
    `p99 (${fpsStats.p99}) should be greater than avg (${fpsStats.avg})`);
});

// Test that the fix is working: ensure we're using frametime, not FPS
test('FPS stats use frametime data, not FPS data', () => {
  // Create inconsistent data to verify which source is used
  // If FPS is used directly: avg would be 50
  // If frametime is used: avg should be 1000/20 = 50
  const runData = {
    Label: 'Test Run Verification',
    DataFPS: Array(10).fill(50), // If this is used, avg = 50
    DataFrameTime: Array(10).fill(20) // If this is used, avg = 1000/20 = 50
  };

  const processed = processRun(runData, 0);
  const fpsStats = processed.stats.FPS;

  // Both should give 50, but let's create a more distinct test
  // Use frametimes that would give different result if FPS was averaged
  const runData2 = {
    Label: 'Test Run Verification 2',
    DataFPS: [100, 50], // Direct average would be 75 FPS (WRONG)
    DataFrameTime: [10, 20] // Average frametime = 15ms, so 1000/15 = 66.67 FPS (CORRECT)
  };

  const processed2 = processRun(runData2, 0);
  const fpsStats2 = processed2.stats.FPS;

  // The correct average should be 66.67 (from frametime), not 75 (from FPS)
  assertApprox(fpsStats2.avg, 66.67, 0.1, 
    `Average FPS should be ~66.67 (from frametime), got ${fpsStats2.avg}`);
});

// Test fallback when no frametime data is available
test('FPS stats fallback when no frametime data', () => {
  const runData = {
    Label: 'Test Run No Frametime',
    DataFPS: Array(10).fill(60),
    DataFrameTime: [] // No frametime data
  };

  const processed = processRun(runData, 0);
  const fpsStats = processed.stats.FPS;

  // Should still have some stats (using FPS data directly as fallback)
  // This might not be accurate, but should not crash
  assertApprox(fpsStats.avg, 60, 0.1, `Should fallback to FPS data when frametime missing`);
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
