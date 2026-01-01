// Web Worker for heavy benchmark calculations
// This offloads CPU-intensive operations from the main thread to prevent UI freezing

const OUTLIER_LOW_PERCENTILE = 0.01
const OUTLIER_HIGH_PERCENTILE = 0.97
const MAX_DENSITY_POINTS = 100
const MAX_FRAMETIME_FOR_INVALID_FPS = 1000000

// Helper functions (duplicated from main thread for worker isolation)
function calculateAverage(data) {
  if (!data || data.length === 0) return 0
  return data.reduce((acc, value) => acc + value, 0) / data.length
}

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

  let array = Object.keys(counts).map(key => [parseInt(key), counts[key]]).sort((a, b) => a[0] - b[0])

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

// Calculate FPS statistics for a single dataset
function calculateFpsStats(label, data) {
  const filtered = filterOutliers(data)
  return {
    label: label,
    data: data,
    min: calculatePercentileFPS(data, 1),
    avg: calculateAverageFPS(data),
    max: calculatePercentileFPS(data, 97),
    stddev: calculateStandardDeviation(data),
    variance: calculateVariance(data),
    filteredOutliers: filtered,
    densityData: countOccurrences(filtered)
  }
}

// Calculate frametime statistics for a single dataset
function calculateFrametimeStats(label, data) {
  const filtered = filterOutliers(data)
  return {
    label: label,
    data: data,
    min: calculatePercentile(data, 1),
    avg: calculateAverage(data),
    max: calculatePercentile(data, 97),
    stddev: calculateStandardDeviation(data),
    variance: calculateVariance(data),
    filteredOutliers: filtered,
    densityData: countOccurrences(filtered)
  }
}

// Message handler
self.onmessage = function(e) {
  const { type, data } = e.data

  try {
    switch (type) {
      case 'calculateFpsStats': {
        // Calculate stats for all FPS datasets
        const results = data.fpsDataArrays.map(d => 
          calculateFpsStats(d.label, d.data)
        )
        self.postMessage({ type: 'fpsStatsComplete', data: results })
        break
      }

      case 'calculateFrametimeStats': {
        // Calculate stats for all frametime datasets
        const results = data.frametimeDataArrays.map(d => 
          calculateFrametimeStats(d.label, d.data)
        )
        self.postMessage({ type: 'frametimeStatsComplete', data: results })
        break
      }

      case 'calculateSummaryStats': {
        // Calculate summary statistics
        const results = {
          fpsAverages: data.fpsDataArrays.map(d => calculateAverageFPS(d.data)),
          frametimeAverages: data.frameTimeDataArrays.map(d => calculateAverage(d.data)),
          cpuLoadAverages: data.cpuLoadDataArrays.map(d => calculateAverage(d.data)),
          gpuLoadAverages: data.gpuLoadDataArrays.map(d => calculateAverage(d.data)),
          gpuCoreClockAverages: data.gpuCoreClockDataArrays.map(d => calculateAverage(d.data)),
          gpuMemClockAverages: data.gpuMemClockDataArrays.map(d => calculateAverage(d.data)),
          cpuPowerAverages: data.cpuPowerDataArrays.map(d => calculateAverage(d.data)),
          gpuPowerAverages: data.gpuPowerDataArrays.map(d => calculateAverage(d.data))
        }
        self.postMessage({ type: 'summaryStatsComplete', data: results })
        break
      }

      default:
        self.postMessage({ type: 'error', error: `Unknown message type: ${type}` })
    }
  } catch (error) {
    self.postMessage({ type: 'error', error: error.message })
  }
}
