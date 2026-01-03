#!/usr/bin/env node

/**
 * Unit tests for benchmarkDataProcessor.js
 * 
 * Tests that FPS statistics are correctly calculated from frametime data
 * using both linear interpolation and MangoHud threshold methods.
 * Now supports async processRun with Web Workers.
 * 
 * Run with: node tests/benchmarkDataProcessor.test.js
 */

import { processRun } from '../src/utils/benchmarkDataProcessor.js';

// Simple test runner
let testsPassed = 0;
let testsFailed = 0;

async function test(description, fn) {
  try {
    await fn();
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

// Run all tests
(async () => {
  await test('Linear interpolation: Percentile calculation uses linear interpolation', async () => {
    const values = [10, 20, 30, 40, 50, 60, 70, 80, 90, 100];
    const runData = {
      Label: 'Test Interpolation',
      DataFPS: [],
      DataFrameTime: [],
      DataCPULoad: values
    };

    const processed = await processRun(runData, 0);
    const stats = processed.stats.CPULoad;

    assertApprox(stats.p01, 10.9, 0.01, `1st percentile should be ~10.9, got ${stats.p01}`);
    assertApprox(stats.p97, 97.3, 0.01, `97th percentile should be ~97.3, got ${stats.p97}`);
  });

  await test('MangoHud threshold: Percentile calculation uses floor-based approach', async () => {
    const values = [10, 20, 30, 40, 50, 60, 70, 80, 90, 100];
    const runData = {
      Label: 'Test MangoHud',
      DataFPS: [],
      DataFrameTime: [],
      DataCPULoad: values
    };

    const processed = await processRun(runData, 0);
    const statsMangoHud = processed.statsMangoHud.CPULoad;

    assertApprox(statsMangoHud.p01, 10, 0.01, `MangoHud 1st percentile should be 10, got ${statsMangoHud.p01}`);
    assertApprox(statsMangoHud.p97, 100, 0.01, `MangoHud 97th percentile should be 100, got ${statsMangoHud.p97}`);
  });

  await test('FPS stats calculated from frametime - constant 60 FPS', async () => {
    const runData = {
      Label: 'Test Run',
      DataFPS: Array(100).fill(60),
      DataFrameTime: Array(100).fill(16.667)
    };

    const processed = await processRun(runData, 0);
    const fpsStats = processed.stats.FPS;

    assertApprox(fpsStats.avg, 60, 0.1, `Average FPS should be ~60, got ${fpsStats.avg}`);
    assertApprox(fpsStats.min, 60, 0.1, `Min FPS should be ~60, got ${fpsStats.min}`);
    assertApprox(fpsStats.max, 60, 0.1, `Max FPS should be ~60, got ${fpsStats.max}`);
  });

  await test('FPS stats calculated from frametime - varying frametimes', async () => {
    const frametimes = [10, 20, 30];
    const runData = {
      Label: 'Test Run Varying',
      DataFPS: frametimes.map(ft => 1000 / ft),
      DataFrameTime: frametimes
    };

    const processed = await processRun(runData, 0);
    const fpsStats = processed.stats.FPS;

    assertApprox(fpsStats.avg, 50, 0.1, `Average FPS should be ~50, got ${fpsStats.avg}`);
    assertApprox(fpsStats.max, 100, 0.1, `Max FPS should be ~100, got ${fpsStats.max}`);
    assertApprox(fpsStats.min, 33.33, 0.1, `Min FPS should be ~33.33, got ${fpsStats.min}`);
  });

  await test('FPS percentiles calculated correctly from frametime', async () => {
    const frametimes = Array.from({ length: 100 }, (_, i) => 10 + i);
    const runData = {
      Label: 'Test Run Percentiles',
      DataFPS: frametimes.map(ft => 1000 / ft),
      DataFrameTime: frametimes
    };

    const processed = await processRun(runData, 0);
    const fpsStats = processed.stats.FPS;

    assertLessThan(fpsStats.p01, fpsStats.avg, 
      `p01 (${fpsStats.p01}) should be less than avg (${fpsStats.avg})`);
    assertGreaterThan(fpsStats.p97, fpsStats.avg, 
      `p97 (${fpsStats.p97}) should be greater than avg (${fpsStats.avg})`);
  });

  await test('FPS stats use frametime data, not FPS data', async () => {
    const runData2 = {
      Label: 'Test Run Verification 2',
      DataFPS: [100, 50],
      DataFrameTime: [10, 20]
    };

    const processed2 = await processRun(runData2, 0);
    const fpsStats2 = processed2.stats.FPS;

    assertApprox(fpsStats2.avg, 66.67, 0.1, 
      `Average FPS should be ~66.67 (from frametime), got ${fpsStats2.avg}`);
  });

  await test('FPS stats fallback when no frametime data', async () => {
    const runData = {
      Label: 'Test Run No Frametime',
      DataFPS: Array(10).fill(60),
      DataFrameTime: []
    };

    const processed = await processRun(runData, 0);
    const fpsStats = processed.stats.FPS;

    assertApprox(fpsStats.avg, 60, 0.1, `Should fallback to FPS data when frametime missing`);
  });

  await test('Both calculation methods are present in processed data', async () => {
    const runData = {
      Label: 'Test Both Methods',
      DataFPS: Array(10).fill(60),
      DataFrameTime: Array(10).fill(16.667)
    };

    const processed = await processRun(runData, 0);
    
    if (!processed.stats) {
      throw new Error('stats object is missing');
    }
    if (!processed.statsMangoHud) {
      throw new Error('statsMangoHud object is missing');
    }
    if (!processed.stats.FPS) {
      throw new Error('stats.FPS is missing');
    }
    if (!processed.statsMangoHud.FPS) {
      throw new Error('statsMangoHud.FPS is missing');
    }
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
})();
