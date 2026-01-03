/**
 * Web Worker for calculating statistics in parallel
 * This worker handles one calculation method (either linear-interpolation or mangohud-threshold)
 * 
 * NOTE: If you modify calculation functions, also update:
 * - /debugcalc page (views/DebugCalc.vue)
 * - utils/statsCalculations.js (shared utilities)
 */

import {
  calculatePercentileLinearInterpolation,
  calculatePercentileMangoHudThreshold,
  calculateStats,
  calculateFPSStatsFromFrametime
} from '../utils/statsCalculations'

// Listen for messages from main thread
self.onmessage = function(e) {
  const { runData, calculationMethod, metrics } = e.data
  
  const stats = {}
  const frametimeData = runData.DataFrameTime
  
  metrics.forEach(metric => {
    const backendFieldName = 'Data' + metric
    const data = runData[backendFieldName]
    
    if (!data || data.length === 0) {
      stats[metric] = { min: 0, max: 0, avg: 0, p01: 0, p97: 0, stddev: 0, variance: 0, density: [] }
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
