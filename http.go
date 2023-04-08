// Package shttp contains some golang HTTP utilities
package shttp

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gosolu/solu"
)

type ctxKeyType struct{}

var (
	ctxMetricKey ctxKeyType
	ctxTraceKey  ctxKeyType
	ctxLogKey    ctxKeyType
)

// EnableMetric enable http metrics
func EnableMetric(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxMetricKey, true)
}

// DisableMetric disable http metrics
func DisableMetric(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxMetricKey, false)
}

func EnableTrace(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTraceKey, true)
}

func DisableTrace(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTraceKey, false)
}

func hasEnabledTrace(ctx context.Context) bool {
	val := ctx.Value(ctxTraceKey)
	if val == nil {
		return false
	}
	b, ok := val.(bool)
	if !ok {
		return false
	}
	return b
}

func EnableLog(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxLogKey, true)
}

func DisableLog(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxLogKey, false)
}

type Logger interface {
	Log(template string, args ...string)
}

const (
	TraceparentHeader = "traceparent"
	TracestateHeader  = "tracestate"
)

func metricLabels(req *http.Request, res *http.Response, dur time.Duration) []string {
	if req == nil && res != nil {
		req = res.Request
	}
	if req == nil {
		return nil
	}
	var statusCode int
	if res != nil {
		statusCode = res.StatusCode
	}
	labels := make([]string, 0, len(requestLabels))
	labels = append(labels, req.Method)
	labels = append(labels, req.Host)
	labels = append(labels, req.URL.Path)
	labels = append(labels, strconv.Itoa(statusCode))
	labels = append(labels, strconv.FormatInt(dur.Milliseconds(), 10))
	return labels
}

func doWithClient(ctx context.Context, req *http.Request, client *http.Client) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("invalid request")
	}
	if client == nil {
		return nil, fmt.Errorf("invalid client")
	}

	if hasEnabledTrace(ctx) {
		// add trace header
		if req.Header.Get(TraceparentHeader) == "" {
			trace := solu.TraceparentValue(ctx)
			req.Header.Set(TraceparentHeader, trace)
		}
	}

	res, err := client.Do(req)
	return res, err
}

// Do http request, use http.DefaultClient.Do
// Wrap request with metrics and trace headers
func Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return doWithClient(ctx, req, http.DefaultClient)
}

// DoClient do http request, use client
// Wrap request with metrics and trace headers
func DoClient(ctx context.Context, req *http.Request, client *http.Client) (*http.Response, error) {
	return doWithClient(ctx, req, client)
}

func mergeTrace(ctx context.Context, res *http.Response) context.Context {
	if res == nil {
		return ctx
	}
	header := res.Header.Get(TraceparentHeader)
	if header == "" {
		return ctx
	}
	tid, sid, err := solu.ParseTraceparent(header)
	if err != nil {
		return ctx
	}
	if solu.TraceID(ctx) == "" {
		ctx = solu.TraceWith(ctx, tid)
	}
	if solu.SpanID(ctx) == "" {
		ctx = solu.SpanWith(ctx, sid)
	}
	return ctx
}

// InheritTrace extract trace informations from http response and return a context
// with trace ID and span ID if them exist.
func InheritTrace(res *http.Response) context.Context {
	ctx := context.Background()
	return mergeTrace(ctx, res)
}

// FulfillTrace extract trace informations from http response and fulfill income
// context with trace or span IDs.
func FulfillTrace(ctx context.Context, res *http.Response) context.Context {
	return mergeTrace(ctx, res)
}
