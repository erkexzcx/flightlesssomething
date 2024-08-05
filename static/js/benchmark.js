// Common chart options
const commonChartOptions = {
    chart: {backgroundColor: null, style: {color: '#FFFFFF'}, animation: false},
    title: {style: {color: '#FFFFFF', fontSize: '16px'}},
    subtitle: {style: {color: '#FFFFFF', fontSize: '12px'}},
    xAxis: {labels: {style: {color: '#FFFFFF'}}, lineColor: '#FFFFFF', tickColor: '#FFFFFF'},
    yAxis: {labels: {style: {color: '#FFFFFF'}}, gridLineColor: 'rgba(255, 255, 255, 0.1)', title: {text: false}},
    tooltip: {backgroundColor: '#1E1E1E', borderColor: '#FFFFFF', style: {color: '#FFFFFF'}},
    legend: {itemStyle: {color: '#FFFFFF'}},
    credits: {enabled: false},
    plotOptions: {series: {animation: false}}
};

// Highcharts global options
Highcharts.setOptions({chart: {animation: false}, plotOptions: {series: {animation: false}}});

const colors = Highcharts.getOptions().colors;

function getLineChartOptions(title, description, unit, maxY = null) {
    return {
        ...commonChartOptions,
        chart: {...commonChartOptions.chart, type: 'line', zooming: {type: 'x'}},
        title: {...commonChartOptions.title, text: title},
        subtitle: {...commonChartOptions.subtitle, text: description},
        xAxis: {...commonChartOptions.xAxis, labels: {enabled: false}}, // Hide X-axis labels
        yAxis: {...commonChartOptions.yAxis, max: maxY, labels: {...commonChartOptions.yAxis.labels, formatter: function() {return this.value.toFixed(2) + ' ' + unit;}}},
        tooltip: {...commonChartOptions.tooltip, pointFormat: `<span style="color:{series.color}">{series.name}</span>: <b>{point.y:.2f} ${unit}</b><br/>`},
        plotOptions: {line: {marker: {enabled: false, symbol: 'circle', radius: 1.5, states: {hover: {enabled: true}}}, lineWidth: 1}},
        legend: {...commonChartOptions.legend, enabled: true},
        series: [],
        exporting: {buttons: {contextButton: {menuItems: ['viewFullscreen', 'printChart', 'separator', 'downloadPNG', 'downloadJPEG', 'downloadPDF', 'downloadSVG', 'separator', 'downloadCSV', 'downloadXLS']}}}
    };
}

function getBarChartOptions(title, unit, maxY = null) {
    return {
        ...commonChartOptions,
        chart: {...commonChartOptions.chart, type: 'bar'},
        title: {...commonChartOptions.title, text: title},
        xAxis: {...commonChartOptions.xAxis, categories: [], title: {text: null}},
        yAxis: {...commonChartOptions.yAxis, min: 0, max: maxY, title: {text: unit, align: 'high', style: {color: '#FFFFFF'}}, labels: {...commonChartOptions.yAxis.labels, formatter: function() {return this.value.toFixed(2) + ' ' + unit;}}},
        tooltip: {...commonChartOptions.tooltip, valueSuffix: ' ' + unit, formatter: function() {return `<b>${this.point.category}</b>: ${this.y.toFixed(2)} ${unit}`;}},
        plotOptions: {bar: {dataLabels: {enabled: true, style: {color: '#FFFFFF'}, formatter: function() {return this.y.toFixed(2) + ' ' + unit;}}}},
        legend: {enabled: false},
        series: []
    };
}

function createChart(chartId, title, description, unit, dataArrays, maxY = null) {
    const options = getLineChartOptions(title, description, unit, maxY);
    options.series = dataArrays.map((dataArray, index) => ({name: dataArray.label, data: dataArray.data, color: colors[index % colors.length]}));
    Highcharts.chart(chartId, options);
}

function createBarChart(chartId, title, unit, categories, data, colors, maxY = null) {
    const options = getBarChartOptions(title, unit, maxY);
    options.xAxis.categories = categories;
    options.series = [{name: title, data: data, colorByPoint: true, colors: colors}];
    Highcharts.chart(chartId, options);
}

function calculateAverage(data) {
    return data.reduce((acc, value) => acc + value, 0) / data.length;
}

function calculatePercentile(data, percentile) {
    data.sort((a, b) => a - b);
    return data[Math.ceil(percentile / 100 * data.length) - 1];
}

// Create line charts
createChart('fpsChart', 'FPS', 'More is better', 'fps', fpsDataArrays);
createChart('frameTimeChart', 'Frametime', 'Less is better', 'ms', frameTimeDataArrays);
createChart('cpuLoadChart', 'CPU Load', '', '%', cpuLoadDataArrays, 100);
createChart('gpuLoadChart', 'GPU Load', '', '%', gpuLoadDataArrays, 100);
createChart('cpuTempChart', 'CPU Temperature', '', '°C', cpuTempDataArrays);
createChart('gpuTempChart', 'GPU Temperature', '', '°C', gpuTempDataArrays);
createChart('gpuCoreClockChart', 'GPU Core Clock', '', 'MHz', gpuCoreClockDataArrays);
createChart('gpuMemClockChart', 'GPU Memory Clock', '', 'MHz', gpuMemClockDataArrays);
createChart('gpuVRAMUsedChart', 'GPU VRAM Usage', '', 'GB', gpuVRAMUsedDataArrays);
createChart('gpuPowerChart', 'GPU Power', '', 'W', gpuPowerDataArrays);
createChart('ramUsedChart', 'RAM Usage', '', 'GB', ramUsedDataArrays);
createChart('swapUsedChart', 'SWAP Usage', '', 'GB', swapUsedDataArrays);

// Calculate average CPU and GPU load
const cpuLoadAverages = cpuLoadDataArrays.map(dataArray => calculateAverage(dataArray.data));
const gpuLoadAverages = gpuLoadDataArrays.map(dataArray => calculateAverage(dataArray.data));

// Create bar charts for average CPU and GPU load
createBarChart('cpuLoadSummaryChart', 'Average CPU Load', '%', cpuLoadDataArrays.map(dataArray => dataArray.label), cpuLoadAverages, colors, 100);
createBarChart('gpuLoadSummaryChart', 'Average GPU Load', '%', gpuLoadDataArrays.map(dataArray => dataArray.label), gpuLoadAverages, colors, 100);

// Calculate and render min, max, and average FPS
const categories = [];
const minFPSData = [];
const avgFPSData1 = [];
const maxFPSData = [];

fpsDataArrays.forEach(dataArray => {
    categories.push(dataArray.label);
    minFPSData.push(calculatePercentile(dataArray.data, 1));
    avgFPSData1.push(calculateAverage(dataArray.data));
    maxFPSData.push(calculatePercentile(dataArray.data, 97));
});

Highcharts.chart('minMaxAvgChart', {
    ...commonChartOptions,
    chart: {...commonChartOptions.chart, type: 'bar'},
    title: {...commonChartOptions.title, text: 'Min/Avg/Max FPS'},
    subtitle: {...commonChartOptions.subtitle, text: 'More is better'},
    xAxis: {...commonChartOptions.xAxis, categories: categories},
    yAxis: {...commonChartOptions.yAxis, title: {text: 'FPS', align: 'high', style: {color: '#FFFFFF'}}},
    tooltip: {...commonChartOptions.tooltip, valueSuffix: ' FPS', formatter: function() {return `<b>${this.series.name}</b>: ${this.y.toFixed(2)} FPS`;}},
    plotOptions: {bar: {dataLabels: {enabled: true, style: {color: '#FFFFFF'}, formatter: function() {return this.y.toFixed(2) + ' fps';}}}},
    legend: {...commonChartOptions.legend, reversed: true, enabled: true},
    series: [{name: '97th', data: maxFPSData, color: '#00FF00'}, {name: 'AVG', data: avgFPSData1, color: '#0000FF'}, {name: '1%', data: minFPSData, color: '#FF0000'}]
});

// Calculate average FPS for each filename
const avgFPSData2 = fpsDataArrays.map(dataArray => calculateAverage(dataArray.data));

// Calculate FPS as a percentage of the first element
const firstFPS = avgFPSData2[0];
const percentageFPSData = avgFPSData2.map(fps => (fps / firstFPS) * 100);

// Ensure the minimum FPS percentage is 100%
const minPercentage = Math.min(...percentageFPSData);
const normalizedPercentageFPSData = percentageFPSData.map(percentage => percentage - minPercentage + 100);

// Create an array of objects to sort both categories and data together
const sortedData = fpsDataArrays.map((dataArray, index) => ({
    label: dataArray.label,
    percentage: normalizedPercentageFPSData[index]
}));

// Sort the array by percentage
sortedData.sort((a, b) => a.percentage - b.percentage);

// Extract sorted categories and data
const sortedCategories = sortedData.map(item => item.label);
const sortedPercentageFPSData = sortedData.map(item => item.percentage);

// Create bar chart for FPS percentage
Highcharts.chart('avgChart', {
    ...commonChartOptions,
    chart: {...commonChartOptions.chart, type: 'bar'},
    title: {...commonChartOptions.title, text: 'Avg FPS comparison in %'},
    subtitle: {...commonChartOptions.subtitle, text: 'More is better'},
    xAxis: {...commonChartOptions.xAxis, categories: sortedCategories},
    yAxis: {...commonChartOptions.yAxis, min: 95, title: {text: 'Percentage (%)', align: 'high', style: {color: '#FFFFFF'}}},
    tooltip: {...commonChartOptions.tooltip, valueSuffix: ' %', formatter: function() {return `<b>${this.point.category}</b>: ${this.y.toFixed(2)} %`;}},
    plotOptions: {bar: {dataLabels: {enabled: true, style: {color: '#FFFFFF'}, formatter: function() {return this.y.toFixed(2) + ' %';}}}},
    legend: {enabled: false},
    series: [{name: 'FPS Percentage', data: sortedPercentageFPSData, colorByPoint: true, colors: colors}]
});

// Function to filter out the top and bottom 3% of FPS values
function filterOutliers(data) {
    data.sort((a, b) => a - b);
    return data.slice(Math.floor(data.length * 0.01), Math.ceil(data.length * 0.97));
}

// Function to count occurrences of each FPS value
function countFPS(data) {
    const counts = {};
    data.forEach(fps => {
        const roundedFPS = Math.round(fps);
        counts[roundedFPS] = (counts[roundedFPS] || 0) + 1;
    });

    let fpsArray = Object.keys(counts).map(key => [parseInt(key), counts[key]]).sort((a, b) => a[0] - b[0]);

    while (fpsArray.length > 100) {
        let minDiff = Infinity;
        let minIndex = -1;

        for (let i = 0; i < fpsArray.length - 1; i++) {
            const diff = fpsArray[i + 1][0] - fpsArray[i][0];
            if (diff < minDiff) {
                minDiff = diff;
                minIndex = i;
            }
        }

        fpsArray[minIndex][1] += fpsArray[minIndex + 1][1];
        fpsArray[minIndex][0] = (fpsArray[minIndex][0] + fpsArray[minIndex + 1][0]) / 2;
        fpsArray.splice(minIndex + 1, 1);
    }

    return fpsArray;
}

// Calculate counts for each dataset after filtering outliers
const densityData = fpsDataArrays.map(dataArray => ({name: dataArray.label, data: countFPS(filterOutliers(dataArray.data))}));

// Create the chart
Highcharts.chart('densityChart', {
    ...commonChartOptions,
    chart: {...commonChartOptions.chart, type: 'areaspline'},
    title: {...commonChartOptions.title, text: 'FPS Density'},
    xAxis: {...commonChartOptions.xAxis, title: {text: 'FPS', style: {color: '#FFFFFF'}}, labels: {style: {color: '#FFFFFF'}}}, // Show X-axis labels in white
    tooltip: {...commonChartOptions.tooltip, shared: true, formatter: function() {return `<b>${this.points[0].series.name}</b>: ${this.points[0].y} points at ~${Math.round(this.points[0].x)} FPS`;}},
    plotOptions: {areaspline: {fillOpacity: 0.5, marker: {enabled: false}}},
    legend: {...commonChartOptions.legend, enabled: true},
    series: densityData
});

function calculateStandardDeviation(data) {
    const mean = calculateAverage(data);
    const squaredDiffs = data.map(value => Math.pow(value - mean, 2));
    const avgSquaredDiff = calculateAverage(squaredDiffs);
    return Math.sqrt(avgSquaredDiff);
}

function calculateVariance(data) {
    const mean = calculateAverage(data);
    const squaredDiffs = data.map(value => Math.pow(value - mean, 2));
    return calculateAverage(squaredDiffs);
}

const sdvCategories = fpsDataArrays.map(dataArray => dataArray.label);
const standardDeviations = fpsDataArrays.map(dataArray => calculateStandardDeviation(dataArray.data));
const variances = fpsDataArrays.map(dataArray => calculateVariance(dataArray.data));

Highcharts.chart('sdvChart', {
    ...commonChartOptions,
    chart: {...commonChartOptions.chart, type: 'bar'},
    title: {...commonChartOptions.title, text: 'FPS Stability'},
    subtitle: {...commonChartOptions.subtitle, text: 'Measures of FPS consistency (std. dev.) and spread (variance). Less is better.'},
    xAxis: {...commonChartOptions.xAxis, categories: sdvCategories},
    yAxis: {...commonChartOptions.yAxis, title: {text: 'Value', align: 'high', style: {color: '#FFFFFF'}}},
    tooltip: {...commonChartOptions.tooltip, formatter: function() {return `<b>${this.series.name}</b>: ${this.y.toFixed(2)}`;}},
    plotOptions: {bar: {dataLabels: {enabled: true, style: {color: '#FFFFFF'}, formatter: function() {return this.y.toFixed(2);}}}},
    legend: {...commonChartOptions.legend, enabled: true},
    series: [
        {name: 'Std. Dev.', data: standardDeviations, color: '#FF5733'},
        {name: 'Variance', data: variances, color: '#33FF57'}
    ]
});
