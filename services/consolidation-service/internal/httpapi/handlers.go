package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/consolidation-service/internal/consdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type Store interface {
	CreateGroup(ctx context.Context, scope []uuid.UUID, arg consdb.CreateConsolidationGroupParams) (consdb.CreateConsolidationGroupRow, error)
	GetGroup(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (consdb.ConsConsolidationGroup, error)
	ListGroups(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]consdb.ConsConsolidationGroup, int64, error)
	UpdateGroup(ctx context.Context, scope []uuid.UUID, arg consdb.UpdateConsolidationGroupParams) (int32, error)
	DeleteGroup(ctx context.Context, scope []uuid.UUID, arg consdb.DeleteConsolidationGroupParams) error
	AddMember(ctx context.Context, scope []uuid.UUID, arg consdb.CreateConsolidationMemberParams) (consdb.CreateConsolidationMemberRow, error)
	ListMembers(ctx context.Context, scope []uuid.UUID, groupID uuid.UUID) ([]consdb.ConsConsolidationMember, error)
	DeleteMember(ctx context.Context, scope []uuid.UUID, arg consdb.DeleteConsolidationMemberParams) error
	CreateRule(ctx context.Context, scope []uuid.UUID, arg consdb.CreateEliminationRuleParams) (consdb.CreateEliminationRuleRow, error)
	ListRules(ctx context.Context, scope []uuid.UUID, groupID *uuid.UUID, limit, offset int32) ([]consdb.ConsEliminationRule, int64, error)
	CreateRun(ctx context.Context, scope []uuid.UUID, arg consdb.CreateConsolidationRunParams) (consdb.CreateConsolidationRunRow, error)
	GetRunDetail(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (consdb.ConsConsolidationRun, []consdb.ConsEliminationEntry, []consdb.ConsConsolidationAdjustment, error)
	ListRuns(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]consdb.ConsConsolidationRun, int64, error)
	PostEliminations(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, rowVer int32, entries []consdb.CreateEliminationEntryParams, actor uuid.UUID) (int32, error)
	PostAdjustments(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, rowVer int32, adjs []consdb.CreateConsolidationAdjustmentParams, actor uuid.UUID) (int32, error)
	FinalizeRunAndEmit(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, rowVer int32, actor, corr uuid.UUID) (int32, error)
}

type generatedConsSummary struct {
	ICElimTotal string
	NCIEquity   string
	NCINetInc   string
	CTAReserve  string
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
	mu   sync.RWMutex
	gen  map[uuid.UUID]generatedConsSummary
}

func NewServer(repo Store) *Server {
	return &Server{
		repo: repo,
		idem: httpx.NewIdemStore(),
		gen:  make(map[uuid.UUID]generatedConsSummary),
	}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /consolidation-groups", s.listGroups)
	mux.HandleFunc("POST /consolidation-groups", s.createGroup)
	mux.HandleFunc("GET /consolidation-groups/{id}", s.getGroup)
	mux.HandleFunc("PUT /consolidation-groups/{id}", s.updateGroup)
	mux.HandleFunc("DELETE /consolidation-groups/{id}", s.deleteGroup)
	mux.HandleFunc("GET /consolidation-groups/{id}/members", s.listMembers)
	mux.HandleFunc("POST /consolidation-groups/{id}/members", s.addMember)
	mux.HandleFunc("DELETE /consolidation-groups/{id}/members/{memberId}", s.deleteMember)
	mux.HandleFunc("GET /elimination-rules", s.listRules)
	mux.HandleFunc("POST /elimination-rules", s.createRule)
	mux.HandleFunc("GET /consolidation-runs", s.listRuns)
	mux.HandleFunc("POST /consolidation-runs", s.createRun)
	mux.HandleFunc("GET /consolidation-runs/{id}", s.getRun)
	mux.HandleFunc("GET /consolidation-runs/{id}/statements", s.consolidatedStatements)
	mux.HandleFunc("POST /consolidation-runs/{id}/generate-eliminations", s.generateEliminations)
	mux.HandleFunc("POST /consolidation-runs/{id}/eliminate", s.postEliminations)
	mux.HandleFunc("POST /consolidation-runs/{id}/adjustments", s.postAdjustments)
	mux.HandleFunc("POST /consolidation-runs/{id}/finalize", s.finalizeRun)
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

func rawJSONOrEmptyObj(b []byte) json.RawMessage {
	if len(bytes.TrimSpace(b)) == 0 {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(b)
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
	if err := httpx.RequireMaxAdminTier(r, httpx.AdminTierHolding); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
}

func groupDTO(g consdb.ConsConsolidationGroup) map[string]any {
	return map[string]any{
		"id":                       g.ID,
		"organizationId":           g.OrganizationID,
		"code":                     g.Code,
		"name":                     g.Name,
		"parentGroupId":            g.ParentGroupID,
		"presentationCurrencyCode": g.PresentationCurrencyCode,
		"isSystem":                 g.OrganizationID == platform.SystemOrgID,
		"rowVersion":               g.RowVersion,
	}
}

func memberDTO(m consdb.ConsConsolidationMember) map[string]any {
	return map[string]any{
		"id":                   m.ID,
		"organizationId":       m.OrganizationID,
		"consolidationGroupId": m.ConsolidationGroupID,
		"companyRef":           m.CompanyRef,
		"ownershipPercentage":  platform.FormatNumeric(m.OwnershipPercentage, 4),
		"consolidationMethod":  m.ConsolidationMethod,
		"rowVersion":           m.RowVersion,
	}
}

func ruleDTO(r consdb.ConsEliminationRule) map[string]any {
	return map[string]any{
		"id":                   r.ID,
		"organizationId":       r.OrganizationID,
		"consolidationGroupId": r.ConsolidationGroupID,
		"ruleType":             r.RuleType,
		"config":               rawJSONOrEmptyObj(r.Config),
		"isActive":             r.IsActive,
		"isSystem":             r.OrganizationID == platform.SystemOrgID,
		"rowVersion":           r.RowVersion,
	}
}

func runDTO(r consdb.ConsConsolidationRun) map[string]any {
	return map[string]any{
		"id":                   r.ID,
		"organizationId":       r.OrganizationID,
		"consolidationGroupId": r.ConsolidationGroupID,
		"fiscalPeriodRef":      r.FiscalPeriodRef,
		"status":               r.Status,
		"version":              r.Version,
		"rowVersion":           r.RowVersion,
	}
}

func eliminationEntryDTO(e consdb.ConsEliminationEntry) map[string]any {
	return map[string]any{
		"id":                 e.ID,
		"organizationId":     e.OrganizationID,
		"consolidationRunId": e.ConsolidationRunID,
		"eliminationRuleId":  e.EliminationRuleID,
		"accountRef":         e.AccountRef,
		"debitAmount":        platform.FormatNumeric(e.DebitAmount, 4),
		"creditAmount":       platform.FormatNumeric(e.CreditAmount, 4),
		"description":        e.Description,
		"rowVersion":         e.RowVersion,
	}
}

func adjustmentDTO(a consdb.ConsConsolidationAdjustment) map[string]any {
	return map[string]any{
		"id":                 a.ID,
		"organizationId":     a.OrganizationID,
		"consolidationRunId": a.ConsolidationRunID,
		"accountRef":         a.AccountRef,
		"debitAmount":        platform.FormatNumeric(a.DebitAmount, 4),
		"creditAmount":       platform.FormatNumeric(a.CreditAmount, 4),
		"adjustmentType":     a.AdjustmentType,
		"notes":              a.Notes,
		"rowVersion":         a.RowVersion,
	}
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
	out := make([]map[string]any, 0, len(rows))
	for _, g := range rows {
		out = append(out, groupDTO(g))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) getGroup(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	g, err := s.repo.GetGroup(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, g.RowVersion, groupDTO(g))
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

func (s *Server) updateGroup(w http.ResponseWriter, r *http.Request) {
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
	var req struct {
		Name                     string     `json:"name"`
		ParentGroupID            *uuid.UUID `json:"parentGroupId,omitempty"`
		PresentationCurrencyCode string     `json:"presentationCurrencyCode"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	curr := strings.ToUpper(strings.TrimSpace(req.PresentationCurrencyCode))
	if strings.TrimSpace(req.Name) == "" || len(curr) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("name and 3-letter presentationCurrencyCode are required"))
		return
	}
	next, err := s.repo.UpdateGroup(r.Context(), scope, consdb.UpdateConsolidationGroupParams{
		ID: id, RowVersion: rowVer, Name: strings.TrimSpace(req.Name),
		ParentGroupID: req.ParentGroupID, PresentationCurrencyCode: curr, UpdatedBy: &user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "rowVersion": next})
}

func (s *Server) deleteGroup(w http.ResponseWriter, r *http.Request) {
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
	if err := s.repo.DeleteGroup(r.Context(), scope, consdb.DeleteConsolidationGroupParams{
		ID: id, RowVersion: rowVer, UpdatedBy: &user,
	}); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	out := make([]map[string]any, 0, len(rows))
	for _, m := range rows {
		out = append(out, memberDTO(m))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(1, len(out), len(out))})
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
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
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
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID, "rowVersion": created.RowVersion})
}

func (s *Server) deleteMember(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	memberID, err := httpx.ParseUUIDParam("memberId", r.PathValue("memberId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.repo.DeleteMember(r.Context(), scope, consdb.DeleteConsolidationMemberParams{
		ID: memberID, RowVersion: rowVer, UpdatedBy: &user,
	}); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	out := make([]map[string]any, 0, len(rows))
	for _, rl := range rows {
		out = append(out, ruleDTO(rl))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
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
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
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
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID, "rowVersion": created.RowVersion})
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
	elimDTOs := make([]map[string]any, 0, len(elims))
	for _, e := range elims {
		elimDTOs = append(elimDTOs, eliminationEntryDTO(e))
	}
	adjDTOs := make([]map[string]any, 0, len(adjs))
	for _, a := range adjs {
		adjDTOs = append(adjDTOs, adjustmentDTO(a))
	}
	httpx.WriteIDResponse(w, http.StatusOK, run.RowVersion, map[string]any{
		"run": runDTO(run), "eliminations": elimDTOs, "adjustments": adjDTOs,
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
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
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
		drRat, crRat := platform.NumericToRat(dr), platform.NumericToRat(cr)
		if drRat.Sign() < 0 || crRat.Sign() < 0 || (drRat.Sign() > 0 && crRat.Sign() > 0) || (drRat.Sign() == 0 && crRat.Sign() == 0) {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_ENTRY_LINE", "each elimination entry must have strictly one positive debit or credit amount"))
			return
		}
		totDr.Add(totDr, drRat)
		totCr.Add(totCr, crRat)
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
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, next, map[string]any{"id": runID, "status": "Eliminated", "rowVersion": next})
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
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
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
		drRat, crRat := platform.NumericToRat(dr), platform.NumericToRat(cr)
		if drRat.Sign() < 0 || crRat.Sign() < 0 || (drRat.Sign() > 0 && crRat.Sign() > 0) || (drRat.Sign() == 0 && crRat.Sign() == 0) {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_ENTRY_LINE", "each adjustment entry must have strictly one positive debit or credit amount"))
			return
		}
		totDr.Add(totDr, drRat)
		totCr.Add(totCr, crRat)
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
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, next, map[string]any{"id": runID, "status": "Adjusted", "rowVersion": next})
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
	idemKey, reqHash, release, done := beginIdemOptional(w, r, s.idem, scope, nil)
	if done {
		return
	}
	defer release()
	next, err := s.repo.FinalizeRunAndEmit(r.Context(), scope, runID, rowVer, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, next, map[string]any{"id": runID, "status": "Finalized", "rowVersion": next})
}

func (s *Server) generateEliminations(w http.ResponseWriter, r *http.Request) {
	scope, _, ok := requireWrite(w, r)
	if !ok {
		return
	}
	if err := httpx.RequireMaxAdminTier(r, httpx.AdminTierHolding); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	runID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		IntercompanyBalances []struct {
			FromEntityID string `json:"fromEntityId"`
			ToEntityID   string `json:"toEntityId"`
			AccountCode  string `json:"accountCode"`
			Amount       string `json:"amount"`
		} `json:"intercompanyBalances"`
		SubsidiaryEquity []struct {
			EntityID         string `json:"entityId"`
			EquityAmount     string `json:"equityAmount"`
			NetIncome        string `json:"netIncome"`
			OwnershipPercent string `json:"ownershipPercent"`
		} `json:"subsidiaryEquity"`
		FXTranslations []struct {
			EntityID           string `json:"entityId"`
			FunctionalCurrency string `json:"functionalCurrency"`
			ReportingCurrency  string `json:"reportingCurrency"`
			ForeignNetAssets   string `json:"foreignNetAssets"`
			HistoricalRate     string `json:"historicalRate"`
			ClosingRate        string `json:"closingRate"`
		} `json:"fxTranslations"`
	}
	idemKey, reqHash, release, done := beginIdemOptional(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if _, _, _, err := s.repo.GetRunDetail(r.Context(), scope, runID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	hundred := big.NewRat(100, 1)
	icTotal := new(big.Rat)
	nciEquityTotal := new(big.Rat)
	nciNetIncTotal := new(big.Rat)
	ctaTotal := new(big.Rat)
	entries := make([]map[string]any, 0)

	for _, ic := range req.IntercompanyBalances {
		num, perr := platform.ParseNumeric(ic.Amount)
		if perr != nil {
			httpx.WriteError(w, r, httpx.BadRequest("intercompanyBalances[].amount is invalid"))
			return
		}
		amt := platform.NumericToRat(num)
		if amt == nil || amt.Sign() <= 0 {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_AMOUNT", "intercompany amount must be > 0"))
			return
		}
		icTotal.Add(icTotal, amt)
		entries = append(entries, map[string]any{
			"type":         "IC_ELIMINATION",
			"fromEntityId": ic.FromEntityID,
			"toEntityId":   ic.ToEntityID,
			"accountCode":  ic.AccountCode,
			"debit":        amt.FloatString(4),
			"credit":       amt.FloatString(4),
		})
	}

	for _, sub := range req.SubsidiaryEquity {
		ownNum, perr := platform.ParseNumeric(sub.OwnershipPercent)
		if perr != nil {
			httpx.WriteError(w, r, httpx.BadRequest("ownershipPercent is invalid"))
			return
		}
		ownPct := platform.NumericToRat(ownNum)
		if ownPct == nil || ownPct.Sign() < 0 || ownPct.Cmp(hundred) > 0 {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_OWNERSHIP", "ownershipPercent must be between 0 and 100"))
			return
		}
		nciPct := new(big.Rat).Sub(hundred, ownPct)
		nciRatio := new(big.Rat).Quo(nciPct, hundred)

		eqRat := new(big.Rat)
		if strings.TrimSpace(sub.EquityAmount) != "" {
			if n, e := platform.ParseNumeric(sub.EquityAmount); e == nil {
				eqRat = platform.NumericToRat(n)
			}
		}
		niRat := new(big.Rat)
		if strings.TrimSpace(sub.NetIncome) != "" {
			if n, e := platform.ParseNumeric(sub.NetIncome); e == nil {
				niRat = platform.NumericToRat(n)
			}
		}
		nciEq := new(big.Rat).Mul(eqRat, nciRatio)
		nciNI := new(big.Rat).Mul(niRat, nciRatio)
		nciEquityTotal.Add(nciEquityTotal, nciEq)
		nciNetIncTotal.Add(nciNetIncTotal, nciNI)
		entries = append(entries, map[string]any{
			"type":             "NCI_ALLOCATION",
			"entityId":         sub.EntityID,
			"nciPercent":       nciPct.FloatString(4),
			"nciEquity":        nciEq.FloatString(4),
			"nciNetIncome":     nciNI.FloatString(4),
			"ownershipPercent": ownPct.FloatString(4),
		})
	}

	for _, fx := range req.FXTranslations {
		fnaNum, err1 := platform.ParseNumeric(fx.ForeignNetAssets)
		hrNum, err2 := platform.ParseNumeric(fx.HistoricalRate)
		crNum, err3 := platform.ParseNumeric(fx.ClosingRate)
		if err1 != nil || err2 != nil || err3 != nil {
			httpx.WriteError(w, r, httpx.BadRequest("fxTranslations numeric fields are invalid"))
			return
		}
		diff := new(big.Rat).Sub(platform.NumericToRat(crNum), platform.NumericToRat(hrNum))
		cta := new(big.Rat).Mul(platform.NumericToRat(fnaNum), diff)
		ctaTotal.Add(ctaTotal, cta)
		entries = append(entries, map[string]any{
			"type":               "CTA_TRANSLATION",
			"entityId":           fx.EntityID,
			"functionalCurrency": fx.FunctionalCurrency,
			"reportingCurrency":  fx.ReportingCurrency,
			"ctaAmount":          cta.FloatString(4),
		})
	}

	nciCombined := new(big.Rat).Add(nciEquityTotal, nciNetIncTotal)
	s.mu.Lock()
	s.gen[runID] = generatedConsSummary{
		ICElimTotal: icTotal.FloatString(4),
		NCIEquity:   nciEquityTotal.FloatString(4),
		NCINetInc:   nciNetIncTotal.FloatString(4),
		CTAReserve:  ctaTotal.FloatString(4),
	}
	s.mu.Unlock()

	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, 1, map[string]any{
		"runId":              runID,
		"icEliminationTotal": icTotal.FloatString(4),
		"nciEquityTotal":     nciEquityTotal.FloatString(4),
		"nciNetIncomeTotal":  nciNetIncTotal.FloatString(4),
		"nciTotal":           nciCombined.FloatString(4),
		"ctaTotal":           ctaTotal.FloatString(4),
		"balanced":           true,
		"entries":            entries,
	})
}

func (s *Server) consolidatedStatements(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	runID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	run, elims, adjs, err := s.repo.GetRunDetail(r.Context(), scope, runID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	totDr := new(big.Rat)
	totCr := new(big.Rat)
	for _, e := range elims {
		if d := platform.NumericToRat(e.DebitAmount); d != nil {
			totDr.Add(totDr, d)
		}
		if c := platform.NumericToRat(e.CreditAmount); c != nil {
			totCr.Add(totCr, c)
		}
	}
	for _, a := range adjs {
		if d := platform.NumericToRat(a.DebitAmount); d != nil {
			totDr.Add(totDr, d)
		}
		if c := platform.NumericToRat(a.CreditAmount); c != nil {
			totCr.Add(totCr, c)
		}
	}

	s.mu.RLock()
	gen := s.gen[runID]
	s.mu.RUnlock()
	if gen.NCIEquity == "" {
		gen.NCIEquity = "0.0000"
	}
	if gen.NCINetInc == "" {
		gen.NCINetInc = "0.0000"
	}
	if gen.CTAReserve == "" {
		gen.CTAReserve = "0.0000"
	}
	if gen.ICElimTotal == "" {
		gen.ICElimTotal = "0.0000"
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"runId":            run.ID,
		"status":           run.Status,
		"currency":         "IDR",
		"eliminationCount": len(elims),
		"adjustmentCount":  len(adjs),
		"balanceSheet": map[string]any{
			"totalEliminationsDebit":  totDr.FloatString(4),
			"totalEliminationsCredit": totCr.FloatString(4),
			"icEliminationTotal":      gen.ICElimTotal,
			"nciEquity":               gen.NCIEquity,
			"ctaReserve":              gen.CTAReserve,
			"isBalanced":              totDr.Cmp(totCr) == 0,
		},
		"incomeStatement": map[string]any{
			"nciNetIncome": gen.NCINetInc,
			"status":       run.Status,
		},
	})
}
