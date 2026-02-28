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
 * Calculate percentile using MangoHud's exact formula
 * MangoHud formula: idx = floor(val * n - 1) on descending sorted data
 * To match MangoHud when called with percentile p on ascending data:
 *   idx = n - 1 - floor((1 - p/100) * n - 1)
 * 
 * NOTE: This uses inverted semantics compared to standard percentiles
 * When FS calls this with p=99 for frametimes, it matches MangoHud's val=0.01
 * 
 * @param {Array<number>} sortedData - Pre-sorted array of numeric values (ascending order)
 * @param {number} percentile - Percentile to calculate (0-100)
 * @returns {number} The calculated percentile value
 */
export function calculatePercentileMangoHudThreshold(sortedData, percentile) {
  if (!sortedData || sortedData.length === 0) {
    return 0
  }
  
  const n = sortedData.length
  
  // MangoHud uses val=(100-p)/100 on descending data
  // idx_desc = floor((100-p)/100 * n - 1)
  // Convert to ascending: idx_asc = n - 1 - idx_desc
  const valMango = (100 - percentile) / 100
  const idxDesc = Math.floor(valMango * n - 1)
  const idx = n - 1 - idxDesc
  
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
 * @returns {Object} Statistics object with all percentiles matching backend MetricStats
 */
export function calculateStats(values, calculationMethod = 'linear-interpolation') {
  if (!values || values.length === 0) {
    return { min: 0, max: 0, avg: 0, median: 0, p01: 0, p05: 0, p10: 0, p25: 0, p75: 0, p90: 0, p95: 0, p97: 0, p99: 0, iqr: 0, stddev: 0, variance: 0, count: 0, density: [] }
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
  
  const p25 = calculatePercentile(sorted, 25)
  const p75 = calculatePercentile(sorted, 75)
  
  return {
    min: sorted[0],
    max: sorted[sorted.length - 1],
    avg: avg,
    median: calculatePercentile(sorted, 50),
    p01: calculatePercentile(sorted, 1),
    p05: calculatePercentile(sorted, 5),
    p10: calculatePercentile(sorted, 10),
    p25: p25,
    p75: p75,
    p90: calculatePercentile(sorted, 90),
    p95: calculatePercentile(sorted, 95),
    p97: calculatePercentile(sorted, 97),
    p99: calculatePercentile(sorted, 99),
    iqr: p75 - p25,
    stddev: stddev,
    variance: variance,
    count: values.length,
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
 * @returns {Object} Statistics object with all percentiles matching backend MetricStats
 */
export function calculateFPSStatsFromFrametime(frametimeValues, calculationMethod = 'linear-interpolation') {
  if (!frametimeValues || frametimeValues.length === 0) {
    return { min: 0, max: 0, avg: 0, median: 0, p01: 0, p05: 0, p10: 0, p25: 0, p75: 0, p90: 0, p95: 0, p97: 0, p99: 0, iqr: 0, stddev: 0, variance: 0, count: 0, density: [] }
  }

  // Sort frametime values
  const sorted = [...frametimeValues].sort((a, b) => a - b)
  
  // Select percentile calculation method
  const calculatePercentile = calculationMethod === 'mangohud-threshold' 
    ? calculatePercentileMangoHudThreshold 
    : calculatePercentileLinearInterpolation
  
  // Calculate FPS percentiles from frametime percentiles (inverted relationship)
  // Low frametime = high FPS, so percentiles are inverted
  // FPS Px = 1000 / FT P(100-x)
  const ftP01 = calculatePercentile(sorted, 1)
  const ftP03 = calculatePercentile(sorted, 3)
  const ftP05 = calculatePercentile(sorted, 5)
  const ftP10 = calculatePercentile(sorted, 10)
  const ftP25 = calculatePercentile(sorted, 25)
  const ftP75 = calculatePercentile(sorted, 75)
  const ftP90 = calculatePercentile(sorted, 90)
  const ftP95 = calculatePercentile(sorted, 95)
  const ftP99 = calculatePercentile(sorted, 99)
  
  const safeDiv = (ft) => ft > 0 ? 1000 / ft : 0
  
  const fpsP01 = safeDiv(ftP99)
  const fpsP05 = safeDiv(ftP95)
  const fpsP10 = safeDiv(ftP90)
  const fpsP25 = safeDiv(ftP75)
  const fpsP75 = safeDiv(ftP25)
  const fpsP90 = safeDiv(ftP10)
  const fpsP95 = safeDiv(ftP05)
  const fpsP97 = safeDiv(ftP03)
  const fpsP99 = safeDiv(ftP01)
  const fpsIQR = fpsP75 - fpsP25
  
  // Calculate average FPS from average frametime
  const avgFrametime = frametimeValues.reduce((acc, val) => acc + val, 0) / frametimeValues.length
  const avgFPS = avgFrametime > 0 ? 1000 / avgFrametime : 0
  
  // Convert all frametime values to FPS for min/max, stddev, median, and density calculation
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
  
  // Median from sorted FPS values
  const sortedFPS = [...fpsValues].sort((a, b) => a - b)
  const medianFPS = calculatePercentile(sortedFPS, 50)
  
  return {
    min: minFPS,
    max: maxFPS,
    avg: avgFPS,
    median: medianFPS,
    p01: fpsP01,
    p05: fpsP05,
    p10: fpsP10,
    p25: fpsP25,
    p75: fpsP75,
    p90: fpsP90,
    p95: fpsP95,
    p97: fpsP97,
    p99: fpsP99,
    iqr: fpsIQR,
    stddev: stddev,
    variance: variance,
    count: frametimeValues.length,
    density: calculateDensityData(fpsValues, calculationMethod)
  }
}
