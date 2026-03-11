package memory

import (
	"github.com/qdrant/go-client/qdrant"
	"gorm.io/gorm"
)

type DBManager struct {
	DB        *gorm.DB
	QdrantCli *QdrantManager
}

type QdrantManager struct {
	Client     *qdrant.Client
	Collection string
}
