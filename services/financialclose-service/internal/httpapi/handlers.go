package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/financialclose-service/internal/fcldb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type Store interface {
	CreateTemplate(ctx context.Context, scope []uuid.UUID, arg fcldb.CreateChecklistTemplateParams, tasks []fcldb.CreateTaskTemplateParams) (fcldb.CreateChecklistTemplateRow, error)
	ListTemplates(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]fcldb.FclClosingChecklistTemplate, int64, error)
	CreateClosingRun(ctx context.Context, scope []uuid.UUID, arg fcldb.CreateClosingRunParams, templateID *uuid.UUID) (fcldb.CreateClosingRunRow, error)
	GetClosingRun(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (fcldb.FclClosingRun, []fcldb.FclClosingTask, error)
	ListClosingRuns(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]fcldb.FclClosingRun, int64, error)
	CompleteTask(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, actor uuid.UUID) (int32, error)
	TransitionClosingRunAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, targetStatus string, actor, corr uuid.UUID) (int32, error)
	UpsertPeriodStatus(ctx context.Context, scope []uuid.UUID, arg fcldb.UpsertPeriodStatusParams) (fcldb.UpsertPeriodStatusRow, error)
	ListPeriodStatuses(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]fcldb.FclPeriodStatus, int64, error)
	RequestReopening(ctx context.Context, scope []uuid.UUID, arg fcldb.CreatePeriodReopeningParams) (fcldb.CreatePeriodReopeningRow, error)
	ApproveReopeningAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, approver, corr uuid.UUID) (int32, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server { return &Server{repo: repo, idem: httpx.NewIdemStore()} }

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /closing-templates", s.listTemplates)
	mux.HandleFunc("POST /closing-templates", s.createTemplate)
	mux.HandleFunc("GET /closing-runs", s.listRuns)
	mux.HandleFunc("POST /closing-runs", s.createRun)
	mux.HandleFunc("GET /closing-runs/{id}", s.getRun)
	mux.HandleFunc("POST /closing-runs/{id}/close", func(w http.ResponseWriter, r *http.Request) { s.transitionRun(w, r, "Closed") })
	mux.HandleFunc("POST /closing-runs/{id}/lock", func(w http.ResponseWriter, r *http.Request) { s.transitionRun(w, r, "Locked") })
	mux.HandleFunc("POST /closing-tasks/{id}/complete", s.completeTask)
	mux.HandleFunc("GET /period-statuses", s.listPeriodStatuses)
	mux.HandleFunc("PUT /period-statuses", s.upsertPeriodStatus)
	mux.HandleFunc("POST /period-reopenings", s.requestReopening)
	mux.HandleFunc("POST /period-reopenings/{id}/approve", s.approveReopening)
}

func requireRead(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	if err := httpx.RequirePermission(r, "financialclose:read"); err != nil {
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
	if err := httpx.RequirePermission(r, "financialclose:write"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
}

func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListTemplates(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: rows, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createTemplate(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		Code                     string    `json:"code"`
		Name                     string    `json:"name"`
		FinancialOrganizationRef uuid.UUID `json:"financialOrganizationRef"`
		Tasks                    []struct {
			TaskOrder   int32  `json:"taskOrder"`
			TaskName    string `json:"taskName"`
			IsMandatory *bool  `json:"isMandatory,omitempty"`
		} `json:"tasks"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" || req.FinancialOrganizationRef == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("code, name, and financialOrganizationRef are required"))
		return
	}
	tasks := make([]fcldb.CreateTaskTemplateParams, 0, len(req.Tasks))
	for _, t := range req.Tasks {
		mand := true
		if t.IsMandatory != nil {
			mand = *t.IsMandatory
		}
		tasks = append(tasks, fcldb.CreateTaskTemplateParams{
			OrganizationID: scope[0], TaskOrder: t.TaskOrder, TaskName: strings.TrimSpace(t.TaskName),
			IsMandatory: mand, CreatedBy: user,
		})
	}
	created, err := s.repo.CreateTemplate(r.Context(), scope, fcldb.CreateChecklistTemplateParams{
		OrganizationID: scope[0], Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name),
		FinancialOrganizationRef: req.FinancialOrganizationRef, CreatedBy: user,
	}, tasks)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID, "rowVersion": created.RowVersion})
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var status *string
	if st := strings.TrimSpace(r.URL.Query().Get("status")); st != "" {
		status = &st
	}
	rows, total, err := s.repo.ListClosingRuns(r.Context(), scope, status, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: rows, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createRun(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		FinancialOrganizationRef uuid.UUID  `json:"financialOrganizationRef"`
		FiscalPeriodRef          uuid.UUID  `json:"fiscalPeriodRef"`
		RunType                  string     `json:"runType"`
		TemplateID               *uuid.UUID `json:"templateId,omitempty"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if req.FinancialOrganizationRef == uuid.Nil || req.FiscalPeriodRef == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("financialOrganizationRef and fiscalPeriodRef are required"))
		return
	}
	rt := strings.TrimSpace(req.RunType)
	if rt == "" {
		rt = "PeriodEnd"
	}
	if rt != "PeriodEnd" && rt != "YearEnd" {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_RUN_TYPE", "runType must be PeriodEnd or YearEnd"))
		return
	}
	created, err := s.repo.CreateClosingRun(r.Context(), scope, fcldb.CreateClosingRunParams{
		OrganizationID: scope[0], FinancialOrganizationRef: req.FinancialOrganizationRef,
		FiscalPeriodRef: req.FiscalPeriodRef, RunType: rt, CreatedBy: user,
	}, req.TemplateID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{
		"id": created.ID, "status": created.Status, "rowVersion": created.RowVersion,
	})
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	run, tasks, err := s.repo.GetClosingRun(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, run.RowVersion, map[string]any{"run": run, "tasks": tasks})
}

func (s *Server) completeTask(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.repo.CompleteTask(r.Context(), scope, id, rowVer, user)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": "Completed", "rowVersion": next})
}

func (s *Server) transitionRun(w http.ResponseWriter, r *http.Request, target string) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.repo.TransitionClosingRunAndEmit(r.Context(), scope, id, rowVer, target, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": target, "rowVersion": next})
}

func (s *Server) listPeriodStatuses(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListPeriodStatuses(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: rows, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) upsertPeriodStatus(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		FinancialOrganizationRef uuid.UUID `json:"financialOrganizationRef"`
		FiscalPeriodRef          uuid.UUID `json:"fiscalPeriodRef"`
		Module                   string    `json:"module"`
		Status                   string    `json:"status"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.FinancialOrganizationRef == uuid.Nil || req.FiscalPeriodRef == uuid.Nil || strings.TrimSpace(req.Module) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("financialOrganizationRef, fiscalPeriodRef, and module are required"))
		return
	}
	if req.Status != "Open" && req.Status != "PendingClose" && req.Status != "Closed" && req.Status != "Locked" {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_STATUS", "invalid period status"))
		return
	}
	row, err := s.repo.UpsertPeriodStatus(r.Context(), scope, fcldb.UpsertPeriodStatusParams{
		OrganizationID: scope[0], FinancialOrganizationRef: req.FinancialOrganizationRef,
		FiscalPeriodRef: req.FiscalPeriodRef, Module: strings.TrimSpace(req.Module), Status: req.Status, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, map[string]any{"id": row.ID, "status": row.Status, "rowVersion": row.RowVersion})
}

func (s *Server) requestReopening(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		FinancialOrganizationRef uuid.UUID `json:"financialOrganizationRef"`
		FiscalPeriodRef          uuid.UUID `json:"fiscalPeriodRef"`
		Reason                   string    `json:"reason"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.FinancialOrganizationRef == uuid.Nil || req.FiscalPeriodRef == uuid.Nil || strings.TrimSpace(req.Reason) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("financialOrganizationRef, fiscalPeriodRef, and reason are required"))
		return
	}
	created, err := s.repo.RequestReopening(r.Context(), scope, fcldb.CreatePeriodReopeningParams{
		OrganizationID: scope[0], FinancialOrganizationRef: req.FinancialOrganizationRef,
		FiscalPeriodRef: req.FiscalPeriodRef, RequestedBy: user, Reason: strings.TrimSpace(req.Reason), CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID, "rowVersion": created.RowVersion})
}

func (s *Server) approveReopening(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.repo.ApproveReopeningAndEmit(r.Context(), scope, id, rowVer, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": "Reopened", "rowVersion": next})
}
