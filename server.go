package shttp

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

type (
	abortContextType struct{}
	abortValueType   struct {
		status   int
		reason   string
		redirect bool
	}
)

var (
	abortContextKey abortContextType

	abortDefaultReason = "Server encounter an error"
)

func isAborted(ctx context.Context) bool {
	val, ok := ctx.Value(abortContextKey).(*abortValueType)
	if !ok {
		return false
	}
	if val == nil {
		return false
	}
	return val.status != 0
}

func Abort(ctx context.Context) context.Context {
	val := &abortValueType{
		status: http.StatusInternalServerError,
		reason: abortDefaultReason,
	}
	return context.WithValue(ctx, abortContextKey, val)
}

func AbortWithStatus(ctx context.Context, status int) context.Context {
	val := &abortValueType{
		status: status,
		reason: abortDefaultReason,
	}
	return context.WithValue(ctx, abortContextKey, val)
}

func AbortWithStatusReason(ctx context.Context, status int, reason string) context.Context {
	val := &abortValueType{
		status: status,
		reason: reason,
	}
	return context.WithValue(ctx, abortContextKey, val)
}

func AbortWithError(ctx context.Context, status int, err error) context.Context {
	var reason string
	if err != nil {
		reason = err.Error()
	}
	val := &abortValueType{
		status: status,
		reason: reason,
	}
	return context.WithValue(ctx, abortContextKey, val)
}

func Redirect(ctx context.Context, status int, path string) context.Context {
	val := &abortValueType{
		status:   status,
		reason:   path,
		redirect: true,
	}
	return context.WithValue(ctx, abortContextKey, val)
}

type contextHooks struct {
	sync.Mutex

	hooks []ContextHook
}

func (ch *contextHooks) Add(hook ContextHook) {
	ch.Lock()
	defer ch.Unlock()
	ch.hooks = append(ch.hooks, hook)
}

var (
	gBaseHooks = &contextHooks{
		hooks: make([]ContextHook, 0),
	}

	gConnHooks = &contextHooks{
		hooks: make([]ContextHook, 0),
	}
)

// AddBaseHooks add middlewares to base context
func AddBaseHooks(hooks ...ContextHook) {
	for _, hook := range hooks {
		gBaseHooks.Add(hook)
	}
}

// AddConnHooks add connection hooks
func AddConnHooks(hooks ...ContextHook) {
	for _, hook := range hooks {
		gConnHooks.Add(hook)
	}
}

type startupContextType struct{}

var startupContextKey startupContextType

func isValidContext(ctx context.Context, ts ...int64) bool {
	if ctx == nil {
		return false
	}
	val := ctx.Value(startupContextKey)
	if val == nil {
		return false
	}
	if start, ok := val.(int64); !ok || start <= 0 {
		return false
	} else if len(ts) > 0 && start != ts[0] {
		return false
	}
	return true
}

func ListenAndServe(addr string, handler http.Handler) error {
	if handler != nil {
		gRouter.NotFound = handler
	}

	startTime := time.Now().Unix()

	baseContext := func(ln net.Listener) context.Context {
		gBaseHooks.Lock()
		defer gBaseHooks.Unlock()

		ctx := context.Background()
		ctx = context.WithValue(ctx, startupContextKey, startTime)
		for _, fn := range gBaseHooks.hooks {
			fnCtx := fn(ctx)
			// check startup context to verify context
			if !isValidContext(fnCtx, startTime) {
				slog.WarnContext(ctx, "base hook returns invalid context")
				continue
			}
			ctx = fnCtx
		}
		return ctx
	}

	connContext := func(ctx context.Context, c net.Conn) context.Context {
		gConnHooks.Lock()
		defer gConnHooks.Unlock()

		for _, fn := range gConnHooks.hooks {
			fnCtx := fn(ctx)
			// check startup context to verify context
			if !isValidContext(fnCtx, startTime) {
				slog.WarnContext(ctx, "connection hook returns invalid context")
				continue
			}
			ctx = fnCtx
		}
		return ctx
	}
	server := &http.Server{
		Addr:        addr,
		Handler:     gRouter,
		BaseContext: baseContext,
		ConnContext: connContext,
	}
	return server.ListenAndServe()
}
