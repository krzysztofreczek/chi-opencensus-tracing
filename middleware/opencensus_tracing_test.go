package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opencensus.io/trace"
)

func TestOpencensusTracing_open_span(t *testing.T) {
	exporter := registerTestExporter()

	req, _ := http.NewRequest("GET", "/test", nil)

	r := chi.NewRouter()
	r.Use(OpencensusTracing())

	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Test call received")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expectedNumberOfSpans := 1
	if len(exporter.collected) != expectedNumberOfSpans {
		t.Fatalf(
			"Expected to collect %d span(s), while there were %d span(s) collected",
			expectedNumberOfSpans,
			len(exporter.collected),
		)
	}

	spanData := exporter.collected[0]

	expectedSpanName := "[GET] /test"
	if spanData.Name != expectedSpanName {
		t.Fatalf(
			"Expected to collect a span of name '%s', while the actual name was '%s'",
			expectedSpanName,
			spanData.Name,
		)
	}

	if spanData.EndTime.IsZero() {
		t.Fatal("Expected the span to be closed")
	}
}

func TestOpencensusTracing_link_to_parent_span(t *testing.T) {
	exporter := registerTestExporter()

	req, _ := http.NewRequest("GET", "/test", nil)

	r := chi.NewRouter()
	r.Use(OpencensusTracing())

	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Test call received")
	})

	ctx, parent := trace.StartSpan(context.Background(), "parent span")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req.WithContext(ctx))

	parent.End()

	expectedNumberOfSpans := 2
	if len(exporter.collected) != expectedNumberOfSpans {
		t.Fatalf(
			"Expected to collect %d span(s), while there were %d span(s) collected",
			expectedNumberOfSpans,
			len(exporter.collected),
		)
	}

	spanData := exporter.collected[0]

	expectedSpanName := "[GET] /test"
	if spanData.Name != expectedSpanName {
		t.Fatalf(
			"Expected to collect a span of name '%s', while the actual name was '%s'",
			expectedSpanName,
			spanData.Name,
		)
	}

	spanParentData := exporter.collected[1]
	expectedSpanParentDataName := "parent span"
	if spanParentData.Name != expectedSpanParentDataName {
		t.Fatalf(
			"Expected to collect parent span of name '%s', while the actual name was '%s'",
			expectedSpanParentDataName,
			spanParentData.Name,
		)
	}
}

func TestOpencensusTracing_url_params_in_attributes(t *testing.T) {
	exporter := registerTestExporter()

	req, _ := http.NewRequest("GET", "/test/foo", nil)

	r := chi.NewRouter()
	r.Use(OpencensusTracing())

	r.Get("/test/{param_name}", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Test call received")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expectedNumberOfSpans := 1
	if len(exporter.collected) != expectedNumberOfSpans {
		t.Fatalf(
			"Expected to collect %d span(s), while there were %d span(s) collected",
			expectedNumberOfSpans,
			len(exporter.collected),
		)
	}

	spanData := exporter.collected[0]

	expectedSpanName := "[GET] /test/{param_name}"
	if spanData.Name != expectedSpanName {
		t.Fatalf(
			"Expected to collect a span of name '%s', while the actual name was '%s'",
			expectedSpanName,
			spanData.Name,
		)
	}

	expectedParameterName := "param_name"
	attribute, attributeSet := spanData.Attributes[expectedParameterName]
	if !attributeSet {
		t.Fatalf("Expected the span to have parameter attribute of name '%s' set", expectedParameterName)
	}

	expectedParameterAttribute := "foo"
	if attribute != expectedParameterAttribute {
		t.Fatalf("Expected the span attribute of name '%s' to have value '%s'", expectedParameterName, expectedParameterAttribute)
	}
}

func TestOpencensusTracing_payload_attributes(t *testing.T) {
	exporter := registerTestExporter()

	reqBody := []byte("REQUEST")
	req, _ := http.NewRequest("POST", "/test", bytes.NewReader(reqBody))

	r := chi.NewRouter()
	r.Use(OpencensusTracing())

	r.Post("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("RESPONSE"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expectedNumberOfSpans := 1
	if len(exporter.collected) != expectedNumberOfSpans {
		t.Fatalf(
			"Expected to collect %d span(s), while there were %d span(s) collected",
			expectedNumberOfSpans,
			len(exporter.collected),
		)
	}

	spanData := exporter.collected[0]

	expectedParameterName := "request_payload"
	attribute, attributeSet := spanData.Attributes[expectedParameterName]
	if !attributeSet {
		t.Fatalf("Expected the span to have parameter attribute of name '%s' set", expectedParameterName)
	}

	expectedParameterAttribute := "REQUEST"
	if attribute != expectedParameterAttribute {
		t.Fatalf("Expected the span attribute of name '%s' to have value '%s'", expectedParameterName, expectedParameterAttribute)
	}

	expectedParameterName = "response_payload"
	attribute, attributeSet = spanData.Attributes[expectedParameterName]
	if !attributeSet {
		t.Fatalf("Expected the span to have parameter attribute of name '%s' set", expectedParameterName)
	}

	expectedParameterAttribute = "RESPONSE"
	if attribute != expectedParameterAttribute {
		t.Fatalf("Expected the span attribute of name '%s' to have value '%s'", expectedParameterName, expectedParameterAttribute)
	}
}

func TestOpencensusTracing_payload_attributes_no_request_body_no_response_body(t *testing.T) {
	exporter := registerTestExporter()

	req, _ := http.NewRequest("GET", "/test", nil)

	r := chi.NewRouter()
	r.Use(OpencensusTracing())

	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Test call received")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expectedNumberOfSpans := 1
	if len(exporter.collected) != expectedNumberOfSpans {
		t.Fatalf(
			"Expected to collect %d span(s), while there were %d span(s) collected",
			expectedNumberOfSpans,
			len(exporter.collected),
		)
	}

	spanData := exporter.collected[0]

	expectedParameterName := "request_payload"
	attribute, attributeSet := spanData.Attributes[expectedParameterName]
	if !attributeSet {
		t.Fatalf("Expected the span to have parameter attribute of name '%s' set", expectedParameterName)
	}

	if attribute != "" {
		t.Fatalf("Expected the span attribute of name '%s' to have value '%s'", expectedParameterName, "")
	}

	expectedParameterName = "response_payload"
	_, attributeSet = spanData.Attributes[expectedParameterName]
	if !attributeSet {
		t.Fatalf("Expected the span to have parameter attribute of name '%s' set", expectedParameterName)
	}

	if attribute != "" {
		t.Fatalf("Expected the span attribute of name '%s' to have value '%s'", expectedParameterName, "")
	}
}

func TestOpencensusTracing_message_received_event_added(t *testing.T) {
	exporter := registerTestExporter()

	reqBody := []byte("REQUEST")
	req, _ := http.NewRequest("POST", "/test", bytes.NewReader(reqBody))
	req.Header.Set(headerNameOpencensusSpanEventIDKey, "100")

	r := chi.NewRouter()
	r.Use(OpencensusTracing())

	r.Post("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("RESPONSE"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expectedNumberOfSpans := 1
	if len(exporter.collected) != expectedNumberOfSpans {
		t.Fatalf(
			"Expected to collect %d span(s), while there were %d span(s) collected",
			expectedNumberOfSpans,
			len(exporter.collected),
		)
	}

	spanData := exporter.collected[0]

	expectedNumberOfMessageEvents := 1
	if len(spanData.MessageEvents) != expectedNumberOfSpans {
		t.Fatalf(
			"Expected to collect %d message event(s), while there were %d collected",
			expectedNumberOfMessageEvents,
			len(spanData.MessageEvents),
		)
	}

	messageEvent := spanData.MessageEvents[0]

	expectedMessageID := int64(100)
	if messageEvent.MessageID != expectedMessageID {
		t.Fatalf("Expected message ID to be set to '%d'", expectedMessageID)
	}

	if messageEvent.EventType != trace.MessageEventTypeRecv {
		t.Fatalf("Expected message type to be '%d'", trace.MessageEventTypeRecv)
	}

	if messageEvent.UncompressedByteSize != req.ContentLength {
		t.Fatalf("Expected message size to be '%d'", req.ContentLength)
	}

	if messageEvent.CompressedByteSize != 0 {
		t.Fatal("Expected message size to be '0'")
	}
}

func TestOpencensusTracing_message_sent_event_added(t *testing.T) {
	exporter := registerTestExporter()

	reqBody := []byte("REQUEST")
	req, _ := http.NewRequest("POST", "/test", bytes.NewReader(reqBody))

	ctx, span := trace.StartSpan(context.Background(), "testSpan")
	AddTracingSpanToRequest(ctx, req)
	span.End()

	expectedNumberOfSpans := 1
	if len(exporter.collected) != expectedNumberOfSpans {
		t.Fatalf(
			"Expected to collect %d span(s), while there were %d span(s) collected",
			expectedNumberOfSpans,
			len(exporter.collected),
		)
	}

	spanData := exporter.collected[0]

	expectedNumberOfMessageEvents := 1
	if len(spanData.MessageEvents) != expectedNumberOfSpans {
		t.Fatalf(
			"Expected to collect %d message event(s), while there were %d collected",
			expectedNumberOfMessageEvents,
			len(spanData.MessageEvents),
		)
	}

	messageEvent := spanData.MessageEvents[0]

	if messageEvent.MessageID == 0 {
		t.Fatal("Expected message ID to be set to random int64")
	}

	if messageEvent.EventType != trace.MessageEventTypeSent {
		t.Fatalf("Expected message type to be '%d'", trace.MessageEventTypeSent)
	}

	if messageEvent.UncompressedByteSize != req.ContentLength {
		t.Fatalf("Expected message size to be '%d'", req.ContentLength)
	}

	if messageEvent.CompressedByteSize != 0 {
		t.Fatal("Expected message size to be '0'")
	}
}

type exporterMock struct {
	collected []*trace.SpanData
}

func newExporterMock() *exporterMock {
	return &exporterMock{
		collected: make([]*trace.SpanData, 0),
	}
}

func (t *exporterMock) ExportSpan(s *trace.SpanData) {
	t.collected = append(t.collected, s)
}

func registerTestExporter() *exporterMock {
	exporter := newExporterMock()
	trace.RegisterExporter(exporter)
	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.ProbabilitySampler(1.0),
	})
	return exporter
}
