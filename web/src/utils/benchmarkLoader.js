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

  // Get content length for progress calculation
  const contentLength = response.headers.get('Content-Length')
  const total = contentLength ? parseInt(contentLength, 10) : 0

  let loaded = 0
  const reader = response.body.getReader()
  const chunks = []

  // Read the response body with progress tracking
  while (true) {
    const { done, value } = await reader.read()
    
    if (done) break
    
    chunks.push(value)
    loaded += value.length
    
    // Report download progress
    if (onDownloadProgress && total > 0) {
      const progress = Math.round((loaded / total) * 100)
      onDownloadProgress(progress)
    } else if (onDownloadProgress) {
      // If content-length is missing, show indeterminate progress
      onDownloadProgress(-1)
    }
  }

  // Combine chunks into a single string
  const blob = new Blob(chunks)
  const text = await blob.text()

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
    const requestId = Math.random().toString(36).substring(7)

    // Simulate progress during parsing (since we can't track actual JSON.parse progress)
    let progressInterval
    if (onProgress) {
      let simulatedProgress = 0
      progressInterval = setInterval(() => {
        simulatedProgress += 10
        if (simulatedProgress < 90) {
          onProgress(simulatedProgress)
        }
      }, 50) // Update every 50ms
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
