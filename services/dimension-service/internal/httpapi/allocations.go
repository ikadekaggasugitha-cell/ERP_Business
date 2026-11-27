package httpapi

import (
	"encoding/json"
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/dimension-service/internal/dimensiondb"
	"github.com/sinergiinvestasiproperti/erp/services/dimension-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// validateTargets enforces BR: at least one target, no self-allocation, no
// duplicates, each percentage in (0,100], and the total equals exactly 100.
// Percentages are summed with exact rational arithmetic (never floats).
func validateTargets(source uuid.UUID, targets []allocTarget) error {
	if len(targets) == 0 {
		return httpx.BadRequest("at least one allocation target is required")
	}
	total := new(big.Rat)
	hundred := big.NewRat(100, 1)
	seen := map[uuid.UUID]bool{}
	for _, t := range targets {
		if t.CostCenterID == source {
			return httpx.Unprocessable("ALLOCATION_SELF_TARGET", "allocation target must differ from source cost center")
		}
		if seen[t.CostCenterID] {
			return httpx.Unprocessable("ALLOCATION_DUPLICATE_TARGET", "duplicate allocation target")
		}
		seen[t.CostCenterID] = true
		p, ok := new(big.Rat).SetString(strings.TrimSpace(t.Percentage))
		if !ok || p.Sign() <= 0 || p.Cmp(hundred) > 0 {
			return httpx.Unprocessable("INVALID_PERCENTAGE", "percentage must be a number in (0, 100]")
		}
		total.Add(total, p)
	}
	if total.Cmp(hundred) != 0 {
		return httpx.Unprocessable("ALLOCATION_TOTAL_NOT_100", "allocation percentages must total exactly 100")
	}
	return nil
}

func (s *Server) listAllocationRules(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListAllocationRules(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	data := make([]allocResp, 0, len(rows))
	for _, row := range rows {
		data = append(data, allocDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: data, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createAllocationRule(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req allocReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.AllocationBasis) == "" || req.SourceCostCenterID == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("name, allocationBasis and sourceCostCenterId are required"))
		return
	}
	if err := validateTargets(req.SourceCostCenterID, req.Targets); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	raw, _ := json.Marshal(req.Targets)
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	created, err := s.repo.CreateAllocationRule(r.Context(), scope, dimensiondb.CreateAllocationRuleParams{
		OrganizationID: scope[0], SourceCostCenterID: req.SourceCostCenterID, Name: strings.TrimSpace(req.Name),
		AllocationBasis: strings.TrimSpace(req.AllocationBasis), Targets: raw, IsActive: active, CreatedBy: user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, allocResp{
		ID: created.ID, SourceCostCenterID: req.SourceCostCenterID, Name: strings.TrimSpace(req.Name),
		AllocationBasis: strings.TrimSpace(req.AllocationBasis), Targets: raw, IsActive: active, RowVersion: created.RowVersion,
	})
}

func (s *Server) getAllocationRule(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "allocation rule")
	if !ok {
		return
	}
	row, err := s.repo.GetAllocationRule(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, allocDTO(row))
}

func (s *Server) updateAllocationRule(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "allocation rule")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req allocReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.AllocationBasis) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name and allocationBasis are required"))
		return
	}
	existing, err := s.repo.GetAllocationRule(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if !guardSystemOrgMutation(w, r, existing.OrganizationID) {
		return
	}
	if err := validateTargets(existing.SourceCostCenterID, req.Targets); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	raw, _ := json.Marshal(req.Targets)
	active := existing.IsActive
	if req.IsActive != nil {
		active = *req.IsActive
	}
	next, err := s.repo.UpdateAllocationRule(r.Context(), scope, dimensiondb.UpdateAllocationRuleParams{
		ID: id, RowVersion: rowVer, Name: strings.TrimSpace(req.Name),
		AllocationBasis: strings.TrimSpace(req.AllocationBasis), Targets: raw, IsActive: active, UpdatedBy: &user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, allocResp{
		ID: id, SourceCostCenterID: existing.SourceCostCenterID, Name: strings.TrimSpace(req.Name),
		AllocationBasis: strings.TrimSpace(req.AllocationBasis), Targets: raw, IsActive: active, RowVersion: next,
	})
}

func (s *Server) deleteAllocationRule(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "allocation rule")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if existing, err := s.repo.GetAllocationRule(r.Context(), scope, id); err == nil {
		if !guardSystemOrgMutation(w, r, existing.OrganizationID) {
			return
		}
	}
	if err := s.repo.DeleteAllocationRule(r.Context(), scope, dimensiondb.DeleteAllocationRuleParams{
		ID: id, RowVersion: rowVer, UpdatedBy: &user,
	}); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) executeAllocationRule(w http.ResponseWriter, r *http.Request) {
	scope, _, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "allocation rule")
	if !ok {
		return
	}
	var req struct {
		Amount         string     `json:"amount"`
		FiscalPeriodID *uuid.UUID `json:"fiscalPeriodId,omitempty"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	amtNum, err := platform.ParseNumeric(req.Amount)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("amount must be a valid decimal string"))
		return
	}
	amtRat := platform.NumericToRat(amtNum)
	if amtRat == nil || amtRat.Sign() <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("amount must be > 0"))
		return
	}
	formattedAmt := platform.FormatNumeric(amtNum, 4)
	rule, err := s.repo.ExecuteAllocationRule(r.Context(), scope, id, formattedAmt, req.FiscalPeriodID, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	distributions := repository.ComputeDistributions(rule.Targets, formattedAmt)
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, rule.RowVersion, map[string]any{
		"ruleId":             rule.ID,
		"sourceCostCenterId": rule.SourceCostCenterID,
		"totalAmount":        formattedAmt,
		"distributions":      distributions,
	})
}

func (s *Server) listCostCenterAssignments(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var ccID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("costCenterId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid costCenterId"))
			return
		}
		ccID = &parsed
	}
	rows, total, err := s.repo.ListCostCenterAssignments(r.Context(), scope, ccID, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	data := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		data = append(data, map[string]any{
			"id": row.ID, "costCenterId": row.CostCenterID, "organizationRef": row.OrganizationRef, "rowVersion": row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: data, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createCostCenterAssignment(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		CostCenterID    uuid.UUID `json:"costCenterId"`
		OrganizationRef uuid.UUID `json:"organizationRef"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if req.CostCenterID == uuid.Nil || req.OrganizationRef == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("costCenterId and organizationRef are required"))
		return
	}
	created, err := s.repo.CreateCostCenterAssignment(r.Context(), scope, dimensiondb.CreateCostCenterAssignmentParams{
		OrganizationID: scope[0], CostCenterID: req.CostCenterID, OrganizationRef: req.OrganizationRef, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{
		"id": created.ID, "costCenterId": req.CostCenterID, "organizationRef": req.OrganizationRef, "rowVersion": created.RowVersion,
	})
}

func (s *Server) deleteCostCenterAssignment(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "cost center assignment")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.repo.DeleteCostCenterAssignment(r.Context(), scope, dimensiondb.DeleteCostCenterAssignmentParams{
		ID: id, RowVersion: rowVer, UpdatedBy: &user,
	}); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listProfitCenterAssignments(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var pcID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("profitCenterId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid profitCenterId"))
			return
		}
		pcID = &parsed
	}
	rows, total, err := s.repo.ListProfitCenterAssignments(r.Context(), scope, pcID, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	data := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		data = append(data, map[string]any{
			"id": row.ID, "profitCenterId": row.ProfitCenterID, "organizationRef": row.OrganizationRef, "rowVersion": row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: data, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createProfitCenterAssignment(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		ProfitCenterID  uuid.UUID `json:"profitCenterId"`
		OrganizationRef uuid.UUID `json:"organizationRef"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if req.ProfitCenterID == uuid.Nil || req.OrganizationRef == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("profitCenterId and organizationRef are required"))
		return
	}
	created, err := s.repo.CreateProfitCenterAssignment(r.Context(), scope, dimensiondb.CreateProfitCenterAssignmentParams{
		OrganizationID: scope[0], ProfitCenterID: req.ProfitCenterID, OrganizationRef: req.OrganizationRef, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{
		"id": created.ID, "profitCenterId": req.ProfitCenterID, "organizationRef": req.OrganizationRef, "rowVersion": created.RowVersion,
	})
}

func (s *Server) deleteProfitCenterAssignment(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "profit center assignment")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.repo.DeleteProfitCenterAssignment(r.Context(), scope, dimensiondb.DeleteProfitCenterAssignmentParams{
		ID: id, RowVersion: rowVer, UpdatedBy: &user,
	}); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
