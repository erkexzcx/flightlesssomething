/**
 * Web Worker for CPU-intensive benchmark calculations
 * Offloads heavy computation from the main UI thread to improve responsiveness
 */

/* global self */

// Constants for data processing
const OUTLIER_LOW_PERCENTILE = 0.01  // Remove bottom 1% outliers
const OUTLIER_HIGH_PERCENTILE = 0.97 // Remove top 3% outliers
const MAX_DENSITY_POINTS = 100       // Maximum points for density charts
const MAX_FRAMETIME_FOR_INVALID_FPS = 1000000  // Very large frametime for invalid FPS (0 or negative)

// Helper calculation functions
function calculateAverage(data) {
  if (!data || data.length === 0) return 0
  return data.reduce((acc, value) => acc + value, 0) / data.length
}

// Calculate average FPS using harmonic mean (via frametimes)
function calculateAverageFPS(fpsData) {
  if (!fpsData || fpsData.length === 0) return 0
  
  const frametimes = fpsData.map(fps => fps > 0 ? 1000 / fps : MAX_FRAMETIME_FOR_INVALID_FPS)
  const sumFrametimes = frametimes.reduce((acc, ft) => acc + ft, 0)
  const avgFrametime = sumFrametimes / frametimes.length
  
  return avgFrametime > 0 ? 1000 / avgFrametime : 0
}

function calculatePercentile(data, percentile) {
  if (!data || data.length === 0) return 0
  const sorted = [...data].sort((a, b) => a - b)
  return sorted[Math.ceil(percentile / 100 * sorted.length) - 1]
}

// Calculate percentile FPS using harmonic mean method (via frametimes)
function calculatePercentileFPS(fpsData, percentile) {
  if (!fpsData || fpsData.length === 0) return 0
  
  const frametimes = fpsData.map(fps => fps > 0 ? 1000 / fps : MAX_FRAMETIME_FOR_INVALID_FPS)
  const invertedPercentile = 100 - percentile
  const sorted = [...frametimes].sort((a, b) => a - b)
  const n = sorted.length
  
  const index = Math.round((invertedPercentile / 100) * (n + 1))
  const clampedIndex = Math.max(0, Math.min(index, n - 1))
  const frametimePercentile = sorted[clampedIndex]
  
  return frametimePercentile > 0 ? 1000 / frametimePercentile : 0
}

function calculateStandardDeviation(data) {
  if (!data || data.length === 0) return 0
  const mean = calculateAverage(data)
  const squaredDiffs = data.map(value => Math.pow(value - mean, 2))
  const avgSquaredDiff = calculateAverage(squaredDiffs)
  return Math.sqrt(avgSquaredDiff)
}

function calculateVariance(data) {
  if (!data || data.length === 0) return 0
  const mean = calculateAverage(data)
  const squaredDiffs = data.map(value => Math.pow(value - mean, 2))
  return calculateAverage(squaredDiffs)
}

function filterOutliers(data) {
  if (!data || data.length === 0) return []
  const sorted = [...data].sort((a, b) => a - b)
  return sorted.slice(
    Math.floor(sorted.length * OUTLIER_LOW_PERCENTILE), 
    Math.ceil(sorted.length * OUTLIER_HIGH_PERCENTILE)
  )
}

function countOccurrences(data) {
  const counts = {}
  data.forEach(value => {
    const rounded = Math.round(value)
    counts[rounded] = (counts[rounded] || 0) + 1
  })

  const array = Object.keys(counts).map(key => [parseInt(key), counts[key]]).sort((a, b) => a[0] - b[0])

  while (array.length > MAX_DENSITY_POINTS) {
    let minDiff = Infinity
    let minIndex = -1

    for (let i = 0; i < array.length - 1; i++) {
      const diff = array[i + 1][0] - array[i][0]
      if (diff < minDiff) {
        minDiff = diff
        minIndex = i
      }
    }

    array[minIndex][1] += array[minIndex + 1][1]
    array[minIndex][0] = (array[minIndex][0] + array[minIndex + 1][0]) / 2
    array.splice(minIndex + 1, 1)
  }

  return array
}

// Simple decimation function for line charts
function decimateForLineChart(data, targetPoints = 2000) {
  if (!data || data.length <= targetPoints) {
    return data
  }
  
  const step = Math.ceil(data.length / targetPoints)
  const decimated = []
  
  decimated.push(data[0])
  
  for (let i = step; i < data.length - 1; i += step) {
    decimated.push(data[i])
  }
  
  if (data.length > 1) {
    decimated.push(data[data.length - 1])
  }
  
  return decimated
}

// Calculate FPS statistics for all runs
function calculateFPSStats(fpsDataArrays) {
  return fpsDataArrays.map(d => {
    const filtered = filterOutliers(d.data)
    return {
      label: d.label,
      min: calculatePercentileFPS(d.data, 1),
      avg: calculateAverageFPS(d.data),
      max: calculatePercentileFPS(d.data, 97),
      stddev: calculateStandardDeviation(d.data),
      variance: calculateVariance(d.data),
      densityData: countOccurrences(filtered)
    }
  })
}

// Calculate frametime statistics for all runs
function calculateFrametimeStats(frameTimeDataArrays) {
  return frameTimeDataArrays.map(d => {
    const filtered = filterOutliers(d.data)
    return {
      label: d.label,
      min: calculatePercentile(d.data, 1),
      avg: calculateAverage(d.data),
      max: calculatePercentile(d.data, 97),
      stddev: calculateStandardDeviation(d.data),
      variance: calculateVariance(d.data),
      densityData: countOccurrences(filtered)
    }
  })
}

// Calculate summary statistics (averages) for all metrics
function calculateSummaryStats(dataArrays) {
  return {
    fpsAverages: dataArrays.fpsDataArrays.map(d => calculateAverageFPS(d.data)),
    frametimeAverages: dataArrays.frameTimeDataArrays.map(d => calculateAverage(d.data)),
    cpuLoadAverages: dataArrays.cpuLoadDataArrays.map(d => calculateAverage(d.data)),
    gpuLoadAverages: dataArrays.gpuLoadDataArrays.map(d => calculateAverage(d.data)),
    gpuCoreClockAverages: dataArrays.gpuCoreClockDataArrays.map(d => calculateAverage(d.data)),
    gpuMemClockAverages: dataArrays.gpuMemClockDataArrays.map(d => calculateAverage(d.data)),
    cpuPowerAverages: dataArrays.cpuPowerDataArrays.map(d => calculateAverage(d.data)),
    gpuPowerAverages: dataArrays.gpuPowerDataArrays.map(d => calculateAverage(d.data))
  }
}

// Decimate all line chart data for efficient rendering
function decimateAllLineChartData(dataArrays, targetPoints = 2000) {
  return {
    fpsDataArrays: dataArrays.fpsDataArrays.map(d => ({
      label: d.label,
      data: decimateForLineChart(d.data, targetPoints)
    })),
    frameTimeDataArrays: dataArrays.frameTimeDataArrays.map(d => ({
      label: d.label,
      data: decimateForLineChart(d.data, targetPoints)
    })),
    cpuLoadDataArrays: dataArrays.cpuLoadDataArrays.map(d => ({
      label: d.label,
      data: decimateForLineChart(d.data, targetPoints)
    })),
    gpuLoadDataArrays: dataArrays.gpuLoadDataArrays.map(d => ({
      label: d.label,
      data: decimateForLineChart(d.data, targetPoints)
    })),
    cpuTempDataArrays: dataArrays.cpuTempDataArrays.map(d => ({
      label: d.label,
      data: decimateForLineChart(d.data, targetPoints)
    })),
    cpuPowerDataArrays: dataArrays.cpuPowerDataArrays.map(d => ({
      label: d.label,
      data: decimateForLineChart(d.data, targetPoints)
    })),
    gpuTempDataArrays: dataArrays.gpuTempDataArrays.map(d => ({
      label: d.label,
      data: decimateForLineChart(d.data, targetPoints)
    })),
    gpuCoreClockDataArrays: dataArrays.gpuCoreClockDataArrays.map(d => ({
      label: d.label,
      data: decimateForLineChart(d.data, targetPoints)
    })),
    gpuMemClockDataArrays: dataArrays.gpuMemClockDataArrays.map(d => ({
      label: d.label,
      data: decimateForLineChart(d.data, targetPoints)
    })),
    gpuVRAMUsedDataArrays: dataArrays.gpuVRAMUsedDataArrays.map(d => ({
      label: d.label,
      data: decimateForLineChart(d.data, targetPoints)
    })),
    gpuPowerDataArrays: dataArrays.gpuPowerDataArrays.map(d => ({
      label: d.label,
      data: decimateForLineChart(d.data, targetPoints)
    })),
    ramUsedDataArrays: dataArrays.ramUsedDataArrays.map(d => ({
      label: d.label,
      data: decimateForLineChart(d.data, targetPoints)
    })),
    swapUsedDataArrays: dataArrays.swapUsedDataArrays.map(d => ({
      label: d.label,
      data: decimateForLineChart(d.data, targetPoints)
    }))
  }
}

// Message handler
self.addEventListener('message', (event) => {
  const { type, payload, taskId } = event.data

  try {
    let result

    switch (type) {
      case 'calculateFPSStats':
        result = calculateFPSStats(payload.fpsDataArrays)
        break

      case 'calculateFrametimeStats':
        result = calculateFrametimeStats(payload.frameTimeDataArrays)
        break

      case 'calculateSummaryStats':
        result = calculateSummaryStats(payload.dataArrays)
        break

      case 'decimateLineChartData':
        result = decimateAllLineChartData(payload.dataArrays, payload.targetPoints || 2000)
        break

      case 'calculateAll': {
        // Calculate everything at once for initial load
        const fpsStats = calculateFPSStats(payload.dataArrays.fpsDataArrays)
        const frametimeStats = calculateFrametimeStats(payload.dataArrays.frameTimeDataArrays)
        const summaryStats = calculateSummaryStats(payload.dataArrays)
        const decimatedData = decimateAllLineChartData(payload.dataArrays, payload.targetPoints || 2000)
        
        result = {
          fpsStats,
          frametimeStats,
          summaryStats,
          decimatedData
        }
        break
      }

      default:
        throw new Error(`Unknown task type: ${type}`)
    }

    // Send result back to main thread
    self.postMessage({
      type: 'success',
      taskId,
      result
    })
  } catch (error) {
    // Send error back to main thread
    self.postMessage({
      type: 'error',
      taskId,
      error: error.message
    })
  }
})
