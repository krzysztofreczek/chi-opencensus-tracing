package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"strconv"

	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

const (
	headerNameOpencensusSpan           = "X-Opencensus-Span"
	headerNameOpencensusSpanEventIDKey = "X-Opencensus-event-id"
	spanRequestPayloadAttributeKey     = "event-payload"
	spanResponsePayloadAttributeKey    = "event-response"
)

// AddTracingSpanToRequest resolves span data from the provided context and injects it to the request
func AddTracingSpanToRequest(ctx context.Context, r *http.Request) {
	body := decorateRequestBody(r)
	r.Body = body

	span := trace.FromContext(ctx)
	if span == nil {
		return
	}

	eID := generateEventID()
	eIDString := strconv.FormatInt(eID, 10)
	r.Header.Set(headerNameOpencensusSpanEventIDKey, eIDString)

	// TODO: test it!
	span.AddMessageSendEvent(eID, r.ContentLength, 0)
	span.AddAttributes(trace.StringAttribute(spanRequestPayloadAttributeKey, string(body.Payload())))

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
				ctx, span = trace.StartSpanWithRemoteParent(ctx, spanName(r), parentSpanContext)
				span.AddLink(trace.Link{
					TraceID:    parentSpanContext.TraceID,
					SpanID:     parentSpanContext.SpanID,
					Type:       trace.LinkTypeParent,
					Attributes: nil,
				})
			} else {
				ctx, span = trace.StartSpan(ctx, spanName(r))
			}

			defer func() {
				eIDString := r.Header.Get(headerNameOpencensusSpanEventIDKey)
				eID, _ := strconv.ParseInt(eIDString, 10, 64)

				// TODO: test it!
				span.AddMessageReceiveEvent(eID, ww.ContentLength(), 0)
				span.AddAttributes(trace.StringAttribute(spanResponsePayloadAttributeKey, string(ww.Payload())))
			}()

			defer func() {
				if ww.StatusCode() < 400 {
					span.SetStatus(trace.Status{
						Code:    trace.StatusCodeOK,
						Message: "OK",
					})
				} else {
					span.SetStatus(trace.Status{
						Code:    trace.StatusCodeUnknown,
						Message: fmt.Sprintf("Response status code: %d", ww.StatusCode()),
					})
				}
				span.End()
			}()

			// TODO: test it!
			span.AddAttributes(trace.StringAttribute(spanRequestPayloadAttributeKey, string(body.Payload())))

			next.ServeHTTP(ww, r.WithContext(ctx))
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

func generateEventID() int64 {
	maxUint64 := ^uint64(0)
	maxInt64 := int64(maxUint64 >> 1)

	eID, err := rand.Int(rand.Reader, big.NewInt(maxInt64))
	if err != nil {
		return 0
	}

	return eID.Int64()
}
