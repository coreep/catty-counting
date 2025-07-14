package deps

import (
	"log/slog"
	"google.golang.org/genai"
)

// Dependency provider to be held inside context.
// HARD-LIMITED to logger and DB connection. NEVER EVER expand it.
type Deps struct {
	logger *slog.Logger
	llm *genai.Client
	// .toai todo[add attribute db *gorm.DB]
}

func NewDeps(logger *slog.Logger, llm *genai.Client) *Deps {
	return &Deps{
		logger: logger,
		llm: llm,
		// .toai todo[add attribute db *gorm.DB to parameters of the function and use it here]
	}
}

// new *Deps = merge(ctx.deps, with)
func (deps *Deps) Combine(with *Deps) *Deps {
	newDeps := &Deps{logger: deps.Logger(), llm: deps.LLM()}
	if with.logger != nil {
		newDeps.logger = with.logger
	}
	if with.llm != nil {
		newDeps.llm = with.llm
	}
	// .toai todo[add simmilar if for db]
	return newDeps
}

func (deps *Deps) Logger() *slog.Logger {
	return deps.logger
}

func (deps *Deps) LLM() *genai.Client {
	return deps.llm
}

//.toai todo[add simmilar DB() method as Logger()]
