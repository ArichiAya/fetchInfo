// Package httpapi 组装业务服务的 HTTP 层:路由、handler、鉴权中间件。
package httpapi

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// fail 统一错误体,与 docs/api.md 的约定一致:
//
//	{ "error": { "code": "...", "message": "..." } }
func fail(c *gin.Context, status int, code, message string) {
	log.Print(status, gin.H{"error": gin.H{"code": code, "message": message}})
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}

// GET /api/business/subscriptions
// [{ "code": "000001", "name": "平安银行", "subscribedAt": "2026-08-01T09:15:00+08:00" }]
func GetSubscriptionsHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 查询subsribe表,返回订阅列表
		queryStr := `SELECT symbol, starttime FROM subscribe`
		rows, err := pool.Query(c.Request.Context(), queryStr)
		if err != nil {
			fail(c, 500, "INTERNAL", "Failed to query database")
			return
		}
		defer rows.Close()

		var subscriptions []map[string]interface{}
		for rows.Next() {
			var symbol string
			var starttime time.Time
			if err := rows.Scan(&symbol, &starttime); err != nil {
				fail(c, 500, "INTERNAL", "Failed to scan row")
				return
			}
			subscriptions = append(subscriptions, map[string]interface{}{
				"code":         symbol,
				"name":         symbol,
				"subscribedAt": starttime,
			})
		}

		c.JSON(200, subscriptions)
	}
}

// POST /register
// { "username": "alice", "password": "secret123" }
// 201 { "code": 201 }
func RegisterHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, 400, "INVALID_PARAM", "Invalid request body")
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			fail(c, 500, "INTERNAL", "Failed to hash password")
			return
		}

		queryStr := `INSERT INTO users (username, password, role) VALUES ($1, $2, $3)`
		_, err = pool.Exec(c.Request.Context(), queryStr, req.Username, string(hashedPassword), "user")
		if err != nil {
			fail(c, 500, "INTERNAL", "Failed to register user")
			return
		}

		// 不返回 id/username:前端注册后只提示成功再切回登录,拿不到也用不上
		c.JSON(201, gin.H{"code": 201})
	}
}
