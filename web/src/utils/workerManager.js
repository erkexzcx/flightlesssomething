/**
 * Worker Manager - Manages Web Worker lifecycle and communication
 * Provides a Promise-based API for offloading calculations to a Web Worker
 */

/* global Worker */

class WorkerManager {
  constructor(workerUrl) {
    this.worker = null
    this.workerUrl = workerUrl
    this.taskId = 0
    this.pendingTasks = new Map()
  }

  /**
   * Initialize the worker
   */
  init() {
    if (this.worker) {
      return // Already initialized
    }

    try {
      this.worker = new Worker(this.workerUrl)
      
      // Handle messages from worker
      this.worker.addEventListener('message', (event) => {
        const { type, taskId, result, error } = event.data
        const task = this.pendingTasks.get(taskId)
        
        if (!task) {
          console.warn(`Received response for unknown task: ${taskId}`)
          return
        }
        
        this.pendingTasks.delete(taskId)
        
        if (type === 'success') {
          task.resolve(result)
        } else if (type === 'error') {
          task.reject(new Error(error))
        }
      })
      
      // Handle worker errors
      this.worker.addEventListener('error', (event) => {
        console.error('Worker error:', event)
        // Reject all pending tasks
        for (const [taskId, task] of this.pendingTasks.entries()) {
          task.reject(new Error(`Worker error: ${event.message}`))
        }
        this.pendingTasks.clear()
      })
    } catch (error) {
      console.error('Failed to initialize worker:', error)
      throw error
    }
  }

  /**
   * Send a task to the worker and return a Promise
   */
  async runTask(type, payload) {
    if (!this.worker) {
      this.init()
    }

    const taskId = ++this.taskId
    
    return new Promise((resolve, reject) => {
      this.pendingTasks.set(taskId, { resolve, reject })
      
      try {
        this.worker.postMessage({
          type,
          payload,
          taskId
        })
      } catch (error) {
        this.pendingTasks.delete(taskId)
        reject(error)
      }
    })
  }

  /**
   * Calculate FPS statistics
   */
  async calculateFPSStats(fpsDataArrays) {
    return this.runTask('calculateFPSStats', { fpsDataArrays })
  }

  /**
   * Calculate frametime statistics
   */
  async calculateFrametimeStats(frameTimeDataArrays) {
    return this.runTask('calculateFrametimeStats', { frameTimeDataArrays })
  }

  /**
   * Calculate summary statistics
   */
  async calculateSummaryStats(dataArrays) {
    return this.runTask('calculateSummaryStats', { dataArrays })
  }

  /**
   * Decimate line chart data
   */
  async decimateLineChartData(dataArrays, targetPoints = 2000) {
    return this.runTask('decimateLineChartData', { dataArrays, targetPoints })
  }

  /**
   * Calculate all statistics at once (most efficient for initial load)
   */
  async calculateAll(dataArrays, targetPoints = 2000) {
    return this.runTask('calculateAll', { dataArrays, targetPoints })
  }

  /**
   * Terminate the worker
   */
  terminate() {
    if (this.worker) {
      this.worker.terminate()
      this.worker = null
      
      // Reject all pending tasks
      for (const [taskId, task] of this.pendingTasks.entries()) {
        task.reject(new Error('Worker terminated'))
      }
      this.pendingTasks.clear()
    }
  }
}

// Create a singleton instance
let workerManager = null

/**
 * Get or create the worker manager instance
 */
export function getWorkerManager() {
  if (!workerManager) {
    // Use Vite's special syntax for importing workers
    // This allows Vite to properly bundle the worker
    workerManager = new WorkerManager(
      new URL('../workers/benchmarkCalculations.worker.js', import.meta.url)
    )
  }
  return workerManager
}

/**
 * Terminate the worker (cleanup)
 */
export function terminateWorker() {
  if (workerManager) {
    workerManager.terminate()
    workerManager = null
  }
}
