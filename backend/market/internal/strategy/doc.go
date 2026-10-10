// Package strategy 定义交易触发条件与求值逻辑。
//
// 条件模型(示例):
//
//	type Condition struct {
//	    ID     int64
//	    Symbol string
//	    Rule   string   // 如 "ma5 上穿 ma20"
//	    Action string   // 如 "buy 100 @ market"
//	}
//
// 条件配置持久化在 business(PostgreSQL),trader 周期性拉取/订阅更新;
// analysis 的"手动分析请求触发交易"也通过本包暴露的求值入口执行,
// 保证手动触发与实时触发的求值逻辑唯一、不重复实现。
package strategy
