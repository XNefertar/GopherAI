package aihelper

import (
	"GopherAI/config"
	"fmt"
	"os"
	"sort"
	"strings"
)

type ModelDescriptor struct {
	Type           string `json:"type"`
	Key            string `json:"key"`
	Label          string `json:"label"`
	Description    string `json:"description,omitempty"`
	RequiresKB     bool   `json:"requiresKB"`
	SupportsStream bool   `json:"supportsStream"`
	Available      bool   `json:"available"`
	DisabledReason string `json:"disabledReason,omitempty"`
	IsDefault      bool   `json:"isDefault"`
	Sort           int    `json:"sort"`
}

func ListModelDescriptors() []ModelDescriptor {
	mainReady, mainReason := validateMainChatConfig()
	ragReady, ragReason := validateRAGConfig(mainReady, mainReason)
	mcpReady, mcpReason := validateMCPConfig(mainReady, mainReason)
	ollamaReady, ollamaReason := validateOllamaConfig()
	autoReady, autoReason := validateAutoRouteConfig(mainReady, mainReason)

	models := []ModelDescriptor{
		{
			Type:           ModelTypeOpenAI,
			Key:            "openai",
			Label:          "基础聊天",
			Description:    "通用对话，适合常规问答场景",
			RequiresKB:     false,
			SupportsStream: true,
			Available:      mainReady,
			DisabledReason: mainReason,
			Sort:           10,
		},
		{
			Type:           ModelTypeRAG,
			Key:            "rag",
			Label:          "知识库问答",
			Description:    "结合知识库检索结果生成回答，新会话必须绑定知识库",
			RequiresKB:     true,
			SupportsStream: true,
			Available:      ragReady,
			DisabledReason: ragReason,
			Sort:           20,
		},
		{
			Type:           ModelTypeMCP,
			Key:            "mcp",
			Label:          "工具调用",
			Description:    "通过 MCP 服务调用外部工具完成任务",
			RequiresKB:     false,
			SupportsStream: true,
			Available:      mcpReady,
			DisabledReason: mcpReason,
			Sort:           30,
		},
		{
			Type:           ModelTypeOllama,
			Key:            "ollama",
			Label:          "本地模型 Ollama",
			Description:    "使用本地 Ollama 模型进行推理",
			RequiresKB:     false,
			SupportsStream: true,
			Available:      ollamaReady,
			DisabledReason: ollamaReason,
			Sort:           40,
		},
		{
			Type:           ModelTypeAuto,
			Key:            "auto",
			Label:          "自动路由",
			Description:    "根据问题类型自动选择合适模型",
			RequiresKB:     false,
			SupportsStream: true,
			Available:      autoReady,
			DisabledReason: autoReason,
			Sort:           50,
		},
	}

	defaultType := resolveDefaultModelType(models)
	for i := range models {
		models[i].IsDefault = models[i].Type == defaultType
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].Sort < models[j].Sort
	})
	return models
}

func GetDefaultModelType() string {
	return resolveDefaultModelType(ListModelDescriptors())
}

func resolveDefaultModelType(models []ModelDescriptor) string {
	for _, model := range models {
		if model.Type == ModelTypeOpenAI && model.Available {
			return model.Type
		}
	}
	for _, model := range models {
		if model.Available {
			return model.Type
		}
	}
	return ModelTypeOpenAI
}

func validateMainChatConfig() (bool, string) {
	missing := make([]string, 0, 3)
	if !hasEnvValue("OPENAI_API_KEY") {
		missing = append(missing, "OPENAI_API_KEY")
	}
	if !hasEnvValue("OPENAI_MODEL_NAME") {
		missing = append(missing, "OPENAI_MODEL_NAME")
	}
	if !hasEnvValue("OPENAI_BASE_URL") {
		missing = append(missing, "OPENAI_BASE_URL")
	}
	if len(missing) > 0 {
		return false, fmt.Sprintf("缺少配置：%s", strings.Join(missing, ", "))
	}
	return true, ""
}

func validateRAGConfig(mainReady bool, mainReason string) (bool, string) {
	if !mainReady {
		return false, mainReason
	}

	cfg := config.GetConfig().RagModelConfig
	missing := make([]string, 0, 3)
	if !hasAnyEnvValue("RAG_EMBEDDING_API_KEY", "RAG_API_KEY") {
		missing = append(missing, "RAG_EMBEDDING_API_KEY")
	}
	if strings.TrimSpace(cfg.RagEmbeddingBaseURL) == "" {
		missing = append(missing, "ragModelConfig.embeddingBaseUrl")
	}
	if strings.TrimSpace(cfg.RagEmbeddingModel) == "" {
		missing = append(missing, "ragModelConfig.embeddingModel")
	}
	if len(missing) > 0 {
		return false, fmt.Sprintf("缺少配置：%s", strings.Join(missing, ", "))
	}
	return true, ""
}

func validateMCPConfig(mainReady bool, mainReason string) (bool, string) {
	if !mainReady {
		return false, mainReason
	}
	return true, ""
}

func validateOllamaConfig() (bool, string) {
	missing := make([]string, 0, 2)
	if !hasEnvValue("OLLAMA_MODEL_NAME") {
		missing = append(missing, "OLLAMA_MODEL_NAME")
	}
	if !hasEnvValue("OLLAMA_BASE_URL") {
		missing = append(missing, "OLLAMA_BASE_URL")
	}
	if len(missing) > 0 {
		return false, fmt.Sprintf("缺少配置：%s", strings.Join(missing, ", "))
	}
	return true, ""
}

func validateAutoRouteConfig(mainReady bool, mainReason string) (bool, string) {
	if !mainReady {
		return false, mainReason
	}
	return true, ""
}

func hasEnvValue(key string) bool {
	return strings.TrimSpace(os.Getenv(key)) != ""
}

func hasAnyEnvValue(keys ...string) bool {
	for _, key := range keys {
		if hasEnvValue(key) {
			return true
		}
	}
	return false
}
