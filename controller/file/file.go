package file

import (
	"GopherAI/common/code"
	"GopherAI/controller"
	"GopherAI/service/file"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type (
	UploadFileResponse struct {
		FilePath string `json:"file_path,omitempty"`
		controller.Response
	}
)

func UploadRagFile(c *gin.Context) {
	kbID := c.Param("kbID")
	res := new(UploadFileResponse)

	if kbID == "" {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	uploadedFile, err := c.FormFile("file")
	if err != nil {
		log.Println("FormFile fail ", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	username := c.GetString("userName")
	if username == "" {
		log.Println("Username not found in context")
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidToken))
		return
	}

	// 用 c.Request.Context() 而不是 c：
	// 1) 客户端断开 / HTTP 超时会自动 cancel，向下游 Redis、DAO 传播取消信号；
	// 2) service 层签名是标准 context.Context，不依赖 gin。
	ctx := c.Request.Context()

	//indexer 会在 service 层根据实际文件名创建
	filePath, err := file.UploadRagFile(ctx, username, kbID, uploadedFile)
	if err != nil {
		log.Println("UploadFile fail ", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.FilePath = filePath
	c.JSON(http.StatusOK, res)
}
