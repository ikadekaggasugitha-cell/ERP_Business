package telemetry_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sinergiinvestasiproperti/erp/services/shared/telemetry"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestParseAndFormatTraceparent(t *testing.T) {
	t.Parallel()
	valid := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	tc, ok := telemetry.ParseTraceparent(valid)
	if !ok {
		t.Fatalf("expected valid traceparent, got ok=false")
	}
	if tc.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" || tc.SpanID != "00f067aa0ba902b7" || !tc.Sampled() {
		t.Fatalf("unexpected parsed trace context: %+v", tc)
	}
	if got := telemetry.FormatTraceparent(tc); got != valid {
		t.Fatalf("roundtrip mismatch: got %q want %q", got, valid)
	}

	invalids := []string{
		"",
		"not-a-header",
		"ff-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"00-00000000000000000000000000000000-00f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01",
		"00-short-00f067aa0ba902b7-01",
	}
	for _, raw := range invalids {
		if _, ok := telemetry.ParseTraceparent(raw); ok {
			t.Fatalf("expected invalid traceparent %q to fail", raw)
		}
	}
}

func TestExtractOrGenerateAndInject(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set(telemetry.HeaderTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	h.Set(telemetry.HeaderTracestate, "erp=1")

	parent, child := telemetry.ExtractOrGenerate(h)
	if !parent.Valid() || !child.Valid() {
		t.Fatalf("expected valid parent and child: parent=%+v child=%+v", parent, child)
	}
	if child.TraceID != parent.TraceID {
		t.Fatalf("child traceID %s != parent traceID %s", child.TraceID, parent.TraceID)
	}
	if child.SpanID == parent.SpanID {
		t.Fatal("child spanID must differ from parent spanID")
	}

	ctx := telemetry.WithTraceContext(context.Background(), child)
	out := http.Header{}
	telemetry.Inject(ctx, out)
	if out.Get(telemetry.HeaderTraceparent) != child.String() {
		t.Fatalf("unexpected injected traceparent: %q", out.Get(telemetry.HeaderTraceparent))
	}
	if out.Get(telemetry.HeaderTracestate) != "erp=1" {
		t.Fatalf("unexpected injected tracestate: %q", out.Get(telemetry.HeaderTracestate))
	}
}

func TestOTLPExporterFlushesOnShutdown(t *testing.T) {
	t.Parallel()
	var (
		mu       sync.Mutex
		received []map[string]any
		gotPath  string
	)
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			mu.Lock()
			defer mu.Unlock()
			gotPath = r.URL.Path
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode otlp payload: %v", err)
			} else {
				received = append(received, body)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("{}")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	tr := telemetry.NewTracerWithClient("gl-service", "http://otel-collector:4318", client)
	if !tr.Enabled() {
		t.Fatal("expected tracer to be enabled")
	}
	_, child := telemetry.ExtractOrGenerate(http.Header{})
	ctx := telemetry.WithTraceContext(context.Background(), child)
	_, sp := tr.StartSpan(ctx, "GET /journal-entries", 2, "")
	sp.SetAttribute("http.method", "GET")
	sp.SetStatus(telemetry.StatusOK, "")
	sp.End()

	shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := tr.Shutdown(shutCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotPath != "/v1/traces" {
		t.Fatalf("expected /v1/traces, got %q", gotPath)
	}
	if len(received) != 1 {
		t.Fatalf("expected 1 exported batch, got %d", len(received))
	}
}

func TestNoopTracerWhenEndpointUnset(t *testing.T) {
	t.Parallel()
	tr := telemetry.NewTracer("gl-service", "")
	if tr.Enabled() {
		t.Fatal("expected disabled tracer when endpoint is empty")
	}
	_, sp := tr.StartSpan(context.Background(), "noop", 1, "")
	if sp != nil {
		t.Fatal("expected nil span for disabled tracer")
	}
	if err := tr.Shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected error on noop shutdown: %v", err)
	}
}
