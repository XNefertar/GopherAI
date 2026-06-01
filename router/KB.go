package router

import (
	"GopherAI/controller/file"

	"github.com/gin-gonic/gin"
)

func KBRouter(r *gin.RouterGroup) {
	r.POST("/:kbID/files", file.UploadRagFile)
}
