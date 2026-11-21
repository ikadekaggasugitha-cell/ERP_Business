package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

// CorrelationUUID converts the request Correlation ID into a UUID for DB storage.
func CorrelationUUID(r *http.Request) uuid.UUID {
	if u, err := uuid.Parse(Correlation(r.Context())); err == nil {
		return u
	}
	return uuid.New()
}

// ParseUUIDParam parses a required path/query UUID or returns a structured 400 error.
func ParseUUIDParam(field, raw string) (uuid.UUID, error) {
	u, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, BadRequest(field + " tidak valid", Detail{Field: field, Issue: "must be valid UUID"})
	}
	return u, nil
}

// DecodeJSONIdempotent reads the raw body once so the caller can hash it for
// idempotency payload-conflict checks while decoding strict JSON via DecodeJSON.
func DecodeJSONIdempotent(w http.ResponseWriter, r *http.Request, v any) (reqHash string, err error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return "", &APIError{Status: http.StatusRequestEntityTooLarge, Code: "PAYLOAD_TOO_LARGE", Message: "request body exceeds 1 MiB limit"}
		}
		return "", BadRequest("gagal membaca body JSON")
	}
	reqHash = HashBody(raw)
	r.Body = io.NopCloser(bytes.NewReader(raw))
	if err := DecodeJSON(w, r, v); err != nil {
		return "", err
	}
	return reqHash, nil
}

// WriteIDResponse writes a JSON payload with an ETag (rowVersion) header.
func WriteIDResponse(w http.ResponseWriter, status int, version int32, v any) {
	if version > 0 {
		w.Header().Set("ETag", strconvQuoteVersion(version))
	}
	WriteJSON(w, status, v)
}

func strconvQuoteVersion(v int32) string {
	return `"` + itoa32(v) + `"`
}

func itoa32(v int32) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ServeHTTP starts the HTTP server, waits for SIGINT/SIGTERM, drains HTTP first,
// then cancels background workers and waits for wg before returning.
func ServeHTTP(ctx context.Context, cancel context.CancelFunc, service, addr string, handler http.Handler, wg *sync.WaitGroup, cleanup ...func()) {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("%s listening on %s", service, addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-stop:
	case err := <-errCh:
		log.Fatalf("%s http error: %v", service, err)
	}

	// 1. Drain in-flight HTTP requests before cancelling background workers/DB.
	sc, scCancel := context.WithTimeout(context.Background(), 10*time.Second)
	_ = srv.Shutdown(sc)
	scCancel()

	// 2. Stop relay & consumers cleanly.
	cancel()
	if wg != nil {
		wg.Wait()
	}
	for _, fn := range cleanup {
		if fn != nil {
			fn()
		}
	}
	log.Printf("%s stopped cleanly", service)
}

// Getenv returns os.Getenv(k) if non-empty, or fallback.
func Getenv(k, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return fallback
}

// SplitCSV splits comma-separated configuration strings such as KAFKA_BROKERS.
func SplitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ScopeKey formats an org-scope slice as a deterministic idempotency partition key.
func ScopeKey(scope []uuid.UUID) string {
	parts := make([]string, len(scope))
	for i, u := range scope {
		parts[i] = u.String()
	}
	return strings.Join(parts, ",")
}

// BeginIdempotent decodes dst via DecodeJSONIdempotent and guards against duplicate
// or conflicting Idempotency-Key mutations. Keys are scoped by HTTP method + path
// to prevent cross-endpoint collisions. When done is false, callers should defer release().
func BeginIdempotent(w http.ResponseWriter, r *http.Request, idem *IdemStore, scope []uuid.UUID, dst any) (idemKey, reqHash string, release func(), done bool) {
	reqHash, err := DecodeJSONIdempotent(w, r, dst)
	if err != nil {
		WriteError(w, r, err)
		return "", "", func() {}, true
	}
	rawKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if rawKey != "" {
		route := strings.TrimSpace(r.Pattern)
		if route == "" {
			route = r.Method + " " + r.URL.Path
		}
		idemKey = route + "|" + rawKey
	}
	sk := ScopeKey(scope)
	if idem != nil {
		if idem.ReplayWithHash(w, sk, idemKey, reqHash) {
			return idemKey, reqHash, func() {}, true
		}
		switch idem.ReserveWithHash(sk, idemKey, reqHash) {
		case ReserveCompleted:
			_ = idem.ReplayWithHash(w, sk, idemKey, reqHash)
			return idemKey, reqHash, func() {}, true
		case ReserveHashMismatch:
			WriteError(w, r, Conflict("IDEMPOTENCY_KEY_REUSED", "Idempotency-Key reused with different request payload"))
			return idemKey, reqHash, func() {}, true
		case ReserveInFlight:
			WriteError(w, r, Conflict("REQUEST_IN_PROGRESS", "Idempotency-Key sedang diproses"))
			return idemKey, reqHash, func() {}, true
		default:
			return idemKey, reqHash, func() { idem.Unreserve(sk, idemKey) }, false
		}
	}
	return idemKey, reqHash, func() {}, false
}

// SaveIDResponse marshals v, records it in idem (if key != ""), sets ETag when version > 0, and writes JSON.
func SaveIDResponse(w http.ResponseWriter, idem *IdemStore, scope []uuid.UUID, key, reqHash string, status int, version int32, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		cid := w.Header().Get("X-Correlation-Id")
		log.Printf("[error] cid=%s code=INTERNAL_ERROR status=500: encode json failed: %v", cid, err)
		WriteJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]any{
				"code":          "INTERNAL_ERROR",
				"message":       "internal server error",
				"correlationId": cid,
				"details":       nil,
			},
		})
		return
	}
	etag := ""
	if version > 0 {
		etag = strconvQuoteVersion(version)
		w.Header().Set("ETag", etag)
	}
	if idem != nil && key != "" {
		idem.SaveWithHashAndETag(ScopeKey(scope), key, reqHash, etag, status, body)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
