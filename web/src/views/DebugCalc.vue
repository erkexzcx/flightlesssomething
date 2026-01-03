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
            <h5 class="card-title">Results</h5>
            
            <!-- FPS Statistics -->
            <div class="mb-4">
              <h6 class="text-primary">FPS Statistics</h6>
              <div class="row">
                <div class="col-md-6">
                  <h6 class="text-muted small mt-3">Linear Interpolation Method</h6>
                  <table class="table table-sm table-bordered">
                    <tbody>
                      <tr>
                        <th>1% FPS (Low)</th>
                        <td>{{ formatNumber(results.fps.linear.p01) }}</td>
                      </tr>
                      <tr>
                        <th>Average FPS</th>
                        <td>{{ formatNumber(results.fps.linear.avg) }}</td>
                      </tr>
                      <tr>
                        <th>97th Percentile FPS</th>
                        <td>{{ formatNumber(results.fps.linear.p97) }}</td>
                      </tr>
                      <tr>
                        <th>Standard Deviation</th>
                        <td>{{ formatNumber(results.fps.linear.stddev) }}</td>
                      </tr>
                      <tr>
                        <th>Variance</th>
                        <td>{{ formatNumber(results.fps.linear.variance) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <div class="col-md-6">
                  <h6 class="text-muted small mt-3">MangoHud Threshold Method</h6>
                  <table class="table table-sm table-bordered">
                    <tbody>
                      <tr>
                        <th>1% FPS (Low)</th>
                        <td>{{ formatNumber(results.fps.mangohud.p01) }}</td>
                      </tr>
                      <tr>
                        <th>Average FPS</th>
                        <td>{{ formatNumber(results.fps.mangohud.avg) }}</td>
                      </tr>
                      <tr>
                        <th>97th Percentile FPS</th>
                        <td>{{ formatNumber(results.fps.mangohud.p97) }}</td>
                      </tr>
                      <tr>
                        <th>Standard Deviation</th>
                        <td>{{ formatNumber(results.fps.mangohud.stddev) }}</td>
                      </tr>
                      <tr>
                        <th>Variance</th>
                        <td>{{ formatNumber(results.fps.mangohud.variance) }}</td>
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
                      <tr>
                        <th>1% Frametime (High)</th>
                        <td>{{ formatNumber(results.frametime.linear.p01) }}</td>
                      </tr>
                      <tr>
                        <th>Average Frametime</th>
                        <td>{{ formatNumber(results.frametime.linear.avg) }}</td>
                      </tr>
                      <tr>
                        <th>97th Percentile Frametime</th>
                        <td>{{ formatNumber(results.frametime.linear.p97) }}</td>
                      </tr>
                      <tr>
                        <th>Standard Deviation</th>
                        <td>{{ formatNumber(results.frametime.linear.stddev) }}</td>
                      </tr>
                      <tr>
                        <th>Variance</th>
                        <td>{{ formatNumber(results.frametime.linear.variance) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <div class="col-md-6">
                  <h6 class="text-muted small mt-3">MangoHud Threshold Method</h6>
                  <table class="table table-sm table-bordered">
                    <tbody>
                      <tr>
                        <th>1% Frametime (High)</th>
                        <td>{{ formatNumber(results.frametime.mangohud.p01) }}</td>
                      </tr>
                      <tr>
                        <th>Average Frametime</th>
                        <td>{{ formatNumber(results.frametime.mangohud.avg) }}</td>
                      </tr>
                      <tr>
                        <th>97th Percentile Frametime</th>
                        <td>{{ formatNumber(results.frametime.mangohud.p97) }}</td>
                      </tr>
                      <tr>
                        <th>Standard Deviation</th>
                        <td>{{ formatNumber(results.frametime.mangohud.stddev) }}</td>
                      </tr>
                      <tr>
                        <th>Variance</th>
                        <td>{{ formatNumber(results.frametime.mangohud.variance) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
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

function resetToExample() {
  inputData.value = EXAMPLE_DATA
  results.value = null
  error.value = null
  parsedData.value = null
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

const spreadsheetData = computed(() => {
  if (!results.value || !parsedData.value) return ''

  const lines = []
  
  // Add header with context
  lines.push('NOTE: This export is for verification purposes only.')
  lines.push('Compare the "FlightlessSomething" column (calculated by this app) with the "Formula Result" column (calculated by your spreadsheet).')
  lines.push('Both values should match to verify the calculations are correct.')
  lines.push('')
  
  // Add raw data
  lines.push('fps,frametime')
  const maxLength = Math.max(parsedData.value.fpsValues.length, parsedData.value.frametimeValues.length)
  for (let i = 0; i < maxLength; i++) {
    const fps = i < parsedData.value.fpsValues.length ? parsedData.value.fpsValues[i] : ''
    const frametime = i < parsedData.value.frametimeValues.length ? parsedData.value.frametimeValues[i] : ''
    lines.push(`${fps},${frametime}`)
  }
  
  // Add blank line
  lines.push('')
  
  // Calculate row numbers (accounting for header lines)
  const headerLines = 4 // NOTE lines + blank line
  const dataStartRow = headerLines + 2 // After header lines + data header row
  const fpsStartRow = dataStartRow
  const fpsEndRow = fpsStartRow + parsedData.value.fpsValues.length - 1
  const ftStartRow = dataStartRow
  const ftEndRow = ftStartRow + parsedData.value.frametimeValues.length - 1
  
  // Add FPS statistics - Linear Interpolation
  lines.push('FPS Statistics - Linear Interpolation')
  lines.push('Metric,FlightlessSomething,Formula,Formula Result')
  
  // For FPS calculated from frametime
  if (parsedData.value.frametimeValues.length > 0) {
    lines.push(`1% FPS (Low),${formatNumber(results.value.fps.linear.p01)},=1000/PERCENTILE.INC(B${ftStartRow}:B${ftEndRow},0.99),`)
    lines.push(`Average FPS,${formatNumber(results.value.fps.linear.avg)},=1000/AVERAGE(B${ftStartRow}:B${ftEndRow}),`)
    lines.push(`97th Percentile FPS,${formatNumber(results.value.fps.linear.p97)},=1000/PERCENTILE.INC(B${ftStartRow}:B${ftEndRow},0.03),`)
    lines.push(`Standard Deviation,${formatNumber(results.value.fps.linear.stddev)},=STDEV(1000/B${ftStartRow}:B${ftEndRow}),`)
    lines.push(`Variance,${formatNumber(results.value.fps.linear.variance)},=VAR(1000/B${ftStartRow}:B${ftEndRow}),`)
  } else {
    lines.push(`1% FPS (Low),${formatNumber(results.value.fps.linear.p01)},=PERCENTILE.INC(A${fpsStartRow}:A${fpsEndRow},0.01),`)
    lines.push(`Average FPS,${formatNumber(results.value.fps.linear.avg)},=AVERAGE(A${fpsStartRow}:A${fpsEndRow}),`)
    lines.push(`97th Percentile FPS,${formatNumber(results.value.fps.linear.p97)},=PERCENTILE.INC(A${fpsStartRow}:A${fpsEndRow},0.97),`)
    lines.push(`Standard Deviation,${formatNumber(results.value.fps.linear.stddev)},=STDEV(A${fpsStartRow}:A${fpsEndRow}),`)
    lines.push(`Variance,${formatNumber(results.value.fps.linear.variance)},=VAR(A${fpsStartRow}:A${fpsEndRow}),`)
  }
  
  lines.push('')
  
  // Add FPS statistics - MangoHud Threshold
  lines.push('FPS Statistics - MangoHud Threshold')
  lines.push('Metric,FlightlessSomething,Formula,Formula Result')
  
  if (parsedData.value.frametimeValues.length > 0) {
    lines.push(`1% FPS (Low),${formatNumber(results.value.fps.mangohud.p01)},=1000/INDEX(SORT(B${ftStartRow}:B${ftEndRow}),FLOOR(COUNT(B${ftStartRow}:B${ftEndRow})*0.99,1)),`)
    lines.push(`Average FPS,${formatNumber(results.value.fps.mangohud.avg)},=1000/AVERAGE(B${ftStartRow}:B${ftEndRow}),`)
    lines.push(`97th Percentile FPS,${formatNumber(results.value.fps.mangohud.p97)},=1000/INDEX(SORT(B${ftStartRow}:B${ftEndRow}),FLOOR(COUNT(B${ftStartRow}:B${ftEndRow})*0.03,1)),`)
    lines.push(`Standard Deviation,${formatNumber(results.value.fps.mangohud.stddev)},=STDEV(1000/B${ftStartRow}:B${ftEndRow}),`)
    lines.push(`Variance,${formatNumber(results.value.fps.mangohud.variance)},=VAR(1000/B${ftStartRow}:B${ftEndRow}),`)
  } else {
    lines.push(`1% FPS (Low),${formatNumber(results.value.fps.mangohud.p01)},=INDEX(SORT(A${fpsStartRow}:A${fpsEndRow}),FLOOR(COUNT(A${fpsStartRow}:A${fpsEndRow})*0.01,1)),`)
    lines.push(`Average FPS,${formatNumber(results.value.fps.mangohud.avg)},=AVERAGE(A${fpsStartRow}:A${fpsEndRow}),`)
    lines.push(`97th Percentile FPS,${formatNumber(results.value.fps.mangohud.p97)},=INDEX(SORT(A${fpsStartRow}:A${fpsEndRow}),FLOOR(COUNT(A${fpsStartRow}:A${fpsEndRow})*0.97,1)),`)
    lines.push(`Standard Deviation,${formatNumber(results.value.fps.mangohud.stddev)},=STDEV(A${fpsStartRow}:A${fpsEndRow}),`)
    lines.push(`Variance,${formatNumber(results.value.fps.mangohud.variance)},=VAR(A${fpsStartRow}:A${fpsEndRow}),`)
  }
  
  lines.push('')
  
  // Add Frametime statistics - Linear Interpolation
  lines.push('Frametime Statistics - Linear Interpolation')
  lines.push('Metric,FlightlessSomething,Formula,Formula Result')
  lines.push(`1% Frametime (High),${formatNumber(results.value.frametime.linear.p01)},=PERCENTILE.INC(B${ftStartRow}:B${ftEndRow},0.01),`)
  lines.push(`Average Frametime,${formatNumber(results.value.frametime.linear.avg)},=AVERAGE(B${ftStartRow}:B${ftEndRow}),`)
  lines.push(`97th Percentile Frametime,${formatNumber(results.value.frametime.linear.p97)},=PERCENTILE.INC(B${ftStartRow}:B${ftEndRow},0.97),`)
  lines.push(`Standard Deviation,${formatNumber(results.value.frametime.linear.stddev)},=STDEV(B${ftStartRow}:B${ftEndRow}),`)
  lines.push(`Variance,${formatNumber(results.value.frametime.linear.variance)},=VAR(B${ftStartRow}:B${ftEndRow}),`)
  
  lines.push('')
  
  // Add Frametime statistics - MangoHud Threshold
  lines.push('Frametime Statistics - MangoHud Threshold')
  lines.push('Metric,FlightlessSomething,Formula,Formula Result')
  lines.push(`1% Frametime (High),${formatNumber(results.value.frametime.mangohud.p01)},=INDEX(SORT(B${ftStartRow}:B${ftEndRow}),FLOOR(COUNT(B${ftStartRow}:B${ftEndRow})*0.01,1)),`)
  lines.push(`Average Frametime,${formatNumber(results.value.frametime.mangohud.avg)},=AVERAGE(B${ftStartRow}:B${ftEndRow}),`)
  lines.push(`97th Percentile Frametime,${formatNumber(results.value.frametime.mangohud.p97)},=INDEX(SORT(B${ftStartRow}:B${ftEndRow}),FLOOR(COUNT(B${ftStartRow}:B${ftEndRow})*0.97,1)),`)
  lines.push(`Standard Deviation,${formatNumber(results.value.frametime.mangohud.stddev)},=STDEV(B${ftStartRow}:B${ftEndRow}),`)
  lines.push(`Variance,${formatNumber(results.value.frametime.mangohud.variance)},=VAR(B${ftStartRow}:B${ftEndRow}),`)
  
  return lines.join('\n')
})

// LibreOffice Calc / Excel compatible export
const spreadsheetDataLibreOffice = computed(() => {
  if (!results.value || !parsedData.value) return ''

  const lines = []
  
  // Add raw data (no header text, start directly with column headers)
  lines.push('fps\tframetime')
  const maxLength = Math.max(parsedData.value.fpsValues.length, parsedData.value.frametimeValues.length)
  const dataStartRow = 2 // After column headers
  
  for (let i = 0; i < maxLength; i++) {
    const fps = i < parsedData.value.fpsValues.length ? parsedData.value.fpsValues[i] : ''
    const frametime = i < parsedData.value.frametimeValues.length ? parsedData.value.frametimeValues[i] : ''
    lines.push(`${fps}\t${frametime}`)
  }
  
  lines.push('')
  
  const fpsStartRow = dataStartRow
  const fpsEndRow = dataStartRow + parsedData.value.fpsValues.length - 1
  const ftStartRow = dataStartRow
  const ftEndRow = dataStartRow + parsedData.value.frametimeValues.length - 1
  
  // Add FPS statistics - Linear Interpolation
  lines.push('FPS Statistics - Linear Interpolation')
  lines.push('Metric\tFlightlessSomething\tFormula\tFormula Result')
  
  // Use FPS column (A) directly for all FPS calculations
  lines.push(`1% FPS (Low)\t${formatNumber(results.value.fps.linear.p01)}\t=PERCENTILE(A${fpsStartRow}:A${fpsEndRow};0.01)\t=PERCENTILE(A${fpsStartRow}:A${fpsEndRow};0.01)`)
  lines.push(`Average FPS\t${formatNumber(results.value.fps.linear.avg)}\t=AVERAGE(A${fpsStartRow}:A${fpsEndRow})\t=AVERAGE(A${fpsStartRow}:A${fpsEndRow})`)
  lines.push(`97th Percentile FPS\t${formatNumber(results.value.fps.linear.p97)}\t=PERCENTILE(A${fpsStartRow}:A${fpsEndRow};0.97)\t=PERCENTILE(A${fpsStartRow}:A${fpsEndRow};0.97)`)
  lines.push(`Standard Deviation\t${formatNumber(results.value.fps.linear.stddev)}\t=STDEV(A${fpsStartRow}:A${fpsEndRow})\t=STDEV(A${fpsStartRow}:A${fpsEndRow})`)
  lines.push(`Variance\t${formatNumber(results.value.fps.linear.variance)}\t=VAR(A${fpsStartRow}:A${fpsEndRow})\t=VAR(A${fpsStartRow}:A${fpsEndRow})`)
  
  lines.push('')
  
  // Add FPS statistics - MangoHud Threshold
  lines.push('FPS Statistics - MangoHud Threshold')
  lines.push('Metric\tFlightlessSomething\tFormula\tFormula Result')
  
  const fpsCount = parsedData.value.fpsValues.length
  const fpsP01Index = Math.floor(fpsCount * 0.01) + 1  // SMALL is 1-indexed
  const fpsP97Index = fpsCount - Math.floor(fpsCount * 0.97) + 1  // LARGE is 1-indexed from the largest
  
  lines.push(`1% FPS (Low)\t${formatNumber(results.value.fps.mangohud.p01)}\t=SMALL(A${fpsStartRow}:A${fpsEndRow};${fpsP01Index})\t=SMALL(A${fpsStartRow}:A${fpsEndRow};${fpsP01Index})`)
  lines.push(`Average FPS\t${formatNumber(results.value.fps.mangohud.avg)}\t=AVERAGE(A${fpsStartRow}:A${fpsEndRow})\t=AVERAGE(A${fpsStartRow}:A${fpsEndRow})`)
  lines.push(`97th Percentile FPS\t${formatNumber(results.value.fps.mangohud.p97)}\t=LARGE(A${fpsStartRow}:A${fpsEndRow};${fpsP97Index})\t=LARGE(A${fpsStartRow}:A${fpsEndRow};${fpsP97Index})`)
  lines.push(`Standard Deviation\t${formatNumber(results.value.fps.mangohud.stddev)}\t=STDEV(A${fpsStartRow}:A${fpsEndRow})\t=STDEV(A${fpsStartRow}:A${fpsEndRow})`)
  lines.push(`Variance\t${formatNumber(results.value.fps.mangohud.variance)}\t=VAR(A${fpsStartRow}:A${fpsEndRow})\t=VAR(A${fpsStartRow}:A${fpsEndRow})`)
  
  lines.push('')
  
  // Add Frametime statistics - Linear Interpolation
  lines.push('Frametime Statistics - Linear Interpolation')
  lines.push('Metric\tFlightlessSomething\tFormula\tFormula Result')
  lines.push(`1% Frametime (High)\t${formatNumber(results.value.frametime.linear.p01)}\t=PERCENTILE(B${ftStartRow}:B${ftEndRow};0.01)\t=PERCENTILE(B${ftStartRow}:B${ftEndRow};0.01)`)
  lines.push(`Average Frametime\t${formatNumber(results.value.frametime.linear.avg)}\t=AVERAGE(B${ftStartRow}:B${ftEndRow})\t=AVERAGE(B${ftStartRow}:B${ftEndRow})`)
  lines.push(`97th Percentile Frametime\t${formatNumber(results.value.frametime.linear.p97)}\t=PERCENTILE(B${ftStartRow}:B${ftEndRow};0.97)\t=PERCENTILE(B${ftStartRow}:B${ftEndRow};0.97)`)
  lines.push(`Standard Deviation\t${formatNumber(results.value.frametime.linear.stddev)}\t=STDEV(B${ftStartRow}:B${ftEndRow})\t=STDEV(B${ftStartRow}:B${ftEndRow})`)
  lines.push(`Variance\t${formatNumber(results.value.frametime.linear.variance)}\t=VAR(B${ftStartRow}:B${ftEndRow})\t=VAR(B${ftStartRow}:B${ftEndRow})`)
  
  lines.push('')
  
  // Add Frametime statistics - MangoHud Threshold
  lines.push('Frametime Statistics - MangoHud Threshold')
  lines.push('Metric\tFlightlessSomething\tFormula\tFormula Result')
  
  const ftCount = parsedData.value.frametimeValues.length
  const ftP01Index = Math.floor(ftCount * 0.01) + 1  // SMALL is 1-indexed
  const ftP97Index = ftCount - Math.floor(ftCount * 0.97) + 1  // LARGE is 1-indexed from the largest
  
  lines.push(`1% Frametime (High)\t${formatNumber(results.value.frametime.mangohud.p01)}\t=SMALL(B${ftStartRow}:B${ftEndRow};${ftP01Index})\t=SMALL(B${ftStartRow}:B${ftEndRow};${ftP01Index})`)
  lines.push(`Average Frametime\t${formatNumber(results.value.frametime.mangohud.avg)}\t=AVERAGE(B${ftStartRow}:B${ftEndRow})\t=AVERAGE(B${ftStartRow}:B${ftEndRow})`)
  lines.push(`97th Percentile Frametime\t${formatNumber(results.value.frametime.mangohud.p97)}\t=LARGE(B${ftStartRow}:B${ftEndRow};${ftP97Index})\t=LARGE(B${ftStartRow}:B${ftEndRow};${ftP97Index})`)
  lines.push(`Standard Deviation\t${formatNumber(results.value.frametime.mangohud.stddev)}\t=STDEV(B${ftStartRow}:B${ftEndRow})\t=STDEV(B${ftStartRow}:B${ftEndRow})`)
  lines.push(`Variance\t${formatNumber(results.value.frametime.mangohud.variance)}\t=VAR(B${ftStartRow}:B${ftEndRow})\t=VAR(B${ftStartRow}:B${ftEndRow})`)
  
  return lines.join('\n')
})

function formatNumber(value) {
  if (value === null || value === undefined) return 'N/A'
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
