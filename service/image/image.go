package image

import (
	"GopherAI/common/aihelper"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/schema"
)

const defaultVisionPrompt = "请详细描述这张图片的内容，包括其中的物体、场景、人物、文字、颜色等所有细节。"

// RecognizeImage 同步识别（兼容旧接口），返回完整 AI 描述。
func RecognizeImage(file *multipart.FileHeader) (string, error) {
	return RecognizeImageStream(context.Background(), file, defaultVisionPrompt, nil)
}

// RecognizeImageStream 流式识别图片，支持自定义问题。
// cb 为 nil 时不推送流，直接返回聚合结果。
func RecognizeImageStream(ctx context.Context, file *multipart.FileHeader, question string, cb aihelper.StreamCallback) (string, error) {
	// 1. 读取文件
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer src.Close()

	buf, err := io.ReadAll(src)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	// 2. 检测 MIME
	mimeType := http.DetectContentType(buf)
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = "image/jpeg"
	}

	// 3. Base64 编码为 data URL
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(buf))

	// 4. 兜底问题
	if strings.TrimSpace(question) == "" {
		question = defaultVisionPrompt
	}

	// 5. 构建多模态消息
	msg := &schema.Message{
		Role: schema.User,
		MultiContent: []schema.ChatMessagePart{
			{
				Type: schema.ChatMessagePartTypeImageURL,
				ImageURL: &schema.ChatMessageImageURL{URL: dataURL},
			},
			{
				Type: schema.ChatMessagePartTypeText,
				Text: question,
			},
		},
	}

	// 6. 创建视觉模型并调用
	visionLLM, err := aihelper.NewVisionLLM(ctx)
	if err != nil {
		log.Println("create vision llm failed:", err)
		return "", fmt.Errorf("create vision llm: %w", err)
	}

	return aihelper.StreamFromLLM(ctx, visionLLM, []*schema.Message{msg}, cb)
}
