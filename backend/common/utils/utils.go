// Package utils 用于存放公共的工具函数
package utils

// SafeRatio:分母<=0 时返回兜底值,拒绝 NaN/Inf 进入下游
func SafeRatio(num, den, fallback float64) float64 {
	if den <= 0 {
		return fallback
	}
	return num / den
}
