package file

import (
	"GopherAI/common/rag"
	"GopherAI/config"
	"GopherAI/utils"
	"context"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
)

// 上传rag相关文件（这里只允许文本文件）
// 其实可以直接将其向量化进行保存，但这边依旧存储到服务器上以便后续可以在服务器上查看历史RAG文件
func UploadRagFile(ctx context.Context, username, kbID string, uploadedFile *multipart.FileHeader) (string, error) {
	// 校验文件类型和文件名
	if err := utils.ValidateFile(uploadedFile); err != nil {
		log.Printf("File validation failed: %v", err)
		return "", err
	}

	// 创建用户目录
	userDir := filepath.Join("uploads", username)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		log.Printf("Failed to create user directory %s: %v", userDir, err)
		return "", err
	}

	// 生成UUID作为唯一文件名
	fileID := utils.GenerateUUID()

	ext := filepath.Ext(uploadedFile.Filename)
	filename := fileID + ext
	filePath := filepath.Join(userDir, filename)

	// 打开上传的文件
	src, err := uploadedFile.Open()
	if err != nil {
		log.Printf("Failed to open uploaded uploadedFile: %v", err)
		return "", err
	}
	defer src.Close()

	// 创建目标文件
	dst, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create destination uploadedFile %s: %v", filePath, err)
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		log.Printf("Failed to copy uploadedFile content: %v", err)
		return "", err
	}

	log.Printf("File uploaded successfully: %s", filePath)

	// 创建 RAG 索引器并对文件进行向量化
	indexer, err := rag.NewRAGIndexer(ctx, kbID, config.GetConfig().RagModelConfig.RagEmbeddingModel)
	if err != nil {
		log.Printf("Failed to create RAG indexer: %v", err)
		// 删除已上传的文件
		os.Remove(filePath)
		return "", err
	}

	// 读取文件内容并创建向量索引
	if err := indexer.IndexFile(ctx, kbID, fileID, filePath); err != nil {
		log.Printf("Failed to index uploadedFile: %v", err)
		// 删除已上传的文件和索引
		os.Remove(filePath)
		rag.DeleteIndex(ctx, kbID)
		return "", err
	}

	log.Printf("File indexed successfully: %s", filename)
	return filePath, nil
}
