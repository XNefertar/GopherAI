package kb

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

<<<<<<< Updated upstream
=======
	"GopherAI/common/logger"
>>>>>>> Stashed changes
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
<<<<<<< Updated upstream
=======
	if err != nil {
		logger.With("userName", owner, "kbID", kbObj.ID).Error("CreateKB failed", "error", err)
	}
>>>>>>> Stashed changes
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
<<<<<<< Updated upstream
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
=======
	kbs, err := daoKb.ListKBByOwner(ctx, owner)
	if err != nil {
		logger.With("userName", owner).Error("ListKB failed", "error", err)
	}
	return kbs, err
}

func DeleteKB(ctx context.Context, owner, kbID string) error {
	err := daoKb.SoftDeleteKB(ctx, kbID)
	if err != nil {
		logger.With("userName", owner, "kbID", kbID).Error("DeleteKB failed", "error", err)
	}
	return err
}

func ListFiles(ctx context.Context, kbID string) ([]model.KBFile, error) {
	files, err := daoKb.ListKBFileByID(ctx, kbID)
	if err != nil {
		logger.With("kbID", kbID).Error("ListFiles failed", "error", err)
	}
	return files, err
}

func AddFileToKB(ctx context.Context, owner, kbID string, uploaded *multipart.FileHeader) error {
	fileID := utils.GenerateUUID()
	l := logger.With("userName", owner, "kbID", kbID, "fileID", fileID)

	// 判断文件有效性
	if err := utils.ValidateFile(uploaded); err != nil {
		l.Error("file validation failed", "error", err)
>>>>>>> Stashed changes
		return err
	}

	// 获取id对应知识库
	kbObj, err := daoKb.GetKBByID(ctx, kbID)
	if err != nil {
<<<<<<< Updated upstream
=======
		l.Error("GetKBByID failed", "error", err)
>>>>>>> Stashed changes
		return err
	}
	// 鉴权
	if kbObj.UserName != owner {
<<<<<<< Updated upstream
=======
		l.Warn("kb access denied", "kbOwner", kbObj.UserName)
>>>>>>> Stashed changes
		return fmt.Errorf("knowledge base %s does not belong to user %s", kbID, owner)
	}

	// 计算唯一的存储地址
<<<<<<< Updated upstream
	fileID := utils.GenerateUUID()
	storedDir := filepath.Join("uploads", owner, kbID)
	if err := os.MkdirAll(storedDir, 0755); err != nil {
=======
	storedDir := filepath.Join("uploads", owner, kbID)
	if err := os.MkdirAll(storedDir, 0755); err != nil {
		l.Error("create storage directory failed", "error", err)
>>>>>>> Stashed changes
		return err
	}
	ext := filepath.Ext(uploaded.Filename)
	storedPath := filepath.Join(storedDir, fileID+ext)

	// 将上传文件拷贝到目标地址
	src, err := uploaded.Open()
	if err != nil {
<<<<<<< Updated upstream
=======
		l.Error("open uploaded file failed", "error", err)
>>>>>>> Stashed changes
		return err
	}
	defer src.Close()

	dst, err := os.Create(storedPath)
	if err != nil {
<<<<<<< Updated upstream
=======
		l.Error("create destination file failed", "error", err)
>>>>>>> Stashed changes
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		_ = os.Remove(storedPath)
<<<<<<< Updated upstream
=======
		l.Error("copy file failed", "error", err)
>>>>>>> Stashed changes
		return err
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(storedPath)
<<<<<<< Updated upstream
		return err
	}

=======
		l.Error("close destination file failed", "error", err)
		return err
	}

	l.Info("file uploaded", "path", storedPath)

>>>>>>> Stashed changes
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
<<<<<<< Updated upstream
=======
		l.Error("CreateKBFile failed", "error", err)
>>>>>>> Stashed changes
		return err
	}

	cleanup := func() {
		_ = os.Remove(storedPath)
		_ = daoKb.SoftDeleteKBFile(ctx, fileID)
	}

	// 创建 RAG 索引器并对文件进行向量化
	indexer, err := rag.NewRAGIndexer(ctx, kbID, config.GetConfig().RagModelConfig.RagEmbeddingModel)
	if err != nil {
<<<<<<< Updated upstream
=======
		l.Error("create RAG indexer failed", "error", err)
>>>>>>> Stashed changes
		cleanup()
		return err
	}

	chunkCount, err := indexer.IndexFile(ctx, kbID, fileID, storedPath)
	if err != nil {
<<<<<<< Updated upstream
=======
		l.Error("index file failed", "error", err)
>>>>>>> Stashed changes
		cleanup()
		return err
	}

	if err := daoKb.MarkKBFileIndexed(ctx, fileID, chunkCount); err != nil {
<<<<<<< Updated upstream
		return err
	}

=======
		l.Error("mark file indexed failed", "error", err)
		return err
	}

	l.Info("file indexed", "chunkCount", chunkCount)
>>>>>>> Stashed changes
	return nil
}

func RemoveFileFromKB(ctx context.Context, owner, kbID, fileID string) error {
<<<<<<< Updated upstream
	return daoKb.SoftDeleteKBFile(ctx, fileID)
=======
	l := logger.With("userName", owner, "kbID", kbID, "fileID", fileID)
	err := daoKb.SoftDeleteKBFile(ctx, fileID)
	if err != nil {
		l.Error("RemoveFileFromKB failed", "error", err)
	}
	return err
>>>>>>> Stashed changes
}
