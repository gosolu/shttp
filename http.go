// Package shttp contains some golang HTTP utilities
package shttp

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gosolu/slog"
	"github.com/gosolu/solu"
)

var (
	features = "metric"

	enableMetric = false
	enableTrace  = false
	enableSlog   = false
)

func init() {
	fs := strings.Split(features, ",")
	for _, f := range fs {
		switch strings.ToLower(f) {
		case "metric":
			enableMetric = true
		case "trace":
			enableTrace = true
		case "slog":
			enableSlog = true
		}
	}
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

	if enableTrace {
		// add trace header
		if req.Header.Get(TraceparentHeader) == "" {
			trace := solu.TraceparentValue(ctx)
			req.Header.Set(TraceparentHeader, trace)
		}
	}

	start := time.Now()
	res, err := client.Do(req)

	if enableMetric {
		labels := metricLabels(req, res, time.Now().Sub(start))
		if err != nil {
			httpErrorCounter.WithLabelValues(labels...).Inc()
		} else {
			httpRequestCounter.WithLabelValues(labels...).Inc()
		}
	}
	if enableSlog {
		fields := []slog.Field{
			slog.Str("method", req.Method),
			slog.Str("host", req.URL.Host),
			slog.Str("path", req.URL.Path),
		}
		if res != nil {
			fields = append(fields, slog.I("code", res.StatusCode), slog.F64("duration", time.Now().Sub(start).Seconds()))
		}
		if err != nil {
			fields = append(fields, slog.Err(err))
		}
		slog.In(ctx).With(fields...).Info("Request")
	}

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
