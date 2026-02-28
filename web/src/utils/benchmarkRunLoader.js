// Utility for loading benchmark data in parallel using a worker pool pattern.
// Concurrency is based on navigator.hardwareConcurrency (available CPU cores).
// Backend sends pre-calculated stats and downsampled series, so no heavy processing needed.

import { processRun } from './benchmarkDataProcessor.js'

/**
 * Determine optimal concurrency for parallel run loading.
 * Uses navigator.hardwareConcurrency when available, capped to avoid
 * overwhelming the server or browser connection limits.
 * @param {number} totalRuns - Total number of runs to load
 * @returns {number} Number of concurrent requests to use
 */
export function getConcurrency(totalRuns) {
  const cores = (typeof navigator !== 'undefined' && navigator.hardwareConcurrency) || 4
  // Cap at 6 to stay within browser per-origin connection limits (typically 6)
  return Math.max(1, Math.min(cores, totalRuns, 6))
}

/**
 * Downloads benchmark runs in parallel and maps them to the chart-ready format.
 * Uses a worker pool pattern: N workers pull from a shared queue of run indices.
 * Backend sends pre-calculated stats, so processing is just a lightweight format mapping.
 * 
 * @param {number} benchmarkId - The benchmark ID
 * @param {number} totalRuns - Total number of runs to download
 * @param {Object} callbacks - Progress callbacks
 * @param {Function} callbacks.onRunDownloadStart - Called when starting to download a run (runIndex, totalRuns)
 * @param {Function} callbacks.onRunDownloadProgress - Called with download progress for current run (progress 0-100)
 * @param {Function} callbacks.onRunDownloadComplete - Called when a run download completes (runIndex, runData, totalRuns)
 * @param {Function} callbacks.onRunProcessComplete - Called when a run is processed (runIndex, completedCount, totalRuns)
 * @param {Function} callbacks.onError - Called on error (error, runIndex)
 * @returns {Promise<Array>} Array of processed benchmark runs in original index order
 */
export async function loadBenchmarkRunsIncremental(benchmarkId, totalRuns, callbacks = {}) {
  const {
    onRunDownloadStart,
    onRunDownloadProgress,
    onRunDownloadComplete,
    onRunProcessComplete,
    onError
  } = callbacks

  if (totalRuns === 0) {
    return []
  }

  const concurrency = getConcurrency(totalRuns)
  const processedRuns = new Array(totalRuns)
  let nextIndex = 0
  let completedCount = 0

  async function loadRun(runIndex) {
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
    processedRuns[runIndex] = processedRun
    completedCount++

    // Notify processing complete with completed count
    if (onRunProcessComplete) {
      onRunProcessComplete(runIndex, completedCount, totalRuns)
    }
  }

  // Worker pool: each worker pulls the next available run index
  async function worker() {
    while (nextIndex < totalRuns) {
      const runIndex = nextIndex++
      try {
        await loadRun(runIndex)
      } catch (error) {
        if (onError) {
          onError(error, runIndex)
        }
        throw error
      }
    }
  }

  // Launch concurrent workers
  const workers = Array.from({ length: concurrency }, () => worker())
  await Promise.all(workers)

  // Filter out any undefined slots (shouldn't happen, but defensive)
  return processedRuns.filter(run => run !== undefined)
}
