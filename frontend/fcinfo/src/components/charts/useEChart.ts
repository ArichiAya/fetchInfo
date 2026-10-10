/**
 * ECharts 初始化/重绘/销毁的通用逻辑
 * 用 ResizeObserver 监听容器尺寸 —— 覆盖窗口缩放和
 * 隐藏容器变为可见（如 el-tabs 切换）两种场景
 */
import { onMounted, onBeforeUnmount, watch, type Ref } from 'vue'
import * as echarts from 'echarts/core'
import type { EChartsCoreOption } from 'echarts/core'

export function useEChart(
  elRef: Ref<HTMLElement | undefined>,
  option: Ref<EChartsCoreOption>
) {
  let chart: ReturnType<typeof echarts.init> | null = null
  let observer: ResizeObserver | null = null

  const render = () => {
    if (!elRef.value) return
    if (!chart) chart = echarts.init(elRef.value)
    chart.setOption(option.value)
  }

  const resize = () => chart?.resize()

  onMounted(() => {
    render()
    observer = new ResizeObserver(resize)
    if (elRef.value) observer.observe(elRef.value)
  })

  // option 为 computed，依赖变化即整体重建
  watch(option, render)

  onBeforeUnmount(() => {
    observer?.disconnect()
    observer = null
    chart?.dispose()
    chart = null
  })
}
