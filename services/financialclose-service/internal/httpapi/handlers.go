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

	"github.com/sinergiinvestasiproperti/erp/services/financialclose-service/internal/fcldb"
	"github.com/sinergiinvestasiproperti/erp/services/financialclose-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type Store interface {
	CreateTemplate(ctx context.Context, scope []uuid.UUID, arg fcldb.CreateChecklistTemplateParams, tasks []fcldb.CreateTaskTemplateParams) (fcldb.CreateChecklistTemplateRow, error)
	ListTemplates(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]fcldb.FclClosingChecklistTemplate, int64, error)
	CreateClosingRun(ctx context.Context, scope []uuid.UUID, arg fcldb.CreateClosingRunParams, templateID *uuid.UUID) (fcldb.CreateClosingRunRow, error)
	GetClosingRun(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (fcldb.FclClosingRun, []fcldb.FclClosingTask, error)
	ListClosingRuns(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]fcldb.FclClosingRun, int64, error)
	CompleteTask(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, actor uuid.UUID) (int32, error)
	RunFXRevaluationAndEmit(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, positions []repository.FXPositionInput, corr uuid.UUID) (repository.FXRevaluationResult, error)
	TransitionClosingRunAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, targetStatus string, actor, corr uuid.UUID) (int32, error)
	UpsertPeriodStatus(ctx context.Context, scope []uuid.UUID, arg fcldb.UpsertPeriodStatusParams) (fcldb.UpsertPeriodStatusRow, error)
	ListPeriodStatuses(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]fcldb.FclPeriodStatus, int64, error)
	ListReopenings(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]fcldb.FclPeriodReopening, int64, error)
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
	mux.HandleFunc("POST /closing-runs/{id}/fx-revaluation", s.fxRevaluation)
	mux.HandleFunc("POST /closing-runs/{id}/auto-reversals", s.autoReversals)
	mux.HandleFunc("POST /closing-runs/{id}/verify-subledgers", s.verifySubledgers)
	mux.HandleFunc("POST /closing-runs/{id}/close", func(w http.ResponseWriter, r *http.Request) { s.transitionRun(w, r, "Closed") })
	mux.HandleFunc("POST /closing-runs/{id}/lock", func(w http.ResponseWriter, r *http.Request) { s.transitionRun(w, r, "Locked") })
	mux.HandleFunc("POST /closing-tasks/{id}/complete", s.completeTask)
	mux.HandleFunc("GET /period-statuses", s.listPeriodStatuses)
	mux.HandleFunc("PUT /period-statuses", s.upsertPeriodStatus)
	mux.HandleFunc("GET /period-reopenings", s.listReopenings)
	mux.HandleFunc("POST /period-reopenings", s.requestReopening)
	mux.HandleFunc("POST /period-reopenings/{id}/approve", s.approveReopening)
}

func beginIdemOptional(w http.ResponseWriter, r *http.Request, idem *httpx.IdemStore, scope []uuid.UUID, dst any) (string, string, func(), bool) {
	if dst == nil {
		var empty map[string]any
		dst = &empty
	}
	if r.Body == nil {
		r.Body = io.NopCloser(strings.NewReader("{}"))
	} else {
		raw, err := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
		if err == nil && len(bytes.TrimSpace(raw)) == 0 {
			r.Body = io.NopCloser(strings.NewReader("{}"))
		} else {
			r.Body = io.NopCloser(bytes.NewReader(raw))
		}
	}
	return httpx.BeginIdempotent(w, r, idem, scope, dst)
}

func fmtTS(ts pgtype.Timestamptz) any {
	if !ts.Valid {
		return nil
	}
	return ts.Time.UTC().Format(time.RFC3339)
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

func templateDTO(t fcldb.FclClosingChecklistTemplate) map[string]any {
	return map[string]any{
		"id":                       t.ID,
		"organizationId":           t.OrganizationID,
		"code":                     t.Code,
		"name":                     t.Name,
		"financialOrganizationRef": t.FinancialOrganizationRef,
		"isSystem":                 t.OrganizationID == platform.SystemOrgID,
		"rowVersion":               t.RowVersion,
	}
}

func runDTO(r fcldb.FclClosingRun) map[string]any {
	return map[string]any{
		"id":                       r.ID,
		"organizationId":           r.OrganizationID,
		"financialOrganizationRef": r.FinancialOrganizationRef,
		"fiscalPeriodRef":          r.FiscalPeriodRef,
		"runType":                  r.RunType,
		"status":                   r.Status,
		"closedAt":                 fmtTS(r.ClosedAt),
		"closedBy":                 r.ClosedBy,
		"rowVersion":               r.RowVersion,
	}
}

func taskDTO(t fcldb.FclClosingTask) map[string]any {
	return map[string]any{
		"id":             t.ID,
		"organizationId": t.OrganizationID,
		"closingRunId":   t.ClosingRunID,
		"taskName":       t.TaskName,
		"isMandatory":    t.IsMandatory,
		"assigneeRef":    t.AssigneeRef,
		"status":         t.Status,
		"completedAt":    fmtTS(t.CompletedAt),
		"rowVersion":     t.RowVersion,
	}
}

func periodStatusDTO(p fcldb.FclPeriodStatus) map[string]any {
	return map[string]any{
		"id":                       p.ID,
		"organizationId":           p.OrganizationID,
		"financialOrganizationRef": p.FinancialOrganizationRef,
		"fiscalPeriodRef":          p.FiscalPeriodRef,
		"module":                   p.Module,
		"status":                   p.Status,
		"rowVersion":               p.RowVersion,
	}
}

func reopeningDTO(rp fcldb.FclPeriodReopening) map[string]any {
	return map[string]any{
		"id":                       rp.ID,
		"organizationId":           rp.OrganizationID,
		"financialOrganizationRef": rp.FinancialOrganizationRef,
		"fiscalPeriodRef":          rp.FiscalPeriodRef,
		"requestedBy":              rp.RequestedBy,
		"approvedBy":               rp.ApprovedBy,
		"reason":                   rp.Reason,
		"reopenedAt":               fmtTS(rp.ReopenedAt),
		"rowVersion":               rp.RowVersion,
	}
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
	out := make([]map[string]any, 0, len(rows))
	for _, t := range rows {
		out = append(out, templateDTO(t))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
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
	out := make([]map[string]any, 0, len(rows))
	for _, run := range rows {
		out = append(out, runDTO(run))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
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
	taskDTOs := make([]map[string]any, 0, len(tasks))
	for _, t := range tasks {
		taskDTOs = append(taskDTOs, taskDTO(t))
	}
	httpx.WriteIDResponse(w, http.StatusOK, run.RowVersion, map[string]any{"run": runDTO(run), "tasks": taskDTOs})
}

func (s *Server) fxRevaluation(w http.ResponseWriter, r *http.Request) {
	scope, _, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		Positions []repository.FXPositionInput `json:"positions"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if len(req.Positions) == 0 {
		httpx.WriteError(w, r, httpx.BadRequest("at least one FX position is required"))
		return
	}
	res, err := s.repo.RunFXRevaluationAndEmit(r.Context(), scope, id, req.Positions, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, 0, res)
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
	idemKey, reqHash, release, done := beginIdemOptional(w, r, s.idem, scope, nil)
	if done {
		return
	}
	defer release()
	next, err := s.repo.CompleteTask(r.Context(), scope, id, rowVer, user)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, next, map[string]any{"id": id, "status": "Completed", "rowVersion": next})
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
	idemKey, reqHash, release, done := beginIdemOptional(w, r, s.idem, scope, nil)
	if done {
		return
	}
	defer release()
	next, err := s.repo.TransitionClosingRunAndEmit(r.Context(), scope, id, rowVer, target, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, next, map[string]any{"id": id, "status": target, "rowVersion": next})
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
	out := make([]map[string]any, 0, len(rows))
	for _, p := range rows {
		out = append(out, periodStatusDTO(p))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
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

func (s *Server) listReopenings(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListReopenings(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, rp := range rows {
		out = append(out, reopeningDTO(rp))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
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
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
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
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID, "rowVersion": created.RowVersion})
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
	idemKey, reqHash, release, done := beginIdemOptional(w, r, s.idem, scope, nil)
	if done {
		return
	}
	defer release()
	next, err := s.repo.ApproveReopeningAndEmit(r.Context(), scope, id, rowVer, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, next, map[string]any{"id": id, "status": "Reopened", "rowVersion": next})
}

func (s *Server) autoReversals(w http.ResponseWriter, r *http.Request) {
	scope, _, ok := requireWrite(w, r)
	if !ok {
		return
	}
	runID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		ReversalDate string `json:"reversalDate"`
		Entries      []struct {
			Reference         string `json:"reference"`
			AccountCode       string `json:"accountCode"`
			ContraAccountCode string `json:"contraAccountCode"`
			Description       string `json:"description"`
			Amount            string `json:"amount"`
		} `json:"entries"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	revDate := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	if strings.TrimSpace(req.ReversalDate) != "" {
		if _, perr := platform.ParseDate(req.ReversalDate); perr != nil {
			httpx.WriteError(w, r, httpx.BadRequest("reversalDate must be YYYY-MM-DD"))
			return
		}
		revDate = strings.TrimSpace(req.ReversalDate)
	}
	if _, _, err := s.repo.GetClosingRun(r.Context(), scope, runID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	total := new(big.Rat)
	reversals := make([]map[string]any, 0, len(req.Entries))
	for i, e := range req.Entries {
		num, perr := platform.ParseNumeric(e.Amount)
		if perr != nil {
			httpx.WriteError(w, r, httpx.BadRequest("entries[].amount is invalid"))
			return
		}
		amt := platform.NumericToRat(num)
		if amt == nil || amt.Sign() <= 0 {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_AMOUNT", "reversal entry amount must be > 0"))
			return
		}
		total.Add(total, amt)
		ref := strings.TrimSpace(e.Reference)
		if ref == "" {
			ref = "REV-" + runID.String()[:8]
		}
		reversals = append(reversals, map[string]any{
			"lineNo":      i + 1,
			"reference":   ref,
			"reversalDate": revDate,
			"debitAccount":  strings.TrimSpace(e.ContraAccountCode),
			"creditAccount": strings.TrimSpace(e.AccountCode),
			"description":   strings.TrimSpace(e.Description),
			"amount":        amt.FloatString(4),
			"status":        "PreviewGenerated",
		})
	}

	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, 1, map[string]any{
		"closingRunId": runID,
		"reversalDate": revDate,
		"count":        len(reversals),
		"totalAmount":  total.FloatString(4),
		"reversals":    reversals,
	})
}

func (s *Server) verifySubledgers(w http.ResponseWriter, r *http.Request) {
	scope, _, ok := requireWrite(w, r)
	if !ok {
		return
	}
	runID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	run, tasks, err := s.repo.GetClosingRun(r.Context(), scope, runID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	subledgers := []string{"GL", "AP", "AR", "INVENTORY", "FIXED_ASSET"}
	openTaskBySub := make(map[string]int, len(subledgers))
	for _, t := range tasks {
		if strings.EqualFold(t.Status, "Completed") || strings.EqualFold(t.Status, "Skipped") {
			continue
		}
		upper := strings.ToUpper(t.TaskName)
		for _, sub := range subledgers {
			if strings.Contains(upper, sub) {
				openTaskBySub[sub]++
			}
		}
	}

	checks := make([]map[string]any, 0, len(subledgers))
	passed := 0
	failed := 0
	for _, sub := range subledgers {
		if n := openTaskBySub[sub]; n > 0 {
			failed++
			checks = append(checks, map[string]any{
				"subledger": sub,
				"status":    "FAILED",
				"openTasks": n,
				"detail":    "pending closing tasks remain for subledger",
			})
		} else {
			passed++
			checks = append(checks, map[string]any{
				"subledger": sub,
				"status":    "PASSED",
				"openTasks": 0,
				"detail":    "subledger ready for period close",
			})
		}
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"closingRunId": run.ID,
		"runStatus":    run.Status,
		"readyToClose": failed == 0,
		"passedCount":  passed,
		"failedCount":  failed,
		"checks":       checks,
	})
}
