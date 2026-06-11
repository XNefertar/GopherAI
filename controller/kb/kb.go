package kb

import (
	"GopherAI/common/code"
<<<<<<< Updated upstream
	"GopherAI/controller"
	"GopherAI/model"
	serviceKb "GopherAI/service/kb"
	"log"
=======
	"GopherAI/common/logger"
	"GopherAI/controller"
	"GopherAI/model"
	serviceKb "GopherAI/service/kb"
>>>>>>> Stashed changes
	"net/http"

	"github.com/gin-gonic/gin"
)

type (
	CreateKBResponse struct {
		controller.Response
		KB *model.KnowledgeBase `json:"kb,omitempty"`
	}

	ListKBResponse struct {
		controller.Response
		KBs []model.KnowledgeBase `json:"kbs,omitempty"`
	}

	ListFilesResponse struct {
		controller.Response
		Files []model.KBFile `json:"files,omitempty"`
	}
)

func CreateKB(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	res := new(CreateKBResponse)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	owner := c.GetString("userName")
	kb, err := serviceKb.CreateKB(c.Request.Context(), owner, req.Name, req.Description)
	if err != nil {
<<<<<<< Updated upstream
		log.Printf("CreateKB failed: %v", err)
=======
		logger.With("userName", owner).Error("CreateKB failed", "error", err)
>>>>>>> Stashed changes
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.KB = kb
	c.JSON(http.StatusOK, res)
}

func ListKB(c *gin.Context) {
	owner := c.GetString("userName")
	res := new(ListKBResponse)
	kbs, err := serviceKb.ListKB(c.Request.Context(), owner)
	if err != nil {
<<<<<<< Updated upstream
		log.Printf("ListKB failed: %v", err)
=======
		logger.With("userName", owner).Error("ListKB failed", "error", err)
>>>>>>> Stashed changes
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.KBs = kbs
	c.JSON(http.StatusOK, res)
}

func AddFileToKB(c *gin.Context) {
	kbID := c.Param("kbID")
<<<<<<< Updated upstream
	res := new(controller.Response)

	if kbID == "" {
=======
	owner := c.GetString("userName")
	l := logger.With("userName", owner, "kbID", kbID)
	res := new(controller.Response)

	if kbID == "" {
		l.Error("kbID is empty")
>>>>>>> Stashed changes
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	uploadedFile, err := c.FormFile("file")
	if err != nil {
<<<<<<< Updated upstream
		log.Printf("FormFile failed: %v", err)
=======
		l.Error("FormFile failed", "error", err)
>>>>>>> Stashed changes
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

<<<<<<< Updated upstream
	owner := c.GetString("userName")
	ctx := c.Request.Context()

	if err := serviceKb.AddFileToKB(ctx, owner, kbID, uploadedFile); err != nil {
		log.Printf("AddFileToKB failed: %v", err)
=======
	ctx := c.Request.Context()

	if err := serviceKb.AddFileToKB(ctx, owner, kbID, uploadedFile); err != nil {
		l.Error("AddFileToKB failed", "error", err)
>>>>>>> Stashed changes
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	c.JSON(http.StatusOK, res)
}

func ListFiles(c *gin.Context) {
	kbID := c.Param("kbID")
	res := new(ListFilesResponse)

	files, err := serviceKb.ListFiles(c.Request.Context(), kbID)
	if err != nil {
<<<<<<< Updated upstream
		log.Printf("ListFiles failed: %v", err)
=======
		logger.With("kbID", kbID).Error("ListFiles failed", "error", err)
>>>>>>> Stashed changes
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.Files = files
	c.JSON(http.StatusOK, res)
}

func RemoveFile(c *gin.Context) {
	kbID := c.Param("kbID")
	fileID := c.Param("fileID")
	owner := c.GetString("userName")
<<<<<<< Updated upstream
	res := new(controller.Response)

	if err := serviceKb.RemoveFileFromKB(c.Request.Context(), owner, kbID, fileID); err != nil {
		log.Printf("RemoveFile failed: %v", err)
=======
	l := logger.With("userName", owner, "kbID", kbID, "fileID", fileID)
	res := new(controller.Response)

	if err := serviceKb.RemoveFileFromKB(c.Request.Context(), owner, kbID, fileID); err != nil {
		l.Error("RemoveFile failed", "error", err)
>>>>>>> Stashed changes
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	c.JSON(http.StatusOK, res)
}
