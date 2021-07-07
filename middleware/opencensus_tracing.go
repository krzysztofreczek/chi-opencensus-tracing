package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

const (
	headerNameOpencensusSpan           = "X-Opencensus-Span"
	headerNameOpencensusSpanEventIDKey = "X-Opencensus-Event-ID"
	spanRequestPayloadAttributeKey     = "request_payload"
	spanResponsePayloadAttributeKey    = "response_payload"
	payloadSizeLimit                   = 256
	payloadTruncatedMessage            = "...[payload has been truncated]"
)

// AddTracingSpanToRequest resolves span data from the provided context and injects it to the request
func AddTracingSpanToRequest(ctx context.Context, r *http.Request) {
	span := trace.FromContext(ctx)
	if span == nil {
		return
	}
	addSpanMessageSentEvent(span, r)
	setSpanHeader(span.SpanContext(), r)
}

// OpencensusTracing implements a simple middleware handler
// for adding an opencensus tracing span to the request context
func OpencensusTracing() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := decorateResponseWriter(w)

			body := decorateRequestBody(r)
			r.Body = body

			ctx := r.Context()
			var span *trace.Span

			parentSpanContext, ok := getSpanContext(r)
			if ok {
				ctx, span = trace.StartSpanWithRemoteParent(ctx, "", parentSpanContext)
				span.AddLink(trace.Link{
					TraceID:    parentSpanContext.TraceID,
					SpanID:     parentSpanContext.SpanID,
					Type:       trace.LinkTypeParent,
					Attributes: nil,
				})
			} else {
				ctx, span = trace.StartSpan(ctx, "")
			}

			defer closeSpan(span, ww)
			defer setSpanResponsePayloadAttribute(span, ww)
			defer setSpanRequestPayloadAttribute(span, body)
			defer addSpanMessageReceiveEvent(span, r)
			defer setSpanNameAndURLAttributes(span, r)

			next.ServeHTTP(ww, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
}

func setSpanHeader(sc trace.SpanContext, r *http.Request) {
	bin := propagation.Binary(sc)
	b64 := base64.StdEncoding.EncodeToString(bin)
	r.Header.Set(headerNameOpencensusSpan, b64)
}

func getSpanContext(r *http.Request) (sc trace.SpanContext, ok bool) {
	b64 := r.Header.Get(headerNameOpencensusSpan)
	if b64 == "" {
		return trace.SpanContext{}, false
	}

	bin, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return trace.SpanContext{}, false
	}

	return propagation.FromBinary(bin)
}

func closeSpan(span *trace.Span, w *responseWriterDecorator) {
	if w.StatusCode() < 400 {
		span.SetStatus(trace.Status{
			Code:    trace.StatusCodeOK,
			Message: "OK",
		})
	} else {
		span.SetStatus(trace.Status{
			Code:    trace.StatusCodeUnknown,
			Message: fmt.Sprintf("Response status code: %d", w.StatusCode()),
		})
	}
	span.End()
}

func addSpanMessageReceiveEvent(span *trace.Span, r *http.Request) {
	eIDString := r.Header.Get(headerNameOpencensusSpanEventIDKey)
	eID, _ := strconv.ParseInt(eIDString, 10, 64)
	span.AddMessageReceiveEvent(eID, r.ContentLength, 0)
}

func addSpanMessageSentEvent(span *trace.Span, r *http.Request) {
	eID := generateEventID()
	eIDString := strconv.FormatInt(eID, 10)
	r.Header.Set(headerNameOpencensusSpanEventIDKey, eIDString)
	span.AddMessageSendEvent(eID, r.ContentLength, 0)
}

func setSpanRequestPayloadAttribute(span *trace.Span, body *requestBodyDecorator) {
	var payload string
	if body != nil {
		payload = string(body.Payload())
	}
	if len(payload) > payloadSizeLimit {
		payload = payload[:payloadSizeLimit-len(payloadTruncatedMessage)]
		payload += payloadTruncatedMessage
	}
	span.AddAttributes(trace.StringAttribute(spanRequestPayloadAttributeKey, payload))
}

func setSpanResponsePayloadAttribute(span *trace.Span, w *responseWriterDecorator) {
	payload := string(w.Payload())
	if len(payload) > payloadSizeLimit {
		payload = payload[:payloadSizeLimit-len(payloadTruncatedMessage)]
		payload += payloadTruncatedMessage
	}
	span.AddAttributes(trace.StringAttribute(spanResponsePayloadAttributeKey, payload))
}

func setSpanNameAndURLAttributes(span *trace.Span, r *http.Request) {
	rCtx := chi.RouteContext(r.Context())

	spanName := fmt.Sprintf("[%s] %s", r.Method, rCtx.RoutePattern())
	span.SetName(spanName)

	for _, key := range rCtx.URLParams.Keys {
		span.AddAttributes(trace.StringAttribute(key, rCtx.URLParam(key)))
	}
}

func generateEventID() int64 {
	eID, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return 0
	}
	return eID.Int64()
}
