package bench

import "github.com/gin-gonic/gin"

// PingResponse 是 bench 接口的统一返回结构。
type PingResponse struct {
	Status string `json:"status"`
	App    string `json:"app"`
}

// Ping 用于做最简单的 HTTP 基线压测，无需鉴权。
func Ping(c *gin.Context) {
	c.JSON(200, PingResponse{
		Status: "ok",
		App:    "gopherai-v2",
	})
}

// PingAuth 用于在挂载 JWT 中间件后做带鉴权的 HTTP 基线压测。
func PingAuth(c *gin.Context) {
	c.JSON(200, PingResponse{
		Status: "ok",
		App:    "gopherai-v2",
	})
}
