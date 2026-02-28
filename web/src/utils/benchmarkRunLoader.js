// Utility for loading benchmark data incrementally - one run at a time
// Backend sends pre-calculated stats and downsampled series, so no heavy processing needed

import { processRun } from './benchmarkDataProcessor.js'

/**
 * Downloads benchmark runs one-by-one and maps them to the chart-ready format.
 * Backend sends pre-calculated stats, so processing is just a lightweight format mapping.
 * 
 * @param {number} benchmarkId - The benchmark ID
 * @param {number} totalRuns - Total number of runs to download
 * @param {Object} callbacks - Progress callbacks
 * @param {Function} callbacks.onRunDownloadStart - Called when starting to download a run (runIndex, totalRuns)
 * @param {Function} callbacks.onRunDownloadProgress - Called with download progress for current run (progress 0-100)
 * @param {Function} callbacks.onRunDownloadComplete - Called when a run download completes (runIndex, runData, totalRuns)
 * @param {Function} callbacks.onRunProcessComplete - Called when a run is processed (runIndex, totalRuns)
 * @param {Function} callbacks.onError - Called on error (error, runIndex)
 * @returns {Promise<Array>} Array of processed benchmark runs
 */
export async function loadBenchmarkRunsIncremental(benchmarkId, totalRuns, callbacks = {}) {
  const {
    onRunDownloadStart,
    onRunDownloadProgress,
    onRunDownloadComplete,
    onRunProcessComplete,
    onError
  } = callbacks

  const processedRuns = []

  for (let runIndex = 0; runIndex < totalRuns; runIndex++) {
    try {
      // Notify start of download for this run
      if (onRunDownloadStart) {
        onRunDownloadStart(runIndex, totalRuns)
      }

      // Download this run (pre-calculated data, much smaller than raw)
      const url = `/api/benchmarks/${benchmarkId}/runs/${runIndex}`
      const response = await fetch(url, {
        credentials: 'include'
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || `Failed to load run ${runIndex}`)
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let text = ''

      // Read response body
      while (true) {
        const { done, value } = await reader.read()
        
        if (done) break
        
        text += decoder.decode(value, { stream: true })
        
        // Report indeterminate progress if callback exists
        if (onRunDownloadProgress) {
          onRunDownloadProgress(-1)
        }
      }
      
      text += decoder.decode()

      // Parse JSON for this run (small pre-calculated payload)
      const runData = JSON.parse(text)
      
      // Notify download complete
      if (onRunDownloadComplete) {
        onRunDownloadComplete(runIndex, runData, totalRuns)
      }

      // Map backend format to chart-ready format (lightweight, no computation)
      const processedRun = processRun(runData, runIndex)
      processedRuns.push(processedRun)

      // Notify processing complete
      if (onRunProcessComplete) {
        onRunProcessComplete(runIndex, totalRuns)
      }

    } catch (error) {
      if (onError) {
        onError(error, runIndex)
      }
      throw error
    }
  }

  return processedRuns
}
