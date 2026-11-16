// Package httpapi wires org-service REST endpoints (/organization-types,
// /organizations, /org-units, hierarchy, activate).
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/org-service/internal/orgdb"
	"github.com/sinergiinvestasiproperti/erp/services/org-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type Store interface {
	CreateOrganizationType(ctx context.Context, scope []uuid.UUID, arg orgdb.CreateOrganizationTypeParams) (orgdb.CreateOrganizationTypeRow, error)
	ListOrganizationTypesWithCount(ctx context.Context, scope []uuid.UUID, arg orgdb.ListOrganizationTypesParams) ([]orgdb.ListOrganizationTypesRow, int64, error)
	CreateAndEmit(ctx context.Context, scope []uuid.UUID, arg orgdb.CreateOrganizationParams, ev orgdb.InsertOutboxParams) (orgdb.CreateOrganizationRow, error)
	GetWithHierarchy(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (orgdb.OrgOrganization, []orgdb.ListOrganizationDescendantsRow, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, arg orgdb.ListOrganizationsParams) ([]orgdb.ListOrganizationsRow, int64, error)
	UpdateAndEmit(ctx context.Context, scope []uuid.UUID, arg orgdb.UpdateOrganizationParams, ev orgdb.InsertOutboxParams) (orgdb.UpdateOrganizationRow, error)
	ActivateAndEmit(ctx context.Context, scope []uuid.UUID, arg orgdb.ActivateOrganizationParams, ev orgdb.InsertOutboxParams) (int32, error)
	DeleteOrganization(ctx context.Context, scope []uuid.UUID, arg orgdb.DeleteOrganizationParams) (int64, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /organization-types", s.createType)
	mux.HandleFunc("GET /organization-types", s.listTypes)

	mux.HandleFunc("POST /organizations", s.create)
	mux.HandleFunc("GET /organizations", s.list)
	mux.HandleFunc("GET /organizations/{id}", s.get)
	mux.HandleFunc("PUT /organizations/{id}", s.update)
	mux.HandleFunc("POST /organizations/{id}/activate", s.activate)
	mux.HandleFunc("DELETE /organizations/{id}", s.deleteOrg)
	mux.HandleFunc("GET /organizations/{id}/hierarchy", s.hierarchy)

	mux.HandleFunc("GET /org-units", s.list)
	mux.HandleFunc("GET /org-units/{id}", s.get)
	mux.HandleFunc("GET /org-units/{id}/hierarchy", s.hierarchy)
}

type typeReq struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	LevelHint *int32 `json:"levelHint"`
}

type orgReq struct {
	Code               string `json:"code"`
	Name               string `json:"name"`
	OrganizationTypeID string `json:"organizationTypeId"`
	ParentID           string `json:"parentId"`
	EffectiveFrom      string `json:"effectiveFrom"`
	EffectiveTo        string `json:"effectiveTo"`
}

func (s *Server) createType(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
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
	var req orgReq
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
	typeID, err := httpx.ParseUUIDParam("organizationTypeId", req.OrganizationTypeID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	effFromStr := strings.TrimSpace(req.EffectiveFrom)
	if effFromStr == "" {
		effFromStr = "2026-01-01"
	}
	effFrom, err := platform.ParseDate(effFromStr)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("effectiveFrom harus YYYY-MM-DD"))
		return
	}
	var effTo pgtype.Date
	if strings.TrimSpace(req.EffectiveTo) != "" {
		effTo, err = platform.ParseDate(req.EffectiveTo)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("effectiveTo harus YYYY-MM-DD"))
			return
		}
		if effTo.Time.Before(effFrom.Time) {
			httpx.WriteError(w, r, httpx.BadRequest("effectiveTo tidak boleh sebelum effectiveFrom"))
			return
		}
	}
	var parentID *uuid.UUID
	if pRaw := strings.TrimSpace(req.ParentID); pRaw != "" {
		p, err := httpx.ParseUUIDParam("parentId", pRaw)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		parentID = &p
	}

	payload, _ := json.Marshal(map[string]any{"code": code, "name": name, "parentId": parentID})
	ev := orgdb.InsertOutboxParams{
		AggregateType:  "Organization",
		EventType:      "org.unit.upserted",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	created, err := s.repo.CreateAndEmit(r.Context(), scope, orgdb.CreateOrganizationParams{
		OrganizationID:     scope[0],
		Code:               code,
		Name:               name,
		OrganizationTypeID: typeID,
		ParentID:           parentID,
		EffectiveFrom:      effFrom,
		EffectiveTo:        effTo,
		CreatedBy:          actor,
	}, ev)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.BadRequest("parentId tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode organisasi sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, created.RowVersion, map[string]any{
		"id":         created.ID,
		"code":       code,
		"name":       name,
		"status":     created.Status,
		"rowVersion": created.RowVersion,
	})
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
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
	var statusPtr *string
	if st := strings.TrimSpace(r.URL.Query().Get("status")); st != "" {
		statusPtr = &st
	}
	var parentPtr *uuid.UUID
	if pr := strings.TrimSpace(r.URL.Query().Get("parentId")); pr != "" {
		pid, err := httpx.ParseUUIDParam("parentId", pr)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		parentPtr = &pid
	}
	rows, total, err := s.repo.ListWithCount(r.Context(), scope, orgdb.ListOrganizationsParams{
		Limit:    limit,
		Offset:   offset,
		Status:   statusPtr,
		ParentID: parentPtr,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, o := range rows {
		out = append(out, map[string]any{
			"id":                 o.ID,
			"organizationId":     o.OrganizationID,
			"code":               o.Code,
			"name":               o.Name,
			"organizationTypeId": o.OrganizationTypeID,
			"parentId":           o.ParentID,
			"path":               o.Path,
			"status":             o.Status,
			"effectiveFrom":      platform.FormatDate(o.EffectiveFrom),
			"effectiveTo":        formatOptionalDate(o.EffectiveTo),
			"rowVersion":         o.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
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
	o, desc, err := s.repo.GetWithHierarchy(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("organization tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, o.RowVersion, map[string]any{
		"id":                 o.ID,
		"organizationId":     o.OrganizationID,
		"code":               o.Code,
		"name":               o.Name,
		"organizationTypeId": o.OrganizationTypeID,
		"parentId":           o.ParentID,
		"path":               o.Path,
		"status":             o.Status,
		"effectiveFrom":      platform.FormatDate(o.EffectiveFrom),
		"effectiveTo":        formatOptionalDate(o.EffectiveTo),
		"rowVersion":         o.RowVersion,
		"descendantCount":    len(desc),
	})
}

func (s *Server) hierarchy(w http.ResponseWriter, r *http.Request) {
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
	_, desc, err := s.repo.GetWithHierarchy(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("organization tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(desc))
	for _, d := range desc {
		out = append(out, map[string]any{
			"id":                 d.ID,
			"code":               d.Code,
			"name":               d.Name,
			"organizationTypeId": d.OrganizationTypeID,
			"parentId":           d.ParentID,
			"path":               d.Path,
			"status":             d.Status,
			"depth":              d.Depth,
			"rowVersion":         d.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(0, len(out), len(out)),
	})
}

func (s *Server) update(w http.ResponseWriter, r *http.Request) {
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
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req orgReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name wajib diisi"))
		return
	}
	typeID, err := httpx.ParseUUIDParam("organizationTypeId", req.OrganizationTypeID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	effFrom, err := platform.ParseDate(req.EffectiveFrom)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("effectiveFrom harus YYYY-MM-DD"))
		return
	}
	var effTo pgtype.Date
	if strings.TrimSpace(req.EffectiveTo) != "" {
		effTo, err = platform.ParseDate(req.EffectiveTo)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("effectiveTo harus YYYY-MM-DD"))
			return
		}
		if effTo.Time.Before(effFrom.Time) {
			httpx.WriteError(w, r, httpx.BadRequest("effectiveTo tidak boleh sebelum effectiveFrom"))
			return
		}
	}
	existing, _, err := s.repo.GetWithHierarchy(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("organization tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.RowVersion != ver {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "row_version organisasi tidak cocok"))
		return
	}
	payload, _ := json.Marshal(map[string]any{"code": existing.Code, "name": name, "path": existing.Path})
	res, err := s.repo.UpdateAndEmit(r.Context(), scope, orgdb.UpdateOrganizationParams{
		ID:                 id,
		RowVersion:         ver,
		Name:               name,
		OrganizationTypeID: typeID,
		EffectiveFrom:      effFrom,
		EffectiveTo:        effTo,
		UpdatedBy:          &actor,
	}, orgdb.InsertOutboxParams{
		AggregateType:  "Organization",
		AggregateID:    id,
		EventType:      "org.unit.upserted",
		Payload:        payload,
		OrganizationID: existing.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "organisasi berubah secara bersamaan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, res.RowVersion, map[string]any{
		"id":         id,
		"code":       existing.Code,
		"name":       name,
		"status":     res.Status,
		"rowVersion": res.RowVersion,
	})
}

func (s *Server) activate(w http.ResponseWriter, r *http.Request) {
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
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	existing, _, err := s.repo.GetWithHierarchy(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("organization tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.Status != "Draft" || existing.RowVersion != ver {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "organisasi bukan Draft atau row_version tidak cocok"))
		return
	}
	payload, _ := json.Marshal(map[string]any{"code": existing.Code, "name": existing.Name, "status": "Active"})
	newVer, err := s.repo.ActivateAndEmit(r.Context(), scope, orgdb.ActivateOrganizationParams{
		ID:         id,
		RowVersion: ver,
		UpdatedBy:  &actor,
	}, orgdb.InsertOutboxParams{
		AggregateType:  "Organization",
		AggregateID:    id,
		EventType:      "org.unit.upserted",
		Payload:        payload,
		OrganizationID: existing.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "organisasi berubah secara bersamaan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, map[string]any{
		"id":         id,
		"code":       existing.Code,
		"name":       existing.Name,
		"status":     "Active",
		"rowVersion": newVer,
	})
}

func (s *Server) deleteOrg(w http.ResponseWriter, r *http.Request) {
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
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if _, _, err := s.repo.GetWithHierarchy(r.Context(), scope, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("organization tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	n, err := s.repo.DeleteOrganization(r.Context(), scope, orgdb.DeleteOrganizationParams{
		ID:         id,
		RowVersion: ver,
		UpdatedBy:  &actor,
	})
	if err != nil {
		if errors.Is(err, repository.ErrHasChildren) {
			httpx.WriteError(w, r, httpx.Conflict("HAS_CHILDREN", err.Error()))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "row_version organisasi tidak cocok"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func formatOptionalDate(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := platform.FormatDate(d)
	return &s
}
