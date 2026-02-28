#!/usr/bin/env node

/**
 * Unit tests for benchmarkDataProcessor.js
 * 
 * Tests that processRun correctly maps pre-calculated backend data
 * to the format expected by the frontend charts.
 * Since the backend now pre-calculates all stats, processRun is a simple mapping.
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

function assertEquals(actual, expected, message) {
  if (actual !== expected) {
    throw new Error(message || `Expected ${expected} but got ${actual}`);
  }
}

console.log('Running benchmarkDataProcessor tests...\n');

test('processRun maps label from backend data', () => {
  const runData = {
    label: 'Test Run 1',
    specOS: 'Linux',
    specCPU: 'AMD Ryzen 7',
    specGPU: 'RTX 3080',
    specRAM: '32GB',
    totalDataPoints: 100,
    series: {},
    stats: {},
    statsMangoHud: {}
  };

  const processed = processRun(runData, 0);
  assertEquals(processed.label, 'Test Run 1', 'Label should be mapped');
  assertEquals(processed.runIndex, 0, 'Run index should be set');
});

test('processRun maps spec fields correctly', () => {
  const runData = {
    label: 'Test',
    specOS: 'Steam Runtime 3',
    specCPU: 'AMD Ryzen 7 9800X3D',
    specGPU: 'AMD RX 9070 XT',
    specRAM: '32 GB',
    specLinuxKernel: '6.17.8-cachyos',
    specLinuxScheduler: 'performance',
    totalDataPoints: 50,
    series: {},
    stats: {},
    statsMangoHud: {}
  };

  const processed = processRun(runData, 1);
  assertEquals(processed.specOS, 'Steam Runtime 3', 'specOS should be mapped');
  assertEquals(processed.specCPU, 'AMD Ryzen 7 9800X3D', 'specCPU should be mapped');
  assertEquals(processed.specGPU, 'AMD RX 9070 XT', 'specGPU should be mapped');
  assertEquals(processed.specRAM, '32 GB', 'specRAM should be mapped');
  assertEquals(processed.specLinuxKernel, '6.17.8-cachyos', 'specLinuxKernel should be mapped');
  assertEquals(processed.specLinuxScheduler, 'performance', 'specLinuxScheduler should be mapped');
  assertEquals(processed.totalDataPoints, 50, 'totalDataPoints should be mapped');
});

test('processRun maps series data through', () => {
  const fpsPoints = [[0, 100], [10, 200], [20, 150]];
  const runData = {
    label: 'Test',
    specOS: '',
    specCPU: '',
    specGPU: '',
    specRAM: '',
    totalDataPoints: 100,
    series: { FPS: fpsPoints, FrameTime: [[0, 10], [10, 5]] },
    stats: {},
    statsMangoHud: {}
  };

  const processed = processRun(runData, 0);
  assertEquals(processed.series.FPS.length, 3, 'FPS series should have 3 points');
  assertEquals(processed.series.FPS[0][1], 100, 'First FPS value should be 100');
  assertEquals(processed.series.FrameTime.length, 2, 'FrameTime series should have 2 points');
});

test('processRun maps stats and statsMangoHud through', () => {
  const runData = {
    label: 'Test',
    specOS: '',
    specCPU: '',
    specGPU: '',
    specRAM: '',
    totalDataPoints: 100,
    series: {},
    stats: {
      FPS: { min: 50, max: 200, avg: 120, median: 115, p01: 55, p97: 190, stddev: 30, variance: 900, count: 100, density: [[50, 1], [100, 5]] }
    },
    statsMangoHud: {
      FPS: { min: 50, max: 200, avg: 120, median: 110, p01: 60, p97: 185, stddev: 30, variance: 900, count: 100, density: [[55, 2], [100, 4]] }
    }
  };

  const processed = processRun(runData, 0);

  assertEquals(processed.stats.FPS.min, 50, 'Stats FPS min should be 50');
  assertEquals(processed.stats.FPS.avg, 120, 'Stats FPS avg should be 120');
  assertEquals(processed.stats.FPS.p01, 55, 'Stats FPS p01 should be 55');
  assertEquals(processed.stats.FPS.density.length, 2, 'Stats FPS density should have 2 entries');

  assertEquals(processed.statsMangoHud.FPS.p01, 60, 'StatsMangoHud FPS p01 should be 60');
  assertEquals(processed.statsMangoHud.FPS.p97, 185, 'StatsMangoHud FPS p97 should be 185');
});

test('processRun provides defaults for missing fields', () => {
  const runData = {};

  const processed = processRun(runData, 5);
  assertEquals(processed.label, 'Run 6', 'Default label should be Run N+1');
  assertEquals(processed.runIndex, 5, 'Run index should be 5');
  assertEquals(processed.specOS, '', 'Missing specOS should default to empty string');
  assertEquals(processed.totalDataPoints, 0, 'Missing totalDataPoints should default to 0');
  assertEquals(typeof processed.series, 'object', 'Series should be an object');
  assertEquals(typeof processed.stats, 'object', 'Stats should be an object');
  assertEquals(typeof processed.statsMangoHud, 'object', 'StatsMangoHud should be an object');
});

test('processRun is synchronous (no async needed)', () => {
  const runData = {
    label: 'Sync Test',
    specOS: 'Linux',
    specCPU: 'CPU',
    specGPU: 'GPU',
    specRAM: '16GB',
    totalDataPoints: 10,
    series: { FPS: [[0, 60]] },
    stats: { FPS: { min: 60, max: 60, avg: 60, p01: 60, p97: 60, stddev: 0, variance: 0, count: 10, density: [] } },
    statsMangoHud: { FPS: { min: 60, max: 60, avg: 60, p01: 60, p97: 60, stddev: 0, variance: 0, count: 10, density: [] } }
  };

  // processRun should return a plain object, not a Promise
  const result = processRun(runData, 0);
  if (result instanceof Promise) {
    throw new Error('processRun should be synchronous, not return a Promise');
  }
  assertEquals(result.label, 'Sync Test', 'Should return data synchronously');
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
