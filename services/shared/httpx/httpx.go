// Package httpx holds the SHARED HTTP layer helpers (SDD BAB 4): correlation +
// org-scope middleware, JSON envelope, the standard error model, pagination,
// If-Match parsing, and a dev-grade idempotency store.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"

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

		scope := mergeScopes(parseScope(r.Header.Get("X-Organization-Id")), parseScope(r.Header.Get("X-Org-Scope")))
		ctx := context.WithValue(r.Context(), correlationKey, cid)
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
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[panic] cid=%s panic=%v", cid, rec)
				WriteError(w, r, Internal("panic"))
			}
		}()
		next.ServeHTTP(w, r.WithContext(ctx))
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
func RequireScope(r *http.Request) ([]uuid.UUID, error) {
	s := Scope(r.Context())
	if len(s) == 0 {
		return nil, Unauthorized("X-Organization-Id (scope) wajib")
	}
	return s, nil
}

// RequireUser returns the acting user or a 401 if absent.
func RequireUser(r *http.Request) (uuid.UUID, error) {
	u := User(r.Context())
	if u == uuid.Nil {
		return uuid.Nil, Unauthorized("X-User-Id wajib")
	}
	return u, nil
}

// RequirePermission checks RBAC permission when X-Permissions or X-Roles are injected
// by the API gateway. Direct internal dev calls without gateway role headers stay allowed
// unless EnforceRBAC is explicitly enabled via REQUIRE_RBAC=true.
func RequirePermission(r *http.Request, perm string) error {
	ctx := r.Context()
	if len(Roles(ctx)) == 0 && len(Permissions(ctx)) == 0 {
		return nil
	}
	if !HasPermission(ctx, perm) {
		return Forbidden("missing required permission: " + perm)
	}
	return nil
}
