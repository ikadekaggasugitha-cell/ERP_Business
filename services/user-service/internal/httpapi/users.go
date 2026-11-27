package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/user-service/internal/userdb"
)

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
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if req.RoleID == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("roleId wajib diisi"))
		return
	}
	orgRef := scope[0]
	if rawOrg := strings.TrimSpace(req.OrganizationRefID); rawOrg != "" {
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
	// Tiered Admin Anti-Escalation Guard: an admin can only assign roles strictly below their own tier.
	if len(httpx.Roles(r.Context())) > 0 {
		callerTier := httpx.CallerAdminTier(r.Context())
		targetTier := roleTierForCode(req.RoleCode)
		if !httpx.CanDelegateTier(callerTier, targetTier) {
			httpx.WriteError(w, r, httpx.Forbidden("admin tier tidak cukup untuk menetapkan role ini (privilege escalation ditolak)"))
			return
		}
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
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, row.RowVersion, map[string]any{
		"id":                row.ID,
		"userId":            userID,
		"roleId":            req.RoleID,
		"organizationRefId": orgRef,
		"rowVersion":        row.RowVersion,
	})
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
	// Tiered Admin Anti-Escalation Guard: an admin can only revoke roles strictly below their own tier.
	if len(httpx.Roles(r.Context())) > 0 {
		callerTier := httpx.CallerAdminTier(r.Context())
		targetTier := roleTierForCode(r.URL.Query().Get("roleCode"))
		if !httpx.CanDelegateTier(callerTier, targetTier) {
			httpx.WriteError(w, r, httpx.Forbidden("admin tier tidak cukup untuk mencabut role ini"))
			return
		}
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
