package kb

import (
	"GopherAI/common/code"
	"GopherAI/controller"
	serviceKb "GopherAI/service/kb"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateKB(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	res := new(controller.Response)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	owner := c.GetString("userName")
	kb, err := serviceKb.CreateKB(c.Request.Context(), owner, req.Name, req.Description)
	if err != nil {
		log.Printf("CreateKB failed: %v", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.Data = kb
	c.JSON(http.StatusOK, res)
}

func ListKB(c *gin.Context) {
	owner := c.GetString("userName")
	res := new(controller.Response)
	kbs, err := serviceKb.ListKB(c.Request.Context(), owner)
	if err != nil {
		log.Printf("ListKB failed: %v", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.Data = kbs
	c.JSON(http.StatusOK, res)
}

func AddFileToKB(c *gin.Context) {
	kbID := c.Param("kbID")
	res := new(controller.Response)

	if kbID == "" {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	uploadedFile, err := c.FormFile("file")
	if err != nil {
		log.Printf("FormFile failed: %v", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	owner := c.GetString("userName")
	ctx := c.Request.Context()

	if err := serviceKb.AddFileToKB(ctx, owner, kbID, uploadedFile); err != nil {
		log.Printf("AddFileToKB failed: %v", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	c.JSON(http.StatusOK, res)
}

func ListFiles(c *gin.Context) {
	kbID := c.Param("kbID")
	res := new(controller.Response)

	files, err := serviceKb.ListFiles(c.Request.Context(), kbID)
	if err != nil {
		log.Printf("ListFiles failed: %v", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.Data = files
	c.JSON(http.StatusOK, res)
}

func RemoveFile(c *gin.Context) {
	kbID := c.Param("kbID")
	fileID := c.Param("fileID")
	owner := c.GetString("userName")
	res := new(controller.Response)

	if err := serviceKb.RemoveFileFromKB(c.Request.Context(), owner, kbID, fileID); err != nil {
		log.Printf("RemoveFile failed: %v", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	c.JSON(http.StatusOK, res)
}
