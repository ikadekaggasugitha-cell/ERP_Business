package httpapi

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/project-service/internal/prjdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var st *string
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
		st = &v
	}
	rows, total, err := s.store.ListProjects(r.Context(), scope, st, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]projectDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toProjectDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		Code            string  `json:"code"`
		Name            string  `json:"name"`
		ProjectType     string  `json:"projectType"`
		OrganizationRef *string `json:"organizationRef"`
		CurrencyCode    string  `json:"currencyCode"`
		StartDate       *string `json:"startDate"`
		EndDate         *string `json:"endDate"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	ccy := strings.ToUpper(strings.TrimSpace(req.CurrencyCode))
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.ProjectType) == "" || len(ccy) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("code, name, projectType, and 3-letter currencyCode are required"))
		return
	}
	startDate, err := parseOptDate(req.StartDate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	endDate, err := parseOptDate(req.EndDate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if startDate.Valid && endDate.Valid && endDate.Time.Before(startDate.Time) {
		httpx.WriteError(w, r, httpx.BadRequest("endDate cannot precede startDate"))
		return
	}
	orgRef := scope[0]
	if req.OrganizationRef != nil && strings.TrimSpace(*req.OrganizationRef) != "" {
		parsed, e := uuid.Parse(*req.OrganizationRef)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid organizationRef"))
			return
		}
		orgRef = parsed
	}
	out, err := s.store.CreateProject(r.Context(), scope, prjdb.CreateProjectParams{
		OrganizationID: scope[0], Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name),
		ProjectType: strings.TrimSpace(req.ProjectType), OrganizationRef: orgRef,
		CurrencyCode: ccy, StartDate: startDate, EndDate: endDate, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "Draft", "rowVersion": out.RowVersion})
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	row, err := s.store.GetProject(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, toProjectDTO(row))
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		Name        string  `json:"name"`
		ProjectType string  `json:"projectType"`
		StartDate   *string `json:"startDate"`
		EndDate     *string `json:"endDate"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.ProjectType) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name and projectType are required"))
		return
	}
	startDate, err := parseOptDate(req.StartDate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	endDate, err := parseOptDate(req.EndDate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if startDate.Valid && endDate.Valid && endDate.Time.Before(startDate.Time) {
		httpx.WriteError(w, r, httpx.BadRequest("endDate cannot precede startDate"))
		return
	}
	next, err := s.store.UpdateProject(r.Context(), scope, prjdb.UpdateProjectParams{
		ID: id, Name: strings.TrimSpace(req.Name), ProjectType: strings.TrimSpace(req.ProjectType),
		StartDate: startDate, EndDate: endDate, UpdatedBy: &user, RowVersion: ver,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "rowVersion": next})
}

func (s *Server) transitionProject(from, to string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope, user, ok := requireWrite(w, r)
		if !ok {
			return
		}
		id, err := httpx.ParsePathUUID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		ver, err := httpx.IfMatch(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req struct{}
		idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
		if handled {
			return
		}
		defer release()
		next, err := s.store.TransitionProject(r.Context(), scope, id, user, ver, from, to)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": to, "rowVersion": next})
	}
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.store.DeleteProject(r.Context(), scope, id, user, ver); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listWBS(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	pid, err := httpx.ParsePathUUID(r, "projectId")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListWBSElements(r.Context(), scope, pid)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]wbsDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toWBSDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) createWBS(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		ProjectID string  `json:"projectId"`
		ParentID  *string `json:"parentId"`
		Code      string  `json:"code"`
		Name      string  `json:"name"`
		LevelNo   int32   `json:"levelNo"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	pid, err := uuid.Parse(req.ProjectID)
	if err != nil || strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("projectId, code, and name are required"))
		return
	}
	var parentID *uuid.UUID
	if req.ParentID != nil && strings.TrimSpace(*req.ParentID) != "" {
		p, e := uuid.Parse(*req.ParentID)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid parentId"))
			return
		}
		parentID = &p
	}
	level := req.LevelNo
	if level <= 0 {
		level = 1
	}
	out, err := s.store.CreateWBSElement(r.Context(), scope, prjdb.CreateWBSElementParams{
		OrganizationID: scope[0], ProjectID: pid, ParentID: parentID,
		Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name), LevelNo: level, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) updateWBS(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		ParentID *string `json:"parentId"`
		Name     string  `json:"name"`
		LevelNo  int32   `json:"levelNo"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name is required"))
		return
	}
	var parentID *uuid.UUID
	if req.ParentID != nil && strings.TrimSpace(*req.ParentID) != "" {
		p, e := uuid.Parse(*req.ParentID)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid parentId"))
			return
		}
		parentID = &p
	}
	level := req.LevelNo
	if level <= 0 {
		level = 1
	}
	next, err := s.store.UpdateWBSElement(r.Context(), scope, prjdb.UpdateWBSElementParams{
		ID: id, ParentID: parentID, Name: strings.TrimSpace(req.Name), LevelNo: level, UpdatedBy: &user, RowVersion: ver,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "rowVersion": next})
}

func (s *Server) deleteWBS(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.store.DeleteWBSElement(r.Context(), scope, id, user, ver); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
