package httpapi

import (
	"context"
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/payroll-service/internal/calc"
	"github.com/sinergiinvestasiproperti/erp/services/payroll-service/internal/payrolldb"
	"github.com/sinergiinvestasiproperti/erp/services/payroll-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Store abstracts payroll persistence for unit tests.
type Store interface {
	CreateRun(ctx context.Context, scope []uuid.UUID, orgID uuid.UUID, runNo string, month, year int32, corrID, createdBy uuid.UUID) (repository.PayrollRunDetail, error)
	GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (repository.PayrollRunDetail, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]payrolldb.ListPayrollRunsRow, int64, error)
	CalculateAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, summaryGross *big.Rat, employees []calc.EmployeeInput, updatedBy uuid.UUID) (repository.PayrollRunDetail, error)
	ApproveAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, updatedBy, corrID uuid.UUID) (repository.PayrollRunDetail, error)
	PostAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, glJournalRef string, updatedBy, corrID uuid.UUID) (repository.PayrollRunDetail, error)
	DeleteDraft(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, updatedBy uuid.UUID) error
}

type Handler struct {
	store Store
	idem  *httpx.IdemStore
}

func NewRouter(store Store, metrics *httpx.Metrics, checks ...httpx.ReadinessCheck) http.Handler {
	h := &Handler{store: store, idem: httpx.NewIdemStore()}
	if metrics == nil {
		metrics = httpx.NewMetrics("payroll-service")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "payroll-service"})
	})
	mux.HandleFunc("GET /readyz", httpx.ReadinessHandler("payroll-service", checks...))
	mux.HandleFunc("GET /metrics", metrics.Handler())

	mux.HandleFunc("POST /payroll-runs", h.handleCreateRun)
	mux.HandleFunc("GET /payroll-runs", h.handleListRuns)
	mux.HandleFunc("GET /payroll-runs/{id}", h.handleGetRun)
	mux.HandleFunc("DELETE /payroll-runs/{id}", h.handleDeleteDraftRun)
	mux.HandleFunc("POST /payroll-runs/{id}/calculate", h.handleCalculateRun)
	mux.HandleFunc("POST /payroll-runs/{id}/approve", h.handleApproveRun)
	mux.HandleFunc("POST /payroll-runs/{id}/post", h.handlePostRun)

	return metrics.Wrap(httpx.Middleware(mux))
}

func parseNonNegRat(field, raw string) (*big.Rat, error) {
	if strings.TrimSpace(raw) == "" {
		return new(big.Rat), nil
	}
	num, err := platform.ParseNumeric(raw)
	if err != nil {
		return nil, httpx.BadRequest(field+" tidak valid", httpx.Detail{Field: field, Issue: err.Error()})
	}
	rat := platform.NumericToRat(num)
	if rat == nil || rat.Sign() < 0 {
		return nil, httpx.BadRequest(field+" harus >= 0", httpx.Detail{Field: field, Issue: "must be >= 0"})
	}
	return rat, nil
}

func (h *Handler) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "payroll:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		RunNo       string `json:"runNo"`
		PeriodMonth int32  `json:"periodMonth"`
		PeriodYear  int32  `json:"periodYear"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, h.idem, scope, &req)
	if done {
		return
	}
	defer release()

	runNo := strings.TrimSpace(req.RunNo)
	if runNo == "" || req.PeriodMonth < 1 || req.PeriodMonth > 12 || req.PeriodYear < 2000 || req.PeriodYear > 2100 {
		httpx.WriteError(w, r, httpx.BadRequest("runNo, periodMonth (1..12), dan periodYear wajib valid"))
		return
	}
	detail, err := h.store.CreateRun(r.Context(), scope, scope[0], runNo, req.PeriodMonth, req.PeriodYear, httpx.CorrelationUUID(r), user)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, h.idem, scope, idemKey, reqHash, http.StatusCreated, detail.RowVersion, detail)
}

func (h *Handler) handleGetRun(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "payroll:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	detail, err := h.store.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.RowVersion, detail)
}

func (h *Handler) handleListRuns(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "payroll:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var status *string
	if s := strings.TrimSpace(r.URL.Query().Get("status")); s != "" {
		status = &s
	}
	rows, total, err := h.store.ListWithCount(r.Context(), scope, status, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		item := map[string]any{
			"id":             row.ID.String(),
			"organizationId": row.OrganizationID.String(),
			"runNo":          row.RunNo,
			"periodMonth":    row.PeriodMonth,
			"periodYear":     row.PeriodYear,
			"status":         row.Status,
			"rowVersion":     row.RowVersion,
		}
		if row.GlJournalRef != nil {
			item["glJournalRef"] = *row.GlJournalRef
		}
		out = append(out, item)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (h *Handler) handleDeleteDraftRun(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "payroll:write"); err != nil {
		httpx.WriteError(w, r, err)
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
	if err := h.store.DeleteDraft(r.Context(), scope, id, ver, user); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleCalculateRun(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "payroll:write"); err != nil {
		httpx.WriteError(w, r, err)
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
		TotalAmount string `json:"totalAmount"`
		Employees   []struct {
			EmployeeID      string `json:"employeeId"`
			BasicSalary     string `json:"basicSalary"`
			Allowance       string `json:"allowance"`
			Bonus           string `json:"bonus"`
			OtherDeductions string `json:"otherDeductions"`
			PTKP            string `json:"ptkp"`
			HasNPWP         *bool  `json:"hasNpwp"`
		} `json:"employees"`
	}
	if err := httpx.DecodeOptionalJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	summaryGross, err := parseNonNegRat("totalAmount", req.TotalAmount)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if len(req.Employees) == 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("EMPTY_EMPLOYEES", "minimal 1 karyawan wajib disertakan dalam perhitungan payroll"))
		return
	}
	empInputs := make([]calc.EmployeeInput, 0, len(req.Employees))
	for _, emp := range req.Employees {
		empID, err := httpx.ParseUUIDParam("employees.employeeId", emp.EmployeeID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		basic, err := parseNonNegRat("employees.basicSalary", emp.BasicSalary)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		allow, err := parseNonNegRat("employees.allowance", emp.Allowance)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		bonus, err := parseNonNegRat("employees.bonus", emp.Bonus)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		otherDed, err := parseNonNegRat("employees.otherDeductions", emp.OtherDeductions)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		hasNPWP := true
		if emp.HasNPWP != nil {
			hasNPWP = *emp.HasNPWP
		}
		empInputs = append(empInputs, calc.EmployeeInput{
			EmployeeID:      empID,
			BasicSalary:     basic,
			Allowance:       allow,
			Bonus:           bonus,
			OtherDeductions: otherDed,
			PTKP:            emp.PTKP,
			HasNPWP:         hasNPWP,
		})
	}
	detail, err := h.store.CalculateAndEmit(r.Context(), scope, id, ver, summaryGross, empInputs, user)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.RowVersion, detail)
}

func (h *Handler) handleApproveRun(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "payroll:write"); err != nil {
		httpx.WriteError(w, r, err)
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
	detail, err := h.store.ApproveAndEmit(r.Context(), scope, id, ver, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.RowVersion, detail)
}

func (h *Handler) handlePostRun(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "payroll:write"); err != nil {
		httpx.WriteError(w, r, err)
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
		GLJournalRef string `json:"glJournalRef"`
	}
	if err := httpx.DecodeOptionalJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	detail, err := h.store.PostAndEmit(r.Context(), scope, id, ver, strings.TrimSpace(req.GLJournalRef), user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.RowVersion, detail)
}
