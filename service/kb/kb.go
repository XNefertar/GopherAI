package kb

import (
	"context"
	"mime/multipart"

	daoKb "GopherAI/dao/kb"
	"GopherAI/model"

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
	// TODO: implement actual file storage and vector indexing
	return nil
}

func RemoveFileFromKB(ctx context.Context, owner, kbID, fileID string) error {
	return daoKb.SoftDeleteKBFile(ctx, fileID)
}
