// Package gateway 适配掘金交易能力:下单、撤单、查持仓、接收委托/成交回报。
//
// 掘金官方 SDK 以 Python 为主,Go 侧无官方客户端时采用桥接方案:
// 本包只负责下发指令与接收回报的队列语义,具体下单由 Python worker 执行:
//
//	trader --(指令队列:Redis list / 消息)--> Python worker --> 掘金交易SDK
//	Python worker --(回报:Redis pub/sub / 消息)--> trader --> business(PostgreSQL)
//
// 接口形态(下单/撤单/状态查询)在接掘金实测后定型。
package gateway
