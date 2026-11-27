package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/org-service/internal/orgdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

func (s *Server) createType(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "org:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequireMaxAdminTier(r, httpx.AdminTierHolding); err != nil {
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
	var req typeReq
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan name wajib diisi"))
		return
	}
	created, err := s.repo.CreateOrganizationType(r.Context(), scope, orgdb.CreateOrganizationTypeParams{
		OrganizationID: scope[0],
		Code:           code,
		Name:           name,
		LevelHint:      req.LevelHint,
		IsActive:       true,
		CreatedBy:      actor,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode tipe organisasi sudah ada"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, created.RowVersion, map[string]any{
		"id":         created.ID,
		"code":       code,
		"name":       name,
		"rowVersion": created.RowVersion,
	})
}

func (s *Server) listTypes(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "org:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListOrganizationTypesWithCount(r.Context(), scope, orgdb.ListOrganizationTypesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, t := range rows {
		out = append(out, map[string]any{
			"id":             t.ID,
			"organizationId": t.OrganizationID,
			"code":           t.Code,
			"name":           t.Name,
			"levelHint":      t.LevelHint,
			"isActive":       t.IsActive,
			"rowVersion":     t.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) descendants(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "org:read"); err != nil {
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
	org, desc, err := s.repo.GetWithHierarchy(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("organization tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	ids := make([]uuid.UUID, 0, len(desc)+1)
	ids = append(ids, org.ID)
	for _, d := range desc {
		if d.ID != org.ID {
			ids = append(ids, d.ID)
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"organizationId": org.ID,
		"descendantIds":  ids,
	})
}

func (s *Server) getAttributes(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "org:read"); err != nil {
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
	attrs, err := s.repo.ListAttributes(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if attrs == nil {
		attrs = map[string]any{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"organizationId": id,
		"attributes":     attrs,
	})
}

func (s *Server) putAttributes(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "org:write"); err != nil {
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
	var req struct {
		Attributes map[string]any `json:"attributes"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if len(req.Attributes) == 0 {
		httpx.WriteError(w, r, httpx.BadRequest("attributes wajib diisi"))
		return
	}
	if err := s.repo.UpsertAttributes(r.Context(), scope, scope[0], id, req.Attributes, actor); err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"organizationId": id,
		"attributes":     req.Attributes,
	})
}
