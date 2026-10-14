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
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ArichiAya/fetch-info/business/internal/httpapi/v1"
	"github.com/ArichiAya/fetch-info/common/auth"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

const (
	databaseURL string = "postgres://iiot_admin:changeme_please_2024@192.168.80.131:5432/mktdata"
	port        string = "8556"
)

func main() {
	// 设置日志输出到文件和标准输出
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// 组合文件和标准输出
	multiWriter := io.MultiWriter(os.Stdout, file)
	log.SetOutput(multiWriter)

	if err := godotenv.Load(".env"); err != nil {
		log.Println("load .env error:", err)
	}

	jwtKey := os.Getenv("jwt_key")
	if jwtKey == "" {
		log.Fatal("JWT key is not set")
	}

	gin.DefaultWriter = multiWriter
	gin.DefaultErrorWriter = multiWriter

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	router := gin.Default()
	jwtAuthMiddleware, err := auth.NewAuthMiddleware(pool, jwtKey)
	if err != nil {
		log.Fatal("Failed to create auth middleware:", err)
	}

	router.POST("/login", jwtAuthMiddleware.LoginHandler)
	router.POST("/register", httpapi.RegisterHandler(pool))
	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Resource not found"}})
	})

	authGroup := router.Group("/api", jwtAuthMiddleware.MiddlewareFunc())
	authGroup.GET("/refresh_token", jwtAuthMiddleware.RefreshHandler)
	authGroup.GET("/business/subscriptions", httpapi.GetSubscriptionsHandler(pool))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	log.Printf("Server is running on port %s\n", port)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	<-shutdown
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
