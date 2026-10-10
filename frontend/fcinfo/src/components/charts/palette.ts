/**
 * 图表色板（已通过色板校验器验证：色盲安全、对比度达标）
 * 亮/暗两套取自同一组色相，暗色为同色相针对暗表面的步进
 */
export interface ChartPalette {
  surface: string
  textPrimary: string
  textSecondary: string
  muted: string
  gridline: string
  baseline: string
  blue: string
  amber: string
  green: string
  red: string
  orange: string
  aqua: string
}

export const LIGHT: ChartPalette = {
  surface: '#fcfcfb',
  textPrimary: '#0b0b0b',
  textSecondary: '#52514e',
  muted: '#898781',
  gridline: '#e1e0d9',
  baseline: '#c3c2b7',
  blue: '#2a78d6',
  amber: '#eda100',
  green: '#008300',
  red: '#e34948',
  orange: '#eb6834',
  aqua: '#1baf7a'
}

export const DARK: ChartPalette = {
  surface: '#1a1a19',
  textPrimary: '#ffffff',
  textSecondary: '#c3c2b7',
  muted: '#898781',
  gridline: '#2c2c2a',
  baseline: '#383835',
  blue: '#3987e5',
  amber: '#c98500',
  green: '#008300',
  red: '#e66767',
  orange: '#d95926',
  aqua: '#199e70'
}
