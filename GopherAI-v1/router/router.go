package router

import (
	"GopherAI/middleware/jwt"
	"os"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	var r *gin.Engine

	if os.Getenv("BENCH_MODE") == "http-baseline" {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
		r.Use(gin.Recovery())
	} else {
		r = gin.Default()
	}

	enterRouter := r.Group("/api/v1")
	{
		RegisterUserRouter(enterRouter.Group("/user"))
		RegisterBenchRouter(enterRouter.Group("/bench"))
	}
	//后续登录的接口需要jwt鉴权
	{
		AIGroup := enterRouter.Group("/AI")
		AIGroup.Use(jwt.Auth())
		AIRouter(AIGroup)
	}

	{
		ImageGroup := enterRouter.Group("/image")
		ImageGroup.Use(jwt.Auth())
		ImageRouter(ImageGroup)
	}

	return r
}
