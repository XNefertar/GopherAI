package router

import (
	"GopherAI/middleware/jwt"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func InitRouter() *gin.Engine {

	r := gin.Default()

	// /metrics 端点暴露所有注册到默认 Registry 的 Prometheus 指标。
	// 故意放在 /api/v1 之外且不走 JWT 鉴权：
	//   - 便于 Prometheus scrape 直接拉取；
	//   - 生产环境可以通过 Nginx / 服务网格把它限制在内网访问。
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	enterRouter := r.Group("/api/v1")
	{
		RegisterUserRouter(enterRouter.Group("/user"))
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

	{
		FileGroup := enterRouter.Group("/file")
		FileGroup.Use(jwt.Auth())
		FileRouter(FileGroup)
	}

	return r
}
