package httpapi

import (
	"context"
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/budget-service/internal/budgetdb"
	"github.com/sinergiinvestasiproperti/erp/services/budget-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type Store interface {
	CreateBudget(ctx context.Context, scope []uuid.UUID, arg budgetdb.CreateBudgetParams) (budgetdb.CreateBudgetRow, error)
	GetBudget(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (budgetdb.BudBudget, []budgetdb.BudBudgetLine, error)
	ListBudgets(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]budgetdb.BudBudget, int64, error)
	TransitionBudgetAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, targetStatus string, actor, corr uuid.UUID) (int32, error)
	AddBudgetLine(ctx context.Context, scope []uuid.UUID, arg budgetdb.CreateBudgetLineParams) (budgetdb.CreateBudgetLineRow, error)
	CreatePolicy(ctx context.Context, scope []uuid.UUID, arg budgetdb.CreateBudgetPolicyParams) (budgetdb.CreateBudgetPolicyRow, error)
	ListPolicies(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]budgetdb.BudBudgetControlPolicy, int64, error)
	CheckAvailability(ctx context.Context, scope []uuid.UUID, budgetID uuid.UUID, accountRef *uuid.UUID, periodNo *int32, txType *string, requested *big.Rat) (repository.BudgetCheckResult, error)
	CreateCommitmentAndEmit(ctx context.Context, scope []uuid.UUID, arg budgetdb.CreateBudgetCommitmentParams, corr uuid.UUID) (budgetdb.CreateBudgetCommitmentRow, error)
	TransitionCommitment(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, status string, actor uuid.UUID) (int32, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server { return &Server{repo: repo, idem: httpx.NewIdemStore()} }

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /budgets", s.listBudgets)
	mux.HandleFunc("POST /budgets", s.createBudget)
	mux.HandleFunc("GET /budgets/{id}", s.getBudget)
	mux.HandleFunc("POST /budgets/{id}/submit", func(w http.ResponseWriter, r *http.Request) { s.transitionBudget(w, r, "Submitted") })
	mux.HandleFunc("POST /budgets/{id}/approve", func(w http.ResponseWriter, r *http.Request) { s.transitionBudget(w, r, "Approved") })
	mux.HandleFunc("POST /budgets/{id}/close", func(w http.ResponseWriter, r *http.Request) { s.transitionBudget(w, r, "Closed") })
	mux.HandleFunc("POST /budgets/{id}/lines", s.addBudgetLine)
	mux.HandleFunc("POST /budgets/check", s.checkBudget)
	mux.HandleFunc("GET /budget-policies", s.listPolicies)
	mux.HandleFunc("POST /budget-policies", s.createPolicy)
	mux.HandleFunc("POST /budget-commitments", s.createCommitment)
	mux.HandleFunc("POST /budget-commitments/{id}/release", func(w http.ResponseWriter, r *http.Request) { s.transitionCommitment(w, r, "Released") })
	mux.HandleFunc("POST /budget-commitments/{id}/consume", func(w http.ResponseWriter, r *http.Request) { s.transitionCommitment(w, r, "Consumed") })
}

func requireRead(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	if err := httpx.RequirePermission(r, "budget:read"); err != nil {
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
	if err := httpx.RequirePermission(r, "budget:write"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
}

func (s *Server) listBudgets(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var status *string
	if st := strings.TrimSpace(r.URL.Query().Get("status")); st != "" {
		status = &st
	}
	rows, total, err := s.repo.ListBudgets(r.Context(), scope, status, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: rows, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createBudget(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		Code                     string    `json:"code"`
		Name                     string    `json:"name"`
		FinancialOrganizationRef uuid.UUID `json:"financialOrganizationRef"`
		FiscalYearRef            uuid.UUID `json:"fiscalYearRef"`
		Version                  int32     `json:"version"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" || req.FinancialOrganizationRef == uuid.Nil || req.FiscalYearRef == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("code, name, financialOrganizationRef, and fiscalYearRef are required"))
		return
	}
	ver := req.Version
	if ver <= 0 {
		ver = 1
	}
	created, err := s.repo.CreateBudget(r.Context(), scope, budgetdb.CreateBudgetParams{
		OrganizationID: scope[0], Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name),
		FinancialOrganizationRef: req.FinancialOrganizationRef, FiscalYearRef: req.FiscalYearRef,
		Version: ver, Status: "Draft", IsActive: false, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{
		"id": created.ID, "code": strings.TrimSpace(req.Code), "status": created.Status, "rowVersion": created.RowVersion,
	})
}

func (s *Server) getBudget(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	b, lines, err := s.repo.GetBudget(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, b.RowVersion, map[string]any{"budget": b, "lines": lines})
}

func (s *Server) transitionBudget(w http.ResponseWriter, r *http.Request, target string) {
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
	next, err := s.repo.TransitionBudgetAndEmit(r.Context(), scope, id, rowVer, target, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": target, "rowVersion": next})
}

func (s *Server) addBudgetLine(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	budgetID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		AccountRef    uuid.UUID  `json:"accountRef"`
		CostCenterRef *uuid.UUID `json:"costCenterRef,omitempty"`
		ProjectRef    *uuid.UUID `json:"projectRef,omitempty"`
		PeriodNo      int32      `json:"periodNo"`
		BudgetAmount  string     `json:"budgetAmount"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.AccountRef == uuid.Nil || req.PeriodNo < 1 || req.PeriodNo > 12 {
		httpx.WriteError(w, r, httpx.BadRequest("accountRef and periodNo (1..12) are required"))
		return
	}
	amt, err := platform.ParseNumeric(req.BudgetAmount)
	if err != nil || platform.NumericToRat(amt).Sign() < 0 {
		httpx.WriteError(w, r, httpx.BadRequest("budgetAmount must be >= 0"))
		return
	}
	created, err := s.repo.AddBudgetLine(r.Context(), scope, budgetdb.CreateBudgetLineParams{
		OrganizationID: scope[0], BudgetID: budgetID, AccountRef: req.AccountRef,
		CostCenterRef: req.CostCenterRef, ProjectRef: req.ProjectRef, PeriodNo: req.PeriodNo, BudgetAmount: amt, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID, "rowVersion": created.RowVersion})
}

func (s *Server) checkBudget(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	var req struct {
		BudgetID        uuid.UUID  `json:"budgetId"`
		AccountRef      *uuid.UUID `json:"accountRef,omitempty"`
		PeriodNo        *int32     `json:"periodNo,omitempty"`
		TransactionType *string    `json:"transactionType,omitempty"`
		RequestedAmount string     `json:"requestedAmount"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.BudgetID == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("budgetId is required"))
		return
	}
	amt, err := platform.ParseNumeric(req.RequestedAmount)
	if err != nil || platform.NumericToRat(amt).Sign() <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("requestedAmount must be > 0"))
		return
	}
	res, err := s.repo.CheckAvailability(r.Context(), scope, req.BudgetID, req.AccountRef, req.PeriodNo, req.TransactionType, platform.NumericToRat(amt))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if !res.Allowed {
		httpx.WriteJSON(w, http.StatusUnprocessableEntity, res)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (s *Server) listPolicies(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListPolicies(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: rows, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createPolicy(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		AccountRef      *uuid.UUID `json:"accountRef,omitempty"`
		TransactionType *string    `json:"transactionType,omitempty"`
		ControlType     string     `json:"controlType"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.ControlType != "HardStop" && req.ControlType != "SoftWarning" && req.ControlType != "ApprovalOverride" {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_CONTROL_TYPE", "controlType must be HardStop, SoftWarning, or ApprovalOverride"))
		return
	}
	created, err := s.repo.CreatePolicy(r.Context(), scope, budgetdb.CreateBudgetPolicyParams{
		OrganizationID: scope[0], AccountRef: req.AccountRef, TransactionType: req.TransactionType,
		ControlType: req.ControlType, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID, "rowVersion": created.RowVersion})
}

func (s *Server) createCommitment(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		BudgetID          uuid.UUID `json:"budgetId"`
		SourceDocumentRef string    `json:"sourceDocumentRef"`
		CommittedAmount   string    `json:"committedAmount"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if req.BudgetID == uuid.Nil || strings.TrimSpace(req.SourceDocumentRef) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("budgetId and sourceDocumentRef are required"))
		return
	}
	amt, err := platform.ParseNumeric(req.CommittedAmount)
	if err != nil || platform.NumericToRat(amt).Sign() <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("committedAmount must be > 0"))
		return
	}
	created, err := s.repo.CreateCommitmentAndEmit(r.Context(), scope, budgetdb.CreateBudgetCommitmentParams{
		OrganizationID: scope[0], BudgetID: req.BudgetID, SourceDocumentRef: strings.TrimSpace(req.SourceDocumentRef),
		CommittedAmount: amt, CreatedBy: user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{
		"id": created.ID, "status": created.Status, "rowVersion": created.RowVersion,
	})
}

func (s *Server) transitionCommitment(w http.ResponseWriter, r *http.Request, status string) {
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
	next, err := s.repo.TransitionCommitment(r.Context(), scope, id, rowVer, status, user)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": status, "rowVersion": next})
}
