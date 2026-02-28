<template>
  <div class="container-fluid">
    <div class="row">
      <div class="col-lg-8 mx-auto">
        <h2 class="mb-4">Debug Calculator</h2>
        
        <!-- Input Section -->
        <div class="card mb-4">
          <div class="card-body">
            <h5 class="card-title">Input Data</h5>
            <p class="text-muted small">
              Paste your benchmark data below (tab-separated or space-separated). 
              First row should contain headers (fps, frametime). Subsequent rows should contain numeric values.
            </p>
            <textarea
              v-model="inputData"
              class="form-control font-monospace"
              rows="15"
              placeholder="fps	frametime
100	10.0
200	5.0
..."
            ></textarea>
            <div class="mt-3">
              <button class="btn btn-primary" @click="calculate">
                <i class="fa-solid fa-calculator"></i> Calculate
              </button>
              <button class="btn btn-secondary ms-2" @click="resetToExample">
                <i class="fa-solid fa-rotate-left"></i> Reset to Example
              </button>
            </div>
          </div>
        </div>

        <!-- Error Display -->
        <div v-if="error" class="alert alert-danger" role="alert">
          <i class="fa-solid fa-exclamation-triangle"></i> {{ error }}
        </div>

        <!-- Results Section -->
        <div v-if="results" class="card mb-4">
          <div class="card-body">
            <h5 class="card-title">Results (Client-Side Calculation)</h5>
            
            <!-- FPS Statistics -->
            <div class="mb-4">
              <h6 class="text-primary">FPS Statistics</h6>
              <div class="row">
                <div class="col-md-6">
                  <h6 class="text-muted small mt-3">Linear Interpolation Method</h6>
                  <table class="table table-sm table-bordered">
                    <tbody>
                      <tr v-for="field in allStatFields" :key="'fps-lin-' + field.key">
                        <th>{{ field.label }}</th>
                        <td>{{ formatNumber(results.fps.linear[field.key]) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <div class="col-md-6">
                  <h6 class="text-muted small mt-3">Mangohud</h6>
                  <table class="table table-sm table-bordered">
                    <tbody>
                      <tr v-for="field in allStatFields" :key="'fps-mh-' + field.key">
                        <th>{{ field.label }}</th>
                        <td>{{ formatNumber(results.fps.mangohud[field.key]) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </div>

            <!-- Frametime Statistics -->
            <div class="mb-4">
              <h6 class="text-success">Frametime Statistics</h6>
              <div class="row">
                <div class="col-md-6">
                  <h6 class="text-muted small mt-3">Linear Interpolation Method</h6>
                  <table class="table table-sm table-bordered">
                    <tbody>
                      <tr v-for="field in allStatFields" :key="'ft-lin-' + field.key">
                        <th>{{ field.label }}</th>
                        <td>{{ formatNumber(results.frametime.linear[field.key]) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <div class="col-md-6">
                  <h6 class="text-muted small mt-3">Mangohud</h6>
                  <table class="table table-sm table-bordered">
                    <tbody>
                      <tr v-for="field in allStatFields" :key="'ft-mh-' + field.key">
                        <th>{{ field.label }}</th>
                        <td>{{ formatNumber(results.frametime.mangohud[field.key]) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Backend Verification Section -->
        <div v-if="results" class="card mb-4">
          <div class="card-body">
            <h5 class="card-title">Backend Verification</h5>
            <p class="text-muted small">
              <i class="fa-solid fa-info-circle"></i>
              Compare client-side calculations with backend pre-computed values.
            </p>
            <button class="btn btn-outline-info" @click="verifyWithBackend" :disabled="backendLoading">
              <span v-if="backendLoading">
                <span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span>
                Verifying...
              </span>
              <span v-else>
                <i class="fa-solid fa-check-double"></i> Verify with Backend
              </span>
            </button>
            <div v-if="backendError" class="alert alert-danger mt-3">
              <i class="fa-solid fa-exclamation-triangle"></i> {{ backendError }}
            </div>
            <div v-if="backendResults" class="mt-3">
              <table class="table table-sm table-bordered">
                <thead>
                  <tr>
                    <th>Metric</th>
                    <th>Client (Linear)</th>
                    <th>Backend (Linear)</th>
                    <th>Match</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="field in allStatFieldKeys" :key="'fps-' + field">
                    <td>FPS {{ field }}</td>
                    <td>{{ formatNumber(results.fps.linear[field]) }}</td>
                    <td>{{ formatNumber(backendResults.linear.fps?.[field]) }}</td>
                    <td>
                      <span :class="matchClass(results.fps.linear[field], backendResults.linear.fps?.[field])">
                        {{ matchLabel(results.fps.linear[field], backendResults.linear.fps?.[field]) }}
                      </span>
                    </td>
                  </tr>
                  <tr v-for="field in allStatFieldKeys" :key="'ft-' + field">
                    <td>Frametime {{ field }}</td>
                    <td>{{ formatNumber(results.frametime.linear[field]) }}</td>
                    <td>{{ formatNumber(backendResults.linear.frameTime?.[field]) }}</td>
                    <td>
                      <span :class="matchClass(results.frametime.linear[field], backendResults.linear.frameTime?.[field])">
                        {{ matchLabel(results.frametime.linear[field], backendResults.linear.frameTime?.[field]) }}
                      </span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <!-- Spreadsheet Export Section -->
        <div v-if="results" class="card mb-4">
          <div class="card-body">
            <h5 class="card-title">Spreadsheet Export for Verification</h5>
            <p class="text-muted small">
              <i class="fa-solid fa-info-circle"></i> 
              <strong>LibreOffice Calc / Excel compatible export.</strong>
              Copy the data below and paste it into LibreOffice Calc or Excel.
              The export includes raw data, FlightlessSomething's calculated values, and spreadsheet formulas.
              Compare the "FlightlessSomething" values with the "Formula Result" values in your spreadsheet to verify accuracy.
            </p>
            <textarea
              v-model="spreadsheetDataLibreOffice"
              class="form-control font-monospace"
              rows="20"
              readonly
            ></textarea>
            <div class="mt-2">
              <button class="btn btn-sm btn-outline-primary" @click="copyToClipboardLibreOffice">
                <i class="fa-solid fa-copy"></i> Copy to Clipboard
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { 
  calculateStats, 
  calculateFPSStatsFromFrametime 
} from '../utils/statsCalculations'

// All stat fields matching backend MetricStats, in display order
const allStatFields = [
  { key: 'min', label: 'Min' },
  { key: 'max', label: 'Max' },
  { key: 'avg', label: 'Average' },
  { key: 'median', label: 'Median' },
  { key: 'p01', label: '1st Percentile' },
  { key: 'p05', label: '5th Percentile' },
  { key: 'p10', label: '10th Percentile' },
  { key: 'p25', label: '25th Percentile' },
  { key: 'p75', label: '75th Percentile' },
  { key: 'p90', label: '90th Percentile' },
  { key: 'p95', label: '95th Percentile' },
  { key: 'p97', label: '97th Percentile' },
  { key: 'p99', label: '99th Percentile' },
  { key: 'iqr', label: 'IQR (P75-P25)' },
  { key: 'stddev', label: 'Standard Deviation' },
  { key: 'variance', label: 'Variance' },
  { key: 'count', label: 'Count' },
]
const allStatFieldKeys = allStatFields.map(f => f.key)

const EXAMPLE_DATA = `fps	frametime
383.357	2.60854
426.733	2.34338
358.585	2.78874
415.468	2.40692
398.055	2.51222
256.292	3.9018
660.507	1.51399
463.909	2.1556
364.767	2.74148
485.929	2.05791
367.709	2.71954
393.879	2.53885
394.858	2.53255
428.923	2.33142
380.448	2.62848
408.238	2.44955
310.068	3.2251
667.648	1.49779
342.687	2.91812
480.15	2.08268
379.853	2.6326
382.622	2.61355
408.703	2.44676
415.977	2.40398
399.626	2.50234
403.382	2.47904
406.354	2.46091
342.426	2.92034
543.757	1.83906
450.025	2.2221
373.265	2.67906
436.834	2.2892
378.937	2.63896
420.753	2.37669
333.582	2.99776
423.51	2.36122
388.765	2.57225
236.697	4.22482
630.597	1.5858
690.239	1.44877
375.115	2.66585
457.41	2.18622
376.539	2.65577
375.046	2.66634
359.014	2.78541
477.112	2.09594
307.294	3.25422
342.563	2.91917
345.223	2.89668
479.734	2.08449`

const inputData = ref(EXAMPLE_DATA)
const results = ref(null)
const error = ref(null)
const parsedData = ref(null)
const backendResults = ref(null)
const backendLoading = ref(false)
const backendError = ref(null)

function resetToExample() {
  inputData.value = EXAMPLE_DATA
  results.value = null
  error.value = null
  parsedData.value = null
  backendResults.value = null
  backendError.value = null
}

function parseInput(input) {
  const lines = input.trim().split('\n')
  if (lines.length < 2) {
    throw new Error('Input must contain at least a header row and one data row')
  }

  // Parse header
  const header = lines[0].split(/\s+/)
  const fpsIndex = header.findIndex(h => h.toLowerCase() === 'fps')
  const frametimeIndex = header.findIndex(h => h.toLowerCase() === 'frametime')

  if (fpsIndex === -1 && frametimeIndex === -1) {
    throw new Error('Header must contain "fps" and/or "frametime" columns')
  }

  const fpsValues = []
  const frametimeValues = []

  // Parse data rows
  for (let i = 1; i < lines.length; i++) {
    const line = lines[i].trim()
    if (!line) continue // Skip empty lines

    const values = line.split(/\s+/)
    
    if (fpsIndex !== -1 && values[fpsIndex]) {
      const fps = parseFloat(values[fpsIndex])
      if (!isNaN(fps)) {
        fpsValues.push(fps)
      }
    }
    
    if (frametimeIndex !== -1 && values[frametimeIndex]) {
      const frametime = parseFloat(values[frametimeIndex])
      if (!isNaN(frametime)) {
        frametimeValues.push(frametime)
      }
    }
  }

  if (fpsValues.length === 0 && frametimeValues.length === 0) {
    throw new Error('No valid numeric data found')
  }

  return { fpsValues, frametimeValues }
}

function calculate() {
  try {
    error.value = null
    results.value = null

    const data = parseInput(inputData.value)
    parsedData.value = data

    // Calculate FPS statistics
    const fpsLinear = data.frametimeValues.length > 0
      ? calculateFPSStatsFromFrametime(data.frametimeValues, 'linear-interpolation')
      : calculateStats(data.fpsValues, 'linear-interpolation')

    const fpsMangoHud = data.frametimeValues.length > 0
      ? calculateFPSStatsFromFrametime(data.frametimeValues, 'mangohud-threshold')
      : calculateStats(data.fpsValues, 'mangohud-threshold')

    // Calculate Frametime statistics
    const frametimeLinear = calculateStats(data.frametimeValues, 'linear-interpolation')
    const frametimeMangoHud = calculateStats(data.frametimeValues, 'mangohud-threshold')

    results.value = {
      fps: {
        linear: fpsLinear,
        mangohud: fpsMangoHud
      },
      frametime: {
        linear: frametimeLinear,
        mangohud: frametimeMangoHud
      }
    }
  } catch (err) {
    error.value = err.message
  }
}

async function verifyWithBackend() {
  if (!parsedData.value) return
  backendLoading.value = true
  backendError.value = null
  backendResults.value = null
  try {
    const response = await fetch('/api/debugcalc', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        fps: parsedData.value.fpsValues,
        frameTime: parsedData.value.frametimeValues
      })
    })
    if (!response.ok) {
      const data = await response.json().catch(() => ({}))
      throw new Error(data.error || 'Backend verification failed')
    }
    backendResults.value = await response.json()
  } catch (err) {
    backendError.value = err.message
  } finally {
    backendLoading.value = false
  }
}

function matchClass(clientVal, backendVal) {
  if (clientVal === null || clientVal === undefined || backendVal === null || backendVal === undefined) return 'text-muted'
  return Math.abs(clientVal - backendVal) <= 0.1 ? 'text-success' : 'text-danger'
}

function matchLabel(clientVal, backendVal) {
  if (clientVal === null || clientVal === undefined || backendVal === null || backendVal === undefined) return 'N/A'
  return Math.abs(clientVal - backendVal) <= 0.1 ? '✓ Match' : '✗ Mismatch'
}

const spreadsheetData = computed(() => {
  if (!results.value || !parsedData.value) return ''

  const lines = []
  
  // Add header with context
  lines.push('NOTE: This export is for verification purposes only.')
  lines.push('Compare the "FlightlessSomething" column (calculated by this app) with the "Formula Result" column (calculated by your spreadsheet).')
  lines.push('Both values should match to verify the calculations are correct.')
  lines.push('')
  
  // Add raw data
  // Calculate row numbers (accounting for header lines)
  const headerLines = 4 // NOTE lines + blank line
  const dataStartRow = headerLines + 2 // After header lines + data header row
  const maxLength = Math.max(parsedData.value.fpsValues.length, parsedData.value.frametimeValues.length)
  const hasFT = parsedData.value.frametimeValues.length > 0
  const fpsStartRow = dataStartRow
  const fpsEndRow = fpsStartRow + parsedData.value.fpsValues.length - 1
  const ftStartRow = dataStartRow
  const ftEndRow = ftStartRow + parsedData.value.frametimeValues.length - 1
  
  // When FPS is derived from frametime, add a helper column C with =1000/B{row}
  if (hasFT) {
    lines.push('fps,frametime,fps_from_ft')
  } else {
    lines.push('fps,frametime')
  }
  for (let i = 0; i < maxLength; i++) {
    const fps = i < parsedData.value.fpsValues.length ? parsedData.value.fpsValues[i] : ''
    const frametime = i < parsedData.value.frametimeValues.length ? parsedData.value.frametimeValues[i] : ''
    if (hasFT) {
      lines.push(`${fps},${frametime},=1000/B${dataStartRow + i}`)
    } else {
      lines.push(`${fps},${frametime}`)
    }
  }
  
  // Add blank line
  lines.push('')
  
  // Helper: MangoHud INDEX formula with MIN/MAX clamping (comma separator for CSV/Excel, no leading =)
  // Uses INT() instead of FLOOR() because FLOOR errors on negative values in LibreOffice
  const mangoHudFormulaCSV = (range, pDecimal) => {
    return `INDEX(SORT(${range}),MIN(MAX(COUNT(${range})-INT(${pDecimal}*COUNT(${range})-1),1),COUNT(${range})))`
  }
  
  // Add FPS statistics - Linear Interpolation
  lines.push('FPS Statistics - Linear Interpolation')
  lines.push('Metric,FlightlessSomething,Formula,Formula Result')
  
  if (hasFT) {
    const ftR = `B${ftStartRow}:B${ftEndRow}`
    const fpsFromFtR = `C${ftStartRow}:C${ftEndRow}`
    lines.push(`Min,${formatNumber(results.value.fps.linear.min)},=1000/MAX(${ftR}),`)
    lines.push(`Max,${formatNumber(results.value.fps.linear.max)},=1000/MIN(${ftR}),`)
    lines.push(`Average,${formatNumber(results.value.fps.linear.avg)},=1000/AVERAGE(${ftR}),`)
    lines.push(`Median,${formatNumber(results.value.fps.linear.median)},=MEDIAN(${fpsFromFtR}),`)
    lines.push(`1st Percentile,${formatNumber(results.value.fps.linear.p01)},=1000/PERCENTILE.INC(${ftR},0.99),`)
    lines.push(`5th Percentile,${formatNumber(results.value.fps.linear.p05)},=1000/PERCENTILE.INC(${ftR},0.95),`)
    lines.push(`10th Percentile,${formatNumber(results.value.fps.linear.p10)},=1000/PERCENTILE.INC(${ftR},0.90),`)
    lines.push(`25th Percentile,${formatNumber(results.value.fps.linear.p25)},=1000/PERCENTILE.INC(${ftR},0.75),`)
    lines.push(`75th Percentile,${formatNumber(results.value.fps.linear.p75)},=1000/PERCENTILE.INC(${ftR},0.25),`)
    lines.push(`90th Percentile,${formatNumber(results.value.fps.linear.p90)},=1000/PERCENTILE.INC(${ftR},0.10),`)
    lines.push(`95th Percentile,${formatNumber(results.value.fps.linear.p95)},=1000/PERCENTILE.INC(${ftR},0.05),`)
    lines.push(`97th Percentile,${formatNumber(results.value.fps.linear.p97)},=1000/PERCENTILE.INC(${ftR},0.03),`)
    lines.push(`99th Percentile,${formatNumber(results.value.fps.linear.p99)},=1000/PERCENTILE.INC(${ftR},0.01),`)
    lines.push(`IQR (P75-P25),${formatNumber(results.value.fps.linear.iqr)},=1000/PERCENTILE.INC(${ftR},0.25)-1000/PERCENTILE.INC(${ftR},0.75),`)
    lines.push(`Standard Deviation,${formatNumber(results.value.fps.linear.stddev)},=STDEV(${fpsFromFtR}),`)
    lines.push(`Variance,${formatNumber(results.value.fps.linear.variance)},=VAR(${fpsFromFtR}),`)
    lines.push(`Count,${formatNumber(results.value.fps.linear.count)},=COUNT(${ftR}),`)
  } else {
    const fpsR = `A${fpsStartRow}:A${fpsEndRow}`
    lines.push(`Min,${formatNumber(results.value.fps.linear.min)},=MIN(${fpsR}),`)
    lines.push(`Max,${formatNumber(results.value.fps.linear.max)},=MAX(${fpsR}),`)
    lines.push(`Average,${formatNumber(results.value.fps.linear.avg)},=AVERAGE(${fpsR}),`)
    lines.push(`Median,${formatNumber(results.value.fps.linear.median)},=MEDIAN(${fpsR}),`)
    lines.push(`1st Percentile,${formatNumber(results.value.fps.linear.p01)},=PERCENTILE.INC(${fpsR},0.01),`)
    lines.push(`5th Percentile,${formatNumber(results.value.fps.linear.p05)},=PERCENTILE.INC(${fpsR},0.05),`)
    lines.push(`10th Percentile,${formatNumber(results.value.fps.linear.p10)},=PERCENTILE.INC(${fpsR},0.10),`)
    lines.push(`25th Percentile,${formatNumber(results.value.fps.linear.p25)},=PERCENTILE.INC(${fpsR},0.25),`)
    lines.push(`75th Percentile,${formatNumber(results.value.fps.linear.p75)},=PERCENTILE.INC(${fpsR},0.75),`)
    lines.push(`90th Percentile,${formatNumber(results.value.fps.linear.p90)},=PERCENTILE.INC(${fpsR},0.90),`)
    lines.push(`95th Percentile,${formatNumber(results.value.fps.linear.p95)},=PERCENTILE.INC(${fpsR},0.95),`)
    lines.push(`97th Percentile,${formatNumber(results.value.fps.linear.p97)},=PERCENTILE.INC(${fpsR},0.97),`)
    lines.push(`99th Percentile,${formatNumber(results.value.fps.linear.p99)},=PERCENTILE.INC(${fpsR},0.99),`)
    lines.push(`IQR (P75-P25),${formatNumber(results.value.fps.linear.iqr)},=PERCENTILE.INC(${fpsR},0.75)-PERCENTILE.INC(${fpsR},0.25),`)
    lines.push(`Standard Deviation,${formatNumber(results.value.fps.linear.stddev)},=STDEV(${fpsR}),`)
    lines.push(`Variance,${formatNumber(results.value.fps.linear.variance)},=VAR(${fpsR}),`)
    lines.push(`Count,${formatNumber(results.value.fps.linear.count)},=COUNT(${fpsR}),`)
  }
  
  lines.push('')
  
  // Add FPS statistics - Mangohud
  lines.push('FPS Statistics - Mangohud')
  lines.push('Metric,FlightlessSomething,Formula,Formula Result')
  
  if (hasFT) {
    const ftR = `B${ftStartRow}:B${ftEndRow}`
    const fpsFromFtR = `C${ftStartRow}:C${ftEndRow}`
    lines.push(`Min,${formatNumber(results.value.fps.mangohud.min)},=1000/MAX(${ftR}),`)
    lines.push(`Max,${formatNumber(results.value.fps.mangohud.max)},=1000/MIN(${ftR}),`)
    lines.push(`Average,${formatNumber(results.value.fps.mangohud.avg)},=1000/AVERAGE(${ftR}),`)
    lines.push(`Median,${formatNumber(results.value.fps.mangohud.median)},=${mangoHudFormulaCSV(fpsFromFtR, 0.50)},`)
    lines.push(`1st Percentile,${formatNumber(results.value.fps.mangohud.p01)},=1000/${mangoHudFormulaCSV(ftR, 0.01)},`)
    lines.push(`5th Percentile,${formatNumber(results.value.fps.mangohud.p05)},=1000/${mangoHudFormulaCSV(ftR, 0.05)},`)
    lines.push(`10th Percentile,${formatNumber(results.value.fps.mangohud.p10)},=1000/${mangoHudFormulaCSV(ftR, 0.10)},`)
    lines.push(`25th Percentile,${formatNumber(results.value.fps.mangohud.p25)},=1000/${mangoHudFormulaCSV(ftR, 0.25)},`)
    lines.push(`75th Percentile,${formatNumber(results.value.fps.mangohud.p75)},=1000/${mangoHudFormulaCSV(ftR, 0.75)},`)
    lines.push(`90th Percentile,${formatNumber(results.value.fps.mangohud.p90)},=1000/${mangoHudFormulaCSV(ftR, 0.90)},`)
    lines.push(`95th Percentile,${formatNumber(results.value.fps.mangohud.p95)},=1000/${mangoHudFormulaCSV(ftR, 0.95)},`)
    lines.push(`97th Percentile,${formatNumber(results.value.fps.mangohud.p97)},=1000/${mangoHudFormulaCSV(ftR, 0.97)},`)
    lines.push(`99th Percentile,${formatNumber(results.value.fps.mangohud.p99)},=1000/${mangoHudFormulaCSV(ftR, 0.99)},`)
    lines.push(`IQR (P75-P25),${formatNumber(results.value.fps.mangohud.iqr)},=1000/${mangoHudFormulaCSV(ftR, 0.75)}-1000/${mangoHudFormulaCSV(ftR, 0.25)},`)
    lines.push(`Standard Deviation,${formatNumber(results.value.fps.mangohud.stddev)},=STDEV(${fpsFromFtR}),`)
    lines.push(`Variance,${formatNumber(results.value.fps.mangohud.variance)},=VAR(${fpsFromFtR}),`)
    lines.push(`Count,${formatNumber(results.value.fps.mangohud.count)},=COUNT(${ftR}),`)
  } else {
    const fpsR = `A${fpsStartRow}:A${fpsEndRow}`
    lines.push(`Min,${formatNumber(results.value.fps.mangohud.min)},=MIN(${fpsR}),`)
    lines.push(`Max,${formatNumber(results.value.fps.mangohud.max)},=MAX(${fpsR}),`)
    lines.push(`Average,${formatNumber(results.value.fps.mangohud.avg)},=AVERAGE(${fpsR}),`)
    lines.push(`Median,${formatNumber(results.value.fps.mangohud.median)},=${mangoHudFormulaCSV(fpsR, 0.50)},`)
    lines.push(`1st Percentile,${formatNumber(results.value.fps.mangohud.p01)},=${mangoHudFormulaCSV(fpsR, 0.99)},`)
    lines.push(`5th Percentile,${formatNumber(results.value.fps.mangohud.p05)},=${mangoHudFormulaCSV(fpsR, 0.95)},`)
    lines.push(`10th Percentile,${formatNumber(results.value.fps.mangohud.p10)},=${mangoHudFormulaCSV(fpsR, 0.90)},`)
    lines.push(`25th Percentile,${formatNumber(results.value.fps.mangohud.p25)},=${mangoHudFormulaCSV(fpsR, 0.75)},`)
    lines.push(`75th Percentile,${formatNumber(results.value.fps.mangohud.p75)},=${mangoHudFormulaCSV(fpsR, 0.25)},`)
    lines.push(`90th Percentile,${formatNumber(results.value.fps.mangohud.p90)},=${mangoHudFormulaCSV(fpsR, 0.10)},`)
    lines.push(`95th Percentile,${formatNumber(results.value.fps.mangohud.p95)},=${mangoHudFormulaCSV(fpsR, 0.05)},`)
    lines.push(`97th Percentile,${formatNumber(results.value.fps.mangohud.p97)},=${mangoHudFormulaCSV(fpsR, 0.03)},`)
    lines.push(`99th Percentile,${formatNumber(results.value.fps.mangohud.p99)},=${mangoHudFormulaCSV(fpsR, 0.01)},`)
    lines.push(`IQR (P75-P25),${formatNumber(results.value.fps.mangohud.iqr)},=${mangoHudFormulaCSV(fpsR, 0.25)}-${mangoHudFormulaCSV(fpsR, 0.75)},`)
    lines.push(`Standard Deviation,${formatNumber(results.value.fps.mangohud.stddev)},=STDEV(${fpsR}),`)
    lines.push(`Variance,${formatNumber(results.value.fps.mangohud.variance)},=VAR(${fpsR}),`)
    lines.push(`Count,${formatNumber(results.value.fps.mangohud.count)},=COUNT(${fpsR}),`)
  }
  
  lines.push('')
  
  // Add Frametime statistics - Linear Interpolation
  lines.push('Frametime Statistics - Linear Interpolation')
  lines.push('Metric,FlightlessSomething,Formula,Formula Result')
  {
    const ftR = `B${ftStartRow}:B${ftEndRow}`
    lines.push(`Min,${formatNumber(results.value.frametime.linear.min)},=MIN(${ftR}),`)
    lines.push(`Max,${formatNumber(results.value.frametime.linear.max)},=MAX(${ftR}),`)
    lines.push(`Average,${formatNumber(results.value.frametime.linear.avg)},=AVERAGE(${ftR}),`)
    lines.push(`Median,${formatNumber(results.value.frametime.linear.median)},=MEDIAN(${ftR}),`)
    lines.push(`1st Percentile,${formatNumber(results.value.frametime.linear.p01)},=PERCENTILE.INC(${ftR},0.01),`)
    lines.push(`5th Percentile,${formatNumber(results.value.frametime.linear.p05)},=PERCENTILE.INC(${ftR},0.05),`)
    lines.push(`10th Percentile,${formatNumber(results.value.frametime.linear.p10)},=PERCENTILE.INC(${ftR},0.10),`)
    lines.push(`25th Percentile,${formatNumber(results.value.frametime.linear.p25)},=PERCENTILE.INC(${ftR},0.25),`)
    lines.push(`75th Percentile,${formatNumber(results.value.frametime.linear.p75)},=PERCENTILE.INC(${ftR},0.75),`)
    lines.push(`90th Percentile,${formatNumber(results.value.frametime.linear.p90)},=PERCENTILE.INC(${ftR},0.90),`)
    lines.push(`95th Percentile,${formatNumber(results.value.frametime.linear.p95)},=PERCENTILE.INC(${ftR},0.95),`)
    lines.push(`97th Percentile,${formatNumber(results.value.frametime.linear.p97)},=PERCENTILE.INC(${ftR},0.97),`)
    lines.push(`99th Percentile,${formatNumber(results.value.frametime.linear.p99)},=PERCENTILE.INC(${ftR},0.99),`)
    lines.push(`IQR (P75-P25),${formatNumber(results.value.frametime.linear.iqr)},=PERCENTILE.INC(${ftR},0.75)-PERCENTILE.INC(${ftR},0.25),`)
    lines.push(`Standard Deviation,${formatNumber(results.value.frametime.linear.stddev)},=STDEV(${ftR}),`)
    lines.push(`Variance,${formatNumber(results.value.frametime.linear.variance)},=VAR(${ftR}),`)
    lines.push(`Count,${formatNumber(results.value.frametime.linear.count)},=COUNT(${ftR}),`)
  }
  
  lines.push('')
  
  // Add Frametime statistics - Mangohud
  lines.push('Frametime Statistics - Mangohud')
  lines.push('Metric,FlightlessSomething,Formula,Formula Result')
  {
    const ftR = `B${ftStartRow}:B${ftEndRow}`
    lines.push(`Min,${formatNumber(results.value.frametime.mangohud.min)},=MIN(${ftR}),`)
    lines.push(`Max,${formatNumber(results.value.frametime.mangohud.max)},=MAX(${ftR}),`)
    lines.push(`Average,${formatNumber(results.value.frametime.mangohud.avg)},=AVERAGE(${ftR}),`)
    lines.push(`Median,${formatNumber(results.value.frametime.mangohud.median)},=${mangoHudFormulaCSV(ftR, 0.50)},`)
    lines.push(`1st Percentile,${formatNumber(results.value.frametime.mangohud.p01)},=${mangoHudFormulaCSV(ftR, 0.99)},`)
    lines.push(`5th Percentile,${formatNumber(results.value.frametime.mangohud.p05)},=${mangoHudFormulaCSV(ftR, 0.95)},`)
    lines.push(`10th Percentile,${formatNumber(results.value.frametime.mangohud.p10)},=${mangoHudFormulaCSV(ftR, 0.90)},`)
    lines.push(`25th Percentile,${formatNumber(results.value.frametime.mangohud.p25)},=${mangoHudFormulaCSV(ftR, 0.75)},`)
    lines.push(`75th Percentile,${formatNumber(results.value.frametime.mangohud.p75)},=${mangoHudFormulaCSV(ftR, 0.25)},`)
    lines.push(`90th Percentile,${formatNumber(results.value.frametime.mangohud.p90)},=${mangoHudFormulaCSV(ftR, 0.10)},`)
    lines.push(`95th Percentile,${formatNumber(results.value.frametime.mangohud.p95)},=${mangoHudFormulaCSV(ftR, 0.05)},`)
    lines.push(`97th Percentile,${formatNumber(results.value.frametime.mangohud.p97)},=${mangoHudFormulaCSV(ftR, 0.03)},`)
    lines.push(`99th Percentile,${formatNumber(results.value.frametime.mangohud.p99)},=${mangoHudFormulaCSV(ftR, 0.01)},`)
    lines.push(`IQR (P75-P25),${formatNumber(results.value.frametime.mangohud.iqr)},=${mangoHudFormulaCSV(ftR, 0.25)}-${mangoHudFormulaCSV(ftR, 0.75)},`)
    lines.push(`Standard Deviation,${formatNumber(results.value.frametime.mangohud.stddev)},=STDEV(${ftR}),`)
    lines.push(`Variance,${formatNumber(results.value.frametime.mangohud.variance)},=VAR(${ftR}),`)
    lines.push(`Count,${formatNumber(results.value.frametime.mangohud.count)},=COUNT(${ftR}),`)
  }
  
  return lines.join('\n')
})

// LibreOffice Calc / Excel compatible export
const spreadsheetDataLibreOffice = computed(() => {
  if (!results.value || !parsedData.value) return ''

  const lines = []
  
  // Add raw data (no header text, start directly with column headers)
  const maxLength = Math.max(parsedData.value.fpsValues.length, parsedData.value.frametimeValues.length)
  const dataStartRow = 2 // After column headers
  const hasFT = parsedData.value.frametimeValues.length > 0
  
  // When FPS is derived from frametime, add a helper column C with =1000/B{row}
  // so that Median/StdDev/Variance formulas can reference actual FPS values
  if (hasFT) {
    lines.push('fps\tframetime\tfps_from_ft')
  } else {
    lines.push('fps\tframetime')
  }
  
  for (let i = 0; i < maxLength; i++) {
    const fps = i < parsedData.value.fpsValues.length ? parsedData.value.fpsValues[i] : ''
    const frametime = i < parsedData.value.frametimeValues.length ? parsedData.value.frametimeValues[i] : ''
    if (hasFT) {
      lines.push(`${fps}\t${frametime}\t=1000/B${dataStartRow + i}`)
    } else {
      lines.push(`${fps}\t${frametime}`)
    }
  }
  
  lines.push('')
  
  const fpsStartRow = dataStartRow
  const fpsEndRow = dataStartRow + parsedData.value.fpsValues.length - 1
  const ftStartRow = dataStartRow
  const ftEndRow = dataStartRow + parsedData.value.frametimeValues.length - 1
  
  // Helper: LibreOffice PERCENTILE formula (semicolon separator)
  const linearPercentileFormula = (range, p) => `=PERCENTILE(${range};${p})`
  // Helper: MangoHud INDEX formula with MIN/MAX clamping (returns without leading =)
  // MangoHud: idx = floor(val * n - 1) on descending, ascending: idx = n - 1 - floor((1-p/100)*n - 1)
  // Excel 1-based row = n - floor((1-p/100)*n - 1), clamped to [1, n]
  // Uses INT() instead of FLOOR(;1) because FLOOR errors on negative values in LibreOffice
  const mangoHudFormula = (range, pDecimal) => {
    const r = range
    return `INDEX(SORT(${r});MIN(MAX(COUNT(${r})-INT(${pDecimal}*COUNT(${r})-1);1);COUNT(${r})))`
  }
  
  // Helper to add a row and advance currentRow
  const addRow = (label, value, formula, ref) => {
    ref.row++
    lines.push(`${label}\t${formatNumber(value)}\t${formula}\t=IF(ABS(B${ref.row}-C${ref.row})<=0.1;"TRUE";"FALSE")`)
  }
  
  // Helper to add a section separator with title and advance currentRow
  const addSectionHeader = (title, ref) => {
    lines.push('')
    lines.push(title)
    lines.push('Metric\tFlightlessSomething\tSpreadsheet\tMatch')
    ref.row += 3
  }
  
  const ref = { row: fpsEndRow + 1 } // will be incremented before use
  
  // ===== FPS Statistics - Linear Interpolation =====
  addSectionHeader('FPS Statistics - Linear Interpolation', ref)
  
  if (hasFT) {
    const ftR = `B${ftStartRow}:B${ftEndRow}`
    const fpsFromFtR = `C${ftStartRow}:C${ftEndRow}`
    addRow('Min', results.value.fps.linear.min, `=1000/MAX(${ftR})`, ref)
    addRow('Max', results.value.fps.linear.max, `=1000/MIN(${ftR})`, ref)
    addRow('Average', results.value.fps.linear.avg, `=1000/AVERAGE(${ftR})`, ref)
    addRow('Median', results.value.fps.linear.median, `=MEDIAN(${fpsFromFtR})`, ref)
    addRow('1st Percentile', results.value.fps.linear.p01, `=1000/PERCENTILE(${ftR};0.99)`, ref)
    addRow('5th Percentile', results.value.fps.linear.p05, `=1000/PERCENTILE(${ftR};0.95)`, ref)
    addRow('10th Percentile', results.value.fps.linear.p10, `=1000/PERCENTILE(${ftR};0.90)`, ref)
    addRow('25th Percentile', results.value.fps.linear.p25, `=1000/PERCENTILE(${ftR};0.75)`, ref)
    addRow('75th Percentile', results.value.fps.linear.p75, `=1000/PERCENTILE(${ftR};0.25)`, ref)
    addRow('90th Percentile', results.value.fps.linear.p90, `=1000/PERCENTILE(${ftR};0.10)`, ref)
    addRow('95th Percentile', results.value.fps.linear.p95, `=1000/PERCENTILE(${ftR};0.05)`, ref)
    addRow('97th Percentile', results.value.fps.linear.p97, `=1000/PERCENTILE(${ftR};0.03)`, ref)
    addRow('99th Percentile', results.value.fps.linear.p99, `=1000/PERCENTILE(${ftR};0.01)`, ref)
    addRow('IQR (P75-P25)', results.value.fps.linear.iqr, `=1000/PERCENTILE(${ftR};0.25)-1000/PERCENTILE(${ftR};0.75)`, ref)
    addRow('Standard Deviation', results.value.fps.linear.stddev, `=STDEV(${fpsFromFtR})`, ref)
    addRow('Variance', results.value.fps.linear.variance, `=VAR(${fpsFromFtR})`, ref)
    addRow('Count', results.value.fps.linear.count, `=COUNT(${ftR})`, ref)
  } else {
    const fpsR = `A${fpsStartRow}:A${fpsEndRow}`
    addRow('Min', results.value.fps.linear.min, `=MIN(${fpsR})`, ref)
    addRow('Max', results.value.fps.linear.max, `=MAX(${fpsR})`, ref)
    addRow('Average', results.value.fps.linear.avg, `=AVERAGE(${fpsR})`, ref)
    addRow('Median', results.value.fps.linear.median, `=MEDIAN(${fpsR})`, ref)
    addRow('1st Percentile', results.value.fps.linear.p01, linearPercentileFormula(fpsR, 0.01), ref)
    addRow('5th Percentile', results.value.fps.linear.p05, linearPercentileFormula(fpsR, 0.05), ref)
    addRow('10th Percentile', results.value.fps.linear.p10, linearPercentileFormula(fpsR, 0.10), ref)
    addRow('25th Percentile', results.value.fps.linear.p25, linearPercentileFormula(fpsR, 0.25), ref)
    addRow('75th Percentile', results.value.fps.linear.p75, linearPercentileFormula(fpsR, 0.75), ref)
    addRow('90th Percentile', results.value.fps.linear.p90, linearPercentileFormula(fpsR, 0.90), ref)
    addRow('95th Percentile', results.value.fps.linear.p95, linearPercentileFormula(fpsR, 0.95), ref)
    addRow('97th Percentile', results.value.fps.linear.p97, linearPercentileFormula(fpsR, 0.97), ref)
    addRow('99th Percentile', results.value.fps.linear.p99, linearPercentileFormula(fpsR, 0.99), ref)
    addRow('IQR (P75-P25)', results.value.fps.linear.iqr, `=PERCENTILE(${fpsR};0.75)-PERCENTILE(${fpsR};0.25)`, ref)
    addRow('Standard Deviation', results.value.fps.linear.stddev, `=STDEV(${fpsR})`, ref)
    addRow('Variance', results.value.fps.linear.variance, `=VAR(${fpsR})`, ref)
    addRow('Count', results.value.fps.linear.count, `=COUNT(${fpsR})`, ref)
  }
  
  // ===== FPS Statistics - Mangohud =====
  addSectionHeader('FPS Statistics - Mangohud', ref)
  
  if (hasFT) {
    const ftR = `B${ftStartRow}:B${ftEndRow}`
    const fpsFromFtR = `C${ftStartRow}:C${ftEndRow}`
    addRow('Min', results.value.fps.mangohud.min, `=1000/MAX(${ftR})`, ref)
    addRow('Max', results.value.fps.mangohud.max, `=1000/MIN(${ftR})`, ref)
    addRow('Average', results.value.fps.mangohud.avg, `=1000/AVERAGE(${ftR})`, ref)
    addRow('Median', results.value.fps.mangohud.median, `=${mangoHudFormula(fpsFromFtR, 0.50)}`, ref)
    addRow('1st Percentile', results.value.fps.mangohud.p01, `=1000/${mangoHudFormula(ftR, 0.01)}`, ref)
    addRow('5th Percentile', results.value.fps.mangohud.p05, `=1000/${mangoHudFormula(ftR, 0.05)}`, ref)
    addRow('10th Percentile', results.value.fps.mangohud.p10, `=1000/${mangoHudFormula(ftR, 0.10)}`, ref)
    addRow('25th Percentile', results.value.fps.mangohud.p25, `=1000/${mangoHudFormula(ftR, 0.25)}`, ref)
    addRow('75th Percentile', results.value.fps.mangohud.p75, `=1000/${mangoHudFormula(ftR, 0.75)}`, ref)
    addRow('90th Percentile', results.value.fps.mangohud.p90, `=1000/${mangoHudFormula(ftR, 0.90)}`, ref)
    addRow('95th Percentile', results.value.fps.mangohud.p95, `=1000/${mangoHudFormula(ftR, 0.95)}`, ref)
    addRow('97th Percentile', results.value.fps.mangohud.p97, `=1000/${mangoHudFormula(ftR, 0.97)}`, ref)
    addRow('99th Percentile', results.value.fps.mangohud.p99, `=1000/${mangoHudFormula(ftR, 0.99)}`, ref)
    addRow('IQR (P75-P25)', results.value.fps.mangohud.iqr, `=1000/${mangoHudFormula(ftR, 0.75)}-1000/${mangoHudFormula(ftR, 0.25)}`, ref)
    addRow('Standard Deviation', results.value.fps.mangohud.stddev, `=STDEV(${fpsFromFtR})`, ref)
    addRow('Variance', results.value.fps.mangohud.variance, `=VAR(${fpsFromFtR})`, ref)
    addRow('Count', results.value.fps.mangohud.count, `=COUNT(${ftR})`, ref)
  } else {
    const fpsR = `A${fpsStartRow}:A${fpsEndRow}`
    addRow('Min', results.value.fps.mangohud.min, `=MIN(${fpsR})`, ref)
    addRow('Max', results.value.fps.mangohud.max, `=MAX(${fpsR})`, ref)
    addRow('Average', results.value.fps.mangohud.avg, `=AVERAGE(${fpsR})`, ref)
    addRow('Median', results.value.fps.mangohud.median, `=${mangoHudFormula(fpsR, 0.50)}`, ref)
    addRow('1st Percentile', results.value.fps.mangohud.p01, `=${mangoHudFormula(fpsR, 0.99)}`, ref)
    addRow('5th Percentile', results.value.fps.mangohud.p05, `=${mangoHudFormula(fpsR, 0.95)}`, ref)
    addRow('10th Percentile', results.value.fps.mangohud.p10, `=${mangoHudFormula(fpsR, 0.90)}`, ref)
    addRow('25th Percentile', results.value.fps.mangohud.p25, `=${mangoHudFormula(fpsR, 0.75)}`, ref)
    addRow('75th Percentile', results.value.fps.mangohud.p75, `=${mangoHudFormula(fpsR, 0.25)}`, ref)
    addRow('90th Percentile', results.value.fps.mangohud.p90, `=${mangoHudFormula(fpsR, 0.10)}`, ref)
    addRow('95th Percentile', results.value.fps.mangohud.p95, `=${mangoHudFormula(fpsR, 0.05)}`, ref)
    addRow('97th Percentile', results.value.fps.mangohud.p97, `=${mangoHudFormula(fpsR, 0.03)}`, ref)
    addRow('99th Percentile', results.value.fps.mangohud.p99, `=${mangoHudFormula(fpsR, 0.01)}`, ref)
    addRow('IQR (P75-P25)', results.value.fps.mangohud.iqr, `=${mangoHudFormula(fpsR, 0.25)}-${mangoHudFormula(fpsR, 0.75)}`, ref)
    addRow('Standard Deviation', results.value.fps.mangohud.stddev, `=STDEV(${fpsR})`, ref)
    addRow('Variance', results.value.fps.mangohud.variance, `=VAR(${fpsR})`, ref)
    addRow('Count', results.value.fps.mangohud.count, `=COUNT(${fpsR})`, ref)
  }
  
  // ===== Frametime Statistics - Linear Interpolation =====
  addSectionHeader('Frametime Statistics - Linear Interpolation', ref)
  {
    const ftR = `B${ftStartRow}:B${ftEndRow}`
    addRow('Min', results.value.frametime.linear.min, `=MIN(${ftR})`, ref)
    addRow('Max', results.value.frametime.linear.max, `=MAX(${ftR})`, ref)
    addRow('Average', results.value.frametime.linear.avg, `=AVERAGE(${ftR})`, ref)
    addRow('Median', results.value.frametime.linear.median, `=MEDIAN(${ftR})`, ref)
    addRow('1st Percentile', results.value.frametime.linear.p01, linearPercentileFormula(ftR, 0.01), ref)
    addRow('5th Percentile', results.value.frametime.linear.p05, linearPercentileFormula(ftR, 0.05), ref)
    addRow('10th Percentile', results.value.frametime.linear.p10, linearPercentileFormula(ftR, 0.10), ref)
    addRow('25th Percentile', results.value.frametime.linear.p25, linearPercentileFormula(ftR, 0.25), ref)
    addRow('75th Percentile', results.value.frametime.linear.p75, linearPercentileFormula(ftR, 0.75), ref)
    addRow('90th Percentile', results.value.frametime.linear.p90, linearPercentileFormula(ftR, 0.90), ref)
    addRow('95th Percentile', results.value.frametime.linear.p95, linearPercentileFormula(ftR, 0.95), ref)
    addRow('97th Percentile', results.value.frametime.linear.p97, linearPercentileFormula(ftR, 0.97), ref)
    addRow('99th Percentile', results.value.frametime.linear.p99, linearPercentileFormula(ftR, 0.99), ref)
    addRow('IQR (P75-P25)', results.value.frametime.linear.iqr, `=PERCENTILE(${ftR};0.75)-PERCENTILE(${ftR};0.25)`, ref)
    addRow('Standard Deviation', results.value.frametime.linear.stddev, `=STDEV(${ftR})`, ref)
    addRow('Variance', results.value.frametime.linear.variance, `=VAR(${ftR})`, ref)
    addRow('Count', results.value.frametime.linear.count, `=COUNT(${ftR})`, ref)
  }
  
  // ===== Frametime Statistics - Mangohud =====
  addSectionHeader('Frametime Statistics - Mangohud', ref)
  {
    const ftR = `B${ftStartRow}:B${ftEndRow}`
    addRow('Min', results.value.frametime.mangohud.min, `=MIN(${ftR})`, ref)
    addRow('Max', results.value.frametime.mangohud.max, `=MAX(${ftR})`, ref)
    addRow('Average', results.value.frametime.mangohud.avg, `=AVERAGE(${ftR})`, ref)
    addRow('Median', results.value.frametime.mangohud.median, `=${mangoHudFormula(ftR, 0.50)}`, ref)
    addRow('1st Percentile', results.value.frametime.mangohud.p01, `=${mangoHudFormula(ftR, 0.99)}`, ref)
    addRow('5th Percentile', results.value.frametime.mangohud.p05, `=${mangoHudFormula(ftR, 0.95)}`, ref)
    addRow('10th Percentile', results.value.frametime.mangohud.p10, `=${mangoHudFormula(ftR, 0.90)}`, ref)
    addRow('25th Percentile', results.value.frametime.mangohud.p25, `=${mangoHudFormula(ftR, 0.75)}`, ref)
    addRow('75th Percentile', results.value.frametime.mangohud.p75, `=${mangoHudFormula(ftR, 0.25)}`, ref)
    addRow('90th Percentile', results.value.frametime.mangohud.p90, `=${mangoHudFormula(ftR, 0.10)}`, ref)
    addRow('95th Percentile', results.value.frametime.mangohud.p95, `=${mangoHudFormula(ftR, 0.05)}`, ref)
    addRow('97th Percentile', results.value.frametime.mangohud.p97, `=${mangoHudFormula(ftR, 0.03)}`, ref)
    addRow('99th Percentile', results.value.frametime.mangohud.p99, `=${mangoHudFormula(ftR, 0.01)}`, ref)
    addRow('IQR (P75-P25)', results.value.frametime.mangohud.iqr, `=${mangoHudFormula(ftR, 0.25)}-${mangoHudFormula(ftR, 0.75)}`, ref)
    addRow('Standard Deviation', results.value.frametime.mangohud.stddev, `=STDEV(${ftR})`, ref)
    addRow('Variance', results.value.frametime.mangohud.variance, `=VAR(${ftR})`, ref)
    addRow('Count', results.value.frametime.mangohud.count, `=COUNT(${ftR})`, ref)
  }
  
  return lines.join('\n')
})

function formatNumber(value) {
  if (value === null || value === undefined) return 'N/A'
  if (Number.isInteger(value)) return value.toString()
  return value.toFixed(2)
}

async function copyToClipboard() {
  try {
    await navigator.clipboard.writeText(spreadsheetData.value)
    alert('Copied to clipboard!')
  } catch (err) {
    console.error('Failed to copy to clipboard:', err)
    alert('Failed to copy to clipboard. Please select and copy manually.')
  }
}

async function copyToClipboardLibreOffice() {
  try {
    await navigator.clipboard.writeText(spreadsheetDataLibreOffice.value)
    alert('Copied to clipboard!')
  } catch (err) {
    console.error('Failed to copy to clipboard:', err)
    alert('Failed to copy to clipboard. Please select and copy manually.')
  }
}
</script>

<style scoped>
.font-monospace {
  font-family: 'Courier New', monospace;
  font-size: 0.9em;
}

.card {
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.table {
  margin-bottom: 0;
}

.table th {
  background-color: rgba(0, 0, 0, 0.2);
  font-weight: 600;
}
</style>
