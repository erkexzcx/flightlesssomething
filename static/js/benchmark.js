Highcharts.setOptions({
    chart: {
        animation: false
    },
    plotOptions: {
        series: {
            animation: false
        }
    }
});


var colors = Highcharts.getOptions().colors;

function getLineChartOptions(title, description, unit, maxY = null) {
    return {
        chart: {
            type: 'line',
            backgroundColor: null, // Set background to transparent
            style: {
                color: '#FFFFFF'
            },
            zooming: {
                type: 'x'
            }
        },
        title: {
            text: title,
            style: {
                color: '#FFFFFF',
                fontSize: '16px'
            }
        },
        subtitle: {
            text: description,
            style: {
                color: '#FFFFFF',
                fontSize: '12px'
            }
        },
        xAxis: {
            lineColor: '#FFFFFF',
            tickColor: '#FFFFFF',
            labels: {
                enabled: false
            }
        },
        yAxis: {
            title: {
                text: null
            },
            labels: {
                formatter: function() {
                    return this.value.toFixed(2) + ' ' + unit;
                },
                style: {
                    color: '#FFFFFF'
                }
            },
            gridLineColor: 'rgba(255, 255, 255, 0.1)',
            max: maxY
        },
        legend: {
            align: 'center',
            verticalAlign: 'bottom',
            itemStyle: {
                color: '#FFFFFF'
            }
        },
        tooltip: {
            shared: false,
            pointFormat: '<span style="color:{series.color}">{series.name}</span>: <b>{point.y:.2f} ' + unit + '</b><br/>', // Include unit in tooltip
            backgroundColor: '#1E1E1E',
            borderColor: '#FFFFFF',
            style: {
                color: '#FFFFFF'
            }
        },
        plotOptions: {
            line: {
                marker: {
                    enabled: false,
                    symbol: 'circle',
                    lineColor: null,
                    radius: 1.5,
                    states: {
                        hover: {
                            enabled: true,
                        }
                    }
                },
                lineWidth: 1,
                animation: false
            }
        },
        credits: {
            enabled: false
        },
        series: [],
        exporting: {
            buttons: {
                contextButton: {
                    menuItems: [
                        'viewFullscreen',
                        'printChart',
                        'separator',
                        'downloadPNG',
                        'downloadJPEG',
                        'downloadPDF',
                        'downloadSVG',
                        'separator',
                        'downloadCSV',
                        'downloadXLS'
                    ]
                }
            }
        }
    };
}

function getBarChartOptions(title, unit, maxY = null) {
    return {
        chart: {
            type: 'bar',
            backgroundColor: null, // Set background to transparent
            style: {
                color: '#FFFFFF'
            }
        },
        title: {
            text: title,
            style: {
                color: '#FFFFFF',
                fontSize: '16px'
            }
        },
        xAxis: {
            categories: [],
            title: {
                text: null
            },
            labels: {
                style: {
                    color: '#FFFFFF'
                }
            }
        },
        yAxis: {
            min: 0,
            max: maxY,
            title: {
                text: unit,
                align: 'high',
                style: {
                    color: '#FFFFFF'
                }
            },
            labels: {
                overflow: 'justify',
                style: {
                    color: '#FFFFFF'
                },
                formatter: function() {
                    return this.value.toFixed(2) + ' ' + unit;
                }
            },
            gridLineColor: 'rgba(255, 255, 255, 0.1)'
        },
        tooltip: {
            valueSuffix: ' ' + unit,
            backgroundColor: '#1E1E1E',
            borderColor: '#FFFFFF',
            style: {
                color: '#FFFFFF'
            },
            formatter: function() {
                return '<b>' + this.point.category + '</b>: ' + this.y.toFixed(2) + ' ' + unit;
            }
        },
        plotOptions: {
            bar: {
                dataLabels: {
                    enabled: true,
                    style: {
                        color: '#FFFFFF'
                    },
                    formatter: function() {
                        return this.y.toFixed(2) + ' ' + unit;
                    }
                }
            }
        },
        legend: {
            enabled: false, // Disable legend
        },
        credits: {
            enabled: false
        },
        series: []
    };
}

function createChart(chartId, title, description, unit, dataArrays, maxY = null) {
    var options = getLineChartOptions(title, description, unit, maxY);
    options.series = dataArrays.map(function(dataArray, index) {
        return {name: dataArray.label, data: dataArray.data, color: colors[index % colors.length]};
    });

    Highcharts.chart(chartId, options);
}

function createBarChart(chartId, title, unit, categories, data, colors, maxY = null) {
    var options = getBarChartOptions(title, unit, maxY);
    options.xAxis.categories = categories;
    options.series = [{
        name: title,
        data: data,
        colorByPoint: true,
        colors: colors
    }];

    Highcharts.chart(chartId, options);
}

function calculateAverage(data) {
    const sum = data.reduce((acc, value) => acc + value, 0);
    return sum / data.length;
}

function calculatePercentile(data, percentile) {
    data.sort((a, b) => a - b);
    const index = Math.ceil(percentile / 100 * data.length) - 1;
    return data[index];
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
var cpuLoadAverages = cpuLoadDataArrays.map(function(dataArray) {
    return calculateAverage(dataArray.data);
});

var gpuLoadAverages = gpuLoadDataArrays.map(function(dataArray) {
    return calculateAverage(dataArray.data);
});

// Create bar charts for average CPU and GPU load
createBarChart('cpuLoadSummaryChart', 'Average CPU Load', '%', cpuLoadDataArrays.map(function(dataArray) { return dataArray.label; }), cpuLoadAverages, colors, 100);
createBarChart('gpuLoadSummaryChart', 'Average GPU Load', '%', gpuLoadDataArrays.map(function(dataArray) { return dataArray.label; }), gpuLoadAverages, colors, 100);

// Calculate and render min, max, and average FPS
var categories = [];
var minFPSData = [];
var avgFPSData = [];
var maxFPSData = [];

fpsDataArrays.forEach(function(dataArray) {
    var minFPS = calculatePercentile(dataArray.data, 1);
    var avgFPS = calculateAverage(dataArray.data);
    var maxFPS = calculatePercentile(dataArray.data, 97);

    categories.push(dataArray.label);
    minFPSData.push(minFPS);
    avgFPSData.push(avgFPS);
    maxFPSData.push(maxFPS);
});

Highcharts.chart('minMaxAvgChart', {
    chart: {
        type: 'bar',
        backgroundColor: null
    },
    title: {
        text: 'Min/Avg/Max FPS',
        style: {
            color: '#FFFFFF',
            fontSize: '16px'
        }
    },
    subtitle: {
        text: 'More is better',
        style: {
            color: '#FFFFFF'
        }
    },
    xAxis: {
        categories: categories,
        title: {
            text: null
        },
        labels: {
            style: {
                color: '#FFFFFF'
            }
        }
    },
    yAxis: {
        min: 0,
        title: {
            text: 'FPS',
            align: 'high',
            style: {
                color: '#FFFFFF'
            }
        },
        labels: {
            overflow: 'justify',
            style: {
                color: '#FFFFFF'
            }
        },
        gridLineColor: 'rgba(255, 255, 255, 0.1)'
    },
    tooltip: {
        valueSuffix: ' FPS',
        backgroundColor: '#1E1E1E',
        borderColor: '#FFFFFF',
        style: {
            color: '#FFFFFF'
        },
        formatter: function() {
            return '<b>' + this.series.name + '</b>: ' + this.y.toFixed(2) + ' FPS';
        }
    },
    plotOptions: {
        bar: {
            dataLabels: {
                enabled: true,
                style: {
                    color: '#FFFFFF'
                },
                formatter: function() {
                    return this.y.toFixed(2) + ' fps';
                }
            }
        }
    },
    legend: {
        reversed: true,
        itemStyle: {
            color: '#FFFFFF'
        }
    },
    credits: {
        enabled: false
    },
    series: [{
        name: '97th',
        data: maxFPSData,
        color: '#00FF00'
    }, {
        name: 'AVG',
        data: avgFPSData,
        color: '#0000FF'
    }, {
        name: '1%',
        data: minFPSData,
        color: '#FF0000'
    }]
});

// Calculate average FPS for each filename
var avgFPSData = fpsDataArrays.map(function(dataArray) {
    return calculateAverage(dataArray.data);
});

// Calculate FPS as a percentage of the first element
var firstFPS = avgFPSData[0];
var percentageFPSData = avgFPSData.map(function(fps) {
    return (fps / firstFPS) * 100;
});

// Create bar chart for FPS percentage
Highcharts.chart('avgChart', {
    chart: {
        type: 'bar',
        backgroundColor: null
    },
    title: {
        text: 'Average FPS in %',
        style: {
            color: '#FFFFFF',
            fontSize: '16px'
        }
    },
    xAxis: {
        categories: fpsDataArrays.map(function(dataArray) { return dataArray.label; }),
        title: {
            text: null
        },
        labels: {
            style: {
                color: '#FFFFFF'
            }
        }
    },
    yAxis: {
        min: 0,
        title: {
            text: 'Percentage (%)',
            align: 'high',
            style: {
                color: '#FFFFFF'
            }
        },
        labels: {
            overflow: 'justify',
            style: {
                color: '#FFFFFF'
            }
        },
        gridLineColor: 'rgba(255, 255, 255, 0.1)'
    },
    tooltip: {
        valueSuffix: ' %',
        backgroundColor: '#1E1E1E',
        borderColor: '#FFFFFF',
        style: {
            color: '#FFFFFF'
        },
        formatter: function() {
            return '<b>' + this.point.category + '</b>: ' + this.y.toFixed(2) + ' %';
        }
    },
    plotOptions: {
        bar: {
            dataLabels: {
                enabled: true,
                style: {
                    color: '#FFFFFF'
                },
                formatter: function() {
                    return this.y.toFixed(2) + ' %';
                }
            }
        }
    },
    legend: {
        enabled: false
    },
    credits: {
        enabled: false
    },
    series: [{
        name: 'FPS Percentage',
        data: percentageFPSData,
        colorByPoint: true,
        colors: colors
    }]
});

// Function to filter out the top and bottom 3% of FPS values
function filterOutliers(data) {
    data.sort((a, b) => a - b);
    var start = Math.floor(data.length * 0.01); // Ignore bottom 1%
    var end = Math.ceil(data.length * 0.97); // Ignore top 1%
    return data.slice(start, end);
}

// Function to count occurrences of each FPS value
function countFPS(data) {
    var counts = {};
    data.forEach(function(fps) {
        var roundedFPS = Math.round(fps);
        counts[roundedFPS] = (counts[roundedFPS] || 0) + 1;
    });

    var fpsArray = Object.keys(counts).map(function(key) {
        return [parseInt(key), counts[key]];
    }).sort(function(a, b) {
        return a[0] - b[0];
    });

    // Combine closest FPS values until we have 100 or fewer points
    while (fpsArray.length > 100) {
        var minDiff = Infinity;
        var minIndex = -1;

        // Find the pair with the smallest difference
        for (var i = 0; i < fpsArray.length - 1; i++) {
            var diff = fpsArray[i + 1][0] - fpsArray[i][0];
            if (diff < minDiff) {
                minDiff = diff;
                minIndex = i;
            }
        }

        // Combine the closest pair
        fpsArray[minIndex][1] += fpsArray[minIndex + 1][1];
        fpsArray[minIndex][0] = (fpsArray[minIndex][0] + fpsArray[minIndex + 1][0]) / 2;
        fpsArray.splice(minIndex + 1, 1);
    }

    return fpsArray;
}

// Calculate counts for each dataset after filtering outliers
var densityData = fpsDataArrays.map(function(dataArray) {
    var filteredData = filterOutliers(dataArray.data);
    return {
        name: dataArray.label,
        data: countFPS(filteredData)
    };
});

// Create the chart
Highcharts.chart('densityChart', {
    chart: {
        type: 'areaspline',
        backgroundColor: null
    },
    title: {
        text: 'FPS Density',
        style: {
            color: '#FFFFFF',
            fontSize: '16px'
        }
    },
    xAxis: {
        title: {
            text: 'FPS',
            style: {
                color: '#FFFFFF'
            }
        },
        labels: {
            style: {
                color: '#FFFFFF'
            }
        }
    },
    yAxis: {
        title: {
            text: 'Count',
            style: {
                color: '#FFFFFF'
            }
        },
        labels: {
            style: {
                color: '#FFFFFF'
            }
        },
        gridLineColor: 'rgba(255, 255, 255, 0.1)'
    },
    tooltip: {
        shared: true,
        backgroundColor: '#1E1E1E',
        borderColor: '#FFFFFF',
        style: {
            color: '#FFFFFF'
        },
        formatter: function() {
            var points = this.points;
            var tooltipText = '<b>' + points[0].series.name + '</b>: ' + points[0].y + ' points at ~' + Math.round(points[0].x) + ' FPS';
            return tooltipText;
        }
    },
    plotOptions: {
        areaspline: {
            fillOpacity: 0.5,
            marker: {
                enabled: false
            }
        }
    },
    legend: {
        enabled: true,
        itemStyle: {
            color: '#FFFFFF'
        }
    },
    credits: {
        enabled: false
    },
    series: densityData
});

function calculateSpikes(data, threshold) {
    if (data.length < 6) {
        throw new Error("Data length must be greater than or equal to 6.");
    }

    let spikeCount = 0;

    // Helper function to calculate the moving average with a minimum of 6 points
    function movingAverage(arr, index) {
        const windowSize = Math.max(6, Math.ceil(arr.length * 0.05)); // 5 % of the data
        const halfWindowSize = Math.floor(windowSize / 2);
        const start = Math.max(0, index - halfWindowSize);
        const end = Math.min(arr.length - 1, index + halfWindowSize);
        const actualWindowSize = end - start + 1;

        let sum = 0;
        for (let i = start; i <= end; i++) {
            sum += arr[i];
        }
        return sum / actualWindowSize;
    }

    for (let i = 0; i < data.length; i++) {
        const currentPoint = data[i];
        const movingAvg = movingAverage(data, i);

        const change = Math.abs(currentPoint - movingAvg) / movingAvg * 100;

        if (change > threshold) {
            spikeCount++;
        }
    }

    return (spikeCount / data.length) * 100;
}

function updateSpikesChart(threshold) {
    document.getElementById('spikeThresholdValue').innerText = threshold + '%';

    var spikePercentages = fpsDataArrays.map(function(dataArray) {
        return calculateSpikes(dataArray.data, threshold);
    });

    Highcharts.chart('spikesChart', {
        chart: {
            type: 'bar',
            backgroundColor: null
        },
        title: {
            text: 'FPS Spikes',
            style: {
                color: '#FFFFFF',
                fontSize: '16px'
            }
        },
        subtitle: {
            text: 'Less is better',
            style: {
                color: '#FFFFFF',
                fontSize: '12px'
            }
        },
        xAxis: {
            categories: categories,
            title: {
                text: null
            },
            labels: {
                style: {
                    color: '#FFFFFF'
                }
            }
        },
        yAxis: {
            min: 0,
            title: {
                text: 'Percentage (%)',
                align: 'high',
                style: {
                    color: '#FFFFFF'
                }
            },
            labels: {
                overflow: 'justify',
                style: {
                    color: '#FFFFFF'
                }
            },
            gridLineColor: 'rgba(255, 255, 255, 0.1)'
        },
        tooltip: {
            valueSuffix: ' %',
            backgroundColor: '#1E1E1E',
            borderColor: '#FFFFFF',
            style: {
                color: '#FFFFFF'
            },
            formatter: function() {
                return '<b>' + this.point.category + '</b>: ' + this.y.toFixed(2) + ' %';
            }
        },
        plotOptions: {
            bar: {
                dataLabels: {
                    enabled: true,
                    style: {
                        color: '#FFFFFF'
                    },
                    formatter: function() {
                        return this.y.toFixed(2) + ' %';
                    }
                }
            }
        },
        legend: {
            enabled: false
        },
        credits: {
            enabled: false
        },
        series: [{
            name: 'Spike Percentage',
            data: spikePercentages,
            colorByPoint: true,
            colors: colors
        }]
    });
}

// Initial render of spikes chart
updateSpikesChart(document.getElementById('spikeThreshold').value);
