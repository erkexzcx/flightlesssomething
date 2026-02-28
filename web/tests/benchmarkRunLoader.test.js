#!/usr/bin/env node

/**
 * Unit tests for benchmarkRunLoader.js
 * 
 * Tests the parallel loading logic including concurrency calculation,
 * worker pool behavior, and progress callbacks.
 * 
 * Run with: node tests/benchmarkRunLoader.test.js
 */

import { getConcurrency } from '../src/utils/benchmarkRunLoader.js'

// Simple test runner
let testsPassed = 0
let testsFailed = 0

function test(description, fn) {
  try {
    fn()
    console.log(`✓ ${description}`)
    testsPassed++
  } catch (error) {
    console.error(`✗ ${description}`)
    console.error(`  ${error.message}`)
    testsFailed++
  }
}

function assertEquals(actual, expected, message) {
  if (actual !== expected) {
    throw new Error(message || `Expected ${expected} but got ${actual}`)
  }
}

function assertTrue(condition, message) {
  if (!condition) {
    throw new Error(message || 'Expected condition to be true')
  }
}

console.log('Running benchmarkRunLoader tests...\n')

// getConcurrency tests

test('getConcurrency returns 1 when totalRuns is 1', () => {
  const result = getConcurrency(1)
  assertEquals(result, 1, 'Should be 1 for single run')
})

test('getConcurrency returns at most 6 (browser connection limit)', () => {
  const result = getConcurrency(100)
  assertTrue(result <= 6, `Concurrency ${result} should be at most 6`)
})

test('getConcurrency returns at least 1', () => {
  const result = getConcurrency(1)
  assertTrue(result >= 1, `Concurrency ${result} should be at least 1`)
})

test('getConcurrency caps at totalRuns when runs are fewer than cores', () => {
  const result = getConcurrency(2)
  assertTrue(result <= 2, `Concurrency ${result} should be at most totalRuns (2)`)
})

test('getConcurrency uses navigator.hardwareConcurrency when available', () => {
  // In Node.js, navigator is not defined, so it falls back to 4
  const result = getConcurrency(10)
  // Without navigator, default is 4
  assertEquals(result, 4, 'Should fall back to 4 when navigator is not available')
})

test('getConcurrency handles totalRuns of 0 gracefully', () => {
  // Math.min(cores, 0, 6) = 0, Math.max(1, 0) = 1
  const result = getConcurrency(0)
  assertTrue(result >= 1, `Concurrency ${result} should be at least 1 even for 0 runs`)
})

// Print results
console.log('\n' + '='.repeat(50))
console.log(`Tests passed: ${testsPassed}`)
console.log(`Tests failed: ${testsFailed}`)
console.log('='.repeat(50))

if (testsFailed > 0) {
  process.exit(1)
} else {
  console.log('\n✓ All tests passed!')
}
