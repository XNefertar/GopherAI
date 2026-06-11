package kb

import (
<<<<<<< Updated upstream
=======
	"GopherAI/common/logger"
>>>>>>> Stashed changes
	"GopherAI/common/mysql"
	"GopherAI/model"
	"context"
)

const (
	DefaultKBName = "__default__"
)

func CreateKB(ctx context.Context, kb *model.KnowledgeBase) error {
	err := mysql.DB.WithContext(ctx).Create(kb).Error
<<<<<<< Updated upstream
=======
	if err != nil {
		logger.With("userName", kb.UserName, "kbID", kb.ID).Error("CreateKB failed", "error", err)
	}
>>>>>>> Stashed changes
	return err
}

func GetKBByID(ctx context.Context, kbID string) (*model.KnowledgeBase, error) {
	kb := new(model.KnowledgeBase)
	err := mysql.DB.WithContext(ctx).Where("id = ?", kbID).First(kb).Error
<<<<<<< Updated upstream
=======
	if err != nil {
		logger.With("kbID", kbID).Error("GetKBByID failed", "error", err)
	}
>>>>>>> Stashed changes
	return kb, err
}

func ListKBByOwner(ctx context.Context, owner string) ([]model.KnowledgeBase, error) {
	var kbList []model.KnowledgeBase
	err := mysql.DB.WithContext(ctx).Where("user_name = ?", owner).Find(&kbList).Error
<<<<<<< Updated upstream
=======
	if err != nil {
		logger.With("userName", owner).Error("ListKBByOwner failed", "error", err)
	}
>>>>>>> Stashed changes
	return kbList, err
}

func SoftDeleteKB(ctx context.Context, kbID string) error {
	err := mysql.DB.WithContext(ctx).Where("id = ?", kbID).Delete(&model.KnowledgeBase{}).Error
<<<<<<< Updated upstream
=======
	if err != nil {
		logger.With("kbID", kbID).Error("SoftDeleteKB failed", "error", err)
	}
>>>>>>> Stashed changes
	return err
}

func GetDefaultKBByOwner(ctx context.Context, owner string) (*model.KnowledgeBase, error) {
	kb := new(model.KnowledgeBase)
	err := mysql.DB.WithContext(ctx).Where("user_name = ? AND name = ?", owner, DefaultKBName).First(kb).Error
<<<<<<< Updated upstream
=======
	if err != nil {
		logger.With("userName", owner).Error("GetDefaultKBByOwner failed", "error", err)
	}
>>>>>>> Stashed changes
	return kb, err
}

func CreateKBFile(ctx context.Context, f *model.KBFile) error {
<<<<<<< Updated upstream
	return mysql.DB.WithContext(ctx).Create(f).Error
=======
	err := mysql.DB.WithContext(ctx).Create(f).Error
	if err != nil {
		logger.With("userName", f.UserName, "kbID", f.KBID, "fileID", f.ID).Error("CreateKBFile failed", "error", err)
	}
	return err
>>>>>>> Stashed changes
}

func GetKBFileByID(ctx context.Context, fileID string) (*model.KBFile, error) {
	kbFile := new(model.KBFile)
	err := mysql.DB.WithContext(ctx).Where("id = ?", fileID).First(&kbFile).Error
<<<<<<< Updated upstream
=======
	if err != nil {
		logger.With("fileID", fileID).Error("GetKBFileByID failed", "error", err)
	}
>>>>>>> Stashed changes
	return kbFile, err
}

func ListKBFileByID(ctx context.Context, kbID string) ([]model.KBFile, error) {
	var kbFileList []model.KBFile
	err := mysql.DB.WithContext(ctx).Where("kb_id = ?", kbID).Find(&kbFileList).Error
<<<<<<< Updated upstream
=======
	if err != nil {
		logger.With("kbID", kbID).Error("ListKBFileByID failed", "error", err)
	}
>>>>>>> Stashed changes
	return kbFileList, err
}

func MarkKBFileIndexed(ctx context.Context, fileID string, chunkCount int) error {
	err := mysql.DB.WithContext(ctx).
		Model(&model.KBFile{}).
		Where("id = ?", fileID).
		Updates(map[string]interface{}{
			"Status":     "indexed",
			"ChunkCount": chunkCount,
		}).Error
<<<<<<< Updated upstream
=======
	if err != nil {
		logger.With("fileID", fileID).Error("MarkKBFileIndexed failed", "error", err)
	}
>>>>>>> Stashed changes
	return err
}

func SoftDeleteKBFile(ctx context.Context, fileID string) error {
	err := mysql.DB.WithContext(ctx).Where("id = ?", fileID).Delete(&model.KBFile{}).Error
<<<<<<< Updated upstream
=======
	if err != nil {
		logger.With("fileID", fileID).Error("SoftDeleteKBFile failed", "error", err)
	}
>>>>>>> Stashed changes
	return err
}
