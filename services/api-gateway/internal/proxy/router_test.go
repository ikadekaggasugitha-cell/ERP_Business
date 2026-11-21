package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterMatchAndStripPrefix(t *testing.T) {
	t.Setenv("GL_SERVICE_URL", "http://127.0.0.1:18081")
	t.Setenv("CURRENCY_SERVICE_URL", "http://127.0.0.1:18100")

	router, err := NewRouter()
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	rp, ok := router.Match("/api/v1/journal-entries/123")
	if !ok || rp == nil {
		t.Fatal("expected match for /api/v1/journal-entries/123")
	}

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/api/v1/journal-entries/123?page=1", nil)
	rp.Director(req)
	if req.URL.Path != "/journal-entries/123" {
		t.Fatalf("rewritten path = %q, want /journal-entries/123", req.URL.Path)
	}
	if req.URL.Host != "127.0.0.1:18081" {
		t.Fatalf("target host = %q, want 127.0.0.1:18081", req.URL.Host)
	}

	if _, ok := router.Match("/api/v1/exchange-rates"); !ok {
		t.Fatal("expected match for /api/v1/exchange-rates")
	}
	if _, ok := router.Match("/api/v1/exchange-rate-types"); !ok {
		t.Fatal("expected match for /api/v1/exchange-rate-types")
	}
	if _, ok := router.Match("/api/v1/unknown-resource"); ok {
		t.Fatal("unexpected match for /api/v1/unknown-resource")
	}
}

func TestRouterParityFixesRouting(t *testing.T) {
	t.Setenv("GL_SERVICE_URL", "http://127.0.0.1:18081")
	t.Setenv("COA_SERVICE_URL", "http://127.0.0.1:18085")
	t.Setenv("ORG_SERVICE_URL", "http://127.0.0.1:18091")
	t.Setenv("PAYROLL_SERVICE_URL", "http://127.0.0.1:18098")

	router, err := NewRouter()
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	cases := []struct {
		name      string
		method    string
		path      string
		wantHost  string
		wantPath  string
		wantQuery string
	}{
		{
			name:      "list ledgers routes to coa-service",
			method:    http.MethodGet,
			path:      "/api/v1/ledgers?page=1&size=25",
			wantHost:  "127.0.0.1:18085",
			wantPath:  "/ledgers",
			wantQuery: "page=1&size=25",
		},
		{
			name:     "create ledger routes to coa-service",
			method:   http.MethodPost,
			path:     "/api/v1/ledgers",
			wantHost: "127.0.0.1:18085",
			wantPath: "/ledgers",
		},
		{
			name:     "ledger detail routes to coa-service",
			method:   http.MethodGet,
			path:     "/api/v1/ledgers/10000000-0000-0000-0000-000000000002",
			wantHost: "127.0.0.1:18085",
			wantPath: "/ledgers/10000000-0000-0000-0000-000000000002",
		},
		{
			name:     "journal-entries remain on gl-service",
			method:   http.MethodGet,
			path:     "/api/v1/journal-entries",
			wantHost: "127.0.0.1:18081",
			wantPath: "/journal-entries",
		},
		{
			name:     "trial-balance remains on gl-service",
			method:   http.MethodGet,
			path:     "/api/v1/trial-balance",
			wantHost: "127.0.0.1:18081",
			wantPath: "/trial-balance",
		},
		{
			name:      "list org-units routes to org-service",
			method:    http.MethodGet,
			path:      "/api/v1/org-units?limit=50",
			wantHost:  "127.0.0.1:18091",
			wantPath:  "/org-units",
			wantQuery: "limit=50",
		},
		{
			name:     "get org-unit routes to org-service",
			method:   http.MethodGet,
			path:     "/api/v1/org-units/10000000-0000-0000-0000-000000000041",
			wantHost: "127.0.0.1:18091",
			wantPath: "/org-units/10000000-0000-0000-0000-000000000041",
		},
		{
			name:     "org-unit hierarchy routes to org-service",
			method:   http.MethodGet,
			path:     "/api/v1/org-units/10000000-0000-0000-0000-000000000041/hierarchy",
			wantHost: "127.0.0.1:18091",
			wantPath: "/org-units/10000000-0000-0000-0000-000000000041/hierarchy",
		},
		{
			name:     "organizations remain on org-service",
			method:   http.MethodGet,
			path:     "/api/v1/organizations/10000000-0000-0000-0000-000000000041/hierarchy",
			wantHost: "127.0.0.1:18091",
			wantPath: "/organizations/10000000-0000-0000-0000-000000000041/hierarchy",
		},
		{
			name:     "payroll-runs routes to payroll-service",
			method:   http.MethodGet,
			path:     "/api/v1/payroll-runs",
			wantHost: "127.0.0.1:18098",
			wantPath: "/payroll-runs",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://gateway.local"+tc.path, nil)
			rp, ok := router.Match(req.URL.Path)
			if !ok || rp == nil {
				t.Fatalf("expected Match(%q) = true", req.URL.Path)
			}
			rp.Director(req)
			if req.URL.Host != tc.wantHost {
				t.Fatalf("Host = %q, want %q", req.URL.Host, tc.wantHost)
			}
			if req.URL.Path != tc.wantPath {
				t.Fatalf("Path = %q, want %q", req.URL.Path, tc.wantPath)
			}
			if req.URL.RawQuery != tc.wantQuery {
				t.Fatalf("RawQuery = %q, want %q", req.URL.RawQuery, tc.wantQuery)
			}
		})
	}

	unmatched := []string{
		"/api/v1/payslips",
		"/api/v1/payslips/123",
		"/api/v1/ledgers-extra",
		"/api/v1/org-units-extra",
	}
	for _, path := range unmatched {
		if _, ok := router.Match(path); ok {
			t.Fatalf("expected Match(%q) = false", path)
		}
	}
}

func TestRouterExtendedServicesRouting(t *testing.T) {
	t.Setenv("FIXEDASSET_SERVICE_URL", "http://127.0.0.1:18101")
	t.Setenv("BUDGET_SERVICE_URL", "http://127.0.0.1:18103")
	t.Setenv("FINANCIALCLOSE_SERVICE_URL", "http://127.0.0.1:18104")
	t.Setenv("CONSOLIDATION_SERVICE_URL", "http://127.0.0.1:18105")
	t.Setenv("PROJECT_SERVICE_URL", "http://127.0.0.1:18106")
	t.Setenv("ASSET_SERVICE_URL", "http://127.0.0.1:18107")
	t.Setenv("MAINTENANCE_SERVICE_URL", "http://127.0.0.1:18108")
	t.Setenv("CRM_SERVICE_URL", "http://127.0.0.1:18109")

	router, err := NewRouter()
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	cases := []struct {
		path     string
		wantHost string
	}{
		{"/api/v1/fixed-assets", "127.0.0.1:18101"},
		{"/api/v1/budgets/check", "127.0.0.1:18103"},
		{"/api/v1/closing-runs", "127.0.0.1:18104"},
		{"/api/v1/consolidation-runs", "127.0.0.1:18105"},
		{"/api/v1/project-billings", "127.0.0.1:18106"},
		{"/api/v1/asset-movements", "127.0.0.1:18107"},
		{"/api/v1/maintenance-work-orders", "127.0.0.1:18108"},
		{"/api/v1/opportunities", "127.0.0.1:18109"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "http://gateway.local"+tc.path, nil)
		rp, ok := router.Match(req.URL.Path)
		if !ok || rp == nil {
			t.Fatalf("expected Match(%q) = true", tc.path)
		}
		rp.Director(req)
		if req.URL.Host != tc.wantHost {
			t.Fatalf("%s routed to %s, want %s", tc.path, req.URL.Host, tc.wantHost)
		}
	}
}

