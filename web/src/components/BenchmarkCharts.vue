<template>
  <div>
    <!-- Specifications Table -->
    <div class="row mb-4" v-if="benchmarkData && benchmarkData.length > 0">
      <div class="col-12">
        <h5 class="text-center" style="font-size: 16px; font-weight: bold;">Specifications</h5>
        <div class="table-responsive">
          <table class="table table-sm table-bordered">
            <thead>
              <tr>
                <th scope="col">Label</th>
                <th scope="col">OS</th>
                <th scope="col">GPU</th>
                <th scope="col">CPU</th>
                <th scope="col">RAM</th>
                <th scope="col">OS specific</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(data, index) in benchmarkData" :key="index">
                <th scope="row">{{ data.label }}</th>
                <td>{{ data.specOS || '-' }}</td>
                <td>{{ data.specGPU || '-' }}</td>
                <td>{{ data.specCPU || '-' }}</td>
                <td>{{ data.specRAM || '-' }}</td>
                <td>{{ formatOSSpecific(data) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Calculation Method Selector -->
    <div class="row mb-3" v-if="benchmarkData && benchmarkData.length > 0">
      <div class="col-12">
        <div class="calculation-method-selector">
          <div class="row g-2 justify-content-center justify-content-md-start">
            <div class="col-12 col-md-auto text-center text-md-start">
              <label class="form-label mb-0">
                <strong>Calculation method:</strong>
              </label>
            </div>
            <div class="col-12 col-md-auto d-flex justify-content-center justify-content-md-start">
              <div class="btn-group btn-group-sm method-toggle" role="group">
                <input 
                  type="radio" 
                  class="btn-check" 
                  name="calculationMethod" 
                  id="linearInterpolation" 
                  autocomplete="off"
                  :checked="appStore.calculationMethod === 'linear-interpolation'"
                  @change="setCalculationMethod('linear-interpolation')"
                >
                <label class="btn btn-outline-primary" for="linearInterpolation">
                  Linear Interpolation
                </label>

                <input 
                  type="radio" 
                  class="btn-check" 
                  name="calculationMethod" 
                  id="mangoHudThreshold" 
                  autocomplete="off"
                  :checked="appStore.calculationMethod === 'mangohud-threshold'"
                  @change="setCalculationMethod('mangohud-threshold')"
                >
                <label class="btn btn-outline-primary" for="mangoHudThreshold">
                  Mangohud
                </label>
              </div>
              <button 
                class="btn btn-sm btn-outline-secondary ms-2" 
                type="button"
                data-bs-toggle="modal"
                data-bs-target="#calculationMethodModal"
                title="Learn more about calculation methods"
              >
                <i class="fas fa-info-circle"></i>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Calculation Method Info Modal -->
    <div class="modal fade" id="calculationMethodModal" tabindex="-1" aria-labelledby="calculationMethodModalLabel" aria-hidden="true">
      <div class="modal-dialog modal-xl">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title" id="calculationMethodModalLabel">
              <i class="fas fa-info-circle"></i> Calculation Methods Explained
            </h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
          </div>
          <div class="modal-body">
            <div class="row">
              <div class="col-md-6">
                <h5><strong>Linear Interpolation</strong></h5>
                <p>
                  Uses mathematical interpolation between adjacent data points when calculating percentiles. 
                  This is the standard scientific approach used by statistical tools like NumPy, R, and Excel, 
                  providing the most mathematically precise percentile values.
                </p>
                
                <h6 class="mt-3">Example:</h6>
                <div class="example-box p-3 bg-dark rounded">
                  <p class="mb-2"><strong>Dataset:</strong> [10, 20, 30, 40, 50, 60, 70, 80, 90, 100]</p>
                  <p class="mb-2"><strong>97th percentile:</strong></p>
                  <ul class="mb-0">
                    <li>Position: 0.97 × 9 = 8.73 (between indices 8 and 9)</li>
                    <li>Values: 90 and 100</li>
                    <li>Result: 90 × (1 - 0.73) + 100 × 0.73 = <strong>97.3</strong></li>
                  </ul>
                </div>
              </div>

              <div class="col-md-6">
                <h5><strong>Mangohud</strong></h5>
                <p>
                  Uses a simpler floor-based approach without interpolation. This method is used by MangoHud 
                  (a popular Linux gaming overlay) for real-time performance monitoring, making it directly 
                  comparable with MangoHud screenshots and community benchmarks.
                </p>
                
                <h6 class="mt-3">Example:</h6>
                <div class="example-box p-3 bg-dark rounded">
                  <p class="mb-2"><strong>Dataset:</strong> [10, 20, 30, 40, 50, 60, 70, 80, 90, 100]</p>
                  <p class="mb-2"><strong>97th percentile:</strong></p>
                  <ul class="mb-0">
                    <li>Position: floor(0.97 × 10) = floor(9.7) = 9</li>
                    <li>Value at index 9: <strong>100</strong></li>
                    <li>Result: <strong>100</strong> (no interpolation)</li>
                  </ul>
                </div>
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Tabs for different chart categories -->
    <ul class="nav nav-tabs" id="chartTabs" role="tablist">
      <li class="nav-item" role="presentation">
        <button 
          class="nav-link active" 
          id="fps-tab" 
          data-bs-toggle="tab" 
          data-bs-target="#fps" 
          type="button" 
          role="tab"
          @click="handleTabClick('fps')"
        >
          <i class="fas fa-tachometer-alt"></i> FPS
        </button>
      </li>
      <li class="nav-item" role="presentation">
        <button 
          class="nav-link" 
          id="frametime-tab" 
          data-bs-toggle="tab" 
          data-bs-target="#frametime" 
          type="button" 
          role="tab"
          @click="handleTabClick('frametime')"
        >
          <i class="fas fa-clock"></i> Frametime
        </button>
      </li>
      <li class="nav-item" role="presentation">
        <button 
          class="nav-link" 
          id="summary-tab" 
          data-bs-toggle="tab" 
          data-bs-target="#summary" 
          type="button" 
          role="tab"
          @click="handleTabClick('summary')"
        >
          <i class="fas fa-chart-bar"></i> Summary
        </button>
      </li>
      <li class="nav-item" role="presentation">
        <button 
          class="nav-link" 
          id="more-metrics-tab" 
          data-bs-toggle="tab" 
          data-bs-target="#more-metrics" 
          type="button" 
          role="tab"
          @click="handleTabClick('more-metrics')"
        >
          <i class="fas fa-chart-line"></i> All data
        </button>
      </li>
    </ul>

    <div class="tab-content" id="chartTabsContent">
      <!-- FPS Tab -->
      <div class="tab-pane fade show active" id="fps" role="tabpanel">
        <div v-if="!renderedTabs.fps" class="text-center my-5">
          <div class="spinner-border" role="status">
            <span class="visually-hidden">Rendering charts...</span>
          </div>
          <p class="text-muted mt-2">Rendering FPS charts...</p>
        </div>
        <div v-show="renderedTabs.fps">
          <div ref="fpsChart2" style="height:250pt;"></div>
          <div ref="fpsMinMaxAvgChart" style="height:500pt;"></div>
          <div ref="fpsDensityChart" style="height:250pt;"></div>
          <!-- Comparison charts - only show when there are multiple runs -->
          <div v-if="benchmarkData && benchmarkData.length > 1">
            <!-- Baseline selector and mode toggle -->
            <div class="baseline-selector mb-3">
              <div class="row g-2 justify-content-center justify-content-md-start">
                <div class="col-12 col-md-auto d-flex align-items-center justify-content-center justify-content-md-start">
                  <label for="fpsBaselineSelect" class="form-label mb-0 me-2">Baseline (0%):</label>
                </div>
                <div class="col-12 col-md-auto d-flex justify-content-center justify-content-md-start">
                  <select 
                    id="fpsBaselineSelect" 
                    class="form-select form-select-sm"
                    v-model="fpsBaselineIndex"
                    @change="renderFPSComparison"
                  >
                    <option :value="null">Auto (slowest)</option>
                    <option 
                      v-for="(data, index) in benchmarkData" 
                      :key="index" 
                      :value="index"
                    >
                      {{ data.label }}
                    </option>
                  </select>
                </div>
                <div class="col-12 col-md-auto d-flex justify-content-center justify-content-md-start">
                  <div class="btn-group btn-group-sm w-100" role="group">
                    <input 
                      type="radio" 
                      class="btn-check" 
                      name="fpsComparisonMode" 
                      id="fpsNumbersMode" 
                      autocomplete="off"
                      :checked="comparisonMode === 'numbers'"
                      @change="appStore.setComparisonMode('numbers'); renderFPSComparison()"
                    >
                    <label class="btn btn-outline-primary" for="fpsNumbersMode">
                      Numbers
                    </label>
                    <input 
                      type="radio" 
                      class="btn-check" 
                      name="fpsComparisonMode" 
                      id="fpsNumbersDiffMode" 
                      autocomplete="off"
                      :checked="comparisonMode === 'numbers-diff'"
                      @change="appStore.setComparisonMode('numbers-diff'); renderFPSComparison()"
                    >
                    <label class="btn btn-outline-primary" for="fpsNumbersDiffMode">
                      Numbers diff
                    </label>
                    <input 
                      type="radio" 
                      class="btn-check" 
                      name="fpsComparisonMode" 
                      id="fpsPercentageMode" 
                      autocomplete="off"
                      :checked="comparisonMode === 'percentage'"
                      @change="appStore.setComparisonMode('percentage'); renderFPSComparison()"
                    >
                    <label class="btn btn-outline-primary" for="fpsPercentageMode">
                      Percent diff
                    </label>
                  </div>
                </div>
              </div>
            </div>
            <div ref="fpsAvgChart" style="height:250pt;"></div>
            <div class="row">
              <div class="col-12 col-md-6">
                <div ref="fpsStddevChart" style="height:400pt;"></div>
              </div>
              <div class="col-12 col-md-6">
                <div ref="fpsVarianceChart" style="height:400pt;"></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Frametime Tab -->
      <div class="tab-pane fade" id="frametime" role="tabpanel">
        <div v-if="!renderedTabs.frametime" class="text-center my-5">
          <div class="spinner-border" role="status">
            <span class="visually-hidden">Rendering charts...</span>
          </div>
          <p class="text-muted mt-2">Rendering Frametime charts...</p>
        </div>
        <div v-show="renderedTabs.frametime">
          <div ref="frameTimeChart2" style="height:250pt;"></div>
          <div ref="frametimeMinMaxAvgChart" style="height:500pt;"></div>
          <div ref="frametimeDensityChart" style="height:250pt;"></div>
          <!-- Comparison charts - only show when there are multiple runs -->
          <div v-if="benchmarkData && benchmarkData.length > 1">
            <!-- Baseline selector and mode toggle -->
            <div class="baseline-selector mb-3">
              <div class="row g-2 justify-content-center justify-content-md-start">
                <div class="col-12 col-md-auto d-flex align-items-center justify-content-center justify-content-md-start">
                  <label for="frametimeBaselineSelect" class="form-label mb-0 me-2">Baseline (0%):</label>
                </div>
                <div class="col-12 col-md-auto d-flex justify-content-center justify-content-md-start">
                  <select 
                    id="frametimeBaselineSelect" 
                    class="form-select form-select-sm"
                    v-model="frametimeBaselineIndex"
                    @change="renderFrametimeComparison"
                  >
                    <option :value="null">Auto (fastest)</option>
                    <option 
                      v-for="(data, index) in benchmarkData" 
                      :key="index" 
                      :value="index"
                    >
                      {{ data.label }}
                    </option>
                  </select>
                </div>
                <div class="col-12 col-md-auto d-flex justify-content-center justify-content-md-start">
                  <div class="btn-group btn-group-sm w-100" role="group">
                    <input 
                      type="radio" 
                      class="btn-check" 
                      name="frametimeComparisonMode" 
                      id="frametimeNumbersMode" 
                      autocomplete="off"
                      :checked="comparisonMode === 'numbers'"
                      @change="appStore.setComparisonMode('numbers'); renderFrametimeComparison()"
                    >
                    <label class="btn btn-outline-primary" for="frametimeNumbersMode">
                      Numbers
                    </label>
                    <input 
                      type="radio" 
                      class="btn-check" 
                      name="frametimeComparisonMode" 
                      id="frametimeNumbersDiffMode" 
                      autocomplete="off"
                      :checked="comparisonMode === 'numbers-diff'"
                      @change="appStore.setComparisonMode('numbers-diff'); renderFrametimeComparison()"
                    >
                    <label class="btn btn-outline-primary" for="frametimeNumbersDiffMode">
                      Numbers diff
                    </label>
                    <input 
                      type="radio" 
                      class="btn-check" 
                      name="frametimeComparisonMode" 
                      id="frametimePercentageMode" 
                      autocomplete="off"
                      :checked="comparisonMode === 'percentage'"
                      @change="appStore.setComparisonMode('percentage'); renderFrametimeComparison()"
                    >
                    <label class="btn btn-outline-primary" for="frametimePercentageMode">
                      Percent diff
                    </label>
                  </div>
                </div>
              </div>
            </div>
            <div ref="frametimeAvgChart" style="height:250pt;"></div>
            <div class="row">
              <div class="col-12 col-md-6">
                <div ref="frametimeStddevChart" style="height:400pt;"></div>
              </div>
              <div class="col-12 col-md-6">
                <div ref="frametimeVarianceChart" style="height:400pt;"></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Summary Tab -->
      <div class="tab-pane fade" id="summary" role="tabpanel">
        <div v-if="!renderedTabs.summary" class="text-center my-5">
          <div class="spinner-border" role="status">
            <span class="visually-hidden">Rendering charts...</span>
          </div>
          <p class="text-muted mt-2">Rendering Summary charts...</p>
        </div>
        <div v-show="renderedTabs.summary">
          <div class="row">
            <div class="col-md-6">
              <div ref="fpsSummaryChart" style="height:250pt;"></div>
            </div>
            <div class="col-md-6">
              <div ref="frametimeSummaryChart" style="height:250pt;"></div>
            </div>
          </div>
          <div class="row">
            <div class="col-md-6">
              <div ref="cpuLoadSummaryChart" style="height:250pt;"></div>
            </div>
            <div class="col-md-6">
              <div ref="gpuLoadSummaryChart" style="height:250pt;"></div>
            </div>
          </div>
          <div class="row">
            <div class="col-md-6">
              <div ref="gpuCoreClockSummaryChart" style="height:250pt;"></div>
            </div>
            <div class="col-md-6">
              <div ref="gpuMemClockSummaryChart" style="height:250pt;"></div>
            </div>
          </div>
          <div class="row">
            <div class="col-md-6">
              <div ref="cpuPowerSummaryChart" style="height:250pt;"></div>
            </div>
            <div class="col-md-6">
              <div ref="gpuPowerSummaryChart" style="height:250pt;"></div>
            </div>
          </div>
        </div>
      </div>

      <!-- More Metrics Tab -->
      <div class="tab-pane fade" id="more-metrics" role="tabpanel">
        <div v-if="!renderedTabs['more-metrics']" class="text-center my-5">
          <div class="spinner-border" role="status">
            <span class="visually-hidden">Rendering charts...</span>
          </div>
          <p class="text-muted mt-2">Rendering All Data charts...</p>
        </div>
        <div v-show="renderedTabs['more-metrics']">
          <div ref="fpsChart" style="height:250pt;"></div>
          <div ref="frameTimeChart" style="height:250pt;"></div>
          <div ref="cpuLoadChart" style="height:250pt;"></div>
          <div ref="gpuLoadChart" style="height:250pt;"></div>
          <div ref="cpuTempChart" style="height:250pt;"></div>
          <div ref="cpuPowerChart" style="height:250pt;"></div>
          <div ref="gpuTempChart" style="height:250pt;"></div>
          <div ref="gpuCoreClockChart" style="height:250pt;"></div>
          <div ref="gpuMemClockChart" style="height:250pt;"></div>
          <div ref="gpuVRAMUsedChart" style="height:250pt;"></div>
          <div ref="gpuPowerChart" style="height:250pt;"></div>
          <div ref="ramUsedChart" style="height:250pt;"></div>
          <div ref="swapUsedChart" style="height:250pt;"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch, nextTick, computed } from 'vue'
import { useAppStore } from '../stores/app'
import Highcharts from 'highcharts'
import HighchartsBoost from 'highcharts/modules/boost'
import HighchartsExporting from 'highcharts/modules/exporting'
import HighchartsExportData from 'highcharts/modules/export-data'
import HighchartsFullScreen from 'highcharts/modules/full-screen'

// Initialize Highcharts modules
HighchartsBoost(Highcharts)
HighchartsExporting(Highcharts)
HighchartsExportData(Highcharts)
HighchartsFullScreen(Highcharts)

// Use app store for calculation mode
const appStore = useAppStore()

// Get theme-aware colors
const getThemeColors = computed(() => {
  const isDark = appStore.theme === 'dark'
  return {
    textColor: isDark ? '#FFFFFF' : '#000000',
    // Increased contrast for light theme grid lines (from 0.1 to 0.2 opacity)
    gridLineColor: isDark ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.2)',
    lineColor: isDark ? '#FFFFFF' : '#000000',
    tooltipBg: isDark ? '#1E1E1E' : '#F5F5F5',
    tooltipBorder: isDark ? '#FFFFFF' : '#000000',
    // Use softer background for light theme (light gray) and Bootstrap dark for dark theme
    chartBg: isDark ? '#212529' : '#F5F5F5',
    // Bar border color - white for dark theme, black for light theme
    barBorderColor: isDark ? '#FFFFFF' : '#000000',
  }
})

const props = defineProps({
  benchmarkData: {
    type: Array,
    default: () => []
  }
})

// Track which tabs have been rendered
const renderedTabs = ref({
  fps: false,
  frametime: false,
  summary: false,
  'more-metrics': false
})

// Track selected baseline for comparison charts
const fpsBaselineIndex = ref(null) // null means auto (slowest)
const frametimeBaselineIndex = ref(null) // null means auto (fastest)

// Track display mode for comparison charts using app store for persistence
// Mode can be: 'percentage', 'numbers', or 'numbers-diff'
const comparisonMode = computed(() => appStore.comparisonMode)

// Refs for chart containers
const fpsChart = ref(null)
const fpsChart2 = ref(null)
const frameTimeChart = ref(null)
const frameTimeChart2 = ref(null)
const cpuLoadChart = ref(null)
const gpuLoadChart = ref(null)
const cpuTempChart = ref(null)
const cpuPowerChart = ref(null)
const gpuTempChart = ref(null)
const gpuCoreClockChart = ref(null)
const gpuMemClockChart = ref(null)
const gpuVRAMUsedChart = ref(null)
const gpuPowerChart = ref(null)
const ramUsedChart = ref(null)
const swapUsedChart = ref(null)
const fpsMinMaxAvgChart = ref(null)
const fpsDensityChart = ref(null)
const fpsAvgChart = ref(null)
const fpsStddevChart = ref(null)
const fpsVarianceChart = ref(null)
const frametimeMinMaxAvgChart = ref(null)
const frametimeDensityChart = ref(null)
const frametimeAvgChart = ref(null)
const frametimeStddevChart = ref(null)
const frametimeVarianceChart = ref(null)
const fpsSummaryChart = ref(null)
const frametimeSummaryChart = ref(null)
const cpuLoadSummaryChart = ref(null)
const gpuLoadSummaryChart = ref(null)
const gpuCoreClockSummaryChart = ref(null)
const gpuMemClockSummaryChart = ref(null)
const cpuPowerSummaryChart = ref(null)
const gpuPowerSummaryChart = ref(null)

// Track current device pixel ratio for HiDPI display support
const currentPixelRatio = ref(window.devicePixelRatio || 1)

// Common chart options - computed to react to theme changes and DPI changes
const commonChartOptions = computed(() => {
  const colors = getThemeColors.value
  return {
    chart: { 
      backgroundColor: colors.chartBg, 
      style: { color: colors.textColor }, 
      animation: false, 
      boost: { 
        useGPUTranslations: true, 
        usePreallocated: true,
        seriesThreshold: 1,  // Enable boost for all series
        // HiDPI/4K Display Support: Scale canvas to match device pixel ratio
        // This ensures crisp rendering on Retina displays and 4K monitors
        // The canvas dimensions are multiplied by this ratio and then scaled back via transform
        pixelRatio: currentPixelRatio.value
      } 
    },
    title: { style: { color: colors.textColor, fontSize: '16px' } },
    subtitle: { style: { color: colors.textColor, fontSize: '12px' } },
    xAxis: { labels: { style: { color: colors.textColor } }, lineColor: colors.lineColor, tickColor: colors.lineColor },
    yAxis: { labels: { style: { color: colors.textColor } }, gridLineColor: colors.gridLineColor, title: { text: false } },
    tooltip: { backgroundColor: colors.tooltipBg, borderColor: colors.tooltipBorder, style: { color: colors.textColor } },
    legend: { itemStyle: { color: colors.textColor } },
    credits: { enabled: false },
    plotOptions: { series: { animation: false, turboThreshold: 0 } }  // turboThreshold: 0 allows any number of points
  }
})

// Set global Highcharts options
// Note: Per-chart pixelRatio is set in commonChartOptions computed property
// This global setting serves as a fallback
Highcharts.setOptions({ 
  chart: { animation: false }, 
  plotOptions: { series: { animation: false, turboThreshold: 0 } },
  boost: { 
    enabled: true, 
    // HiDPI/4K Display Support: This is a fallback value, actual value comes from computed options
    pixelRatio: window.devicePixelRatio || 1 
  }
})

// Helper functions
function formatOSSpecific(data) {
  const parts = []
  // Processed data uses specOSSpecific object
  if (data.specOSSpecific?.SpecLinuxKernel) parts.push(data.specOSSpecific.SpecLinuxKernel)
  if (data.specOSSpecific?.SpecLinuxScheduler) parts.push(data.specOSSpecific.SpecLinuxScheduler)
  return parts.length > 0 ? parts.join(' ') : '-'
}

// Simple decimation function for line charts to improve rendering performance
// This keeps every Nth point to reduce the number of points rendered
function decimateForLineChart(data, targetPoints = 2000) {
  if (!data || data.length <= targetPoints) {
    return data
  }
  
  const step = Math.ceil(data.length / targetPoints)
  const decimated = []
  
  // Always include first point
  decimated.push(data[0])
  
  // Include every Nth point
  for (let i = step; i < data.length - 1; i += step) {
    decimated.push(data[i])
  }
  
  // Always include last point
  if (data.length > 1) {
    decimated.push(data[data.length - 1])
  }
  
  return decimated
}

function calculateAverage(data) {
  if (!data || data.length === 0) return 0
  return data.reduce((acc, value) => acc + value, 0) / data.length
}

// Wrapper function to render all FPS comparison charts (avg, stddev, variance)
function renderFPSComparison() {
  renderFPSComparisonChart()
  renderFPSStddevChart()
  renderFPSVarianceChart()
}

// Wrapper function to render all Frametime comparison charts (avg, stddev, variance)
function renderFrametimeComparison() {
  renderFrametimeComparisonChart()
  renderFrametimeStddevChart()
  renderFrametimeVarianceChart()
}

function renderFPSComparisonChart() {
  if (!fpsAvgChart.value || !props.benchmarkData || props.benchmarkData.length === 0) return
  
  const stats = fpsStats.value
  if (!stats || stats.length === 0) return
  
  const fpsAverages = stats.map(s => s.avg)
  
  // Determine baseline FPS
  let baselineFPS
  if (fpsBaselineIndex.value !== null && fpsBaselineIndex.value >= 0 && fpsBaselineIndex.value < fpsAverages.length) {
    baselineFPS = fpsAverages[fpsBaselineIndex.value]
  } else {
    // Auto mode: use slowest (minimum FPS)
    baselineFPS = Math.min(...fpsAverages)
  }
  
  const colors = getThemeColors.value
  const chartOpts = commonChartOptions.value
  
  if (comparisonMode.value === 'percentage') {
    // Percentage mode - show percentage differences from baseline
    // Calculate percentage differences from baseline (0% = baseline)
    const percentageFPSData = fpsAverages.map(fps => ((fps - baselineFPS) / baselineFPS) * 100)

    const sortedData = stats.map((s, index) => ({
      label: s.label,
      percentage: percentageFPSData[index]
    })).sort((a, b) => a.percentage - b.percentage)

    const sortedCategories = sortedData.map(item => item.label)
    const sortedPercentages = sortedData.map(item => item.percentage)

    // Determine min/max for y-axis with some padding
    const minPercentage = Math.min(...sortedPercentages)
    const maxPercentage = Math.max(...sortedPercentages)
    const padding = Math.max(Math.abs(minPercentage), Math.abs(maxPercentage)) * 0.1
    const yAxisMin = minPercentage - padding
    const yAxisMax = maxPercentage + padding

    Highcharts.chart(fpsAvgChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'Avg FPS comparison in %' },
      subtitle: { ...chartOpts.subtitle, text: 'More is better' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { 
        ...chartOpts.yAxis, 
        min: yAxisMin,
        max: yAxisMax,
        title: { text: 'Difference (%)', align: 'high', style: { color: colors.textColor } },
        plotLines: [{
          value: 0,
          color: colors.lineColor,
          width: 2,
          zIndex: 4
        }]
      },
      tooltip: { 
        ...chartOpts.tooltip, 
        valueSuffix: ' %', 
        formatter: function() { 
          const sign = this.y >= 0 ? '+' : ''
          return `<b>${this.point.category}</b>: ${sign}${this.y.toFixed(2)} %` 
        } 
      },
      plotOptions: { 
        bar: { 
          borderColor: colors.barBorderColor,
          borderWidth: 1,
          dataLabels: { 
            enabled: true, 
            style: { color: colors.textColor }, 
            formatter: function() { 
              const sign = this.y >= 0 ? '+' : ''
              return sign + this.y.toFixed(2) + ' %' 
            } 
          } 
        } 
      },
      legend: { enabled: false },
      series: [{ name: 'FPS Difference', data: sortedPercentages, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })
  } else if (comparisonMode.value === 'numbers-diff') {
    // Numbers diff mode - show absolute differences from baseline
    const differenceData = fpsAverages.map(fps => fps - baselineFPS)

    const sortedData = stats.map((s, index) => ({
      label: s.label,
      difference: differenceData[index]
    })).sort((a, b) => a.difference - b.difference)

    const sortedCategories = sortedData.map(item => item.label)
    const sortedDifferences = sortedData.map(item => item.difference)

    // Determine min/max for y-axis with some padding
    const minDiff = Math.min(...sortedDifferences)
    const maxDiff = Math.max(...sortedDifferences)
    const padding = Math.max(Math.abs(minDiff), Math.abs(maxDiff)) * 0.1
    const yAxisMin = minDiff - padding
    const yAxisMax = maxDiff + padding

    Highcharts.chart(fpsAvgChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'Avg FPS comparison' },
      subtitle: { ...chartOpts.subtitle, text: 'More is better' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { 
        ...chartOpts.yAxis, 
        min: yAxisMin,
        max: yAxisMax,
        title: { text: 'Difference (FPS)', align: 'high', style: { color: colors.textColor } },
        plotLines: [{
          value: 0,
          color: colors.lineColor,
          width: 2,
          zIndex: 4
        }]
      },
      tooltip: { 
        ...chartOpts.tooltip, 
        valueSuffix: ' fps', 
        formatter: function() { 
          const sign = this.y >= 0 ? '+' : ''
          return `<b>${this.point.category}</b>: ${sign}${this.y.toFixed(2)} fps` 
        } 
      },
      plotOptions: { 
        bar: { 
          borderColor: colors.barBorderColor,
          borderWidth: 1,
          dataLabels: { 
            enabled: true, 
            style: { color: colors.textColor }, 
            formatter: function() { 
              const sign = this.y >= 0 ? '+' : ''
              return sign + this.y.toFixed(2) + ' fps' 
            } 
          } 
        } 
      },
      legend: { enabled: false },
      series: [{ name: 'FPS Difference', data: sortedDifferences, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })
  } else {
    // Numbers mode - show absolute FPS values
    const sortedData = stats.map((s, index) => ({
      label: s.label,
      value: fpsAverages[index]
    })).sort((a, b) => a.value - b.value)

    const sortedCategories = sortedData.map(item => item.label)
    const sortedValues = sortedData.map(item => item.value)

    Highcharts.chart(fpsAvgChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'Avg FPS comparison' },
      subtitle: { ...chartOpts.subtitle, text: 'More is better' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { 
        ...chartOpts.yAxis, 
        title: { text: 'FPS', align: 'high', style: { color: colors.textColor } }
      },
      tooltip: { 
        ...chartOpts.tooltip, 
        valueSuffix: ' fps', 
        formatter: function() { 
          return `<b>${this.point.category}</b>: ${this.y.toFixed(2)} fps` 
        } 
      },
      plotOptions: { 
        bar: { 
          borderColor: colors.barBorderColor,
          borderWidth: 1,
          dataLabels: { 
            enabled: true, 
            style: { color: colors.textColor }, 
            formatter: function() { 
              return this.y.toFixed(2) + ' fps' 
            } 
          } 
        } 
      },
      legend: { enabled: false },
      series: [{ name: 'Avg FPS', data: sortedValues, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })
  }
}

function renderFrametimeComparisonChart() {
  if (!frametimeAvgChart.value || !props.benchmarkData || props.benchmarkData.length === 0) return
  
  const stats = frametimeStats.value
  if (!stats || stats.length === 0) return
  
  const frametimeAverages = stats.map(s => s.avg)
  
  // Determine baseline frametime
  let baselineFrametime
  if (frametimeBaselineIndex.value !== null && frametimeBaselineIndex.value >= 0 && frametimeBaselineIndex.value < frametimeAverages.length) {
    baselineFrametime = frametimeAverages[frametimeBaselineIndex.value]
  } else {
    // Auto mode: use fastest (minimum frametime)
    baselineFrametime = Math.min(...frametimeAverages)
  }
  
  const colors = getThemeColors.value
  const chartOpts = commonChartOptions.value

  if (comparisonMode.value === 'percentage') {
    // Percentage mode
    // Calculate percentage differences from baseline (0% = baseline)
    const percentageData = frametimeAverages.map(ft => ((ft - baselineFrametime) / baselineFrametime) * 100)

    const sortedData = stats.map((s, index) => ({
      label: s.label,
      percentage: percentageData[index]
    })).sort((a, b) => a.percentage - b.percentage)

    const sortedCategories = sortedData.map(item => item.label)
    const sortedPercentages = sortedData.map(item => item.percentage)

    // Determine min/max for y-axis with some padding
    const minPercentage = Math.min(...sortedPercentages)
    const maxPercentage = Math.max(...sortedPercentages)
    const padding = Math.max(Math.abs(minPercentage), Math.abs(maxPercentage)) * 0.1
    const yAxisMin = minPercentage - padding
    const yAxisMax = maxPercentage + padding

    Highcharts.chart(frametimeAvgChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'Avg Frametime comparison in %' },
      subtitle: { ...chartOpts.subtitle, text: 'Less is better' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { 
        ...chartOpts.yAxis, 
        min: yAxisMin,
        max: yAxisMax,
        title: { text: 'Difference (%)', align: 'high', style: { color: colors.textColor } },
        plotLines: [{
          value: 0,
          color: colors.lineColor,
          width: 2,
          zIndex: 4
        }]
      },
      tooltip: { 
        ...chartOpts.tooltip, 
        valueSuffix: ' %', 
        formatter: function() { 
          const sign = this.y >= 0 ? '+' : ''
          return `<b>${this.point.category}</b>: ${sign}${this.y.toFixed(2)} %` 
        } 
      },
      plotOptions: { 
        bar: { 
          borderColor: colors.barBorderColor,
          borderWidth: 1,
          dataLabels: { 
            enabled: true, 
            style: { color: colors.textColor }, 
            formatter: function() { 
              const sign = this.y >= 0 ? '+' : ''
              return sign + this.y.toFixed(2) + ' %' 
            } 
          } 
        } 
      },
      legend: { enabled: false },
      series: [{ name: 'Frametime Difference', data: sortedPercentages, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })
  } else if (comparisonMode.value === 'numbers-diff') {

    // Numbers diff mode - show absolute differences from baseline
    const differenceData = frametimeAverages.map(ft => ft - baselineFrametime)

    const sortedData = stats.map((s, index) => ({
      label: s.label,
      difference: differenceData[index]
    })).sort((a, b) => a.difference - b.difference)

    const sortedCategories = sortedData.map(item => item.label)
    const sortedDifferences = sortedData.map(item => item.difference)

    // Determine min/max for y-axis with some padding
    const minDiff = Math.min(...sortedDifferences)
    const maxDiff = Math.max(...sortedDifferences)
    const padding = Math.max(Math.abs(minDiff), Math.abs(maxDiff)) * 0.1
    const yAxisMin = minDiff - padding
    const yAxisMax = maxDiff + padding

    Highcharts.chart(frametimeAvgChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'Avg Frametime comparison' },
      subtitle: { ...chartOpts.subtitle, text: 'Less is better' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { 
        ...chartOpts.yAxis, 
        min: yAxisMin,
        max: yAxisMax,
        title: { text: 'Difference (ms)', align: 'high', style: { color: colors.textColor } },
        plotLines: [{
          value: 0,
          color: colors.lineColor,
          width: 2,
          zIndex: 4
        }]
      },
      tooltip: { 
        ...chartOpts.tooltip, 
        valueSuffix: ' ms', 
        formatter: function() { 
          const sign = this.y >= 0 ? '+' : ''
          return `<b>${this.point.category}</b>: ${sign}${this.y.toFixed(2)} ms` 
        } 
      },
      plotOptions: { 
        bar: { 
          borderColor: colors.barBorderColor,
          borderWidth: 1,
          dataLabels: { 
            enabled: true, 
            style: { color: colors.textColor }, 
            formatter: function() { 
              const sign = this.y >= 0 ? '+' : ''
              return sign + this.y.toFixed(2) + ' ms' 
            } 
          } 
        } 
      },
      legend: { enabled: false },
      series: [{ name: 'Frametime Difference', data: sortedDifferences, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })
  }
}

function getLineChartOptions(title, description, unit, maxY = null) {
  const chartOpts = commonChartOptions.value
  return {
    ...chartOpts,
    chart: { ...chartOpts.chart, type: 'line', zooming: { type: 'x' } },
    title: { ...chartOpts.title, text: title },
    subtitle: { ...chartOpts.subtitle, text: description },
    xAxis: { ...chartOpts.xAxis, labels: { enabled: false } },
    yAxis: { ...chartOpts.yAxis, max: maxY, labels: { ...chartOpts.yAxis.labels, formatter: function() { return this.value.toFixed(2) + ' ' + unit } } },
    tooltip: { ...chartOpts.tooltip, pointFormat: `<span style="color:{series.color}">{series.name}</span>: <b>{point.y:.2f} ${unit}</b><br/>` },
    plotOptions: { line: { marker: { enabled: false, symbol: 'circle', radius: 1.5, states: { hover: { enabled: true } } }, lineWidth: 1 } },
    legend: { ...chartOpts.legend, enabled: true },
    series: [],
    exporting: { buttons: { contextButton: { menuItems: ['viewFullscreen', 'printChart', 'separator', 'downloadPNG', 'downloadJPEG', 'downloadPDF', 'downloadSVG', 'separator', 'downloadCSV', 'downloadXLS'] } } }
  }
}

function getBarChartOptions(title, unit, maxY = null) {
  const colors = getThemeColors.value
  const chartOpts = commonChartOptions.value
  return {
    ...chartOpts,
    chart: { ...chartOpts.chart, type: 'bar' },
    title: { ...chartOpts.title, text: title },
    xAxis: { ...chartOpts.xAxis, categories: [], title: { text: null } },
    yAxis: { ...chartOpts.yAxis, min: 0, max: maxY, title: { text: unit, align: 'high', style: { color: colors.textColor } }, labels: { ...chartOpts.yAxis.labels, formatter: function() { return this.value.toFixed(2) + ' ' + unit } } },
    tooltip: { ...chartOpts.tooltip, valueSuffix: ' ' + unit, formatter: function() { return `<b>${this.point.category}</b>: ${this.y.toFixed(2)} ${unit}` } },
    plotOptions: { 
      bar: { 
        borderColor: colors.barBorderColor,
        borderWidth: 1,
        dataLabels: { enabled: true, style: { color: colors.textColor }, formatter: function() { return this.y.toFixed(2) + ' ' + unit } } 
      } 
    },
    legend: { enabled: false },
    series: []
  }
}

function createChart(element, title, description, unit, dataArrays, maxY = null) {
  if (!element || !dataArrays || dataArrays.length === 0) return
  
  const options = getLineChartOptions(title, description, unit, maxY)
  // Decimate data only for line chart rendering to improve performance
  // Statistics are calculated from full data before this function is called
  const chartColors = Highcharts.getOptions().colors
  options.series = dataArrays.map((dataArray, index) => ({
    name: dataArray.label,
    data: decimateForLineChart(dataArray.data || [], 2000),
    color: chartColors[index % chartColors.length]
  }))
  
  Highcharts.chart(element, options)
}

function createBarChart(element, title, unit, categories, data, chartColors, maxY = null) {
  if (!element || !categories || !data) return
  
  const options = getBarChartOptions(title, unit, maxY)
  options.xAxis.categories = categories
  options.series = [{ name: title, data: data, colorByPoint: true, colors: chartColors }]
  
  Highcharts.chart(element, options)
}

// Computed properties to cache data arrays - only recalculated when benchmarkData changes
const dataArrays = computed(() => {
  if (!props.benchmarkData || props.benchmarkData.length === 0) {
    return {
      fpsDataArrays: [],
      frameTimeDataArrays: [],
      cpuLoadDataArrays: [],
      gpuLoadDataArrays: [],
      cpuTempDataArrays: [],
      cpuPowerDataArrays: [],
      gpuTempDataArrays: [],
      gpuCoreClockDataArrays: [],
      gpuMemClockDataArrays: [],
      gpuVRAMUsedDataArrays: [],
      gpuPowerDataArrays: [],
      ramUsedDataArrays: [],
      swapUsedDataArrays: []
    }
  }
  
  // Processed data structure: d.series.FPS = [[x1, y1], [x2, y2], ...]
  // We only need the Y values for compatibility with existing chart code
  const extractY = (series) => series.map(point => point[1])
  
  return {
    fpsDataArrays: props.benchmarkData.map(d => ({ label: d.label, data: extractY(d.series?.FPS || []) })),
    frameTimeDataArrays: props.benchmarkData.map(d => ({ label: d.label, data: extractY(d.series?.FrameTime || []) })),
    cpuLoadDataArrays: props.benchmarkData.map(d => ({ label: d.label, data: extractY(d.series?.CPULoad || []) })),
    gpuLoadDataArrays: props.benchmarkData.map(d => ({ label: d.label, data: extractY(d.series?.GPULoad || []) })),
    cpuTempDataArrays: props.benchmarkData.map(d => ({ label: d.label, data: extractY(d.series?.CPUTemp || []) })),
    cpuPowerDataArrays: props.benchmarkData.map(d => ({ label: d.label, data: extractY(d.series?.CPUPower || []) })),
    gpuTempDataArrays: props.benchmarkData.map(d => ({ label: d.label, data: extractY(d.series?.GPUTemp || []) })),
    gpuCoreClockDataArrays: props.benchmarkData.map(d => ({ label: d.label, data: extractY(d.series?.GPUCoreClock || []) })),
    gpuMemClockDataArrays: props.benchmarkData.map(d => ({ label: d.label, data: extractY(d.series?.GPUMemClock || []) })),
    gpuVRAMUsedDataArrays: props.benchmarkData.map(d => ({ label: d.label, data: extractY(d.series?.GPUVRAMUsed || []) })),
    gpuPowerDataArrays: props.benchmarkData.map(d => ({ label: d.label, data: extractY(d.series?.GPUPower || []) })),
    ramUsedDataArrays: props.benchmarkData.map(d => ({ label: d.label, data: extractY(d.series?.RAMUsed || []) })),
    swapUsedDataArrays: props.benchmarkData.map(d => ({ label: d.label, data: extractY(d.series?.SwapUsed || []) }))
  }
})

// Computed properties using PRE-CALCULATED statistics from FULL data
// Statistics are calculated during incremental loading from full datasets (before downsampling)
// This ensures 100% accuracy for bar charts and percentile panels
const fpsStats = computed(() => {
  if (!props.benchmarkData || props.benchmarkData.length === 0) return null
  
  // Select stats based on calculation method
  const statsKey = appStore.calculationMethod === 'mangohud-threshold' ? 'statsMangoHud' : 'stats'
  
  return props.benchmarkData.map((run) => {
    const stats = run[statsKey]?.FPS || { min: 0, max: 0, avg: 0, p01: 0, p97: 0, density: [] }
    const seriesData = dataArrays.value.fpsDataArrays.find(d => d.label === run.label)?.data || []
    
    return {
      label: run.label,
      data: seriesData, // Downsampled data for line charts ONLY
      min: stats.p01,  // Use pre-calculated 1st percentile from FULL data
      avg: stats.avg,  // Use pre-calculated average from FULL data  
      max: stats.p97,  // Use pre-calculated 97th percentile from FULL data
      stddev: stats.stddev || 0,  // Use pre-calculated stddev from FULL data
      variance: stats.variance || 0,  // Use pre-calculated variance from FULL data
      // Use pre-calculated density from FULL data (calculated during download from all points)
      densityData: stats.density
    }
  })
})

// Computed properties using PRE-CALCULATED statistics from FULL data
const frametimeStats = computed(() => {
  if (!props.benchmarkData || props.benchmarkData.length === 0) return null
  
  // Select stats based on calculation method
  const statsKey = appStore.calculationMethod === 'mangohud-threshold' ? 'statsMangoHud' : 'stats'
  
  return props.benchmarkData.map((run) => {
    const stats = run[statsKey]?.FrameTime || { min: 0, max: 0, avg: 0, p01: 0, p97: 0, density: [] }
    const seriesData = dataArrays.value.frameTimeDataArrays.find(d => d.label === run.label)?.data || []
    
    return {
      label: run.label,
      data: seriesData, // Downsampled data for line charts ONLY
      min: stats.p01,  // Use pre-calculated 1st percentile from FULL data
      avg: stats.avg,  // Use pre-calculated average from FULL data
      max: stats.p97,  // Use pre-calculated 97th percentile from FULL data
      stddev: stats.stddev || 0,  // Use pre-calculated stddev from FULL data
      variance: stats.variance || 0,  // Use pre-calculated variance from FULL data
      // Use pre-calculated density from FULL data (calculated during download from all points)
      densityData: stats.density
    }
  })
})

// Computed properties using PRE-CALCULATED averages from FULL data for summary charts
const summaryStats = computed(() => {
  if (!props.benchmarkData || props.benchmarkData.length === 0) return null
  
  // Select stats based on calculation method
  const statsKey = appStore.calculationMethod === 'mangohud-threshold' ? 'statsMangoHud' : 'stats'
  
  return {
    fpsAverages: props.benchmarkData.map(run => run[statsKey]?.FPS?.avg || 0),
    frametimeAverages: props.benchmarkData.map(run => run[statsKey]?.FrameTime?.avg || 0),
    cpuLoadAverages: props.benchmarkData.map(run => run[statsKey]?.CPULoad?.avg || 0),
    gpuLoadAverages: props.benchmarkData.map(run => run[statsKey]?.GPULoad?.avg || 0),
    gpuCoreClockAverages: props.benchmarkData.map(run => run[statsKey]?.GPUCoreClock?.avg || 0),
    gpuMemClockAverages: props.benchmarkData.map(run => run[statsKey]?.GPUMemClock?.avg || 0),
    cpuPowerAverages: props.benchmarkData.map(run => run[statsKey]?.CPUPower?.avg || 0),
    gpuPowerAverages: props.benchmarkData.map(run => run[statsKey]?.GPUPower?.avg || 0)
  }
})

function renderFPSTab() {
  if (!props.benchmarkData || props.benchmarkData.length === 0) return
  
  const { fpsDataArrays } = dataArrays.value
  const stats = fpsStats.value
  
  // Create FPS charts
  createChart(fpsChart2.value, 'FPS', 'More is better', 'fps', fpsDataArrays)

  // FPS Min/Max/Avg chart
  if (fpsMinMaxAvgChart.value && stats) {
    const colors = getThemeColors.value
    const chartOpts = commonChartOptions.value
    const categories = stats.map(s => s.label)
    const minFPSData = stats.map(s => s.min)
    const avgFPSData = stats.map(s => s.avg)
    const maxFPSData = stats.map(s => s.max)

    Highcharts.chart(fpsMinMaxAvgChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'Min/Avg/Max FPS' },
      subtitle: { ...chartOpts.subtitle, text: 'More is better' },
      xAxis: { ...chartOpts.xAxis, categories: categories },
      yAxis: { ...chartOpts.yAxis, title: { text: 'FPS', align: 'high', style: { color: colors.textColor } } },
      tooltip: { ...chartOpts.tooltip, valueSuffix: ' FPS', formatter: function() { return `<b>${this.series.name}</b>: ${this.y.toFixed(2)} FPS` } },
      plotOptions: { bar: { borderColor: colors.barBorderColor, borderWidth: 1, dataLabels: { enabled: true, style: { color: colors.textColor }, formatter: function() { return this.y.toFixed(2) + ' fps' } } } },
      legend: { ...chartOpts.legend, reversed: true, enabled: true },
      series: [
        { name: '97th', data: maxFPSData, color: '#00FF00' },
        { name: 'AVG', data: avgFPSData, color: '#0000FF' },
        { name: '1%', data: minFPSData, color: '#FF0000' }
      ]
    })
  }

  // FPS Density chart
  if (fpsDensityChart.value && stats) {
    const colors = getThemeColors.value
    const chartOpts = commonChartOptions.value
    const densityData = stats.map(s => ({
      name: s.label,
      data: s.densityData
    }))

    Highcharts.chart(fpsDensityChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'areaspline' },
      title: { ...chartOpts.title, text: 'FPS Density' },
      xAxis: { ...chartOpts.xAxis, title: { text: 'FPS', style: { color: colors.textColor } }, labels: { style: { color: colors.textColor } } },
      tooltip: { ...chartOpts.tooltip, shared: true, formatter: function() { return `<b>${this.points[0].series.name}</b>: ${this.points[0].y} points at ~${Math.round(this.points[0].x)} FPS` } },
      plotOptions: { areaspline: { fillOpacity: 0.5, marker: { enabled: false } } },
      legend: { ...chartOpts.legend, enabled: true },
      series: densityData
    })
  }

  // FPS Average comparison chart - render via separate function
  renderFPSComparisonChart()

  // FPS Standard Deviation and Variance charts - render via separate functions
  renderFPSStddevChart()
  renderFPSVarianceChart()
}

// FPS Standard Deviation chart - separate function to allow toggling with comparison mode
function renderFPSStddevChart() {
  if (!fpsStddevChart.value || !props.benchmarkData || props.benchmarkData.length === 0) return
  
  const stats = fpsStats.value
  if (!stats || stats.length === 0) return
  
  const colors = getThemeColors.value
  const chartOpts = commonChartOptions.value
  
  if (comparisonMode.value === 'percentage') {
    // Percentage mode - show percentage difference from baseline
    const stddevValues = stats.map(s => s.stddev)
    
    // Determine baseline stddev
    let baselineStddev
    const fpsAverages = stats.map(s => s.avg)
    if (fpsBaselineIndex.value !== null && fpsBaselineIndex.value >= 0 && fpsBaselineIndex.value < stddevValues.length) {
      baselineStddev = stddevValues[fpsBaselineIndex.value]
    } else {
      // Auto mode: use stddev of slowest (min FPS)
      const minFPSIndex = fpsAverages.indexOf(Math.min(...fpsAverages))
      baselineStddev = stddevValues[minFPSIndex]
    }
    
    // Calculate percentage differences from baseline
    const percentageData = stddevValues.map(val => ((val - baselineStddev) / baselineStddev) * 100)
    
    const sortedData = stats.map((s, index) => ({
      label: s.label,
      percentage: percentageData[index]
    })).sort((a, b) => a.percentage - b.percentage)
    
    const sortedCategories = sortedData.map(item => item.label)
    const sortedPercentages = sortedData.map(item => item.percentage)
    
    // Determine min/max for y-axis with some padding
    const minPercentage = Math.min(...sortedPercentages)
    const maxPercentage = Math.max(...sortedPercentages)
    const padding = Math.max(Math.abs(minPercentage), Math.abs(maxPercentage)) * 0.1
    const yAxisMin = minPercentage - padding
    const yAxisMax = maxPercentage + padding
    
    Highcharts.chart(fpsStddevChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'FPS Standard Deviation in %' },
      subtitle: { ...chartOpts.subtitle, text: 'Measures FPS consistency. Less is better.' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { 
        ...chartOpts.yAxis, 
        min: yAxisMin,
        max: yAxisMax,
        title: { text: 'Difference (%)', align: 'high', style: { color: colors.textColor } },
        plotLines: [{
          value: 0,
          color: colors.lineColor,
          width: 2,
          zIndex: 4
        }]
      },
      tooltip: { 
        ...chartOpts.tooltip, 
        formatter: function() { 
          const sign = this.y >= 0 ? '+' : ''
          return `<b>${this.point.category}</b>: ${sign}${this.y.toFixed(2)} %` 
        } 
      },
      plotOptions: { 
        bar: { 
          borderColor: colors.barBorderColor, 
          borderWidth: 1, 
          dataLabels: { 
            enabled: true, 
            style: { color: colors.textColor }, 
            formatter: function() { 
              const sign = this.y >= 0 ? '+' : ''
              return sign + this.y.toFixed(2) + ' %' 
            } 
          } 
        } 
      },
      legend: { enabled: false },
      series: [{ name: 'Std. Dev. Difference', data: sortedPercentages, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })
  } else if (comparisonMode.value === 'numbers-diff') {

    // Numbers diff mode - show absolute differences from baseline
    const stddevValues = stats.map(s => s.stddev)
    
    // Determine baseline stddev
    let baselineStddev
    const fpsAverages = stats.map(s => s.avg)
    if (fpsBaselineIndex.value !== null && fpsBaselineIndex.value >= 0 && fpsBaselineIndex.value < stddevValues.length) {
      baselineStddev = stddevValues[fpsBaselineIndex.value]
    } else {
      // Auto mode: use stddev of slowest (min FPS)
      const minFPSIndex = fpsAverages.indexOf(Math.min(...fpsAverages))
      baselineStddev = stddevValues[minFPSIndex]
    }
    
    // Calculate absolute differences from baseline
    const differenceData = stddevValues.map(val => val - baselineStddev)
    
    const sortedData = stats.map((s, index) => ({
      label: s.label,
      difference: differenceData[index]
    })).sort((a, b) => a.difference - b.difference)
    
    const sortedCategories = sortedData.map(item => item.label)
    const sortedDifferences = sortedData.map(item => item.difference)
    
    // Determine min/max for y-axis with some padding
    const minDiff = Math.min(...sortedDifferences)
    const maxDiff = Math.max(...sortedDifferences)
    const padding = Math.max(Math.abs(minDiff), Math.abs(maxDiff)) * 0.1
    const yAxisMin = minDiff - padding
    const yAxisMax = maxDiff + padding

    Highcharts.chart(fpsStddevChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'FPS Standard Deviation' },
      subtitle: { ...chartOpts.subtitle, text: 'Measures FPS consistency. Less is better.' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { 
        ...chartOpts.yAxis, 
        min: yAxisMin,
        max: yAxisMax,
        title: { text: 'Difference', align: 'high', style: { color: colors.textColor } },
        plotLines: [{
          value: 0,
          color: colors.lineColor,
          width: 2,
          zIndex: 4
        }]
      },
      tooltip: { 
        ...chartOpts.tooltip, 
        formatter: function() { 
          const sign = this.y >= 0 ? '+' : ''
          return `<b>${this.point.category}</b>: ${sign}${this.y.toFixed(2)}` 
        } 
      },
      plotOptions: { 
        bar: { 
          borderColor: colors.barBorderColor, 
          borderWidth: 1, 
          dataLabels: { 
            enabled: true, 
            style: { color: colors.textColor }, 
            formatter: function() { 
              const sign = this.y >= 0 ? '+' : ''
              return sign + this.y.toFixed(2) 
            } 
          } 
        } 
      },
      legend: { enabled: false },
      series: [{ name: 'Std. Dev. Difference', data: sortedDifferences, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })  } else {
    // Numbers mode - show absolute stddev values
    const sortedData = stats.map((s, index) => ({
      label: s.label,
      value: stats[index].stddev
    })).sort((a, b) => a.value - b.value)
    
    const sortedCategories = sortedData.map(item => item.label)
    const sortedValues = sortedData.map(item => item.value)

    Highcharts.chart(fpsStddevChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'FPS Standard Deviation' },
      subtitle: { ...chartOpts.subtitle, text: 'Measures FPS consistency. Less is better.' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { ...chartOpts.yAxis, title: { text: 'Std. Dev.', align: 'high', style: { color: colors.textColor } } },
      tooltip: { ...chartOpts.tooltip, formatter: function() { return `<b>${this.point.category}</b>: ${this.y.toFixed(2)}` } },
      plotOptions: { bar: { borderColor: colors.barBorderColor, borderWidth: 1, dataLabels: { enabled: true, style: { color: colors.textColor }, formatter: function() { return this.y.toFixed(2) } } } },
      legend: { enabled: false },
      series: [{ name: 'Std. Dev.', data: sortedValues, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })
  }
}

// FPS Variance chart - separate function to allow toggling with comparison mode
function renderFPSVarianceChart() {
  if (!fpsVarianceChart.value || !props.benchmarkData || props.benchmarkData.length === 0) return
  
  const stats = fpsStats.value
  if (!stats || stats.length === 0) return
  
  const colors = getThemeColors.value
  const chartOpts = commonChartOptions.value
  
  if (comparisonMode.value === 'percentage') {
    // Percentage mode - show percentage difference from baseline
    const varianceValues = stats.map(s => s.variance)
    
    // Determine baseline variance
    let baselineVariance
    const fpsAverages = stats.map(s => s.avg)
    if (fpsBaselineIndex.value !== null && fpsBaselineIndex.value >= 0 && fpsBaselineIndex.value < varianceValues.length) {
      baselineVariance = varianceValues[fpsBaselineIndex.value]
    } else {
      // Auto mode: use variance of slowest (min FPS)
      const minFPSIndex = fpsAverages.indexOf(Math.min(...fpsAverages))
      baselineVariance = varianceValues[minFPSIndex]
    }
    
    // Calculate percentage differences from baseline
    const percentageData = varianceValues.map(val => ((val - baselineVariance) / baselineVariance) * 100)
    
    const sortedData = stats.map((s, index) => ({
      label: s.label,
      percentage: percentageData[index]
    })).sort((a, b) => a.percentage - b.percentage)
    
    const sortedCategories = sortedData.map(item => item.label)
    const sortedPercentages = sortedData.map(item => item.percentage)
    
    // Determine min/max for y-axis with some padding
    const minPercentage = Math.min(...sortedPercentages)
    const maxPercentage = Math.max(...sortedPercentages)
    const padding = Math.max(Math.abs(minPercentage), Math.abs(maxPercentage)) * 0.1
    const yAxisMin = minPercentage - padding
    const yAxisMax = maxPercentage + padding
    
    Highcharts.chart(fpsVarianceChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'FPS Variance in %' },
      subtitle: { ...chartOpts.subtitle, text: 'Measures FPS spread. Less is better.' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { 
        ...chartOpts.yAxis, 
        min: yAxisMin,
        max: yAxisMax,
        title: { text: 'Difference (%)', align: 'high', style: { color: colors.textColor } },
        plotLines: [{
          value: 0,
          color: colors.lineColor,
          width: 2,
          zIndex: 4
        }]
      },
      tooltip: { 
        ...chartOpts.tooltip, 
        formatter: function() { 
          const sign = this.y >= 0 ? '+' : ''
          return `<b>${this.point.category}</b>: ${sign}${this.y.toFixed(2)} %` 
        } 
      },
      plotOptions: { 
        bar: { 
          borderColor: colors.barBorderColor, 
          borderWidth: 1, 
          dataLabels: { 
            enabled: true, 
            style: { color: colors.textColor }, 
            formatter: function() { 
              const sign = this.y >= 0 ? '+' : ''
              return sign + this.y.toFixed(2) + ' %' 
            } 
          } 
        } 
      },
      legend: { enabled: false },
      series: [{ name: 'Variance Difference', data: sortedPercentages, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })
  } else if (comparisonMode.value === 'numbers-diff') {

    // Numbers diff mode - show absolute differences from baseline
    const varianceValues = stats.map(s => s.variance)
    
    // Determine baseline variance
    let baselineVariance
    const fpsAverages = stats.map(s => s.avg)
    if (fpsBaselineIndex.value !== null && fpsBaselineIndex.value >= 0 && fpsBaselineIndex.value < varianceValues.length) {
      baselineVariance = varianceValues[fpsBaselineIndex.value]
    } else {
      // Auto mode: use variance of slowest (min FPS)
      const minFPSIndex = fpsAverages.indexOf(Math.min(...fpsAverages))
      baselineVariance = varianceValues[minFPSIndex]
    }
    
    // Calculate absolute differences from baseline
    const differenceData = varianceValues.map(val => val - baselineVariance)
    
    const sortedData = stats.map((s, index) => ({
      label: s.label,
      difference: differenceData[index]
    })).sort((a, b) => a.difference - b.difference)
    
    const sortedCategories = sortedData.map(item => item.label)
    const sortedDifferences = sortedData.map(item => item.difference)
    
    // Determine min/max for y-axis with some padding
    const minDiff = Math.min(...sortedDifferences)
    const maxDiff = Math.max(...sortedDifferences)
    const padding = Math.max(Math.abs(minDiff), Math.abs(maxDiff)) * 0.1
    const yAxisMin = minDiff - padding
    const yAxisMax = maxDiff + padding

    Highcharts.chart(fpsVarianceChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'FPS Variance' },
      subtitle: { ...chartOpts.subtitle, text: 'Measures FPS spread. Less is better.' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { 
        ...chartOpts.yAxis, 
        min: yAxisMin,
        max: yAxisMax,
        title: { text: 'Difference', align: 'high', style: { color: colors.textColor } },
        plotLines: [{
          value: 0,
          color: colors.lineColor,
          width: 2,
          zIndex: 4
        }]
      },
      tooltip: { 
        ...chartOpts.tooltip, 
        formatter: function() { 
          const sign = this.y >= 0 ? '+' : ''
          return `<b>${this.point.category}</b>: ${sign}${this.y.toFixed(2)}` 
        } 
      },
      plotOptions: { 
        bar: { 
          borderColor: colors.barBorderColor, 
          borderWidth: 1, 
          dataLabels: { 
            enabled: true, 
            style: { color: colors.textColor }, 
            formatter: function() { 
              const sign = this.y >= 0 ? '+' : ''
              return sign + this.y.toFixed(2) 
            } 
          } 
        } 
      },
      legend: { enabled: false },
      series: [{ name: 'Variance Difference', data: sortedDifferences, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })  } else {
    // Numbers mode - show absolute variance values
    const sortedData = stats.map((s, index) => ({
      label: s.label,
      value: stats[index].variance
    })).sort((a, b) => a.value - b.value)
    
    const sortedCategories = sortedData.map(item => item.label)
    const sortedValues = sortedData.map(item => item.value)

    Highcharts.chart(fpsVarianceChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'FPS Variance' },
      subtitle: { ...chartOpts.subtitle, text: 'Measures FPS spread. Less is better.' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { ...chartOpts.yAxis, title: { text: 'Variance', align: 'high', style: { color: colors.textColor } } },
      tooltip: { ...chartOpts.tooltip, formatter: function() { return `<b>${this.point.category}</b>: ${this.y.toFixed(2)}` } },
      plotOptions: { bar: { borderColor: colors.barBorderColor, borderWidth: 1, dataLabels: { enabled: true, style: { color: colors.textColor }, formatter: function() { return this.y.toFixed(2) } } } },
      legend: { enabled: false },
      series: [{ name: 'Variance', data: sortedValues, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })
  }
}

function renderFrametimeTab() {
  if (!props.benchmarkData || props.benchmarkData.length === 0) return
  
  const { frameTimeDataArrays } = dataArrays.value
  const stats = frametimeStats.value
  
  createChart(frameTimeChart2.value, 'Frametime', 'Less is better', 'ms', frameTimeDataArrays)

  // Frametime Min/Max/Avg chart
  if (frametimeMinMaxAvgChart.value && stats) {
    const colors = getThemeColors.value
    const chartOpts = commonChartOptions.value
    const categories = stats.map(s => s.label)
    const minData = stats.map(s => s.min)
    const avgData = stats.map(s => s.avg)
    const maxData = stats.map(s => s.max)

    Highcharts.chart(frametimeMinMaxAvgChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'Min/Avg/Max Frametime' },
      subtitle: { ...chartOpts.subtitle, text: 'Less is better' },
      xAxis: { ...chartOpts.xAxis, categories: categories },
      yAxis: { ...chartOpts.yAxis, title: { text: 'Frametime (ms)', align: 'high', style: { color: colors.textColor } } },
      tooltip: { ...chartOpts.tooltip, valueSuffix: ' ms', formatter: function() { return `<b>${this.series.name}</b>: ${this.y.toFixed(2)} ms` } },
      plotOptions: { bar: { borderColor: colors.barBorderColor, borderWidth: 1, dataLabels: { enabled: true, style: { color: colors.textColor }, formatter: function() { return this.y.toFixed(2) + ' ms' } } } },
      legend: { ...chartOpts.legend, reversed: true, enabled: true },
      series: [
        { name: '97th', data: maxData, color: '#00FF00' },
        { name: 'AVG', data: avgData, color: '#0000FF' },
        { name: '1%', data: minData, color: '#FF0000' }
      ]
    })
  }

  // Frametime Density chart
  if (frametimeDensityChart.value && stats) {
    const colors = getThemeColors.value
    const chartOpts = commonChartOptions.value
    const densityData = stats.map(s => ({
      name: s.label,
      data: s.densityData
    }))

    Highcharts.chart(frametimeDensityChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'areaspline' },
      title: { ...chartOpts.title, text: 'Frametime Density' },
      xAxis: { ...chartOpts.xAxis, title: { text: 'Frametime (ms)', style: { color: colors.textColor } }, labels: { style: { color: colors.textColor } } },
      tooltip: { ...chartOpts.tooltip, shared: true, formatter: function() { return `<b>${this.points[0].series.name}</b>: ${this.points[0].y} points at ~${Math.round(this.points[0].x)} ms` } },
      plotOptions: { areaspline: { fillOpacity: 0.5, marker: { enabled: false } } },
      legend: { ...chartOpts.legend, enabled: true },
      series: densityData
    })
  }

  // Frametime Average comparison chart - render via separate function
  renderFrametimeComparisonChart()

  // Frametime Standard Deviation and Variance charts - render via separate functions
  renderFrametimeStddevChart()
  renderFrametimeVarianceChart()
}

// Frametime Standard Deviation chart - separate function to allow toggling with comparison mode
function renderFrametimeStddevChart() {
  if (!frametimeStddevChart.value || !props.benchmarkData || props.benchmarkData.length === 0) return
  
  const stats = frametimeStats.value
  if (!stats || stats.length === 0) return
  
  const colors = getThemeColors.value
  const chartOpts = commonChartOptions.value
  
  if (comparisonMode.value === 'percentage') {
    // Percentage mode - show percentage difference from baseline
    const stddevValues = stats.map(s => s.stddev)
    
    // Determine baseline stddev
    let baselineStddev
    const frametimeAverages = stats.map(s => s.avg)
    if (frametimeBaselineIndex.value !== null && frametimeBaselineIndex.value >= 0 && frametimeBaselineIndex.value < stddevValues.length) {
      baselineStddev = stddevValues[frametimeBaselineIndex.value]
    } else {
      // Auto mode: use stddev of fastest (min frametime)
      const minFrametimeIndex = frametimeAverages.indexOf(Math.min(...frametimeAverages))
      baselineStddev = stddevValues[minFrametimeIndex]
    }
    
    // Calculate percentage differences from baseline
    const percentageData = stddevValues.map(val => ((val - baselineStddev) / baselineStddev) * 100)
    
    const sortedData = stats.map((s, index) => ({
      label: s.label,
      percentage: percentageData[index]
    })).sort((a, b) => a.percentage - b.percentage)
    
    const sortedCategories = sortedData.map(item => item.label)
    const sortedPercentages = sortedData.map(item => item.percentage)
    
    // Determine min/max for y-axis with some padding
    const minPercentage = Math.min(...sortedPercentages)
    const maxPercentage = Math.max(...sortedPercentages)
    const padding = Math.max(Math.abs(minPercentage), Math.abs(maxPercentage)) * 0.1
    const yAxisMin = minPercentage - padding
    const yAxisMax = maxPercentage + padding
    
    Highcharts.chart(frametimeStddevChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'Frametime Standard Deviation in %' },
      subtitle: { ...chartOpts.subtitle, text: 'Measures Frametime consistency. Less is better.' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { 
        ...chartOpts.yAxis, 
        min: yAxisMin,
        max: yAxisMax,
        title: { text: 'Difference (%)', align: 'high', style: { color: colors.textColor } },
        plotLines: [{
          value: 0,
          color: colors.lineColor,
          width: 2,
          zIndex: 4
        }]
      },
      tooltip: { 
        ...chartOpts.tooltip, 
        formatter: function() { 
          const sign = this.y >= 0 ? '+' : ''
          return `<b>${this.point.category}</b>: ${sign}${this.y.toFixed(2)} %` 
        } 
      },
      plotOptions: { 
        bar: { 
          borderColor: colors.barBorderColor, 
          borderWidth: 1, 
          dataLabels: { 
            enabled: true, 
            style: { color: colors.textColor }, 
            formatter: function() { 
              const sign = this.y >= 0 ? '+' : ''
              return sign + this.y.toFixed(2) + ' %' 
            } 
          } 
        } 
      },
      legend: { enabled: false },
      series: [{ name: 'Std. Dev. Difference', data: sortedPercentages, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })
  } else if (comparisonMode.value === 'numbers-diff') {

    // Numbers diff mode - show absolute differences from baseline
    const stddevValues = stats.map(s => s.stddev)
    
    // Determine baseline stddev
    let baselineStddev
    const frametimeAverages = stats.map(s => s.avg)
    if (frametimeBaselineIndex.value !== null && frametimeBaselineIndex.value >= 0 && frametimeBaselineIndex.value < stddevValues.length) {
      baselineStddev = stddevValues[frametimeBaselineIndex.value]
    } else {
      // Auto mode: use stddev of fastest (min frametime)
      const minFrametimeIndex = frametimeAverages.indexOf(Math.min(...frametimeAverages))
      baselineStddev = stddevValues[minFrametimeIndex]
    }
    
    // Calculate absolute differences from baseline
    const differenceData = stddevValues.map(val => val - baselineStddev)
    
    const sortedData = stats.map((s, index) => ({
      label: s.label,
      difference: differenceData[index]
    })).sort((a, b) => a.difference - b.difference)
    
    const sortedCategories = sortedData.map(item => item.label)
    const sortedDifferences = sortedData.map(item => item.difference)
    
    // Determine min/max for y-axis with some padding
    const minDiff = Math.min(...sortedDifferences)
    const maxDiff = Math.max(...sortedDifferences)
    const padding = Math.max(Math.abs(minDiff), Math.abs(maxDiff)) * 0.1
    const yAxisMin = minDiff - padding
    const yAxisMax = maxDiff + padding

    Highcharts.chart(frametimeStddevChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'Frametime Standard Deviation' },
      subtitle: { ...chartOpts.subtitle, text: 'Measures Frametime consistency. Less is better.' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { 
        ...chartOpts.yAxis, 
        min: yAxisMin,
        max: yAxisMax,
        title: { text: 'Difference (ms)', align: 'high', style: { color: colors.textColor } },
        plotLines: [{
          value: 0,
          color: colors.lineColor,
          width: 2,
          zIndex: 4
        }]
      },
      tooltip: { 
        ...chartOpts.tooltip, 
        formatter: function() { 
          const sign = this.y >= 0 ? '+' : ''
          return `<b>${this.point.category}</b>: ${sign}${this.y.toFixed(2)} ms` 
        } 
      },
      plotOptions: { 
        bar: { 
          borderColor: colors.barBorderColor, 
          borderWidth: 1, 
          dataLabels: { 
            enabled: true, 
            style: { color: colors.textColor }, 
            formatter: function() { 
              const sign = this.y >= 0 ? '+' : ''
              return sign + this.y.toFixed(2) + ' ms' 
            } 
          } 
        } 
      },
      legend: { enabled: false },
      series: [{ name: 'Std. Dev. Difference', data: sortedDifferences, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })  } else {
    // Numbers mode - show absolute stddev values
    const sortedData = stats.map((s, index) => ({
      label: s.label,
      value: stats[index].stddev
    })).sort((a, b) => a.value - b.value)
    
    const sortedCategories = sortedData.map(item => item.label)
    const sortedValues = sortedData.map(item => item.value)

    Highcharts.chart(frametimeStddevChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'Frametime Standard Deviation' },
      subtitle: { ...chartOpts.subtitle, text: 'Measures Frametime consistency. Less is better.' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { ...chartOpts.yAxis, title: { text: 'Std. Dev. (ms)', align: 'high', style: { color: colors.textColor } } },
      tooltip: { ...chartOpts.tooltip, formatter: function() { return `<b>${this.point.category}</b>: ${this.y.toFixed(2)} ms` } },
      plotOptions: { bar: { borderColor: colors.barBorderColor, borderWidth: 1, dataLabels: { enabled: true, style: { color: colors.textColor }, formatter: function() { return this.y.toFixed(2) + ' ms' } } } },
      legend: { enabled: false },
      series: [{ name: 'Std. Dev.', data: sortedValues, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })
  }
}

// Frametime Variance chart - separate function to allow toggling with comparison mode
function renderFrametimeVarianceChart() {
  if (!frametimeVarianceChart.value || !props.benchmarkData || props.benchmarkData.length === 0) return
  
  const stats = frametimeStats.value
  if (!stats || stats.length === 0) return
  
  const colors = getThemeColors.value
  const chartOpts = commonChartOptions.value
  
  if (comparisonMode.value === 'percentage') {
    // Percentage mode - show percentage difference from baseline
    const varianceValues = stats.map(s => s.variance)
    
    // Determine baseline variance
    let baselineVariance
    const frametimeAverages = stats.map(s => s.avg)
    if (frametimeBaselineIndex.value !== null && frametimeBaselineIndex.value >= 0 && frametimeBaselineIndex.value < varianceValues.length) {
      baselineVariance = varianceValues[frametimeBaselineIndex.value]
    } else {
      // Auto mode: use variance of fastest (min frametime)
      const minFrametimeIndex = frametimeAverages.indexOf(Math.min(...frametimeAverages))
      baselineVariance = varianceValues[minFrametimeIndex]
    }
    
    // Calculate percentage differences from baseline
    const percentageData = varianceValues.map(val => ((val - baselineVariance) / baselineVariance) * 100)
    
    const sortedData = stats.map((s, index) => ({
      label: s.label,
      percentage: percentageData[index]
    })).sort((a, b) => a.percentage - b.percentage)
    
    const sortedCategories = sortedData.map(item => item.label)
    const sortedPercentages = sortedData.map(item => item.percentage)
    
    // Determine min/max for y-axis with some padding
    const minPercentage = Math.min(...sortedPercentages)
    const maxPercentage = Math.max(...sortedPercentages)
    const padding = Math.max(Math.abs(minPercentage), Math.abs(maxPercentage)) * 0.1
    const yAxisMin = minPercentage - padding
    const yAxisMax = maxPercentage + padding
    
    Highcharts.chart(frametimeVarianceChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'Frametime Variance in %' },
      subtitle: { ...chartOpts.subtitle, text: 'Measures Frametime spread. Less is better.' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { 
        ...chartOpts.yAxis, 
        min: yAxisMin,
        max: yAxisMax,
        title: { text: 'Difference (%)', align: 'high', style: { color: colors.textColor } },
        plotLines: [{
          value: 0,
          color: colors.lineColor,
          width: 2,
          zIndex: 4
        }]
      },
      tooltip: { 
        ...chartOpts.tooltip, 
        formatter: function() { 
          const sign = this.y >= 0 ? '+' : ''
          return `<b>${this.point.category}</b>: ${sign}${this.y.toFixed(2)} %` 
        } 
      },
      plotOptions: { 
        bar: { 
          borderColor: colors.barBorderColor, 
          borderWidth: 1, 
          dataLabels: { 
            enabled: true, 
            style: { color: colors.textColor }, 
            formatter: function() { 
              const sign = this.y >= 0 ? '+' : ''
              return sign + this.y.toFixed(2) + ' %' 
            } 
          } 
        } 
      },
      legend: { enabled: false },
      series: [{ name: 'Variance Difference', data: sortedPercentages, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })
  } else if (comparisonMode.value === 'numbers-diff') {

    // Numbers diff mode - show absolute differences from baseline
    const varianceValues = stats.map(s => s.variance)
    
    // Determine baseline variance
    let baselineVariance
    const frametimeAverages = stats.map(s => s.avg)
    if (frametimeBaselineIndex.value !== null && frametimeBaselineIndex.value >= 0 && frametimeBaselineIndex.value < varianceValues.length) {
      baselineVariance = varianceValues[frametimeBaselineIndex.value]
    } else {
      // Auto mode: use variance of fastest (min frametime)
      const minFrametimeIndex = frametimeAverages.indexOf(Math.min(...frametimeAverages))
      baselineVariance = varianceValues[minFrametimeIndex]
    }
    
    // Calculate absolute differences from baseline
    const differenceData = varianceValues.map(val => val - baselineVariance)
    
    const sortedData = stats.map((s, index) => ({
      label: s.label,
      difference: differenceData[index]
    })).sort((a, b) => a.difference - b.difference)
    
    const sortedCategories = sortedData.map(item => item.label)
    const sortedDifferences = sortedData.map(item => item.difference)
    
    // Determine min/max for y-axis with some padding
    const minDiff = Math.min(...sortedDifferences)
    const maxDiff = Math.max(...sortedDifferences)
    const padding = Math.max(Math.abs(minDiff), Math.abs(maxDiff)) * 0.1
    const yAxisMin = minDiff - padding
    const yAxisMax = maxDiff + padding

    Highcharts.chart(frametimeVarianceChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'Frametime Variance' },
      subtitle: { ...chartOpts.subtitle, text: 'Measures Frametime spread. Less is better.' },
      xAxis: { ...chartOpts.xAxis, categories: sortedCategories },
      yAxis: { 
        ...chartOpts.yAxis, 
        min: yAxisMin,
        max: yAxisMax,
        title: { text: 'Difference (ms²)', align: 'high', style: { color: colors.textColor } },
        plotLines: [{
          value: 0,
          color: colors.lineColor,
          width: 2,
          zIndex: 4
        }]
      },
      tooltip: { 
        ...chartOpts.tooltip, 
        formatter: function() { 
          const sign = this.y >= 0 ? '+' : ''
          return `<b>${this.point.category}</b>: ${sign}${this.y.toFixed(2)} ms²` 
        } 
      },
      plotOptions: { 
        bar: { 
          borderColor: colors.barBorderColor, 
          borderWidth: 1, 
          dataLabels: { 
            enabled: true, 
            style: { color: colors.textColor }, 
            formatter: function() { 
              const sign = this.y >= 0 ? '+' : ''
              return sign + this.y.toFixed(2) + ' ms²' 
            } 
          } 
        } 
      },
      legend: { enabled: false },
      series: [{ name: 'Variance Difference', data: sortedDifferences, colorByPoint: true, colors: Highcharts.getOptions().colors }]
    })
  }
}

function renderSummaryTab() {
  if (!props.benchmarkData || props.benchmarkData.length === 0) return
  
  const arrays = dataArrays.value
  const stats = summaryStats.value
  
  if (!stats) return

  // Create summary bar charts using pre-calculated averages
  const chartColors = Highcharts.getOptions().colors
  createBarChart(fpsSummaryChart.value, 'Average FPS', 'fps', arrays.fpsDataArrays.map(d => d.label), stats.fpsAverages, chartColors)
  createBarChart(frametimeSummaryChart.value, 'Average Frametime', 'ms', arrays.frameTimeDataArrays.map(d => d.label), stats.frametimeAverages, chartColors)
  createBarChart(cpuLoadSummaryChart.value, 'Average CPU Load', '%', arrays.cpuLoadDataArrays.map(d => d.label), stats.cpuLoadAverages, chartColors, 100)
  createBarChart(gpuLoadSummaryChart.value, 'Average GPU Load', '%', arrays.gpuLoadDataArrays.map(d => d.label), stats.gpuLoadAverages, chartColors, 100)
  createBarChart(gpuCoreClockSummaryChart.value, 'Average GPU Core Clock', 'MHz', arrays.gpuCoreClockDataArrays.map(d => d.label), stats.gpuCoreClockAverages, chartColors)
  createBarChart(gpuMemClockSummaryChart.value, 'Average GPU Memory Clock', 'MHz', arrays.gpuMemClockDataArrays.map(d => d.label), stats.gpuMemClockAverages, chartColors)
  createBarChart(cpuPowerSummaryChart.value, 'Average CPU Power', 'W', arrays.cpuPowerDataArrays.map(d => d.label), stats.cpuPowerAverages, chartColors)
  createBarChart(gpuPowerSummaryChart.value, 'Average GPU Power', 'W', arrays.gpuPowerDataArrays.map(d => d.label), stats.gpuPowerAverages, chartColors)
}

function renderMoreMetricsTab() {
  if (!props.benchmarkData || props.benchmarkData.length === 0) return
  
  const {
    fpsDataArrays,
    frameTimeDataArrays,
    cpuLoadDataArrays,
    gpuLoadDataArrays,
    cpuTempDataArrays,
    cpuPowerDataArrays,
    gpuTempDataArrays,
    gpuCoreClockDataArrays,
    gpuMemClockDataArrays,
    gpuVRAMUsedDataArrays,
    gpuPowerDataArrays,
    ramUsedDataArrays,
    swapUsedDataArrays
  } = dataArrays.value

  // Create line charts
  createChart(fpsChart.value, 'FPS', 'More is better', 'fps', fpsDataArrays)
  createChart(frameTimeChart.value, 'Frametime', 'Less is better', 'ms', frameTimeDataArrays)
  createChart(cpuLoadChart.value, 'CPU Load', '', '%', cpuLoadDataArrays, 100)
  createChart(gpuLoadChart.value, 'GPU Load', '', '%', gpuLoadDataArrays, 100)
  createChart(cpuTempChart.value, 'CPU Temperature', '', '°C', cpuTempDataArrays)
  createChart(cpuPowerChart.value, 'CPU Power', '', 'W', cpuPowerDataArrays)
  createChart(gpuTempChart.value, 'GPU Temperature', '', '°C', gpuTempDataArrays)
  createChart(gpuCoreClockChart.value, 'GPU Core Clock', '', 'MHz', gpuCoreClockDataArrays)
  createChart(gpuMemClockChart.value, 'GPU Memory Clock', '', 'MHz', gpuMemClockDataArrays)
  createChart(gpuVRAMUsedChart.value, 'GPU VRAM Usage', '', 'GB', gpuVRAMUsedDataArrays)
  createChart(gpuPowerChart.value, 'GPU Power', '', 'W', gpuPowerDataArrays)
  createChart(ramUsedChart.value, 'RAM Usage', '', 'GB', ramUsedDataArrays)
  createChart(swapUsedChart.value, 'SWAP Usage', '', 'GB', swapUsedDataArrays)
}

// Handle tab clicks
function handleTabClick(tabName) {
  // Check if already rendered
  if (renderedTabs.value[tabName]) {
    return
  }

  // Use nextTick to ensure DOM is ready
  nextTick(() => {
    switch (tabName) {
      case 'fps':
        renderFPSTab()
        break
      case 'frametime':
        renderFrametimeTab()
        break
      case 'summary':
        renderSummaryTab()
        break
      case 'more-metrics':
        renderMoreMetricsTab()
        break
      default:
        console.warn(`Unknown tab name: ${tabName}`)
        return
    }
    renderedTabs.value[tabName] = true
  })
}

// Shared function to re-render all rendered tabs
// Used by theme changes, DPI changes, and calculation method changes
function reRenderAllTabs() {
  nextTick(() => {
    if (renderedTabs.value.fps) {
      renderFPSTab()
    }
    if (renderedTabs.value.frametime) {
      renderFrametimeTab()
    }
    if (renderedTabs.value.summary) {
      renderSummaryTab()
    }
    if (renderedTabs.value['more-metrics']) {
      renderMoreMetricsTab()
    }
  })
}

// Handle calculation method change
function setCalculationMethod(method) {
  appStore.setCalculationMethod(method)
  // Re-render all tabs to update charts with new calculation method
  reRenderAllTabs()
}

// Handle device pixel ratio changes (e.g., moving window between displays with different DPI)
// This ensures charts remain crisp when moved between standard and HiDPI displays
// Debounced to avoid excessive checks during window resizing
let updatePixelRatioTimeout = null
function updatePixelRatio() {
  // Clear any pending update
  if (updatePixelRatioTimeout) {
    clearTimeout(updatePixelRatioTimeout)
  }
  
  // Debounce to avoid excessive re-renders during window resize
  updatePixelRatioTimeout = setTimeout(() => {
    const newPixelRatio = window.devicePixelRatio || 1
    if (newPixelRatio !== currentPixelRatio.value) {
      currentPixelRatio.value = newPixelRatio
      reRenderAllTabs()
    }
  }, 200) // 200ms debounce
}

onMounted(() => {
  // Only render the first tab (FPS) on mount
  if (props.benchmarkData && props.benchmarkData.length > 0) {
    nextTick(() => {
      renderFPSTab()
      renderedTabs.value.fps = true
    })
  }

  // Listen for window resize events which may indicate DPI changes
  // This handles cases like:
  // - Moving window between displays with different pixel densities
  // - Browser zoom changes
  // - Display settings changes
  window.addEventListener('resize', updatePixelRatio)
})

onUnmounted(() => {
  window.removeEventListener('resize', updatePixelRatio)
  // Clear any pending timeout
  if (updatePixelRatioTimeout) {
    clearTimeout(updatePixelRatioTimeout)
  }
})

watch(() => props.benchmarkData, () => {
  // Reset rendered tabs when data changes
  renderedTabs.value = {
    fps: false,
    frametime: false,
    summary: false,
    'more-metrics': false
  }
  
  // Re-render the active tab
  if (props.benchmarkData && props.benchmarkData.length > 0) {
    nextTick(() => {
      renderFPSTab()
      renderedTabs.value.fps = true
    })
  }
}, { deep: true })

// Watch for theme changes and re-render all rendered tabs
watch(() => appStore.theme, () => {
  reRenderAllTabs()
})
</script>

<style scoped>
.table-responsive {
  overflow-x: auto;
}

.table {
  white-space: nowrap;
}

/* Chart container panels - targets all chart divs identified by ref attributes */
.tab-pane > div[ref*="Chart"],
.tab-pane > div > div[ref*="Chart"] {
  background-color: var(--bs-secondary-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: 8px;
  padding: 15px;
  margin-bottom: 20px;
}

.baseline-selector {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 10px;
  background-color: var(--bs-secondary-bg);
  border-radius: 5px;
}

.baseline-selector .form-label {
  margin-bottom: 0;
  font-size: 14px;
  font-weight: 500;
}

.baseline-selector .form-select {
  max-width: 300px;
}

.calculation-method-selector {
  display: flex;
  align-items: center;
  padding: 15px;
  background-color: var(--bs-secondary-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: 8px;
}

.calculation-method-selector .form-label {
  margin-bottom: 0;
  font-size: 14px;
  white-space: nowrap;
}

.calculation-method-selector .method-toggle {
  flex-wrap: wrap;
}

.calculation-method-selector .method-toggle .btn {
  font-size: 14px;
}

.example-box {
  font-size: 14px;
  font-family: 'Courier New', monospace;
}

.example-box ul {
  padding-left: 20px;
}

</style>
