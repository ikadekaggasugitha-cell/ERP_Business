package httpapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

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

func fmtOptTS(ts pgtype.Timestamptz) *string {
	if !ts.Valid {
		return nil
	}
	s := ts.Time.UTC().Format(time.RFC3339)
	return &s
}

func fmtOptDate(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := platform.FormatDate(d)
	return &s
}

type scheduleDTO struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	AssetRef       uuid.UUID `json:"assetRef"`
	ScheduleType   string    `json:"scheduleType"`
	IntervalValue  int32     `json:"intervalValue"`
	IntervalUnit   string    `json:"intervalUnit"`
	NextDueDate    *string   `json:"nextDueDate,omitempty"`
	IsActive       bool      `json:"isActive"`
	IsSystem       bool      `json:"isSystem"`
	CreatedAt      *string   `json:"createdAt,omitempty"`
	UpdatedAt      *string   `json:"updatedAt,omitempty"`
	RowVersion     int32     `json:"rowVersion"`
}

func toScheduleDTO(r mntdb.MntMaintenanceSchedule) scheduleDTO {
	return scheduleDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, AssetRef: r.AssetRef,
		ScheduleType: r.ScheduleType, IntervalValue: r.IntervalValue, IntervalUnit: r.IntervalUnit,
		NextDueDate: fmtOptDate(r.NextDueDate), IsActive: r.IsActive,
		IsSystem: r.OrganizationID == platform.SystemOrgID,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type workOrderDTO struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  uuid.UUID  `json:"organizationId"`
	WoNo            string     `json:"woNo"`
	AssetRef        uuid.UUID  `json:"assetRef"`
	MaintenanceType string     `json:"maintenanceType"`
	Status          string     `json:"status"`
	AssigneeRef     *uuid.UUID `json:"assigneeRef,omitempty"`
	VendorRef       *uuid.UUID `json:"vendorRef,omitempty"`
	ScheduledDate   *string    `json:"scheduledDate,omitempty"`
	DowntimeStart   *string    `json:"downtimeStart,omitempty"`
	DowntimeEnd     *string    `json:"downtimeEnd,omitempty"`
	CreatedAt       *string    `json:"createdAt,omitempty"`
	UpdatedAt       *string    `json:"updatedAt,omitempty"`
	RowVersion      int32      `json:"rowVersion"`
}

func toWorkOrderDTO(r mntdb.MntWorkOrder) workOrderDTO {
	return workOrderDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, WoNo: r.WoNo, AssetRef: r.AssetRef,
		MaintenanceType: r.MaintenanceType, Status: r.Status,
		AssigneeRef: r.AssigneeRef, VendorRef: r.VendorRef,
		ScheduledDate: fmtOptDate(r.ScheduledDate),
		DowntimeStart: fmtOptTS(r.DowntimeStart), DowntimeEnd: fmtOptTS(r.DowntimeEnd),
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type partDTO struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	WorkOrderID    uuid.UUID `json:"workOrderId"`
	ItemRef        uuid.UUID `json:"itemRef"`
	Quantity       string    `json:"quantity"`
	UnitCost       string    `json:"unitCost"`
	CreatedAt      *string   `json:"createdAt,omitempty"`
	UpdatedAt      *string   `json:"updatedAt,omitempty"`
	RowVersion     int32     `json:"rowVersion"`
}

func toPartDTO(r mntdb.MntWorkOrderPart) partDTO {
	return partDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, WorkOrderID: r.WorkOrderID,
		ItemRef: r.ItemRef, Quantity: platform.FormatNumeric(r.Quantity, 4),
		UnitCost: platform.FormatNumeric(r.UnitCost, 4),
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type checklistDTO struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	WorkOrderID    uuid.UUID `json:"workOrderId"`
	StepNo         int32     `json:"stepNo"`
	Description    string    `json:"description"`
	IsDone         bool      `json:"isDone"`
	CreatedAt      *string   `json:"createdAt,omitempty"`
	UpdatedAt      *string   `json:"updatedAt,omitempty"`
	RowVersion     int32     `json:"rowVersion"`
}

func toChecklistDTO(r mntdb.MntWorkOrderChecklist) checklistDTO {
	return checklistDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, WorkOrderID: r.WorkOrderID,
		StepNo: r.StepNo, Description: r.Description, IsDone: r.IsDone,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type costDTO struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	WorkOrderID    uuid.UUID `json:"workOrderId"`
	CostType       string    `json:"costType"`
	Amount         string    `json:"amount"`
	CreatedAt      *string   `json:"createdAt,omitempty"`
	UpdatedAt      *string   `json:"updatedAt,omitempty"`
	RowVersion     int32     `json:"rowVersion"`
}

func toCostDTO(r mntdb.MntMaintenanceCost) costDTO {
	return costDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, WorkOrderID: r.WorkOrderID,
		CostType: r.CostType, Amount: platform.FormatNumeric(r.Amount, 4),
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
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

func (s *Server) listSchedules(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.store.ListSchedules(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]scheduleDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toScheduleDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createSchedule(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		AssetRef      string  `json:"assetRef"`
		ScheduleType  string  `json:"scheduleType"`
		IntervalValue int32   `json:"intervalValue"`
		IntervalUnit  string  `json:"intervalUnit"`
		NextDueDate   *string `json:"nextDueDate"`
		IsActive      *bool   `json:"isActive"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	assetRef, err := uuid.Parse(req.AssetRef)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid assetRef is required"))
		return
	}
	if req.ScheduleType != "TimeBased" && req.ScheduleType != "UsageBased" {
		httpx.WriteError(w, r, httpx.BadRequest("scheduleType must be TimeBased or UsageBased"))
		return
	}
	switch req.IntervalUnit {
	case "Day", "Month", "Hour", "Cycle":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("intervalUnit must be Day, Month, Hour, or Cycle"))
		return
	}
	if req.IntervalValue <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("intervalValue must be > 0"))
		return
	}
	nd, err := parseOptDate(req.NextDueDate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	out, err := s.store.CreateSchedule(r.Context(), scope, mntdb.CreateMaintenanceScheduleParams{
		OrganizationID: scope[0], AssetRef: assetRef, ScheduleType: req.ScheduleType,
		IntervalValue: req.IntervalValue, IntervalUnit: req.IntervalUnit, NextDueDate: nd, IsActive: active,
		CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) updateSchedule(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
		ScheduleType  string  `json:"scheduleType"`
		IntervalValue int32   `json:"intervalValue"`
		IntervalUnit  string  `json:"intervalUnit"`
		NextDueDate   *string `json:"nextDueDate"`
		IsActive      bool    `json:"isActive"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if (req.ScheduleType != "TimeBased" && req.ScheduleType != "UsageBased") || req.IntervalValue <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("valid scheduleType and positive intervalValue required"))
		return
	}
	nd, err := parseOptDate(req.NextDueDate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.store.UpdateSchedule(r.Context(), scope, mntdb.UpdateMaintenanceScheduleParams{
		ID: id, ScheduleType: req.ScheduleType, IntervalValue: req.IntervalValue, IntervalUnit: req.IntervalUnit,
		NextDueDate: nd, IsActive: req.IsActive, UpdatedBy: &user, RowVersion: ver,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "rowVersion": next})
}

func (s *Server) deleteSchedule(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
	if err := s.store.DeleteSchedule(r.Context(), scope, id, user, ver); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listWorkOrders(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var st *string
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
		st = &v
	}
	rows, total, err := s.store.ListWorkOrders(r.Context(), scope, st, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]workOrderDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toWorkOrderDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createWorkOrder(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		WoNo            string  `json:"woNo"`
		AssetRef        string  `json:"assetRef"`
		MaintenanceType string  `json:"maintenanceType"`
		AssigneeRef     *string `json:"assigneeRef"`
		VendorRef       *string `json:"vendorRef"`
		ScheduledDate   *string `json:"scheduledDate"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	assetRef, err := uuid.Parse(req.AssetRef)
	if err != nil || strings.TrimSpace(req.WoNo) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("woNo and valid assetRef are required"))
		return
	}
	if req.MaintenanceType != "Preventive" && req.MaintenanceType != "Corrective" {
		httpx.WriteError(w, r, httpx.BadRequest("maintenanceType must be Preventive or Corrective"))
		return
	}
	var assigneeRef, vendorRef *uuid.UUID
	if req.AssigneeRef != nil && strings.TrimSpace(*req.AssigneeRef) != "" {
		u, e := uuid.Parse(*req.AssigneeRef)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid assigneeRef"))
			return
		}
		assigneeRef = &u
	}
	if req.VendorRef != nil && strings.TrimSpace(*req.VendorRef) != "" {
		u, e := uuid.Parse(*req.VendorRef)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid vendorRef"))
			return
		}
		vendorRef = &u
	}
	sd, err := parseOptDate(req.ScheduledDate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := s.store.CreateWorkOrder(r.Context(), scope, mntdb.CreateWorkOrderParams{
		OrganizationID: scope[0], WoNo: strings.TrimSpace(req.WoNo), AssetRef: assetRef,
		MaintenanceType: req.MaintenanceType, AssigneeRef: assigneeRef, VendorRef: vendorRef,
		ScheduledDate: sd, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "Open", "rowVersion": out.RowVersion})
}

func (s *Server) getWorkOrder(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	row, err := s.store.GetWorkOrder(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, toWorkOrderDTO(row))
}

func (s *Server) assignWorkOrder(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
		AssigneeRef *string `json:"assigneeRef"`
		VendorRef   *string `json:"vendorRef"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	var assigneeRef, vendorRef *uuid.UUID
	if req.AssigneeRef != nil && strings.TrimSpace(*req.AssigneeRef) != "" {
		u, e := uuid.Parse(*req.AssigneeRef)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid assigneeRef"))
			return
		}
		assigneeRef = &u
	}
	if req.VendorRef != nil && strings.TrimSpace(*req.VendorRef) != "" {
		u, e := uuid.Parse(*req.VendorRef)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid vendorRef"))
			return
		}
		vendorRef = &u
	}
	if assigneeRef == nil && vendorRef == nil {
		httpx.WriteError(w, r, httpx.BadRequest("assigneeRef or vendorRef is required"))
		return
	}
	next, err := s.store.AssignWorkOrder(r.Context(), scope, mntdb.AssignWorkOrderParams{
		ID: id, AssigneeRef: assigneeRef, VendorRef: vendorRef, UpdatedBy: &user, RowVersion: ver,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Assigned", "rowVersion": next})
}

func (s *Server) startWorkOrder(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
	var req struct{}
	idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	next, err := s.store.StartWorkOrder(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "InProgress", "rowVersion": next})
}

func (s *Server) completeWorkOrder(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
	var req struct{}
	idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	next, totalCost, err := s.store.CompleteWorkOrderAndEmit(r.Context(), scope, id, scope[0], user, httpx.CorrelationUUID(r), ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Completed", "totalCost": totalCost, "rowVersion": next})
}

func (s *Server) closeWorkOrder(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
	var req struct{}
	idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	next, err := s.store.CloseWorkOrder(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Closed", "rowVersion": next})
}

func (s *Server) listParts(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	woID, err := httpx.ParseUUIDParam("woId", r.PathValue("woId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListParts(r.Context(), scope, woID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]partDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toPartDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) addPart(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	woID, err := httpx.ParseUUIDParam("woId", r.PathValue("woId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		ItemRef     string  `json:"itemRef"`
		Quantity    string  `json:"quantity"`
		UnitCost    string  `json:"unitCost"`
		WarehouseID *string `json:"warehouseId"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	itemRef, err := uuid.Parse(req.ItemRef)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid itemRef is required"))
		return
	}
	qty, err := parsePositiveDecimal(req.Quantity, "quantity")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	uc, err := platform.ParseOptionalNumeric(req.UnitCost)
	if err != nil || (platform.NumericToRat(uc) != nil && platform.NumericToRat(uc).Sign() < 0) {
		httpx.WriteError(w, r, httpx.BadRequest("unitCost must be a non-negative decimal string"))
		return
	}
	var whID *uuid.UUID
	if req.WarehouseID != nil && strings.TrimSpace(*req.WarehouseID) != "" {
		u, e := uuid.Parse(*req.WarehouseID)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid warehouseId"))
			return
		}
		whID = &u
	}
	out, err := s.store.AddPartAndEmit(r.Context(), scope, mntdb.CreateWorkOrderPartParams{
		OrganizationID: scope[0], WorkOrderID: woID, ItemRef: itemRef,
		Quantity: qty, UnitCost: uc, CreatedBy: user,
	}, whID, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) listChecklist(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	woID, err := httpx.ParseUUIDParam("woId", r.PathValue("woId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListChecklist(r.Context(), scope, woID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]checklistDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toChecklistDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) addChecklist(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	woID, err := httpx.ParseUUIDParam("woId", r.PathValue("woId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		StepNo      int32  `json:"stepNo"`
		Description string `json:"description"`
		IsDone      bool   `json:"isDone"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	if req.StepNo <= 0 || strings.TrimSpace(req.Description) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("stepNo > 0 and description are required"))
		return
	}
	out, err := s.store.AddChecklist(r.Context(), scope, mntdb.CreateWorkOrderChecklistParams{
		OrganizationID: scope[0], WorkOrderID: woID, StepNo: req.StepNo,
		Description: strings.TrimSpace(req.Description), IsDone: req.IsDone, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) updateChecklist(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
		Description string `json:"description"`
		IsDone      bool   `json:"isDone"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Description) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("description is required"))
		return
	}
	next, err := s.store.UpdateChecklist(r.Context(), scope, mntdb.UpdateWorkOrderChecklistParams{
		ID: id, Description: strings.TrimSpace(req.Description), IsDone: req.IsDone, UpdatedBy: &user, RowVersion: ver,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "rowVersion": next})
}

func (s *Server) listCosts(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	woID, err := httpx.ParseUUIDParam("woId", r.PathValue("woId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListCosts(r.Context(), scope, woID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]costDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toCostDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) addCost(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	woID, err := httpx.ParseUUIDParam("woId", r.PathValue("woId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		CostType string `json:"costType"`
		Amount   string `json:"amount"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	switch req.CostType {
	case "Labor", "Part", "Vendor":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("costType must be Labor, Part, or Vendor"))
		return
	}
	amt, err := parsePositiveDecimal(req.Amount, "amount")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := s.store.AddCost(r.Context(), scope, mntdb.CreateMaintenanceCostParams{
		OrganizationID: scope[0], WorkOrderID: woID, CostType: req.CostType,
		Amount: amt, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}
