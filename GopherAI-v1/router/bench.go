package router

import (
	"GopherAI/controller/bench"
	"GopherAI/middleware/jwt"

	"github.com/gin-gonic/gin"
)

func RegisterBenchRouter(r *gin.RouterGroup) {
	r.GET("/ping", bench.Ping)

	auth := r.Group("")
	auth.Use(jwt.Auth())
	auth.GET("/ping-auth", bench.PingAuth)
}
