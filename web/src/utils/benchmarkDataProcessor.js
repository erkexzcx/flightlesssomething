/**
 * Process benchmark run data incrementally to avoid browser freezing.
 * This extracts only the necessary data for charts and discards raw data.
 */

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

// Calculate percentile with linear interpolation (matches scientific/numpy method)
// This provides more accurate percentile values than simple floor-based indexing
function calculatePercentileLinearInterpolation(sortedData, percentile) {
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

// Calculate percentile using MangoHud's frametime-based threshold method (without interpolation)
// This uses a simple floor-based approach to find the percentile value
function calculatePercentileMangoHudThreshold(sortedData, percentile) {
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

// Calculate statistics for an array of values
function calculateStats(values, calculationMethod = 'linear-interpolation') {
  if (!values || values.length === 0) {
    return { min: 0, max: 0, avg: 0, p01: 0, p97: 0, stddev: 0, variance: 0, density: [] }
  }

  const sorted = [...values].sort((a, b) => a - b)
  const sum = values.reduce((acc, val) => acc + val, 0)
  const avg = sum / values.length
  
  // Calculate variance and standard deviation from FULL data
  const squaredDiffs = values.map(val => Math.pow(val - avg, 2))
  const variance = squaredDiffs.reduce((acc, val) => acc + val, 0) / values.length
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
    stddev: stddev,  // Pre-calculated from FULL data
    variance: variance,  // Pre-calculated from FULL data
    density: calculateDensityData(values, calculationMethod) // Pre-calculate density from FULL data
  }
}

// Calculate FPS statistics from frametime data
// This is the correct way to calculate FPS statistics, as averaging FPS values directly is incorrect
function calculateFPSStatsFromFrametime(frametimeValues, calculationMethod = 'linear-interpolation') {
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
    p97: fpsP97,
    stddev: stddev,
    variance: variance,
    density: calculateDensityData(fpsValues, calculationMethod)
  }
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
