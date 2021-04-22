package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opencensus.io/trace"
)

func TestOpencensusTracing(t *testing.T) {
	exporter := registerTestExporter()

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(OpencensusTracing())

	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Test call received")
	})
	r.ServeHTTP(w, req)

	expectedNumberOfSpans := 1
	if len(exporter.collected) != expectedNumberOfSpans {
		t.Fatalf(
			"Expected to collect %d span, while there were %d spans collected",
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
