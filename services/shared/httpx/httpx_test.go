package httpx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestDecodeJSONRejectsTrailingDataAndUnknownFields(t *testing.T) {
	var dst struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}{"extra":1}`))
	rec := httptest.NewRecorder()
	if err := DecodeJSON(rec, req, &dst); err == nil {
		t.Fatal("expected trailing JSON error")
	}

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"unknown":1}`))
	rec = httptest.NewRecorder()
	if err := DecodeJSON(rec, req, &dst); err == nil {
		t.Fatal("expected unknown field error")
	}

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("   \n"))
	rec = httptest.NewRecorder()
	if err := DecodeOptionalJSON(rec, req, &dst); err != nil {
		t.Fatalf("expected nil on whitespace-only optional body, got %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}{"extra":1}`))
	rec = httptest.NewRecorder()
	if err := DecodeOptionalJSON(rec, req, &dst); err == nil {
		t.Fatal("expected trailing JSON error in DecodeOptionalJSON")
	}
}

func TestWriteErrorUnwrapsAPIErrorAndMasks5xx(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", Unprocessable("GL_JOURNAL_UNBALANCED", "debit != credit"))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	WriteError(rec, req, wrapped)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "GL_JOURNAL_UNBALANCED") {
		t.Fatalf("missing wrapped error code in %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	WriteError(rec, req, fmt.Errorf("secret db credentials leaked"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret db credentials") {
		t.Fatalf("5xx body leaked internal error: %s", rec.Body.String())
	}
}

func TestMapDBErrorSQLSTATE(t *testing.T) {
	rlsErr := &pgconn.PgError{Code: "42501", Message: "new row violates row-level security policy"}
	ae := MapDBError(rlsErr, "").(*APIError)
	if ae.Status != http.StatusNotFound {
		t.Fatalf("42501 mapped to %d, want 404", ae.Status)
	}

	dupErr := &pgconn.PgError{Code: "23505"}
	ae = MapDBError(dupErr, "duplicate journal").(*APIError)
	if ae.Status != http.StatusConflict {
		t.Fatalf("23505 mapped to %d, want 409", ae.Status)
	}
}

func TestIdemStoreNeverEvictsInFlightAndDetectsHashMismatch(t *testing.T) {
	s := &IdemStore{m: make(map[string]*idemEntry), ttl: time.Hour, max: 2}
	if res := s.ReserveWithHash("org1", "inflight", "h1"); res != ReserveAcquired {
		t.Fatalf("reserve inflight = %v", res)
	}
	_ = s.ReserveWithHash("org1", "done1", "h2")
	s.SaveWithHash("org1", "done1", "h2", http.StatusCreated, []byte(`{"id":"1"}`))
	// Inserting another entry hits max=2; evictLocked must evict done1, NOT inflight.
	if res := s.ReserveWithHash("org1", "k3", "h3"); res != ReserveAcquired {
		t.Fatalf("reserve k3 = %v", res)
	}
	if _, ok := s.m["org1|inflight"]; !ok {
		t.Fatal("in-flight key was evicted")
	}

	rec := httptest.NewRecorder()
	s.SaveWithHash("org1", "k3", "h3", http.StatusCreated, []byte(`{"id":"3"}`))
	if !s.ReplayWithHash(rec, "org1", "k3", "different-hash") {
		t.Fatal("expected ReplayWithHash to write 409 conflict on hash mismatch")
	}
	if rec.Code != http.StatusConflict {
		t.Fatalf("hash mismatch status = %d, want 409", rec.Code)
	}
}

func TestMiddlewareExtractsScopeRolesAndPermissions(t *testing.T) {
	orgA := uuid.NewString()
	sysOrg := "00000000-0000-0000-0000-000000000000"
	uid := uuid.NewString()

	var gotScope []uuid.UUID
	var hasGLApprove bool
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotScope = Scope(r.Context())
		hasGLApprove = HasPermission(r.Context(), "gl.approve")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/journal-entries", nil)
	req.Header.Set("X-Organization-Id", orgA+","+sysOrg)
	req.Header.Set("X-User-Id", uid)
	req.Header.Set("X-Roles", "FINANCE_MANAGER")
	req.Header.Set("X-Permissions", "gl.read,gl.approve")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if len(gotScope) != 2 || gotScope[0].String() != orgA || gotScope[1].String() != sysOrg {
		t.Fatalf("unexpected scope: %+v", gotScope)
	}
	if !hasGLApprove {
		t.Fatal("expected gl.approve permission to be true")
	}
}

func TestBeginIdempotentAndSaveIDResponse(t *testing.T) {
	idem := NewIdemStore()
	scope := []uuid.UUID{uuid.New()}

	type payload struct {
		Name string `json:"name"`
	}
	var p1 payload
	req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name":"alpha"}`))
	req.Header.Set("Idempotency-Key", "k-1")
	rec := httptest.NewRecorder()
	k, h, rel, done := BeginIdempotent(rec, req, idem, scope, &p1)
	if done || p1.Name != "alpha" {
		t.Fatalf("unexpected BeginIdempotent result done=%v name=%q", done, p1.Name)
	}
	SaveIDResponse(rec, idem, scope, k, h, http.StatusCreated, 1, map[string]string{"id": "1"})
	rel()
	if rec.Header().Get("ETag") != `"1"` {
		t.Fatalf("expected ETag \"1\", got %q", rec.Header().Get("ETag"))
	}

	// Same key + same payload replays cleanly.
	var p2 payload
	req2 := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name":"alpha"}`))
	req2.Header.Set("Idempotency-Key", "k-1")
	rec2 := httptest.NewRecorder()
	_, _, _, done2 := BeginIdempotent(rec2, req2, idem, scope, &p2)
	if !done2 || rec2.Code != http.StatusCreated || rec2.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatalf("expected replay 201, got done=%v status=%d", done2, rec2.Code)
	}

	// Same key + different payload rejects with 409 conflict.
	var p3 payload
	req3 := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name":"beta"}`))
	req3.Header.Set("Idempotency-Key", "k-1")
	rec3 := httptest.NewRecorder()
	_, _, _, done3 := BeginIdempotent(rec3, req3, idem, scope, &p3)
	if !done3 || rec3.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got done=%v status=%d", done3, rec3.Code)
	}
}

func TestRequirePermissionEnforceRBAC(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if err := RequirePermission(req, "gl.read"); err != nil {
		t.Fatalf("expected nil when REQUIRE_RBAC unset and no headers, got %v", err)
	}

	t.Setenv("REQUIRE_RBAC", "true")
	if err := RequirePermission(req, "gl.read"); err == nil {
		t.Fatal("expected forbidden error when REQUIRE_RBAC=true and headers absent")
	}
}

func TestMetricsStatusWriterSupportsFlusherAndUnwrap(t *testing.T) {
	rec := httptest.NewRecorder()
	m := NewMetrics("test-svc")
	var flushed bool
	h := m.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Fatalf("Flush via ResponseController failed: %v", err)
		}
		flushed = rec.Flushed
	}))
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/stream", nil))
	if !flushed || rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected flushed=%v code=%d", flushed, rec.Code)
	}
}

func TestBeginIdempotentScopesByRouteAndReplaysETag(t *testing.T) {
	idem := NewIdemStore()
	scope := []uuid.UUID{uuid.New()}

	type payload struct {
		Name string `json:"name"`
	}
	var p1 payload
	req1 := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name":"alpha"}`))
	req1.Header.Set("Idempotency-Key", "shared-key")
	rec1 := httptest.NewRecorder()
	k1, h1, rel1, done1 := BeginIdempotent(rec1, req1, idem, scope, &p1)
	if done1 {
		t.Fatal("expected first /items request not to replay")
	}
	SaveIDResponse(rec1, idem, scope, k1, h1, http.StatusCreated, 3, map[string]string{"id": "1"})
	rel1()

	// Replay on /items returns ETag "3".
	var p1Replay payload
	req1Replay := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name":"alpha"}`))
	req1Replay.Header.Set("Idempotency-Key", "shared-key")
	rec1Replay := httptest.NewRecorder()
	_, _, _, doneReplay := BeginIdempotent(rec1Replay, req1Replay, idem, scope, &p1Replay)
	if !doneReplay || rec1Replay.Header().Get("ETag") != `"3"` {
		t.Fatalf("expected replay with ETag \"3\", got done=%v etag=%q", doneReplay, rec1Replay.Header().Get("ETag"))
	}

	// Same Idempotency-Key on a DIFFERENT route (/parties) must NOT collide with /items.
	var p2 payload
	req2 := httptest.NewRequest(http.MethodPost, "/parties", strings.NewReader(`{"name":"beta"}`))
	req2.Header.Set("Idempotency-Key", "shared-key")
	rec2 := httptest.NewRecorder()
	_, _, rel2, done2 := BeginIdempotent(rec2, req2, idem, scope, &p2)
	defer rel2()
	if done2 || p2.Name != "beta" {
		t.Fatalf("expected /parties with same Idempotency-Key not to collide with /items, got done=%v", done2)
	}
}

func TestGatewayHMACAndSystemOrgMutationGuard(t *testing.T) {
	t.Setenv("REQUIRE_GATEWAY_SIGNATURE", "true")
	t.Setenv("GATEWAY_HMAC_SECRET", "unit-test-hmac-secret-32-bytes-ok")

	tenantID := uuid.New()
	var capturedScope []uuid.UUID
	var scopeErr error

	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedScope, scopeErr = RequireScope(r)
		if scopeErr != nil {
			WriteError(w, r, scopeErr)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	// 1. Unsigned request is rejected when REQUIRE_GATEWAY_SIGNATURE=true.
	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	req.Header.Set("X-Organization-Id", tenantID.String())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on unsigned request, got %d", rec.Code)
	}

	// 2. Signed request with tenant + systemOrg strips systemOrg on POST.
	req = httptest.NewRequest(http.MethodPost, "/items", nil)
	req.Header.Set("X-Organization-Id", tenantID.String()+",00000000-0000-0000-0000-000000000000")
	req.Header.Set("X-Org-Scope", tenantID.String()+",00000000-0000-0000-0000-000000000000")
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Roles", "FINANCE_MANAGER")
	SignGatewayRequest(req, []byte("unit-test-hmac-secret-32-bytes-ok"))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || len(capturedScope) != 1 || capturedScope[0] != tenantID {
		t.Fatalf("expected tenant-only scope [%s], got code=%d scope=%v", tenantID, rec.Code, capturedScope)
	}

	// 3. Signed POST with systemOrg as primary active org is rejected with 403.
	req = httptest.NewRequest(http.MethodPost, "/items", nil)
	req.Header.Set("X-Organization-Id", "00000000-0000-0000-0000-000000000000")
	SignGatewayRequest(req, []byte("unit-test-hmac-secret-32-bytes-ok"))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when systemOrg is active org on POST, got %d", rec.Code)
	}
}

func TestTieredAdminDelegationMatrix(t *testing.T) {
	if !CanDelegateTier(AdminTierSystem, AdminTierHolding) {
		t.Fatal("SYSTEM_ADMIN must be able to delegate HOLDING_ADMIN")
	}
	if !CanDelegateTier(AdminTierHolding, AdminTierSubsidiary) {
		t.Fatal("HOLDING_ADMIN must be able to delegate SUBSIDIARY_ADMIN")
	}
	if CanDelegateTier(AdminTierHolding, AdminTierSystem) || CanDelegateTier(AdminTierHolding, AdminTierHolding) {
		t.Fatal("HOLDING_ADMIN must NOT be able to delegate SYSTEM_ADMIN or peer HOLDING_ADMIN")
	}
	if !CanDelegateTier(AdminTierSubsidiary, AdminTierBranch) || !CanDelegateTier(AdminTierSubsidiary, AdminTierNone) {
		t.Fatal("SUBSIDIARY_ADMIN must be able to delegate BRANCH_ADMIN and functional roles")
	}
	if CanDelegateTier(AdminTierSubsidiary, AdminTierHolding) || CanDelegateTier(AdminTierSubsidiary, AdminTierSubsidiary) {
		t.Fatal("SUBSIDIARY_ADMIN must NOT be able to delegate HOLDING_ADMIN or peer SUBSIDIARY_ADMIN")
	}
	if !CanDelegateTier(AdminTierBranch, AdminTierNone) || CanDelegateTier(AdminTierBranch, AdminTierBranch) {
		t.Fatal("BRANCH_ADMIN must only delegate functional roles")
	}
	if CanDelegateTier(AdminTierNone, AdminTierNone) {
		t.Fatal("non-admin must not delegate roles")
	}
}
