package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/api-gateway/internal/auth"
	"github.com/sinergiinvestasiproperti/erp/services/api-gateway/internal/proxy"
	"github.com/sinergiinvestasiproperti/erp/services/api-gateway/internal/sse"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/telemetry"
)

const (
	systemOrgID   = "00000000-0000-0000-0000-000000000000"
	defaultOrgID  = "11111111-1111-1111-1111-111111111111" // HLD-01 (Head Office / Holding)
	branchOrgID   = "22222222-2222-2222-2222-222222222222" // SUB-PRO (Subsidiary 1)
	subLogOrgID   = "33333333-3333-3333-3333-333333333333" // SUB-LOG (Subsidiary 2)
	brcJktOrgID   = "44444444-4444-4444-4444-444444444444" // BRC-JKT (Branch under SUB-PRO)
	defaultUID    = "00000000-0000-0000-0000-000000000101"
	sessionCookie = "erp_session"
	refreshCookie = "erp_refresh"
)

type authRuntime struct {
	secret   []byte
	devAuth  bool
	oidc     *auth.OIDCVerifier
	sessions *auth.SessionStore
}

type orgSummary struct {
	ID                 string  `json:"id"`
	Code               string  `json:"code"`
	Name               string  `json:"name"`
	Type               string  `json:"type"`
	ParentID           *string `json:"parentId"`
	FunctionalCurrency string  `json:"functionalCurrency"`
}

type userSummary struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	FullName    string   `json:"fullName"`
	Email       string   `json:"email"`
	ActiveOrgID string   `json:"activeOrgId"`
	AdminTier   int      `json:"adminTier"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

var devOrganizations = []orgSummary{
	{
		ID:                 defaultOrgID,
		Code:               "HLD-01",
		Name:               "PT Sinergi Investasi Properti (Head Office / Holding)",
		Type:               "HOLDING",
		ParentID:           nil,
		FunctionalCurrency: "IDR",
	},
	{
		ID:                 branchOrgID,
		Code:               "SUB-PRO",
		Name:               "PT Sinergi Properti Jakarta (Anak Perusahaan)",
		Type:               "COMPANY",
		ParentID:           strPtr(defaultOrgID),
		FunctionalCurrency: "IDR",
	},
	{
		ID:                 subLogOrgID,
		Code:               "SUB-LOG",
		Name:               "PT Sinergi Logistik Nusantara (Anak Perusahaan)",
		Type:               "COMPANY",
		ParentID:           strPtr(defaultOrgID),
		FunctionalCurrency: "IDR",
	},
	{
		ID:                 brcJktOrgID,
		Code:               "BRC-JKT",
		Name:               "Cabang Jakarta Selatan (Branch)",
		Type:               "BRANCH",
		ParentID:           strPtr(branchOrgID),
		FunctionalCurrency: "IDR",
	},
}

var rolePermissions = map[string][]string{
	"SYSTEM_ADMIN": {
		"gl.create", "gl.read", "gl.update", "gl.approve",
		"ap.create", "ap.read", "ap.update", "ap.approve",
		"ar.create", "ar.read", "ar.update", "ar.approve",
		"tax.create", "tax.read", "tax.update", "tax.approve",
		"treasury.create", "treasury.read", "treasury.update", "treasury.approve",
		"procurement.create", "procurement.read", "procurement.update", "procurement.approve",
		"inventory.create", "inventory.read", "inventory.update",
		"sales.create", "sales.read", "sales.update",
		"audit.read", "reporting.read", "org.read",
		"gl:read", "gl:write", "gl:approve",
		"ap:read", "ap:write", "ap:approve",
		"ar:read", "ar:write", "ar:approve",
		"coa:read", "coa:write",
		"tax:read", "tax:write", "tax:approve",
		"currency:read", "currency:write",
		"treasury:read", "treasury:write", "treasury:approve",
		"workflow:read", "workflow:write", "workflow:approve", "workflow.approve",
		"inventory:read", "inventory:write",
		"procurement:read", "procurement:write", "procurement:approve",
		"sales:read", "sales:write",
		"manufacturing:read", "manufacturing:write",
		"hr:read", "hr:write",
		"payroll:read", "payroll:write", "payroll:approve", "payroll.approve",
		"masterdata:read", "masterdata:write",
		"org:read", "org:write",
		"user:read", "user:write",
		"security:read", "security:write",
		"audit:read",
		"dimension:read", "dimension:write", "localization:read", "localization:write",
		"fixedasset.read", "fixedasset.create", "fixedasset.update", "fixedasset:read", "fixedasset:write",
		"budget.read", "budget.create", "budget.update", "budget.approve", "budget:read", "budget:write", "budget:approve",
		"financialclose.read", "financialclose.create", "financialclose.update", "financialclose.approve", "financialclose:read", "financialclose:write", "financialclose:approve",
		"consolidation.read", "consolidation.create", "consolidation.update", "consolidation.approve", "consolidation:read", "consolidation:write", "consolidation:approve",
		"project.read", "project.create", "project.update", "project.approve", "project:read", "project:write", "project:approve",
		"asset.read", "asset.create", "asset.update", "asset:read", "asset:write",
		"maintenance.read", "maintenance.create", "maintenance.update", "maintenance:read", "maintenance:write",
		"crm.read", "crm.create", "crm.update", "crm:read", "crm:write",
	},
	"HOLDING_ADMIN": {
		"gl.create", "gl.read", "gl.update", "gl.approve",
		"ap.create", "ap.read", "ap.update", "ap.approve",
		"ar.create", "ar.read", "ar.update", "ar.approve",
		"tax.create", "tax.read", "tax.update", "tax.approve",
		"treasury.create", "treasury.read", "treasury.update", "treasury.approve",
		"procurement.create", "procurement.read", "procurement.update", "procurement.approve",
		"inventory.create", "inventory.read", "inventory.update",
		"sales.create", "sales.read", "sales.update",
		"audit.read", "reporting.read", "org.read",
		"gl:read", "gl:write", "gl:approve",
		"ap:read", "ap:write", "ap:approve",
		"ar:read", "ar:write", "ar:approve",
		"coa:read", "coa:write",
		"tax:read", "tax:write", "tax:approve",
		"currency:read", "currency:write",
		"treasury:read", "treasury:write", "treasury:approve",
		"workflow:read", "workflow:write", "workflow:approve", "workflow.approve",
		"inventory:read", "inventory:write",
		"procurement:read", "procurement:write", "procurement:approve",
		"sales:read", "sales:write",
		"manufacturing:read", "manufacturing:write",
		"hr:read", "hr:write",
		"payroll:read", "payroll:write", "payroll:approve", "payroll.approve",
		"masterdata:read", "masterdata:write",
		"org:read", "org:write",
		"user:read", "user:write",
		"security:read", "security:write",
		"audit:read",
		"dimension:read", "dimension:write", "localization:read", "localization:write",
		"fixedasset.read", "fixedasset.create", "fixedasset.update", "fixedasset:read", "fixedasset:write",
		"budget.read", "budget.create", "budget.update", "budget.approve", "budget:read", "budget:write", "budget:approve",
		"financialclose.read", "financialclose.create", "financialclose.update", "financialclose.approve", "financialclose:read", "financialclose:write", "financialclose:approve",
		"consolidation.read", "consolidation.create", "consolidation.update", "consolidation.approve", "consolidation:read", "consolidation:write", "consolidation:approve",
		"project.read", "project.create", "project.update", "project.approve", "project:read", "project:write", "project:approve",
		"asset.read", "asset.create", "asset.update", "asset:read", "asset:write",
		"maintenance.read", "maintenance.create", "maintenance.update", "maintenance:read", "maintenance:write",
		"crm.read", "crm.create", "crm.update", "crm:read", "crm:write",
	},
	"SUBSIDIARY_ADMIN": {
		"gl.create", "gl.read", "gl.update", "gl.approve",
		"ap.create", "ap.read", "ap.update", "ap.approve",
		"ar.create", "ar.read", "ar.update", "ar.approve",
		"tax.create", "tax.read", "tax.update", "tax.approve",
		"treasury.create", "treasury.read", "treasury.update", "treasury.approve",
		"procurement.create", "procurement.read", "procurement.update", "procurement.approve",
		"inventory.create", "inventory.read", "inventory.update",
		"sales.create", "sales.read", "sales.update",
		"audit.read", "reporting.read", "org.read",
		"gl:read", "gl:write", "gl:approve",
		"ap:read", "ap:write", "ap:approve",
		"ar:read", "ar:write", "ar:approve",
		"coa:read", "coa:write",
		"tax:read", "tax:write", "tax:approve",
		"currency:read", "currency:write",
		"treasury:read", "treasury:write", "treasury:approve",
		"workflow:read", "workflow:write", "workflow:approve", "workflow.approve",
		"inventory:read", "inventory:write",
		"procurement:read", "procurement:write", "procurement:approve",
		"sales:read", "sales:write",
		"manufacturing:read", "manufacturing:write",
		"hr:read", "hr:write",
		"payroll:read", "payroll:write", "payroll:approve", "payroll.approve",
		"masterdata:read", "masterdata:write",
		"org:read", "org:write",
		"user:read", "user:write",
		"security:read", "security:write",
		"audit:read",
		"dimension:read", "dimension:write", "localization:read", "localization:write",
		"fixedasset.read", "fixedasset.create", "fixedasset.update", "fixedasset:read", "fixedasset:write",
		"budget.read", "budget.create", "budget.update", "budget.approve", "budget:read", "budget:write", "budget:approve",
		"financialclose.read", "financialclose.create", "financialclose.update", "financialclose.approve", "financialclose:read", "financialclose:write", "financialclose:approve",
		"project.read", "project.create", "project.update", "project.approve", "project:read", "project:write", "project:approve",
		"asset.read", "asset.create", "asset.update", "asset:read", "asset:write",
		"maintenance.read", "maintenance.create", "maintenance.update", "maintenance:read", "maintenance:write",
		"crm.read", "crm.create", "crm.update", "crm:read", "crm:write",
	},
	"BRANCH_ADMIN": {
		"gl.create", "gl.read", "gl.update",
		"ap.create", "ap.read", "ap.update",
		"ar.create", "ar.read", "ar.update",
		"tax.read", "treasury.read",
		"procurement.create", "procurement.read", "procurement.update",
		"inventory.create", "inventory.read", "inventory.update",
		"sales.create", "sales.read", "sales.update",
		"audit.read", "reporting.read", "org.read",
		"gl:read", "gl:write",
		"ap:read", "ap:write",
		"ar:read", "ar:write",
		"coa:read", "tax:read", "currency:read", "treasury:read",
		"workflow:read", "workflow:write",
		"inventory:read", "inventory:write",
		"procurement:read", "procurement:write",
		"sales:read", "sales:write",
		"manufacturing:read", "manufacturing:write",
		"hr:read", "hr:write",
		"payroll:read",
		"masterdata:read", "masterdata:write",
		"org:read",
		"user:read", "user:write",
		"audit:read",
		"dimension:read", "localization:read",
		"fixedasset.read", "fixedasset:read",
		"budget.read", "budget:read",
		"project.read", "project:read", "project:write",
		"asset.read", "asset:read", "asset:write",
		"maintenance.read", "maintenance:read", "maintenance:write",
		"crm.read", "crm:read", "crm:write",
	},
	"FINANCE_MANAGER": {
		"gl.create", "gl.read", "gl.update", "gl.approve",
		"ap.create", "ap.read", "ap.update", "ap.approve",
		"ar.create", "ar.read", "ar.update", "ar.approve",
		"tax.create", "tax.read", "tax.update", "tax.approve",
		"treasury.create", "treasury.read", "treasury.update", "treasury.approve",
		"procurement.read", "procurement.approve",
		"inventory.read",
		"sales.read",
		"audit.read", "reporting.read", "org.read",
		"gl:read", "gl:write", "gl:approve",
		"ap:read", "ap:write", "ap:approve",
		"ar:read", "ar:write", "ar:approve",
		"coa:read", "coa:write",
		"tax:read", "tax:write", "tax:approve",
		"currency:read", "currency:write",
		"treasury:read", "treasury:write", "treasury:approve",
		"workflow:read", "workflow:write", "workflow:approve", "workflow.approve",
		"inventory:read",
		"procurement:read", "procurement:approve",
		"sales:read",
		"manufacturing:read",
		"payroll:read", "payroll:write", "payroll:approve", "payroll.approve",
		"masterdata:read", "masterdata:write",
		"org:read",
		"audit:read",
		"dimension:read", "dimension:write", "localization:read",
		"fixedasset.read", "fixedasset.create", "fixedasset.update", "fixedasset:read", "fixedasset:write",
		"budget.read", "budget.create", "budget.update", "budget.approve", "budget:read", "budget:write", "budget:approve",
		"financialclose.read", "financialclose.create", "financialclose.update", "financialclose.approve", "financialclose:read", "financialclose:write", "financialclose:approve",
		"consolidation.read", "consolidation.create", "consolidation.update", "consolidation.approve", "consolidation:read", "consolidation:write", "consolidation:approve",
		"project.read", "project:read", "asset.read", "asset:read", "maintenance.read", "maintenance:read", "crm.read", "crm:read",
	},
	"ACCOUNTANT": {
		"gl.create", "gl.read", "gl.update",
		"ap.create", "ap.read", "ap.update",
		"ar.create", "ar.read", "ar.update",
		"tax.create", "tax.read", "tax.update",
		"treasury.create", "treasury.read", "treasury.update",
		"reporting.read", "org.read",
		"ap:read", "ap:write",
		"ar:read", "ar:write",
		"coa:read", "coa:write",
		"tax:read", "tax:write",
		"currency:read", "currency:write",
		"treasury:read", "treasury:write",
		"workflow:read", "workflow:write",
		"inventory:read",
		"procurement:read",
		"sales:read",
		"manufacturing:read",
		"masterdata:read",
		"org:read",
		"dimension:read", "localization:read",
		"fixedasset.read", "fixedasset.create", "fixedasset:read", "fixedasset:write",
		"budget.read", "budget:read",
		"financialclose.read", "financialclose:read", "financialclose:write",
		"consolidation.read", "consolidation:read",
	},
	"PROCUREMENT_MANAGER": {
		"procurement.create", "procurement.read", "procurement.update", "procurement.approve",
		"inventory.create", "inventory.read", "inventory.update",
		"ap.read", "org.read",
		"procurement:read", "procurement:write", "procurement:approve",
		"inventory:read", "inventory:write",
		"ap:read",
		"masterdata:read", "masterdata:write",
		"workflow:read", "workflow:write", "workflow:approve", "workflow.approve",
		"org:read",
		"budget.read", "budget:read",
		"asset.read", "asset:read", "asset:write",
		"maintenance.read", "maintenance:read", "maintenance:write",
	},
	"SALES_MANAGER": {
		"sales.create", "sales.read", "sales.update",
		"ar.create", "ar.read", "ar.update",
		"inventory.read", "org.read",
		"sales:read", "sales:write",
		"ar:read", "ar:write",
		"inventory:read",
		"masterdata:read", "masterdata:write",
		"workflow:read", "workflow:write",
		"org:read",
		"crm.read", "crm.create", "crm.update", "crm:read", "crm:write",
		"project.read", "project:read", "project:write",
	},
	"HR_MANAGER": {
		"hr:read", "hr:write",
		"payroll:read", "payroll:write", "payroll:approve", "payroll.approve",
		"workflow:read", "workflow:write", "workflow:approve", "workflow.approve",
		"org.read", "org:read",
		"user:read",
	},
	"TAX_TREASURY_OFFICER": {
		"tax.create", "tax.read", "tax.update", "tax.approve", "tax:read", "tax:write", "tax:approve",
		"treasury.create", "treasury.read", "treasury.update", "treasury.approve", "treasury:read", "treasury:write", "treasury:approve",
		"currency:read", "currency:write",
		"ap.read", "ap:read", "ar.read", "ar:read", "gl.read", "coa:read", "reporting.read", "org.read", "org:read", "workflow:read",
	},
	"WAREHOUSE_MANAGER": {
		"inventory.create", "inventory.read", "inventory.update", "inventory:read", "inventory:write",
		"asset.read", "asset.create", "asset.update", "asset:read", "asset:write",
		"procurement.read", "procurement:read", "sales.read", "sales:read", "maintenance.read", "maintenance:read",
		"masterdata:read", "reporting.read", "org.read", "org:read", "workflow:read",
	},
	"OPERATIONS_MANAGER": {
		"manufacturing:read", "manufacturing:write",
		"project.read", "project.create", "project.update", "project.approve", "project:read", "project:write", "project:approve",
		"asset.read", "asset.create", "asset.update", "asset:read", "asset:write",
		"maintenance.read", "maintenance.create", "maintenance.update", "maintenance:read", "maintenance:write",
		"inventory.create", "inventory.read", "inventory.update", "inventory:read", "inventory:write",
		"procurement.read", "procurement:read", "dimension:read", "masterdata:read", "reporting.read", "org.read", "org:read", "workflow:read",
	},
	"STAFF_OPERATOR": {
		"inventory.create", "inventory.read", "inventory.update", "inventory:read", "inventory:write",
		"procurement.create", "procurement.read", "procurement.update", "procurement:read", "procurement:write",
		"sales.create", "sales.read", "sales.update", "sales:read", "sales:write",
		"manufacturing:read", "manufacturing:write",
		"project.read", "project.create", "project:read", "project:write",
		"asset.read", "asset:read", "maintenance.read", "maintenance:read", "crm.read", "crm:read",
		"masterdata:read", "org.read", "org:read", "workflow:read",
	},
	"AUDITOR": {
		"gl.read", "ap.read", "ar.read", "tax.read", "treasury.read",
		"procurement.read", "inventory.read", "sales.read",
		"audit.read", "reporting.read", "org.read",
		"ap:read", "ar:read", "coa:read", "tax:read", "currency:read",
		"treasury:read", "workflow:read", "inventory:read", "procurement:read",
		"sales:read", "manufacturing:read", "hr:read", "payroll:read",
		"masterdata:read", "org:read", "user:read", "security:read", "audit:read",
		"dimension:read", "localization:read",
		"fixedasset.read", "fixedasset:read",
		"budget.read", "budget:read",
		"financialclose.read", "financialclose:read",
		"consolidation.read", "consolidation:read",
		"project.read", "project:read",
		"asset.read", "asset:read",
		"maintenance.read", "maintenance:read",
		"crm.read", "crm:read",
	},
}

func permissionsForRoles(roles []string) []string {
	var out []string
	seen := make(map[string]struct{}, 64)
	for _, role := range roles {
		for _, perm := range rolePermissions[strings.ToUpper(strings.TrimSpace(role))] {
			if _, ok := seen[perm]; !ok {
				seen[perm] = struct{}{}
				out = append(out, perm)
			}
		}
	}
	return out
}

func validateProductionConfig(appEnv, jwtSecret string, devAuthMode, allowAnon bool) error {
	if !strings.EqualFold(strings.TrimSpace(appEnv), "production") {
		return nil
	}
	s := strings.TrimSpace(jwtSecret)
	if s == "" || s == "dev-erp-jwt-secret-change-in-prod" || s == "dev-erp-jwt-secret-change-me" || len(s) < 32 {
		return errors.New("JWT_SECRET must be set to a strong >=32-byte secret when APP_ENV=production")
	}
	if devAuthMode {
		return errors.New("DEV_AUTH_MODE must be false when APP_ENV=production")
	}
	if allowAnon {
		return errors.New("DEV_AUTH_ALLOW_ANON must be false when APP_ENV=production")
	}
	return nil
}

func main() {
	addr := getenv("HTTP_ADDR", ":8000")
	appEnv := getenv("APP_ENV", "development")
	secretStr := getenv("JWT_SECRET", "dev-erp-jwt-secret-change-in-prod")
	if secretStr == "dev-erp-jwt-secret-change-in-prod" || secretStr == "dev-erp-jwt-secret-change-me" {
		log.Println("WARN: using default dev JWT_SECRET; set JWT_SECRET in production")
	}
	jwtSecret := []byte(secretStr)
	devAuthMode := strings.EqualFold(getenv("DEV_AUTH_MODE", "true"), "true")
	allowAnon := strings.EqualFold(getenv("DEV_AUTH_ALLOW_ANON", "false"), "true")
	if err := validateProductionConfig(appEnv, secretStr, devAuthMode, allowAnon); err != nil {
		log.Fatalf("gateway production config check failed: %v", err)
	}
	brokers := splitCSV(os.Getenv("KAFKA_BROKERS"))
	allowedOrigins := splitCSV(getenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000,http://localhost:5173"))
	defaultAllowExt := "true"
	if strings.EqualFold(appEnv, "production") {
		defaultAllowExt = "false"
	}
	allowChromeExt := strings.EqualFold(getenv("CORS_ALLOW_CHROME_EXTENSIONS", defaultAllowExt), "true")
	allowedExtIDs := splitCSV(os.Getenv("CORS_ALLOWED_EXTENSION_IDS"))

	router, err := proxy.NewRouter()
	if err != nil {
		log.Fatalf("gateway router: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := sse.NewHub()
	go hub.StartKafkaBridge(ctx, brokers, getenv("KAFKA_GROUP_ID", "api-gateway-sse"))

	tracer := telemetry.InitFromEnv("api-gateway")
	sessions := auth.NewSessionStore(os.Getenv("REDIS_URL"))
	var oidcVerifier *auth.OIDCVerifier
	if issuer := strings.TrimSpace(os.Getenv("OIDC_ISSUER_URL")); issuer != "" || strings.TrimSpace(os.Getenv("OIDC_JWKS_URL")) != "" {
		v, err := auth.NewOIDCVerifier(auth.OIDCConfig{
			IssuerURL: issuer,
			Audience:  os.Getenv("OIDC_AUDIENCE"),
			JWKSURL:   os.Getenv("OIDC_JWKS_URL"),
		})
		if err != nil {
			log.Fatalf("oidc verifier init failed: %v", err)
		}
		oidcVerifier = v
	}

	metrics := httpx.NewMetrics("api-gateway")
	mux := newGatewayMux(router, hub, jwtSecret, devAuthMode, allowAnon, brokers, metrics, oidcVerifier, sessions)
	rps := 200
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv("RATE_LIMIT_RPS"))); err == nil && v > 0 {
		rps = v
	}
	handler := withRecovery(metrics.Wrap(withCORS(allowedOrigins, allowChromeExt, allowedExtIDs)(withRateLimit(rps, rps*2, sessions)(mux))))
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // SSE connections are long-lived; REST routes set per-request write deadlines
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("api-gateway listening on %s (devAuth=%v, oidc=%v, kafka=%v)", addr, devAuthMode, oidcVerifier != nil, brokers)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	cancel()
	sc, scCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer scCancel()
	_ = srv.Shutdown(sc)
	_ = tracer.Shutdown(sc)
	log.Println("api-gateway stopped")
}

func newGatewayMux(router *proxy.Router, hub *sse.Hub, jwtSecret []byte, devAuthMode, allowAnon bool, extra ...any) *http.ServeMux {
	var brokers []string
	var metrics *httpx.Metrics
	var oidcVerifier *auth.OIDCVerifier
	var sessions *auth.SessionStore
	for _, item := range extra {
		switch v := item.(type) {
		case []string:
			brokers = v
		case *httpx.Metrics:
			metrics = v
		case *auth.OIDCVerifier:
			oidcVerifier = v
		case *auth.SessionStore:
			sessions = v
		}
	}
	if metrics == nil {
		metrics = httpx.NewMetrics("api-gateway")
	}
	if sessions == nil {
		sessions = auth.NewSessionStore("")
	}
	rt := &authRuntime{
		secret:   jwtSecret,
		devAuth:  devAuthMode,
		oidc:     oidcVerifier,
		sessions: sessions,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "api-gateway"})
	})
	mux.HandleFunc("GET /readyz", httpx.ReadinessHandler("api-gateway", func(c context.Context) error {
		return eventbus.CheckBrokers(c, brokers)
	}))
	mux.HandleFunc("GET /metrics", metrics.Handler())
	mux.HandleFunc("GET /openapi.yaml", handleOpenAPISpec())

	mux.HandleFunc("POST /api/v1/auth/login", handleLogin(rt))
	mux.HandleFunc("POST /api/v1/auth/refresh", handleRefresh(rt))
	mux.HandleFunc("POST /api/v1/auth/logout", handleLogout(rt))
	mux.HandleFunc("GET /api/v1/auth/me", handleMe(rt))
	mux.HandleFunc("POST /api/v1/auth/stream-ticket", handleStreamTicket(rt))
	mux.HandleFunc("GET /api/v1/stream/events", handleSSE(hub, rt, allowAnon))

	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(30 * time.Second))
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10 MiB gateway payload cap
		}
		_, childTC := telemetry.ExtractOrGenerate(r.Header)
		r.Header.Set(telemetry.HeaderTraceparent, childTC.String())
		w.Header().Set(telemetry.HeaderTraceparent, childTC.String())

		rp, ok := router.Match(r.URL.Path)
		if !ok {
			writeErr(w, http.StatusNotFound, "ROUTE_NOT_FOUND", "No upstream service configured for path", r.Header.Get("X-Correlation-Id"))
			return
		}

		// Strip client-supplied privileged headers before authentication.
		r.Header.Del("X-User-Id")
		r.Header.Del("X-Roles")
		r.Header.Del("X-Permissions")
		r.Header.Del("X-Org-Scope")
		r.Header.Del("X-Admin-Tier")
		r.Header.Del("X-Gateway-Signature")
		r.Header.Del("X-Gateway-Timestamp")

		corrID := strings.TrimSpace(r.Header.Get("X-Correlation-Id"))
		if _, err := uuid.Parse(corrID); err != nil {
			corrID = uuid.NewString()
			r.Header.Set("X-Correlation-Id", corrID)
		}

		isReadMethod := r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions

		claims, err := extractClaimsWithRuntime(r, rt, "access")
		if err != nil {
			if !allowAnon {
				writeErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "Valid Bearer token or session required", corrID)
				return
			}
			// Explicit dev-only fallback when DEV_AUTH_ALLOW_ANON=true.
			orgID := validUUIDOrDefault(r.Header.Get("X-Organization-Id"), defaultOrgID)
			if orgID == systemOrgID {
				orgID = defaultOrgID
			}
			scope := orgID
			if isReadMethod {
				scope = orgID + "," + systemOrgID
			}
			r.Header.Set("X-Organization-Id", scope)
			r.Header.Set("X-Org-Scope", scope)
			r.Header.Set("X-User-Id", defaultUID)
			r.Header.Set("X-Roles", "SYSTEM_ADMIN")
			r.Header.Set("X-Admin-Tier", strconv.Itoa(httpx.AdminTierSystem))
			r.Header.Set("X-Permissions", strings.Join(permissionsForRoles([]string{"SYSTEM_ADMIN"}), ","))
			httpx.SignGatewayRequest(r, jwtSecret)
			rp.ServeHTTP(w, r)
			return
		}

		requestedOrg := strings.TrimSpace(r.Header.Get("X-Organization-Id"))
		if requestedOrg == "" {
			requestedOrg = claims.OrgID
		}
		if _, err := uuid.Parse(requestedOrg); err != nil {
			writeErr(w, http.StatusBadRequest, "INVALID_ORG_ID", "X-Organization-Id must be a valid UUID", corrID)
			return
		}
		if requestedOrg == systemOrgID {
			writeErr(w, http.StatusForbidden, "ORG_SCOPE_DENIED", "System organization cannot be selected as active tenant", corrID)
			return
		}
		if !slices.Contains(claims.OrgScope, requestedOrg) && !slices.Contains(claims.Roles, "SYSTEM_ADMIN") {
			writeErr(w, http.StatusForbidden, "ORG_SCOPE_DENIED", "Requested organization is outside caller scope", corrID)
			return
		}

		scopeList := []string{requestedOrg}
		for _, id := range claims.OrgScope {
			if id != requestedOrg && id != systemOrgID {
				scopeList = append(scopeList, id)
			}
		}
		if isReadMethod && !slices.Contains(scopeList, systemOrgID) {
			scopeList = append(scopeList, systemOrgID)
		}

		effectivePerms := slices.Clone(claims.Permissions)
		for _, rpPerm := range permissionsForRoles(claims.Roles) {
			if !slices.Contains(effectivePerms, rpPerm) {
				effectivePerms = append(effectivePerms, rpPerm)
			}
		}

		adminTier := httpx.AdminTierForRoles(claims.Roles)
		r.Header.Set("X-Organization-Id", strings.Join(scopeList, ","))
		r.Header.Set("X-Org-Scope", strings.Join(scopeList, ","))
		r.Header.Set("X-User-Id", claims.Subject)
		r.Header.Set("X-Roles", strings.Join(claims.Roles, ","))
		r.Header.Set("X-Admin-Tier", strconv.Itoa(adminTier))
		r.Header.Set("X-Permissions", strings.Join(effectivePerms, ","))
		httpx.SignGatewayRequest(r, jwtSecret)
		rp.ServeHTTP(w, r)
	})
	return mux
}

func handleLogin(rt *authRuntime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			OrgCode        string `json:"orgCode"`
			OrganizationID string `json:"organizationId"`
			Username       string `json:"username"`
			Password       string `json:"password"`
			IDToken        string `json:"idToken"`
			AccessToken    string `json:"accessToken"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "INVALID_JSON", "Malformed JSON request", "")
			return
		}
		if !rt.devAuth {
			if rt.oidc == nil {
				writeErr(w, http.StatusNotImplemented, "OIDC_REQUIRED", "DEV_AUTH_MODE is disabled; configure OIDC_ISSUER_URL or OIDC_JWKS_URL", "")
				return
			}
			rawOIDC := firstNonEmptyStr(req.IDToken, req.AccessToken, strings.TrimPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer "))
			if rawOIDC == "" {
				writeErr(w, http.StatusUnauthorized, "OIDC_TOKEN_REQUIRED", "OIDC idToken/accessToken or Bearer token is required", "")
				return
			}
			oidcClaims, err := rt.oidc.Verify(r.Context(), rawOIDC, time.Now().UTC())
			if err != nil {
				writeErr(w, http.StatusUnauthorized, "OIDC_INVALID_TOKEN", "OIDC token verification failed: "+err.Error(), "")
				return
			}
			activeOrg := resolveOrg(firstNonEmptyStr(req.OrganizationID, oidcClaims.OrgID), req.OrgCode)
			if len(oidcClaims.OrgScope) == 0 {
				oidcClaims.OrgScope = []string{activeOrg.ID}
			}
			oidcClaims.OrgID = activeOrg.ID
			if oidcClaims.JTI == "" {
				oidcClaims.JTI = uuid.NewString()
			}
			perms := oidcClaims.Permissions
			for _, p := range permissionsForRoles(oidcClaims.Roles) {
				if !slices.Contains(perms, p) {
					perms = append(perms, p)
				}
			}
			oidcClaims.Permissions = perms
			adminTier := httpx.AdminTierForRoles(oidcClaims.Roles)
			oidcClaims.AdminTier = adminTier
			now := time.Now().UTC()
			streamClaims := oidcClaims
			streamClaims.TokenUse = "stream"
			streamClaims.IssuedAt = now.Unix()
			streamClaims.ExpiresAt = now.Add(60 * time.Second).Unix()
			streamTicket, err := auth.SignHS256(streamClaims, rt.secret)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "TOKEN_SIGN_ERROR", err.Error(), "")
				return
			}
			refreshTok, _ := rt.sessions.IssueRefreshToken(r.Context(), oidcClaims, 7*24*time.Hour)
			setSessionCookies(w, r, rawOIDC, refreshTok, 28800)
			writeJSON(w, http.StatusOK, map[string]any{
				"token":        rawOIDC,
				"refreshToken": refreshTok,
				"streamTicket": streamTicket,
				"expiresIn":    28800,
				"user": userSummary{
					ID:          oidcClaims.Subject,
					Username:    oidcClaims.Username,
					FullName:    oidcClaims.FullName,
					Email:       oidcClaims.Email,
					ActiveOrgID: activeOrg.ID,
					AdminTier:   adminTier,
					Roles:       oidcClaims.Roles,
					Permissions: perms,
				},
				"activeOrg":     activeOrg,
				"organizations": filterOrgsForScope(oidcClaims.OrgScope, oidcClaims.Roles),
			})
			return
		}

		req.Username = strings.TrimSpace(req.Username)
		if req.Username == "" || strings.TrimSpace(req.Password) == "" {
			writeErr(w, http.StatusBadRequest, "VALIDATION_ERROR", "username and password are required", "")
			return
		}

		activeOrg := resolveOrg(req.OrganizationID, req.OrgCode)
		userID := defaultUID
		if req.Username != "admin.finance" && req.Username != "admin.ho" {
			userID = uuid.NewSHA1(uuid.NameSpaceOID, []byte("erp-user:"+req.Username)).String()
		}

		var (
			roles      []string
			orgScope   []string
			fullName   string
			email      string
			visibleOrg []orgSummary
		)
		if dyn, ok := resolveDynamicUser(r.Context(), req.Username, rt.secret, req.OrganizationID); ok {
			userID = dyn.UserID
			fullName = dyn.FullName
			email = dyn.Email
			roles = dyn.Roles
			orgScope = dyn.OrgScope
			activeOrg = dyn.ActiveOrg
			visibleOrg = dyn.Organizations
		} else {
			var defaultActiveOrg orgSummary
			roles, orgScope, defaultActiveOrg, fullName = resolveDevPersona(req.Username, activeOrg)
			if req.OrganizationID == "" && req.OrgCode == "" {
				activeOrg = defaultActiveOrg
			} else if !slices.Contains(orgScope, activeOrg.ID) {
				// Prevent a subsidiary/branch persona from selecting an org outside its hierarchy at login.
				if slices.Contains(roles, "SYSTEM_ADMIN") {
					orgScope = append([]string{activeOrg.ID}, orgScope...)
				} else {
					activeOrg = defaultActiveOrg
				}
			}
			email = req.Username
			if !strings.Contains(email, "@") {
				email += "@sinergi.example.com"
			}
			visibleOrg = filterOrgsForScope(orgScope, roles)
		}
		perms := permissionsForRoles(roles)
		adminTier := httpx.AdminTierForRoles(roles)

		now := time.Now().UTC()

		accessClaims := auth.Claims{
			JTI:         uuid.NewString(),
			Subject:     userID,
			Username:    req.Username,
			FullName:    fullName,
			Email:       email,
			OrgID:       activeOrg.ID,
			OrgScope:    orgScope,
			Roles:       roles,
			Permissions: perms,
			AdminTier:   adminTier,
			TokenUse:    "access",
			IssuedAt:    now.Unix(),
			ExpiresAt:   now.Add(8 * time.Hour).Unix(),
		}
		token, err := auth.SignHS256(accessClaims, rt.secret)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "TOKEN_SIGN_ERROR", err.Error(), "")
			return
		}

		streamClaims := accessClaims
		streamClaims.TokenUse = "stream"
		streamClaims.ExpiresAt = now.Add(60 * time.Second).Unix()
		streamTicket, err := auth.SignHS256(streamClaims, rt.secret)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "TOKEN_SIGN_ERROR", err.Error(), "")
			return
		}

		refreshTok, _ := rt.sessions.IssueRefreshToken(r.Context(), accessClaims, 7*24*time.Hour)
		setSessionCookies(w, r, token, refreshTok, 28800)

		writeJSON(w, http.StatusOK, map[string]any{
			"token":        token,
			"refreshToken": refreshTok,
			"streamTicket": streamTicket,
			"expiresIn":    28800,
			"user": userSummary{
				ID:          userID,
				Username:    req.Username,
				FullName:    fullName,
				Email:       email,
				ActiveOrgID: activeOrg.ID,
				AdminTier:   adminTier,
				Roles:       roles,
				Permissions: perms,
			},
			"activeOrg":     activeOrg,
			"organizations": visibleOrg,
		})
	}
}

func setSessionCookies(w http.ResponseWriter, r *http.Request, accessTok, refreshTok string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    accessTok,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookie(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
	if refreshTok != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     refreshCookie,
			Value:    refreshTok,
			Path:     "/api/v1/auth",
			HttpOnly: true,
			Secure:   isSecureCookie(r),
			SameSite: http.SameSiteLaxMode,
			MaxAge:   7 * 24 * 3600,
		})
	}
}

func handleRefresh(rt *authRuntime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refreshToken"`
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		rawRefresh := strings.TrimSpace(req.RefreshToken)
		if rawRefresh == "" {
			if c, err := r.Cookie(refreshCookie); err == nil {
				rawRefresh = strings.TrimSpace(c.Value)
			}
		}
		if rawRefresh == "" {
			writeErr(w, http.StatusUnauthorized, "REFRESH_TOKEN_REQUIRED", "Refresh token is required", "")
			return
		}
		baseClaims, nextRefresh, err := rt.sessions.RotateRefreshToken(r.Context(), rawRefresh, 7*24*time.Hour)
		if err != nil {
			code := "INVALID_REFRESH_TOKEN"
			if errors.Is(err, auth.ErrRefreshTokenReused) {
				code = "REFRESH_TOKEN_REUSED"
			}
			writeErr(w, http.StatusUnauthorized, code, err.Error(), "")
			return
		}
		now := time.Now().UTC()
		accessClaims := baseClaims
		accessClaims.JTI = uuid.NewString()
		accessClaims.TokenUse = "access"
		accessClaims.IssuedAt = now.Unix()
		accessClaims.ExpiresAt = now.Add(8 * time.Hour).Unix()
		token, err := auth.SignHS256(accessClaims, rt.secret)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "TOKEN_SIGN_ERROR", err.Error(), "")
			return
		}
		streamClaims := accessClaims
		streamClaims.TokenUse = "stream"
		streamClaims.ExpiresAt = now.Add(60 * time.Second).Unix()
		streamTicket, err := auth.SignHS256(streamClaims, rt.secret)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "TOKEN_SIGN_ERROR", err.Error(), "")
			return
		}
		setSessionCookies(w, r, token, nextRefresh, 28800)
		writeJSON(w, http.StatusOK, map[string]any{
			"token":        token,
			"refreshToken": nextRefresh,
			"streamTicket": streamTicket,
			"expiresIn":    28800,
		})
	}
}

func handleOpenAPISpec() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		candidates := []string{}
		if envPath := strings.TrimSpace(os.Getenv("OPENAPI_SPEC_PATH")); envPath != "" {
			candidates = append(candidates, envPath)
		}
		candidates = append(candidates,
			"docs/openapi/erp-api-v1.openapi.yaml",
			"../../docs/openapi/erp-api-v1.openapi.yaml",
			"../../../docs/openapi/erp-api-v1.openapi.yaml",
			"../../../../docs/openapi/erp-api-v1.openapi.yaml",
			"/app/docs/openapi/erp-api-v1.openapi.yaml",
		)
		for _, p := range candidates {
			if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
				w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
				w.Header().Set("Cache-Control", "public, max-age=300")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
				return
			}
		}
		writeErr(w, http.StatusNotFound, "OPENAPI_NOT_FOUND", "OpenAPI specification file not found", "")
	}
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

type dynamicUserSession struct {
	UserID        string
	FullName      string
	Email         string
	Roles         []string
	OrgScope      []string
	ActiveOrg     orgSummary
	Organizations []orgSummary
}

var internalHTTPClient = &http.Client{Timeout: 1500 * time.Millisecond}

func doInternalGET(ctx context.Context, rawURL string, scopeHeader string, secret []byte, dst any) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("X-Organization-Id", scopeHeader)
	req.Header.Set("X-Org-Scope", scopeHeader)
	req.Header.Set("X-User-Id", defaultUID)
	req.Header.Set("X-Roles", "SYSTEM_ADMIN")
	req.Header.Set("X-Admin-Tier", "0")
	req.Header.Set("X-Permissions", "user:read,security:read,org:read")
	httpx.SignGatewayRequest(req, secret)
	resp, err := internalHTTPClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return json.NewDecoder(resp.Body).Decode(dst) == nil
}

// resolveDynamicUser looks up a newly provisioned user from user-service, resolves role codes
// from security-service, and expands org subtrees via org-service so admin-created users can log in immediately.
func resolveDynamicUser(ctx context.Context, username string, secret []byte, requestedOrgID string) (dynamicUserSession, bool) {
	userSvcURL := strings.TrimRight(strings.TrimSpace(os.Getenv("USER_SERVICE_URL")), "/")
	if userSvcURL == "" {
		return dynamicUserSession{}, false
	}
	lookupScope := strings.Join([]string{defaultOrgID, branchOrgID, subLogOrgID, brcJktOrgID}, ",")
	if _, err := uuid.Parse(strings.TrimSpace(requestedOrgID)); err == nil && requestedOrgID != systemOrgID {
		lookupScope = requestedOrgID + "," + lookupScope
	}
	var uResp struct {
		ID                    string  `json:"id"`
		OrganizationID        string  `json:"organizationId"`
		Username              string  `json:"username"`
		Email                 string  `json:"email"`
		FullName              string  `json:"fullName"`
		Status                string  `json:"status"`
		DefaultOrganizationID *string `json:"defaultOrganizationId"`
		Roles                 []struct {
			OrganizationRefID string `json:"organizationRefId"`
			RoleID            string `json:"roleId"`
		} `json:"roles"`
	}
	if !doInternalGET(ctx, userSvcURL+"/users/lookup?username="+url.QueryEscape(username), lookupScope, secret, &uResp) {
		return dynamicUserSession{}, false
	}
	if uResp.ID == "" || !strings.EqualFold(uResp.Status, "Active") {
		return dynamicUserSession{}, false
	}

	// Resolve role UUIDs -> role codes via security-service.
	roleCodeByID := map[string]string{}
	if secURL := strings.TrimRight(strings.TrimSpace(os.Getenv("SECURITY_SERVICE_URL")), "/"); secURL != "" {
		var rList struct {
			Data []struct {
				ID   string `json:"id"`
				Code string `json:"code"`
			} `json:"data"`
		}
		if doInternalGET(ctx, secURL+"/roles?size=100", lookupScope+","+systemOrgID, secret, &rList) {
			for _, r := range rList.Data {
				roleCodeByID[r.ID] = r.Code
			}
		}
	}

	var roles []string
	var assignedOrgs []string
	for _, a := range uResp.Roles {
		if a.OrganizationRefID != "" && !slices.Contains(assignedOrgs, a.OrganizationRefID) {
			assignedOrgs = append(assignedOrgs, a.OrganizationRefID)
		}
		if code, ok := roleCodeByID[a.RoleID]; ok && code != "" && !slices.Contains(roles, code) {
			roles = append(roles, code)
		}
	}
	if len(roles) == 0 {
		roles = []string{"VIEWER"}
	}
	if len(assignedOrgs) == 0 {
		if uResp.DefaultOrganizationID != nil && *uResp.DefaultOrganizationID != "" {
			assignedOrgs = append(assignedOrgs, *uResp.DefaultOrganizationID)
		} else if uResp.OrganizationID != "" {
			assignedOrgs = append(assignedOrgs, uResp.OrganizationID)
		}
	}

	// Expand subtrees via org-service for tiered admins (HOLDING_ADMIN / SUBSIDIARY_ADMIN / BRANCH_ADMIN).
	orgScope := slices.Clone(assignedOrgs)
	orgSvcURL := strings.TrimRight(strings.TrimSpace(os.Getenv("ORG_SERVICE_URL")), "/")
	if orgSvcURL != "" && httpx.AdminTierForRoles(roles) <= httpx.AdminTierBranch {
		for _, oid := range assignedOrgs {
			var descResp struct {
				DescendantIDs []string `json:"descendantIds"`
			}
			if doInternalGET(ctx, orgSvcURL+"/organizations/"+oid+"/descendants", lookupScope, secret, &descResp) {
				for _, did := range descResp.DescendantIDs {
					if did != "" && did != systemOrgID && !slices.Contains(orgScope, did) {
						orgScope = append(orgScope, did)
					}
				}
			}
		}
	}

	// Build visible organization summaries from org-service (fallback to devOrganizations).
	var visibleOrgs []orgSummary
	if orgSvcURL != "" && len(orgScope) > 0 {
		var oList struct {
			Data []struct {
				ID       string  `json:"id"`
				Code     string  `json:"code"`
				Name     string  `json:"name"`
				ParentID *string `json:"parentId"`
			} `json:"data"`
		}
		if doInternalGET(ctx, orgSvcURL+"/organizations?size=100", strings.Join(orgScope, ","), secret, &oList) {
			for _, o := range oList.Data {
				if slices.Contains(orgScope, o.ID) {
					visibleOrgs = append(visibleOrgs, orgSummary{
						ID: o.ID, Code: o.Code, Name: o.Name, Type: "COMPANY", ParentID: o.ParentID, FunctionalCurrency: "IDR",
					})
				}
			}
		}
	}
	if len(visibleOrgs) == 0 {
		visibleOrgs = filterOrgsForScope(orgScope, roles)
	}

	activeID := ""
	if uResp.DefaultOrganizationID != nil && slices.Contains(orgScope, *uResp.DefaultOrganizationID) {
		activeID = *uResp.DefaultOrganizationID
	} else if len(orgScope) > 0 {
		activeID = orgScope[0]
	}
	activeOrg := resolveOrg(activeID, "")
	for _, o := range visibleOrgs {
		if o.ID == activeID {
			activeOrg = o
			break
		}
	}
	return dynamicUserSession{
		UserID:        uResp.ID,
		FullName:      uResp.FullName,
		Email:         uResp.Email,
		Roles:         roles,
		OrgScope:      orgScope,
		ActiveOrg:     activeOrg,
		Organizations: visibleOrgs,
	}, true
}

func resolveDevPersona(username string, requested orgSummary) (roles []string, orgScope []string, defaultOrg orgSummary, fullName string) {
	u := strings.ToLower(strings.TrimSpace(username))
	switch u {
	case "admin.finance", "admin.system":
		return []string{"SYSTEM_ADMIN", "HOLDING_ADMIN", "FINANCE_MANAGER", "ACCOUNTANT"},
			[]string{defaultOrgID, branchOrgID, subLogOrgID, brcJktOrgID},
			devOrganizations[0],
			"Chief Financial Officer (Head Office)"
	case "admin.ho":
		return []string{"HOLDING_ADMIN", "FINANCE_MANAGER"},
			[]string{defaultOrgID, branchOrgID, subLogOrgID, brcJktOrgID},
			devOrganizations[0],
			"Admin Utama Head Office (Holding)"
	case "admin.properti", "admin.sub":
		return []string{"SUBSIDIARY_ADMIN"},
			[]string{branchOrgID, brcJktOrgID},
			devOrganizations[1],
			"Admin Anak Perusahaan (Properti)"
	case "admin.logistik":
		return []string{"SUBSIDIARY_ADMIN"},
			[]string{subLogOrgID},
			devOrganizations[2],
			"Admin Anak Perusahaan (Logistik)"
	case "admin.cabang", "admin.jkt":
		return []string{"BRANCH_ADMIN"},
			[]string{brcJktOrgID},
			devOrganizations[3],
			"Admin Cabang Jakarta Selatan"
	case "finance.mgr":
		return []string{"FINANCE_MANAGER"},
			[]string{defaultOrgID, branchOrgID},
			devOrganizations[0],
			"Kepala Divisi Finance & Accounting"
	case "tax.treasury":
		return []string{"TAX_TREASURY_OFFICER"},
			[]string{defaultOrgID},
			devOrganizations[0],
			"Tax & Treasury Officer"
	case "procurement.mgr":
		return []string{"PROCUREMENT_MANAGER"},
			[]string{branchOrgID},
			devOrganizations[1],
			"Kepala Divisi Procurement"
	case "warehouse.mgr":
		return []string{"WAREHOUSE_MANAGER"},
			[]string{subLogOrgID},
			devOrganizations[2],
			"Kepala Divisi Warehouse & Logistik"
	case "sales.mgr":
		return []string{"SALES_MANAGER"},
			[]string{branchOrgID},
			devOrganizations[1],
			"Kepala Divisi Sales & CRM"
	case "ops.mgr":
		return []string{"OPERATIONS_MANAGER"},
			[]string{branchOrgID, brcJktOrgID},
			devOrganizations[1],
			"Kepala Divisi Operations & Proyek"
	case "hr.mgr":
		return []string{"HR_MANAGER"},
			[]string{defaultOrgID},
			devOrganizations[0],
			"Kepala Divisi HR & Payroll"
	case "auditor.it", "auditor.internal":
		return []string{"AUDITOR"},
			[]string{defaultOrgID, branchOrgID, subLogOrgID, brcJktOrgID},
			devOrganizations[0],
			"Internal Auditor & IT Governance"
	default:
		return []string{"ACCOUNTANT"},
			[]string{requested.ID},
			requested,
			username
	}
}

func filterOrgsForScope(orgScope, roles []string) []orgSummary {
	if slices.Contains(roles, "SYSTEM_ADMIN") {
		return devOrganizations
	}
	out := make([]orgSummary, 0, len(orgScope))
	for _, o := range devOrganizations {
		if slices.Contains(orgScope, o.ID) {
			out = append(out, o)
		}
	}
	if len(out) == 0 {
		return []orgSummary{devOrganizations[0]}
	}
	return out
}

func handleLogout(rt *authRuntime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if claims, err := extractClaimsWithRuntime(r, rt, "access"); err == nil && claims.JTI != "" {
			ttl := time.Until(time.Unix(claims.ExpiresAt, 0))
			if ttl < time.Minute {
				ttl = time.Minute
			}
			_ = rt.sessions.RevokeJTI(r.Context(), claims.JTI, ttl)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookie,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   isSecureCookie(r),
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     refreshCookie,
			Value:    "",
			Path:     "/api/v1/auth",
			HttpOnly: true,
			Secure:   isSecureCookie(r),
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		w.WriteHeader(http.StatusNoContent)
	}
}

func isSecureCookie(r *http.Request) bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SECURE")), "true") {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SECURE")), "false") {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		return true
	}
	if r != nil {
		if r.TLS != nil || strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https") {
			return true
		}
	}
	return false
}

func handleMe(rt *authRuntime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := extractClaimsWithRuntime(r, rt, "access")
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "Valid Bearer token or session required", "")
			return
		}
		activeOrg := resolveOrg(claims.OrgID, "")
		writeJSON(w, http.StatusOK, map[string]any{
			"user": userSummary{
				ID:          claims.Subject,
				Username:    claims.Username,
				FullName:    claims.FullName,
				Email:       claims.Email,
				ActiveOrgID: claims.OrgID,
				AdminTier:   httpx.AdminTierForRoles(claims.Roles),
				Roles:       claims.Roles,
				Permissions: claims.Permissions,
			},
			"activeOrg":     activeOrg,
			"organizations": filterOrgsForScope(claims.OrgScope, claims.Roles),
		})
	}
}

func handleStreamTicket(rt *authRuntime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := extractClaimsWithRuntime(r, rt, "access")
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "Valid Bearer token or session required", "")
			return
		}
		now := time.Now().UTC()
		ticketClaims := claims
		ticketClaims.TokenUse = "stream"
		ticketClaims.IssuedAt = now.Unix()
		ticketClaims.ExpiresAt = now.Add(60 * time.Second).Unix()
		ticket, err := auth.SignHS256(ticketClaims, rt.secret)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "TOKEN_SIGN_ERROR", err.Error(), "")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ticket":    ticket,
			"expiresIn": 60,
		})
	}
}

func handleSSE(hub *sse.Hub, rt *authRuntime, allowAnon bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeErr(w, http.StatusInternalServerError, "SSE_UNSUPPORTED", "Streaming unsupported", "")
			return
		}

		var orgID string
		var claims auth.Claims
		var err error

		if ticket := strings.TrimSpace(r.URL.Query().Get("ticket")); ticket != "" {
			claims, err = auth.VerifyHS256(ticket, rt.secret, time.Now().UTC())
			if err == nil && claims.TokenUse != "stream" {
				err = auth.ErrInvalidToken
			}
		} else {
			claims, err = extractClaimsWithRuntime(r, rt, "access")
		}

		if err != nil {
			if !allowAnon {
				writeErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "Valid stream ticket, Bearer token, or session required", "")
				return
			}
			orgID = validUUIDOrDefault(r.URL.Query().Get("orgId"), defaultOrgID)
			if orgID == systemOrgID {
				orgID = defaultOrgID
			}
		} else {
			orgID = strings.TrimSpace(r.URL.Query().Get("orgId"))
			if orgID == "" {
				orgID = claims.OrgID
			}
			if _, perr := uuid.Parse(orgID); perr != nil {
				writeErr(w, http.StatusBadRequest, "INVALID_ORG_ID", "orgId must be a valid UUID", "")
				return
			}
			if orgID == systemOrgID {
				writeErr(w, http.StatusForbidden, "ORG_SCOPE_DENIED", "System organization cannot be selected as active tenant", "")
				return
			}
			if !slices.Contains(claims.OrgScope, orgID) && !slices.Contains(claims.Roles, "SYSTEM_ADMIN") {
				writeErr(w, http.StatusForbidden, "ORG_SCOPE_DENIED", "Requested organization is outside caller scope", "")
				return
			}
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		sub, cancel := hub.Subscribe(orgID)
		defer cancel()

		_, _ = fmt.Fprintf(w, "event: connected\ndata: %s\n\n", mustMarshalJSON(map[string]string{
			"organizationId": orgID,
			"status":         "connected",
		}))
		flusher.Flush()

		keepalive := time.NewTicker(20 * time.Second)
		defer keepalive.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case ev, ok := <-sub.Ch:
				if !ok {
					return
				}
				payload, err := json.Marshal(ev)
				if err != nil {
					continue
				}
				_, _ = fmt.Fprintf(w, "id: %s\ndata: %s\n\n", ev.EventID, payload)
				flusher.Flush()
			case <-keepalive.C:
				_, _ = fmt.Fprint(w, ": keepalive\n\n")
				flusher.Flush()
			}
		}
	}
}

func extractClaims(r *http.Request, secret []byte, expectedUse string) (auth.Claims, error) {
	return extractClaimsWithRuntime(r, &authRuntime{secret: secret, devAuth: true}, expectedUse)
}

func extractClaimsWithRuntime(r *http.Request, rt *authRuntime, expectedUse string) (auth.Claims, error) {
	var raw string
	if hdr := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(hdr, "Bearer ") {
		raw = strings.TrimSpace(strings.TrimPrefix(hdr, "Bearer "))
	} else if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		raw = c.Value
	}
	if raw == "" {
		return auth.Claims{}, auth.ErrInvalidToken
	}

	now := time.Now().UTC()
	var (
		claims auth.Claims
		err    error
	)
	if !rt.devAuth && expectedUse != "stream" && rt.oidc != nil {
		claims, err = rt.oidc.Verify(r.Context(), raw, now)
		if err != nil {
			// Also allow gateway-issued HS256 access tokens produced by /auth/refresh.
			if hsClaims, hsErr := auth.VerifyHS256(raw, rt.secret, now); hsErr == nil {
				claims, err = hsClaims, nil
			}
		}
	} else {
		claims, err = auth.VerifyHS256(raw, rt.secret, now)
		if err != nil && rt.oidc != nil && expectedUse != "stream" {
			claims, err = rt.oidc.Verify(r.Context(), raw, now)
		}
	}
	if err != nil {
		return auth.Claims{}, err
	}
	if expectedUse != "" && claims.TokenUse != expectedUse {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	if rt.sessions != nil && claims.JTI != "" && rt.sessions.IsRevoked(r.Context(), claims.JTI) {
		return auth.Claims{}, auth.ErrTokenRevoked
	}
	return claims, nil
}

func isAllowedChromeExtensionOrigin(origin string, allowAll bool, allowedExtIDs []string) bool {
	if !strings.HasPrefix(origin, "chrome-extension://") {
		return false
	}
	extID := strings.TrimSpace(strings.TrimPrefix(origin, "chrome-extension://"))
	extID = strings.TrimSuffix(extID, "/")
	if len(allowedExtIDs) > 0 {
		return slices.Contains(allowedExtIDs, extID)
	}
	return allowAll && extID != ""
}

func withCORS(allowedOrigins []string, allowChromeExt bool, extraExtIDs ...[]string) func(http.Handler) http.Handler {
	var allowedExtIDs []string
	if len(extraExtIDs) > 0 {
		allowedExtIDs = extraExtIDs[0]
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				allowed := slices.Contains(allowedOrigins, origin) ||
					isAllowedChromeExtensionOrigin(origin, allowChromeExt, allowedExtIDs)
				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Organization-Id, X-Correlation-Id, Idempotency-Key, If-Match")
					w.Header().Set("Access-Control-Expose-Headers", "ETag, X-Correlation-Id")
				}
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				cid := strings.TrimSpace(r.Header.Get("X-Correlation-Id"))
				log.Printf("[gateway-panic] cid=%s panic=%v", cid, rec)
				writeErr(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", cid)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type rateBucket struct {
	tokens float64
	last   time.Time
}

func withRateLimit(rps, burst int, extra ...*auth.SessionStore) func(http.Handler) http.Handler {
	var sessions *auth.SessionStore
	if len(extra) > 0 {
		sessions = extra[0]
	}
	if rps <= 0 {
		rps = 200
	}
	if burst < rps {
		burst = rps * 2
	}
	var (
		mu      sync.Mutex
		buckets = make(map[string]*rateBucket)
	)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}
			clientIP := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
			if idx := strings.IndexByte(clientIP, ','); idx > 0 {
				clientIP = strings.TrimSpace(clientIP[:idx])
			}
			if clientIP == "" {
				host, _, err := net.SplitHostPort(r.RemoteAddr)
				if err == nil && host != "" {
					clientIP = host
				} else {
					clientIP = r.RemoteAddr
				}
			}
			if sessions != nil {
				if allowed, usedRedis := sessions.AllowRateLimit(r.Context(), clientIP, burst, time.Second); usedRedis {
					if !allowed {
						w.Header().Set("Retry-After", "1")
						writeErr(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests; please retry shortly", r.Header.Get("X-Correlation-Id"))
						return
					}
					next.ServeHTTP(w, r)
					return
				}
			}
			now := time.Now()
			mu.Lock()
			if len(buckets) > 10000 {
				for k, b := range buckets {
					if now.Sub(b.last) > 2*time.Minute {
						delete(buckets, k)
					}
				}
			}
			b, ok := buckets[clientIP]
			if !ok {
				b = &rateBucket{tokens: float64(burst), last: now}
				buckets[clientIP] = b
			} else {
				elapsed := now.Sub(b.last).Seconds()
				b.tokens += elapsed * float64(rps)
				if b.tokens > float64(burst) {
					b.tokens = float64(burst)
				}
				b.last = now
			}
			allowed := false
			if b.tokens >= 1.0 {
				b.tokens -= 1.0
				allowed = true
			}
			mu.Unlock()
			if !allowed {
				w.Header().Set("Retry-After", "1")
				writeErr(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests; please retry shortly", r.Header.Get("X-Correlation-Id"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func resolveOrg(orgID, orgCode string) orgSummary {
	for _, o := range devOrganizations {
		if orgID != "" && strings.EqualFold(o.ID, orgID) {
			return o
		}
		if orgCode != "" && strings.EqualFold(o.Code, orgCode) {
			return o
		}
	}
	if _, err := uuid.Parse(orgID); err == nil && orgID != systemOrgID {
		code := orgCode
		if code == "" {
			code = "CUSTOM"
		}
		return orgSummary{
			ID:                 orgID,
			Code:               code,
			Name:               "Organization " + code,
			Type:               "COMPANY",
			FunctionalCurrency: "IDR",
		}
	}
	return devOrganizations[0]
}

func validUUIDOrDefault(v, d string) string {
	v = strings.TrimSpace(v)
	if _, err := uuid.Parse(v); err == nil {
		return v
	}
	return d
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg, corrID string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":          code,
			"message":       msg,
			"correlationId": corrID,
		},
	})
}

func mustMarshalJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func strPtr(s string) *string { return &s }
