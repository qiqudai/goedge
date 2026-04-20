(function () {
  window.cnn = window.cnn || {};
  window.cnn.dashboard = window.cnn.dashboard || {};

  function hexToRgb(hex) {
    if (!hex || hex[0] !== '#') return null;
    var value = hex.substring(1);
    if (value.length === 3) {
      value = value[0] + value[0] + value[1] + value[1] + value[2] + value[2];
    }
    if (value.length !== 6) return null;
    var r = parseInt(value.substring(0, 2), 16);
    var g = parseInt(value.substring(2, 4), 16);
    var b = parseInt(value.substring(4, 6), 16);
    if (Number.isNaN(r) || Number.isNaN(g) || Number.isNaN(b)) return null;
    return { r: r, g: g, b: b };
  }

  function buildGradient(color) {
    if (!window.echarts || !window.echarts.graphic) return color;
    var rgb = hexToRgb(color);
    if (!rgb) return color;
    var start = 'rgba(' + rgb.r + ',' + rgb.g + ',' + rgb.b + ',0.25)';
    var end = 'rgba(' + rgb.r + ',' + rgb.g + ',' + rgb.b + ',0.05)';
    return new window.echarts.graphic.LinearGradient(0, 0, 0, 1, [
      { offset: 0, color: start },
      { offset: 1, color: end }
    ]);
  }

  function getThemeColors() {
    var isDark = document.documentElement.getAttribute('data-theme') === 'dark';
    return isDark
      ? { axisLabel: '#b0b6c3', axisLine: '#3a3f47', splitLine: '#343a43' }
      : { axisLabel: '#606266', axisLine: '#e4e7ed', splitLine: '#eef1f6' };
  }

  window.cnn.dashboard.render = function (domId, payload) {
    if (!payload || !window.echarts) return;
    var dom = document.getElementById(domId);
    if (!dom) return;

    var chart = dom.__cnnChart;
    if (!chart) {
      chart = window.echarts.init(dom);
      dom.__cnnChart = chart;
    }

    var series = payload.series || {};
    var color = series.color || '#3a7bff';
    var tc = getThemeColors();
    var option = {
      tooltip: { trigger: 'axis' },
      grid: { left: 40, right: 20, top: 20, bottom: 30 },
      xAxis: {
        type: 'category',
        data: payload.xAxis || [],
        boundaryGap: false,
        axisLine: { lineStyle: { color: tc.axisLine } },
        axisLabel: { color: tc.axisLabel }
      },
      yAxis: {
        type: 'value',
        axisLine: { show: false },
        splitLine: { lineStyle: { color: tc.splitLine } },
        axisLabel: { color: tc.axisLabel }
      },
      series: [
        {
          name: series.name || '',
          type: 'line',
          data: series.data || [],
          smooth: true,
          symbol: 'circle',
          symbolSize: 4,
          lineStyle: { color: color, width: 2 },
          itemStyle: { color: color },
          areaStyle: { color: buildGradient(color) }
        }
      ]
    };

    chart.setOption(option, true);

    if (!dom.__cnnResize) {
      dom.__cnnResize = function () {
        if (dom.__cnnChart) {
          dom.__cnnChart.resize();
        }
      };
      window.addEventListener('resize', dom.__cnnResize);
    }
  };
})();

(function () {
  window.cnn = window.cnn || {};
  window.cnn.realtime = window.cnn.realtime || {};

  window.cnn.realtime.renderTraffic = function (domId, payload) {
    if (!payload || !window.echarts) return;
    var dom = document.getElementById(domId);
    if (!dom) return;

    var chart = dom.__cnnChart;
    if (!chart) {
      chart = window.echarts.init(dom);
      dom.__cnnChart = chart;
    }

    var isDark = document.documentElement.getAttribute('data-theme') === 'dark';
    var tc = isDark
      ? { axisLabel: '#b0b6c3', axisLine: '#3a3f47', splitLine: '#343a43' }
      : { axisLabel: '#606266', axisLine: '#e4e7ed', splitLine: '#eef1f6' };

    var colors = ['#3a7bff', '#4bc58e'];
    var legendData = (payload.series || []).map(function (s) { return s.name; });

    var option = {
      tooltip: { trigger: 'axis' },
      legend: { data: legendData, textStyle: { color: tc.axisLabel } },
      grid: { left: 60, right: 20, top: 40, bottom: 30 },
      xAxis: {
        type: 'category',
        data: payload.xAxis || [],
        boundaryGap: false,
        axisLine: { lineStyle: { color: tc.axisLine } },
        axisLabel: { color: tc.axisLabel, rotate: 30 }
      },
      yAxis: {
        type: 'value',
        axisLine: { show: false },
        splitLine: { lineStyle: { color: tc.splitLine } },
        axisLabel: { color: tc.axisLabel }
      },
      series: (payload.series || []).map(function (s, i) {
        var color = (s.lineStyle && s.lineStyle.color) || colors[i % colors.length];
        return {
          name: s.name,
          type: 'line',
          data: s.data || [],
          smooth: true,
          lineStyle: { color: color, width: 2 },
          itemStyle: { color: color }
        };
      })
    };

    chart.setOption(option, true);

    if (!dom.__cnnResize) {
      dom.__cnnResize = function () {
        if (dom.__cnnChart) dom.__cnnChart.resize();
      };
      window.addEventListener('resize', dom.__cnnResize);
    }
  };
})();

(function () {
  window.cnn = window.cnn || {};
  window.cnn.usage = window.cnn.usage || {};

  window.cnn.usage.renderChart = function (domId, xAxis, values, unit) {
    if (!window.echarts) return;
    var dom = document.getElementById(domId);
    if (!dom) return;

    var chart = dom.__cnnChart;
    if (!chart) {
      chart = window.echarts.init(dom);
      dom.__cnnChart = chart;
    }

    var isDark = document.documentElement.getAttribute('data-theme') === 'dark';
    var tc = isDark
      ? { axisLabel: '#b0b6c3', axisLine: '#3a3f47', splitLine: '#343a43' }
      : { axisLabel: '#606266', axisLine: '#e4e7ed', splitLine: '#eef1f6' };

    var color = '#3a7bff';
    var option = {
      tooltip: { trigger: 'axis' },
      grid: { left: 60, right: 20, top: 20, bottom: 40 },
      xAxis: {
        type: 'category',
        data: xAxis || [],
        boundaryGap: false,
        axisLine: { lineStyle: { color: tc.axisLine } },
        axisLabel: { color: tc.axisLabel, rotate: 30 }
      },
      yAxis: {
        type: 'value',
        name: unit || '',
        axisLine: { show: false },
        splitLine: { lineStyle: { color: tc.splitLine } },
        axisLabel: { color: tc.axisLabel }
      },
      series: [{
        name: '流量',
        type: 'line',
        smooth: true,
        data: values || [],
        lineStyle: { color: color, width: 2 },
        itemStyle: { color: color },
        areaStyle: {}
      }]
    };

    chart.setOption(option, true);

    if (!dom.__cnnResize) {
      dom.__cnnResize = function () {
        if (dom.__cnnChart) dom.__cnnChart.resize();
      };
      window.addEventListener('resize', dom.__cnnResize);
    }
  };
})();
