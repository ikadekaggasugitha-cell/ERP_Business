// Package httpapi implements Cost Center, Profit Center and Allocation Rule
// endpoints with RLS scope, optimistic locking (If-Match) and idempotency.
package httpapi

import (
	"context"
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
	if scope[0] == platform.SystemOrgID && !httpx.HasRole(r.Context(), "SYSTEM_ADMIN") {
		httpx.WriteError(w, r, httpx.Forbidden("data dimensi sistem tidak dapat dimodifikasi oleh non-SYSTEM_ADMIN"))
		return nil, uuid.Nil, false
	}
	return scope, user, true
}

func guardSystemOrgMutation(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) bool {
	if orgID == platform.SystemOrgID && !httpx.HasRole(r.Context(), "SYSTEM_ADMIN") {
		httpx.WriteError(w, r, httpx.Forbidden("data dimensi sistem tidak dapat dimodifikasi oleh non-SYSTEM_ADMIN"))
		return false
	}
	return true
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
	id, err := httpx.ParsePathUUID(r, "id")
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
