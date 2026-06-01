package kb

import (
	"GopherAI/common/mysql"
	"GopherAI/model"
	"context"
	"time"
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
	err := mysql.DB.WithContext(ctx).Model(&model.KnowledgeBase{}).Where("id = ?", kbID).Update("DeletedAt", time.Now()).Error
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

func GetKBFileByID(ctx context.Context, fileID string) (*model.KBFile, error)
func ListKBFileByID(ctx context.Context, kbID string) ([]model.KBFile, error)
func MarkKBFileIndexed(ctx context.Context, fileID string, chunkCount int) error
func SoftDeleteKBFile(ctx context.Context, fileID string) error
