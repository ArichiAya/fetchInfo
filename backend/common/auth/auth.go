package auth

import (
	"context"
	"log"
	"net/http"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

var (
	// identityKey 是 gin-jwt 的 IdentityKey,也是 claims 里存角色的键
	identityKey = "role"
	// userIdKey 是 claims 里存用户主键的键
	userIdKey = "id"
)

// User demo
type User struct {
	UserId   int
	UserName string
	role     string
}

// func main() {
// 	engine := gin.Default()
// 	// the jwt middleware
// 	authMiddleware, err := jwt.New(initParams())
// 	if err != nil {
// 		log.Fatal("JWT Error:" + err.Error())
// 	}

// 	// register middleware
// 	engine.Use(handlerMiddleware(authMiddleware))

// 	// register route
// 	registerRoute(engine, authMiddleware)

// 	// start http server
// 	if err = http.ListenAndServe(":"+port, engine); err != nil {
// 		log.Fatal(err)
// 	}
// }

func NewAuthMiddleware(pool *pgxpool.Pool, jwtKey string) (*jwt.GinJWTMiddleware, error) {
	return jwt.New(initParams(pool, jwtKey))
}

// func registerRoute(r *gin.Engine, handle *jwt.GinJWTMiddleware) {
// 	r.POST("/login", handle.LoginHandler)
// 	r.NoRoute(handle.MiddlewareFunc(), handleNoRoute())

// 	auth := r.Group("/auth", handle.MiddlewareFunc())
// 	auth.GET("/refresh_token", handle.RefreshHandler)
// 	auth.GET("/hello", helloHandler)
// }

func HandlerMiddleware(authMiddleware *jwt.GinJWTMiddleware) gin.HandlerFunc {
	return func(context *gin.Context) {
		errInit := authMiddleware.MiddlewareInit()
		if errInit != nil {
			log.Fatal("authMiddleware.MiddlewareInit() Error:" + errInit.Error())
		}
	}
}

func initParams(pool *pgxpool.Pool, jwtKey string) *jwt.GinJWTMiddleware {
	return &jwt.GinJWTMiddleware{
		Realm:       "test zone",
		Key:         []byte(jwtKey),
		Timeout:     time.Hour,
		MaxRefresh:  time.Hour,
		IdentityKey: identityKey,
		PayloadFunc: payloadFunc(),

		IdentityHandler: identityHandler(),
		Authenticator:   authenticator(pool),
		Authorizator:    authorizator(),
		Unauthorized:    unauthorized(),
		TokenLookup:     "header: Authorization",
		// TokenLookup: "query:token",
		// TokenLookup: "cookie:token",
		TokenHeadName: "Bearer",
		TimeFunc:      time.Now,
	}
}

func payloadFunc() func(data interface{}) jwt.MapClaims {
	return func(data interface{}) jwt.MapClaims {
		if v, ok := data.(*User); ok {
			return jwt.MapClaims{
				identityKey: v.role,
				userIdKey:   v.UserId,
			}
		}
		return jwt.MapClaims{}
	}
}

func identityHandler() func(c *gin.Context) interface{} {
	return func(c *gin.Context) interface{} {
		claims := jwt.ExtractClaims(c)
		user := &User{}
		if role, ok := claims[identityKey].(string); ok {
			user.role = role
		}
		// JWT 经 JSON 往返后数字是 float64,直接断言 int 会 panic,
		// 而 gin 的 Recovery 会把它变成 500 空 body —— 每个请求都中。
		if id, ok := claims[userIdKey].(float64); ok {
			user.UserId = int(id)
		}
		return user
	}
}

func authenticator(pool *pgxpool.Pool) func(c *gin.Context) (interface{}, error) {
	return func(c *gin.Context) (interface{}, error) {
		var loginVals login
		if err := c.ShouldBind(&loginVals); err != nil {
			return "", jwt.ErrMissingLoginValues
		}
		username := loginVals.Username
		password := loginVals.Password

		// 连接pgsql数据库，取出用户信息进行验证
		queryStr := "SELECT id, username, password, role FROM users WHERE username=$1"
		row := pool.QueryRow(context.Background(), queryStr, username)
		var id int
		var hashedPassword string
		var role string
		err := row.Scan(&id, &username, &hashedPassword, &role)
		if err != nil {
			log.Printf("Failed to query user: %v", err)
			return nil, jwt.ErrFailedAuthentication
		}

		// 从password取出hash salt,然后用hash salt对password进行hash，和数据库中的hash进行比对
		err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
		if err != nil {
			log.Printf("Failed to compare password: %v", err)
			return nil, jwt.ErrFailedAuthentication
		}

		return &User{
			UserId:   id,
			UserName: username,
			role:     role,
		}, nil

		// if (userID == "admin" && password == "admin") || (userID == "test" && password == "test") {
		// 	return &User{
		// 		UserName:  userID,
		// 		LastName:  "Bo-Yi",
		// 		FirstName: "Wu",
		// 	}, nil
		// }
		// return nil, jwt.ErrFailedAuthentication
	}
}

func authorizator() func(data interface{}, c *gin.Context) bool {
	return func(data interface{}, c *gin.Context) bool {
		if _, ok := data.(*User); !ok {
			return false
		}
		// 查redis看用户是否登出
		return true
	}
}

func unauthorized() func(c *gin.Context, code int, message string) {
	return func(c *gin.Context, code int, message string) {
		c.JSON(code, gin.H{
			"code":    code,
			"message": message,
		})
	}
}

// func handleNoRoute() func(c *gin.Context) {
// 	return func(c *gin.Context) {
// 		claims := jwt.ExtractClaims(c)
// 		log.Printf("NoRoute claims: %#v\n", claims)
// 		c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
// 	}
// }

// func helloHandler(c *gin.Context) {
// 	claims := jwt.ExtractClaims(c)
// 	user, _ := c.Get(identityKey)
// 	c.JSON(200, gin.H{
// 		"userID":   claims[identityKey],
// 		"userName": user.(*User).UserName,
// 		"text":     "Hello World.",
// 	})
// }

// 自定义角色校验中间件
func RoleAuth(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// gin-jwt 解析成功后，会将 claims 或 user 存入上下文
		claims := jwt.ExtractClaims(c)

		role, ok := claims[identityKey].(string)
		if !ok || role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "permission denied",
				},
			})
			c.Abort() // 终止后续请求
			return
		}

		c.Next()
	}
}
