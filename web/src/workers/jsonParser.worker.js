// Web Worker for parsing large JSON data in background thread
// This prevents the main UI thread from freezing during JSON parsing

self.addEventListener('message', (event) => {
  const { type, data, requestId } = event.data

  if (type === 'parse') {
    try {
      // Parse the JSON string in the background thread
      const parsed = JSON.parse(data)
      
      // Send the parsed result back to the main thread
      self.postMessage({
        type: 'success',
        requestId,
        data: parsed
      })
    } catch (error) {
      // Send error back to main thread
      self.postMessage({
        type: 'error',
        requestId,
        error: {
          message: error.message,
          name: error.name
        }
      })
    }
  }
})
