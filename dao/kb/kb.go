package kb

import (
	"GopherAI/common/mysql"
	"GopherAI/model"
	"context"
)

const (
	DefaultKBName = "__default__"
)

func CreateKB(ctx context.Context, kb *model.KnowledgeBase) error {
	err := mysql.DB.WithContext(ctx).Create(kb).Error
	return err
}

func GetKBByID(ctx context.Context, kbID string) (*model.KnowledgeBase, error) {
	kb := new(model.KnowledgeBase)
	err := mysql.DB.WithContext(ctx).Where("id = ?", kbID).First(kb).Error
	return kb, err
}

func ListKBByOwner(ctx context.Context, owner string) ([]model.KnowledgeBase, error) {
	var kbList []model.KnowledgeBase
	err := mysql.DB.WithContext(ctx).Where("user_name = ?", owner).Find(&kbList).Error
	return kbList, err
}

func SoftDeleteKB(ctx context.Context, kbID string) error {
	err := mysql.DB.WithContext(ctx).Where("id = ?", kbID).Delete(&model.KnowledgeBase{}).Error
	return err
}

func GetDefaultKBByOwner(ctx context.Context, owner string) (*model.KnowledgeBase, error) {
	kb := new(model.KnowledgeBase)
	err := mysql.DB.WithContext(ctx).Where("user_name = ? AND name = ?", owner, DefaultKBName).First(kb).Error
	return kb, err
}

func CreateKBFile(ctx context.Context, f *model.KBFile) error {
	return mysql.DB.WithContext(ctx).Create(f).Error
}

func GetKBFileByID(ctx context.Context, fileID string) (*model.KBFile, error) {
	kbFile := new(model.KBFile)
	err := mysql.DB.WithContext(ctx).Where("id = ?", fileID).First(&kbFile).Error
	return kbFile, err
}

func ListKBFileByID(ctx context.Context, kbID string) ([]model.KBFile, error) {
	var kbFileList []model.KBFile
	err := mysql.DB.WithContext(ctx).Where("kb_id = ?", kbID).Find(&kbFileList).Error
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
	return err
}

func SoftDeleteKBFile(ctx context.Context, fileID string) error {
	err := mysql.DB.WithContext(ctx).Where("id = ?", fileID).Delete(&model.KBFile{}).Error
	return err
}
