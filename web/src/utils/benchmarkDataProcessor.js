/**
 * Process benchmark run data from pre-calculated backend response.
 * The backend now sends pre-calculated stats and downsampled series,
 * so this module just maps the response to the format expected by charts.
 */

/**
 * Process a single pre-calculated benchmark run for chart rendering.
 * The backend sends data with all stats pre-computed - no client-side calculation needed.
 * @param {Object} runData - Pre-calculated benchmark data from the API
 * @param {number} runIndex - Index of this run
 * @returns {Object} Data ready for charts
 */
export function processRun(runData, runIndex) {
  return {
    // Metadata
    runIndex,
    label: runData.label || `Run ${runIndex + 1}`,
    specOS: runData.specOS || '',
    specGPU: runData.specGPU || '',
    specCPU: runData.specCPU || '',
    specRAM: runData.specRAM || '',
    specLinuxKernel: runData.specLinuxKernel || '',
    specLinuxScheduler: runData.specLinuxScheduler || '',
    totalDataPoints: runData.totalDataPoints || 0,

    // Pre-computed downsampled time-series data for line charts
    series: runData.series || {},

    // Pre-computed statistical summaries (Linear Interpolation method)
    stats: runData.stats || {},

    // Pre-computed statistical summaries (MangoHud method)
    statsMangoHud: runData.statsMangoHud || {}
  }
}
