// Utility for loading benchmark data incrementally - one run at a time
// This prevents browser freezing and provides detailed progress tracking

/**
 * Downloads benchmark runs one-by-one and processes them incrementally
 * @param {number} benchmarkId - The benchmark ID
 * @param {number} totalRuns - Total number of runs to download
 * @param {Object} callbacks - Progress callbacks
 * @param {Function} callbacks.onRunDownloadStart - Called when starting to download a run (runIndex, totalRuns)
 * @param {Function} callbacks.onRunDownloadProgress - Called with download progress for current run (progress 0-100)
 * @param {Function} callbacks.onRunDownloadComplete - Called when a run download completes (runIndex, runData)
 * @param {Function} callbacks.onRunProcessComplete - Called when a run is processed (runIndex, totalRuns)
 * @param {Function} callbacks.onError - Called on error (error, runIndex)
 * @returns {Promise<Array>} Array of all benchmark runs
 */
export async function loadBenchmarkRunsIncremental(benchmarkId, totalRuns, callbacks = {}) {
  const {
    onRunDownloadStart,
    onRunDownloadProgress,
    onRunDownloadComplete,
    onRunProcessComplete,
    onError
  } = callbacks

  const runs = []

  for (let runIndex = 0; runIndex < totalRuns; runIndex++) {
    try {
      // Notify start of download for this run
      if (onRunDownloadStart) {
        onRunDownloadStart(runIndex, totalRuns)
      }

      // Download this run
      const url = `/api/benchmarks/${benchmarkId}/runs/${runIndex}`
      const response = await fetch(url, {
        credentials: 'include'
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || `Failed to load run ${runIndex}`)
      }

      // Track download progress if Content-Length is available
      const contentLength = response.headers.get('Content-Length')
      const total = contentLength ? parseInt(contentLength, 10) : 0

      let loaded = 0
      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let text = ''

      // Read response body with progress tracking
      while (true) {
        const { done, value } = await reader.read()
        
        if (done) break
        
        text += decoder.decode(value, { stream: true })
        loaded += value.length
        
        if (onRunDownloadProgress && total > 0) {
          const progress = Math.round((loaded / total) * 100)
          onRunDownloadProgress(progress)
        }
      }
      
      text += decoder.decode()

      // Parse JSON for this run (small enough to do on main thread)
      const runData = JSON.parse(text)
      
      // Notify download complete
      if (onRunDownloadComplete) {
        onRunDownloadComplete(runIndex, runData)
      }

      // Store the run
      runs.push(runData)

      // Notify processing complete
      if (onRunProcessComplete) {
        onRunProcessComplete(runIndex, totalRuns)
      }

      // Small delay to allow UI to update
      await new Promise(resolve => setTimeout(resolve, 10))

    } catch (error) {
      if (onError) {
        onError(error, runIndex)
      }
      throw error
    }
  }

  return runs
}
