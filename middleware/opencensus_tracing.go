package middleware

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

const HeaderNameOpencensusSpan = "X-Opencensus-Span"

// AddTracingSpanToRequest resolves span data from the provided context and injects it to the request
func AddTracingSpanToRequest(ctx context.Context, req *http.Request) {
	span := trace.FromContext(ctx)
	if span == nil {
		return
	}
	setSpanHeader(span.SpanContext(), req)
}

// OpencensusTracing implements a simple middleware handler
// for adding an opencensus tracing span to the request context
func OpencensusTracing() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			parentSpanContext, ok := getSpanContext(r)
			if ok {
				c, span := trace.StartSpanWithRemoteParent(ctx, spanName(r), parentSpanContext)
				span.AddLink(trace.Link{
					TraceID:    parentSpanContext.TraceID,
					SpanID:     parentSpanContext.SpanID,
					Type:       trace.LinkTypeParent,
					Attributes: nil,
				})
				span.End()
				ctx = c
			} else {
				c, span := trace.StartSpan(ctx, spanName(r))
				span.End()
				ctx = c
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
}

func spanName(r *http.Request) string {
	return fmt.Sprintf("[%s] %s", r.Method, r.URL.String())
}

func setSpanHeader(sc trace.SpanContext, r *http.Request) {
	bin := propagation.Binary(sc)
	b64 := base64.StdEncoding.EncodeToString(bin)
	r.Header.Set(HeaderNameOpencensusSpan, b64)
}

func getSpanContext(r *http.Request) (sc trace.SpanContext, ok bool) {
	b64 := r.Header.Get(HeaderNameOpencensusSpan)
	if b64 == "" {
		return trace.SpanContext{}, false
	}

	bin, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return trace.SpanContext{}, false
	}

	return propagation.FromBinary(bin)
}
