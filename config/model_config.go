package config

import (
	"fmt"
	"os"
	"strings"
)

// ModelConfig 聚合所有 AI/模型提供方的配置，统一从环境变量读取。
//
// 设计动机：
//   - 此前 OPENAI/VISION/TITLE/OLLAMA/RAG 等密钥散落在 model.go、summary.go、rag.go、options.go
//     等多个文件里各自 os.Getenv，既无法集中校验，也容易在部署时漏配。
//   - 现在所有模型配置在进程启动时一次性收口到本结构体，并在 InitConfig 阶段做 fail-fast 校验，
//     任何缺失都会在启动期直接暴露，而不是运行时偶发报错。
type ModelConfig struct {
	// 主聊天模型（OpenAI 兼容接口）
	OpenAIKey     string
	OpenAIModel   string
	OpenAIBaseURL string

	// 视觉多模态模型（可选，未配置则回退到主模型）
	VisionKey     string
	VisionModel   string
	VisionBaseURL string

	// 会话标题生成（GLM 等，可选）
	TitleKey     string
	TitleModel   string
	TitleBaseURL string

	// 本地 Ollama 模型（可选）
	OllamaModelName string
	OllamaBaseURL   string

	// RAG 向量化密钥（可选；可用 RAG_API_KEY 作为别名）
	RagEmbeddingAPIKey string
}

// loadModelConfig 从环境变量一次性收口所有模型配置。
func loadModelConfig() ModelConfig {
	return ModelConfig{
		OpenAIKey:          env("OPENAI_API_KEY"),
		OpenAIModel:        env("OPENAI_MODEL_NAME"),
		OpenAIBaseURL:      env("OPENAI_BASE_URL"),
		VisionKey:          env("VISION_API_KEY"),
		VisionModel:        env("VISION_MODEL_NAME"),
		VisionBaseURL:      env("VISION_BASE_URL"),
		TitleKey:           env("TITLE_API_KEY"),
		TitleModel:         env("TITLE_MODEL_NAME"),
		TitleBaseURL:       env("TITLE_BASE_URL"),
		OllamaModelName:    env("OLLAMA_MODEL_NAME"),
		OllamaBaseURL:      env("OLLAMA_BASE_URL"),
		RagEmbeddingAPIKey: firstEnv("RAG_EMBEDDING_API_KEY", "RAG_API_KEY"),
	}
}

func env(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

// firstEnv 按优先级返回第一个非空的环境变量值，用于处理别名（如 RAG_API_KEY）。
func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := env(k); v != "" {
			return v
		}
	}
	return ""
}

// Validate 对模型配置做 fail-fast 校验：核心主模型密钥缺失直接返回错误，
// 由 InitConfig 在启动期暴露，避免“能启动但运行时才崩”。
func (m ModelConfig) Validate() error {
	var missing []string
	if m.OpenAIKey == "" {
		missing = append(missing, "OPENAI_API_KEY")
	}
	if m.OpenAIModel == "" {
		missing = append(missing, "OPENAI_MODEL_NAME")
	}
	if m.OpenAIBaseURL == "" {
		missing = append(missing, "OPENAI_BASE_URL")
	}
	if len(missing) > 0 {
		return fmt.Errorf("model config missing required env: %s", strings.Join(missing, ", "))
	}
	return nil
}
