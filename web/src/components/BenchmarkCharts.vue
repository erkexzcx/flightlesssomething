<template>
  <div>
    <!-- Loading state while preparing data -->
    <div v-if="!componentReady" class="text-center my-5">
      <div class="spinner-border" role="status">
        <span class="visually-hidden">Preparing visualization...</span>
      </div>
      <p class="text-muted mt-2">Preparing benchmark visualization...</p>
    </div>

    <!-- Main content -->
    <div v-show="componentReady">
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
                <th scope="row">{{ data.Label }}</th>
                <td>{{ data.SpecOS || '-' }}</td>
                <td>{{ data.SpecGPU || '-' }}</td>
                <td>{{ data.SpecCPU || '-' }}</td>
                <td>{{ data.SpecRAM || '-' }}</td>
                <td>{{ formatOSSpecific(data) }}</td>
              </tr>
            </tbody>
          </table>
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
        <div v-if="!renderedTabs.fps || calculatingStats" class="text-center my-5">
          <div class="spinner-border" role="status">
            <span class="visually-hidden">{{ calculatingStats ? 'Calculating...' : 'Rendering charts...' }}</span>
          </div>
          <p class="text-muted mt-2">{{ calculatingStats ? calculationProgress : 'Rendering FPS charts...' }}</p>
        </div>
        <div v-show="renderedTabs.fps && !calculatingStats">
          <div ref="fpsChart2" style="height:250pt;"></div>
          <div ref="fpsMinMaxAvgChart" style="height:500pt;"></div>
          <div ref="fpsDensityChart" style="height:250pt;"></div>
          <!-- Dropdown for selecting baseline -->
          <div v-if="benchmarkData && benchmarkData.length > 1" class="baseline-selector mb-2">
            <label for="fpsBaselineSelect" class="form-label me-2">Baseline (0%):</label>
            <select 
              id="fpsBaselineSelect" 
              class="form-select form-select-sm d-inline-block w-auto"
              v-model="fpsBaselineIndex"
              @change="renderFPSComparisonChart"
            >
              <option :value="null">Auto (slowest)</option>
              <option 
                v-for="(data, index) in benchmarkData" 
                :key="index" 
                :value="index"
              >
                {{ data.Label }}
              </option>
            </select>
          </div>
          <div ref="fpsAvgChart" style="height:250pt;"></div>
          <div ref="fpsStddevVarianceChart" style="height:400pt;"></div>
        </div>
      </div>

      <!-- Frametime Tab -->
      <div class="tab-pane fade" id="frametime" role="tabpanel">
        <div v-if="!renderedTabs.frametime || calculatingStats" class="text-center my-5">
          <div class="spinner-border" role="status">
            <span class="visually-hidden">{{ calculatingStats ? 'Calculating...' : 'Rendering charts...' }}</span>
          </div>
          <p class="text-muted mt-2">{{ calculatingStats ? calculationProgress : 'Rendering Frametime charts...' }}</p>
        </div>
        <div v-show="renderedTabs.frametime && !calculatingStats">
          <div ref="frameTimeChart2" style="height:250pt;"></div>
          <div ref="frametimeMinMaxAvgChart" style="height:500pt;"></div>
          <div ref="frametimeDensityChart" style="height:250pt;"></div>
          <!-- Dropdown for selecting baseline -->
          <div v-if="benchmarkData && benchmarkData.length > 1" class="baseline-selector mb-2">
            <label for="frametimeBaselineSelect" class="form-label me-2">Baseline (0%):</label>
            <select 
              id="frametimeBaselineSelect" 
              class="form-select form-select-sm d-inline-block w-auto"
              v-model="frametimeBaselineIndex"
              @change="renderFrametimeComparisonChart"
            >
              <option :value="null">Auto (fastest)</option>
              <option 
                v-for="(data, index) in benchmarkData" 
                :key="index" 
                :value="index"
              >
                {{ data.Label }}
              </option>
            </select>
          </div>
          <div ref="frametimeAvgChart" style="height:250pt;"></div>
          <div ref="frametimeStddevVarianceChart" style="height:400pt;"></div>
        </div>
      </div>

      <!-- Summary Tab -->
      <div class="tab-pane fade" id="summary" role="tabpanel">
        <div v-if="!renderedTabs.summary || calculatingStats" class="text-center my-5">
          <div class="spinner-border" role="status">
            <span class="visually-hidden">{{ calculatingStats ? 'Calculating...' : 'Rendering charts...' }}</span>
          </div>
          <p class="text-muted mt-2">{{ calculatingStats ? calculationProgress : 'Rendering Summary charts...' }}</p>
        </div>
        <div v-show="renderedTabs.summary && !calculatingStats">
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
        <div v-if="!renderedTabs['more-metrics'] || calculatingStats" class="text-center my-5">
          <div class="spinner-border" role="status">
            <span class="visually-hidden">{{ calculatingStats ? 'Calculating...' : 'Rendering charts...' }}</span>
          </div>
          <p class="text-muted mt-2">{{ calculatingStats ? calculationProgress : 'Rendering All Data charts...' }}</p>
        </div>
        <div v-show="renderedTabs['more-metrics'] && !calculatingStats">
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
    </div> <!-- End v-show="componentReady" -->
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
import BenchmarkWorker from '../workers/benchmark-calculations.worker.js?worker'

// Initialize Highcharts modules
HighchartsBoost(Highcharts)
HighchartsExporting(Highcharts)
HighchartsExportData(Highcharts)
HighchartsFullScreen(Highcharts)

// Use app store for calculation mode
const appStore = useAppStore()

// Web Worker for async calculations
let calculationWorker = null

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

// Constants for data processing
const OUTLIER_LOW_PERCENTILE = 0.01  // Remove bottom 1% outliers
const OUTLIER_HIGH_PERCENTILE = 0.97 // Remove top 3% outliers
const MAX_DENSITY_POINTS = 100       // Maximum points for density charts
const MAX_FRAMETIME_FOR_INVALID_FPS = 1000000  // Very large frametime for invalid FPS (0 or negative)

const props = defineProps({
  benchmarkData: {
    type: Array,
    default: () => []
  }
})

// Track if component is ready to render
const componentReady = ref(false)

// Track which tabs have been rendered
const renderedTabs = ref({
  fps: false,
  frametime: false,
  summary: false,
  'more-metrics': false
})

// Track calculation state
const calculatingStats = ref(false)
const calculationProgress = ref('')

// Track selected baseline for comparison charts
const fpsBaselineIndex = ref(null) // null means auto (slowest)
const frametimeBaselineIndex = ref(null) // null means auto (fastest)

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
const fpsStddevVarianceChart = ref(null)
const frametimeMinMaxAvgChart = ref(null)
const frametimeDensityChart = ref(null)
const frametimeAvgChart = ref(null)
const frametimeStddevVarianceChart = ref(null)
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
  if (data.SpecLinuxKernel) parts.push(data.SpecLinuxKernel)
  if (data.SpecLinuxScheduler) parts.push(data.SpecLinuxScheduler)
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

  const colors = getThemeColors.value
  const chartOpts = commonChartOptions.value

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

  const colors = getThemeColors.value
  const chartOpts = commonChartOptions.value

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
  
  return {
    fpsDataArrays: props.benchmarkData.map(d => ({ label: d.Label, data: d.DataFPS || [] })),
    frameTimeDataArrays: props.benchmarkData.map(d => ({ label: d.Label, data: d.DataFrameTime || [] })),
    cpuLoadDataArrays: props.benchmarkData.map(d => ({ label: d.Label, data: d.DataCPULoad || [] })),
    gpuLoadDataArrays: props.benchmarkData.map(d => ({ label: d.Label, data: d.DataGPULoad || [] })),
    cpuTempDataArrays: props.benchmarkData.map(d => ({ label: d.Label, data: d.DataCPUTemp || [] })),
    cpuPowerDataArrays: props.benchmarkData.map(d => ({ label: d.Label, data: d.DataCPUPower || [] })),
    gpuTempDataArrays: props.benchmarkData.map(d => ({ label: d.Label, data: d.DataGPUTemp || [] })),
    gpuCoreClockDataArrays: props.benchmarkData.map(d => ({ label: d.Label, data: d.DataGPUCoreClock || [] })),
    gpuMemClockDataArrays: props.benchmarkData.map(d => ({ label: d.Label, data: d.DataGPUMemClock || [] })),
    gpuVRAMUsedDataArrays: props.benchmarkData.map(d => ({ label: d.Label, data: d.DataGPUVRAMUsed || [] })),
    gpuPowerDataArrays: props.benchmarkData.map(d => ({ label: d.Label, data: d.DataGPUPower || [] })),
    ramUsedDataArrays: props.benchmarkData.map(d => ({ label: d.Label, data: d.DataRAMUsed || [] })),
    swapUsedDataArrays: props.benchmarkData.map(d => ({ label: d.Label, data: d.DataSwapUsed || [] }))
  }
})

// Statistical calculations - moved from computed to ref for async updates
const fpsStats = ref(null)
const frametimeStats = ref(null)
const summaryStats = ref(null)

// Initialize web worker and set up message handlers
function initializeWorker() {
  if (calculationWorker) {
    calculationWorker.terminate()
  }
  
  calculationWorker = new BenchmarkWorker()
  
  calculationWorker.onmessage = (e) => {
    const { type, data, error } = e.data
    
    switch (type) {
      case 'fpsStatsComplete':
        fpsStats.value = data
        calculationProgress.value = 'FPS calculations complete'
        break
      case 'frametimeStatsComplete':
        frametimeStats.value = data
        calculationProgress.value = 'Frametime calculations complete'
        break
      case 'summaryStatsComplete':
        summaryStats.value = data
        calculationProgress.value = 'Summary calculations complete'
        calculatingStats.value = false
        break
      case 'error':
        console.error('Worker error:', error)
        calculatingStats.value = false
        break
    }
  }
  
  calculationWorker.onerror = (error) => {
    console.error('Worker error:', error)
    calculatingStats.value = false
  }
}

// Calculate statistics using the web worker
async function calculateStatistics() {
  if (!props.benchmarkData || props.benchmarkData.length === 0) {
    fpsStats.value = null
    frametimeStats.value = null
    summaryStats.value = null
    return
  }
  
  calculatingStats.value = true
  calculationProgress.value = 'Starting calculations...'
  
  const arrays = dataArrays.value
  
  // Use requestIdleCallback to schedule worker tasks when browser is idle
  // This prevents blocking the UI thread
  const scheduleWork = (callback) => {
    if ('requestIdleCallback' in window) {
      requestIdleCallback(callback, { timeout: 100 })
    } else {
      setTimeout(callback, 0)
    }
  }
  
  // Schedule calculations sequentially to avoid overwhelming the worker
  scheduleWork(() => {
    calculationProgress.value = 'Calculating FPS statistics...'
    // Convert to plain objects to avoid cloning issues with Vue reactive objects
    const plainArrays = arrays.fpsDataArrays.map(d => ({
      label: d.label,
      data: Array.from(d.data)
    }))
    calculationWorker.postMessage({
      type: 'calculateFpsStats',
      data: { fpsDataArrays: plainArrays }
    })
  })
  
  // Wait a bit before next calculation
  await new Promise(resolve => setTimeout(resolve, 10))
  
  scheduleWork(() => {
    calculationProgress.value = 'Calculating Frametime statistics...'
    const plainArrays = arrays.frameTimeDataArrays.map(d => ({
      label: d.label,
      data: Array.from(d.data)
    }))
    calculationWorker.postMessage({
      type: 'calculateFrametimeStats',
      data: { frametimeDataArrays: plainArrays }
    })
  })
  
  // Wait a bit before next calculation
  await new Promise(resolve => setTimeout(resolve, 10))
  
  scheduleWork(() => {
    calculationProgress.value = 'Calculating Summary statistics...'
    // Convert all arrays to plain objects
    const plainData = {
      fpsDataArrays: arrays.fpsDataArrays.map(d => ({ label: d.label, data: Array.from(d.data) })),
      frameTimeDataArrays: arrays.frameTimeDataArrays.map(d => ({ label: d.label, data: Array.from(d.data) })),
      cpuLoadDataArrays: arrays.cpuLoadDataArrays.map(d => ({ label: d.label, data: Array.from(d.data) })),
      gpuLoadDataArrays: arrays.gpuLoadDataArrays.map(d => ({ label: d.label, data: Array.from(d.data) })),
      gpuCoreClockDataArrays: arrays.gpuCoreClockDataArrays.map(d => ({ label: d.label, data: Array.from(d.data) })),
      gpuMemClockDataArrays: arrays.gpuMemClockDataArrays.map(d => ({ label: d.label, data: Array.from(d.data) })),
      cpuPowerDataArrays: arrays.cpuPowerDataArrays.map(d => ({ label: d.label, data: Array.from(d.data) })),
      gpuPowerDataArrays: arrays.gpuPowerDataArrays.map(d => ({ label: d.label, data: Array.from(d.data) }))
    }
    calculationWorker.postMessage({
      type: 'calculateSummaryStats',
      data: plainData
    })
  })
}

function renderFPSTab() {
  if (!props.benchmarkData || props.benchmarkData.length === 0) return
  
  const { fpsDataArrays } = dataArrays.value
  const stats = fpsStats.value
  
  // Wait for calculations if not ready yet
  if (!stats) {
    return
  }
  
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

  // FPS Stability chart
  if (fpsStddevVarianceChart.value && stats) {
    const colors = getThemeColors.value
    const chartOpts = commonChartOptions.value
    const categories = stats.map(s => s.label)
    const standardDeviations = stats.map(s => s.stddev)
    const variances = stats.map(s => s.variance)

    Highcharts.chart(fpsStddevVarianceChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'FPS Stability' },
      subtitle: { ...chartOpts.subtitle, text: 'Measures of FPS consistency (std. dev.) and spread (variance). Less is better.' },
      xAxis: { ...chartOpts.xAxis, categories: categories },
      yAxis: { ...chartOpts.yAxis, title: { text: 'Value', align: 'high', style: { color: colors.textColor } } },
      tooltip: { ...chartOpts.tooltip, formatter: function() { return `<b>${this.series.name}</b>: ${this.y.toFixed(2)}` } },
      plotOptions: { bar: { borderColor: colors.barBorderColor, borderWidth: 1, dataLabels: { enabled: true, style: { color: colors.textColor }, formatter: function() { return this.y.toFixed(2) } } } },
      legend: { ...chartOpts.legend, enabled: true },
      series: [
        { name: 'Std. Dev.', data: standardDeviations, color: '#FF5733' },
        { name: 'Variance', data: variances, color: '#33FF57' }
      ]
    })
  }
}

function renderFrametimeTab() {
  if (!props.benchmarkData || props.benchmarkData.length === 0) return
  
  const { frameTimeDataArrays } = dataArrays.value
  const stats = frametimeStats.value
  
  // Wait for calculations if not ready yet
  if (!stats) {
    return
  }
  
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

  // Frametime Stability chart
  if (frametimeStddevVarianceChart.value && stats) {
    const colors = getThemeColors.value
    const chartOpts = commonChartOptions.value
    const categories = stats.map(s => s.label)
    const standardDeviations = stats.map(s => s.stddev)
    const variances = stats.map(s => s.variance)

    Highcharts.chart(frametimeStddevVarianceChart.value, {
      ...chartOpts,
      chart: { ...chartOpts.chart, type: 'bar' },
      title: { ...chartOpts.title, text: 'Frametime Stability' },
      subtitle: { ...chartOpts.subtitle, text: 'Measures of Frametime consistency (std. dev.) and spread (variance). Less is better.' },
      xAxis: { ...chartOpts.xAxis, categories: categories },
      yAxis: { ...chartOpts.yAxis, title: { text: 'Value', align: 'high', style: { color: colors.textColor } } },
      tooltip: { ...chartOpts.tooltip, formatter: function() { return `<b>${this.series.name}</b>: ${this.y.toFixed(2)}` } },
      plotOptions: { bar: { borderColor: colors.barBorderColor, borderWidth: 1, dataLabels: { enabled: true, style: { color: colors.textColor }, formatter: function() { return this.y.toFixed(2) } } } },
      legend: { ...chartOpts.legend, enabled: true },
      series: [
        { name: 'Std. Dev.', data: standardDeviations, color: '#FF5733' },
        { name: 'Variance', data: variances, color: '#33FF57' }
      ]
    })
  }
}

function renderSummaryTab() {
  if (!props.benchmarkData || props.benchmarkData.length === 0) return
  
  const arrays = dataArrays.value
  const stats = summaryStats.value
  
  // Wait for calculations if not ready yet
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
// Used by theme changes and DPI changes
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
  // Initialize the web worker
  initializeWorker()
  
  // Defer component rendering to next frame to prevent blocking
  requestAnimationFrame(() => {
    componentReady.value = true
    
    // Calculate statistics if data is available
    if (props.benchmarkData && props.benchmarkData.length > 0) {
      // Use a small delay to allow the UI to render first
      setTimeout(() => {
        calculateStatistics()
      }, 50)
    }
  })

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
  // Terminate the worker
  if (calculationWorker) {
    calculationWorker.terminate()
  }
})

watch(() => props.benchmarkData, () => {
  // Reset component state when data changes
  componentReady.value = false
  
  // Reset rendered tabs when data changes
  renderedTabs.value = {
    fps: false,
    frametime: false,
    summary: false,
    'more-metrics': false
  }
  
  // Defer rendering to next frame
  requestAnimationFrame(() => {
    componentReady.value = true
    
    // Recalculate statistics with new data
    if (props.benchmarkData && props.benchmarkData.length > 0) {
      setTimeout(() => {
        calculateStatistics()
      }, 50)
    } else {
      fpsStats.value = null
      frametimeStats.value = null
      summaryStats.value = null
    }
  })
}, { deep: true })

// Watch for stats completion and render the FPS tab (default)
watch([fpsStats, frametimeStats, summaryStats], () => {
  // Only render when all stats are ready
  if (fpsStats.value && frametimeStats.value && summaryStats.value && !renderedTabs.value.fps) {
    nextTick(() => {
      renderFPSTab()
      renderedTabs.value.fps = true
    })
  }
})

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
</style>
