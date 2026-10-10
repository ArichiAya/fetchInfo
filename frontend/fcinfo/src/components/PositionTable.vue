<script setup lang="ts">
/**
 * 持仓明细表格 —— 图表的表格视图，所有数值不依赖颜色即可获取
 */
import type { PositionItemWithValue } from '@/types/stock'

defineProps<{
  positions: PositionItemWithValue[]
}>()

/** 盈亏颜色：红涨绿跌（与K线一致） */
const profitClass = (profit: number) => (profit >= 0 ? 'profit-up' : 'profit-down')

const fmt = (v: number) => v.toLocaleString()
</script>

<template>
  <el-table :data="positions" size="small" class="position-table">
    <el-table-column prop="name" label="名称" min-width="90" />
    <el-table-column prop="code" label="代码" width="80" />
    <el-table-column label="持股" width="80" align="right">
      <template #default="scope">{{ fmt(scope.row.shares) }}</template>
    </el-table-column>
    <el-table-column label="现价" width="90" align="right">
      <template #default="scope">{{ scope.row.price.toFixed(2) }}</template>
    </el-table-column>
    <el-table-column label="市值" width="110" align="right">
      <template #default="scope">¥{{ fmt(scope.row.marketValue) }}</template>
    </el-table-column>
    <el-table-column label="盈亏" min-width="110" align="right">
      <template #default="scope">
        <span :class="profitClass(scope.row.profit)">
          {{ scope.row.profit >= 0 ? '+' : '' }}{{ fmt(scope.row.profit) }}
          （{{ scope.row.profitPercent >= 0 ? '+' : '' }}{{ scope.row.profitPercent }}%）
        </span>
      </template>
    </el-table-column>
  </el-table>
</template>

<style scoped>
.position-table {
  margin-top: 8px;
  font-variant-numeric: tabular-nums;
}

.profit-up {
  color: var(--rise-color);
}

.profit-down {
  color: var(--fall-color);
}
</style>
