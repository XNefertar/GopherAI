package model

import (
	"time"

	"gorm.io/gorm"
)

type KnowledgeBase struct {
	ID          string `gorm:"primaryKey;type:varchar(36)"`
	UserName    string `gorm:"index;not null"`
	Name        string `gorm:"type:varchar(100);not null"`
	Description string `gorm:"type:varchar(500)"`
	Dimension   int
	EmbedModel  string `gorm:"type:varchar(64)"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type KBFile struct {
	ID         string `gorm:"primaryKey;type:varchar(36)"`
	KBID       string `gorm:"index;not null;type:varchar(36)"`
	UserName   string `gorm:"index;not null"`
	OrigName   string `gorm:"type:varchar(255)"`
	StoredPath string `gorm:"type:varchar(500)"`
	ChunkCount int
	Status     string `gorm:"type:varchar(20)"`
	CreatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}
