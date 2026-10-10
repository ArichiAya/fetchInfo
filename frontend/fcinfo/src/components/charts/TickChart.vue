<script setup lang="ts">
/**
 * 分时走势图：价格线（蓝）+ 均价线（琥珀），下方附成交量（灰，独立坐标轴）
 */
import { computed, ref } from 'vue'
import * as echarts from 'echarts/core'
import { LineChart, BarChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  AxisPointerComponent
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsCoreOption } from 'echarts/core'
import type { TickPoint } from '@/types/stock'
import { LIGHT, DARK } from './palette'
import { useEChart } from './useEChart'

echarts.use([
  LineChart,
  BarChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  AxisPointerComponent,
  CanvasRenderer
])

const props = defineProps<{
  data: TickPoint[]
  dark?: boolean
}>()

const elRef = ref<HTMLElement>()

const option = computed<EChartsCoreOption>(() => {
  const p = props.dark ? DARK : LIGHT
  const times = props.data.map((d) => d.time)
  const prices = props.data.map((d) => d.price)
  const avgPrices = props.data.map((d) => d.avgPrice)
  const volumes = props.data.map((d) => d.volume)

  return {
    animation: false,
    grid: [
      { left: 56, right: 24, top: 40, height: '62%' },
      { left: 56, right: 24, top: '78%', height: '14%' }
    ],
    legend: {
      top: 8,
      itemWidth: 16,
      itemHeight: 2,
      textStyle: { color: p.textSecondary, fontSize: 12 }
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
        data: times,
        boundaryGap: false,
        axisLine: { lineStyle: { color: p.baseline } },
        axisTick: { show: false },
        axisLabel: {
          color: p.muted,
          fontSize: 11,
          interval: 29 // 每 30 分钟标一个刻度
        },
        splitLine: { show: false }
      },
      {
        type: 'category',
        gridIndex: 1,
        data: times,
        boundaryGap: false,
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
    series: [
      {
        name: '价格',
        type: 'line',
        data: prices,
        symbol: 'none',
        lineStyle: { width: 2, color: p.blue },
        // 面积填充：序列色 ~10% 透明度的洗色
        areaStyle: { color: p.blue, opacity: 0.1 },
        // 端点直接标注（文字用文字色，不用序列色）
        endLabel: {
          show: true,
          color: p.textSecondary,
          fontSize: 11,
          formatter: (params: { value: number }) => `现价 ${params.value}`
        },
        emphasis: { disabled: true }
      },
      {
        name: '均价',
        type: 'line',
        data: avgPrices,
        symbol: 'none',
        lineStyle: { width: 2, color: p.amber },
        endLabel: {
          show: true,
          color: p.textSecondary,
          fontSize: 11,
          formatter: (params: { value: number }) => `均价 ${params.value}`
        },
        emphasis: { disabled: true }
      },
      {
        name: '成交量',
        type: 'bar',
        xAxisIndex: 1,
        yAxisIndex: 1,
        data: volumes,
        // 成交量是上下文，用灰（退强调色）
        itemStyle: { color: p.muted, opacity: 0.45 },
        emphasis: { disabled: true }
      }
    ]
  }
})

useEChart(elRef, option)
</script>

<template>
  <div ref="elRef" class="tick-chart" />
</template>

<style scoped>
.tick-chart {
  width: 100%;
  height: 280px;
}
</style>
