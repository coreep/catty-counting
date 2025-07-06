package deps

import (
	"context"
)

// Context+Context-Level-Dependencies
type Context struct {
	context.Context
	deps *Deps
}

func NewContext(cctx context.Context, deps *Deps) Context {
	return Context{Context: cctx, deps: deps}
}

func (ctx *Context) Deps() *Deps {
	return ctx.deps
}

func (ctx *Context) WithDeps(deps *Deps) Context {
	newCctx := context.WithValue(ctx, "", "")
	newDeps := ctx.Deps().Combine(deps)
	newCtx := NewContext(newCctx, newDeps)
	return newCtx
}

func (ctx *Context) WithCancel() (newCtx Context, cancel func()) {
	cctx, cancel := context.WithCancel(ctx)
	newCtx = NewContext(cctx, ctx.Deps())
	return newCtx, cancel
}
