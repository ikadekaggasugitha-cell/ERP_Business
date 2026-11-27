package httpapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/maintenance-service/internal/mntdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type MaintenanceStore interface {
	CreateSchedule(ctx context.Context, scope []uuid.UUID, arg mntdb.CreateMaintenanceScheduleParams) (mntdb.CreateMaintenanceScheduleRow, error)
	ListSchedules(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]mntdb.MntMaintenanceSchedule, int64, error)
	UpdateSchedule(ctx context.Context, scope []uuid.UUID, arg mntdb.UpdateMaintenanceScheduleParams) (int32, error)
	DeleteSchedule(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error
	CreateWorkOrder(ctx context.Context, scope []uuid.UUID, arg mntdb.CreateWorkOrderParams) (mntdb.CreateWorkOrderRow, error)
	GetWorkOrder(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (mntdb.MntWorkOrder, error)
	ListWorkOrders(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]mntdb.MntWorkOrder, int64, error)
	AssignWorkOrder(ctx context.Context, scope []uuid.UUID, arg mntdb.AssignWorkOrderParams) (int32, error)
	StartWorkOrder(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error)
	CompleteWorkOrderAndEmit(ctx context.Context, scope []uuid.UUID, id, org, actor, corr uuid.UUID, version int32) (int32, string, error)
	CloseWorkOrder(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error)
	AddPartAndEmit(ctx context.Context, scope []uuid.UUID, arg mntdb.CreateWorkOrderPartParams, warehouseID *uuid.UUID, corr uuid.UUID) (mntdb.CreateWorkOrderPartRow, error)
	ListParts(ctx context.Context, scope []uuid.UUID, woID uuid.UUID) ([]mntdb.MntWorkOrderPart, error)
	AddChecklist(ctx context.Context, scope []uuid.UUID, arg mntdb.CreateWorkOrderChecklistParams) (mntdb.CreateWorkOrderChecklistRow, error)
	ListChecklist(ctx context.Context, scope []uuid.UUID, woID uuid.UUID) ([]mntdb.MntWorkOrderChecklist, error)
	UpdateChecklist(ctx context.Context, scope []uuid.UUID, arg mntdb.UpdateWorkOrderChecklistParams) (int32, error)
	AddCost(ctx context.Context, scope []uuid.UUID, arg mntdb.CreateMaintenanceCostParams) (mntdb.CreateMaintenanceCostRow, error)
	ListCosts(ctx context.Context, scope []uuid.UUID, woID uuid.UUID) ([]mntdb.MntMaintenanceCost, error)
}

type Server struct {
	store MaintenanceStore
	idem  *httpx.IdemStore
}

func NewServer(store MaintenanceStore) *Server {
	return &Server{store: store, idem: httpx.NewIdemStore()}
}

func beginIdemOptional(w http.ResponseWriter, r *http.Request, idem *httpx.IdemStore, scope []uuid.UUID, dst any) (string, string, func(), bool) {
	if r.Body == nil {
		r.Body = io.NopCloser(strings.NewReader("{}"))
	} else {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("failed to read request body"))
			return "", "", func() {}, true
		}
		if len(bytes.TrimSpace(raw)) == 0 {
			raw = []byte("{}")
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))
	}
	return httpx.BeginIdempotent(w, r, idem, scope, dst)
}


func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /maintenance-schedules", s.listSchedules)
	mux.HandleFunc("POST /maintenance-schedules", s.createSchedule)
	mux.HandleFunc("PUT /maintenance-schedules/{id}", s.updateSchedule)
	mux.HandleFunc("DELETE /maintenance-schedules/{id}", s.deleteSchedule)

	mux.HandleFunc("GET /maintenance-work-orders", s.listWorkOrders)
	mux.HandleFunc("POST /maintenance-work-orders", s.createWorkOrder)
	mux.HandleFunc("GET /maintenance-work-orders/{id}", s.getWorkOrder)
	mux.HandleFunc("POST /maintenance-work-orders/{id}/assign", s.assignWorkOrder)
	mux.HandleFunc("POST /maintenance-work-orders/{id}/start", s.startWorkOrder)
	mux.HandleFunc("POST /maintenance-work-orders/{id}/complete", s.completeWorkOrder)
	mux.HandleFunc("POST /maintenance-work-orders/{id}/close", s.closeWorkOrder)

	mux.HandleFunc("GET /maintenance-work-orders/{woId}/parts", s.listParts)
	mux.HandleFunc("POST /maintenance-work-orders/{woId}/parts", s.addPart)

	mux.HandleFunc("GET /maintenance-work-orders/{woId}/checklist", s.listChecklist)
	mux.HandleFunc("POST /maintenance-work-orders/{woId}/checklist", s.addChecklist)
	mux.HandleFunc("PUT /maintenance-checklist/{id}", s.updateChecklist)

	mux.HandleFunc("GET /maintenance-work-orders/{woId}/costs", s.listCosts)
	mux.HandleFunc("POST /maintenance-work-orders/{woId}/costs", s.addCost)
}

func requireRead(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	if err := httpx.RequirePermission(r, "maintenance:read"); err != nil {
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
	if err := httpx.RequirePermission(r, "maintenance:write"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
}

func parseOptDate(raw *string) (pgtype.Date, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return pgtype.Date{}, nil
	}
	d, err := platform.ParseDate(*raw)
	if err != nil {
		return pgtype.Date{}, httpx.BadRequest("date must be YYYY-MM-DD")
	}
	return d, nil
}

func parsePositiveDecimal(raw, field string) (pgtype.Numeric, error) {
	num, err := platform.ParseNumeric(strings.TrimSpace(raw))
	if err != nil || platform.NumericToRat(num) == nil || platform.NumericToRat(num).Sign() <= 0 {
		return pgtype.Numeric{}, httpx.BadRequest(field + " must be a positive decimal string")
	}
	return num, nil
}

