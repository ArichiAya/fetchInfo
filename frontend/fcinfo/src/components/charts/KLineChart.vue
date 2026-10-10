<script setup lang="ts">
/**
 * 日K线图：蜡烛图（红涨空心 / 绿跌实心）+ MA5/MA10 均线
 * 下方附成交量（红/绿随涨跌，独立坐标轴）
 * 空心/实心 + 图例为色盲安全的二次编码
 */
import { computed, ref } from 'vue'
import * as echarts from 'echarts/core'
import { CandlestickChart, LineChart, BarChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  AxisPointerComponent,
  DataZoomComponent
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsCoreOption } from 'echarts/core'
import type { KLineDay } from '@/types/stock'
import { LIGHT, DARK } from './palette'
import { useEChart } from './useEChart'

echarts.use([
  CandlestickChart,
  LineChart,
  BarChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  AxisPointerComponent,
  DataZoomComponent,
  CanvasRenderer
])

const props = withDefaults(
  defineProps<{
    data: KLineDay[]
    dark?: boolean
    /** K线周期，决定时间轴标签格式 */
    period?: string
  }>(),
  { period: '1d' }
)

/** 时间轴标签：日K 只显示日期，分钟K 显示日期 + 时分 */
function axisLabel(ts: string): string {
  const [date, time = ''] = ts.split('T')
  return props.period === '1d' ? date : `${date} ${time.slice(0, 5)}`
}

const elRef = ref<HTMLElement>()

/** 移动均线：前 n-1 天为 null，从第 n 天开始有值 */
function movingAverage(values: number[], window: number): (number | null)[] {
  const result: (number | null)[] = []
  let sum = 0
  for (let i = 0; i < values.length; i++) {
    sum += values[i]
    if (i >= window) sum -= values[i - window]
    result.push(i >= window - 1 ? Number((sum / window).toFixed(2)) : null)
  }
  return result
}

const option = computed<EChartsCoreOption>(() => {
  const p = props.dark ? DARK : LIGHT
  const dates = props.data.map((d) => axisLabel(d.ts))
  // 蜡烛数据：[open, close, low, high]
  const candles = props.data.map((d) => [d.open, d.close, d.low, d.high])
  const closes = props.data.map((d) => d.close)
  const ma5 = movingAverage(closes, 5)
  const ma10 = movingAverage(closes, 10)
  // 成交量颜色随涨跌
  const volumes = props.data.map((d) => ({
    value: d.volume,
    itemStyle: { color: d.close >= d.open ? p.red : p.green, opacity: 0.55 }
  }))

  return {
    animation: false,
    grid: [
      { left: 56, right: 24, top: 40, height: '56%' },
      { left: 56, right: 24, top: '74%', height: '14%' }
    ],
    legend: {
      top: 8,
      itemWidth: 16,
      itemHeight: 2,
      textStyle: { color: p.textSecondary, fontSize: 12 },
      data: ['日K', 'MA5', 'MA10']
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        lineStyle: { color: p.muted, width: 1 },
        crossStyle: { color: p.muted, width: 1 },
        label: { color: p.textPrimary, backgroundColor: p.surface }
      },
      backgroundColor: p.surface,
      borderColor: p.gridline,
      textStyle: { color: p.textPrimary, fontSize: 12 },
      extraCssText: 'box-shadow: 0 2px 8px rgba(0,0,0,0.12); border-radius: 6px;'
    },
    xAxis: [
      {
        type: 'category',
        data: dates,
        boundaryGap: true,
        axisLine: { lineStyle: { color: p.baseline } },
        axisTick: { show: false },
        axisLabel: { color: p.muted, fontSize: 11 },
        splitLine: { show: false }
      },
      {
        type: 'category',
        gridIndex: 1,
        data: dates,
        boundaryGap: true,
        axisLine: { lineStyle: { color: p.baseline } },
        axisTick: { show: false },
        axisLabel: { show: false },
        splitLine: { show: false }
      }
    ],
    yAxis: [
      {
        type: 'value',
        scale: true,
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          color: p.muted,
          fontSize: 11,
          formatter: (v: number) => v.toLocaleString()
        },
        splitLine: { lineStyle: { color: p.gridline, width: 1 } }
      },
      {
        type: 'value',
        gridIndex: 1,
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: { show: false },
        splitLine: { show: false }
      }
    ],
    dataZoom: [
      { type: 'inside', xAxisIndex: [0, 1], start: 30, end: 100 },
      {
        type: 'slider',
        xAxisIndex: [0, 1],
        bottom: 4,
        height: 16,
        borderColor: p.gridline,
        backgroundColor: 'transparent',
        fillerColor: p.gridline,
        handleStyle: { color: p.muted },
        textStyle: { color: p.muted, fontSize: 10 }
      }
    ],
    series: [
      {
        name: '日K',
        type: 'candlestick',
        data: candles,
        // 红涨空心（表面色填充）/ 绿跌实心 —— 形状即二次编码
        itemStyle: {
          color: 'transparent',
          color0: p.green,
          borderColor: p.red,
          borderColor0: p.green,
          borderWidth: 1.5
        },
        emphasis: { itemStyle: { borderWidth: 2 } }
      },
      {
        name: 'MA5',
        type: 'line',
        data: ma5,
        symbol: 'none',
        lineStyle: { width: 1.5, color: p.blue },
        endLabel: { show: true, color: p.textSecondary, fontSize: 11 },
        emphasis: { disabled: true }
      },
      {
        name: 'MA10',
        type: 'line',
        data: ma10,
        symbol: 'none',
        lineStyle: { width: 1.5, color: p.amber },
        endLabel: { show: true, color: p.textSecondary, fontSize: 11 },
        emphasis: { disabled: true }
      },
      {
        name: '成交量',
        type: 'bar',
        xAxisIndex: 1,
        yAxisIndex: 1,
        data: volumes,
        emphasis: { disabled: true }
      }
    ]
  }
})

useEChart(elRef, option)
</script>

<template>
  <div ref="elRef" class="kline-chart" />
</template>

<style scoped>
.kline-chart {
  width: 100%;
  height: 320px;
}
</style>
