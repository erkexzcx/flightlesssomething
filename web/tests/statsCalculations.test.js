#!/usr/bin/env node

import assert from 'node:assert/strict'
import { calculateStats, calculateFPSStatsFromFrametime } from '../src/utils/statsCalculations.js'

const values = [0, 99, ...Array.from({ length: 50 }, () => [3.14, 3.24, 3.34]).flat()]
for (const method of ['linear-interpolation', 'mangohud-threshold']) {
  const frametime = calculateStats(values, method, 10)
  assert.deepEqual(frametime.density, [[3.1, 50], [3.2, 50], [3.3, 50]], `${method}: decimal frame buckets and outlier filtering`)
  assert.deepEqual(calculateStats(values, method).density, [[3, 150]], `${method}: other metrics retain whole-unit buckets`)
  assert.deepEqual(calculateStats([], method, 10).density, [], `${method}: empty input`)
  assert.deepEqual(calculateStats([3.14], method, 10).density, [[3.1, 1]], `${method}: single sample`)
  assert.deepEqual(calculateStats([3.14, 3.14, 3.14], method, 10).density, [[3.1, 3]], `${method}: identical samples`)
  const fps = calculateFPSStatsFromFrametime(values.filter(value => value > 0), method)
  assert.ok(fps.density.every(([value]) => Number.isInteger(value)), `${method}: FPS retains whole-unit buckets`)
}
console.log('Statistics density regressions passed for both percentile methods.')
