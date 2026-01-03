/**
 * Process benchmark run data incrementally to avoid browser freezing.
 * This extracts only the necessary data for charts and discards raw data.
 */

import {
  calculatePercentileLinearInterpolation,
  calculatePercentileMangoHudThreshold,
  calculateStats,
  calculateFPSStatsFromFrametime
} from './statsCalculations'

// Downsample data points using Largest Triangle Three Buckets (LTTB) algorithm
function downsampleLTTB(data, threshold) {
  // Handle edge cases
  if (!data || data.length === 0) {
    return []
  }
  
  if (data.length <= threshold) {
    return data
  }

  const sampled = []
  const bucketSize = (data.length - 2) / (threshold - 2)

  // Always add first point
  sampled.push(data[0])

  for (let i = 0; i < threshold - 2; i++) {
    const avgRangeStart = Math.floor((i + 1) * bucketSize) + 1
    const avgRangeEnd = Math.min(Math.floor((i + 2) * bucketSize) + 1, data.length)
    const avgRangeLength = avgRangeEnd - avgRangeStart

    let avgX = 0
    let avgY = 0
    let validPoints = 0

    // Calculate average with bounds checking
    for (let j = avgRangeStart; j < avgRangeEnd; j++) {
      if (j >= data.length || !data[j] || !Array.isArray(data[j]) || data[j].length < 2) {
        continue
      }
      avgX += data[j][0]
      avgY += data[j][1]
      validPoints++
    }
    
    if (validPoints === 0) {
      // Skip this bucket if no valid points
      continue
    }
    
    avgX /= validPoints
    avgY /= validPoints

    const rangeStart = Math.floor(i * bucketSize) + 1
    const rangeEnd = Math.min(Math.floor((i + 1) * bucketSize) + 1, data.length)

    let maxArea = -1
    let maxAreaPoint = null

    const lastPoint = sampled[sampled.length - 1]
    if (!lastPoint || !Array.isArray(lastPoint) || lastPoint.length < 2) {
      // If last point is invalid, skip this iteration
      continue
    }
    
    const pointAX = lastPoint[0]
    const pointAY = lastPoint[1]

    for (let j = rangeStart; j < rangeEnd; j++) {
      if (j >= data.length || !data[j] || !Array.isArray(data[j]) || data[j].length < 2) {
        continue
      }
      
      const area = Math.abs(
        (pointAX - avgX) * (data[j][1] - pointAY) -
        (pointAX - data[j][0]) * (avgY - pointAY)
      ) * 0.5

      if (area > maxArea) {
        maxArea = area
        maxAreaPoint = data[j]
      }
    }

    if (maxAreaPoint) {
      sampled.push(maxAreaPoint)
    }
  }

  // Always add last point if it exists and is valid
  const lastDataPoint = data[data.length - 1]
  if (lastDataPoint && Array.isArray(lastDataPoint) && lastDataPoint.length >= 2) {
    sampled.push(lastDataPoint)
  }

  return sampled
}

// Calculate density data for histogram/area charts
// Filters outliers (1st-97th percentile) and counts occurrences
// No arbitrary limit - natural bin count based on data range
// (e.g., FPS 0-2000 = max 2000 bins, FrameTime 0-100 = max 100 bins)
function calculateDensityData(values, calculationMethod = 'linear-interpolation') {
  if (!values || values.length === 0) return []
  
  // Filter outliers (keep only 1st-97th percentile)
  const sorted = [...values].sort((a, b) => a - b)
  const calculatePercentile = calculationMethod === 'mangohud-threshold' 
    ? calculatePercentileMangoHudThreshold 
    : calculatePercentileLinearInterpolation
  const p01Value = calculatePercentile(sorted, 1)
  const p97Value = calculatePercentile(sorted, 97)
  const filtered = sorted.filter(v => v >= p01Value && v <= p97Value)
  
  // Count occurrences (round to integers)
  const counts = {}
  filtered.forEach(value => {
    const rounded = Math.round(value)
    counts[rounded] = (counts[rounded] || 0) + 1
  })
  
  // Convert to array format [[value, count], ...] and sort by value
  // No downsampling - density data is small compared to downsampled series
  const array = Object.keys(counts).map(key => [parseInt(key), counts[key]]).sort((a, b) => a[0] - b[0])
  
  return array
}

/**
 * Process a single benchmark run and extract chart-ready data
 * Uses Web Workers for parallel calculation of both methods
 * @param {Object} runData - Raw benchmark data for one run
 * @param {number} runIndex - Index of this run
 * @param {number} maxPoints - Maximum points to keep for line charts (default: 2000)
 * @returns {Promise<Object>} Processed data ready for charts
 */
export async function processRun(runData, runIndex, maxPoints = 2000) {
  const processed = {
    // Metadata
    runIndex,
    label: runData.Label || `Run ${runIndex + 1}`,
    specOS: runData.SpecOS || '',
    specGPU: runData.SpecGPU || '',
    specCPU: runData.SpecCPU || '',
    specRAM: runData.SpecRAM || '',
    // Build SpecOSSpecific from individual fields since backend sends them separately
    specOSSpecific: {
      SpecLinuxKernel: runData.SpecLinuxKernel || '',
      SpecLinuxScheduler: runData.SpecLinuxScheduler || ''
    },
    
    // Downsampled time-series data for line charts
    series: {},
    
    // Statistical summaries for bar charts (both calculation methods)
    stats: {},
    statsMangoHud: {}
  }

  // Extract all metrics
  // Backend sends these with "Data" prefix (e.g., DataFPS, DataFrameTime)
  const metrics = [
    'FPS', 'FrameTime', 'CPULoad', 'CPUTemp', 'CPUPower',
    'GPULoad', 'GPUTemp', 'GPUCoreClock', 'GPUMemClock',
    'GPUVRAMUsed', 'GPUPower', 'RAMUsed', 'SwapUsed'
  ]

  // Process downsampled series data (not CPU intensive, do in main thread)
  metrics.forEach(metric => {
    const backendFieldName = 'Data' + metric
    const data = runData[backendFieldName]
    
    if (!data || data.length === 0) {
      processed.series[metric] = []
      processed.stats[metric] = { min: 0, max: 0, avg: 0, p01: 0, p97: 0, stddev: 0, variance: 0, density: [] }
      processed.statsMangoHud[metric] = { min: 0, max: 0, avg: 0, p01: 0, p97: 0, stddev: 0, variance: 0, density: [] }
      return
    }

    // Convert to [x, y] format and downsample
    const points = data.map((value, index) => [index, value])
    processed.series[metric] = downsampleLTTB(points, Math.min(maxPoints, data.length))
  })
  
  // Calculate statistics in parallel using Web Workers
  try {
    const results = await calculateStatsInParallel(runData, metrics)
    processed.stats = results.stats
    processed.statsMangoHud = results.statsMangoHud
  } catch (error) {
    console.error('Failed to calculate stats in parallel, falling back to sequential:', error)
    // Fallback to sequential calculation
    const frametimeData = runData.DataFrameTime
    metrics.forEach(metric => {
      const backendFieldName = 'Data' + metric
      const data = runData[backendFieldName]
      
      if (!data || data.length === 0) {
        return
      }
      
      if (metric === 'FPS' && frametimeData && frametimeData.length > 0) {
        processed.stats[metric] = calculateFPSStatsFromFrametime(frametimeData, 'linear-interpolation')
        processed.statsMangoHud[metric] = calculateFPSStatsFromFrametime(frametimeData, 'mangohud-threshold')
      } else {
        processed.stats[metric] = calculateStats(data, 'linear-interpolation')
        processed.statsMangoHud[metric] = calculateStats(data, 'mangohud-threshold')
      }
    })
  }

  return processed
}

/**
 * Calculate statistics in parallel using Web Workers
 * Creates 2 workers - one for each calculation method
 * @param {Object} runData - Raw benchmark data
 * @param {Array} metrics - List of metrics to calculate
 * @returns {Promise<Object>} Object with stats and statsMangoHud
 */
function calculateStatsInParallel(runData, metrics) {
  return new Promise((resolve, reject) => {
    // Create two workers for parallel calculation
    const worker1 = new Worker(new URL('../workers/statsCalculator.worker.js', import.meta.url), { type: 'module' })
    const worker2 = new Worker(new URL('../workers/statsCalculator.worker.js', import.meta.url), { type: 'module' })
    
    const results = {}
    let completedWorkers = 0
    
    const handleWorkerMessage = (e) => {
      const { stats, calculationMethod } = e.data
      
      if (calculationMethod === 'linear-interpolation') {
        results.stats = stats
      } else {
        results.statsMangoHud = stats
      }
      
      completedWorkers++
      
      if (completedWorkers === 2) {
        // Both workers completed
        worker1.terminate()
        worker2.terminate()
        resolve(results)
      }
    }
    
    const handleWorkerError = (error) => {
      worker1.terminate()
      worker2.terminate()
      reject(error)
    }
    
    worker1.onmessage = handleWorkerMessage
    worker1.onerror = handleWorkerError
    
    worker2.onmessage = handleWorkerMessage
    worker2.onerror = handleWorkerError
    
    // Send calculation tasks to workers
    worker1.postMessage({
      runData,
      calculationMethod: 'linear-interpolation',
      metrics
    })
    
    worker2.postMessage({
      runData,
      calculationMethod: 'mangohud-threshold',
      metrics
    })
  })
}
