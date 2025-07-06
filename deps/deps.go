package deps

import (
	"log/slog"
)

// Dependency provider to be held inside context.
// HARD-LIMITED to logger and DB connection. NEVER EVER expand it.
type Deps struct {
	logger *slog.Logger
	// .toai [add attribute db *gorm.DB](todo)
}

func NewDeps(logger *slog.Logger) *Deps {
	return &Deps{
		logger: logger,
		// .toai [add attribute db *gorm.DB to parameters of the function and use it here](todo)
	}
}

// new *Deps = merge(ctx.deps, with)
func (deps *Deps) Combine(with *Deps) *Deps {
	newDeps := &Deps{logger: deps.Logger()}
	if with.logger != nil {
		newDeps.logger = with.logger
	}
	// .toai [add simmilar if for db](todo)
	return newDeps
}

func (deps *Deps) Logger() *slog.Logger {
	return deps.logger
}

//.toai [add simmilar DB() method as Logger()](todo)
