package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/manufacturing-service/internal/manufacturingdb"
	"github.com/sinergiinvestasiproperti/erp/services/manufacturing-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type fakeMfgStore struct {
	boms     map[uuid.UUID]manufacturingdb.GetBillOfMaterialRow
	bomLines map[uuid.UUID][]manufacturingdb.ListBomLinesRow
	wos      map[uuid.UUID]manufacturingdb.GetWorkOrderRow
	outbox   []manufacturingdb.InsertOutboxParams
}

func newFakeMfgStore() *fakeMfgStore {
	return &fakeMfgStore{
		boms:     make(map[uuid.UUID]manufacturingdb.GetBillOfMaterialRow),
		bomLines: make(map[uuid.UUID][]manufacturingdb.ListBomLinesRow),
		wos:      make(map[uuid.UUID]manufacturingdb.GetWorkOrderRow),
	}
}

func (f *fakeMfgStore) CreateBOMWithLines(_ context.Context, _ []uuid.UUID, bom manufacturingdb.CreateBillOfMaterialParams, lines []manufacturingdb.CreateBomLineParams) (manufacturingdb.CreateBillOfMaterialRow, error) {
	id := uuid.New()
	f.boms[id] = manufacturingdb.GetBillOfMaterialRow{
		ID: id, OrganizationID: bom.OrganizationID, Code: bom.Code, ProductItemID: bom.ProductItemID,
		Version: bom.Version, IsActive: true, RowVersion: 1,
	}
	stored := make([]manufacturingdb.ListBomLinesRow, 0, len(lines))
	for _, l := range lines {
		stored = append(stored, manufacturingdb.ListBomLinesRow{
			ID: uuid.New(), OrganizationID: l.OrganizationID, BomID: id, LineNo: l.LineNo,
			ComponentItemID: l.ComponentItemID, Quantity: l.Quantity, Uom: l.Uom, RowVersion: 1,
		})
	}
	f.bomLines[id] = stored
	return manufacturingdb.CreateBillOfMaterialRow{
		ID: id, Code: bom.Code, ProductItemID: bom.ProductItemID, Version: bom.Version, IsActive: true, RowVersion: 1,
	}, nil
}

func (f *fakeMfgStore) GetBOM(_ context.Context, _ []uuid.UUID, id uuid.UUID) (repository.BOMDetail, error) {
	h, ok := f.boms[id]
	if !ok {
		return repository.BOMDetail{}, pgx.ErrNoRows
	}
	return repository.BOMDetail{Header: h, Lines: f.bomLines[id]}, nil
}

func (f *fakeMfgStore) ListBOMsWithCount(_ context.Context, _ []uuid.UUID, _ manufacturingdb.ListBillsOfMaterialParams) ([]manufacturingdb.ListBillsOfMaterialRow, int64, error) {
	out := make([]manufacturingdb.ListBillsOfMaterialRow, 0, len(f.boms))
	for _, b := range f.boms {
		out = append(out, manufacturingdb.ListBillsOfMaterialRow{
			ID: b.ID, OrganizationID: b.OrganizationID, Code: b.Code, ProductItemID: b.ProductItemID,
			Version: b.Version, IsActive: b.IsActive, RowVersion: b.RowVersion,
		})
	}
	return out, int64(len(out)), nil
}

func (f *fakeMfgStore) CreateWorkOrder(_ context.Context, _ []uuid.UUID, arg manufacturingdb.CreateWorkOrderParams) (manufacturingdb.CreateWorkOrderRow, error) {
	id := uuid.New()
	f.wos[id] = manufacturingdb.GetWorkOrderRow{
		ID: id, OrganizationID: arg.OrganizationID, WoNo: arg.WoNo, ProductItemID: arg.ProductItemID,
		BomID: arg.BomID, PlannedQty: arg.PlannedQty, Status: "Planned", SalesOrderRef: arg.SalesOrderRef,
		CorrelationID: arg.CorrelationID, RowVersion: 1,
	}
	return manufacturingdb.CreateWorkOrderRow{ID: id, Status: "Planned", RowVersion: 1}, nil
}

func (f *fakeMfgStore) GetByID(_ context.Context, _ []uuid.UUID, id uuid.UUID) (manufacturingdb.GetWorkOrderRow, error) {
	w, ok := f.wos[id]
	if !ok {
		return manufacturingdb.GetWorkOrderRow{}, pgx.ErrNoRows
	}
	return w, nil
}

func (f *fakeMfgStore) ListWithCount(_ context.Context, _ []uuid.UUID, arg manufacturingdb.ListWorkOrdersParams) ([]manufacturingdb.ListWorkOrdersRow, int64, error) {
	out := make([]manufacturingdb.ListWorkOrdersRow, 0, len(f.wos))
	for _, w := range f.wos {
		if arg.Status != nil && w.Status != *arg.Status {
			continue
		}
		out = append(out, manufacturingdb.ListWorkOrdersRow{
			ID: w.ID, OrganizationID: w.OrganizationID, WoNo: w.WoNo, ProductItemID: w.ProductItemID,
			BomID: w.BomID, PlannedQty: w.PlannedQty, ProducedQty: w.ProducedQty, Status: w.Status,
			SalesOrderRef: w.SalesOrderRef, RowVersion: w.RowVersion,
		})
	}
	return out, int64(len(out)), nil
}

func (f *fakeMfgStore) UpdatePlanned(_ context.Context, _ []uuid.UUID, arg manufacturingdb.UpdatePlannedWorkOrderParams) (int32, int64, error) {
	w, ok := f.wos[arg.ID]
	if !ok || w.Status != "Planned" || w.RowVersion != arg.RowVersion {
		return 0, 0, nil
	}
	w.ProductItemID = arg.ProductItemID
	w.BomID = arg.BomID
	w.PlannedQty = arg.PlannedQty
	w.SalesOrderRef = arg.SalesOrderRef
	w.RowVersion++
	f.wos[arg.ID] = w
	return w.RowVersion, 1, nil
}

func (f *fakeMfgStore) DeletePlanned(_ context.Context, _ []uuid.UUID, arg manufacturingdb.SoftDeletePlannedWorkOrderParams) (int64, error) {
	w, ok := f.wos[arg.ID]
	if !ok || w.Status != "Planned" || w.RowVersion != arg.RowVersion {
		return 0, nil
	}
	delete(f.wos, arg.ID)
	return 1, nil
}

func (f *fakeMfgStore) ReleaseAndEmit(_ context.Context, _ []uuid.UUID, arg manufacturingdb.ReleaseWorkOrderParams, ev manufacturingdb.InsertOutboxParams) (int32, int64, error) {
	w, ok := f.wos[arg.ID]
	if !ok || w.Status != "Planned" || w.RowVersion != arg.RowVersion {
		return 0, 0, nil
	}
	w.Status = "Released"
	w.RowVersion++
	f.wos[arg.ID] = w
	f.outbox = append(f.outbox, ev)
	return w.RowVersion, 1, nil
}

func (f *fakeMfgStore) CompleteAndEmit(_ context.Context, _ []uuid.UUID, arg manufacturingdb.CompleteWorkOrderParams, ev manufacturingdb.InsertOutboxParams) (int32, int64, error) {
	w, ok := f.wos[arg.ID]
	if !ok || (w.Status != "Released" && w.Status != "InProgress") || w.RowVersion != arg.RowVersion {
		return 0, 0, nil
	}
	w.Status = "Completed"
	w.ProducedQty = arg.ProducedQty
	w.RowVersion++
	f.wos[arg.ID] = w
	f.outbox = append(f.outbox, ev)
	return w.RowVersion, 1, nil
}

func doMfgReq(t *testing.T, handler http.Handler, method, path, orgID, userID, perms string, headers map[string]string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("X-Organization-Id", orgID)
	req.Header.Set("X-User-Id", userID)
	if perms != "" {
		req.Header.Set("X-Permissions", perms)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestManufacturingBOMWorkOrderReleaseCompleteAndRBAC(t *testing.T) {
	store := newFakeMfgStore()
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	handler := httpx.Middleware(mux)

	orgID := uuid.NewString()
	userID := uuid.NewString()
	prodID := uuid.NewString()
	compID := uuid.NewString()
	readPerm := "manufacturing:read"
	rwPerms := "manufacturing:read,manufacturing:write"

	// 1. RBAC rejection
	rec := doMfgReq(t, handler, http.MethodPost, "/bill-of-materials", orgID, userID, readPerm, nil, map[string]any{"code": "B1", "productItemId": prodID})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	// 2. Create BOM with line
	bomBody := map[string]any{
		"code": "BOM-01", "productItemId": prodID,
		"lines": []map[string]any{{"lineNo": 1, "componentItemId": compID, "quantity": "2.5000", "uom": "PCS"}},
	}
	rec = doMfgReq(t, handler, http.MethodPost, "/bill-of-materials", orgID, userID, rwPerms, nil, bomBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create BOM expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var bom bomDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &bom)

	// 3. Create Work Order -> PUT -> Release (emits released) -> Complete (emits completed)
	woBody := map[string]any{"woNo": "WO-001", "productItemId": prodID, "bomId": bom.ID, "plannedQty": "10"}
	rec = doMfgReq(t, handler, http.MethodPost, "/work-orders", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "k-wo1"}, woBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create WO expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var wo woDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &wo)

	rec = doMfgReq(t, handler, http.MethodPut, "/work-orders/"+wo.ID, orgID, userID, rwPerms, map[string]string{"If-Match": "1"}, map[string]any{
		"productItemId": prodID, "bomId": bom.ID, "plannedQty": "12",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update WO expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Complete before Release returns 409
	rec = doMfgReq(t, handler, http.MethodPost, "/work-orders/"+wo.ID+"/complete", orgID, userID, rwPerms, map[string]string{"If-Match": "2"}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("complete before release expected 409, got %d", rec.Code)
	}

	// Release (v2->v3) -> Complete (v3->v4)
	rec = doMfgReq(t, handler, http.MethodPost, "/work-orders/"+wo.ID+"/release", orgID, userID, rwPerms, map[string]string{"If-Match": "2"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("release expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	rec = doMfgReq(t, handler, http.MethodPost, "/work-orders/"+wo.ID+"/complete", orgID, userID, rwPerms, map[string]string{"If-Match": "3"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("complete expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if len(store.outbox) != 2 || store.outbox[0].EventType != "manufacturing.workorder.released" || store.outbox[1].EventType != "manufacturing.workorder.completed" {
		t.Fatalf("unexpected outbox events: %+v", store.outbox)
	}

	// 4. Delete Planned WO
	rec = doMfgReq(t, handler, http.MethodPost, "/work-orders", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "k-wo2"}, map[string]any{
		"woNo": "WO-002", "productItemId": prodID, "bomId": bom.ID, "plannedQty": "5",
	})
	var wo2 woDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &wo2)
	rec = doMfgReq(t, handler, http.MethodDelete, "/work-orders/"+wo2.ID, orgID, userID, rwPerms, map[string]string{"If-Match": "1"}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete WO expected 204, got %d", rec.Code)
	}
}

