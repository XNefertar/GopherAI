package kb

import (
	"GopherAI/model"
	"context"
	"mime/multipart"
)

func CreateKB(ctx context.Context, owner, name, desc string) (*model.KnowledgeBase, error) {

}


func EnsureDefaultKB(ctx context.Context, owner string) (*model.KnowledgeBase, error)
func ListKB(ctx context.Context, owner string) ([]model.KnowledgeBase, error)
func DeleteKB(ctx context.Context, owner, kbID string) error
func ListFiles(ctx context.Context, kbID string) ([]model.KBFile, error)

func AddFileToKB(ctx context.Context, owner, kbID string, uploaded *multipart.FileHeader)

func RemoveFileFromKB(ctx context.Context, owner, kbID, fileID string) error
