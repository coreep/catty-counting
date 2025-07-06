package deps

import (
	"log/slog"
)

// Dependency provider to be held inside context.
// HARD-LIMITED to logger and DB connection. NEVER EVER expand it.
type Deps struct {
	logger *slog.Logger
	// .toai todo[add attribute db *gorm.DB]
}

func NewDeps(logger *slog.Logger) *Deps {
	return &Deps{
		logger: logger,
		// .toai todo[add attribute db *gorm.DB to parameters of the function and use it here]
	}
}

// new *Deps = merge(ctx.deps, with)
func (deps *Deps) Combine(with *Deps) *Deps {
	newDeps := &Deps{logger: deps.Logger()}
	if with.logger != nil {
		newDeps.logger = with.logger
	}
	// .toai todo[add simmilar if for db]
	return newDeps
}

func (deps *Deps) Logger() *slog.Logger {
	return deps.logger
}

//.toai todo[add simmilar DB() method as Logger()]
