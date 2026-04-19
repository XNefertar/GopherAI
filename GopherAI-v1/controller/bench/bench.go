package bench

import "github.com/gin-gonic/gin"

type PingResponse struct {
	Status string `json:"status"`
	App    string `json:"app"`
}

func Ping(c *gin.Context) {
	c.JSON(200, PingResponse{
		Status: "ok",
		App:    "gopherai-v1",
	})
}

func PingAuth(c *gin.Context) {
	c.JSON(200, PingResponse{
		Status: "ok",
		App:    "gopherai-v1",
	})
}
