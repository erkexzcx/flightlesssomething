/**
 * Process benchmark run data incrementally to avoid browser freezing.
 * This extracts only the necessary data for charts and discards raw data.
 */

// Downsample data points using Largest Triangle Three Buckets (LTTB) algorithm
function downsampleLTTB(data, threshold) {
  if (data.length <= threshold) {
    return data
  }

  const sampled = []
  const bucketSize = (data.length - 2) / (threshold - 2)

  // Always add first point
  sampled.push(data[0])

  for (let i = 0; i < threshold - 2; i++) {
    const avgRangeStart = Math.floor((i + 1) * bucketSize) + 1
    const avgRangeEnd = Math.floor((i + 2) * bucketSize) + 1
    const avgRangeLength = avgRangeEnd - avgRangeStart

    let avgX = 0
    let avgY = 0

    for (let j = avgRangeStart; j < avgRangeEnd; j++) {
      avgX += data[j][0]
      avgY += data[j][1]
    }
    avgX /= avgRangeLength
    avgY /= avgRangeLength

    const rangeStart = Math.floor(i * bucketSize) + 1
    const rangeEnd = Math.floor((i + 1) * bucketSize) + 1

    let maxArea = -1
    let maxAreaPoint = null

    const pointAX = sampled[sampled.length - 1][0]
    const pointAY = sampled[sampled.length - 1][1]

    for (let j = rangeStart; j < rangeEnd; j++) {
      const area = Math.abs(
        (pointAX - avgX) * (data[j][1] - pointAY) -
        (pointAX - data[j][0]) * (avgY - pointAY)
      ) * 0.5

      if (area > maxArea) {
        maxArea = area
        maxAreaPoint = data[j]
      }
    }

    sampled.push(maxAreaPoint)
  }

  // Always add last point
  sampled.push(data[data.length - 1])

  return sampled
}

// Calculate density data for histogram/area charts
// Filters outliers (1st-99th percentile) and counts occurrences
// No arbitrary limit - natural bin count based on data range
// (e.g., FPS 0-2000 = max 2000 bins, FrameTime 0-100 = max 100 bins)
function calculateDensityData(values) {
  if (!values || values.length === 0) return []
  
  // Filter outliers (keep only 1st-99th percentile)
  const sorted = [...values].sort((a, b) => a - b)
  const low = Math.floor(sorted.length * 0.01)
  const high = Math.ceil(sorted.length * 0.99)
  const filtered = sorted.slice(low, high)
  
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
function calculateStats(values) {
  if (!values || values.length === 0) {
    return { min: 0, max: 0, avg: 0, p01: 0, p99: 0, density: [] }
  }

  const sorted = [...values].sort((a, b) => a - b)
  const sum = values.reduce((acc, val) => acc + val, 0)
  
  return {
    min: sorted[0],
    max: sorted[sorted.length - 1],
    avg: sum / values.length,
    p01: sorted[Math.floor(values.length * 0.01)],
    p99: sorted[Math.floor(values.length * 0.99)],
    density: calculateDensityData(values) // Pre-calculate density from FULL data
  }
}

/**
 * Process a single benchmark run and extract chart-ready data
 * @param {Object} runData - Raw benchmark data for one run
 * @param {number} runIndex - Index of this run
 * @param {number} maxPoints - Maximum points to keep for line charts (default: 2000)
 * @returns {Object} Processed data ready for charts
 */
export function processRun(runData, runIndex, maxPoints = 2000) {
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
    
    // Statistical summaries for bar charts
    stats: {}
  }

  // Extract all metrics
  // Backend sends these with "Data" prefix (e.g., DataFPS, DataFrameTime)
  const metrics = [
    'FPS', 'FrameTime', 'CPULoad', 'CPUTemp', 'CPUPower',
    'GPULoad', 'GPUTemp', 'GPUCoreClock', 'GPUMemClock',
    'GPUVRAMUsed', 'GPUPower', 'RAMUsed', 'SwapUsed'
  ]

  metrics.forEach(metric => {
    // Backend sends data with "Data" prefix
    const backendFieldName = 'Data' + metric
    const data = runData[backendFieldName]
    
    if (!data || data.length === 0) {
      processed.series[metric] = []
      processed.stats[metric] = { min: 0, max: 0, avg: 0, p01: 0, p99: 0, density: [] }
      return
    }

    // Convert to [x, y] format and downsample
    const points = data.map((value, index) => [index, value])
    processed.series[metric] = downsampleLTTB(points, Math.min(maxPoints, data.length))
    
    // Calculate statistics
    processed.stats[metric] = calculateStats(data)
  })

  return processed
}

/**
 * Merge processed runs into a single dataset for charts
 * @param {Array} processedRuns - Array of processed run data
 * @returns {Object} Combined dataset ready for chart rendering
 */
export function mergeProcessedRuns(processedRuns) {
  return {
    runs: processedRuns,
    runCount: processedRuns.length,
    labels: processedRuns.map(r => r.label),
    
    // Helper to get series data for all runs for a specific metric
    getSeriesData: (metric) => {
      return processedRuns.map((run, index) => ({
        name: run.label,
        data: run.series[metric] || [],
        color: undefined // Let Highcharts assign colors
      }))
    },
    
    // Helper to get stats for all runs for a specific metric
    getStats: (metric) => {
      return processedRuns.map(run => run.stats[metric] || { min: 0, max: 0, avg: 0, p01: 0, p99: 0, density: [] })
    }
  }
}

/**
 * Convert processed data back to legacy format for chart component compatibility
 * This creates objects that look like the original benchmark data structure
 * but use downsampled series data instead of full arrays
 * 
 * @param {Array} processedRuns - Array of processed run data
 * @returns {Array} Array in legacy format expected by BenchmarkCharts component
 */
export function convertToLegacyFormat(processedRuns) {
  return processedRuns.map(run => {
    const legacy = {
      // Metadata (using capitalized keys for compatibility)
      Label: run.label || run.Label || '',
      SpecOS: run.specOS || run.SpecOS || '',
      SpecGPU: run.specGPU || run.SpecGPU || '',
      SpecCPU: run.specCPU || run.SpecCPU || '',
      SpecRAM: run.specRAM || run.SpecRAM || '',
      SpecOSSpecific: run.specOSSpecific || run.SpecOSSpecific || {},
      SpecLinuxKernel: run.specLinuxKernel || run.SpecLinuxKernel || '',
      SpecLinuxScheduler: run.specLinuxScheduler || run.SpecLinuxScheduler || ''
    }

    // Convert downsampled series back to simple arrays
    // Extract just the Y values from [x, y] pairs
    const metrics = [
      'FPS', 'FrameTime', 'CPULoad', 'CPUTemp', 'CPUPower',
      'GPULoad', 'GPUTemp', 'GPUCoreClock', 'GPUMemClock',
      'GPUVRAMUsed', 'GPUPower', 'RAMUsed', 'SwapUsed'
    ]

    metrics.forEach(metric => {
      const series = run.series?.[metric] || []
      // Convert [[x1, y1], [x2, y2], ...] to [y1, y2, ...]
      legacy[metric] = series.map(point => point[1])
    })

    return legacy
  })
}
