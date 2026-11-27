package proxy

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
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
		{"TAX_SERVICE_URL", "http://tax-service:8080", []string{"/tax-codes", "/tax-invoices", "/withholding-tax"}},
		{"TREASURY_SERVICE_URL", "http://treasury-cashbank-service:8080", []string{"/bank-accounts", "/payments", "/bank-statement-lines", "/cash-position"}},
		{"USER_SERVICE_URL", "http://user-service:8080", []string{"/users"}},
		{"SECURITY_SERVICE_URL", "http://security-service:8080", []string{"/roles", "/permissions"}},
		{"MASTERDATA_SERVICE_URL", "http://masterdata-service:8080", []string{"/parties", "/items"}},
		{"ORG_SERVICE_URL", "http://org-service:8080", []string{"/organization-types", "/organizations", "/org-units"}},
		{"WORKFLOW_SERVICE_URL", "http://workflow-service:8080", []string{"/workflow-definitions", "/workflow-instances", "/workflow-tasks"}},
		{"PROCUREMENT_SERVICE_URL", "http://procurement-service:8080", []string{"/purchase-orders", "/purchase-requisitions", "/goods-receipts"}},
		{"INVENTORY_SERVICE_URL", "http://inventory-service:8080", []string{"/warehouses", "/reservations", "/stock-movements", "/stock-balances"}},
		{"MANUFACTURING_SERVICE_URL", "http://manufacturing-service:8080", []string{"/bill-of-materials", "/work-orders"}},
		{"SALES_SERVICE_URL", "http://sales-service:8080", []string{"/sales-orders", "/sales-quotations", "/deliveries"}},
		{"HR_SERVICE_URL", "http://hr-service:8080", []string{"/employees", "/positions", "/leave-requests", "/attendance-logs"}},
		{"PAYROLL_SERVICE_URL", "http://payroll-service:8080", []string{"/payroll-runs"}},
		{"REPORTING_SERVICE_URL", "http://reporting-service:8080", []string{"/gl-balance-snapshots", "/sales-summaries", "/inventory-positions", "/financial-statements"}},
		{"CURRENCY_SERVICE_URL", "http://currency-service:8080", []string{"/currencies", "/exchange-rates", "/exchange-rate-types"}},
		{"DIMENSION_SERVICE_URL", "http://dimension-service:8080", []string{"/cost-centers", "/profit-centers", "/allocation-rules", "/cost-center-assignments", "/profit-center-assignments"}},
		{"LOCALIZATION_SERVICE_URL", "http://localization-service:8080", []string{"/languages", "/translations", "/translation-bundles", "/locale-formats", "/user-preferences", "/organization-languages"}},
		{"FIXEDASSET_SERVICE_URL", "http://fixedasset-service:8080", []string{"/asset-categories", "/fixed-assets"}},
		{"BUDGET_SERVICE_URL", "http://budget-service:8080", []string{"/budgets", "/budget-policies", "/budget-commitments"}},
		{"FINANCIALCLOSE_SERVICE_URL", "http://financialclose-service:8080", []string{"/closing-templates", "/closing-runs", "/closing-tasks", "/period-statuses", "/period-reopenings"}},
		{"CONSOLIDATION_SERVICE_URL", "http://consolidation-service:8080", []string{"/consolidation-groups", "/elimination-rules", "/consolidation-runs"}},
		{"PROJECT_SERVICE_URL", "http://project-service:8080", []string{"/projects", "/wbs-elements", "/project-budgets", "/project-billings", "/change-orders"}},
		{"ASSET_SERVICE_URL", "http://asset-service:8080", []string{"/assets", "/asset-movements", "/asset-verifications"}},
		{"MAINTENANCE_SERVICE_URL", "http://maintenance-service:8080", []string{"/maintenance-schedules", "/maintenance-work-orders", "/maintenance-checklist"}},
		{"CRM_SERVICE_URL", "http://crm-service:8080", []string{"/leads", "/opportunities", "/customer-segments", "/interactions", "/support-cases", "/activity-tasks"}},
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
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
		serviceName := cfg.envKey
		rp := httputil.NewSingleHostReverseProxy(target)
		rp.Transport = transport
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
			cid := strings.TrimSpace(r.Header.Get("X-Correlation-Id"))
			log.Printf("[proxy] cid=%s upstream=%s path=%s err=%v", cid, serviceName, r.URL.Path, err)
			if cid != "" {
				w.Header().Set("X-Correlation-Id", cid)
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":          "BAD_GATEWAY",
					"message":       "upstream service unavailable",
					"correlationId": cid,
				},
			})
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

