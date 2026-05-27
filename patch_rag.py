import re

with open("common/rag/rag.go", "r") as f:
    content = f.read()

# Replace NewRAGIndexer signature
content = content.replace("func NewRAGIndexer(filename, embeddingModel string) (*RAGIndexer, error) {", 
                          "func NewRAGIndexer(kbID, embeddingModel string) (*RAGIndexer, error) {")
content = content.replace("redisPkg.InitRedisIndex(ctx, filename, dimension)", "redisPkg.InitRedisIndex(ctx, kbID, dimension)")
content = content.replace("redis.GenerateIndexNamePrefix(filename)", "redis.GenerateIndexNamePrefix(kbID)")

# Replace DocumentToHashes inside indexerConfig
doc_to_hashes_old = """		DocumentToHashes: func(ctx context.Context, doc *schema.Document) (*redisIndexer.Hashes, error) {

			// 从文档的元数据中取出来源信息（例如文件名、URL）
			source := ""
			if s, ok := doc.MetaData["source"].(string); ok {
				source = s
			}

			// 构造 Redis 中实际存储的数据结构（Hash）
			return &redisIndexer.Hashes{
				// Redis Key，一般由“知识库名 + 文档块 ID”组成
				Key: fmt.Sprintf("%s:%s", filename, doc.ID),

				// Redis Hash 中的字段
				Field2Value: map[string]redisIndexer.FieldValue{
					// content：原始文本内容
					// EmbedKey 表示：该字段需要先做向量化，
					// 生成的向量会存入名为 "vector" 的字段中
					"content": {Value: doc.Content, EmbedKey: "vector"},
					// metadata：一些辅助信息，不参与向量计算
					"metadata": {Value: source},
				},
			}, nil
		},"""

doc_to_hashes_new = """		DocumentToHashes: func(ctx context.Context, doc *schema.Document) (*redisIndexer.Hashes, error) {

			source := ""
			if s, ok := doc.MetaData["source"].(string); ok {
				source = s
			}
			filename := ""
			if f, ok := doc.MetaData["filename"].(string); ok {
				filename = f
			}

			return &redisIndexer.Hashes{
				Key: fmt.Sprintf("%s:%s", kbID, doc.ID),
				Field2Value: map[string]redisIndexer.FieldValue{
					"content": {Value: doc.Content, EmbedKey: "vector"},
					"filename": {Value: filename},
					"metadata": {Value: source},
				},
			}, nil
		},"""

# Regex replacement for doc_to_hashes ignoring whitespace exactness
pattern = re.compile(r"DocumentToHashes:.+?},\s*nil\s*},", re.DOTALL)
content = pattern.sub(doc_to_hashes_new, content)

with open("common/rag/rag.go", "w") as f:
    f.write(content)
