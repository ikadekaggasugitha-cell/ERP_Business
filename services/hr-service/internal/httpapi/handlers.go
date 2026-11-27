package httpapi

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/hr-service/internal/hrdb"
	"github.com/sinergiinvestasiproperti/erp/services/hr-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Store abstracts HR persistence for unit tests.
type Store interface {
	CreateAndEmit(ctx context.Context, scope []uuid.UUID, in repository.CreateEmployeeInput) (repository.EmployeeDTO, error)
	GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (repository.EmployeeDTO, error)
	FindByNationalID(ctx context.Context, scope []uuid.UUID, nationalID string) (repository.EmployeeDTO, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]repository.EmployeeDTO, int64, error)
	UpdateAndEmit(ctx context.Context, scope []uuid.UUID, in repository.UpdateEmployeeInput) (repository.EmployeeDTO, error)
	TerminateAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, updatedBy, corrID uuid.UUID) (repository.EmployeeDTO, error)
	Delete(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, updatedBy uuid.UUID) error
	CreatePosition(ctx context.Context, scope []uuid.UUID, in hrdb.CreatePositionParams) (hrdb.CreatePositionRow, error)
	ListPositionsWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]hrdb.ListPositionsRow, int64, error)
	CreateEmployment(ctx context.Context, scope []uuid.UUID, in hrdb.CreateEmploymentParams) (hrdb.CreateEmploymentRow, error)
	ListEmploymentsByEmployee(ctx context.Context, scope []uuid.UUID, employeeID uuid.UUID) ([]hrdb.ListEmploymentsByEmployeeRow, error)
	CreateCompensation(ctx context.Context, scope []uuid.UUID, in hrdb.CreateCompensationParams) (hrdb.CreateCompensationRow, error)
	ListCompensationsByEmployee(ctx context.Context, scope []uuid.UUID, employeeID uuid.UUID) ([]hrdb.ListCompensationsByEmployeeRow, error)
}

type leaveRequestRecord struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	EmployeeID     uuid.UUID  `json:"employeeId"`
	LeaveType      string     `json:"leaveType"`
	StartDate      string     `json:"startDate"`
	EndDate        string     `json:"endDate"`
	DaysCount      int        `json:"daysCount"`
	Reason         string     `json:"reason,omitempty"`
	Status         string     `json:"status"`
	CreatedBy      uuid.UUID  `json:"createdBy"`
	ApprovedBy     *uuid.UUID `json:"approvedBy,omitempty"`
	CreatedAt      string     `json:"createdAt"`
}

type attendanceLogRecord struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	EmployeeID     uuid.UUID `json:"employeeId"`
	LogDate        string    `json:"logDate"`
	Status         string    `json:"status"`
	ClockIn        string    `json:"clockIn,omitempty"`
	ClockOut       string    `json:"clockOut,omitempty"`
	Notes          string    `json:"notes,omitempty"`
	CreatedBy      uuid.UUID `json:"createdBy"`
}

type Handler struct {
	store       Store
	idem        *httpx.IdemStore
	mu          sync.RWMutex
	leaves      []leaveRequestRecord
	attendances []attendanceLogRecord
}

func NewRouter(store Store, metrics *httpx.Metrics, checks ...httpx.ReadinessCheck) http.Handler {
	h := &Handler{store: store, idem: httpx.NewIdemStore()}
	if metrics == nil {
		metrics = httpx.NewMetrics("hr-service")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "hr-service"})
	})
	mux.HandleFunc("GET /readyz", httpx.ReadinessHandler("hr-service", checks...))
	mux.HandleFunc("GET /metrics", metrics.Handler())

	mux.HandleFunc("POST /employees", h.handleCreateEmployee)
	mux.HandleFunc("POST /employees/search", h.handleSearchEmployee)
	mux.HandleFunc("GET /employees", h.handleListEmployees)
	mux.HandleFunc("GET /employees/{id}", h.handleGetEmployee)
	mux.HandleFunc("PUT /employees/{id}", h.handleUpdateEmployee)
	mux.HandleFunc("POST /employees/{id}/terminate", h.handleTerminateEmployee)
	mux.HandleFunc("DELETE /employees/{id}", h.handleDeleteEmployee)

	mux.HandleFunc("POST /positions", h.handleCreatePosition)
	mux.HandleFunc("GET /positions", h.handleListPositions)
	mux.HandleFunc("POST /employees/{id}/employments", h.handleCreateEmployment)
	mux.HandleFunc("GET /employees/{id}/employments", h.handleListEmployments)
	mux.HandleFunc("POST /employees/{id}/compensations", h.handleCreateCompensation)
	mux.HandleFunc("GET /employees/{id}/compensations", h.handleListCompensations)

	mux.HandleFunc("GET /leave-requests", h.handleListLeaveRequests)
	mux.HandleFunc("POST /leave-requests", h.handleCreateLeaveRequest)
	mux.HandleFunc("POST /leave-requests/{id}/approve", h.handleApproveLeaveRequest)
	mux.HandleFunc("GET /attendance-logs", h.handleListAttendanceLogs)
	mux.HandleFunc("POST /attendance-logs", h.handleCreateAttendanceLog)

	return metrics.Wrap(httpx.Middleware(mux))
}

type employeeMutationReq struct {
	EmployeeNo  string `json:"employeeNo"`
	FullName    string `json:"fullName"`
	NationalID  string `json:"nationalId"`
	DateOfBirth string `json:"dateOfBirth"`
	Email       string `json:"email"`
	HireDate    string `json:"hireDate"`
	UserRefID   string `json:"userRefId"`
}

func (h *Handler) handleCreateEmployee(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "hr:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req employeeMutationReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, h.idem, scope, &req)
	if done {
		return
	}
	defer release()

	empNo := strings.TrimSpace(req.EmployeeNo)
	fullName := strings.TrimSpace(req.FullName)
	if empNo == "" || fullName == "" {
		httpx.WriteError(w, r, httpx.BadRequest("employeeNo dan fullName wajib diisi"))
		return
	}
	hireDate, err := platform.ParseDate(req.HireDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("hireDate harus YYYY-MM-DD", httpx.Detail{Field: "hireDate", Issue: err.Error()}))
		return
	}
	dob := strings.TrimSpace(req.DateOfBirth)
	if dob != "" {
		if _, err := platform.ParseDate(dob); err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("dateOfBirth harus YYYY-MM-DD", httpx.Detail{Field: "dateOfBirth", Issue: err.Error()}))
			return
		}
	}
	var userRef *uuid.UUID
	if strings.TrimSpace(req.UserRefID) != "" {
		u, err := httpx.ParseUUIDParam("userRefId", req.UserRefID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		userRef = &u
	}
	dto, err := h.store.CreateAndEmit(r.Context(), scope, repository.CreateEmployeeInput{
		OrganizationID: scope[0],
		EmployeeNo:     empNo,
		FullName:       fullName,
		NationalID:     strings.TrimSpace(req.NationalID),
		DateOfBirth:    dob,
		Email:          strings.TrimSpace(req.Email),
		HireDate:       hireDate,
		UserRefID:      userRef,
		CorrelationID:  httpx.CorrelationUUID(r),
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, h.idem, scope, idemKey, reqHash, http.StatusCreated, dto.RowVersion, dto)
}

func (h *Handler) handleGetEmployee(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "hr:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	dto, err := h.store.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, dto.RowVersion, dto)
}

func (h *Handler) handleListEmployees(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "hr:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if nik := strings.TrimSpace(r.URL.Query().Get("nationalId")); nik != "" {
		dto, err := h.store.FindByNationalID(r.Context(), scope, nik)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
			Data: []repository.EmployeeDTO{dto},
			Page: httpx.NewPageMeta(1, 25, 1),
		})
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
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: rows,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (h *Handler) handleUpdateEmployee(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "hr:write"); err != nil {
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
	var req employeeMutationReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.FullName) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("fullName wajib diisi"))
		return
	}
	dob := strings.TrimSpace(req.DateOfBirth)
	if dob != "" {
		if _, err := platform.ParseDate(dob); err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("dateOfBirth harus YYYY-MM-DD", httpx.Detail{Field: "dateOfBirth", Issue: err.Error()}))
			return
		}
	}
	var userRef *uuid.UUID
	if strings.TrimSpace(req.UserRefID) != "" {
		u, err := httpx.ParseUUIDParam("userRefId", req.UserRefID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		userRef = &u
	}
	dto, err := h.store.UpdateAndEmit(r.Context(), scope, repository.UpdateEmployeeInput{
		ID:            id,
		RowVersion:    ver,
		FullName:      strings.TrimSpace(req.FullName),
		NationalID:    strings.TrimSpace(req.NationalID),
		DateOfBirth:   dob,
		Email:         strings.TrimSpace(req.Email),
		UserRefID:     userRef,
		CorrelationID: httpx.CorrelationUUID(r),
		UpdatedBy:     user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, dto.RowVersion, dto)
}

func (h *Handler) handleTerminateEmployee(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "hr:write"); err != nil {
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
	dto, err := h.store.TerminateAndEmit(r.Context(), scope, id, ver, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, dto.RowVersion, dto)
}

func (h *Handler) handleDeleteEmployee(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "hr:write"); err != nil {
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
	if err := h.store.Delete(r.Context(), scope, id, ver, user); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleCreatePosition(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "hr:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		Code              string `json:"code"`
		Title             string `json:"title"`
		OrganizationRefID string `json:"organizationRefId"`
		ParentPositionID  string `json:"parentPositionId"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, h.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Title) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan title wajib diisi"))
		return
	}
	orgRefID := scope[0]
	if strings.TrimSpace(req.OrganizationRefID) != "" {
		orgRefID, err = httpx.ParseUUIDParam("organizationRefId", req.OrganizationRefID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if orgRefID == platform.SystemOrgID || !hrOrgInScope(scope, orgRefID) {
			httpx.WriteError(w, r, httpx.Forbidden("organizationRefId di luar scope organisasi"))
			return
		}
	}
	var parentID *uuid.UUID
	if strings.TrimSpace(req.ParentPositionID) != "" {
		p, err := httpx.ParseUUIDParam("parentPositionId", req.ParentPositionID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		parentID = &p
	}
	row, err := h.store.CreatePosition(r.Context(), scope, hrdb.CreatePositionParams{
		OrganizationID:    scope[0],
		Code:              strings.TrimSpace(req.Code),
		Title:             strings.TrimSpace(req.Title),
		OrganizationRefID: orgRefID,
		ParentPositionID:  parentID,
		CreatedBy:         user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, h.idem, scope, idemKey, reqHash, http.StatusCreated, row.RowVersion, map[string]any{
		"id":                row.ID.String(),
		"organizationId":    row.OrganizationID.String(),
		"code":              row.Code,
		"title":             row.Title,
		"organizationRefId": row.OrganizationRefID.String(),
		"rowVersion":        row.RowVersion,
	})
}

func (h *Handler) handleListPositions(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "hr:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := h.store.ListPositionsWithCount(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		item := map[string]any{
			"id":                row.ID.String(),
			"organizationId":    row.OrganizationID.String(),
			"code":              row.Code,
			"title":             row.Title,
			"organizationRefId": row.OrganizationRefID.String(),
			"rowVersion":        row.RowVersion,
		}
		if row.ParentPositionID != nil {
			item["parentPositionId"] = row.ParentPositionID.String()
		}
		out = append(out, item)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (h *Handler) handleCreateEmployment(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "hr:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	empID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		PositionID     string `json:"positionId"`
		EmploymentType string `json:"employmentType"`
		EffectiveFrom  string `json:"effectiveFrom"`
		EffectiveTo    string `json:"effectiveTo"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, h.idem, scope, &req)
	if done {
		return
	}
	defer release()

	posID, err := httpx.ParseUUIDParam("positionId", req.PositionID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	effFrom, err := platform.ParseDate(req.EffectiveFrom)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("effectiveFrom harus YYYY-MM-DD"))
		return
	}
	var effTo pgtype.Date
	if strings.TrimSpace(req.EffectiveTo) != "" {
		effTo, err = platform.ParseDate(req.EffectiveTo)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("effectiveTo harus YYYY-MM-DD"))
			return
		}
	}
	row, err := h.store.CreateEmployment(r.Context(), scope, hrdb.CreateEmploymentParams{
		OrganizationID: scope[0],
		EmployeeID:     empID,
		PositionID:     posID,
		EmploymentType: strings.TrimSpace(req.EmploymentType),
		EffectiveFrom:  effFrom,
		EffectiveTo:    effTo,
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, h.idem, scope, idemKey, reqHash, http.StatusCreated, row.RowVersion, map[string]any{
		"id":             row.ID.String(),
		"employeeId":     row.EmployeeID.String(),
		"positionId":     row.PositionID.String(),
		"employmentType": row.EmploymentType,
		"effectiveFrom":  platform.FormatDate(row.EffectiveFrom),
		"effectiveTo":    platform.FormatDate(row.EffectiveTo),
		"rowVersion":     row.RowVersion,
	})
}

func (h *Handler) handleListEmployments(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "hr:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	empID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := h.store.ListEmploymentsByEmployee(r.Context(), scope, empID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":             row.ID.String(),
			"employeeId":     row.EmployeeID.String(),
			"positionId":     row.PositionID.String(),
			"employmentType": row.EmploymentType,
			"effectiveFrom":  platform.FormatDate(row.EffectiveFrom),
			"effectiveTo":    platform.FormatDate(row.EffectiveTo),
			"rowVersion":     row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(1, max(1, len(out)), len(out)),
	})
}

func (h *Handler) handleCreateCompensation(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "hr:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	empID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		ComponentCode string `json:"componentCode"`
		Amount        string `json:"amount"`
		EffectiveFrom string `json:"effectiveFrom"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, h.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if strings.TrimSpace(req.ComponentCode) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("componentCode wajib diisi"))
		return
	}
	amtNum, err := platform.ParseNumeric(req.Amount)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("amount tidak valid"))
		return
	}
	amtRat := platform.NumericToRat(amtNum)
	if amtRat == nil || amtRat.Sign() < 0 {
		httpx.WriteError(w, r, httpx.BadRequest("amount harus >= 0"))
		return
	}
	effFrom, err := platform.ParseDate(req.EffectiveFrom)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("effectiveFrom harus YYYY-MM-DD"))
		return
	}
	row, err := h.store.CreateCompensation(r.Context(), scope, hrdb.CreateCompensationParams{
		OrganizationID: scope[0],
		EmployeeID:     empID,
		ComponentCode:  strings.ToUpper(strings.TrimSpace(req.ComponentCode)),
		Amount:         amtNum,
		EffectiveFrom:  effFrom,
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, h.idem, scope, idemKey, reqHash, http.StatusCreated, row.RowVersion, map[string]any{
		"id":            row.ID.String(),
		"employeeId":    row.EmployeeID.String(),
		"componentCode": row.ComponentCode,
		"amount":        platform.FormatNumeric(row.Amount, 4),
		"effectiveFrom": platform.FormatDate(row.EffectiveFrom),
		"rowVersion":    row.RowVersion,
	})
}

func (h *Handler) handleSearchEmployee(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "hr:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		NationalID string `json:"nationalId"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.NationalID) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("nationalId wajib diisi"))
		return
	}
	emp, err := h.store.FindByNationalID(r.Context(), scope, strings.TrimSpace(req.NationalID))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, emp.RowVersion, emp)
}

func hrOrgInScope(scope []uuid.UUID, target uuid.UUID) bool {
	if target == platform.SystemOrgID {
		return false
	}
	for _, id := range scope {
		if id == target {
			return true
		}
	}
	return false
}

func (h *Handler) handleListCompensations(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "hr:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	empID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := h.store.ListCompensationsByEmployee(r.Context(), scope, empID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":            row.ID.String(),
			"employeeId":    row.EmployeeID.String(),
			"componentCode": row.ComponentCode,
			"amount":        platform.FormatNumeric(row.Amount, 4),
			"effectiveFrom": platform.FormatDate(row.EffectiveFrom),
			"rowVersion":    row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(1, max(1, len(out)), len(out)),
	})
}

func (h *Handler) handleListLeaveRequests(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "hr:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	empFilter := strings.TrimSpace(r.URL.Query().Get("employeeId"))
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))

	h.mu.RLock()
	out := make([]leaveRequestRecord, 0, len(h.leaves))
	for _, lv := range h.leaves {
		if !slices.Contains(scope, lv.OrganizationID) {
			continue
		}
		if empFilter != "" && lv.EmployeeID.String() != empFilter {
			continue
		}
		if statusFilter != "" && !strings.EqualFold(lv.Status, statusFilter) {
			continue
		}
		out = append(out, lv)
	}
	h.mu.RUnlock()

	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(1, max(1, len(out)), len(out)),
	})
}

func (h *Handler) handleCreateLeaveRequest(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "hr:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		EmployeeID string `json:"employeeId"`
		LeaveType  string `json:"leaveType"`
		StartDate  string `json:"startDate"`
		EndDate    string `json:"endDate"`
		Reason     string `json:"reason"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, h.idem, scope, &req)
	if done {
		return
	}
	defer release()

	empID, err := httpx.ParseUUIDParam("employeeId", req.EmployeeID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	leaveType := strings.TrimSpace(req.LeaveType)
	switch leaveType {
	case "Annual", "Sick", "Maternity", "Unpaid", "Personal":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("leaveType harus Annual, Sick, Maternity, Unpaid, atau Personal"))
		return
	}
	startDate, err := platform.ParseDate(req.StartDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("startDate harus YYYY-MM-DD"))
		return
	}
	endDate, err := platform.ParseDate(req.EndDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("endDate harus YYYY-MM-DD"))
		return
	}
	if endDate.Time.Before(startDate.Time) {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_DATE_RANGE", "endDate tidak boleh lebih awal dari startDate"))
		return
	}
	days := int(endDate.Time.Sub(startDate.Time).Hours()/24) + 1

	rec := leaveRequestRecord{
		ID:             uuid.New(),
		OrganizationID: scope[0],
		EmployeeID:     empID,
		LeaveType:      leaveType,
		StartDate:      platform.FormatDate(startDate),
		EndDate:        platform.FormatDate(endDate),
		DaysCount:      days,
		Reason:         strings.TrimSpace(req.Reason),
		Status:         "Pending",
		CreatedBy:      user,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	h.mu.Lock()
	h.leaves = append(h.leaves, rec)
	h.mu.Unlock()

	httpx.SaveIDResponse(w, h.idem, scope, idemKey, reqHash, http.StatusCreated, 1, rec)
}

func (h *Handler) handleApproveLeaveRequest(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "hr:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range h.leaves {
		if h.leaves[i].ID == id && slices.Contains(scope, h.leaves[i].OrganizationID) {
			if h.leaves[i].Status != "Pending" {
				httpx.WriteError(w, r, httpx.Conflict("INVALID_STATE", "leave request tidak berstatus Pending"))
				return
			}
			if h.leaves[i].CreatedBy == user && !httpx.HasRole(r.Context(), "SYSTEM_ADMIN") {
				httpx.WriteError(w, r, httpx.Forbidden("pembuat pengajuan cuti tidak dapat menyetujui pengajuannya sendiri (SoD)"))
				return
			}
			h.leaves[i].Status = "Approved"
			h.leaves[i].ApprovedBy = &user
			httpx.WriteJSON(w, http.StatusOK, h.leaves[i])
			return
		}
	}
	httpx.WriteError(w, r, httpx.NotFound("leave request tidak ditemukan"))
}

func (h *Handler) handleListAttendanceLogs(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "hr:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	empFilter := strings.TrimSpace(r.URL.Query().Get("employeeId"))
	dateFilter := strings.TrimSpace(r.URL.Query().Get("date"))

	h.mu.RLock()
	out := make([]attendanceLogRecord, 0, len(h.attendances))
	for _, a := range h.attendances {
		if !slices.Contains(scope, a.OrganizationID) {
			continue
		}
		if empFilter != "" && a.EmployeeID.String() != empFilter {
			continue
		}
		if dateFilter != "" && a.LogDate != dateFilter {
			continue
		}
		out = append(out, a)
	}
	h.mu.RUnlock()

	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(1, max(1, len(out)), len(out)),
	})
}

func (h *Handler) handleCreateAttendanceLog(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "hr:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		EmployeeID string `json:"employeeId"`
		LogDate    string `json:"logDate"`
		Status     string `json:"status"`
		ClockIn    string `json:"clockIn,omitempty"`
		ClockOut   string `json:"clockOut,omitempty"`
		Notes      string `json:"notes,omitempty"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, h.idem, scope, &req)
	if done {
		return
	}
	defer release()

	empID, err := httpx.ParseUUIDParam("employeeId", req.EmployeeID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	logDate, err := platform.ParseDate(req.LogDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("logDate harus YYYY-MM-DD"))
		return
	}
	status := strings.TrimSpace(req.Status)
	switch status {
	case "Present", "Late", "Absent", "Leave", "Remote":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("status harus Present, Late, Absent, Leave, atau Remote"))
		return
	}

	rec := attendanceLogRecord{
		ID:             uuid.New(),
		OrganizationID: scope[0],
		EmployeeID:     empID,
		LogDate:        platform.FormatDate(logDate),
		Status:         status,
		ClockIn:        strings.TrimSpace(req.ClockIn),
		ClockOut:       strings.TrimSpace(req.ClockOut),
		Notes:          strings.TrimSpace(req.Notes),
		CreatedBy:      user,
	}
	h.mu.Lock()
	h.attendances = append(h.attendances, rec)
	h.mu.Unlock()

	httpx.SaveIDResponse(w, h.idem, scope, idemKey, reqHash, http.StatusCreated, 1, rec)
}
