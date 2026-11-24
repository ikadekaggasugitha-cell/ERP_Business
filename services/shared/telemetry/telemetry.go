// Package telemetry implements W3C Trace Context (traceparent/tracestate)
// propagation and an asynchronous, zero-dependency OTLP/HTTP JSON span exporter.
// When OTEL_EXPORTER_OTLP_ENDPOINT is unset, span export is a zero-overhead no-op
// while W3C traceparent propagation remains active across HTTP boundaries.
package telemetry

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// HeaderTraceparent is the W3C Trace Context header name.
	HeaderTraceparent = "traceparent"
	// HeaderTracestate is the W3C Trace State header name.
	HeaderTracestate = "tracestate"

	// StatusUnset, StatusOK, StatusError match OTLP StatusCode values.
	StatusUnset = 0
	StatusOK    = 1
	StatusError = 2
)

// TraceContext holds W3C trace propagation identifiers.
type TraceContext struct {
	TraceID    string
	SpanID     string
	Flags      uint8
	TraceState string
	Remote     bool
}

// Valid reports whether TraceID (32 hex) and SpanID (16 hex) are non-empty and non-zero.
func (tc TraceContext) Valid() bool {
	return isValidHexID(tc.TraceID, 32) && isValidHexID(tc.SpanID, 16)
}

// Sampled reports whether the W3C sampled flag (0x01) is set.
func (tc TraceContext) Sampled() bool {
	return (tc.Flags & 0x01) != 0
}

// String formats tc as a W3C traceparent header value.
func (tc TraceContext) String() string {
	return FormatTraceparent(tc)
}

// ParseTraceparent parses a W3C traceparent header ("00-<32hex>-<16hex>-<2hex>").
func ParseTraceparent(raw string) (TraceContext, bool) {
	parts := strings.Split(strings.TrimSpace(raw), "-")
	if len(parts) < 4 {
		return TraceContext{}, false
	}
	version := strings.ToLower(parts[0])
	if len(version) != 2 || version == "ff" {
		return TraceContext{}, false
	}
	if _, err := hex.DecodeString(version); err != nil {
		return TraceContext{}, false
	}
	traceID := strings.ToLower(parts[1])
	spanID := strings.ToLower(parts[2])
	flagsHex := strings.ToLower(parts[3])
	if !isValidHexID(traceID, 32) || !isValidHexID(spanID, 16) || len(flagsHex) != 2 {
		return TraceContext{}, false
	}
	flagsBytes, err := hex.DecodeString(flagsHex)
	if err != nil || len(flagsBytes) != 1 {
		return TraceContext{}, false
	}
	return TraceContext{
		TraceID: traceID,
		SpanID:  spanID,
		Flags:   flagsBytes[0],
		Remote:  true,
	}, true
}

// FormatTraceparent encodes tc into a W3C traceparent header string.
func FormatTraceparent(tc TraceContext) string {
	return fmt.Sprintf("00-%s-%s-%02x", tc.TraceID, tc.SpanID, tc.Flags)
}

// NewTraceID generates a random 16-byte (32 lower hex) W3C Trace ID.
func NewTraceID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	// Guard against vanishingly unlikely all-zero ID.
	if isAllZero(b[:]) {
		b[15] = 1
	}
	return hex.EncodeToString(b[:])
}

// NewSpanID generates a random 8-byte (16 lower hex) W3C Span ID.
func NewSpanID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	if isAllZero(b[:]) {
		b[7] = 1
	}
	return hex.EncodeToString(b[:])
}

// ExtractOrGenerate parses W3C traceparent/tracestate from HTTP headers or
// generates a fresh root TraceContext if absent/malformed.
func ExtractOrGenerate(h http.Header) (parent TraceContext, child TraceContext) {
	if h != nil {
		if tc, ok := ParseTraceparent(h.Get(HeaderTraceparent)); ok {
			tc.TraceState = strings.TrimSpace(h.Get(HeaderTracestate))
			child = TraceContext{
				TraceID:    tc.TraceID,
				SpanID:     NewSpanID(),
				Flags:      tc.Flags,
				TraceState: tc.TraceState,
				Remote:     false,
			}
			return tc, child
		}
	}
	child = TraceContext{
		TraceID: NewTraceID(),
		SpanID:  NewSpanID(),
		Flags:   0x01,
		Remote:  false,
	}
	return TraceContext{}, child
}

// Inject writes W3C traceparent (and tracestate if present) onto outbound headers.
func Inject(ctx context.Context, h http.Header) {
	if h == nil {
		return
	}
	tc, ok := FromContext(ctx)
	if !ok || !tc.Valid() {
		return
	}
	h.Set(HeaderTraceparent, FormatTraceparent(tc))
	if tc.TraceState != "" {
		h.Set(HeaderTracestate, tc.TraceState)
	}
}

type ctxKey struct{}

// WithTraceContext stores tc on ctx.
func WithTraceContext(ctx context.Context, tc TraceContext) context.Context {
	return context.WithValue(ctx, ctxKey{}, tc)
}

// FromContext retrieves the active TraceContext from ctx.
func FromContext(ctx context.Context) (TraceContext, bool) {
	if ctx == nil {
		return TraceContext{}, false
	}
	tc, ok := ctx.Value(ctxKey{}).(TraceContext)
	return tc, ok && tc.Valid()
}

// TraceID returns the active W3C traceId from ctx or empty string if none.
func TraceID(ctx context.Context) string {
	if tc, ok := FromContext(ctx); ok {
		return tc.TraceID
	}
	return ""
}

// ---- Span & Async OTLP/HTTP JSON Exporter ----------------------------------

type spanRecord struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Name         string
	Kind         int // 1=INTERNAL, 2=SERVER, 3=CLIENT
	StartNano    int64
	EndNano      int64
	Attributes   map[string]string
	StatusCode   int
	StatusMsg    string
}

// Span represents an active OpenTelemetry span.
type Span struct {
	tracer *Tracer
	rec    spanRecord
	mu     sync.Mutex
	ended  bool
}

// SetAttribute records a string attribute on the span.
func (s *Span) SetAttribute(k, v string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return
	}
	if s.rec.Attributes == nil {
		s.rec.Attributes = make(map[string]string, 4)
	}
	s.rec.Attributes[k] = v
}

// SetStatus sets the span status code (StatusUnset, StatusOK, StatusError) and message.
func (s *Span) SetStatus(code int, msg string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return
	}
	s.rec.StatusCode = code
	s.rec.StatusMsg = msg
}

// End finalizes the span and enqueues it for async OTLP export if enabled.
func (s *Span) End() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return
	}
	s.ended = true
	s.rec.EndNano = time.Now().UTC().UnixNano()
	rec := s.rec
	s.mu.Unlock()

	if s.tracer != nil {
		s.tracer.enqueue(rec)
	}
}

// Tracer exports spans asynchronously to an OTLP/HTTP JSON collector.
type Tracer struct {
	serviceName string
	endpoint    string
	client      *http.Client
	ch          chan spanRecord
	stopCh      chan struct{}
	doneCh      chan struct{}
	closed      atomic.Bool
	flushEvery  time.Duration
	batchSize   int
}

var defaultTracer atomic.Pointer[Tracer]

// Default returns the process-wide default Tracer (may be nil / no-op).
func Default() *Tracer {
	return defaultTracer.Load()
}

// InitFromEnv configures and installs the default Tracer from environment variables:
//   - OTEL_EXPORTER_OTLP_ENDPOINT (e.g. http://otel-collector:4318)
//   - OTEL_SERVICE_NAME (overrides serviceName if set)
//
// When OTEL_EXPORTER_OTLP_ENDPOINT is unset, returns a no-op Tracer.
func InitFromEnv(serviceName string) *Tracer {
	if envName := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")); envName != "" {
		serviceName = envName
	}
	if serviceName == "" {
		serviceName = "erp-service"
	}
	ep := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	tr := NewTracer(serviceName, ep)
	defaultTracer.Store(tr)
	return tr
}

// NewTracer constructs a Tracer targeting endpoint (empty string = zero-overhead no-op).
func NewTracer(serviceName, endpoint string) *Tracer {
	return NewTracerWithClient(serviceName, endpoint, &http.Client{Timeout: 3 * time.Second})
}

// NewTracerWithClient constructs a Tracer with a custom HTTP client (useful for tests/mTLS).
func NewTracerWithClient(serviceName, endpoint string, client *http.Client) *Tracer {
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	ep := normalizeOTLPEndpoint(endpoint)
	t := &Tracer{
		serviceName: serviceName,
		endpoint:    ep,
		client:      client,
		flushEvery:  2 * time.Second,
		batchSize:   64,
	}
	if ep == "" {
		return t
	}
	t.ch = make(chan spanRecord, 1024)
	t.stopCh = make(chan struct{})
	t.doneCh = make(chan struct{})
	go t.loop()
	return t
}

// Enabled reports whether OTLP export is configured.
func (t *Tracer) Enabled() bool {
	return t != nil && t.endpoint != ""
}

// StartSpan starts a child span under ctx (using the default tracer if t is nil).
func (t *Tracer) StartSpan(ctx context.Context, name string, kind int, parentSpanID string) (context.Context, *Span) {
	if t == nil {
		t = Default()
	}
	tc, ok := FromContext(ctx)
	if !ok {
		tc = TraceContext{
			TraceID: NewTraceID(),
			SpanID:  NewSpanID(),
			Flags:   0x01,
		}
		ctx = WithTraceContext(ctx, tc)
	}
	if !t.Enabled() {
		return ctx, nil
	}
	sp := &Span{
		tracer: t,
		rec: spanRecord{
			TraceID:      tc.TraceID,
			SpanID:       tc.SpanID,
			ParentSpanID: parentSpanID,
			Name:         name,
			Kind:         kind,
			StartNano:    time.Now().UTC().UnixNano(),
		},
	}
	return ctx, sp
}

// StartSpan starts a span on the default Tracer.
func StartSpan(ctx context.Context, name string, kind int, parentSpanID string) (context.Context, *Span) {
	return Default().StartSpan(ctx, name, kind, parentSpanID)
}

func (t *Tracer) enqueue(rec spanRecord) {
	if !t.Enabled() || t.closed.Load() {
		return
	}
	select {
	case t.ch <- rec:
	default:
		// Non-blocking drop under backpressure so tracing never degrades request latency.
	}
}

func (t *Tracer) loop() {
	defer close(t.doneCh)
	ticker := time.NewTicker(t.flushEvery)
	defer ticker.Stop()
	batch := make([]spanRecord, 0, t.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		t.exportBatch(batch)
		batch = batch[:0]
	}

	for {
		select {
		case rec := <-t.ch:
			batch = append(batch, rec)
			if len(batch) >= t.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-t.stopCh:
			for {
				select {
				case rec := <-t.ch:
					batch = append(batch, rec)
					if len(batch) >= t.batchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}

// Shutdown flushes queued spans and stops the background exporter goroutine.
func (t *Tracer) Shutdown(ctx context.Context) error {
	if !t.Enabled() {
		return nil
	}
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(t.stopCh)
	select {
	case <-t.doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// OTLP/HTTP JSON wire structures (https://opentelemetry.io/docs/specs/otlp/#json-protobuf-encoding).
type otlpExportRequest struct {
	ResourceSpans []otlpResourceSpans `json:"resourceSpans"`
}

type otlpResourceSpans struct {
	Resource   otlpResource     `json:"resource"`
	ScopeSpans []otlpScopeSpans `json:"scopeSpans"`
}

type otlpResource struct {
	Attributes []otlpKeyValue `json:"attributes"`
}

type otlpScopeSpans struct {
	Scope otlpScope  `json:"scope"`
	Spans []otlpSpan `json:"spans"`
}

type otlpScope struct {
	Name string `json:"name"`
}

type otlpSpan struct {
	TraceID           string         `json:"traceId"`
	SpanID            string         `json:"spanId"`
	ParentSpanID      string         `json:"parentSpanId,omitempty"`
	Name              string         `json:"name"`
	Kind              int            `json:"kind"`
	StartTimeUnixNano string         `json:"startTimeUnixNano"`
	EndTimeUnixNano   string         `json:"endTimeUnixNano"`
	Attributes        []otlpKeyValue `json:"attributes,omitempty"`
	Status            otlpStatus     `json:"status"`
}

type otlpKeyValue struct {
	Key   string       `json:"key"`
	Value otlpAnyValue `json:"value"`
}

type otlpAnyValue struct {
	StringValue string `json:"stringValue"`
}

type otlpStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

func (t *Tracer) exportBatch(batch []spanRecord) {
	spans := make([]otlpSpan, 0, len(batch))
	for _, r := range batch {
		attrs := make([]otlpKeyValue, 0, len(r.Attributes))
		for k, v := range r.Attributes {
			attrs = append(attrs, otlpKeyValue{Key: k, Value: otlpAnyValue{StringValue: v}})
		}
		spans = append(spans, otlpSpan{
			TraceID:           r.TraceID,
			SpanID:            r.SpanID,
			ParentSpanID:      r.ParentSpanID,
			Name:              r.Name,
			Kind:              r.Kind,
			StartTimeUnixNano: strconv.FormatInt(r.StartNano, 10),
			EndTimeUnixNano:   strconv.FormatInt(r.EndNano, 10),
			Attributes:        attrs,
			Status:            otlpStatus{Code: r.StatusCode, Message: r.StatusMsg},
		})
	}
	payload := otlpExportRequest{
		ResourceSpans: []otlpResourceSpans{{
			Resource: otlpResource{
				Attributes: []otlpKeyValue{{
					Key:   "service.name",
					Value: otlpAnyValue{StringValue: t.serviceName},
				}},
			},
			ScopeSpans: []otlpScopeSpans{{
				Scope: otlpScope{Name: "erp.shared.telemetry"},
				Spans: spans,
			}},
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, t.endpoint, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

func normalizeOTLPEndpoint(ep string) string {
	ep = strings.TrimSpace(ep)
	if ep == "" {
		return ""
	}
	ep = strings.TrimRight(ep, "/")
	if !strings.HasSuffix(ep, "/v1/traces") {
		ep += "/v1/traces"
	}
	return ep
}

func isValidHexID(s string, expectedLen int) bool {
	if len(s) != expectedLen {
		return false
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return false
	}
	return !isAllZero(b)
}

func isAllZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}
