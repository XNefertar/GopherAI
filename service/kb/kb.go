package kb

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"GopherAI/common/rag"
	"GopherAI/config"
	daoKb "GopherAI/dao/kb"
	"GopherAI/model"
	"GopherAI/utils"

	"github.com/google/uuid"
)

func CreateKB(ctx context.Context, owner, name, desc string) (*model.KnowledgeBase, error) {
	kbObj := &model.KnowledgeBase{
		ID:          uuid.New().String(),
		UserName:    owner,
		Name:        name,
		Description: desc,
	}
	err := daoKb.CreateKB(ctx, kbObj)
	return kbObj, err
}

func EnsureDefaultKB(ctx context.Context, owner string) (*model.KnowledgeBase, error) {
	kbObj, err := daoKb.GetDefaultKBByOwner(ctx, owner)
	if err == nil && kbObj != nil {
		return kbObj, nil
	}
	return CreateKB(ctx, owner, daoKb.DefaultKBName, "Default Knowledge Base")
}

func ListKB(ctx context.Context, owner string) ([]model.KnowledgeBase, error) {
	return daoKb.ListKBByOwner(ctx, owner)
}

func DeleteKB(ctx context.Context, owner, kbID string) error {
	return daoKb.SoftDeleteKB(ctx, kbID)
}

func ListFiles(ctx context.Context, kbID string) ([]model.KBFile, error) {
	return daoKb.ListKBFileByID(ctx, kbID)
}

func AddFileToKB(ctx context.Context, owner, kbID string, uploaded *multipart.FileHeader) error {
	// 判断文件有效性
	if err := utils.ValidateFile(uploaded); err != nil {
		return err
	}

	// 获取id对应知识库
	kbObj, err := daoKb.GetKBByID(ctx, kbID)
	if err != nil {
		return err
	}
	// 鉴权
	if kbObj.UserName != owner {
		return fmt.Errorf("knowledge base %s does not belong to user %s", kbID, owner)
	}

	// 计算唯一的存储地址
	fileID := utils.GenerateUUID()
	storedDir := filepath.Join("uploads", owner, kbID)
	if err := os.MkdirAll(storedDir, 0755); err != nil {
		return err
	}
	ext := filepath.Ext(uploaded.Filename)
	storedPath := filepath.Join(storedDir, fileID+ext)

	// 将上传文件拷贝到目标地址
	src, err := uploaded.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(storedPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		_ = os.Remove(storedPath)
		return err
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(storedPath)
		return err
	}

	// 创建知识库文件
	kbFile := &model.KBFile{
		ID:         fileID,
		KBID:       kbID,
		UserName:   owner,
		OrigName:   uploaded.Filename,
		StoredPath: storedPath,
		Status:     "pending",
	}
	if err := daoKb.CreateKBFile(ctx, kbFile); err != nil {
		_ = os.Remove(storedPath)
		return err
	}

	cleanup := func() {
		_ = os.Remove(storedPath)
		_ = daoKb.SoftDeleteKBFile(ctx, fileID)
	}

	// 创建 RAG 索引器并对文件进行向量化
	indexer, err := rag.NewRAGIndexer(ctx, kbID, config.GetConfig().RagModelConfig.RagEmbeddingModel)
	if err != nil {
		cleanup()
		return err
	}

	chunkCount, err := indexer.IndexFile(ctx, kbID, fileID, storedPath)
	if err != nil {
		cleanup()
		return err
	}

	if err := daoKb.MarkKBFileIndexed(ctx, fileID, chunkCount); err != nil {
		return err
	}

	return nil
}

func RemoveFileFromKB(ctx context.Context, owner, kbID, fileID string) error {
	return daoKb.SoftDeleteKBFile(ctx, fileID)
}
