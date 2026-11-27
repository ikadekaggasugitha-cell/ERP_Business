package httpapi

import (
	"encoding/json"
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/consolidation-service/internal/consdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

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
	id, err := httpx.ParsePathUUID(r, "id")
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
	groupID, err := httpx.ParsePathUUID(r, "id")
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
	groupID, err := httpx.ParsePathUUID(r, "id")
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
	memberID, err := httpx.ParsePathUUID(r, "memberId")
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
