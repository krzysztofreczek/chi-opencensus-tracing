package middleware

import (
	"bytes"
	"io"
	"net/http"
)

type responseWriterDecorator struct {
	buff       bytes.Buffer
	statusCode int
	w          http.ResponseWriter
}

func (d *responseWriterDecorator) Flush() {
	if w, ok := d.w.(http.Flusher); ok {
		w.Flush()
	}
}

func decorateResponseWriter(w http.ResponseWriter) *responseWriterDecorator {
	return &responseWriterDecorator{
		buff: bytes.Buffer{},
		w:    w,
	}
}

func (d *responseWriterDecorator) Header() http.Header {
	return d.w.Header()
}

func (d *responseWriterDecorator) Write(bytes []byte) (int, error) {
	_, _ = d.buff.Write(bytes)
	return d.w.Write(bytes)
}

func (d *responseWriterDecorator) WriteHeader(statusCode int) {
	d.statusCode = statusCode
	d.w.WriteHeader(statusCode)
}

func (d *responseWriterDecorator) Payload() []byte {
	return d.buff.Bytes()
}

func (d *responseWriterDecorator) StatusCode() int {
	return d.statusCode
}

type requestBodyDecorator struct {
	bodyBytes []byte
	body      io.ReadCloser
}

func decorateRequestBody(r *http.Request) *requestBodyDecorator {
	if r.Body == nil {
		return nil
	}

	return &requestBodyDecorator{
		body: r.Body,
	}
}

func (d *requestBodyDecorator) Read(p []byte) (int, error) {
	n, err := d.body.Read(p)
	for i := 0; i < n; i++ {
		d.bodyBytes = append(d.bodyBytes, p[i])
	}
	return n, err
}

func (d *requestBodyDecorator) Close() error {
	return d.body.Close()
}

func (d *requestBodyDecorator) Payload() []byte {
	return d.bodyBytes
}
