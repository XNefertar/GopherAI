package titlesummary

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// GenerateTitle 使用 GLM-4-Flash 免费 API 从用户的问题总结出标题（5~10 字）
func GenerateTitle(ctx context.Context, userQuestion string) string {
	apiKey := strings.TrimSpace(os.Getenv("TITLE_API_KEY"))
	baseURL := strings.TrimSpace(os.Getenv("TITLE_BASE_URL"))
	model := strings.TrimSpace(os.Getenv("TITLE_MODEL_NAME"))

	if apiKey == "" || baseURL == "" || model == "" {
		log.Println("[titlesummary] env TITLE_API_KEY / TITLE_BASE_URL / TITLE_MODEL_NAME not set, skip")
		return ""
	}

	prompt := fmt.Sprintf(
		`你是一个会话标题生成器。请用 5~10 个字概括用户的第一条问题，直接输出标题，不要多余解释。`+
			`\n用户问题：%s`, userQuestion,
	)

	reqBody := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: "你是一个会话标题生成器，用5~10个字概括用户问题。"},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   20,
		Temperature: 0.3,
	}

	body, _ := json.Marshal(reqBody)

	httpCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodPost,
		strings.TrimRight(baseURL, "/")+"/chat/completions",
		bytes.NewReader(body))
	if err != nil {
		log.Printf("[titlesummary] create request failed: %v", err)
		return ""
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Printf("[titlesummary] request failed: %v", err)
		return ""
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result chatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("[titlesummary] parse response failed: %v", err)
		return ""
	}

	if result.Error != nil {
		log.Printf("[titlesummary] API error: %s", result.Error.Message)
		return ""
	}

	if len(result.Choices) == 0 {
		log.Println("[titlesummary] no choices in response")
		return ""
	}

	title := strings.TrimSpace(result.Choices[0].Message.Content)
	// 清理可能的引号和多余字符
	title = strings.Trim(title, `"'「」『』【】`)
	title = strings.TrimSpace(title)

	log.Printf("[titlesummary] generated title: %q", title)
	return title
}
