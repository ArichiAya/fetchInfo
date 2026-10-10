// Command server:业务 API 服务。
//
// 数据存储:
//   - PostgreSQL:工单、人员、策略条件配置等业务数据
//   - Redis:登录态缓存(session/黑名单,带 TTL)、热点数据
//
// 待实现清单:
//  1. internal/auth:登录签发 JWT,Redis 缓存登录态
//  2. internal/workorder / personnel:PostgreSQL CRUD
//  3. internal/httpapi:路由与中间件
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("[business] 业务 API 服务(骨架,待填充)...")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Println("[business] shutdown")
}
