package deps

import (
	"log/slog"
)

// Dependency provider to be held inside context.
// HARD-LIMITED to logger and DB connection. NEVER EVER expand it.
type Deps struct {
	logger *slog.Logger
	// db connection
}

func NewDeps(logger *slog.Logger) *Deps {
	return &Deps{logger: logger}
}

// new *Deps = merge(ctx.deps, with)
func (deps *Deps) Combine(with *Deps) *Deps {
	newDeps := &Deps{logger: deps.Logger()}
	if with.logger != nil {
		newDeps.logger = with.logger
	}
	return newDeps
}

func (deps *Deps) Logger() *slog.Logger {
	return deps.logger
}
