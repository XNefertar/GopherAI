package router

import (
	"GopherAI/controller/kb"

	"github.com/gin-gonic/gin"
)

func KBRouter(r *gin.RouterGroup) {
	r.POST("", kb.CreateKB)
	r.GET("", kb.ListKB)
	r.POST("/:kbID/files", kb.AddFileToKB)
	r.GET("/:kbID/files", kb.ListFiles)
	r.DELETE("/:kbID/files/:fileID", kb.RemoveFile)
}
