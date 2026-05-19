package runtime

import (
	"context"
	"sync"
)

type ActionCtx struct {
	Context context.Context
	Server  *Server
}

type ActionFunc func(ctx ActionCtx, data map[string]any) (any, error)

type ActionRegistry struct {
	mu sync.RWMutex
	m  map[string]ActionFunc
}

func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{m: map[string]ActionFunc{}}
}

func (r *ActionRegistry) Register(name string, fn ActionFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[name] = fn
}

func (r *ActionRegistry) Get(name string) (ActionFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.m[name]
	return fn, ok
}

