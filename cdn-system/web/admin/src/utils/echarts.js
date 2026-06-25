let echartsPromise

export const loadEcharts = () => {
  if (!echartsPromise) {
    echartsPromise = import('./echarts-core').then(module => module.default)
  }
  return echartsPromise
}
