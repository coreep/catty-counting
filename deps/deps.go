package deps

import (
	"context"
	"log/slog"
)

// Dependency provider to be held inside context
// HARD-LIMITED to logger and DB connection. NEVER EVER expand it.
type Deps struct {
	logger *slog.Logger
	// db connection
}

func NewDeps(logger *slog.Logger) *Deps {
	return &Deps{logger: logger}
}

func (this *Deps) Logger() *slog.Logger {
	return this.logger
}

// Context+Context-Level-Dependencies
type Context struct {
	context.Context
	deps *Deps
}

func NewContext(cctx context.Context, deps *Deps) *Context {
	return &Context{Context: cctx, deps: deps}
}

func (this *Context) Deps() *Deps {
	return this.deps
}
