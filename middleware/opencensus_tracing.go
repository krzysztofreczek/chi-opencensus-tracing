package middleware

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

const (
	headerNameOpencensusSpan           = "X-Opencensus-Span"
	headerNameOpencensusSpanEventIDKey = "X-Opencensus-event-id"
)

// AddTracingSpanToRequest resolves span data from the provided context and injects it to the request
func AddTracingSpanToRequest(ctx context.Context, req *http.Request) {
	span := trace.FromContext(ctx)
	if span == nil {
		return
	}

	// TODO!!!
	eID := time.Now().Unix()
	eIDString := strconv.FormatInt(eID, 10)
	req.Header.Set(headerNameOpencensusSpanEventIDKey, eIDString)
	span.AddMessageSendEvent(eID, req.ContentLength, 0)

	setSpanHeader(span.SpanContext(), req)
}

// OpencensusTracing implements a simple middleware handler
// for adding an opencensus tracing span to the request context
func OpencensusTracing() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := &responseWriterDecorator{
				buff: bytes.Buffer{},
				w:    w,
			}

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

			defer span.End()

			next.ServeHTTP(ww, r.WithContext(ctx))

			var eID int64
			eIDString := r.Header.Get(headerNameOpencensusSpanEventIDKey)
			if eIDString != "" {
				i, _ := strconv.ParseInt(eIDString, 10, 64)
				// TODO!!! handle error
				eID = i
			}

			span.AddMessageReceiveEvent(eID, ww.ContentSize(), 0)
			span.AddAttributes(trace.StringAttribute("response-payload", string(ww.Payload())))

			var responseBytes []byte
			if r.ContentLength > 0 {
				body, err := r.GetBody()
				if err == nil {
					responseBytes, _ = ioutil.ReadAll(body)
				}
			}

			span.AddAttributes(trace.StringAttribute("request-payload", string(responseBytes)))
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

type responseWriterDecorator struct {
	buff bytes.Buffer
	w    http.ResponseWriter
}

func (d *responseWriterDecorator) Header() http.Header {
	return d.w.Header()
}

func (d *responseWriterDecorator) Write(bytes []byte) (int, error) {
	_, _ = d.buff.Write(bytes)
	return d.w.Write(bytes)
}

func (d *responseWriterDecorator) WriteHeader(statusCode int) {
	d.w.WriteHeader(statusCode)
}

func (d *responseWriterDecorator) Payload() []byte {
	return d.buff.Bytes()
}

func (d *responseWriterDecorator) ContentSize() int64 {
	return int64(len(d.buff.Bytes()))
}
