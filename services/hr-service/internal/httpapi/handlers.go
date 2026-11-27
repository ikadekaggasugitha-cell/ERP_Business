package httpapi

import (
	"context"
	"net/http"
	"sync"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/hr-service/internal/hrdb"
	"github.com/sinergiinvestasiproperti/erp/services/hr-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

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
