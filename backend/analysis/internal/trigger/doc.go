// Package trigger 处理"分析请求 → 交易触发(满足条件时)":
//   - 手动触发:用户点分析后立即对当前快照求值,命中即下单
//   - 条件注册:把策略条件登记到 trader(经消息队列或 business 的 DB 表),
//     由 trader 在实时数据上持续求值
//
// 求值逻辑与 market/internal/strategy 保持一致(抽到 common 或经消息协议约定),
// 避免手动/实时两套规则产生偏差。
package trigger
