package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

type RouteRule struct {
	Prefixes []string
	Service  string
	Target   *url.URL
	Proxy    *httputil.ReverseProxy
}

type Router struct {
	rules []RouteRule
}

type routeConfig struct {
	envKey     string
	defaultURL string
	prefixes   []string
}

func NewRouter() (*Router, error) {
	configs := []routeConfig{
		{"GL_SERVICE_URL", "http://gl-service:8080", []string{"/journal-entries", "/trial-balance"}},
		{"AP_SERVICE_URL", "http://ap-service:8080", []string{"/vendor-invoices"}},
		{"AUDIT_SERVICE_URL", "http://audit-service:8080", []string{"/audit-events"}},
		{"AR_SERVICE_URL", "http://ar-service:8080", []string{"/customer-invoices"}},
		{"COA_SERVICE_URL", "http://coa-service:8080", []string{"/charts-of-accounts", "/accounts", "/ledgers", "/fiscal-calendars", "/fiscal-years", "/fiscal-periods", "/financial-organizations"}},
		{"TAX_SERVICE_URL", "http://tax-service:8080", []string{"/tax-codes", "/tax-invoices"}},
		{"TREASURY_SERVICE_URL", "http://treasury-cashbank-service:8080", []string{"/bank-accounts", "/payments", "/bank-statement-lines"}},
		{"USER_SERVICE_URL", "http://user-service:8080", []string{"/users"}},
		{"SECURITY_SERVICE_URL", "http://security-service:8080", []string{"/roles", "/permissions"}},
		{"MASTERDATA_SERVICE_URL", "http://masterdata-service:8080", []string{"/parties", "/items"}},
		{"ORG_SERVICE_URL", "http://org-service:8080", []string{"/organization-types", "/organizations", "/org-units"}},
		{"WORKFLOW_SERVICE_URL", "http://workflow-service:8080", []string{"/workflow-definitions", "/workflow-instances", "/workflow-tasks"}},
		{"PROCUREMENT_SERVICE_URL", "http://procurement-service:8080", []string{"/purchase-orders", "/purchase-requisitions", "/goods-receipts"}},
		{"INVENTORY_SERVICE_URL", "http://inventory-service:8080", []string{"/warehouses", "/reservations", "/stock-movements", "/stock-balances"}},
		{"MANUFACTURING_SERVICE_URL", "http://manufacturing-service:8080", []string{"/bill-of-materials", "/work-orders"}},
		{"SALES_SERVICE_URL", "http://sales-service:8080", []string{"/sales-orders", "/deliveries"}},
		{"HR_SERVICE_URL", "http://hr-service:8080", []string{"/employees", "/positions"}},
		{"PAYROLL_SERVICE_URL", "http://payroll-service:8080", []string{"/payroll-runs"}},
		{"REPORTING_SERVICE_URL", "http://reporting-service:8080", []string{"/gl-balance-snapshots", "/sales-summaries", "/inventory-positions"}},
		{"CURRENCY_SERVICE_URL", "http://currency-service:8080", []string{"/currencies", "/exchange-rates", "/exchange-rate-types"}},
		{"DIMENSION_SERVICE_URL", "http://dimension-service:8080", []string{"/cost-centers", "/profit-centers", "/allocation-rules"}},
		{"LOCALIZATION_SERVICE_URL", "http://localization-service:8080", []string{"/languages", "/translations", "/translation-bundles", "/locale-formats"}},
	}

	rules := make([]RouteRule, 0, len(configs))
	for _, cfg := range configs {
		raw := os.Getenv(cfg.envKey)
		if raw == "" {
			raw = cfg.defaultURL
		}
		target, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid target URL for %s (%s): %w", cfg.envKey, raw, err)
		}
		rp := httputil.NewSingleHostReverseProxy(target)
		origDirector := rp.Director
		rp.Director = func(req *http.Request) {
			origDirector(req)
			req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/v1")
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			if req.URL.RawPath != "" {
				req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, "/api/v1")
			}
			req.Host = target.Host
		}
		rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = fmt.Fprintf(w, `{"error":{"code":"BAD_GATEWAY","message":%q}}`, err.Error())
		}
		rules = append(rules, RouteRule{
			Prefixes: cfg.prefixes,
			Service:  cfg.envKey,
			Target:   target,
			Proxy:    rp,
		})
	}
	return &Router{rules: rules}, nil
}

func (r *Router) Match(apiPath string) (*httputil.ReverseProxy, bool) {
	sub := strings.TrimPrefix(apiPath, "/api/v1")
	for _, rule := range r.rules {
		for _, p := range rule.Prefixes {
			if sub == p || strings.HasPrefix(sub, p+"/") {
				return rule.Proxy, true
			}
		}
	}
	return nil, false
}

