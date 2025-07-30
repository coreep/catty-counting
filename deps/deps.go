package deps

import (
	"log/slog"

	"gocloud.dev/blob"
	"gorm.io/gorm"
)

type Deps struct {
	Logger *slog.Logger
	DBC    *gorm.DB
	Files  *blob.Bucket
}
