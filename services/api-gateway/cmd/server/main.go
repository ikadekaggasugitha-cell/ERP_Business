package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/api-gateway/internal/auth"
	"github.com/sinergiinvestasiproperti/erp/services/api-gateway/internal/proxy"
	"github.com/sinergiinvestasiproperti/erp/services/api-gateway/internal/sse"
)

const (
	systemOrgID  = "00000000-0000-0000-0000-000000000000"
	defaultOrgID = "11111111-1111-1111-1111-111111111111"
	branchOrgID  = "22222222-2222-2222-2222-222222222222"
	defaultUID   = "00000000-0000-0000-0000-000000000101"
	sessionCookie = "erp_session"
)

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
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

var devOrganizations = []orgSummary{
	{
		ID:                 defaultOrgID,
		Code:               "HLD-01",
		Name:               "PT Sinergi Investasi Properti (Holding)",
		Type:               "HOLDING",
		ParentID:           nil,
		FunctionalCurrency: "IDR",
	},
	{
		ID:                 branchOrgID,
		Code:               "BRC-JKT",
		Name:               "PT Sinergi Properti Jakarta (Branch)",
		Type:               "BRANCH",
		ParentID:           strPtr(defaultOrgID),
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
		"ap:read", "ap:write",
		"ar:read", "ar:write",
		"coa:read", "coa:write",
		"tax:read", "tax:write",
		"currency:read", "currency:write",
		"treasury:read", "treasury:write",
		"workflow:read", "workflow:write", "workflow:approve",
		"inventory:read", "inventory:write",
		"procurement:read", "procurement:write",
		"sales:read", "sales:write",
		"manufacturing:read", "manufacturing:write",
		"hr:read", "hr:write",
		"payroll:read", "payroll:write",
		"masterdata:read", "masterdata:write",
		"org:read", "org:write",
		"user:read", "user:write",
		"security:read", "security:write",
		"audit:read",
		"dimension:read", "dimension:write", "localization:read", "localization:write",
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
		"ap:read", "ap:write",
		"ar:read", "ar:write",
		"coa:read", "coa:write",
		"tax:read", "tax:write",
		"currency:read", "currency:write",
		"treasury:read", "treasury:write",
		"workflow:read", "workflow:write", "workflow:approve",
		"inventory:read",
		"procurement:read",
		"sales:read",
		"manufacturing:read",
		"payroll:read", "payroll:write",
		"masterdata:read", "masterdata:write",
		"org:read",
		"audit:read",
		"dimension:read", "dimension:write", "localization:read",
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
	},
	"PROCUREMENT_MANAGER": {
		"procurement.create", "procurement.read", "procurement.update", "procurement.approve",
		"inventory.create", "inventory.read", "inventory.update",
		"ap.read", "org.read",
		"procurement:read", "procurement:write",
		"inventory:read", "inventory:write",
		"ap:read",
		"masterdata:read", "masterdata:write",
		"workflow:read", "workflow:write", "workflow:approve",
		"org:read",
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
	},
	"HR_MANAGER": {
		"hr:read", "hr:write",
		"payroll:read", "payroll:write",
		"workflow:read", "workflow:write", "workflow:approve",
		"org.read", "org:read",
		"user:read",
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

func main() {
	addr := getenv("HTTP_ADDR", ":8000")
	secretStr := getenv("JWT_SECRET", "dev-erp-jwt-secret-change-in-prod")
	if secretStr == "dev-erp-jwt-secret-change-in-prod" {
		log.Println("WARN: using default dev JWT_SECRET; set JWT_SECRET in production")
	}
	jwtSecret := []byte(secretStr)
	devAuthMode := strings.EqualFold(getenv("DEV_AUTH_MODE", "true"), "true")
	allowAnon := strings.EqualFold(getenv("DEV_AUTH_ALLOW_ANON", "false"), "true")
	brokers := splitCSV(os.Getenv("KAFKA_BROKERS"))
	allowedOrigins := splitCSV(getenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000,http://localhost:5173"))
	allowChromeExt := strings.EqualFold(getenv("CORS_ALLOW_CHROME_EXTENSIONS", "true"), "true")

	router, err := proxy.NewRouter()
	if err != nil {
		log.Fatalf("gateway router: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := sse.NewHub()
	go hub.StartKafkaBridge(ctx, brokers, getenv("KAFKA_GROUP_ID", "api-gateway-sse"))

	mux := newGatewayMux(router, hub, jwtSecret, devAuthMode, allowAnon)
	handler := withCORS(allowedOrigins, allowChromeExt)(mux)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // SSE connections are long-lived
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("api-gateway listening on %s (devAuth=%v, kafka=%v)", addr, devAuthMode, brokers)
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
	log.Println("api-gateway stopped")
}

func newGatewayMux(router *proxy.Router, hub *sse.Hub, jwtSecret []byte, devAuthMode, allowAnon bool) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "api-gateway"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready", "service": "api-gateway"})
	})

	mux.HandleFunc("POST /api/v1/auth/login", handleLogin(jwtSecret, devAuthMode))
	mux.HandleFunc("POST /api/v1/auth/logout", handleLogout())
	mux.HandleFunc("GET /api/v1/auth/me", handleMe(jwtSecret))
	mux.HandleFunc("POST /api/v1/auth/stream-ticket", handleStreamTicket(jwtSecret))
	mux.HandleFunc("GET /api/v1/stream/events", handleSSE(hub, jwtSecret, allowAnon))

	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
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

		corrID := strings.TrimSpace(r.Header.Get("X-Correlation-Id"))
		if _, err := uuid.Parse(corrID); err != nil {
			corrID = uuid.NewString()
			r.Header.Set("X-Correlation-Id", corrID)
		}

		claims, err := extractClaims(r, jwtSecret, "access")
		if err != nil {
			if !allowAnon {
				writeErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "Valid Bearer token or session required", corrID)
				return
			}
			// Explicit dev-only fallback when DEV_AUTH_ALLOW_ANON=true.
			orgID := validUUIDOrDefault(r.Header.Get("X-Organization-Id"), defaultOrgID)
			scope := orgID + "," + systemOrgID
			r.Header.Set("X-Organization-Id", scope)
			r.Header.Set("X-Org-Scope", scope)
			r.Header.Set("X-User-Id", defaultUID)
			r.Header.Set("X-Roles", "SYSTEM_ADMIN")
			r.Header.Set("X-Permissions", strings.Join(permissionsForRoles([]string{"SYSTEM_ADMIN"}), ","))
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
		if !slices.Contains(claims.OrgScope, requestedOrg) && !slices.Contains(claims.Roles, "SYSTEM_ADMIN") {
			writeErr(w, http.StatusForbidden, "ORG_SCOPE_DENIED", "Requested organization is outside caller scope", corrID)
			return
		}

		scopeList := []string{requestedOrg}
		for _, id := range claims.OrgScope {
			if id != requestedOrg {
				scopeList = append(scopeList, id)
			}
		}
		if !slices.Contains(scopeList, systemOrgID) {
			scopeList = append(scopeList, systemOrgID)
		}

		effectivePerms := slices.Clone(claims.Permissions)
		for _, rpPerm := range permissionsForRoles(claims.Roles) {
			if !slices.Contains(effectivePerms, rpPerm) {
				effectivePerms = append(effectivePerms, rpPerm)
			}
		}

		r.Header.Set("X-Organization-Id", strings.Join(scopeList, ","))
		r.Header.Set("X-Org-Scope", strings.Join(scopeList, ","))
		r.Header.Set("X-User-Id", claims.Subject)
		r.Header.Set("X-Roles", strings.Join(claims.Roles, ","))
		r.Header.Set("X-Permissions", strings.Join(effectivePerms, ","))
		rp.ServeHTTP(w, r)
	})
	return mux
}

func handleLogin(secret []byte, devAuthMode bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			OrgCode        string `json:"orgCode"`
			OrganizationID string `json:"organizationId"`
			Username       string `json:"username"`
			Password       string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "INVALID_JSON", "Malformed JSON request", "")
			return
		}
		req.Username = strings.TrimSpace(req.Username)
		if req.Username == "" || strings.TrimSpace(req.Password) == "" {
			writeErr(w, http.StatusBadRequest, "VALIDATION_ERROR", "username and password are required", "")
			return
		}
		if !devAuthMode {
			writeErr(w, http.StatusNotImplemented, "OIDC_REQUIRED", "DEV_AUTH_MODE is disabled; configure external OIDC provider", "")
			return
		}

		activeOrg := resolveOrg(req.OrganizationID, req.OrgCode)
		userID := defaultUID
		if req.Username != "admin.finance" {
			userID = uuid.NewSHA1(uuid.NameSpaceOID, []byte("erp-user:"+req.Username)).String()
		}

		roles := []string{"SYSTEM_ADMIN", "FINANCE_MANAGER", "ACCOUNTANT"}
		perms := permissionsForRoles(roles)
		orgScope := []string{defaultOrgID, branchOrgID, systemOrgID}
		if !slices.Contains(orgScope, activeOrg.ID) {
			orgScope = append([]string{activeOrg.ID}, orgScope...)
		}

		now := time.Now().UTC()
		fullName := "Chief Financial Officer"
		if req.Username != "admin.finance" {
			fullName = req.Username
		}
		email := req.Username
		if !strings.Contains(email, "@") {
			email += "@sinergi.example.com"
		}

		accessClaims := auth.Claims{
			Subject:     userID,
			Username:    req.Username,
			FullName:    fullName,
			Email:       email,
			OrgID:       activeOrg.ID,
			OrgScope:    orgScope,
			Roles:       roles,
			Permissions: perms,
			TokenUse:    "access",
			IssuedAt:    now.Unix(),
			ExpiresAt:   now.Add(8 * time.Hour).Unix(),
		}
		token, err := auth.SignHS256(accessClaims, secret)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "TOKEN_SIGN_ERROR", err.Error(), "")
			return
		}

		streamClaims := accessClaims
		streamClaims.TokenUse = "stream"
		streamClaims.ExpiresAt = now.Add(60 * time.Second).Unix()
		streamTicket, err := auth.SignHS256(streamClaims, secret)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "TOKEN_SIGN_ERROR", err.Error(), "")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookie,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   28800,
		})

		writeJSON(w, http.StatusOK, map[string]any{
			"token":        token,
			"streamTicket": streamTicket,
			"expiresIn":    28800,
			"user": userSummary{
				ID:          userID,
				Username:    req.Username,
				FullName:    fullName,
				Email:       email,
				ActiveOrgID: activeOrg.ID,
				Roles:       roles,
				Permissions: perms,
			},
			"activeOrg":     activeOrg,
			"organizations": devOrganizations,
		})
	}
}

func handleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookie,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleMe(secret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := extractClaims(r, secret, "access")
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
				Roles:       claims.Roles,
				Permissions: claims.Permissions,
			},
			"activeOrg":     activeOrg,
			"organizations": devOrganizations,
		})
	}
}

func handleStreamTicket(secret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := extractClaims(r, secret, "access")
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "Valid Bearer token or session required", "")
			return
		}
		now := time.Now().UTC()
		ticketClaims := claims
		ticketClaims.TokenUse = "stream"
		ticketClaims.IssuedAt = now.Unix()
		ticketClaims.ExpiresAt = now.Add(60 * time.Second).Unix()
		ticket, err := auth.SignHS256(ticketClaims, secret)
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

func handleSSE(hub *sse.Hub, secret []byte, allowAnon bool) http.HandlerFunc {
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
			claims, err = auth.VerifyHS256(ticket, secret, time.Now().UTC())
			if err == nil && claims.TokenUse != "stream" {
				err = auth.ErrInvalidToken
			}
		} else {
			claims, err = extractClaims(r, secret, "access")
		}

		if err != nil {
			if !allowAnon {
				writeErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "Valid stream ticket, Bearer token, or session required", "")
				return
			}
			orgID = validUUIDOrDefault(r.URL.Query().Get("orgId"), defaultOrgID)
		} else {
			orgID = strings.TrimSpace(r.URL.Query().Get("orgId"))
			if orgID == "" {
				orgID = claims.OrgID
			}
			if _, perr := uuid.Parse(orgID); perr != nil {
				writeErr(w, http.StatusBadRequest, "INVALID_ORG_ID", "orgId must be a valid UUID", "")
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
	var raw string
	if hdr := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(hdr, "Bearer ") {
		raw = strings.TrimSpace(strings.TrimPrefix(hdr, "Bearer "))
	} else if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		raw = c.Value
	}
	if raw == "" {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	claims, err := auth.VerifyHS256(raw, secret, time.Now().UTC())
	if err != nil {
		return auth.Claims{}, err
	}
	if expectedUse != "" && claims.TokenUse != "" && claims.TokenUse != expectedUse {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	return claims, nil
}

func withCORS(allowedOrigins []string, allowChromeExt bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				allowed := slices.Contains(allowedOrigins, origin) ||
					(allowChromeExt && strings.HasPrefix(origin, "chrome-extension://"))
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

func resolveOrg(orgID, orgCode string) orgSummary {
	for _, o := range devOrganizations {
		if orgID != "" && strings.EqualFold(o.ID, orgID) {
			return o
		}
		if orgCode != "" && strings.EqualFold(o.Code, orgCode) {
			return o
		}
	}
	if _, err := uuid.Parse(orgID); err == nil {
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
