// Utility for loading benchmark data with progress tracking
// Handles both download and parsing phases with progress callbacks

import JsonParserWorker from '../workers/jsonParser.worker.js?worker'

/**
 * Downloads and parses benchmark data with progress tracking
 * @param {string} url - The URL to fetch data from
 * @param {Object} callbacks - Progress callbacks
 * @param {Function} callbacks.onDownloadProgress - Called with download progress (0-100)
 * @param {Function} callbacks.onParseProgress - Called with parse progress (0-100)
 * @returns {Promise<any>} The parsed JSON data
 */
export async function loadBenchmarkDataWithProgress(url, { onDownloadProgress, onParseProgress } = {}) {
  // Phase 1: Download with progress tracking
  const response = await fetch(url, {
    credentials: 'include'
  })

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}))
    throw new Error(errorData.error || 'Failed to load benchmark data')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let text = ''

  // Read the response body with indeterminate progress tracking
  // Stream directly to string to avoid memory overhead from Blob conversion
  while (true) {
    const { done, value } = await reader.read()
    
    if (done) break
    
    // Decode chunk directly to string instead of accumulating Uint8Arrays
    text += decoder.decode(value, { stream: true })
    
    // Report indeterminate progress (server doesn't send Content-Length)
    if (onDownloadProgress) {
      onDownloadProgress(-1)
    }
  }
  
  // Flush any remaining bytes from the decoder
  text += decoder.decode()

  // Report download complete
  if (onDownloadProgress) {
    onDownloadProgress(100)
  }

  // Phase 2: Parse JSON in Web Worker with progress simulation
  if (onParseProgress) {
    onParseProgress(0)
  }

  const data = await parseJSONInWorker(text, onParseProgress)

  // Report parsing complete
  if (onParseProgress) {
    onParseProgress(100)
  }

  return data
}

/**
 * Parses JSON in a Web Worker to avoid blocking the main thread
 * @param {string} jsonString - The JSON string to parse
 * @param {Function} onProgress - Progress callback for parsing
 * @returns {Promise<any>} The parsed data
 */
function parseJSONInWorker(jsonString, onProgress) {
  return new Promise((resolve, reject) => {
    const worker = new JsonParserWorker()
    // Generate unique request ID to match responses
    const requestId = `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`

    // Simulate progress during parsing (since we can't track actual JSON.parse progress)
    // Use a smoother logarithmic-style progression
    let progressInterval
    if (onProgress) {
      let simulatedProgress = 0
      let updateCount = 0
      // Constants for logarithmic progression calculation
      const INITIAL_SPEED = 8  // Slower initial progress for smoother feel
      const MIN_INCREMENT = 0.5  // Smaller minimum increment for smoother updates
      const MAX_PROGRESS = 90  // Cap at 90%, final 10% happens when parsing completes
      
      progressInterval = setInterval(() => {
        updateCount++
        // Logarithmic-style progression: gradual start, slow end
        // Formula: increment = max(MIN_INCREMENT, INITIAL_SPEED / sqrt(updateCount))
        // This provides smoother progression over longer time periods
        const increment = Math.max(MIN_INCREMENT, INITIAL_SPEED / Math.sqrt(updateCount))
        simulatedProgress = Math.min(simulatedProgress + increment, MAX_PROGRESS)
        onProgress(Math.round(simulatedProgress))
      }, 200) // Update every 200ms for smoother, less jumpy progression
    }

    worker.addEventListener('message', (event) => {
      const { type, requestId: responseId, data, error } = event.data

      if (responseId !== requestId) return

      // Clear progress simulation
      if (progressInterval) {
        clearInterval(progressInterval)
      }

      if (type === 'success') {
        worker.terminate()
        resolve(data)
      } else if (type === 'error') {
        worker.terminate()
        reject(new Error(error.message))
      }
    })

    worker.addEventListener('error', (error) => {
      if (progressInterval) {
        clearInterval(progressInterval)
      }
      worker.terminate()
      reject(new Error(`Worker error: ${error.message}`))
    })

    // Send the JSON string to the worker for parsing
    worker.postMessage({
      type: 'parse',
      data: jsonString,
      requestId
    })
  })
}
