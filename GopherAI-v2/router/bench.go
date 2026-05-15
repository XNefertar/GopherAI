package router

import (
	"GopherAI/controller/bench"
	"GopherAI/middleware/jwt"

	"github.com/gin-gonic/gin"
)

// RegisterBenchRouter 注册压测/基线相关接口。
//   - /ping      ：无鉴权基线
//   - /ping-auth ：带 JWT 鉴权基线
func RegisterBenchRouter(r *gin.RouterGroup) {
	r.GET("/ping", bench.Ping)

	auth := r.Group("")
	auth.Use(jwt.Auth())
	auth.GET("/ping-auth", bench.PingAuth)
}
