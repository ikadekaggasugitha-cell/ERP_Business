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

func (s *Server) provisionUser(w http.ResponseWriter, r *http.Request) {
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
	var req provisionUserReq
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
	if defOrg != nil && !orgInScope(scope, *defOrg) {
		httpx.WriteError(w, r, httpx.Forbidden("defaultOrganizationId berada di luar scope akses"))
		return
	}
	callerHasRoles := len(httpx.Roles(r.Context())) > 0
	callerTier := httpx.CallerAdminTier(r.Context())

	assignParams := make([]userdb.CreateUserOrgAssignmentParams, 0, len(req.Assignments))
	assignSummary := make([]map[string]any, 0, len(req.Assignments))
	for _, a := range req.Assignments {
		if a.RoleID == uuid.Nil {
			httpx.WriteError(w, r, httpx.BadRequest("setiap assignment wajib memiliki roleId"))
			return
		}
		orgRef := scope[0]
		if rawOrg := strings.TrimSpace(a.OrganizationRefID); rawOrg != "" {
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
		if callerHasRoles {
			targetTier := roleTierForCode(a.RoleCode)
			if !httpx.CanDelegateTier(callerTier, targetTier) {
				httpx.WriteError(w, r, httpx.Forbidden("admin tier tidak cukup untuk menetapkan role "+a.RoleCode))
				return
			}
		}
		assignParams = append(assignParams, userdb.CreateUserOrgAssignmentParams{
			OrganizationID:    scope[0],
			OrganizationRefID: orgRef,
			RoleID:            a.RoleID,
			CreatedBy:         actor,
		})
		assignSummary = append(assignSummary, map[string]any{
			"organizationRefId": orgRef,
			"roleId":            a.RoleID,
			"roleCode":          a.RoleCode,
		})
	}
	if defOrg == nil && len(assignParams) > 0 {
		firstOrg := assignParams[0].OrganizationRefID
		defOrg = &firstOrg
	}

	payload, _ := json.Marshal(map[string]any{
		"username":    uname,
		"email":       email,
		"fullName":    fullName,
		"status":      "Active",
		"assignments": assignSummary,
	})
	ev := userdb.InsertOutboxParams{
		AggregateType:  "AppUser",
		EventType:      "user.account.upserted",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	created, createdAssigns, err := s.repo.ProvisionAndEmit(r.Context(), scope, userdb.CreateAppUserParams{
		OrganizationID:        scope[0],
		Username:              uname,
		Email:                 email,
		FullName:              fullName,
		DefaultOrganizationID: defOrg,
		CreatedBy:             actor,
	}, assignParams, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "username atau email sudah terdaftar"))
		return
	}
	outAssigns := make([]map[string]any, 0, len(createdAssigns))
	for i, row := range createdAssigns {
		outAssigns = append(outAssigns, map[string]any{
			"id":                row.ID,
			"userId":            created.ID,
			"organizationRefId": assignParams[i].OrganizationRefID,
			"roleId":            assignParams[i].RoleID,
			"rowVersion":        row.RowVersion,
		})
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, created.RowVersion, map[string]any{
		"id":                    created.ID,
		"username":              uname,
		"email":                 email,
		"fullName":              fullName,
		"status":                created.Status,
		"defaultOrganizationId": defOrg,
		"assignments":           outAssigns,
		"rowVersion":            created.RowVersion,
	})
}

func (s *Server) getByUsername(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "user:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		httpx.WriteError(w, r, httpx.BadRequest("username query param wajib diisi"))
		return
	}
	u, roles, err := s.repo.GetByUsernameWithRoles(r.Context(), scope, username)
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
		"roles":                 formatRoles(roles),
		"rowVersion":            u.RowVersion,
	})
}
