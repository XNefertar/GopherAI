package image

import (
	"GopherAI/common/code"
	"GopherAI/controller"
	svcImage "GopherAI/service/image"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RecognizeImageResponse struct {
	ClassName string `json:"class_name,omitempty"`
	controller.Response
}

// RecognizeImage 同步识别（兼容旧接口）
func RecognizeImage(c *gin.Context) {
	res := new(RecognizeImageResponse)
	file, err := c.FormFile("image")
	if err != nil {
		log.Println("FormFile fail", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	question := c.PostForm("question")
	result, err := svcImage.RecognizeImageStream(c.Request.Context(), file, question, nil)
	if err != nil {
		log.Println("RecognizeImage fail", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.ClassName = result
	c.JSON(http.StatusOK, res)
}

// RecognizeImageStream SSE 流式识别
func RecognizeImageStream(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		log.Println("FormFile fail", err)
		c.JSON(http.StatusOK, (&controller.Response{}).CodeOf(code.CodeInvalidParams))
		return
	}

	question := c.PostForm("question")

	// SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	_, err = svcImage.RecognizeImageStream(c.Request.Context(), file, question, func(msg string) {
		c.SSEvent("message", msg)
		c.Writer.Flush()
	})

	if err != nil {
		log.Println("stream fail:", err)
		c.SSEvent("error", err.Error())
		c.Writer.Flush()
		return
	}

	c.SSEvent("done", "[DONE]")
	c.Writer.Flush()
}
