// Package httpapi 组装业务服务的 HTTP 层:路由、handler、鉴权中间件。
//
// 路由规划(示例):
//
//	POST /api/business/login            登录(签发 JWT,Redis 记 session)
//	POST /api/business/logout
//	GET/POST /api/business/workorders    工单 CRUD
//	GET/POST /api/business/personnel     人员 CRUD
package httpapi
