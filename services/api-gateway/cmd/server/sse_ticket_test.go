package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sinergiinvestasiproperti/erp/services/api-gateway/internal/auth"
	"github.com/sinergiinvestasiproperti/erp/services/api-gateway/internal/proxy"
	"github.com/sinergiinvestasiproperti/erp/services/api-gateway/internal/sse"
)

// TestStreamEventsTicketPolicy guards the SSE auth contract: only short-lived
// stream tickets (or header/session credentials) may open the stream. A
// long-lived access JWT must never be accepted via the ?ticket= URL parameter.
func TestStreamEventsTicketPolicy(t *testing.T) {
	t.Setenv("GL_SERVICE_URL", "http://gl-service.internal:8080")
	router, err := proxy.NewRouter()
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	secret := []byte("test-gateway-secret")
	mux := newGatewayMux(router, sse.NewHub(), secret, true, false)

	now := time.Now().UTC()
	base := auth.Claims{
		Subject:  "user-1",
		OrgID:    "11111111-1111-1111-1111-111111111111",
		OrgScope: []string{"11111111-1111-1111-1111-111111111111"},
	}

	accessClaims := base
	accessClaims.TokenUse = "access"
	accessClaims.IssuedAt = now.Unix()
	accessClaims.ExpiresAt = now.Add(8 * time.Hour).Unix()
	accessTicket, err := auth.SignHS256(accessClaims, secret)
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}

	streamClaims := base
	streamClaims.TokenUse = "stream"
	streamClaims.IssuedAt = now.Unix()
	streamClaims.ExpiresAt = now.Add(60 * time.Second).Unix()
	streamTicket, err := auth.SignHS256(streamClaims, secret)
	if err != nil {
		t.Fatalf("sign stream ticket: %v", err)
	}

	// 1. No ticket and no credentials -> 401.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stream/events", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no-ticket status = %d, want 401", rec.Code)
	}

	// 2. Long-lived access JWT passed as ?ticket= must be rejected.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/stream/events?ticket="+accessTicket, nil).WithContext(ctx)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("access-token-as-ticket status = %d, want 401: long-lived JWTs must not be accepted in URL query parameters", rec.Code)
	}

	// 3. Genuine short-lived stream ticket still connects (200 until ctx ends).
	ctx2, cancel2 := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel2)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/stream/events?ticket="+streamTicket, nil).WithContext(ctx2)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	cancel2()
	if rec.Code != http.StatusOK {
		t.Fatalf("stream-ticket status = %d, want 200", rec.Code)
	}
}
