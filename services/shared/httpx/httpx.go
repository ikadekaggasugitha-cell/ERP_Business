// Package httpx holds the SHARED HTTP layer helpers (SDD BAB 4): correlation +
// org-scope middleware, JSON envelope, the standard error model, pagination,
// If-Match parsing, and a dev-grade idempotency store.
package httpx

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// maxBodyBytes caps request body size to avoid unbounded decode (DoS guard).
const maxBodyBytes = 1 << 20 // 1 MiB

// DecodeJSON safely decodes a single JSON object from r.Body, size-capped,
// rejecting unknown fields and trailing junk tokens.
func DecodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return &APIError{Status: http.StatusRequestEntityTooLarge, Code: "PAYLOAD_TOO_LARGE", Message: "request body exceeds 1 MiB limit"}
		}
		return BadRequest("body JSON tidak valid")
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return BadRequest("body JSON mengandung data tambahan yang tidak valid")
	}
	return nil
}

// DecodeOptionalJSON decodes an optional JSON object body safely: empty bodies
// leave dst untouched and return nil, while non-empty bodies follow DecodeJSON rules.
func DecodeOptionalJSON(w http.ResponseWriter, r *http.Request, v any) error {
	if r.Body == nil {
		return nil
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return &APIError{Status: http.StatusRequestEntityTooLarge, Code: "PAYLOAD_TOO_LARGE", Message: "request body exceeds 1 MiB limit"}
		}
		return BadRequest("gagal membaca body JSON")
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return BadRequest("body JSON tidak valid")
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return BadRequest("body JSON mengandung data tambahan yang tidak valid")
	}
	return nil
}

type ctxKey int

const (
	correlationKey ctxKey = iota
	scopeKey
	userKey
	rolesKey
	permsKey
)

// ---- context accessors -----------------------------------------------------

func Correlation(ctx context.Context) string {
	if v, ok := ctx.Value(correlationKey).(string); ok {
		return v
	}
	return ""
}

func Scope(ctx context.Context) []uuid.UUID {
	if v, ok := ctx.Value(scopeKey).([]uuid.UUID); ok {
		return v
	}
	return nil
}

func ActiveOrg(ctx context.Context) uuid.UUID {
	s := Scope(ctx)
	if len(s) > 0 {
		return s[0]
	}
	return uuid.Nil
}

func User(ctx context.Context) uuid.UUID {
	if v, ok := ctx.Value(userKey).(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}

func Roles(ctx context.Context) []string {
	if v, ok := ctx.Value(rolesKey).([]string); ok {
		return v
	}
	return nil
}

func Permissions(ctx context.Context) []string {
	if v, ok := ctx.Value(permsKey).([]string); ok {
		return v
	}
	return nil
}

func HasRole(ctx context.Context, role string) bool {
	return slices.Contains(Roles(ctx), role)
}

func HasPermission(ctx context.Context, perm string) bool {
	if HasRole(ctx, "SYSTEM_ADMIN") {
		return true
	}
	return slices.Contains(Permissions(ctx), perm)
}

// ---- gateway HMAC trust boundary -------------------------------------------

const defaultDevGatewaySecret = "dev-erp-jwt-secret-change-me"

// GatewaySigningSecret returns the shared HMAC secret used to sign/verify
// X-Gateway-Signature between api-gateway and downstream microservices.
func GatewaySigningSecret() []byte {
	if s := strings.TrimSpace(os.Getenv("GATEWAY_HMAC_SECRET")); s != "" {
		return []byte(s)
	}
	if s := strings.TrimSpace(os.Getenv("JWT_SECRET")); s != "" {
		return []byte(s)
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		return nil
	}
	return []byte(defaultDevGatewaySecret)
}

// ComputeGatewaySignature computes HMAC-SHA256 over the canonical gateway identity headers.
func ComputeGatewaySignature(secret []byte, ts, userID, orgID, orgScope, roles, perms string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(ts))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(userID))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(orgID))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(orgScope))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(roles))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(perms))
	return hex.EncodeToString(mac.Sum(nil))
}

// SignGatewayRequest attaches X-Gateway-Timestamp and X-Gateway-Signature to an outbound proxy request.
func SignGatewayRequest(r *http.Request, secret []byte) {
	if len(secret) == 0 {
		secret = GatewaySigningSecret()
	}
	if len(secret) == 0 {
		return
	}
	ts := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	sig := ComputeGatewaySignature(
		secret,
		ts,
		strings.TrimSpace(r.Header.Get("X-User-Id")),
		strings.TrimSpace(r.Header.Get("X-Organization-Id")),
		strings.TrimSpace(r.Header.Get("X-Org-Scope")),
		strings.TrimSpace(r.Header.Get("X-Roles")),
		strings.TrimSpace(r.Header.Get("X-Permissions")),
	)
	r.Header.Set("X-Gateway-Timestamp", ts)
	r.Header.Set("X-Gateway-Signature", sig)
}

func verifyGatewayHeaders(r *http.Request) bool {
	sig := strings.TrimSpace(r.Header.Get("X-Gateway-Signature"))
	tsRaw := strings.TrimSpace(r.Header.Get("X-Gateway-Timestamp"))
	if sig != "" {
		secret := GatewaySigningSecret()
		if len(secret) == 0 || tsRaw == "" {
			return false
		}
		tsUnix, err := strconv.ParseInt(tsRaw, 10, 64)
		if err != nil {
			return false
		}
		skew := time.Since(time.Unix(tsUnix, 0))
		if skew < -5*time.Minute || skew > 5*time.Minute {
			return false
		}
		expected := ComputeGatewaySignature(
			secret,
			tsRaw,
			strings.TrimSpace(r.Header.Get("X-User-Id")),
			strings.TrimSpace(r.Header.Get("X-Organization-Id")),
			strings.TrimSpace(r.Header.Get("X-Org-Scope")),
			strings.TrimSpace(r.Header.Get("X-Roles")),
			strings.TrimSpace(r.Header.Get("X-Permissions")),
		)
		return hmac.Equal([]byte(sig), []byte(expected))
	}
	// When no X-Gateway-Signature is sent:
	// - In production (or when REQUIRE_GATEWAY_SIGNATURE=true), reject unsigned headers unless TRUST_INTERNAL_DEV=true.
	// - In local dev/test, allow unsigned internal headers unless TRUST_INTERNAL_DEV=false or REQUIRE_GATEWAY_SIGNATURE=true.
	if strings.EqualFold(strings.TrimSpace(os.Getenv("REQUIRE_GATEWAY_SIGNATURE")), "true") {
		return false
	}
	if v := strings.TrimSpace(os.Getenv("TRUST_INTERNAL_DEV")); v != "" {
		return strings.EqualFold(v, "true") || v == "1"
	}
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")
}

// ---- middleware ------------------------------------------------------------

// Middleware injects correlation id, org scope, acting user, roles, and
// permissions into the request context, echoes X-Correlation-Id, and recovers panics.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cid := strings.TrimSpace(r.Header.Get("X-Correlation-Id"))
		if cid == "" {
			cid = uuid.NewString()
		}
		w.Header().Set("X-Correlation-Id", cid)

		ctx := context.WithValue(r.Context(), correlationKey, cid)
		if verifyGatewayHeaders(r) {
			scope := mergeScopes(parseScope(r.Header.Get("X-Organization-Id")), parseScope(r.Header.Get("X-Org-Scope")))
			ctx = context.WithValue(ctx, scopeKey, scope)
			if u, err := uuid.Parse(strings.TrimSpace(r.Header.Get("X-User-Id"))); err == nil {
				ctx = context.WithValue(ctx, userKey, u)
			}
			if roles := parseCSV(r.Header.Get("X-Roles")); len(roles) > 0 {
				ctx = context.WithValue(ctx, rolesKey, roles)
			}
			if perms := parseCSV(r.Header.Get("X-Permissions")); len(perms) > 0 {
				ctx = context.WithValue(ctx, permsKey, perms)
			}
		}
		rWithCtx := r.WithContext(ctx)
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[panic] cid=%s panic=%v", cid, rec)
				WriteError(w, rWithCtx, Internal("panic"))
			}
		}()
		next.ServeHTTP(w, rWithCtx)
	})
}

func parseScope(h string) []uuid.UUID {
	if strings.TrimSpace(h) == "" {
		return nil
	}
	var out []uuid.UUID
	for _, p := range strings.Split(h, ",") {
		if u, err := uuid.Parse(strings.TrimSpace(p)); err == nil && !slices.Contains(out, u) {
			out = append(out, u)
		}
	}
	return out
}

func mergeScopes(primary, secondary []uuid.UUID) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(primary)+len(secondary))
	for _, u := range primary {
		if !slices.Contains(out, u) {
			out = append(out, u)
		}
	}
	for _, u := range secondary {
		if !slices.Contains(out, u) {
			out = append(out, u)
		}
	}
	return out
}

func parseCSV(h string) []string {
	if strings.TrimSpace(h) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(h, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// ---- responses -------------------------------------------------------------

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// PageMeta / ListEnvelope match API Spec 2.8.
type PageMeta struct {
	Number        int `json:"number"`
	Size          int `json:"size"`
	TotalElements int `json:"totalElements"`
	TotalPages    int `json:"totalPages"`
}

type ListEnvelope struct {
	Data any      `json:"data"`
	Page PageMeta `json:"page"`
}

func NewPageMeta(page, size, totalElements int) PageMeta {
	if size <= 0 {
		size = 25
	}
	if page <= 0 {
		page = 1
	}
	totalPages := 0
	if totalElements > 0 {
		totalPages = (totalElements + size - 1) / size
	}
	return PageMeta{
		Number:        page,
		Size:          size,
		TotalElements: totalElements,
		TotalPages:    totalPages,
	}
}

// ---- error model (API Spec 2.9) --------------------------------------------

type Detail struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}

type APIError struct {
	Status  int      `json:"-"`
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []Detail `json:"details,omitempty"`
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

func BadRequest(msg string, d ...Detail) *APIError {
	return &APIError{Status: http.StatusBadRequest, Code: "VALIDATION_ERROR", Message: msg, Details: d}
}
func Unauthorized(msg string) *APIError {
	return &APIError{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: msg}
}
func Forbidden(msg string) *APIError {
	return &APIError{Status: http.StatusForbidden, Code: "FORBIDDEN_SCOPE", Message: msg}
}
func NotFound(msg string) *APIError {
	return &APIError{Status: http.StatusNotFound, Code: "RESOURCE_NOT_FOUND", Message: msg}
}
func Conflict(code, msg string) *APIError {
	return &APIError{Status: http.StatusConflict, Code: code, Message: msg}
}
func Unprocessable(code, msg string, d ...Detail) *APIError {
	return &APIError{Status: http.StatusUnprocessableEntity, Code: code, Message: msg, Details: d}
}
func PreconditionRequired(msg string) *APIError {
	return &APIError{Status: http.StatusPreconditionRequired, Code: "PRECONDITION_REQUIRED", Message: msg}
}
func Internal(msg string) *APIError {
	return &APIError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: msg}
}

// WriteError renders an APIError (or wrapped APIError / generic 500) as the
// standard envelope. Internal 5xx details are logged server-side and masked.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var ae *APIError
	if !errors.As(err, &ae) {
		ae = Internal(err.Error())
	}
	cid := Correlation(r.Context())
	msg := ae.Message
	if ae.Status >= 500 {
		log.Printf("[error] cid=%s code=%s status=%d: %s", cid, ae.Code, ae.Status, ae.Message)
		msg = "internal server error"
	}
	body := map[string]any{
		"error": map[string]any{
			"code":          ae.Code,
			"message":       msg,
			"correlationId": cid,
			"details":       ae.Details,
		},
	}
	WriteJSON(w, ae.Status, body)
}

// MapDBError maps SQLSTATE errors deterministically into APIErrors.
// RLS policy errors (42501) map to 404 to avoid leaking tenant existence.
func MapDBError(err error, dupMsg string) error {
	if err == nil {
		return nil
	}
	var ae *APIError
	if errors.As(err, &ae) {
		return ae
	}
	switch platform.ClassifyPGError(err) {
	case platform.PGNotFound:
		return NotFound("resource not found")
	case platform.PGRLS:
		return NotFound("resource not found")
	case platform.PGUnique:
		if dupMsg == "" {
			dupMsg = "resource already exists"
		}
		return Conflict("DUPLICATE_RESOURCE", dupMsg)
	case platform.PGForeignKey:
		return Unprocessable("FOREIGN_KEY_VIOLATION", "referenced resource does not exist")
	case platform.PGCheck:
		return Unprocessable("CHECK_CONSTRAINT_VIOLATION", "domain invariant violated")
	case platform.PGInvalidInput:
		return BadRequest("invalid input value for field")
	default:
		return Internal(err.Error())
	}
}

// ---- request helpers -------------------------------------------------------

// IfMatch parses the optimistic-lock row_version from the If-Match header.
func IfMatch(r *http.Request) (int32, error) {
	v := strings.TrimSpace(r.Header.Get("If-Match"))
	v = strings.TrimPrefix(v, "W/")
	v = strings.Trim(v, `" `)
	if v == "" {
		return 0, PreconditionRequired("If-Match (rowVersion) wajib untuk mutasi")
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return 0, BadRequest("If-Match bukan angka rowVersion yang valid")
	}
	return int32(n), nil
}

// PaginateParams parses ?page & ?size (default 1/25, max size 100, overflow-safe).
func PaginateParams(r *http.Request) (page, size, limit, offset int32) {
	p, sz := 1, 25
	if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v > 0 {
		p = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && v > 0 {
		sz = v
	}
	if sz > 100 {
		sz = 100
	}
	off64 := int64(p-1) * int64(sz)
	if off64 > math.MaxInt32 {
		off64 = math.MaxInt32 - int64(sz)
		p = int(off64/int64(sz)) + 1
	}
	return int32(p), int32(sz), int32(sz), int32(off64)
}

// Paginate preserves the existing (limit, offset) signature while delegating to PaginateParams.
func Paginate(r *http.Request) (limit, offset int32) {
	_, _, limit, offset = PaginateParams(r)
	return limit, offset
}

// RequireScope returns the caller scope or a 401 if absent (deny-by-default).
// For mutating HTTP methods (POST, PUT, PATCH, DELETE), it rejects requests where
// the active organization is SystemOrgID (00000000-0000-0000-0000-000000000000) with 403
// and strips SystemOrgID from the returned scope slice.
func RequireScope(r *http.Request) ([]uuid.UUID, error) {
	s := Scope(r.Context())
	if len(s) == 0 {
		return nil, Unauthorized("X-Organization-Id (scope) wajib")
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		if s[0] == platform.SystemOrgID {
			return nil, Forbidden("system organization (00000000-0000-0000-0000-000000000000) is read-only and cannot be selected as active tenant")
		}
		s = platform.TenantOnlyScope(s)
		if len(s) == 0 {
			return nil, Forbidden("active tenant organization scope is required for mutations")
		}
	}
	return s, nil
}

// RequireTenantOrg returns the active tenant organization and tenant-only scope, rejecting SystemOrgID.
func RequireTenantOrg(r *http.Request) (uuid.UUID, []uuid.UUID, error) {
	s := Scope(r.Context())
	if len(s) == 0 {
		return uuid.Nil, nil, Unauthorized("X-Organization-Id (scope) wajib")
	}
	if s[0] == platform.SystemOrgID {
		return uuid.Nil, nil, Forbidden("system organization (00000000-0000-0000-0000-000000000000) is read-only")
	}
	ts := platform.TenantOnlyScope(s)
	if len(ts) == 0 {
		return uuid.Nil, nil, Forbidden("active tenant organization required")
	}
	return ts[0], ts, nil
}

// RequireUser returns the acting user or a 401 if absent.
func RequireUser(r *http.Request) (uuid.UUID, error) {
	u := User(r.Context())
	if u == uuid.Nil {
		return uuid.Nil, Unauthorized("X-User-Id wajib")
	}
	return u, nil
}

// EnforceRBAC reports whether REQUIRE_RBAC is enabled (or APP_ENV=production by default unless REQUIRE_RBAC=false),
// forcing RequirePermission to reject requests missing gateway role/permission headers.
func EnforceRBAC() bool {
	v := strings.TrimSpace(os.Getenv("REQUIRE_RBAC"))
	if strings.EqualFold(v, "true") || v == "1" {
		return true
	}
	if strings.EqualFold(v, "false") || v == "0" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")
}

// RequirePermission checks RBAC permission when X-Permissions or X-Roles are injected
// by the API gateway. Direct internal dev calls without gateway role headers stay allowed
// unless EnforceRBAC is explicitly enabled via REQUIRE_RBAC=true.
func RequirePermission(r *http.Request, perm string) error {
	ctx := r.Context()
	if len(Roles(ctx)) == 0 && len(Permissions(ctx)) == 0 {
		if EnforceRBAC() {
			return Forbidden("missing required permission: " + perm)
		}
		return nil
	}
	if !HasPermission(ctx, perm) {
		return Forbidden("missing required permission: " + perm)
	}
	return nil
}
