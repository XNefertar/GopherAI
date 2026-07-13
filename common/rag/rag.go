package rag

import (
	"GopherAI/common/redis"
	redisPkg "GopherAI/common/redis"
	"GopherAI/config"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	embeddingArk "github.com/cloudwego/eino-ext/components/embedding/ark"
	redisIndexer "github.com/cloudwego/eino-ext/components/indexer/redis"
	redisRetriever "github.com/cloudwego/eino-ext/components/retriever/redis"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	redisCli "github.com/redis/go-redis/v9"
)

type RAGIndexer struct {
	embedding embedding.Embedder
	indexer   *redisIndexer.Indexer
}

type RAGQuery struct {
	embedding embedding.Embedder
	retriever retriever.Retriever
}

const (
	defaultChunkSize    = 600
	defaultChunkOverlap = 120
	defaultTopK         = 5
)

// splitTextIntoChunks 将长文本切成多个带重叠的片段，提升检索粒度。
func splitTextIntoChunks(text string, chunkSize, overlap int) []string {
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 5
	}

	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}

	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize
	}

	chunks := make([]string, 0, (len(runes)+step-1)/step)
	for start := 0; start < len(runes); start += step {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		if end == len(runes) {
			break
		}
	}

	return chunks
}

// 构建知识库索引
// 专业说法：文本解析、文本切块、向量化、存储向量
// 通俗理解：把“人能读的文档”，转换成“AI 能按语义搜索的格式”，并存起来
func NewRAGIndexer(ctx context.Context, kbID, embeddingModel string) (*RAGIndexer, error) {
	// 向量的维度大小（等于向量模型输出的数字个数）
	// Redis 在创建向量索引时必须提前知道这个值
	dimension := config.GetConfig().RagModelConfig.RagDimension

	embedder, err := newRAGEmbedder(ctx, embeddingModel)
	if err != nil {
		return nil, err
	}
	// ===============================
	// 2. 初始化 Redis 中的向量索引结构
	// ===============================
	// 可以理解为：先在 Redis 里建好“仓库”，
	// 告诉它以后要存向量，并且每个向量的维度是多少
	if err := redisPkg.InitRedisIndex(ctx, kbID, dimension); err != nil {
		return nil, fmt.Errorf("failed to init redis index: %w", err)
	}

	// 获取 Redis 客户端，用于后续数据写入
	rdb := redisPkg.Rdb

	// ===============================
	// 3. 配置索引器（定义：文档如何被存进 Redis）
	// ===============================
	indexerConfig := &redisIndexer.IndexerConfig{
		Client:    rdb,                                 // Redis 客户端
		KeyPrefix: redis.GenerateIndexNamePrefix(kbID), // 不同知识库使用不同前缀，避免冲突
		BatchSize: 10,                                  // 批量处理文档，提高写入效率

		// 定义：一段文档（Document）在 Redis 中该如何存储
		DocumentToHashes: func(ctx context.Context, doc *schema.Document) (*redisIndexer.Hashes, error) {
			source := ""
			if s, ok := doc.MetaData["source"].(string); ok {
				source = s
			}
			filename := ""
			if f, ok := doc.MetaData["filename"].(string); ok {
				filename = f
			}
			fileID, _ := doc.MetaData["file_id"].(string)
			return &redisIndexer.Hashes{
				Key: fmt.Sprintf("%s:%s:%s", kbID, fileID, doc.ID),
				Field2Value: map[string]redisIndexer.FieldValue{
					"content":  {Value: doc.Content, EmbedKey: "vector"},
					"filename": {Value: filename},
					"file_id":  {Value: fileID},
					"metadata": {Value: source},
				},
			}, nil
		},
	}

	// 将“向量生成器”交给索引器
	// 这样索引器在写入文本时，可以自动完成向量计算
	indexerConfig.Embedding = embedder

	// ===============================
	// 4. 创建最终可用的索引器实例
	// ===============================
	// 此时索引器已经具备：
	// - 文本 → 向量 的能力
	// - 向量写入 Redis 的能力
	idx, err := redisIndexer.NewIndexer(ctx, indexerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexer: %w", err)
	}

	// 返回一个封装好的 RAGIndexer，
	// 后续只需要调用它，就可以把文档加入知识库
	return &RAGIndexer{
		embedding: embedder,
		indexer:   idx,
	}, nil
}

// IndexFile 读取文件内容并创建向量索引
func (r *RAGIndexer) IndexFile(ctx context.Context, kbID, fileID, filePath string) (int, error) {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	ragConfig := config.GetConfig().RagModelConfig
	chunkSize := ragConfig.RagChunkSize
	chunkOverlap := ragConfig.RagChunkOverlap

	// 将长文本切成多个片段，确保 TopK 检索能从多个候选片段中挑选结果。
	chunks := splitTextIntoChunks(string(content), chunkSize, chunkOverlap)
	if len(chunks) == 0 {
		return 0, fmt.Errorf("no valid content chunks found")
	}

	docs := make([]*schema.Document, 0, len(chunks))
	for i, chunk := range chunks {
		docs = append(docs, &schema.Document{
			ID:      fmt.Sprintf("doc_%d", i),
			Content: chunk,
			MetaData: map[string]any{
				"source":      filePath,
				"filename":    filepath.Base(filePath),
				"file_id":     fileID,
				"kb_id":       kbID,
				"chunk_index": i,
			},
		})
	}

	// 使用 indexer 批量存储多个文档片段（会自动进行向量化）
	_, err = r.indexer.Store(ctx, docs)
	if err != nil {
		return 0, fmt.Errorf("failed to store documents: %w", err)
	}

	return len(chunks), nil
}

// DeleteIndex 删除指定知识库的索引（静态方法，不依赖实例）
func DeleteIndex(ctx context.Context, kbID string) error {
	if err := redisPkg.DeleteRedisIndex(ctx, kbID); err != nil {
		return fmt.Errorf("failed to delete redis index: %w", err)
	}
	// TODO: 可以添加清理同一 prefix 下 Hash Key 的逻辑
	return nil
}

func getRAGEmbeddingAPIKey() string {
	return config.GetConfig().Model.RagEmbeddingAPIKey
}

func getRAGEmbeddingAPIType() *embeddingArk.APIType {
	raw := strings.ToLower(strings.TrimSpace(config.GetConfig().RagModelConfig.RagEmbeddingAPIType))

	switch raw {
	case "multimodal", "multi_modal", "multi-modal":
		t := embeddingArk.APITypeMultiModal
		return &t
	case "text", "text_api":
		t := embeddingArk.APITypeText
		return &t
	default:
		t := embeddingArk.APITypeText
		return &t
	}
}

func newRAGEmbedder(ctx context.Context, modelName string) (embedding.Embedder, error) {
	cfg := config.GetConfig().RagModelConfig
	apiKey := getRAGEmbeddingAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("RAG_EMBEDDING_API_KEY is empty")
	}
	if strings.TrimSpace(cfg.RagEmbeddingBaseURL) == "" {
		return nil, fmt.Errorf("rag embedding base url is empty")
	}
	if strings.TrimSpace(modelName) == "" {
		return nil, fmt.Errorf("rag embedding model is empty")
	}

	embedConfig := &embeddingArk.EmbeddingConfig{
		BaseURL: cfg.RagEmbeddingBaseURL,
		APIKey:  apiKey,
		Model:   modelName,
		APIType: getRAGEmbeddingAPIType(),
	}

	embedder, err := embeddingArk.NewEmbedder(ctx, embedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create rag embedder: %w", err)
	}
	return embedder, err
}

// NewRAGQuery 创建 RAG 查询器（用于向量检索和问答）
func NewRAGQuery(ctx context.Context, kbID string) (*RAGQuery, error) {
	cfg := config.GetConfig()
	topK := cfg.RagModelConfig.RagTopK
	if topK <= 0 {
		topK = defaultTopK
	}

	embedder, err := newRAGEmbedder(ctx, cfg.RagModelConfig.RagEmbeddingModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// 创建 retriever
	rdb := redisPkg.Rdb
	indexName := redis.GenerateIndexName(kbID)

	retrieverConfig := &redisRetriever.RetrieverConfig{
		Client:       rdb,
		Index:        indexName,
		Dialect:      2,
		ReturnFields: []string{"content", "metadata", "distance"},
		TopK:         topK,
		VectorField:  "vector",
		DocumentConverter: func(ctx context.Context, doc redisCli.Document) (*schema.Document, error) {
			resp := &schema.Document{
				ID:       doc.ID,
				Content:  "",
				MetaData: map[string]any{},
			}
			for field, val := range doc.Fields {
				if field == "content" {
					resp.Content = val
				} else {
					resp.MetaData[field] = val
				}
			}
			return resp, nil
		},
	}
	retrieverConfig.Embedding = embedder

	rtr, err := redisRetriever.NewRetriever(ctx, retrieverConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create retriever: %w", err)
	}

	return &RAGQuery{
		embedding: embedder,
		retriever: rtr,
	}, nil
}

// RetrieveDocuments 检索相关文档
func (r *RAGQuery) RetrieveDocuments(ctx context.Context, query string) ([]*schema.Document, error) {
	docs, err := r.retriever.Retrieve(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}
	return docs, nil
}

// BuildRAGPrompt 构建包含检索文档的提示词
func BuildRAGPrompt(query string, docs []*schema.Document) string {
	if len(docs) == 0 {
		return query
	}

	var contextBuilder strings.Builder
	for i, doc := range docs {
		fmt.Fprintf(&contextBuilder, "[文档 %d]: %s\n\n", i+1, doc.Content)
	}

	prompt := fmt.Sprintf(`基于以下参考文档回答用户的问题。如果文档中没有相关信息，请说明无法找到相关信息。

参考文档：
%s

用户问题：%s

请提供准确、完整的回答：`, contextBuilder.String(), query)

	return prompt
}
