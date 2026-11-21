package httpapi

import (
	"bytes"
	"context"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/project-service/internal/prjdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type ProjectStore interface {
	CreateProject(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateProjectParams) (prjdb.CreateProjectRow, error)
	GetProject(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (prjdb.PrjProject, error)
	ListProjects(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]prjdb.PrjProject, int64, error)
	UpdateProject(ctx context.Context, scope []uuid.UUID, arg prjdb.UpdateProjectParams) (int32, error)
	TransitionProject(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32, fromStatus, toStatus string) (int32, error)
	DeleteProject(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error
	CreateWBSElement(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateWBSElementParams) (prjdb.CreateWBSElementRow, error)
	ListWBSElements(ctx context.Context, scope []uuid.UUID, projectID uuid.UUID) ([]prjdb.PrjWbsElement, error)
	UpdateWBSElement(ctx context.Context, scope []uuid.UUID, arg prjdb.UpdateWBSElementParams) (int32, error)
	DeleteWBSElement(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error
	CreateBudget(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateProjectBudgetParams) (prjdb.CreateProjectBudgetRow, error)
	ListBudgets(ctx context.Context, scope []uuid.UUID, projectID uuid.UUID) ([]prjdb.PrjProjectBudget, error)
	ApproveBudget(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error)
	CreateCost(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateProjectCostParams) (prjdb.CreateProjectCostRow, error)
	ListCosts(ctx context.Context, scope []uuid.UUID, projectID uuid.UUID) ([]prjdb.PrjProjectCost, error)
	UpsertProgress(ctx context.Context, scope []uuid.UUID, arg prjdb.UpsertProjectProgressParams) (prjdb.UpsertProjectProgressRow, error)
	ListProgress(ctx context.Context, scope []uuid.UUID, wbsID uuid.UUID) ([]prjdb.PrjProjectProgress, error)
	CreateBilling(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateProjectBillingParams) (prjdb.CreateProjectBillingRow, error)
	ListBillings(ctx context.Context, scope []uuid.UUID, projectID *uuid.UUID, limit, offset int32) ([]prjdb.PrjProjectBilling, int64, error)
	ApproveBillingAndEmit(ctx context.Context, scope []uuid.UUID, id, org, actor, corr uuid.UUID, version int32, customerID *uuid.UUID, arRef *string) (int32, string, error)
	CreateChangeOrder(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateChangeOrderParams) (prjdb.CreateChangeOrderRow, error)
	ListChangeOrders(ctx context.Context, scope []uuid.UUID, projectID uuid.UUID) ([]prjdb.PrjChangeOrder, error)
	DecideChangeOrder(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32, status string) (int32, error)
}

type Server struct {
	store ProjectStore
	idem  *httpx.IdemStore
}

func NewServer(store ProjectStore) *Server {
	return &Server{store: store, idem: httpx.NewIdemStore()}
}

func beginIdemOptional(w http.ResponseWriter, r *http.Request, idem *httpx.IdemStore, scope []uuid.UUID, dst any) (string, string, func(), bool) {
	if r.Body == nil {
		r.Body = io.NopCloser(strings.NewReader("{}"))
	} else {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("failed to read request body"))
			return "", "", func() {}, true
		}
		if len(bytes.TrimSpace(raw)) == 0 {
			raw = []byte("{}")
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))
	}
	return httpx.BeginIdempotent(w, r, idem, scope, dst)
}

func fmtOptTS(ts pgtype.Timestamptz) *string {
	if !ts.Valid {
		return nil
	}
	s := ts.Time.UTC().Format(time.RFC3339)
	return &s
}

func fmtOptDate(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := platform.FormatDate(d)
	return &s
}

type projectDTO struct {
	ID              uuid.UUID `json:"id"`
	OrganizationID  uuid.UUID `json:"organizationId"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	ProjectType     string    `json:"projectType"`
	OrganizationRef uuid.UUID `json:"organizationRef"`
	CurrencyCode    string    `json:"currencyCode"`
	StartDate       *string   `json:"startDate,omitempty"`
	EndDate         *string   `json:"endDate,omitempty"`
	Status          string    `json:"status"`
	IsSystem        bool      `json:"isSystem"`
	CreatedAt       *string   `json:"createdAt,omitempty"`
	UpdatedAt       *string   `json:"updatedAt,omitempty"`
	RowVersion      int32     `json:"rowVersion"`
}

func toProjectDTO(r prjdb.PrjProject) projectDTO {
	return projectDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, Code: r.Code, Name: r.Name,
		ProjectType: r.ProjectType, OrganizationRef: r.OrganizationRef,
		CurrencyCode: r.CurrencyCode, StartDate: fmtOptDate(r.StartDate), EndDate: fmtOptDate(r.EndDate),
		Status: r.Status, IsSystem: r.OrganizationID == platform.SystemOrgID,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type wbsDTO struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	ProjectID      uuid.UUID  `json:"projectId"`
	ParentID       *uuid.UUID `json:"parentId,omitempty"`
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	LevelNo        int32      `json:"levelNo"`
	IsSystem       bool       `json:"isSystem"`
	CreatedAt      *string    `json:"createdAt,omitempty"`
	UpdatedAt      *string    `json:"updatedAt,omitempty"`
	RowVersion     int32      `json:"rowVersion"`
}

func toWBSDTO(r prjdb.PrjWbsElement) wbsDTO {
	return wbsDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, ProjectID: r.ProjectID, ParentID: r.ParentID,
		Code: r.Code, Name: r.Name, LevelNo: r.LevelNo,
		IsSystem: r.OrganizationID == platform.SystemOrgID,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type budgetDTO struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	ProjectID      uuid.UUID `json:"projectId"`
	WBSElementID   uuid.UUID `json:"wbsElementId"`
	Version        int32     `json:"version"`
	BudgetAmount   string    `json:"budgetAmount"`
	Status         string    `json:"status"`
	IsBaseline     bool      `json:"isBaseline"`
	CreatedAt      *string   `json:"createdAt,omitempty"`
	UpdatedAt      *string   `json:"updatedAt,omitempty"`
	RowVersion     int32     `json:"rowVersion"`
}

func toBudgetDTO(r prjdb.PrjProjectBudget) budgetDTO {
	return budgetDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, ProjectID: r.ProjectID, WBSElementID: r.WbsElementID,
		Version: r.Version, BudgetAmount: platform.FormatNumeric(r.BudgetAmount, 4),
		Status: r.Status, IsBaseline: r.IsBaseline,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type costDTO struct {
	ID                uuid.UUID `json:"id"`
	OrganizationID    uuid.UUID `json:"organizationId"`
	ProjectID         uuid.UUID `json:"projectId"`
	WBSElementID      uuid.UUID `json:"wbsElementId"`
	CostType          string    `json:"costType"`
	Amount            string    `json:"amount"`
	SourceDocumentRef *string   `json:"sourceDocumentRef,omitempty"`
	PostingDate       string    `json:"postingDate"`
	CreatedAt         *string   `json:"createdAt,omitempty"`
	UpdatedAt         *string   `json:"updatedAt,omitempty"`
	RowVersion        int32     `json:"rowVersion"`
}

func toCostDTO(r prjdb.PrjProjectCost) costDTO {
	return costDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, ProjectID: r.ProjectID, WBSElementID: r.WbsElementID,
		CostType: r.CostType, Amount: platform.FormatNumeric(r.Amount, 4),
		SourceDocumentRef: r.SourceDocumentRef, PostingDate: platform.FormatDate(r.PostingDate),
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type progressDTO struct {
	ID                  uuid.UUID `json:"id"`
	OrganizationID      uuid.UUID `json:"organizationId"`
	WBSElementID        uuid.UUID `json:"wbsElementId"`
	PeriodDate          string    `json:"periodDate"`
	PhysicalProgressPct string    `json:"physicalProgressPct"`
	RecordedAt          *string   `json:"recordedAt,omitempty"`
	CreatedAt           *string   `json:"createdAt,omitempty"`
	UpdatedAt           *string   `json:"updatedAt,omitempty"`
	RowVersion          int32     `json:"rowVersion"`
}

func toProgressDTO(r prjdb.PrjProjectProgress) progressDTO {
	return progressDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, WBSElementID: r.WbsElementID,
		PeriodDate: platform.FormatDate(r.PeriodDate), PhysicalProgressPct: platform.FormatNumeric(r.PhysicalProgressPct, 2),
		RecordedAt: fmtOptTS(r.RecordedAt), CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type billingDTO struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	ProjectID      uuid.UUID `json:"projectId"`
	BillingType    string    `json:"billingType"`
	Amount         string    `json:"amount"`
	ArInvoiceRef   *string   `json:"arInvoiceRef,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      *string   `json:"createdAt,omitempty"`
	UpdatedAt      *string   `json:"updatedAt,omitempty"`
	RowVersion     int32     `json:"rowVersion"`
}

func toBillingDTO(r prjdb.PrjProjectBilling) billingDTO {
	return billingDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, ProjectID: r.ProjectID,
		BillingType: r.BillingType, Amount: platform.FormatNumeric(r.Amount, 4),
		ArInvoiceRef: r.ArInvoiceRef, Status: r.Status,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type changeOrderDTO struct {
	ID                 uuid.UUID `json:"id"`
	OrganizationID     uuid.UUID `json:"organizationId"`
	ProjectID          uuid.UUID `json:"projectId"`
	Description        string    `json:"description"`
	BudgetImpactAmount string    `json:"budgetImpactAmount"`
	ScheduleImpactDays int32     `json:"scheduleImpactDays"`
	Status             string    `json:"status"`
	CreatedAt          *string   `json:"createdAt,omitempty"`
	UpdatedAt          *string   `json:"updatedAt,omitempty"`
	RowVersion         int32     `json:"rowVersion"`
}

func toChangeOrderDTO(r prjdb.PrjChangeOrder) changeOrderDTO {
	return changeOrderDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, ProjectID: r.ProjectID,
		Description: r.Description, BudgetImpactAmount: platform.FormatNumeric(r.BudgetImpactAmount, 4),
		ScheduleImpactDays: r.ScheduleImpactDays, Status: r.Status,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /projects", s.listProjects)
	mux.HandleFunc("POST /projects", s.createProject)
	mux.HandleFunc("GET /projects/{id}", s.getProject)
	mux.HandleFunc("PUT /projects/{id}", s.updateProject)
	mux.HandleFunc("DELETE /projects/{id}", s.deleteProject)
	mux.HandleFunc("POST /projects/{id}/activate", s.transitionProject("Draft", "Active"))
	mux.HandleFunc("POST /projects/{id}/hold", s.transitionProject("Active", "OnHold"))
	mux.HandleFunc("POST /projects/{id}/close", s.transitionProject("Active", "Closed"))

	mux.HandleFunc("GET /projects/{projectId}/wbs-elements", s.listWBS)
	mux.HandleFunc("POST /wbs-elements", s.createWBS)
	mux.HandleFunc("PUT /wbs-elements/{id}", s.updateWBS)
	mux.HandleFunc("DELETE /wbs-elements/{id}", s.deleteWBS)

	mux.HandleFunc("GET /projects/{projectId}/budgets", s.listBudgets)
	mux.HandleFunc("POST /projects/{projectId}/budgets", s.createBudget)
	mux.HandleFunc("POST /project-budgets/{id}/approve", s.approveBudget)

	mux.HandleFunc("GET /projects/{projectId}/costs", s.listCosts)
	mux.HandleFunc("POST /projects/{projectId}/costs", s.createCost)

	mux.HandleFunc("GET /wbs-elements/{wbsId}/progress", s.listProgress)
	mux.HandleFunc("POST /wbs-elements/{wbsId}/progress", s.upsertProgress)

	mux.HandleFunc("GET /project-billings", s.listBillings)
	mux.HandleFunc("POST /project-billings", s.createBilling)
	mux.HandleFunc("POST /project-billings/{id}/approve", s.approveBilling)

	mux.HandleFunc("GET /projects/{projectId}/change-orders", s.listChangeOrders)
	mux.HandleFunc("POST /projects/{projectId}/change-orders", s.createChangeOrder)
	mux.HandleFunc("POST /change-orders/{id}/approve", s.decideChangeOrder("Approved"))
	mux.HandleFunc("POST /change-orders/{id}/reject", s.decideChangeOrder("Rejected"))
}

func requireRead(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	if err := httpx.RequirePermission(r, "project:read"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	return scope, true
}

func requireWrite(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	if err := httpx.RequirePermission(r, "project:write"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
}

func parseOptDate(raw *string) (pgtype.Date, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return pgtype.Date{}, nil
	}
	d, err := platform.ParseDate(*raw)
	if err != nil {
		return pgtype.Date{}, httpx.BadRequest("date must be YYYY-MM-DD")
	}
	return d, nil
}

func parsePositiveNumeric(raw, field string) (pgtype.Numeric, error) {
	num, err := platform.ParseNumeric(strings.TrimSpace(raw))
	if err != nil || platform.NumericToRat(num) == nil || platform.NumericToRat(num).Sign() <= 0 {
		return pgtype.Numeric{}, httpx.BadRequest(field + " must be a positive decimal string")
	}
	return num, nil
}

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
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
	pid, err := httpx.ParseUUIDParam("projectId", r.PathValue("projectId"))
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
	if err := s.store.DeleteWBSElement(r.Context(), scope, id, user, ver); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listBudgets(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	pid, err := httpx.ParseUUIDParam("projectId", r.PathValue("projectId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListBudgets(r.Context(), scope, pid)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]budgetDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toBudgetDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) createBudget(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	pid, err := httpx.ParseUUIDParam("projectId", r.PathValue("projectId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		WBSElementID string `json:"wbsElementId"`
		Version      int32  `json:"version"`
		BudgetAmount string `json:"budgetAmount"`
		IsBaseline   bool   `json:"isBaseline"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	wbsID, err := uuid.Parse(req.WBSElementID)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid wbsElementId is required"))
		return
	}
	amt, err := parsePositiveNumeric(req.BudgetAmount, "budgetAmount")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver := req.Version
	if ver <= 0 {
		ver = 1
	}
	out, err := s.store.CreateBudget(r.Context(), scope, prjdb.CreateProjectBudgetParams{
		OrganizationID: scope[0], ProjectID: pid, WbsElementID: wbsID,
		Version: ver, BudgetAmount: amt, IsBaseline: req.IsBaseline, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "Draft", "rowVersion": out.RowVersion})
}

func (s *Server) approveBudget(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
	var req struct{}
	idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	next, err := s.store.ApproveBudget(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Approved", "rowVersion": next})
}

func (s *Server) listCosts(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	pid, err := httpx.ParseUUIDParam("projectId", r.PathValue("projectId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListCosts(r.Context(), scope, pid)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]costDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toCostDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) createCost(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	pid, err := httpx.ParseUUIDParam("projectId", r.PathValue("projectId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		WBSElementID      string  `json:"wbsElementId"`
		CostType          string  `json:"costType"`
		Amount            string  `json:"amount"`
		SourceDocumentRef *string `json:"sourceDocumentRef"`
		PostingDate       string  `json:"postingDate"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	wbsID, err := uuid.Parse(req.WBSElementID)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid wbsElementId is required"))
		return
	}
	switch req.CostType {
	case "Labor", "Material", "Subcontractor", "Overhead":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("costType must be Labor, Material, Subcontractor, or Overhead"))
		return
	}
	amt, err := parsePositiveNumeric(req.Amount, "amount")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	pDate, err := platform.ParseDate(req.PostingDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("postingDate must be YYYY-MM-DD"))
		return
	}
	out, err := s.store.CreateCost(r.Context(), scope, prjdb.CreateProjectCostParams{
		OrganizationID: scope[0], ProjectID: pid, WbsElementID: wbsID,
		CostType: req.CostType, Amount: amt, SourceDocumentRef: req.SourceDocumentRef,
		PostingDate: pDate, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) listProgress(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	wbsID, err := httpx.ParseUUIDParam("wbsId", r.PathValue("wbsId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListProgress(r.Context(), scope, wbsID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]progressDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toProgressDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) upsertProgress(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	wbsID, err := httpx.ParseUUIDParam("wbsId", r.PathValue("wbsId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		PeriodDate          string `json:"periodDate"`
		PhysicalProgressPct string `json:"physicalProgressPct"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	pDate, err := platform.ParseDate(req.PeriodDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("periodDate must be YYYY-MM-DD"))
		return
	}
	pct, err := platform.ParseNumeric(strings.TrimSpace(req.PhysicalProgressPct))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid physicalProgressPct"))
		return
	}
	rat := platform.NumericToRat(pct)
	if rat == nil || rat.Sign() < 0 || rat.Cmp(big.NewRat(100, 1)) > 0 {
		httpx.WriteError(w, r, httpx.BadRequest("physicalProgressPct must be between 0 and 100"))
		return
	}
	out, err := s.store.UpsertProgress(r.Context(), scope, prjdb.UpsertProjectProgressParams{
		OrganizationID: scope[0], WbsElementID: wbsID, PeriodDate: pDate,
		PhysicalProgressPct: pct, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) listBillings(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var pid *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("projectId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid projectId"))
			return
		}
		pid = &parsed
	}
	rows, total, err := s.store.ListBillings(r.Context(), scope, pid, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]billingDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toBillingDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createBilling(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		ProjectID    string  `json:"projectId"`
		BillingType  string  `json:"billingType"`
		Amount       string  `json:"amount"`
		ArInvoiceRef *string `json:"arInvoiceRef"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	pid, err := uuid.Parse(req.ProjectID)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid projectId is required"))
		return
	}
	switch req.BillingType {
	case "Milestone", "Progress", "TimeMaterial":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("billingType must be Milestone, Progress, or TimeMaterial"))
		return
	}
	amt, err := parsePositiveNumeric(req.Amount, "amount")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := s.store.CreateBilling(r.Context(), scope, prjdb.CreateProjectBillingParams{
		OrganizationID: scope[0], ProjectID: pid, BillingType: req.BillingType,
		Amount: amt, ArInvoiceRef: req.ArInvoiceRef, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "Draft", "rowVersion": out.RowVersion})
}

func (s *Server) approveBilling(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
	var req struct {
		CustomerID   *string `json:"customerId"`
		ArInvoiceRef *string `json:"arInvoiceRef"`
	}
	idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	var custID *uuid.UUID
	if req.CustomerID != nil && strings.TrimSpace(*req.CustomerID) != "" {
		c, e := uuid.Parse(*req.CustomerID)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid customerId"))
			return
		}
		custID = &c
	}
	next, invRef, err := s.store.ApproveBillingAndEmit(r.Context(), scope, id, scope[0], user, httpx.CorrelationUUID(r), ver, custID, req.ArInvoiceRef)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Invoiced", "arInvoiceRef": invRef, "rowVersion": next})
}

func (s *Server) listChangeOrders(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	pid, err := httpx.ParseUUIDParam("projectId", r.PathValue("projectId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListChangeOrders(r.Context(), scope, pid)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]changeOrderDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toChangeOrderDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) createChangeOrder(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	pid, err := httpx.ParseUUIDParam("projectId", r.PathValue("projectId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		Description        string `json:"description"`
		BudgetImpactAmount string `json:"budgetImpactAmount"`
		ScheduleImpactDays int32  `json:"scheduleImpactDays"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	if strings.TrimSpace(req.Description) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("description is required"))
		return
	}
	amt, err := platform.ParseOptionalNumeric(req.BudgetImpactAmount)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid budgetImpactAmount"))
		return
	}
	out, err := s.store.CreateChangeOrder(r.Context(), scope, prjdb.CreateChangeOrderParams{
		OrganizationID: scope[0], ProjectID: pid, Description: strings.TrimSpace(req.Description),
		BudgetImpactAmount: amt, ScheduleImpactDays: req.ScheduleImpactDays, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "Draft", "rowVersion": out.RowVersion})
}

func (s *Server) decideChangeOrder(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope, user, ok := requireWrite(w, r)
		if !ok {
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
		var req struct{}
		idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
		if handled {
			return
		}
		defer release()
		next, err := s.store.DecideChangeOrder(r.Context(), scope, id, user, ver, status)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": status, "rowVersion": next})
	}
}
