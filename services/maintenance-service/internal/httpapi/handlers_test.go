package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/maintenance-service/internal/mntdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type fakeMaintenanceStore struct {
	partRequested bool
	completed     bool
	createCalls   int
	schedules     []mntdb.MntMaintenanceSchedule
	parts         []mntdb.MntWorkOrderPart
}

func (f *fakeMaintenanceStore) CreateSchedule(context.Context, []uuid.UUID, mntdb.CreateMaintenanceScheduleParams) (mntdb.CreateMaintenanceScheduleRow, error) {
	f.createCalls++
	return mntdb.CreateMaintenanceScheduleRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeMaintenanceStore) ListSchedules(context.Context, []uuid.UUID, int32, int32) ([]mntdb.MntMaintenanceSchedule, int64, error) {
	return f.schedules, int64(len(f.schedules)), nil
}
func (f *fakeMaintenanceStore) UpdateSchedule(context.Context, []uuid.UUID, mntdb.UpdateMaintenanceScheduleParams) (int32, error) {
	return 2, nil
}
func (f *fakeMaintenanceStore) DeleteSchedule(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) error {
	return nil
}
func (f *fakeMaintenanceStore) CreateWorkOrder(context.Context, []uuid.UUID, mntdb.CreateWorkOrderParams) (mntdb.CreateWorkOrderRow, error) {
	return mntdb.CreateWorkOrderRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeMaintenanceStore) GetWorkOrder(context.Context, []uuid.UUID, uuid.UUID) (mntdb.MntWorkOrder, error) {
	return mntdb.MntWorkOrder{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeMaintenanceStore) ListWorkOrders(context.Context, []uuid.UUID, *string, int32, int32) ([]mntdb.MntWorkOrder, int64, error) {
	return nil, 0, nil
}
func (f *fakeMaintenanceStore) AssignWorkOrder(context.Context, []uuid.UUID, mntdb.AssignWorkOrderParams) (int32, error) {
	return 2, nil
}
func (f *fakeMaintenanceStore) StartWorkOrder(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) (int32, error) {
	return 3, nil
}
func (f *fakeMaintenanceStore) CompleteWorkOrderAndEmit(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, int32) (int32, string, error) {
	f.completed = true
	return 4, "250000.0000", nil
}
func (f *fakeMaintenanceStore) CloseWorkOrder(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) (int32, error) {
	return 5, nil
}
func (f *fakeMaintenanceStore) AddPartAndEmit(context.Context, []uuid.UUID, mntdb.CreateWorkOrderPartParams, *uuid.UUID, uuid.UUID) (mntdb.CreateWorkOrderPartRow, error) {
	f.partRequested = true
	return mntdb.CreateWorkOrderPartRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeMaintenanceStore) ListParts(context.Context, []uuid.UUID, uuid.UUID) ([]mntdb.MntWorkOrderPart, error) {
	return f.parts, nil
}
func (f *fakeMaintenanceStore) AddChecklist(context.Context, []uuid.UUID, mntdb.CreateWorkOrderChecklistParams) (mntdb.CreateWorkOrderChecklistRow, error) {
	return mntdb.CreateWorkOrderChecklistRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeMaintenanceStore) ListChecklist(context.Context, []uuid.UUID, uuid.UUID) ([]mntdb.MntWorkOrderChecklist, error) {
	return nil, nil
}
func (f *fakeMaintenanceStore) UpdateChecklist(context.Context, []uuid.UUID, mntdb.UpdateWorkOrderChecklistParams) (int32, error) {
	return 2, nil
}
func (f *fakeMaintenanceStore) AddCost(context.Context, []uuid.UUID, mntdb.CreateMaintenanceCostParams) (mntdb.CreateMaintenanceCostRow, error) {
	return mntdb.CreateMaintenanceCostRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeMaintenanceStore) ListCosts(context.Context, []uuid.UUID, uuid.UUID) ([]mntdb.MntMaintenanceCost, error) {
	return nil, nil
}

func newTestMux(store MaintenanceStore) http.Handler {
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	return httpx.Middleware(mux)
}

func authHeaders(r *http.Request, perms string) {
	org := uuid.NewString()
	r.Header.Set("X-Organization-Id", org)
	r.Header.Set("X-User-Id", uuid.NewString())
	r.Header.Set("X-Permissions", perms)
	r.Header.Set("Content-Type", "application/json")
}

func TestMaintenancePartAndCompleteFlow(t *testing.T) {
	store := &fakeMaintenanceStore{}
	h := newTestMux(store)
	woID := uuid.NewString()

	req := httptest.NewRequest(http.MethodPost, "/maintenance-work-orders/"+woID+"/parts", strings.NewReader(`{"itemRef":"`+uuid.NewString()+`","quantity":"2","unitCost":"50000"}`))
	authHeaders(req, "maintenance:write")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || !store.partRequested {
		t.Fatalf("expected 201 and partRequested=true, got %d: %s", rec.Code, rec.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/maintenance-work-orders/"+woID+"/complete", nil)
	authHeaders(req2, "maintenance:write")
	req2.Header.Set("If-Match", `W/"3"`)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK || !store.completed {
		t.Fatalf("expected 200 and completed=true, got %d: %s", rec2.Code, rec2.Body.String())
	}
}

func TestMaintenanceDTOWireFormatAndIdempotency(t *testing.T) {
	nd, _ := platform.ParseDate("2026-07-01")
	qty, _ := platform.ParseNumeric("3")
	uc, _ := platform.ParseNumeric("75000.50")
	store := &fakeMaintenanceStore{
		schedules: []mntdb.MntMaintenanceSchedule{{
			ID: uuid.New(), OrganizationID: platform.SystemOrgID, AssetRef: uuid.New(),
			ScheduleType: "TimeBased", IntervalValue: 30, IntervalUnit: "Day", NextDueDate: nd, IsActive: true, RowVersion: 1,
		}},
		parts: []mntdb.MntWorkOrderPart{{
			ID: uuid.New(), WorkOrderID: uuid.New(), ItemRef: uuid.New(), Quantity: qty, UnitCost: uc, RowVersion: 1,
		}},
	}
	h := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/maintenance-schedules", nil)
	authHeaders(req, "maintenance:read")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var env struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if len(env.Data) != 1 || env.Data[0]["nextDueDate"] != "2026-07-01" || env.Data[0]["isSystem"] != true {
		t.Fatalf("unexpected schedule DTO: %v", env.Data)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/maintenance-work-orders/"+uuid.NewString()+"/parts", nil)
	authHeaders(req2, "maintenance:read")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK || !strings.Contains(rec2.Body.String(), `"unitCost":"75000.5000"`) {
		t.Fatalf("expected formatted unitCost, got %d: %s", rec2.Code, rec2.Body.String())
	}

	org := uuid.NewString()
	user := uuid.NewString()
	body := `{"assetRef":"` + uuid.NewString() + `","scheduleType":"TimeBased","intervalValue":30,"intervalUnit":"Day"}`
	for i := 0; i < 2; i++ {
		r := httptest.NewRequest(http.MethodPost, "/maintenance-schedules", strings.NewReader(body))
		r.Header.Set("X-Organization-Id", org)
		r.Header.Set("X-User-Id", user)
		r.Header.Set("X-Permissions", "maintenance:write")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "idem-mnt-1")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("call %d expected 201, got %d: %s", i, w.Code, w.Body.String())
		}
	}
	if store.createCalls != 1 {
		t.Fatalf("expected 1 create call under idempotency replay, got %d", store.createCalls)
	}
}
