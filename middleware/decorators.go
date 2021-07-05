package middleware

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

type responseWriterDecorator struct {
	buff       bytes.Buffer
	statusCode int
	w          http.ResponseWriter
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
	body      io.Reader
}

func decorateRequestBody(r *http.Request) *requestBodyDecorator {
	d := &requestBodyDecorator{
		body: &bytes.Buffer{},
	}

	if r.ContentLength == 0 {
		return d
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return d
	}

	err = r.Body.Close()
	if err != nil {
		return d
	}

	d.bodyBytes = b
	d.body = bytes.NewReader(b)

	return d
}

func (d *requestBodyDecorator) Read(p []byte) (n int, err error) {
	return d.body.Read(p)
}

func (d *requestBodyDecorator) Close() error {
	return nil
}

func (d *requestBodyDecorator) Payload() []byte {
	return d.bodyBytes
}
