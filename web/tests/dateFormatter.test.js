#!/usr/bin/env node

/**
 * Unit tests for dateFormatter.js
 * 
 * Run with: node tests/dateFormatter.test.js
 */

import { formatRelativeDate } from '../src/utils/dateFormatter.js';

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

function assertMatch(actual, pattern, message) {
  if (!pattern.test(actual)) {
    throw new Error(message || `Expected "${actual}" to match ${pattern}`);
  }
}

function assertNotMatch(actual, pattern, message) {
  if (pattern.test(actual)) {
    throw new Error(message || `Expected "${actual}" to NOT match ${pattern}`);
  }
}

console.log('Running dateFormatter tests...\n');

// Test edge cases
test('handles null input', () => {
  assertEquals(formatRelativeDate(null), 'Unknown');
});

test('handles undefined input', () => {
  assertEquals(formatRelativeDate(undefined), 'Unknown');
});

test('handles empty string', () => {
  assertEquals(formatRelativeDate(''), 'Unknown');
});

test('handles custom fallback text', () => {
  assertEquals(formatRelativeDate(null, 'Never'), 'Never');
  assertEquals(formatRelativeDate('', 'N/A'), 'N/A');
});

test('handles invalid date string', () => {
  assertEquals(formatRelativeDate('invalid-date'), 'Unknown');
  assertEquals(formatRelativeDate('not a date', 'Error'), 'Error');
});

// Helper functions for date generation
function daysAgo(days) {
  const d = new Date();
  d.setDate(d.getDate() - days);
  return d.toISOString();
}

function monthsAgo(months) {
  const d = new Date();
  d.setMonth(d.getMonth() - months);
  return d.toISOString();
}

function yearsAgo(years) {
  const d = new Date();
  d.setFullYear(d.getFullYear() - years);
  return d.toISOString();
}

// Test various time periods
test('1 day ago', () => {
  const result = formatRelativeDate(daysAgo(1));
  assertMatch(result, /a day ago/i, `Expected "a day ago" but got "${result}"`);
});

test('2 days ago', () => {
  const result = formatRelativeDate(daysAgo(2));
  assertMatch(result, /2 days ago/i, `Expected "2 days ago" but got "${result}"`);
});

test('7 days ago (1 week)', () => {
  const result = formatRelativeDate(daysAgo(7));
  assertMatch(result, /7 days ago/i, `Expected "7 days ago" but got "${result}"`);
});

test('14 days ago (2 weeks)', () => {
  const result = formatRelativeDate(daysAgo(14));
  assertMatch(result, /14 days ago/i, `Expected "14 days ago" but got "${result}"`);
});

// Critical bug tests - these were showing "0 months ago" before
test('28 days ago should NOT show "0 months ago"', () => {
  const result = formatRelativeDate(daysAgo(28));
  assertNotMatch(result, /0 months? ago/i, `BUG: Got "${result}" which contains "0 months ago"`);
  assertMatch(result, /(a month|28 days) ago/i, `Expected "a month ago" or "28 days ago" but got "${result}"`);
});

test('29 days ago should NOT show "0 months ago"', () => {
  const result = formatRelativeDate(daysAgo(29));
  assertNotMatch(result, /0 months? ago/i, `BUG: Got "${result}" which contains "0 months ago"`);
  assertMatch(result, /(a month|29 days) ago/i, `Expected "a month ago" or "29 days ago" but got "${result}"`);
});

test('30 days ago should NOT show "0 months ago"', () => {
  const result = formatRelativeDate(daysAgo(30));
  assertNotMatch(result, /0 months? ago/i, `BUG: Got "${result}" which contains "0 months ago"`);
  assertMatch(result, /a month ago/i, `Expected "a month ago" but got "${result}"`);
});

test('31 days ago should NOT show "0 months ago"', () => {
  const result = formatRelativeDate(daysAgo(31));
  assertNotMatch(result, /0 months? ago/i, `BUG: Got "${result}" which contains "0 months ago"`);
  assertMatch(result, /a month ago/i, `Expected "a month ago" but got "${result}"`);
});

test('60 days ago (about 2 months)', () => {
  const result = formatRelativeDate(daysAgo(60));
  assertNotMatch(result, /0 months? ago/i, `BUG: Got "${result}" which contains "0 months ago"`);
  assertMatch(result, /2 months ago/i, `Expected "2 months ago" but got "${result}"`);
});

test('3 months ago', () => {
  const result = formatRelativeDate(monthsAgo(3));
  assertMatch(result, /3 months ago/i, `Expected "3 months ago" but got "${result}"`);
});

test('6 months ago', () => {
  const result = formatRelativeDate(monthsAgo(6));
  assertMatch(result, /6 months ago/i, `Expected "6 months ago" but got "${result}"`);
});

test('11 months ago', () => {
  const result = formatRelativeDate(monthsAgo(11));
  // dayjs may round to "a year ago" for 11 months, which is acceptable
  assertMatch(result, /(11 months|a year) ago/i, `Expected "11 months ago" or "a year ago" but got "${result}"`);
});

test('1 year ago', () => {
  const result = formatRelativeDate(yearsAgo(1));
  assertMatch(result, /a year ago/i, `Expected "a year ago" but got "${result}"`);
});

test('2 years ago', () => {
  const result = formatRelativeDate(yearsAgo(2));
  assertMatch(result, /2 years ago/i, `Expected "2 years ago" but got "${result}"`);
});

// Test with different date formats
test('works with ISO 8601 format', () => {
  const isoDate = new Date(Date.now() - 5 * 24 * 60 * 60 * 1000).toISOString();
  const result = formatRelativeDate(isoDate);
  assertMatch(result, /5 days ago/i, `Expected "5 days ago" but got "${result}"`);
});

test('works with UTC string format', () => {
  const utcDate = new Date(Date.now() - 10 * 24 * 60 * 60 * 1000).toUTCString();
  const result = formatRelativeDate(utcDate);
  assertMatch(result, /10 days ago/i, `Expected "10 days ago" but got "${result}"`);
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
