package api

import (
	"github.com/gin-gonic/gin"
	"net/http"

	"context"

	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/config"
)

type Api struct {
	deps deps.Deps
}

func NewApi(deps deps.Deps) *Api {
	return &Api{deps: deps}
}

func Run(ctx context.Context) {
	// TODO: TOAI: use Gin to create a new server. It should listen on port specified by config.ApiPort()
	// TODO: TOAI: api should log all access to endpoints. Preferably in JSON format. Preferably using deps.Logger (slog.Logger)
	// TODO: TOAI: api should handle panics in handlers and not fail
	// TODO: TOAI: api should provide an endpoint /api/file/:key . It is a file downloading endpoint, that should provide corresponding db.ExposedFile by specified key. The content of file that should be provided to the user should be taken from deps.Files (blob.FileBucket) by corresponding db.ExposedFile.File.BlobKey
}
