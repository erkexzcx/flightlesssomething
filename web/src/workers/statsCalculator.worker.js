/**
 * Web Worker for calculating statistics in parallel
 * This worker handles one calculation method (either linear-interpolation or mangohud-threshold)
 */

// Calculate percentile with linear interpolation
function calculatePercentileLinearInterpolation(sortedData, percentile) {
  if (!sortedData || sortedData.length === 0) {
    return 0
  }
  
  const n = sortedData.length
  const idx = (percentile / 100) * (n - 1)
  const lower = Math.floor(idx)
  const upper = Math.ceil(idx)
  
  if (lower === upper) {
    return sortedData[lower]
  }
  
  const fraction = idx - lower
  return sortedData[lower] * (1 - fraction) + sortedData[upper] * fraction
}

// Calculate percentile using MangoHud's threshold method
function calculatePercentileMangoHudThreshold(sortedData, percentile) {
  if (!sortedData || sortedData.length === 0) {
    return 0
  }
  
  const n = sortedData.length
  const idx = Math.floor((percentile / 100) * n)
  const clampedIdx = Math.min(Math.max(idx, 0), n - 1)
  
  return sortedData[clampedIdx]
}

// Calculate density data
function calculateDensityData(values, calculationMethod) {
  if (!values || values.length === 0) return []
  
  const sorted = [...values].sort((a, b) => a - b)
  const calculatePercentile = calculationMethod === 'mangohud-threshold' 
    ? calculatePercentileMangoHudThreshold 
    : calculatePercentileLinearInterpolation
  const p01Value = calculatePercentile(sorted, 1)
  const p99Value = calculatePercentile(sorted, 99)
  const filtered = sorted.filter(v => v >= p01Value && v <= p99Value)
  
  const counts = {}
  filtered.forEach(value => {
    const rounded = Math.round(value)
    counts[rounded] = (counts[rounded] || 0) + 1
  })
  
  const array = Object.keys(counts).map(key => [parseInt(key), counts[key]]).sort((a, b) => a[0] - b[0])
  
  return array
}

// Calculate statistics
function calculateStats(values, calculationMethod) {
  if (!values || values.length === 0) {
    return { min: 0, max: 0, avg: 0, p01: 0, p99: 0, stddev: 0, variance: 0, density: [] }
  }

  const sorted = [...values].sort((a, b) => a - b)
  const sum = values.reduce((acc, val) => acc + val, 0)
  const avg = sum / values.length
  
  const squaredDiffs = values.map(val => Math.pow(val - avg, 2))
  const variance = squaredDiffs.reduce((acc, val) => acc + val, 0) / values.length
  const stddev = Math.sqrt(variance)
  
  const calculatePercentile = calculationMethod === 'mangohud-threshold' 
    ? calculatePercentileMangoHudThreshold 
    : calculatePercentileLinearInterpolation
  
  return {
    min: sorted[0],
    max: sorted[sorted.length - 1],
    avg: avg,
    p01: calculatePercentile(sorted, 1),
    p99: calculatePercentile(sorted, 99),
    stddev: stddev,
    variance: variance,
    density: calculateDensityData(values, calculationMethod)
  }
}

// Calculate FPS statistics from frametime
function calculateFPSStatsFromFrametime(frametimeValues, calculationMethod) {
  if (!frametimeValues || frametimeValues.length === 0) {
    return { min: 0, max: 0, avg: 0, p01: 0, p99: 0, stddev: 0, variance: 0, density: [] }
  }

  const sorted = [...frametimeValues].sort((a, b) => a - b)
  
  const calculatePercentile = calculationMethod === 'mangohud-threshold' 
    ? calculatePercentileMangoHudThreshold 
    : calculatePercentileLinearInterpolation
  
  const frametimeP01 = calculatePercentile(sorted, 1)
  const frametimeP99 = calculatePercentile(sorted, 99)
  
  const fpsP99 = frametimeP01 > 0 ? 1000 / frametimeP01 : 0
  const fpsP01 = frametimeP99 > 0 ? 1000 / frametimeP99 : 0
  
  const avgFrametime = frametimeValues.reduce((acc, val) => acc + val, 0) / frametimeValues.length
  const avgFPS = avgFrametime > 0 ? 1000 / avgFrametime : 0
  
  const fpsValues = frametimeValues.map(ft => ft > 0 ? 1000 / ft : 0)
  
  const minFrametime = sorted[0]
  const maxFrametime = sorted[sorted.length - 1]
  const maxFPS = minFrametime > 0 ? 1000 / minFrametime : 0
  const minFPS = maxFrametime > 0 ? 1000 / maxFrametime : 0
  
  const fpsSum = fpsValues.reduce((acc, val) => acc + val, 0)
  const fpsMean = fpsSum / fpsValues.length
  const squaredDiffs = fpsValues.map(val => Math.pow(val - fpsMean, 2))
  const variance = squaredDiffs.reduce((acc, val) => acc + val, 0) / fpsValues.length
  const stddev = Math.sqrt(variance)
  
  return {
    min: minFPS,
    max: maxFPS,
    avg: avgFPS,
    p01: fpsP01,
    p99: fpsP99,
    stddev: stddev,
    variance: variance,
    density: calculateDensityData(fpsValues, calculationMethod)
  }
}

// Listen for messages from main thread
self.onmessage = function(e) {
  const { runData, calculationMethod, metrics } = e.data
  
  const stats = {}
  const frametimeData = runData.DataFrameTime
  
  metrics.forEach(metric => {
    const backendFieldName = 'Data' + metric
    const data = runData[backendFieldName]
    
    if (!data || data.length === 0) {
      stats[metric] = { min: 0, max: 0, avg: 0, p01: 0, p99: 0, stddev: 0, variance: 0, density: [] }
      return
    }
    
    // Calculate statistics
    if (metric === 'FPS' && frametimeData && frametimeData.length > 0) {
      stats[metric] = calculateFPSStatsFromFrametime(frametimeData, calculationMethod)
    } else {
      stats[metric] = calculateStats(data, calculationMethod)
    }
  })
  
  // Send results back to main thread
  self.postMessage({ stats, calculationMethod })
}
