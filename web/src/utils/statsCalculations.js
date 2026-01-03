/**
 * Shared statistics calculation utilities
 * 
 * NOTE: If any of these functions are modified or replaced, you MUST also update:
 * - /debugcalc page (views/DebugCalc.vue) which uses these functions directly
 * - All tests that verify calculation accuracy
 */

/**
 * Calculate percentile with linear interpolation (matches scientific/numpy method)
 * This provides more accurate percentile values than simple floor-based indexing
 * 
 * NOTE: Used by /debugcalc page - update that page if this function changes
 * 
 * @param {Array<number>} sortedData - Pre-sorted array of numeric values
 * @param {number} percentile - Percentile to calculate (0-100)
 * @returns {number} The calculated percentile value
 */
export function calculatePercentileLinearInterpolation(sortedData, percentile) {
  if (!sortedData || sortedData.length === 0) {
    return 0
  }
  
  const n = sortedData.length
  // Convert percentile (0-100) to decimal and calculate fractional index
  const idx = (percentile / 100) * (n - 1)
  const lower = Math.floor(idx)
  const upper = Math.ceil(idx)
  
  // If index is exactly on a data point, return it
  if (lower === upper) {
    return sortedData[lower]
  }
  
  // Linear interpolation between adjacent data points
  const fraction = idx - lower
  return sortedData[lower] * (1 - fraction) + sortedData[upper] * fraction
}

/**
 * Calculate percentile using MangoHud's frametime-based threshold method (without interpolation)
 * This uses a simple floor-based approach to find the percentile value
 * 
 * NOTE: Used by /debugcalc page - update that page if this function changes
 * 
 * @param {Array<number>} sortedData - Pre-sorted array of numeric values
 * @param {number} percentile - Percentile to calculate (0-100)
 * @returns {number} The calculated percentile value
 */
export function calculatePercentileMangoHudThreshold(sortedData, percentile) {
  if (!sortedData || sortedData.length === 0) {
    return 0
  }
  
  const n = sortedData.length
  // Convert percentile (0-100) to decimal and calculate index
  // Use floor to get the index without interpolation
  const idx = Math.floor((percentile / 100) * n)
  
  // Clamp index to valid range
  const clampedIdx = Math.min(Math.max(idx, 0), n - 1)
  
  return sortedData[clampedIdx]
}

/**
 * Calculate density data for histogram/area charts
 * Filters outliers (1st-97th percentile) and counts occurrences
 * 
 * NOTE: Used internally by statistics calculations for chart rendering
 * 
 * @param {Array<number>} values - Array of numeric values
 * @param {string} calculationMethod - Either 'linear-interpolation' or 'mangohud-threshold'
 * @returns {Array<Array<number>>} Array of [value, count] pairs
 */
function calculateDensityData(values, calculationMethod) {
  if (!values || values.length === 0) return []
  
  const sorted = [...values].sort((a, b) => a - b)
  const calculatePercentile = calculationMethod === 'mangohud-threshold' 
    ? calculatePercentileMangoHudThreshold 
    : calculatePercentileLinearInterpolation
  const p01Value = calculatePercentile(sorted, 1)
  const p97Value = calculatePercentile(sorted, 97)
  const filtered = sorted.filter(v => v >= p01Value && v <= p97Value)
  
  const counts = {}
  filtered.forEach(value => {
    const rounded = Math.round(value)
    counts[rounded] = (counts[rounded] || 0) + 1
  })
  
  const array = Object.keys(counts).map(key => [parseInt(key), counts[key]]).sort((a, b) => a[0] - b[0])
  
  return array
}

/**
 * Calculate basic statistics for an array of values
 * 
 * NOTE: Used by /debugcalc page - update that page if this function changes
 * 
 * @param {Array<number>} values - Array of numeric values
 * @param {string} calculationMethod - Either 'linear-interpolation' or 'mangohud-threshold'
 * @returns {Object} Statistics object with min, max, avg, p01, p97, stddev, variance
 */
export function calculateStats(values, calculationMethod = 'linear-interpolation') {
  if (!values || values.length === 0) {
    return { min: 0, max: 0, avg: 0, p01: 0, p97: 0, stddev: 0, variance: 0, density: [] }
  }

  const sorted = [...values].sort((a, b) => a - b)
  const sum = values.reduce((acc, val) => acc + val, 0)
  const avg = sum / values.length
  
  // Calculate variance and standard deviation from FULL data
  // Use sample variance (n-1) to match Excel/LibreOffice VAR and STDEV functions
  const squaredDiffs = values.map(val => Math.pow(val - avg, 2))
  const variance = values.length > 1 
    ? squaredDiffs.reduce((acc, val) => acc + val, 0) / (values.length - 1)
    : 0
  const stddev = Math.sqrt(variance)
  
  // Select percentile calculation method
  const calculatePercentile = calculationMethod === 'mangohud-threshold' 
    ? calculatePercentileMangoHudThreshold 
    : calculatePercentileLinearInterpolation
  
  return {
    min: sorted[0],
    max: sorted[sorted.length - 1],
    avg: avg,
    p01: calculatePercentile(sorted, 1),
    p97: calculatePercentile(sorted, 97),
    stddev: stddev,
    variance: variance,
    density: calculateDensityData(values, calculationMethod)
  }
}

/**
 * Calculate FPS statistics from frametime data
 * This is the correct way to calculate FPS statistics, as averaging FPS values directly is incorrect
 * 
 * NOTE: Used by /debugcalc page - update that page if this function changes
 * 
 * @param {Array<number>} frametimeValues - Array of frametime values in milliseconds
 * @param {string} calculationMethod - Either 'linear-interpolation' or 'mangohud-threshold'
 * @returns {Object} Statistics object with min, max, avg, p01, p97, stddev, variance
 */
export function calculateFPSStatsFromFrametime(frametimeValues, calculationMethod = 'linear-interpolation') {
  if (!frametimeValues || frametimeValues.length === 0) {
    return { min: 0, max: 0, avg: 0, p01: 0, p97: 0, stddev: 0, variance: 0, density: [] }
  }

  // Sort frametime values
  const sorted = [...frametimeValues].sort((a, b) => a - b)
  
  // Select percentile calculation method
  const calculatePercentile = calculationMethod === 'mangohud-threshold' 
    ? calculatePercentileMangoHudThreshold 
    : calculatePercentileLinearInterpolation
  
  // Calculate FPS percentiles from frametime percentiles (inverted relationship)
  // Low frametime = high FPS, so percentiles are inverted
  // 3rd percentile frametime (faster) = 97th percentile FPS (p97)
  // 99th percentile frametime (slowest) = 1st percentile FPS (p01)
  const frametimeP03 = calculatePercentile(sorted, 3)
  const frametimeP99 = calculatePercentile(sorted, 99)
  
  // Convert frametime percentiles to FPS
  const fpsP97 = frametimeP03 > 0 ? 1000 / frametimeP03 : 0  // 3rd percentile frametime -> 97th percentile FPS
  const fpsP01 = frametimeP99 > 0 ? 1000 / frametimeP99 : 0  // 99th percentile frametime -> 1st percentile FPS
  
  // Calculate average FPS from average frametime
  const avgFrametime = frametimeValues.reduce((acc, val) => acc + val, 0) / frametimeValues.length
  const avgFPS = avgFrametime > 0 ? 1000 / avgFrametime : 0
  
  // Convert all frametime values to FPS for min/max and density calculation
  const fpsValues = frametimeValues.map(ft => ft > 0 ? 1000 / ft : 0)
  
  // Calculate min/max FPS (note: min frametime = max FPS, max frametime = min FPS)
  const minFrametime = sorted[0]
  const maxFrametime = sorted[sorted.length - 1]
  const maxFPS = minFrametime > 0 ? 1000 / minFrametime : 0
  const minFPS = maxFrametime > 0 ? 1000 / maxFrametime : 0
  
  // Calculate standard deviation and variance from FPS values
  // Use sample variance (n-1) to match Excel/LibreOffice VAR and STDEV functions
  const fpsSum = fpsValues.reduce((acc, val) => acc + val, 0)
  const fpsMean = fpsSum / fpsValues.length
  const squaredDiffs = fpsValues.map(val => Math.pow(val - fpsMean, 2))
  const variance = fpsValues.length > 1
    ? squaredDiffs.reduce((acc, val) => acc + val, 0) / (fpsValues.length - 1)
    : 0
  const stddev = Math.sqrt(variance)
  
  return {
    min: minFPS,
    max: maxFPS,
    avg: avgFPS,
    p01: fpsP01,
    p97: fpsP97,
    stddev: stddev,
    variance: variance,
    density: calculateDensityData(fpsValues, calculationMethod)
  }
}
