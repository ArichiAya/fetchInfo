// Command trader:实时交易触发引擎。
//
// 数据流:
//
//	Kafka(market.bar / market.indicator)实时数据
//	  → internal/strategy 对已注册条件逐条求值
//	  → 命中 → internal/gateway 调掘金交易 SDK 下单
//	  → 委托/成交回报回写 business(PostgreSQL)
//
// 策略条件的注册与管理由 analysis / business 侧完成,trader 只负责消费与求值。
// 独立于 API 运行,保证行情中断或 API 重启时不错过触发时机。
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("[trader] 实时交易触发引擎(骨架,待填充)...")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Println("[trader] shutdown")
}
