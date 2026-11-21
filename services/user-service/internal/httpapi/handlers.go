// Package httpapi is user-service's REST layer (SDD BAB 4). Creating or
// updating an app user emits user.account.upserted in the same transaction.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/user-service/internal/userdb"
)

type Store interface {
	CreateAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.CreateAppUserParams, ev userdb.InsertOutboxParams) (userdb.CreateAppUserRow, error)
	GetWithRoles(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (userdb.UsrAppUser, []userdb.ListUserOrgAssignmentsRow, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, arg userdb.ListAppUsersParams) ([]userdb.ListAppUsersRow, int64, error)
	UpdateAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.UpdateAppUserParams, ev userdb.InsertOutboxParams) (userdb.UpdateAppUserRow, error)
	ChangeStatusAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.ChangeUserStatusParams, ev userdb.InsertOutboxParams) (userdb.ChangeUserStatusRow, error)
	DeleteUser(ctx context.Context, scope []uuid.UUID, arg userdb.DeleteAppUserParams) (int64, error)
	AssignRoleAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.CreateUserOrgAssignmentParams, ev userdb.InsertOutboxParams) (userdb.CreateUserOrgAssignmentRow, error)
	RemoveRoleAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.RemoveUserOrgAssignmentParams, ev userdb.InsertOutboxParams) (int64, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /users", s.create)
	mux.HandleFunc("GET /users", s.list)
	mux.HandleFunc("GET /users/{id}", s.get)
	mux.HandleFunc("PUT /users/{id}", s.update)
	mux.HandleFunc("DELETE /users/{id}", s.deleteUser)
	mux.HandleFunc("POST /users/{id}/disable", s.disable)
	mux.HandleFunc("GET /users/{id}/roles", s.listRoles)
	mux.HandleFunc("POST /users/{id}/roles", s.assignRole)
	mux.HandleFunc("DELETE /users/{id}/roles/{roleId}", s.removeRole)
}

type createReq struct {
	Username              string `json:"username"`
	Email                 string `json:"email"`
	FullName              string `json:"fullName"`
	DefaultOrganizationID string `json:"defaultOrganizationId"`
}

type updateReq struct {
	Email                 string `json:"email"`
	FullName              string `json:"fullName"`
	DefaultOrganizationID string `json:"defaultOrganizationId"`
}

type assignRoleReq struct {
	RoleID            uuid.UUID `json:"roleId"`
	OrganizationRefID uuid.UUID `json:"organizationRefId"`
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "user:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req createReq
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	uname := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)
	fullName := strings.TrimSpace(req.FullName)
	if uname == "" || !validEmail(email) || fullName == "" {
		httpx.WriteError(w, r, httpx.BadRequest("username, email valid, dan fullName wajib diisi"))
		return
	}
	defOrg, err := parseOptionalUUID("defaultOrganizationId", req.DefaultOrganizationID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	payload, _ := json.Marshal(map[string]any{"username": uname, "email": email, "fullName": fullName, "status": "Active"})
	ev := userdb.InsertOutboxParams{
		AggregateType:  "AppUser",
		EventType:      "user.account.upserted",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	created, err := s.repo.CreateAndEmit(r.Context(), scope, userdb.CreateAppUserParams{
		OrganizationID:        scope[0],
		Username:              uname,
		Email:                 email,
		FullName:              fullName,
		DefaultOrganizationID: defOrg,
		CreatedBy:             actor,
	}, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "username atau email sudah terdaftar"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, created.RowVersion, map[string]any{
		"id":         created.ID,
		"username":   uname,
		"email":      email,
		"fullName":   fullName,
		"status":     created.Status,
		"rowVersion": created.RowVersion,
	})
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "user:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var statusPtr *string
	if st := strings.TrimSpace(r.URL.Query().Get("status")); st != "" {
		statusPtr = &st
	}
	rows, total, err := s.repo.ListWithCount(r.Context(), scope, userdb.ListAppUsersParams{
		Limit:  limit,
		Offset: offset,
		Status: statusPtr,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, u := range rows {
		out = append(out, map[string]any{
			"id":                    u.ID,
			"organizationId":        u.OrganizationID,
			"username":              u.Username,
			"email":                 u.Email,
			"fullName":              u.FullName,
			"status":                u.Status,
			"defaultOrganizationId": u.DefaultOrganizationID,
			"rowVersion":            u.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "user:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	u, roles, err := s.repo.GetWithRoles(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("user tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, u.RowVersion, map[string]any{
		"id":                    u.ID,
		"organizationId":        u.OrganizationID,
		"username":              u.Username,
		"email":                 u.Email,
		"fullName":              u.FullName,
		"status":                u.Status,
		"defaultOrganizationId": u.DefaultOrganizationID,
		"rowVersion":            u.RowVersion,
		"roles":                 formatRoles(roles),
	})
}

func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "user:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req updateReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	email := strings.TrimSpace(req.Email)
	fullName := strings.TrimSpace(req.FullName)
	if !validEmail(email) || fullName == "" {
		httpx.WriteError(w, r, httpx.BadRequest("email valid dan fullName wajib diisi"))
		return
	}
	defOrg, err := parseOptionalUUID("defaultOrganizationId", req.DefaultOrganizationID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	existing, _, err := s.repo.GetWithRoles(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("user tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.RowVersion != ver {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "row_version user tidak cocok"))
		return
	}
	payload, _ := json.Marshal(map[string]any{"username": existing.Username, "email": email, "fullName": fullName, "status": existing.Status})
	res, err := s.repo.UpdateAndEmit(r.Context(), scope, userdb.UpdateAppUserParams{
		ID:                    id,
		RowVersion:            ver,
		Email:                 email,
		FullName:              fullName,
		DefaultOrganizationID: defOrg,
		UpdatedBy:             &actor,
	}, userdb.InsertOutboxParams{
		AggregateType:  "AppUser",
		AggregateID:    id,
		EventType:      "user.account.upserted",
		Payload:        payload,
		OrganizationID: existing.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "user berubah secara bersamaan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, "email sudah terdaftar"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, res.RowVersion, map[string]any{
		"id":         id,
		"username":   existing.Username,
		"email":      email,
		"fullName":   fullName,
		"status":     res.Status,
		"rowVersion": res.RowVersion,
	})
}

func (s *Server) disable(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "user:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	existing, _, err := s.repo.GetWithRoles(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("user tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.RowVersion != ver {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "row_version user tidak cocok"))
		return
	}
	payload, _ := json.Marshal(map[string]any{"username": existing.Username, "email": existing.Email, "status": "Inactive"})
	res, err := s.repo.ChangeStatusAndEmit(r.Context(), scope, userdb.ChangeUserStatusParams{
		ID:         id,
		RowVersion: ver,
		Status:     "Inactive",
		UpdatedBy:  &actor,
	}, userdb.InsertOutboxParams{
		AggregateType:  "AppUser",
		AggregateID:    id,
		EventType:      "user.account.upserted",
		Payload:        payload,
		OrganizationID: existing.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "user berubah secara bersamaan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, res.RowVersion, map[string]any{
		"id":         id,
		"username":   existing.Username,
		"status":     res.Status,
		"rowVersion": res.RowVersion,
	})
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "user:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if _, _, err := s.repo.GetWithRoles(r.Context(), scope, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("user tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	n, err := s.repo.DeleteUser(r.Context(), scope, userdb.DeleteAppUserParams{
		ID:         id,
		RowVersion: ver,
		UpdatedBy:  &actor,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "row_version user tidak cocok"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listRoles(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "user:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	_, roles, err := s.repo.GetWithRoles(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("user tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := formatRoles(roles)
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(0, len(out), len(out)),
	})
}

func (s *Server) assignRole(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "user:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	userID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req assignRoleReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.RoleID == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("roleId wajib diisi"))
		return
	}
	orgRef := req.OrganizationRefID
	if orgRef == uuid.Nil {
		orgRef = scope[0]
	}
	if !orgInScope(scope, orgRef) {
		httpx.WriteError(w, r, httpx.Forbidden("organizationRefId berada di luar scope akses"))
		return
	}
	payload, _ := json.Marshal(map[string]any{"userId": userID, "roleId": req.RoleID, "organizationRefId": orgRef})
	row, err := s.repo.AssignRoleAndEmit(r.Context(), scope, userdb.CreateUserOrgAssignmentParams{
		OrganizationID:    scope[0],
		UserID:            userID,
		OrganizationRefID: orgRef,
		RoleID:            req.RoleID,
		CreatedBy:         actor,
	}, userdb.InsertOutboxParams{
		AggregateType:  "AppUser",
		AggregateID:    userID,
		EventType:      "user.account.upserted",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("user tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, "role sudah terpasang"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusCreated, row.RowVersion, map[string]any{
		"id":                row.ID,
		"userId":            userID,
		"roleId":            req.RoleID,
		"organizationRefId": orgRef,
		"rowVersion":        row.RowVersion,
	})
}

func orgInScope(scope []uuid.UUID, target uuid.UUID) bool {
	for _, s := range scope {
		if s == target {
			return true
		}
	}
	return false
}

func (s *Server) removeRole(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "user:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	userID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	roleID, err := httpx.ParseUUIDParam("roleId", r.PathValue("roleId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	orgRef := scope[0]
	if rawOrg := strings.TrimSpace(r.URL.Query().Get("organizationRefId")); rawOrg != "" {
		parsed, perr := httpx.ParseUUIDParam("organizationRefId", rawOrg)
		if perr != nil {
			httpx.WriteError(w, r, perr)
			return
		}
		orgRef = parsed
	}
	if !orgInScope(scope, orgRef) {
		httpx.WriteError(w, r, httpx.Forbidden("organizationRefId berada di luar scope akses"))
		return
	}
	payload, _ := json.Marshal(map[string]any{"userId": userID, "roleId": roleID, "organizationRefId": orgRef, "removed": true})
	n, err := s.repo.RemoveRoleAndEmit(r.Context(), scope, userdb.RemoveUserOrgAssignmentParams{
		UserID:            userID,
		RoleID:            roleID,
		OrganizationRefID: orgRef,
		UpdatedBy:         &actor,
	}, userdb.InsertOutboxParams{
		AggregateType:  "AppUser",
		AggregateID:    userID,
		EventType:      "user.account.upserted",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.NotFound("assignment role tidak ditemukan"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validEmail(s string) bool {
	at := strings.IndexByte(s, '@')
	return at > 0 && at < len(s)-1 && !strings.ContainsAny(s, " \t\r\n")
}

func parseOptionalUUID(field, raw string) (*uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	id, err := httpx.ParseUUIDParam(field, raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func formatRoles(rows []userdb.ListUserOrgAssignmentsRow) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"id":                r.ID,
			"userId":            r.UserID,
			"organizationRefId": r.OrganizationRefID,
			"roleId":            r.RoleID,
			"rowVersion":        r.RowVersion,
		})
	}
	return out
}
