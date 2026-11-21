package httpapi

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/consolidation-service/internal/consdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type Store interface {
	CreateGroup(ctx context.Context, scope []uuid.UUID, arg consdb.CreateConsolidationGroupParams) (consdb.CreateConsolidationGroupRow, error)
	ListGroups(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]consdb.ConsConsolidationGroup, int64, error)
	AddMember(ctx context.Context, scope []uuid.UUID, arg consdb.CreateConsolidationMemberParams) (consdb.CreateConsolidationMemberRow, error)
	ListMembers(ctx context.Context, scope []uuid.UUID, groupID uuid.UUID) ([]consdb.ConsConsolidationMember, error)
	CreateRule(ctx context.Context, scope []uuid.UUID, arg consdb.CreateEliminationRuleParams) (consdb.CreateEliminationRuleRow, error)
	ListRules(ctx context.Context, scope []uuid.UUID, groupID *uuid.UUID, limit, offset int32) ([]consdb.ConsEliminationRule, int64, error)
	CreateRun(ctx context.Context, scope []uuid.UUID, arg consdb.CreateConsolidationRunParams) (consdb.CreateConsolidationRunRow, error)
	GetRunDetail(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (consdb.ConsConsolidationRun, []consdb.ConsEliminationEntry, []consdb.ConsConsolidationAdjustment, error)
	ListRuns(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]consdb.ConsConsolidationRun, int64, error)
	PostEliminations(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, rowVer int32, entries []consdb.CreateEliminationEntryParams, actor uuid.UUID) (int32, error)
	PostAdjustments(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, rowVer int32, adjs []consdb.CreateConsolidationAdjustmentParams, actor uuid.UUID) (int32, error)
	FinalizeRunAndEmit(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, rowVer int32, actor, corr uuid.UUID) (int32, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server { return &Server{repo: repo, idem: httpx.NewIdemStore()} }

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /consolidation-groups", s.listGroups)
	mux.HandleFunc("POST /consolidation-groups", s.createGroup)
	mux.HandleFunc("GET /consolidation-groups/{id}/members", s.listMembers)
	mux.HandleFunc("POST /consolidation-groups/{id}/members", s.addMember)
	mux.HandleFunc("GET /elimination-rules", s.listRules)
	mux.HandleFunc("POST /elimination-rules", s.createRule)
	mux.HandleFunc("GET /consolidation-runs", s.listRuns)
	mux.HandleFunc("POST /consolidation-runs", s.createRun)
	mux.HandleFunc("GET /consolidation-runs/{id}", s.getRun)
	mux.HandleFunc("POST /consolidation-runs/{id}/eliminate", s.postEliminations)
	mux.HandleFunc("POST /consolidation-runs/{id}/adjustments", s.postAdjustments)
	mux.HandleFunc("POST /consolidation-runs/{id}/finalize", s.finalizeRun)
}

func requireRead(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	if err := httpx.RequirePermission(r, "consolidation:read"); err != nil {
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
	if err := httpx.RequirePermission(r, "consolidation:write"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
}

func (s *Server) listGroups(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListGroups(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: rows, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createGroup(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		Code                     string     `json:"code"`
		Name                     string     `json:"name"`
		ParentGroupID            *uuid.UUID `json:"parentGroupId,omitempty"`
		PresentationCurrencyCode string     `json:"presentationCurrencyCode"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	curr := strings.ToUpper(strings.TrimSpace(req.PresentationCurrencyCode))
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" || len(curr) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("code, name, and 3-letter presentationCurrencyCode are required"))
		return
	}
	created, err := s.repo.CreateGroup(r.Context(), scope, consdb.CreateConsolidationGroupParams{
		OrganizationID: scope[0], Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name),
		ParentGroupID: req.ParentGroupID, PresentationCurrencyCode: curr, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID, "rowVersion": created.RowVersion})
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	groupID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.repo.ListMembers(r.Context(), scope, groupID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: rows, Page: httpx.NewPageMeta(1, len(rows), len(rows))})
}

func (s *Server) addMember(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	groupID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		CompanyRef          uuid.UUID `json:"companyRef"`
		OwnershipPercentage string    `json:"ownershipPercentage"`
		ConsolidationMethod string    `json:"consolidationMethod"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.CompanyRef == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("companyRef is required"))
		return
	}
	if req.ConsolidationMethod != "Full" && req.ConsolidationMethod != "Proportional" && req.ConsolidationMethod != "Equity" {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_METHOD", "consolidationMethod must be Full, Proportional, or Equity"))
		return
	}
	pctNum, err := platform.ParseNumeric(req.OwnershipPercentage)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("ownershipPercentage must be valid decimal"))
		return
	}
	pctRat := platform.NumericToRat(pctNum)
	if pctRat.Sign() < 0 || pctRat.Cmp(big.NewRat(100, 1)) > 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_OWNERSHIP", "ownershipPercentage must be in [0, 100]"))
		return
	}
	created, err := s.repo.AddMember(r.Context(), scope, consdb.CreateConsolidationMemberParams{
		OrganizationID: scope[0], ConsolidationGroupID: groupID, CompanyRef: req.CompanyRef,
		OwnershipPercentage: pctNum, ConsolidationMethod: req.ConsolidationMethod, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID, "rowVersion": created.RowVersion})
}

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var groupID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("consolidationGroupId")); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			groupID = &id
		}
	}
	rows, total, err := s.repo.ListRules(r.Context(), scope, groupID, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: rows, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		ConsolidationGroupID uuid.UUID       `json:"consolidationGroupId"`
		RuleType             string          `json:"ruleType"`
		Config               json.RawMessage `json:"config"`
		IsActive             *bool           `json:"isActive,omitempty"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.ConsolidationGroupID == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("consolidationGroupId is required"))
		return
	}
	switch req.RuleType {
	case "ICReceivablePayable", "ICSalesPurchase", "UnrealizedProfit", "InvestmentEquity":
	default:
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_RULE_TYPE", "invalid elimination ruleType"))
		return
	}
	cfg := req.Config
	if len(cfg) == 0 {
		cfg = []byte(`{}`)
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	created, err := s.repo.CreateRule(r.Context(), scope, consdb.CreateEliminationRuleParams{
		OrganizationID: scope[0], ConsolidationGroupID: req.ConsolidationGroupID,
		RuleType: req.RuleType, Config: cfg, IsActive: active, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID, "rowVersion": created.RowVersion})
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListRuns(r.Context(), scope, limit, offset)
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
		ConsolidationGroupID uuid.UUID `json:"consolidationGroupId"`
		FiscalPeriodRef      uuid.UUID `json:"fiscalPeriodRef"`
		Version              int32     `json:"version"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if req.ConsolidationGroupID == uuid.Nil || req.FiscalPeriodRef == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("consolidationGroupId and fiscalPeriodRef are required"))
		return
	}
	ver := req.Version
	if ver <= 0 {
		ver = 1
	}
	created, err := s.repo.CreateRun(r.Context(), scope, consdb.CreateConsolidationRunParams{
		OrganizationID: scope[0], ConsolidationGroupID: req.ConsolidationGroupID,
		FiscalPeriodRef: req.FiscalPeriodRef, Version: ver, CreatedBy: user,
	})
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
	run, elims, adjs, err := s.repo.GetRunDetail(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, run.RowVersion, map[string]any{
		"run": run, "eliminations": elims, "adjustments": adjs,
	})
}

func (s *Server) postEliminations(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	runID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
		Entries []struct {
			EliminationRuleID *uuid.UUID `json:"eliminationRuleId,omitempty"`
			AccountRef        uuid.UUID  `json:"accountRef"`
			DebitAmount       string     `json:"debitAmount"`
			CreditAmount      string     `json:"creditAmount"`
			Description       *string    `json:"description,omitempty"`
		} `json:"entries"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if len(req.Entries) < 2 {
		httpx.WriteError(w, r, httpx.Unprocessable("CONSOLIDATION_UNBALANCED", "at least 2 balanced elimination entries are required"))
		return
	}
	totDr, totCr := new(big.Rat), new(big.Rat)
	params := make([]consdb.CreateEliminationEntryParams, 0, len(req.Entries))
	for _, e := range req.Entries {
		dr, err := platform.ParseOptionalNumeric(e.DebitAmount)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid debitAmount"))
			return
		}
		cr, err := platform.ParseOptionalNumeric(e.CreditAmount)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid creditAmount"))
			return
		}
		totDr.Add(totDr, platform.NumericToRat(dr))
		totCr.Add(totCr, platform.NumericToRat(cr))
		params = append(params, consdb.CreateEliminationEntryParams{
			OrganizationID: scope[0], ConsolidationRunID: runID, EliminationRuleID: e.EliminationRuleID,
			AccountRef: e.AccountRef, DebitAmount: dr, CreditAmount: cr, Description: e.Description, CreatedBy: user,
		})
	}
	if totDr.Sign() <= 0 || totDr.Cmp(totCr) != 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("CONSOLIDATION_UNBALANCED", "elimination entries must balance (debit == credit > 0)"))
		return
	}
	next, err := s.repo.PostEliminations(r.Context(), scope, runID, rowVer, params, user)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": runID, "status": "Eliminated", "rowVersion": next})
}

func (s *Server) postAdjustments(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	runID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
		Adjustments []struct {
			AccountRef     uuid.UUID `json:"accountRef"`
			DebitAmount    string    `json:"debitAmount"`
			CreditAmount   string    `json:"creditAmount"`
			AdjustmentType string    `json:"adjustmentType"`
			Notes          *string   `json:"notes,omitempty"`
		} `json:"adjustments"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if len(req.Adjustments) < 2 {
		httpx.WriteError(w, r, httpx.Unprocessable("CONSOLIDATION_UNBALANCED", "at least 2 balanced adjustment entries are required"))
		return
	}
	totDr, totCr := new(big.Rat), new(big.Rat)
	params := make([]consdb.CreateConsolidationAdjustmentParams, 0, len(req.Adjustments))
	for _, a := range req.Adjustments {
		if a.AdjustmentType != "TopSide" && a.AdjustmentType != "Reclass" && a.AdjustmentType != "Policy" {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_ADJUSTMENT_TYPE", "adjustmentType must be TopSide, Reclass, or Policy"))
			return
		}
		dr, err := platform.ParseOptionalNumeric(a.DebitAmount)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid debitAmount"))
			return
		}
		cr, err := platform.ParseOptionalNumeric(a.CreditAmount)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid creditAmount"))
			return
		}
		totDr.Add(totDr, platform.NumericToRat(dr))
		totCr.Add(totCr, platform.NumericToRat(cr))
		params = append(params, consdb.CreateConsolidationAdjustmentParams{
			OrganizationID: scope[0], ConsolidationRunID: runID, AccountRef: a.AccountRef,
			DebitAmount: dr, CreditAmount: cr, AdjustmentType: a.AdjustmentType, Notes: a.Notes, CreatedBy: user,
		})
	}
	if totDr.Sign() <= 0 || totDr.Cmp(totCr) != 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("CONSOLIDATION_UNBALANCED", "adjustments must balance (debit == credit > 0)"))
		return
	}
	next, err := s.repo.PostAdjustments(r.Context(), scope, runID, rowVer, params, user)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": runID, "status": "Adjusted", "rowVersion": next})
}

func (s *Server) finalizeRun(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	runID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.repo.FinalizeRunAndEmit(r.Context(), scope, runID, rowVer, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": runID, "status": "Finalized", "rowVersion": next})
}
