package httpapi

import (
	"net/http"
	"strings"

	"github.com/sinergiinvestasiproperti/erp/services/dimension-service/internal/dimensiondb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func (s *Server) listCostCenters(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListCostCenters(r.Context(), scope, statusFilter(r), limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	data := make([]centerResp, 0, len(rows))
	for _, row := range rows {
		data = append(data, costCenterDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: data, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createCostCenter(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req centerReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code is required"))
		return
	}
	status, from, to, err := validateCenter(req.Name, req.Status, req.EffectiveFrom, req.EffectiveTo)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	created, err := s.repo.CreateCostCenter(r.Context(), scope, dimensiondb.CreateCostCenterParams{
		OrganizationID: scope[0], Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name), Description: req.Description,
		ParentID: req.ParentID, OwnerRef: req.OwnerRef, Status: status, EffectiveFrom: from, EffectiveTo: to, CreatedBy: user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, centerResp{
		ID: created.ID, Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name), Description: req.Description,
		ParentID: req.ParentID, OwnerRef: req.OwnerRef, Status: status, EffectiveFrom: platform.FormatDate(from),
		EffectiveTo: platform.FormatDate(to), RowVersion: created.RowVersion,
	})
}

func (s *Server) getCostCenter(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "cost center")
	if !ok {
		return
	}
	row, err := s.repo.GetCostCenter(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, costCenterDTO(row))
}

func (s *Server) updateCostCenter(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "cost center")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req centerReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.ParentID != nil && *req.ParentID == id {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_PARENT", "cost center cannot be its own parent"))
		return
	}
	status, from, to, err := validateCenter(req.Name, req.Status, req.EffectiveFrom, req.EffectiveTo)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if existing, err := s.repo.GetCostCenter(r.Context(), scope, id); err == nil {
		if !guardSystemOrgMutation(w, r, existing.OrganizationID) {
			return
		}
	}
	next, err := s.repo.UpdateCostCenter(r.Context(), scope, dimensiondb.UpdateCostCenterParams{
		ID: id, RowVersion: rowVer, Name: strings.TrimSpace(req.Name), Description: req.Description,
		ParentID: req.ParentID, OwnerRef: req.OwnerRef, Status: status, EffectiveFrom: from, EffectiveTo: to, UpdatedBy: &user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"rowVersion": next})
}

func (s *Server) deleteCostCenter(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "cost center")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if existing, err := s.repo.GetCostCenter(r.Context(), scope, id); err == nil {
		if !guardSystemOrgMutation(w, r, existing.OrganizationID) {
			return
		}
	}
	if err := s.repo.DeleteCostCenter(r.Context(), scope, dimensiondb.DeleteCostCenterParams{ID: id, RowVersion: rowVer, UpdatedBy: &user}); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listProfitCenters(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListProfitCenters(r.Context(), scope, statusFilter(r), limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	data := make([]centerResp, 0, len(rows))
	for _, row := range rows {
		data = append(data, profitCenterDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: data, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createProfitCenter(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req centerReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code is required"))
		return
	}
	status, from, to, err := validateCenter(req.Name, req.Status, req.EffectiveFrom, req.EffectiveTo)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	created, err := s.repo.CreateProfitCenter(r.Context(), scope, dimensiondb.CreateProfitCenterParams{
		OrganizationID: scope[0], Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name), ParentID: req.ParentID,
		Status: status, EffectiveFrom: from, EffectiveTo: to, CreatedBy: user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, centerResp{
		ID: created.ID, Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name), ParentID: req.ParentID, Status: status,
		EffectiveFrom: platform.FormatDate(from), EffectiveTo: platform.FormatDate(to), RowVersion: created.RowVersion,
	})
}

func (s *Server) getProfitCenter(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "profit center")
	if !ok {
		return
	}
	row, err := s.repo.GetProfitCenter(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, profitCenterDTO(row))
}

func (s *Server) updateProfitCenter(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "profit center")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req centerReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.ParentID != nil && *req.ParentID == id {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_PARENT", "profit center cannot be its own parent"))
		return
	}
	status, from, to, err := validateCenter(req.Name, req.Status, req.EffectiveFrom, req.EffectiveTo)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if existing, err := s.repo.GetProfitCenter(r.Context(), scope, id); err == nil {
		if !guardSystemOrgMutation(w, r, existing.OrganizationID) {
			return
		}
	}
	next, err := s.repo.UpdateProfitCenter(r.Context(), scope, dimensiondb.UpdateProfitCenterParams{
		ID: id, RowVersion: rowVer, Name: strings.TrimSpace(req.Name), ParentID: req.ParentID, Status: status,
		EffectiveFrom: from, EffectiveTo: to, UpdatedBy: &user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"rowVersion": next})
}

func (s *Server) deleteProfitCenter(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "profit center")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if existing, err := s.repo.GetProfitCenter(r.Context(), scope, id); err == nil {
		if !guardSystemOrgMutation(w, r, existing.OrganizationID) {
			return
		}
	}
	if err := s.repo.DeleteProfitCenter(r.Context(), scope, dimensiondb.DeleteProfitCenterParams{ID: id, RowVersion: rowVer, UpdatedBy: &user}); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
