package shttp

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/julienschmidt/httprouter"
)

var allowTrailingSlash atomic.Bool

var gRouter = &httprouter.Router{
	RedirectTrailingSlash:  false,
	RedirectFixedPath:      true,
	HandleMethodNotAllowed: false,
	HandleOPTIONS:          true,
}

func AllowTrailingSlash(allow bool) {
	allowTrailingSlash.Store(allow)
}

func route(method, path string, handle http.HandlerFunc) {
	gRouter.Handler(method, path, handle)
	if allowTrailingSlash.Load() && !strings.HasSuffix(path, "/") {
		gRouter.Handler(method, path+"/", handle)
	}
}

func GET(path string, handle http.HandlerFunc) {
	route(http.MethodGet, path, handle)
}

func HEAD(path string, handle http.HandlerFunc) {
	route(http.MethodHead, path, handle)
}

func POST(path string, handle http.HandlerFunc) {
	route(http.MethodPost, path, handle)
}

func PUT(path string, handle http.HandlerFunc) {
	route(http.MethodPut, path, handle)
}

func PATCH(path string, handle http.HandlerFunc) {
	route(http.MethodPatch, path, handle)
}

func DELETE(path string, handle http.HandlerFunc) {
	route(http.MethodDelete, path, handle)
}

func CONNECT(path string, handle http.HandlerFunc) {
	route(http.MethodConnect, path, handle)
}

func OPTIONS(path string, handle http.HandlerFunc) {
	route(http.MethodOptions, path, handle)
}

func TRACE(path string, handle http.HandlerFunc) {
	route(http.MethodTrace, path, handle)
}

func NotFound(handle http.HandlerFunc) {
	gRouter.NotFound = handle
}

// Param alias to httprouter Param
type Param = httprouter.Param

// Params alias to httprouter Params
type Params = httprouter.Params

// RouterParams extract http router params from context
func RouterParams(ctx context.Context) Params {
	return httprouter.ParamsFromContext(ctx)
}
