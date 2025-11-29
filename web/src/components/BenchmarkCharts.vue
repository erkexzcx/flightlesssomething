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
          <div ref="fpsAvgChart" style="height:250pt;"></div>
          <div ref="fpsStddevVarianceChart" style="height:400pt;"></div>
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
          <div ref="frametimeAvgChart" style="height:250pt;"></div>
          <div ref="frametimeStddevVarianceChart" style="height:400pt;"></div>
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
import { ref, onMounted, watch, nextTick } from 'vue'
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

// Constants for data processing
const OUTLIER_LOW_PERCENTILE = 0.01  // Remove bottom 1% outliers
const OUTLIER_HIGH_PERCENTILE = 0.97 // Remove top 3% outliers
const MAX_DENSITY_POINTS = 100       // Maximum points for density charts

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

// Common chart options
const commonChartOptions = {
  chart: { 
    backgroundColor: null, 
    style: { color: '#FFFFFF' }, 
    animation: false, 
    boost: { 
      useGPUTranslations: true, 
      usePreallocated: true,
      seriesThreshold: 1  // Enable boost for all series
    } 
  },
  title: { style: { color: '#FFFFFF', fontSize: '16px' } },
  subtitle: { style: { color: '#FFFFFF', fontSize: '12px' } },
  xAxis: { labels: { style: { color: '#FFFFFF' } }, lineColor: '#FFFFFF', tickColor: '#FFFFFF' },
  yAxis: { labels: { style: { color: '#FFFFFF' } }, gridLineColor: 'rgba(255, 255, 255, 0.1)', title: { text: false } },
  tooltip: { backgroundColor: '#1E1E1E', borderColor: '#FFFFFF', style: { color: '#FFFFFF' } },
  legend: { itemStyle: { color: '#FFFFFF' } },
  credits: { enabled: false },
  plotOptions: { series: { animation: false, turboThreshold: 0 } }  // turboThreshold: 0 allows any number of points
}

// Set global options
Highcharts.setOptions({ 
  chart: { animation: false }, 
  plotOptions: { series: { animation: false, turboThreshold: 0 } },
  boost: { enabled: true }
})

const colors = Highcharts.getOptions().colors

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

function calculateAverage(data) {
  if (!data || data.length === 0) return 0
  return data.reduce((acc, value) => acc + value, 0) / data.length
}

function calculatePercentile(data, percentile) {
  if (!data || data.length === 0) return 0
  const sorted = [...data].sort((a, b) => a - b)
  return sorted[Math.ceil(percentile / 100 * sorted.length) - 1]
}

function calculateStandardDeviation(data) {
  if (!data || data.length === 0) return 0
  const mean = calculateAverage(data)
  const squaredDiffs = data.map(value => Math.pow(value - mean, 2))
  const avgSquaredDiff = calculateAverage(squaredDiffs)
  return Math.sqrt(avgSquaredDiff)
}

function calculateVariance(data) {
  if (!data || data.length === 0) return 0
  const mean = calculateAverage(data)
  const squaredDiffs = data.map(value => Math.pow(value - mean, 2))
  return calculateAverage(squaredDiffs)
}

function filterOutliers(data) {
  if (!data || data.length === 0) return []
  const sorted = [...data].sort((a, b) => a - b)
  return sorted.slice(
    Math.floor(sorted.length * OUTLIER_LOW_PERCENTILE), 
    Math.ceil(sorted.length * OUTLIER_HIGH_PERCENTILE)
  )
}

function countOccurrences(data) {
  const counts = {}
  data.forEach(value => {
    const rounded = Math.round(value)
    counts[rounded] = (counts[rounded] || 0) + 1
  })

  let array = Object.keys(counts).map(key => [parseInt(key), counts[key]]).sort((a, b) => a[0] - b[0])

  while (array.length > MAX_DENSITY_POINTS) {
    let minDiff = Infinity
    let minIndex = -1

    for (let i = 0; i < array.length - 1; i++) {
      const diff = array[i + 1][0] - array[i][0]
      if (diff < minDiff) {
        minDiff = diff
        minIndex = i
      }
    }

    array[minIndex][1] += array[minIndex + 1][1]
    array[minIndex][0] = (array[minIndex][0] + array[minIndex + 1][0]) / 2
    array.splice(minIndex + 1, 1)
  }

  return array
}

function getLineChartOptions(title, description, unit, maxY = null) {
  return {
    ...commonChartOptions,
    chart: { ...commonChartOptions.chart, type: 'line', zooming: { type: 'x' } },
    title: { ...commonChartOptions.title, text: title },
    subtitle: { ...commonChartOptions.subtitle, text: description },
    xAxis: { ...commonChartOptions.xAxis, labels: { enabled: false } },
    yAxis: { ...commonChartOptions.yAxis, max: maxY, labels: { ...commonChartOptions.yAxis.labels, formatter: function() { return this.value.toFixed(2) + ' ' + unit } } },
    tooltip: { ...commonChartOptions.tooltip, pointFormat: `<span style="color:{series.color}">{series.name}</span>: <b>{point.y:.2f} ${unit}</b><br/>` },
    plotOptions: { line: { marker: { enabled: false, symbol: 'circle', radius: 1.5, states: { hover: { enabled: true } } }, lineWidth: 1 } },
    legend: { ...commonChartOptions.legend, enabled: true },
    series: [],
    exporting: { buttons: { contextButton: { menuItems: ['viewFullscreen', 'printChart', 'separator', 'downloadPNG', 'downloadJPEG', 'downloadPDF', 'downloadSVG', 'separator', 'downloadCSV', 'downloadXLS'] } } }
  }
}

function getBarChartOptions(title, unit, maxY = null) {
  return {
    ...commonChartOptions,
    chart: { ...commonChartOptions.chart, type: 'bar' },
    title: { ...commonChartOptions.title, text: title },
    xAxis: { ...commonChartOptions.xAxis, categories: [], title: { text: null } },
    yAxis: { ...commonChartOptions.yAxis, min: 0, max: maxY, title: { text: unit, align: 'high', style: { color: '#FFFFFF' } }, labels: { ...commonChartOptions.yAxis.labels, formatter: function() { return this.value.toFixed(2) + ' ' + unit } } },
    tooltip: { ...commonChartOptions.tooltip, valueSuffix: ' ' + unit, formatter: function() { return `<b>${this.point.category}</b>: ${this.y.toFixed(2)} ${unit}` } },
    plotOptions: { bar: { dataLabels: { enabled: true, style: { color: '#FFFFFF' }, formatter: function() { return this.y.toFixed(2) + ' ' + unit } } } },
    legend: { enabled: false },
    series: []
  }
}

function createChart(element, title, description, unit, dataArrays, maxY = null) {
  if (!element || !dataArrays || dataArrays.length === 0) return
  
  const options = getLineChartOptions(title, description, unit, maxY)
  // Decimate data only for line chart rendering to improve performance
  // Statistics are calculated from full data before this function is called
  options.series = dataArrays.map((dataArray, index) => ({
    name: dataArray.label,
    data: decimateForLineChart(dataArray.data || [], 2000),
    color: colors[index % colors.length]
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

function prepareDataArrays() {
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
}

function renderFPSTab() {
  if (!props.benchmarkData || props.benchmarkData.length === 0) return
  
  const { fpsDataArrays } = prepareDataArrays()
  
  // Create FPS charts
  createChart(fpsChart2.value, 'FPS', 'More is better', 'fps', fpsDataArrays)

  // FPS Min/Max/Avg chart
  if (fpsMinMaxAvgChart.value) {
    const categories = fpsDataArrays.map(d => d.label)
    const minFPSData = fpsDataArrays.map(d => calculatePercentile(d.data, 1))
    const avgFPSData = fpsDataArrays.map(d => calculateAverage(d.data))
    const maxFPSData = fpsDataArrays.map(d => calculatePercentile(d.data, 97))

    Highcharts.chart(fpsMinMaxAvgChart.value, {
      ...commonChartOptions,
      chart: { ...commonChartOptions.chart, type: 'bar' },
      title: { ...commonChartOptions.title, text: 'Min/Avg/Max FPS' },
      subtitle: { ...commonChartOptions.subtitle, text: 'More is better' },
      xAxis: { ...commonChartOptions.xAxis, categories: categories },
      yAxis: { ...commonChartOptions.yAxis, title: { text: 'FPS', align: 'high', style: { color: '#FFFFFF' } } },
      tooltip: { ...commonChartOptions.tooltip, valueSuffix: ' FPS', formatter: function() { return `<b>${this.series.name}</b>: ${this.y.toFixed(2)} FPS` } },
      plotOptions: { bar: { dataLabels: { enabled: true, style: { color: '#FFFFFF' }, formatter: function() { return this.y.toFixed(2) + ' fps' } } } },
      legend: { ...commonChartOptions.legend, reversed: true, enabled: true },
      series: [
        { name: '97th', data: maxFPSData, color: '#00FF00' },
        { name: 'AVG', data: avgFPSData, color: '#0000FF' },
        { name: '1%', data: minFPSData, color: '#FF0000' }
      ]
    })
  }

  // FPS Density chart
  if (fpsDensityChart.value) {
    const densityData = fpsDataArrays.map(d => ({
      name: d.label,
      data: countOccurrences(filterOutliers(d.data))
    }))

    Highcharts.chart(fpsDensityChart.value, {
      ...commonChartOptions,
      chart: { ...commonChartOptions.chart, type: 'areaspline' },
      title: { ...commonChartOptions.title, text: 'FPS Density' },
      xAxis: { ...commonChartOptions.xAxis, title: { text: 'FPS', style: { color: '#FFFFFF' } }, labels: { style: { color: '#FFFFFF' } } },
      tooltip: { ...commonChartOptions.tooltip, shared: true, formatter: function() { return `<b>${this.points[0].series.name}</b>: ${this.points[0].y} points at ~${Math.round(this.points[0].x)} FPS` } },
      plotOptions: { areaspline: { fillOpacity: 0.5, marker: { enabled: false } } },
      legend: { ...commonChartOptions.legend, enabled: true },
      series: densityData
    })
  }

  // FPS Average comparison chart
  if (fpsAvgChart.value) {
    const fpsAverages = fpsDataArrays.map(d => calculateAverage(d.data))
    if (fpsAverages.length > 0) {
      const baselineFPS = Math.min(...fpsAverages)
      const percentageFPSData = fpsAverages.map(fps => (fps / baselineFPS) * 100)

      const sortedData = fpsDataArrays.map((d, index) => ({
        label: d.label,
        percentage: percentageFPSData[index]
      })).sort((a, b) => a.percentage - b.percentage)

      const sortedCategories = sortedData.map(item => item.label)
      const sortedPercentages = sortedData.map(item => item.percentage)

      Highcharts.chart(fpsAvgChart.value, {
        ...commonChartOptions,
        chart: { ...commonChartOptions.chart, type: 'bar' },
        title: { ...commonChartOptions.title, text: 'Avg FPS comparison in %' },
        subtitle: { ...commonChartOptions.subtitle, text: 'More is better' },
        xAxis: { ...commonChartOptions.xAxis, categories: sortedCategories },
        yAxis: { ...commonChartOptions.yAxis, min: 95, title: { text: 'Percentage (%)', align: 'high', style: { color: '#FFFFFF' } } },
        tooltip: { ...commonChartOptions.tooltip, valueSuffix: ' %', formatter: function() { return `<b>${this.point.category}</b>: ${this.y.toFixed(2)} %` } },
        plotOptions: { bar: { dataLabels: { enabled: true, style: { color: '#FFFFFF' }, formatter: function() { return this.y.toFixed(2) + ' %' } } } },
        legend: { enabled: false },
        series: [{ name: 'FPS Percentage', data: sortedPercentages, colorByPoint: true, colors: colors }]
      })
    }
  }

  // FPS Stability chart
  if (fpsStddevVarianceChart.value) {
    const categories = fpsDataArrays.map(d => d.label)
    const standardDeviations = fpsDataArrays.map(d => calculateStandardDeviation(d.data))
    const variances = fpsDataArrays.map(d => calculateVariance(d.data))

    Highcharts.chart(fpsStddevVarianceChart.value, {
      ...commonChartOptions,
      chart: { ...commonChartOptions.chart, type: 'bar' },
      title: { ...commonChartOptions.title, text: 'FPS Stability' },
      subtitle: { ...commonChartOptions.subtitle, text: 'Measures of FPS consistency (std. dev.) and spread (variance). Less is better.' },
      xAxis: { ...commonChartOptions.xAxis, categories: categories },
      yAxis: { ...commonChartOptions.yAxis, title: { text: 'Value', align: 'high', style: { color: '#FFFFFF' } } },
      tooltip: { ...commonChartOptions.tooltip, formatter: function() { return `<b>${this.series.name}</b>: ${this.y.toFixed(2)}` } },
      plotOptions: { bar: { dataLabels: { enabled: true, style: { color: '#FFFFFF' }, formatter: function() { return this.y.toFixed(2) } } } },
      legend: { ...commonChartOptions.legend, enabled: true },
      series: [
        { name: 'Std. Dev.', data: standardDeviations, color: '#FF5733' },
        { name: 'Variance', data: variances, color: '#33FF57' }
      ]
    })
  }
}

function renderFrametimeTab() {
  if (!props.benchmarkData || props.benchmarkData.length === 0) return
  
  const { frameTimeDataArrays } = prepareDataArrays()
  
  createChart(frameTimeChart2.value, 'Frametime', 'Less is better', 'ms', frameTimeDataArrays)

  // Frametime Min/Max/Avg chart
  if (frametimeMinMaxAvgChart.value) {
    const categories = frameTimeDataArrays.map(d => d.label)
    const minData = frameTimeDataArrays.map(d => calculatePercentile(d.data, 1))
    const avgData = frameTimeDataArrays.map(d => calculateAverage(d.data))
    const maxData = frameTimeDataArrays.map(d => calculatePercentile(d.data, 97))

    Highcharts.chart(frametimeMinMaxAvgChart.value, {
      ...commonChartOptions,
      chart: { ...commonChartOptions.chart, type: 'bar' },
      title: { ...commonChartOptions.title, text: 'Min/Avg/Max Frametime' },
      subtitle: { ...commonChartOptions.subtitle, text: 'Less is better' },
      xAxis: { ...commonChartOptions.xAxis, categories: categories },
      yAxis: { ...commonChartOptions.yAxis, title: { text: 'Frametime (ms)', align: 'high', style: { color: '#FFFFFF' } } },
      tooltip: { ...commonChartOptions.tooltip, valueSuffix: ' ms', formatter: function() { return `<b>${this.series.name}</b>: ${this.y.toFixed(2)} ms` } },
      plotOptions: { bar: { dataLabels: { enabled: true, style: { color: '#FFFFFF' }, formatter: function() { return this.y.toFixed(2) + ' ms' } } } },
      legend: { ...commonChartOptions.legend, reversed: true, enabled: true },
      series: [
        { name: '97th', data: maxData, color: '#00FF00' },
        { name: 'AVG', data: avgData, color: '#0000FF' },
        { name: '1%', data: minData, color: '#FF0000' }
      ]
    })
  }

  // Frametime Density chart
  if (frametimeDensityChart.value) {
    const densityData = frameTimeDataArrays.map(d => ({
      name: d.label,
      data: countOccurrences(filterOutliers(d.data))
    }))

    Highcharts.chart(frametimeDensityChart.value, {
      ...commonChartOptions,
      chart: { ...commonChartOptions.chart, type: 'areaspline' },
      title: { ...commonChartOptions.title, text: 'Frametime Density' },
      xAxis: { ...commonChartOptions.xAxis, title: { text: 'Frametime (ms)', style: { color: '#FFFFFF' } }, labels: { style: { color: '#FFFFFF' } } },
      tooltip: { ...commonChartOptions.tooltip, shared: true, formatter: function() { return `<b>${this.points[0].series.name}</b>: ${this.points[0].y} points at ~${Math.round(this.points[0].x)} ms` } },
      plotOptions: { areaspline: { fillOpacity: 0.5, marker: { enabled: false } } },
      legend: { ...commonChartOptions.legend, enabled: true },
      series: densityData
    })
  }

  // Frametime Average comparison chart
  if (frametimeAvgChart.value) {
    const frametimeAverages = frameTimeDataArrays.map(d => calculateAverage(d.data))
    if (frametimeAverages.length > 0) {
      const baselineFrametime = Math.min(...frametimeAverages)
      const percentageData = frametimeAverages.map(ft => (ft / baselineFrametime) * 100)

      const sortedData = frameTimeDataArrays.map((d, index) => ({
        label: d.label,
        percentage: percentageData[index]
      })).sort((a, b) => a.percentage - b.percentage)

      const sortedCategories = sortedData.map(item => item.label)
      const sortedPercentages = sortedData.map(item => item.percentage)

      Highcharts.chart(frametimeAvgChart.value, {
        ...commonChartOptions,
        chart: { ...commonChartOptions.chart, type: 'bar' },
        title: { ...commonChartOptions.title, text: 'Avg Frametime comparison in %' },
        subtitle: { ...commonChartOptions.subtitle, text: 'Less is better' },
        xAxis: { ...commonChartOptions.xAxis, categories: sortedCategories },
        yAxis: { ...commonChartOptions.yAxis, min: 95, title: { text: 'Percentage (%)', align: 'high', style: { color: '#FFFFFF' } } },
        tooltip: { ...commonChartOptions.tooltip, valueSuffix: ' %', formatter: function() { return `<b>${this.point.category}</b>: ${this.y.toFixed(2)} %` } },
        plotOptions: { bar: { dataLabels: { enabled: true, style: { color: '#FFFFFF' }, formatter: function() { return this.y.toFixed(2) + ' %' } } } },
        legend: { enabled: false },
        series: [{ name: 'Frametime Percentage', data: sortedPercentages, colorByPoint: true, colors: colors }]
      })
    }
  }

  // Frametime Stability chart
  if (frametimeStddevVarianceChart.value) {
    const categories = frameTimeDataArrays.map(d => d.label)
    const standardDeviations = frameTimeDataArrays.map(d => calculateStandardDeviation(d.data))
    const variances = frameTimeDataArrays.map(d => calculateVariance(d.data))

    Highcharts.chart(frametimeStddevVarianceChart.value, {
      ...commonChartOptions,
      chart: { ...commonChartOptions.chart, type: 'bar' },
      title: { ...commonChartOptions.title, text: 'Frametime Stability' },
      subtitle: { ...commonChartOptions.subtitle, text: 'Measures of Frametime consistency (std. dev.) and spread (variance). Less is better.' },
      xAxis: { ...commonChartOptions.xAxis, categories: categories },
      yAxis: { ...commonChartOptions.yAxis, title: { text: 'Value', align: 'high', style: { color: '#FFFFFF' } } },
      tooltip: { ...commonChartOptions.tooltip, formatter: function() { return `<b>${this.series.name}</b>: ${this.y.toFixed(2)}` } },
      plotOptions: { bar: { dataLabels: { enabled: true, style: { color: '#FFFFFF' }, formatter: function() { return this.y.toFixed(2) } } } },
      legend: { ...commonChartOptions.legend, enabled: true },
      series: [
        { name: 'Std. Dev.', data: standardDeviations, color: '#FF5733' },
        { name: 'Variance', data: variances, color: '#33FF57' }
      ]
    })
  }
}

function renderSummaryTab() {
  if (!props.benchmarkData || props.benchmarkData.length === 0) return
  
  const {
    fpsDataArrays,
    frameTimeDataArrays,
    cpuLoadDataArrays,
    gpuLoadDataArrays,
    gpuCoreClockDataArrays,
    gpuMemClockDataArrays,
    cpuPowerDataArrays,
    gpuPowerDataArrays
  } = prepareDataArrays()

  // Calculate averages for summary charts
  const fpsAverages = fpsDataArrays.map(d => calculateAverage(d.data))
  const frametimeAverages = frameTimeDataArrays.map(d => calculateAverage(d.data))
  const cpuLoadAverages = cpuLoadDataArrays.map(d => calculateAverage(d.data))
  const gpuLoadAverages = gpuLoadDataArrays.map(d => calculateAverage(d.data))
  const gpuCoreClockAverages = gpuCoreClockDataArrays.map(d => calculateAverage(d.data))
  const gpuMemClockAverages = gpuMemClockDataArrays.map(d => calculateAverage(d.data))
  const cpuPowerAverages = cpuPowerDataArrays.map(d => calculateAverage(d.data))
  const gpuPowerAverages = gpuPowerDataArrays.map(d => calculateAverage(d.data))

  // Create summary bar charts
  createBarChart(fpsSummaryChart.value, 'Average FPS', 'fps', fpsDataArrays.map(d => d.label), fpsAverages, colors)
  createBarChart(frametimeSummaryChart.value, 'Average Frametime', 'ms', frameTimeDataArrays.map(d => d.label), frametimeAverages, colors)
  createBarChart(cpuLoadSummaryChart.value, 'Average CPU Load', '%', cpuLoadDataArrays.map(d => d.label), cpuLoadAverages, colors, 100)
  createBarChart(gpuLoadSummaryChart.value, 'Average GPU Load', '%', gpuLoadDataArrays.map(d => d.label), gpuLoadAverages, colors, 100)
  createBarChart(gpuCoreClockSummaryChart.value, 'Average GPU Core Clock', 'MHz', gpuCoreClockDataArrays.map(d => d.label), gpuCoreClockAverages, colors)
  createBarChart(gpuMemClockSummaryChart.value, 'Average GPU Memory Clock', 'MHz', gpuMemClockDataArrays.map(d => d.label), gpuMemClockAverages, colors)
  createBarChart(cpuPowerSummaryChart.value, 'Average CPU Power', 'W', cpuPowerDataArrays.map(d => d.label), cpuPowerAverages, colors)
  createBarChart(gpuPowerSummaryChart.value, 'Average GPU Power', 'W', gpuPowerDataArrays.map(d => d.label), gpuPowerAverages, colors)
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
  } = prepareDataArrays()

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

onMounted(() => {
  // Only render the first tab (FPS) on mount
  if (props.benchmarkData && props.benchmarkData.length > 0) {
    nextTick(() => {
      renderFPSTab()
      renderedTabs.value.fps = true
    })
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
</script>

<style scoped>
.table-responsive {
  overflow-x: auto;
}

.table {
  color: #ffffff;
  white-space: nowrap;
}

.table-bordered {
  border-color: rgba(255, 255, 255, 0.1);
}

.table th,
.table td {
  border-color: rgba(255, 255, 255, 0.1);
}
</style>
