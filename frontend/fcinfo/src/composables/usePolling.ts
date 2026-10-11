import { onScopeDispose, ref } from 'vue'

/**
 * 定时轮询：start() 立刻跑一轮，之后按 interval 重复。
 *
 * start/stop 只是开关，任务本身读的是响应式数据（比如当前选中的股票代码），
 * 所以切换股票不需要重启轮询，下一轮自然就用新值。
 *
 * 三件容易忘的事在这里处理掉了：
 * - 标签页隐藏时停手，重新可见立刻补一轮（后台空转纯属浪费）
 * - 用 setTimeout 链而不是 setInterval，上一轮没结束就不会发下一轮
 * - 组件卸载时自动停止，并 abort 在途请求
 */
export function usePolling(task: (signal: AbortSignal) => unknown, interval: number) {
  const active = ref(false)
  let timer: ReturnType<typeof setTimeout> | undefined
  let controller: AbortController | undefined

  const clearTimer = () => {
    if (timer !== undefined) {
      clearTimeout(timer)
      timer = undefined
    }
  }

  const run = async () => {
    if (!active.value || document.hidden) return
    controller = new AbortController()
    try {
      await task(controller.signal)
    } catch {
      // 错误由 task 自己提示，这里兜底，保证轮询不中断
    } finally {
      if (active.value && !document.hidden) timer = setTimeout(run, interval)
    }
  }

  const onVisibilityChange = () => {
    if (!active.value) return
    clearTimer()
    if (document.hidden) {
      controller?.abort()
    } else {
      void run()
    }
  }

  const start = () => {
    if (active.value) return
    active.value = true
    document.addEventListener('visibilitychange', onVisibilityChange)
    void run()
  }

  const stop = () => {
    if (!active.value) return
    active.value = false
    clearTimer()
    controller?.abort()
    document.removeEventListener('visibilitychange', onVisibilityChange)
  }

  onScopeDispose(stop)

  return { start, stop, active }
}
