package httpapi

import (
	"net/http"
	"strings"

	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/fadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

func (s *Server) listCategories(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListCategories(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, categoryDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) getCategory(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	c, err := s.repo.GetCategory(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, c.RowVersion, categoryDTO(c))
}

func (s *Server) createCategory(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		Code                      string `json:"code"`
		Name                      string `json:"name"`
		DefaultDepreciationMethod string `json:"defaultDepreciationMethod"`
		DefaultUsefulLifeMonths   *int32 `json:"defaultUsefulLifeMonths,omitempty"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code and name are required"))
		return
	}
	method := strings.TrimSpace(req.DefaultDepreciationMethod)
	if method == "" {
		method = "StraightLine"
	}
	if !validMethod(method) {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_METHOD", "invalid depreciation method"))
		return
	}
	created, err := s.repo.CreateCategory(r.Context(), scope, fadb.CreateAssetCategoryParams{
		OrganizationID: scope[0], Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name),
		DefaultDepreciationMethod: method, DefaultUsefulLifeMonths: req.DefaultUsefulLifeMonths, IsActive: true, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{
		"id": created.ID, "code": strings.TrimSpace(req.Code), "name": strings.TrimSpace(req.Name), "rowVersion": created.RowVersion,
	})
}

func (s *Server) updateCategory(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		Name                      string `json:"name"`
		DefaultDepreciationMethod string `json:"defaultDepreciationMethod"`
		DefaultUsefulLifeMonths   *int32 `json:"defaultUsefulLifeMonths,omitempty"`
		IsActive                  *bool  `json:"isActive,omitempty"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name is required"))
		return
	}
	method := strings.TrimSpace(req.DefaultDepreciationMethod)
	if method == "" {
		method = "StraightLine"
	}
	if !validMethod(method) {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_METHOD", "invalid depreciation method"))
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	next, err := s.repo.UpdateCategory(r.Context(), scope, fadb.UpdateAssetCategoryParams{
		ID: id, RowVersion: rowVer, Name: strings.TrimSpace(req.Name),
		DefaultDepreciationMethod: method, DefaultUsefulLifeMonths: req.DefaultUsefulLifeMonths,
		IsActive: active, UpdatedBy: &user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "rowVersion": next})
}

func (s *Server) deleteCategory(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.repo.DeleteCategory(r.Context(), scope, fadb.DeleteAssetCategoryParams{
		ID: id, RowVersion: rowVer, UpdatedBy: &user,
	}); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
