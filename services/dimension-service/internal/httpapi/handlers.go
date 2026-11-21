// Package httpapi implements Cost Center, Profit Center and Allocation Rule
// endpoints with RLS scope, optimistic locking (If-Match) and idempotency.
package httpapi

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/dimension-service/internal/dimensiondb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Store abstracts repository operations for handler unit testing.
type Store interface {
	CreateCostCenter(ctx context.Context, scope []uuid.UUID, arg dimensiondb.CreateCostCenterParams, corr uuid.UUID) (dimensiondb.CreateCostCenterRow, error)
	GetCostCenter(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (dimensiondb.CcCostCenter, error)
	ListCostCenters(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]dimensiondb.CcCostCenter, int64, error)
	UpdateCostCenter(ctx context.Context, scope []uuid.UUID, arg dimensiondb.UpdateCostCenterParams, corr uuid.UUID) (int32, error)
	DeleteCostCenter(ctx context.Context, scope []uuid.UUID, arg dimensiondb.DeleteCostCenterParams) error

	CreateProfitCenter(ctx context.Context, scope []uuid.UUID, arg dimensiondb.CreateProfitCenterParams, corr uuid.UUID) (dimensiondb.CreateProfitCenterRow, error)
	GetProfitCenter(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (dimensiondb.PcProfitCenter, error)
	ListProfitCenters(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]dimensiondb.PcProfitCenter, int64, error)
	UpdateProfitCenter(ctx context.Context, scope []uuid.UUID, arg dimensiondb.UpdateProfitCenterParams, corr uuid.UUID) (int32, error)
	DeleteProfitCenter(ctx context.Context, scope []uuid.UUID, arg dimensiondb.DeleteProfitCenterParams) error

	CreateAllocationRule(ctx context.Context, scope []uuid.UUID, arg dimensiondb.CreateAllocationRuleParams, corr uuid.UUID) (dimensiondb.CreateAllocationRuleRow, error)
	GetAllocationRule(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (dimensiondb.CcCostAllocationRule, error)
	ListAllocationRules(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]dimensiondb.CcCostAllocationRule, int64, error)
	UpdateAllocationRule(ctx context.Context, scope []uuid.UUID, arg dimensiondb.UpdateAllocationRuleParams, corr uuid.UUID) (int32, error)
	DeleteAllocationRule(ctx context.Context, scope []uuid.UUID, arg dimensiondb.DeleteAllocationRuleParams) error
	ExecuteAllocationRule(ctx context.Context, scope []uuid.UUID, ruleID uuid.UUID, amount string, fiscalPeriodID *uuid.UUID, corr uuid.UUID) (dimensiondb.CcCostAllocationRule, error)

	CreateCostCenterAssignment(ctx context.Context, scope []uuid.UUID, arg dimensiondb.CreateCostCenterAssignmentParams) (dimensiondb.CreateCostCenterAssignmentRow, error)
	ListCostCenterAssignments(ctx context.Context, scope []uuid.UUID, costCenterID *uuid.UUID, limit, offset int32) ([]dimensiondb.CcCostCenterAssignment, int64, error)
	DeleteCostCenterAssignment(ctx context.Context, scope []uuid.UUID, arg dimensiondb.DeleteCostCenterAssignmentParams) error

	CreateProfitCenterAssignment(ctx context.Context, scope []uuid.UUID, arg dimensiondb.CreateProfitCenterAssignmentParams) (dimensiondb.CreateProfitCenterAssignmentRow, error)
	ListProfitCenterAssignments(ctx context.Context, scope []uuid.UUID, profitCenterID *uuid.UUID, limit, offset int32) ([]dimensiondb.PcProfitCenterAssignment, int64, error)
	DeleteProfitCenterAssignment(ctx context.Context, scope []uuid.UUID, arg dimensiondb.DeleteProfitCenterAssignmentParams) error
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /cost-centers", s.listCostCenters)
	mux.HandleFunc("POST /cost-centers", s.createCostCenter)
	mux.HandleFunc("GET /cost-centers/{id}", s.getCostCenter)
	mux.HandleFunc("PUT /cost-centers/{id}", s.updateCostCenter)
	mux.HandleFunc("DELETE /cost-centers/{id}", s.deleteCostCenter)

	mux.HandleFunc("GET /profit-centers", s.listProfitCenters)
	mux.HandleFunc("POST /profit-centers", s.createProfitCenter)
	mux.HandleFunc("GET /profit-centers/{id}", s.getProfitCenter)
	mux.HandleFunc("PUT /profit-centers/{id}", s.updateProfitCenter)
	mux.HandleFunc("DELETE /profit-centers/{id}", s.deleteProfitCenter)

	mux.HandleFunc("GET /allocation-rules", s.listAllocationRules)
	mux.HandleFunc("POST /allocation-rules", s.createAllocationRule)
	mux.HandleFunc("GET /allocation-rules/{id}", s.getAllocationRule)
	mux.HandleFunc("PUT /allocation-rules/{id}", s.updateAllocationRule)
	mux.HandleFunc("DELETE /allocation-rules/{id}", s.deleteAllocationRule)
	mux.HandleFunc("POST /allocation-rules/{id}/execute", s.executeAllocationRule)

	mux.HandleFunc("GET /cost-center-assignments", s.listCostCenterAssignments)
	mux.HandleFunc("POST /cost-center-assignments", s.createCostCenterAssignment)
	mux.HandleFunc("DELETE /cost-center-assignments/{id}", s.deleteCostCenterAssignment)

	mux.HandleFunc("GET /profit-center-assignments", s.listProfitCenterAssignments)
	mux.HandleFunc("POST /profit-center-assignments", s.createProfitCenterAssignment)
	mux.HandleFunc("DELETE /profit-center-assignments/{id}", s.deleteProfitCenterAssignment)
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
	if err := httpx.RequirePermission(r, "dimension:write"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
}

func requireRead(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	if err := httpx.RequirePermission(r, "dimension:read"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	return scope, true
}

func pathID(w http.ResponseWriter, r *http.Request, what string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid "+what+" id"))
		return uuid.Nil, false
	}
	return id, true
}

func statusFilter(r *http.Request) *string {
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
		return &v
	}
	return nil
}

var validStatus = map[string]bool{"Draft": true, "Active": true, "Inactive": true, "Closed": true}

// validateCenter checks the fields shared by cost and profit centers.
func validateCenter(name, status, from, to string) (string, pgtype.Date, pgtype.Date, error) {
	if strings.TrimSpace(name) == "" {
		return "", pgtype.Date{}, pgtype.Date{}, httpx.BadRequest("name is required")
	}
	if status == "" {
		status = "Draft"
	}
	if !validStatus[status] {
		return "", pgtype.Date{}, pgtype.Date{}, httpx.Unprocessable("INVALID_STATUS", "status must be Draft, Active, Inactive or Closed")
	}
	f, err := platform.ParseDate(from)
	if err != nil {
		return "", pgtype.Date{}, pgtype.Date{}, httpx.BadRequest("effectiveFrom must be YYYY-MM-DD")
	}
	var t pgtype.Date
	if to != "" {
		if t, err = platform.ParseDate(to); err != nil {
			return "", pgtype.Date{}, pgtype.Date{}, httpx.BadRequest("effectiveTo must be YYYY-MM-DD")
		}
		if t.Time.Before(f.Time) {
			return "", pgtype.Date{}, pgtype.Date{}, httpx.Unprocessable("INVALID_EFFECTIVE_RANGE", "effectiveTo must not precede effectiveFrom")
		}
	}
	return status, f, t, nil
}

// -----------------------------------------------------------------------------
// Cost Centers
// -----------------------------------------------------------------------------

type centerReq struct {
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	ParentID      *uuid.UUID `json:"parentId,omitempty"`
	OwnerRef      *uuid.UUID `json:"ownerRef,omitempty"`
	Status        string     `json:"status,omitempty"`
	EffectiveFrom string     `json:"effectiveFrom"`
	EffectiveTo   string     `json:"effectiveTo,omitempty"`
}

type centerResp struct {
	ID            uuid.UUID  `json:"id"`
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	ParentID      *uuid.UUID `json:"parentId,omitempty"`
	OwnerRef      *uuid.UUID `json:"ownerRef,omitempty"`
	Status        string     `json:"status"`
	EffectiveFrom string     `json:"effectiveFrom"`
	EffectiveTo   string     `json:"effectiveTo,omitempty"`
	RowVersion    int32      `json:"rowVersion"`
}

func costCenterDTO(c dimensiondb.CcCostCenter) centerResp {
	return centerResp{ID: c.ID, Code: c.Code, Name: c.Name, Description: c.Description, ParentID: c.ParentID,
		OwnerRef: c.OwnerRef, Status: c.Status, EffectiveFrom: platform.FormatDate(c.EffectiveFrom),
		EffectiveTo: platform.FormatDate(c.EffectiveTo), RowVersion: c.RowVersion}
}

func profitCenterDTO(c dimensiondb.PcProfitCenter) centerResp {
	return centerResp{ID: c.ID, Code: c.Code, Name: c.Name, ParentID: c.ParentID, Status: c.Status,
		EffectiveFrom: platform.FormatDate(c.EffectiveFrom), EffectiveTo: platform.FormatDate(c.EffectiveTo),
		RowVersion: c.RowVersion}
}

func (s *Server) listCostCenters(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListCostCenters(r.Context(), scope, statusFilter(r), limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	data := make([]centerResp, 0, len(rows))
	for _, row := range rows {
		data = append(data, costCenterDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: data, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createCostCenter(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req centerReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code is required"))
		return
	}
	status, from, to, err := validateCenter(req.Name, req.Status, req.EffectiveFrom, req.EffectiveTo)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	created, err := s.repo.CreateCostCenter(r.Context(), scope, dimensiondb.CreateCostCenterParams{
		OrganizationID: scope[0], Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name), Description: req.Description,
		ParentID: req.ParentID, OwnerRef: req.OwnerRef, Status: status, EffectiveFrom: from, EffectiveTo: to, CreatedBy: user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, centerResp{
		ID: created.ID, Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name), Description: req.Description,
		ParentID: req.ParentID, OwnerRef: req.OwnerRef, Status: status, EffectiveFrom: platform.FormatDate(from),
		EffectiveTo: platform.FormatDate(to), RowVersion: created.RowVersion,
	})
}

func (s *Server) getCostCenter(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "cost center")
	if !ok {
		return
	}
	row, err := s.repo.GetCostCenter(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, costCenterDTO(row))
}

func (s *Server) updateCostCenter(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "cost center")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req centerReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.ParentID != nil && *req.ParentID == id {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_PARENT", "cost center cannot be its own parent"))
		return
	}
	status, from, to, err := validateCenter(req.Name, req.Status, req.EffectiveFrom, req.EffectiveTo)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.repo.UpdateCostCenter(r.Context(), scope, dimensiondb.UpdateCostCenterParams{
		ID: id, RowVersion: rowVer, Name: strings.TrimSpace(req.Name), Description: req.Description,
		ParentID: req.ParentID, OwnerRef: req.OwnerRef, Status: status, EffectiveFrom: from, EffectiveTo: to, UpdatedBy: &user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"rowVersion": next})
}

func (s *Server) deleteCostCenter(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "cost center")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.repo.DeleteCostCenter(r.Context(), scope, dimensiondb.DeleteCostCenterParams{ID: id, RowVersion: rowVer, UpdatedBy: &user}); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// -----------------------------------------------------------------------------
// Profit Centers
// -----------------------------------------------------------------------------

func (s *Server) listProfitCenters(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListProfitCenters(r.Context(), scope, statusFilter(r), limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	data := make([]centerResp, 0, len(rows))
	for _, row := range rows {
		data = append(data, profitCenterDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: data, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createProfitCenter(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req centerReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code is required"))
		return
	}
	status, from, to, err := validateCenter(req.Name, req.Status, req.EffectiveFrom, req.EffectiveTo)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	created, err := s.repo.CreateProfitCenter(r.Context(), scope, dimensiondb.CreateProfitCenterParams{
		OrganizationID: scope[0], Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name), ParentID: req.ParentID,
		Status: status, EffectiveFrom: from, EffectiveTo: to, CreatedBy: user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, centerResp{
		ID: created.ID, Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name), ParentID: req.ParentID, Status: status,
		EffectiveFrom: platform.FormatDate(from), EffectiveTo: platform.FormatDate(to), RowVersion: created.RowVersion,
	})
}

func (s *Server) getProfitCenter(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "profit center")
	if !ok {
		return
	}
	row, err := s.repo.GetProfitCenter(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, profitCenterDTO(row))
}

func (s *Server) updateProfitCenter(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "profit center")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req centerReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.ParentID != nil && *req.ParentID == id {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_PARENT", "profit center cannot be its own parent"))
		return
	}
	status, from, to, err := validateCenter(req.Name, req.Status, req.EffectiveFrom, req.EffectiveTo)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.repo.UpdateProfitCenter(r.Context(), scope, dimensiondb.UpdateProfitCenterParams{
		ID: id, RowVersion: rowVer, Name: strings.TrimSpace(req.Name), ParentID: req.ParentID, Status: status,
		EffectiveFrom: from, EffectiveTo: to, UpdatedBy: &user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"rowVersion": next})
}

func (s *Server) deleteProfitCenter(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "profit center")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.repo.DeleteProfitCenter(r.Context(), scope, dimensiondb.DeleteProfitCenterParams{ID: id, RowVersion: rowVer, UpdatedBy: &user}); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// -----------------------------------------------------------------------------
// Allocation Rules
// -----------------------------------------------------------------------------

type allocTarget struct {
	CostCenterID uuid.UUID `json:"costCenterId"`
	Percentage   string    `json:"percentage"`
}

type allocReq struct {
	SourceCostCenterID uuid.UUID     `json:"sourceCostCenterId"`
	Name               string        `json:"name"`
	AllocationBasis    string        `json:"allocationBasis"`
	Targets            []allocTarget `json:"targets"`
	IsActive           *bool         `json:"isActive,omitempty"`
}

type allocResp struct {
	ID                 uuid.UUID       `json:"id"`
	SourceCostCenterID uuid.UUID       `json:"sourceCostCenterId"`
	Name               string          `json:"name"`
	AllocationBasis    string          `json:"allocationBasis"`
	Targets            json.RawMessage `json:"targets"`
	IsActive           bool            `json:"isActive"`
	RowVersion         int32           `json:"rowVersion"`
}

func allocDTO(a dimensiondb.CcCostAllocationRule) allocResp {
	return allocResp{ID: a.ID, SourceCostCenterID: a.SourceCostCenterID, Name: a.Name, AllocationBasis: a.AllocationBasis,
		Targets: json.RawMessage(a.Targets), IsActive: a.IsActive, RowVersion: a.RowVersion}
}

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
	httpx.WriteJSON(w, http.StatusOK, allocDTO(row))
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
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
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
	rule, err := s.repo.ExecuteAllocationRule(r.Context(), scope, id, platform.FormatNumeric(amtNum, 4), req.FiscalPeriodID, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	// Compute exact target distributions with *big.Rat (no IEEE-754 floats).
	var targets []allocTarget
	_ = json.Unmarshal(rule.Targets, &targets)
	hundred := big.NewRat(100, 1)
	distributions := make([]map[string]any, 0, len(targets))
	for _, t := range targets {
		pct, _ := new(big.Rat).SetString(strings.TrimSpace(t.Percentage))
		if pct == nil {
			continue
		}
		share := new(big.Rat).Quo(new(big.Rat).Mul(amtRat, pct), hundred)
		shareNum, _ := platform.RatToNumeric(share, 4)
		distributions = append(distributions, map[string]any{
			"costCenterId": t.CostCenterID,
			"percentage":   t.Percentage,
			"amount":       platform.FormatNumeric(shareNum, 4),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"ruleId":             rule.ID,
		"sourceCostCenterId": rule.SourceCostCenterID,
		"totalAmount":        platform.FormatNumeric(amtNum, 4),
		"distributions":      distributions,
	})
}

// -----------------------------------------------------------------------------
// Assignments (cc.cost_center_assignment & pc.profit_center_assignment)
// -----------------------------------------------------------------------------

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

