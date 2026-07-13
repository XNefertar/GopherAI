package aihelper

import (
	"fmt"

	appconfig "GopherAI/config"
)

const (
	ModelTypeOpenAI = "1"
	ModelTypeRAG    = "2"
	ModelTypeMCP    = "3"
	ModelTypeOllama = "4"
	// ModelTypeAuto 表示由混合路由器根据请求特征自动选择具体模型，
	// 主要用于成本优化场景：简单问题走低成本模型，复杂/知识/工具型问题走对应模型。
	ModelTypeAuto = "auto"
)

type CreateOptions interface {
	ModelType() string
}

type OpenAIOptions struct{}

func (OpenAIOptions) ModelType() string {
	return ModelTypeOpenAI
}

type RAGOptions struct {
	Username string
	kbID     string
}

func (RAGOptions) ModelType() string {
	return ModelTypeRAG
}

type MCPOptions struct {
	Username string
}

func (MCPOptions) ModelType() string {
	return ModelTypeMCP
}

type OllamaOptions struct {
	BaseURL   string
	ModelName string
}

func (OllamaOptions) ModelType() string {
	return ModelTypeOllama
}

// BuildSessionCreateOptions maps request modelType to typed options used by the factory.
func BuildSessionCreateOptions(modelType, userName, kbID string) (CreateOptions, error) {
	switch modelType {
	case ModelTypeOpenAI:
		return OpenAIOptions{}, nil
	case ModelTypeRAG:
		if userName == "" {
			return nil, fmt.Errorf("RAG model requires userName")
		}
		if kbID == "" {
			return nil, fmt.Errorf("RAG model requires kbID")
		}
		return RAGOptions{
			Username: userName,
			kbID:     kbID,
		}, nil
	case ModelTypeMCP:
		if userName == "" {
			return nil, fmt.Errorf("MCP model requires userName")
		}
		return MCPOptions{Username: userName}, nil
	case ModelTypeOllama:
		mc := appconfig.GetConfig().Model
		if mc.OllamaModelName == "" {
			return nil, fmt.Errorf("Ollama model requires OLLAMA_MODEL_NAME")
		}
		return OllamaOptions{
			BaseURL:   mc.OllamaBaseURL,
			ModelName: mc.OllamaModelName,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}
}
