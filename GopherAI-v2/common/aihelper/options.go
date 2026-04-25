package aihelper

import (
	"fmt"
	"os"
)

const (
	ModelTypeOpenAI = "1"
	ModelTypeRAG    = "2"
	ModelTypeMCP    = "3"
	ModelTypeOllama = "4"
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
func BuildSessionCreateOptions(modelType, userName string) (CreateOptions, error) {
	switch modelType {
	case ModelTypeOpenAI:
		return OpenAIOptions{}, nil
	case ModelTypeRAG:
		if userName == "" {
			return nil, fmt.Errorf("RAG model requires userName")
		}
		return RAGOptions{Username: userName}, nil
	case ModelTypeMCP:
		if userName == "" {
			return nil, fmt.Errorf("MCP model requires userName")
		}
		return MCPOptions{Username: userName}, nil
	case ModelTypeOllama:
		modelName := os.Getenv("OLLAMA_MODEL_NAME")
		if modelName == "" {
			return nil, fmt.Errorf("Ollama model requires OLLAMA_MODEL_NAME")
		}
		baseURL := os.Getenv("OLLAMA_BASE_URL")
		return OllamaOptions{
			BaseURL:   baseURL,
			ModelName: modelName,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}
}
