package httpapi

import (
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
	var typeCode string
	if orgType, terr := s.repo.GetOrganizationTypeByID(r.Context(), scope, typeID); terr == nil {
		typeCode = strings.ToUpper(strings.TrimSpace(orgType.Code))
	}
	// Tiered Admin Guard: Subsidiary/Branch admins cannot create root Holding organizations,
	// cannot create node types above their tier, and can only attach under a parent in their scope.
	if len(httpx.Roles(r.Context())) > 0 && httpx.CallerAdminTier(r.Context()) >= httpx.AdminTierSubsidiary {
		callerTier := httpx.CallerAdminTier(r.Context())
		if typeCode != "" && !canTierCreateOrgType(callerTier, typeCode) {
			httpx.WriteError(w, r, httpx.Forbidden("admin tier tidak diizinkan membuat tipe organisasi "+typeCode))
			return
		}
		if parentID == nil {
			httpx.WriteError(w, r, httpx.Forbidden("hanya Head Office / Holding Admin yang dapat membuat organisasi induk (root)"))
			return
		}
		inScope := false
		for _, sOrg := range scope {
			if sOrg == *parentID {
				inScope = true
				break
			}
		}
		if !inScope {
			httpx.WriteError(w, r, httpx.Forbidden("parentId berada di luar cakupan organisasi admin"))
			return
		}
	}

	payloadMap := map[string]any{
		"code":                    code,
		"name":                    name,
		"typeCode":                typeCode,
		"parentId":                parentID,
		"autoProvisionWarehouse":  req.AutoProvisionWarehouse,
		"autoProvisionCostCenter": req.AutoProvisionCostCenter,
	}
	if len(req.Attributes) > 0 {
		payloadMap["attributes"] = req.Attributes
	}
	payload, _ := json.Marshal(payloadMap)
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
	if len(req.Attributes) > 0 {
		if aerr := s.repo.UpsertAttributes(r.Context(), scope, scope[0], created.ID, req.Attributes, actor); aerr != nil {
			httpx.WriteError(w, r, httpx.MapDBError(aerr, ""))
			return
		}
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, created.RowVersion, map[string]any{
		"id":                      created.ID,
		"code":                    code,
		"name":                    name,
		"typeCode":                typeCode,
		"status":                  created.Status,
		"attributes":              req.Attributes,
		"autoProvisionWarehouse":  req.AutoProvisionWarehouse,
		"autoProvisionCostCenter": req.AutoProvisionCostCenter,
		"rowVersion":              created.RowVersion,
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
	if existing.OrganizationID == platform.SystemOrgID {
		httpx.WriteError(w, r, httpx.Forbidden("system organization tidak dapat dimodifikasi"))
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
	if existing.OrganizationID == platform.SystemOrgID {
		httpx.WriteError(w, r, httpx.Forbidden("system organization tidak dapat dimodifikasi"))
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
	existing, _, err := s.repo.GetWithHierarchy(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("organization tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.OrganizationID == platform.SystemOrgID {
		httpx.WriteError(w, r, httpx.Forbidden("system organization tidak dapat dihapus"))
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
