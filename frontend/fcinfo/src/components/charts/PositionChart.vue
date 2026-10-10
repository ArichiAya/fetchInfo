<script setup lang="ts">
/**
 * 持仓分布图：横向堆叠条形（部分-整体）
 * 段间以 2px 表面色间隙分隔；过窄的段不写内嵌标签，由图例 + 悬浮提示承载
 */
import { computed, ref } from 'vue'
import * as echarts from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsCoreOption } from 'echarts/core'
import type { PositionItemWithValue } from '@/types/stock'
import { LIGHT, DARK } from './palette'
import { useEChart } from './useEChart'

echarts.use([BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

const props = defineProps<{
  positions: PositionItemWithValue[]
  dark?: boolean
}>()

const elRef = ref<HTMLElement>()

const option = computed<EChartsCoreOption>(() => {
  const p = props.dark ? DARK : LIGHT
  const colors = [p.blue, p.orange, p.aqua]
  // 文字用黑/白墨，按段填充色的亮度选择（青在亮色模式下为浅色 → 深墨）
  const inks = ['#ffffff', '#ffffff', props.dark ? '#ffffff' : '#0b0b0b']

  const total = props.positions.reduce((s, x) => s + x.marketValue, 0)
  const pcts = props.positions.map((x) =>
    total > 0 ? Number(((x.marketValue / total) * 100).toFixed(1)) : 0
  )

  const series = props.positions.map((pos, i) => ({
    name: pos.name,
    type: 'bar' as const,
    stack: 'positions',
    // 2px 表面色间隙分离相邻段
    itemStyle: {
      color: colors[i % colors.length],
      borderWidth: 2,
      borderColor: p.surface,
      // 只有最后一个段（数据端）圆角 4px，基线端保持直角
      borderRadius: i === props.positions.length - 1 ? [0, 4, 4, 0] : [0, 0, 0, 0]
    },
    data: [pcts[i]],
    barWidth: 36,
    label: {
      // 段宽 ≥ 8% 才写内嵌标签，否则交给图例 + tooltip
      show: pcts[i] >= 8,
      position: 'inside',
      color: inks[i % inks.length],
      fontSize: 12,
      formatter: `{c}%`
    },
    emphasis: { itemStyle: { opacity: 0.85 } }
  }))

  return {
    animation: false,
    grid: { left: 8, right: 24, top: 12, bottom: 44, containLabel: true },
    legend: {
      bottom: 0,
      itemWidth: 10,
      itemHeight: 10,
      itemGap: 16,
      textStyle: { color: p.textSecondary, fontSize: 12 }
    },
    tooltip: {
      trigger: 'item',
      backgroundColor: p.surface,
      borderColor: p.gridline,
      textStyle: { color: p.textPrimary, fontSize: 12 },
      extraCssText: 'box-shadow: 0 2px 8px rgba(0,0,0,0.12); border-radius: 6px;',
      formatter: (params: any) => {
        const idx = params.seriesIndex as number
        const pos = props.positions[idx]
        if (!pos) return ''
        const value = `¥${pos.marketValue.toLocaleString()}`
        const pct = pcts[idx]
        return `<b>${value}</b><br/>${pos.name} · ${pct}%`
      }
    },
    xAxis: {
      type: 'value',
      max: 100,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: p.muted, fontSize: 11, formatter: '{value}%' },
      splitLine: { lineStyle: { color: p.gridline, width: 1 } }
    },
    yAxis: {
      type: 'category',
      data: [''],
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { show: false }
    },
    series
  }
})

useEChart(elRef, option)
</script>

<template>
  <div v-if="positions.length" ref="elRef" class="position-chart" />
  <div v-else class="empty">暂无持仓</div>
</template>

<style scoped>
.position-chart {
  width: 100%;
  height: 140px;
}

.empty {
  height: 140px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #898781;
  font-size: 13px;
}
</style>
