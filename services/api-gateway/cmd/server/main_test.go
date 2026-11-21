package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sinergiinvestasiproperti/erp/services/api-gateway/internal/auth"
	"github.com/sinergiinvestasiproperti/erp/services/api-gateway/internal/proxy"
	"github.com/sinergiinvestasiproperti/erp/services/api-gateway/internal/sse"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestGatewayAuthAndHeaderForwarding(t *testing.T) {
	var seenUserID, seenRoles, seenPerms, seenOrgHeader, seenOrgScope, seenPath string

	t.Setenv("GL_SERVICE_URL", "http://gl-service.internal:8080")
	router, err := proxy.NewRouter()
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	rp, ok := router.Match("/api/v1/journal-entries")
	if !ok {
		t.Fatal("expected /api/v1/journal-entries route to match")
	}
	rp.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		seenUserID = r.Header.Get("X-User-Id")
		seenRoles = r.Header.Get("X-Roles")
		seenPerms = r.Header.Get("X-Permissions")
		seenOrgHeader = r.Header.Get("X-Organization-Id")
		seenOrgScope = r.Header.Get("X-Org-Scope")
		seenPath = r.URL.Path
		rec := httptest.NewRecorder()
		rec.Header().Set("Content-Type", "application/json")
		rec.WriteHeader(http.StatusOK)
		_, _ = rec.Write([]byte(`{"data":[]}`))
		return rec.Result(), nil
	})
	secret := []byte("test-gateway-secret")
	mux := newGatewayMux(router, sse.NewHub(), secret, true, false)

	// 1. Unauthenticated request is rejected with 401.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/journal-entries", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", rec.Code)
	}

	// 2. Login returns access token + streamTicket + org list.
	loginBody := bytes.NewBufferString(`{"username":"admin.finance","password":"dev","orgCode":"HLD-01"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", loginBody)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var loginResp struct {
		Token        string `json:"token"`
		StreamTicket string `json:"streamTicket"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("unmarshal login response: %v", err)
	}
	if loginResp.Token == "" || loginResp.StreamTicket == "" {
		t.Fatalf("missing tokens in login response: %+v", loginResp)
	}

	// 3. Authenticated request strips spoofed privileged headers and injects JWT claims.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/journal-entries", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	req.Header.Set("X-User-Id", "99999999-9999-9999-9999-999999999999")
	req.Header.Set("X-Roles", "SPOOFED")
	req.Header.Set("X-Organization-Id", branchOrgID)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("proxied status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if seenUserID != defaultUID {
		t.Fatalf("X-User-Id = %q, want %q (spoofed header not stripped)", seenUserID, defaultUID)
	}
	if seenPath != "/journal-entries" {
		t.Fatalf("forwarded URL.Path = %q, want /journal-entries", seenPath)
	}
	if strings.Contains(seenRoles, "SPOOFED") {
		t.Fatalf("spoofed role leaked through: %q", seenRoles)
	}
	if !strings.HasPrefix(seenOrgHeader, branchOrgID+",") || !strings.Contains(seenOrgScope, systemOrgID) {
		t.Fatalf("unexpected forwarded org scope header=%q scope=%q", seenOrgHeader, seenOrgScope)
	}
	for _, wantPerm := range []string{
		"gl.create", "gl.read", "reporting.read",
		"ap:read", "ap:write", "ar:read", "ar:write",
		"coa:read", "coa:write", "tax:read", "tax:write",
		"currency:read", "currency:write", "treasury:read", "treasury:write",
		"workflow:read", "workflow:write", "workflow:approve",
		"inventory:read", "inventory:write", "procurement:read", "procurement:write",
		"sales:read", "sales:write", "manufacturing:read", "manufacturing:write",
		"hr:read", "hr:write", "payroll:read", "payroll:write",
		"masterdata:read", "masterdata:write", "org:read", "org:write",
		"user:read", "user:write", "security:read", "security:write", "audit:read",
	} {
		if !strings.Contains(","+seenPerms+",", ","+wantPerm+",") {
			t.Fatalf("expected X-Permissions to contain %q, got %q", wantPerm, seenPerms)
		}
	}

	// 4. Non-admin token requesting an org outside scope receives 403.
	now := time.Now().UTC()
	restrictedTok, err := auth.SignHS256(auth.Claims{
		Subject:   defaultUID,
		Username:  "staff.jkt",
		OrgID:     defaultOrgID,
		OrgScope:  []string{defaultOrgID},
		Roles:     []string{"ACCOUNTANT"},
		TokenUse:  "access",
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Hour).Unix(),
	}, secret)
	if err != nil {
		t.Fatalf("SignHS256: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/journal-entries", nil)
	req.Header.Set("Authorization", "Bearer "+restrictedTok)
	req.Header.Set("X-Organization-Id", branchOrgID)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-org request status = %d, want 403", rec.Code)
	}
}

func TestSSEStreamEmitsUnnamedDataFrame(t *testing.T) {
	router, err := proxy.NewRouter()
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	hub := sse.NewHub()
	secret := []byte("test-gateway-secret")
	mux := newGatewayMux(router, hub, secret, true, true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stream/events?orgId="+defaultOrgID, nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		mux.ServeHTTP(rec, req)
	}()

	time.Sleep(20 * time.Millisecond)
	hub.Broadcast(sse.StreamEvent{
		EventID:        "evt-sse-101",
		EventType:      "gl.journal.posted",
		OrganizationID: defaultOrgID,
		Payload:        json.RawMessage(`{"journalNo":"JV-2026-0001"}`),
	})
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()
	if !strings.Contains(body, "id: evt-sse-101\ndata: ") {
		t.Fatalf("expected unnamed SSE data frame 'id: evt-sse-101\\ndata: ...', got:\n%s", body)
	}
}

func TestLoginAndRolePermissionsIncludeDotAndColonFormats(t *testing.T) {
	router, err := proxy.NewRouter()
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	secret := []byte("test-gateway-secret")
	mux := newGatewayMux(router, sse.NewHub(), secret, true, false)

	loginBody := bytes.NewBufferString(`{"username":"admin.finance","password":"dev","orgCode":"HLD-01"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", loginBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var loginResp struct {
		Token string      `json:"token"`
		User  userSummary `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("unmarshal login response: %v", err)
	}
	wantPerms := []string{
		"gl.create", "gl.read", "gl.update", "gl.approve", "reporting.read",
		"ap:read", "ap:write", "ar:read", "ar:write",
		"coa:read", "coa:write", "tax:read", "tax:write",
		"currency:read", "currency:write", "treasury:read", "treasury:write",
		"workflow:read", "workflow:write", "workflow:approve",
		"inventory:read", "inventory:write", "procurement:read", "procurement:write",
		"sales:read", "sales:write", "manufacturing:read", "manufacturing:write",
		"hr:read", "hr:write", "payroll:read", "payroll:write",
		"masterdata:read", "masterdata:write", "org:read", "org:write",
		"user:read", "user:write", "security:read", "security:write", "audit:read",
	}
	permSet := make(map[string]bool, len(loginResp.User.Permissions))
	for _, p := range loginResp.User.Permissions {
		permSet[p] = true
	}
	for _, want := range wantPerms {
		if !permSet[want] {
			t.Fatalf("expected login permissions to include %q, got %v", want, loginResp.User.Permissions)
		}
	}
}

func TestGatewayProxiesLedgersToCOAAndOrgUnitsToOrgAndRejectsPayslips(t *testing.T) {
	t.Setenv("GL_SERVICE_URL", "http://gl-service.internal:8080")
	t.Setenv("COA_SERVICE_URL", "http://coa-service.internal:8080")
	t.Setenv("ORG_SERVICE_URL", "http://org-service.internal:8080")
	t.Setenv("PAYROLL_SERVICE_URL", "http://payroll-service.internal:8080")

	router, err := proxy.NewRouter()
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	var glHits int
	glRP, ok := router.Match("/api/v1/journal-entries")
	if !ok {
		t.Fatal("expected /api/v1/journal-entries route to match")
	}
	glRP.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		glHits++
		rec := httptest.NewRecorder()
		rec.WriteHeader(http.StatusOK)
		return rec.Result(), nil
	})

	type capturedReq struct {
		method   string
		host     string
		path     string
		rawQuery string
		perms    string
	}
	var coaReqs []capturedReq
	coaRP, ok := router.Match("/api/v1/ledgers")
	if !ok {
		t.Fatal("expected /api/v1/ledgers route to match")
	}
	if coaRP == glRP {
		t.Fatal("/api/v1/ledgers must not share GL reverse proxy")
	}
	coaRP.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		coaReqs = append(coaReqs, capturedReq{
			method:   r.Method,
			host:     r.URL.Host,
			path:     r.URL.Path,
			rawQuery: r.URL.RawQuery,
			perms:    r.Header.Get("X-Permissions"),
		})
		rec := httptest.NewRecorder()
		rec.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			rec.WriteHeader(http.StatusCreated)
			_, _ = rec.Write([]byte(`{"id":"10000000-0000-0000-0000-000000000002","rowVersion":1}`))
		} else {
			rec.WriteHeader(http.StatusOK)
			_, _ = rec.Write([]byte(`{"data":[]}`))
		}
		return rec.Result(), nil
	})

	var orgReqs []capturedReq
	orgRP, ok := router.Match("/api/v1/org-units")
	if !ok {
		t.Fatal("expected /api/v1/org-units route to match")
	}
	orgRP.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		orgReqs = append(orgReqs, capturedReq{
			method:   r.Method,
			host:     r.URL.Host,
			path:     r.URL.Path,
			rawQuery: r.URL.RawQuery,
			perms:    r.Header.Get("X-Permissions"),
		})
		rec := httptest.NewRecorder()
		rec.Header().Set("Content-Type", "application/json")
		rec.WriteHeader(http.StatusOK)
		_, _ = rec.Write([]byte(`{"data":[]}`))
		return rec.Result(), nil
	})

	secret := []byte("test-gateway-secret")
	mux := newGatewayMux(router, sse.NewHub(), secret, true, false)

	// Unauthenticated requests to newly wired routes are rejected with 401.
	for _, path := range []string{"/api/v1/ledgers", "/api/v1/org-units"} {
		unauthReq := httptest.NewRequest(http.MethodGet, path, nil)
		unauthRec := httptest.NewRecorder()
		mux.ServeHTTP(unauthRec, unauthReq)
		if unauthRec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated GET %s status = %d, want 401", path, unauthRec.Code)
		}
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin.finance","password":"dev","orgCode":"HLD-01"}`))
	loginRec := httptest.NewRecorder()
	mux.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", loginRec.Code, loginRec.Body.String())
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("unmarshal login: %v", err)
	}

	// GET & POST /api/v1/ledgers -> coa-service
	ledgerCases := []struct {
		method    string
		target    string
		body      string
		want      int
		wantQuery string
		perm      string
	}{
		{http.MethodGet, "/api/v1/ledgers?page=1&size=25", "", http.StatusOK, "page=1&size=25", "coa:read"},
		{http.MethodPost, "/api/v1/ledgers", `{"financialOrganizationId":"10000000-0000-0000-0000-000000000001","code":"PRIMARY"}`, http.StatusCreated, "", "coa:write"},
	}
	for _, tc := range ledgerCases {
		req := httptest.NewRequest(tc.method, tc.target, bytes.NewBufferString(tc.body))
		req.Header.Set("Authorization", "Bearer "+loginResp.Token)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("%s %s status = %d, want %d, body = %s", tc.method, tc.target, rec.Code, tc.want, rec.Body.String())
		}
	}
	if glHits != 0 {
		t.Fatalf("expected 0 hits on gl-service for /api/v1/ledgers, got %d", glHits)
	}
	if len(coaReqs) != len(ledgerCases) {
		t.Fatalf("expected %d coa-service requests, got %d", len(ledgerCases), len(coaReqs))
	}
	for i, got := range coaReqs {
		tc := ledgerCases[i]
		if got.host != "coa-service.internal:8080" || got.path != "/ledgers" || got.rawQuery != tc.wantQuery {
			t.Fatalf("coaReq[%d] = %+v, want host=coa-service.internal:8080 path=/ledgers rawQuery=%q", i, got, tc.wantQuery)
		}
		if !strings.Contains(","+got.perms+",", ","+tc.perm+",") {
			t.Fatalf("coaReq[%d] missing required permission %q in %q", i, tc.perm, got.perms)
		}
	}

	// GET /api/v1/org-units, /api/v1/org-units/{id}, and /api/v1/org-units/{id}/hierarchy -> org-service
	orgPaths := []struct {
		reqPath  string
		wantPath string
	}{
		{"/api/v1/org-units", "/org-units"},
		{"/api/v1/org-units/10000000-0000-0000-0000-000000000041", "/org-units/10000000-0000-0000-0000-000000000041"},
		{"/api/v1/org-units/10000000-0000-0000-0000-000000000041/hierarchy", "/org-units/10000000-0000-0000-0000-000000000041/hierarchy"},
	}
	for _, tc := range orgPaths {
		req := httptest.NewRequest(http.MethodGet, tc.reqPath, nil)
		req.Header.Set("Authorization", "Bearer "+loginResp.Token)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, body = %s", tc.reqPath, rec.Code, rec.Body.String())
		}
	}
	if len(orgReqs) != len(orgPaths) {
		t.Fatalf("expected %d org-service requests, got %d", len(orgPaths), len(orgReqs))
	}
	for i, tc := range orgPaths {
		got := orgReqs[i]
		if got.host != "org-service.internal:8080" || got.path != tc.wantPath {
			t.Fatalf("orgReqs[%d] = %+v, want path=%s", i, got, tc.wantPath)
		}
		if !strings.Contains(","+got.perms+",", ",org:read,") {
			t.Fatalf("orgReqs[%d] missing org:read in X-Permissions: %q", i, got.perms)
		}
	}

	// Unimplemented /api/v1/payslips returns clean gateway 404 ROUTE_NOT_FOUND
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payslips", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"ROUTE_NOT_FOUND"`) {
		t.Fatalf("GET /api/v1/payslips = %d %s, want 404 ROUTE_NOT_FOUND", rec.Code, rec.Body.String())
	}
}

func TestRolePermissionsColonAndDotCoverage(t *testing.T) {
	for role, perms := range rolePermissions {
		seen := make(map[string]int, len(perms))
		for _, p := range perms {
			seen[p]++
		}
		for p, count := range seen {
			if count > 1 {
				t.Errorf("rolePermissions[%q] contains duplicate permission %q (%d times)", role, p, count)
			}
		}
	}

	hrPerms := permissionsForRoles([]string{"HR_MANAGER"})
	permSet := make(map[string]bool, len(hrPerms))
	for _, p := range hrPerms {
		permSet[p] = true
	}
	for _, want := range []string{"org.read", "org:read", "hr:read", "payroll:read"} {
		if !permSet[want] {
			t.Fatalf("expected HR_MANAGER permissions to include %q, got %v", want, hrPerms)
		}
	}
}

func TestGatewayRejectsSystemOrgAsActiveTenant(t *testing.T) {
	t.Setenv("GL_SERVICE_URL", "http://gl-service.internal:8080")
	router, err := proxy.NewRouter()
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	secret := []byte("test-gateway-secret")
	mux := newGatewayMux(router, sse.NewHub(), secret, true, false)

	loginBody := bytes.NewBufferString(`{"username":"admin.finance","password":"dev","orgCode":"HLD-01"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", loginBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", rec.Code, rec.Body.String())
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &loginResp)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/journal-entries", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	req.Header.Set("X-Organization-Id", systemOrgID)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when X-Organization-Id is systemOrgID, got %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/stream/events?orgId="+systemOrgID, nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when SSE orgId is systemOrgID, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestGatewayProductionConfigAndMutationScopeAndHMAC(t *testing.T) {
	if err := validateProductionConfig("production", "dev-erp-jwt-secret-change-in-prod", false, false); err == nil {
		t.Fatal("expected error for default JWT_SECRET in production")
	}
	if err := validateProductionConfig("production", "strong-production-secret-at-least-32-bytes", true, false); err == nil {
		t.Fatal("expected error for DEV_AUTH_MODE=true in production")
	}
	if err := validateProductionConfig("production", "strong-production-secret-at-least-32-bytes", false, false); err != nil {
		t.Fatalf("expected valid production config, got %v", err)
	}

	t.Setenv("GL_SERVICE_URL", "http://gl-service.internal:8080")
	router, err := proxy.NewRouter()
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	var seenScope, seenSig string
	rp, _ := router.Match("/api/v1/journal-entries")
	rp.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		seenScope = r.Header.Get("X-Org-Scope")
		seenSig = r.Header.Get("X-Gateway-Signature")
		rec := httptest.NewRecorder()
		rec.WriteHeader(http.StatusCreated)
		return rec.Result(), nil
	})

	secret := []byte("strong-production-secret-at-least-32-bytes")
	mux := newGatewayMux(router, sse.NewHub(), secret, true, false)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin.finance","password":"dev","orgCode":"HLD-01"}`))
	loginRec := httptest.NewRecorder()
	mux.ServeHTTP(loginRec, loginReq)
	var loginResp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(loginRec.Body.Bytes(), &loginResp)

	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/journal-entries", bytes.NewBufferString(`{}`))
	postReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	postRec := httptest.NewRecorder()
	mux.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want 201", postRec.Code)
	}
	if strings.Contains(seenScope, systemOrgID) {
		t.Fatalf("POST request must not include systemOrgID in X-Org-Scope, got %q", seenScope)
	}
	if seenSig == "" {
		t.Fatal("expected X-Gateway-Signature header on proxied request")
	}
}



