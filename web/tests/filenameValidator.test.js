/**
 * Unit tests for filenameValidator.js
 * 
 * Run with: node tests/filenameValidator.test.js
 */

import { hasDateTimePattern, hasAnyDateTimePattern, getDateTimeWarningMessage } from '../src/utils/filenameValidator.js';

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

function assertTrue(actual, message) {
  if (!actual) {
    throw new Error(message || `Expected true but got ${actual}`);
  }
}

function assertFalse(actual, message) {
  if (actual) {
    throw new Error(message || `Expected false but got ${actual}`);
  }
}

console.log('Running filenameValidator tests...\n');

// Test hasDateTimePattern - positive cases
test('detects date/time with hyphens (cs2_2025-12-21_12-40-08)', () => {
  assertTrue(hasDateTimePattern('cs2_2025-12-21_12-40-08'), 'Should detect date/time pattern with hyphens');
});

test('detects date/time with underscores (cs2_2025_12_21_12_40_08)', () => {
  assertTrue(hasDateTimePattern('cs2_2025_12_21_12_40_08'), 'Should detect date/time pattern with underscores');
});

test('detects date/time with mixed separators (test-2025-12-21_12-40-08)', () => {
  assertTrue(hasDateTimePattern('test-2025-12-21_12-40-08'), 'Should detect date/time pattern with mixed separators');
});

test('detects date/time at start of filename (2025-12-21_12-40-08-benchmark)', () => {
  assertTrue(hasDateTimePattern('2025-12-21_12-40-08-benchmark'), 'Should detect date/time pattern at start');
});

test('detects date/time in middle of filename (test_2025-12-21_12-40-08_async)', () => {
  assertTrue(hasDateTimePattern('test_2025-12-21_12-40-08_async'), 'Should detect date/time pattern in middle');
});

test('detects date/time with suffix (cs2_2025-12-21_12-40-08 - Async Off)', () => {
  assertTrue(hasDateTimePattern('cs2_2025-12-21_12-40-08 - Async Off'), 'Should detect date/time pattern with suffix');
});

test('detects date/time in complex filename (benchmark_cs2_2024_01_15_14_30_45_test)', () => {
  assertTrue(hasDateTimePattern('benchmark_cs2_2024_01_15_14_30_45_test'), 'Should detect date/time in complex filename');
});

// Test hasDateTimePattern - negative cases
test('does not detect regular filename (cs2 async on)', () => {
  assertFalse(hasDateTimePattern('cs2 async on'), 'Should not detect in regular filename');
});

test('does not detect partial date (2025-12-21)', () => {
  assertFalse(hasDateTimePattern('2025-12-21'), 'Should not detect partial date without time');
});

test('does not detect partial time (12-40-08)', () => {
  assertFalse(hasDateTimePattern('12-40-08'), 'Should not detect partial time without date');
});

test('does not detect date with wrong format (12-21-2025_12-40-08)', () => {
  assertFalse(hasDateTimePattern('12-21-2025_12-40-08'), 'Should not detect date in MM-DD-YYYY format');
});

test('does not detect random numbers (test_1234_56_78_90_12_34)', () => {
  assertFalse(hasDateTimePattern('test_1234_56_78_90_12_34'), 'Should not detect random numbers');
});

test('does not detect short year (test_25-12-21_12-40-08)', () => {
  assertFalse(hasDateTimePattern('test_25-12-21_12-40-08'), 'Should not detect short year format');
});

test('does not detect filename with spaces (Async On)', () => {
  assertFalse(hasDateTimePattern('Async On'), 'Should not detect in simple filename with spaces');
});

test('does not detect filename with version numbers (v1.2.3)', () => {
  assertFalse(hasDateTimePattern('v1.2.3'), 'Should not detect version numbers');
});

// Test edge cases
test('handles null input', () => {
  assertFalse(hasDateTimePattern(null), 'Should return false for null');
});

test('handles undefined input', () => {
  assertFalse(hasDateTimePattern(undefined), 'Should return false for undefined');
});

test('handles empty string', () => {
  assertFalse(hasDateTimePattern(''), 'Should return false for empty string');
});

test('handles non-string input (number)', () => {
  assertFalse(hasDateTimePattern(123), 'Should return false for number');
});

test('handles non-string input (object)', () => {
  assertFalse(hasDateTimePattern({}), 'Should return false for object');
});

// Test hasAnyDateTimePattern
test('detects date/time in array with one bad filename', () => {
  const filenames = ['cs2 async on', 'cs2_2025-12-21_12-40-08 - Async Off', 'benchmark'];
  assertTrue(hasAnyDateTimePattern(filenames), 'Should detect date/time in array');
});

test('detects date/time when all filenames are bad', () => {
  const filenames = ['cs2_2025-12-21_12-40-08', 'test_2024_01_15_14_30_45'];
  assertTrue(hasAnyDateTimePattern(filenames), 'Should detect date/time in all bad filenames');
});

test('does not detect date/time in array with good filenames', () => {
  const filenames = ['cs2 async on', 'cs2 async off', 'benchmark'];
  assertFalse(hasAnyDateTimePattern(filenames), 'Should not detect in good filenames');
});

test('handles empty array', () => {
  assertFalse(hasAnyDateTimePattern([]), 'Should return false for empty array');
});

test('handles null array', () => {
  assertFalse(hasAnyDateTimePattern(null), 'Should return false for null');
});

test('handles non-array input', () => {
  assertFalse(hasAnyDateTimePattern('not an array'), 'Should return false for non-array');
});

// Test getDateTimeWarningMessage
test('returns a non-empty warning message', () => {
  const message = getDateTimeWarningMessage();
  assertTrue(message.length > 0, 'Warning message should not be empty');
  assertTrue(message.includes('cs2_2025-12-21_12-40-08'), 'Message should contain example');
  assertTrue(message.includes('Async Off'), 'Message should contain suggested replacement');
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
