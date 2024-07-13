package shttp

import (
	"context"
	"fmt"
	"net/http"
)

// IncomeHook hook http request before user-defined handler func.
//
// For example, if you need authorize client before real operations:
//
//	func authHook(r *http.Request) context.Context {
//		auth, err := r.Cookie("auth_key")
//		if err != nil || auth.Valid() != nil {
//			return r.Context()
//		}
//		if auth.Value == "" {
//			return r.Context()
//		}
//		return context.WithValue(r.Context(), "user", auth)
//	}
//
// A hook can abort request or allow it continue.
type IncomeHook func(r *http.Request) context.Context

type ReplyHook func(w http.ResponseWriter, r *http.Request) context.Context

func abortHandleFunc(w http.ResponseWriter, r *http.Request) {
	abortVal := r.Context().Value(abortContextKey)
	if abortVal == nil {
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: invalid nil abort
		return
	}
	abort, ok := abortVal.(*abortValueType)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: incorrect abort value
		return
	}
	if abort.redirect {
		http.Redirect(w, r, abort.reason, abort.status)
		return
	}
	w.WriteHeader(abort.status)
	w.Write([]byte(abort.reason))
}

func WithRequestHooks(handle http.HandlerFunc, hooks ...IncomeHook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, hook := range hooks {
			ctx := hook(r)
			if !isValidContext(ctx) {
				continue
			}
			r = r.WithContext(ctx)
			if isAborted(ctx) {
				abortHandleFunc(w, r)
				return
			}
		}
		fmt.Println("prepare handle")
		handle(w, r)
	}
}

func WithResponseHooks(handle http.HandlerFunc, hooks ...ReplyHook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handle(w, r)

		ctx := r.Context()
		if isAborted(ctx) {
			abortHandleFunc(w, r)
			return
		}

		for _, hook := range hooks {
			ctx := hook(w, r)
			if !isValidContext(ctx) {
				continue
			}
			r = r.WithContext(ctx)
			if isAborted(ctx) {
				abortHandleFunc(w, r)
				return
			}
		}
	}
}
