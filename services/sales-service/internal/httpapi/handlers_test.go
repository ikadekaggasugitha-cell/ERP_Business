package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/sales-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/sales-service/internal/salesdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type memSalesStore struct {
	orders     map[uuid.UUID]repository.SalesOrderDetail
	deliveries map[uuid.UUID]repository.DeliveryDetail
	confirmWH  *uuid.UUID
}

func newMemSalesStore() *memSalesStore {
	return &memSalesStore{
		orders:     make(map[uuid.UUID]repository.SalesOrderDetail),
		deliveries: make(map[uuid.UUID]repository.DeliveryDetail),
	}
}

func (m *memSalesStore) CreateWithLines(_ context.Context, _ []uuid.UUID, in repository.CreateSOInput) (repository.SalesOrderDetail, error) {
	id := uuid.New()
	lines := make([]salesdb.ListSalesOrderLinesRow, 0, len(in.Lines))
	zeroQty, _ := platform.ParseNumeric("0.0000")
	for _, l := range in.Lines {
		lines = append(lines, salesdb.ListSalesOrderLinesRow{
			ID:             uuid.New(),
			OrganizationID: in.OrganizationID,
			SalesOrderID:   id,
			LineNo:         l.LineNo,
			ItemID:         l.ItemID,
			OrderQty:       l.OrderQty,
			UnitPrice:      l.UnitPrice,
			DeliveredQty:   zeroQty,
			RowVersion:     1,
		})
	}
	d := repository.SalesOrderDetail{
		Header: salesdb.GetSalesOrderRow{
			ID:             id,
			OrganizationID: in.OrganizationID,
			SoNo:           in.SoNo,
			CustomerID:     in.CustomerID,
			OrderDate:      in.OrderDate,
			CurrencyCode:   in.CurrencyCode,
			TotalAmount:    in.TotalAmount,
			Status:         "Draft",
			CorrelationID:  in.CorrelationID,
			RowVersion:     1,
		},
		Lines: lines,
	}
	m.orders[id] = d
	return d, nil
}

func (m *memSalesStore) GetByID(_ context.Context, _ []uuid.UUID, id uuid.UUID) (repository.SalesOrderDetail, error) {
	d, ok := m.orders[id]
	if !ok {
		return repository.SalesOrderDetail{}, httpx.NotFound("sales order tidak ditemukan")
	}
	return d, nil
}

func (m *memSalesStore) ListWithCount(_ context.Context, _ []uuid.UUID, status *string, _, _ int32) ([]salesdb.ListSalesOrdersRow, int64, error) {
	var out []salesdb.ListSalesOrdersRow
	for _, d := range m.orders {
		if status != nil && d.Header.Status != *status {
			continue
		}
		out = append(out, salesdb.ListSalesOrdersRow{
			ID:             d.Header.ID,
			OrganizationID: d.Header.OrganizationID,
			SoNo:           d.Header.SoNo,
			CustomerID:     d.Header.CustomerID,
			OrderDate:      d.Header.OrderDate,
			CurrencyCode:   d.Header.CurrencyCode,
			TotalAmount:    d.Header.TotalAmount,
			Status:         d.Header.Status,
			RowVersion:     d.Header.RowVersion,
		})
	}
	return out, int64(len(out)), nil
}

func (m *memSalesStore) UpdateDraftWithLines(_ context.Context, _ []uuid.UUID, in repository.UpdateDraftSOInput) (repository.SalesOrderDetail, error) {
	d, ok := m.orders[in.ID]
	if !ok {
		return repository.SalesOrderDetail{}, httpx.NotFound("sales order tidak ditemukan")
	}
	if d.Header.Status != "Draft" || d.Header.RowVersion != in.RowVersion {
		return repository.SalesOrderDetail{}, httpx.Conflict("VERSION_CONFLICT", "conflict")
	}
	d.Header.CustomerID = in.CustomerID
	d.Header.OrderDate = in.OrderDate
	d.Header.CurrencyCode = in.CurrencyCode
	d.Header.TotalAmount = in.TotalAmount
	d.Header.RowVersion++
	d.Lines = nil
	zeroQty, _ := platform.ParseNumeric("0.0000")
	for _, l := range in.Lines {
		d.Lines = append(d.Lines, salesdb.ListSalesOrderLinesRow{
			ID:             uuid.New(),
			OrganizationID: d.Header.OrganizationID,
			SalesOrderID:   d.Header.ID,
			LineNo:         l.LineNo,
			ItemID:         l.ItemID,
			OrderQty:       l.OrderQty,
			UnitPrice:      l.UnitPrice,
			DeliveredQty:   zeroQty,
			RowVersion:     1,
		})
	}
	m.orders[in.ID] = d
	return d, nil
}

func (m *memSalesStore) DeleteDraft(_ context.Context, _ []uuid.UUID, id uuid.UUID, version int32, _ uuid.UUID) error {
	d, ok := m.orders[id]
	if !ok {
		return httpx.NotFound("sales order tidak ditemukan")
	}
	if d.Header.Status != "Draft" || d.Header.RowVersion != version {
		return httpx.Conflict("VERSION_CONFLICT", "conflict")
	}
	delete(m.orders, id)
	return nil
}

func (m *memSalesStore) ConfirmAndEmit(_ context.Context, _ []uuid.UUID, id uuid.UUID, version int32, whID *uuid.UUID, _, _ uuid.UUID) (repository.SalesOrderDetail, error) {
	d, ok := m.orders[id]
	if !ok {
		return repository.SalesOrderDetail{}, httpx.NotFound("sales order tidak ditemukan")
	}
	if d.Header.Status != "Draft" || d.Header.RowVersion != version {
		return repository.SalesOrderDetail{}, httpx.Conflict("VERSION_CONFLICT", "conflict")
	}
	m.confirmWH = whID
	d.Header.Status = "Confirmed"
	d.Header.RowVersion++
	m.orders[id] = d
	return d, nil
}

func (m *memSalesStore) CreateDelivery(_ context.Context, _ []uuid.UUID, in repository.CreateDeliveryInput) (repository.DeliveryDetail, error) {
	so, ok := m.orders[in.SalesOrderID]
	if !ok {
		return repository.DeliveryDetail{}, httpx.NotFound("sales order tidak ditemukan")
	}
	if so.Header.Status != "Confirmed" || so.Header.RowVersion != in.RowVersion {
		return repository.DeliveryDetail{}, httpx.Conflict("INVALID_STATE", "conflict")
	}
	so.Header.Status = "Delivered"
	so.Header.RowVersion++
	m.orders[in.SalesOrderID] = so

	delID := uuid.New()
	delNo := in.DeliveryNo
	if delNo == "" {
		delNo = "DO-" + so.Header.SoNo
	}
	whID := in.WarehouseID
	if whID == uuid.Nil {
		whID = so.Header.OrganizationID
	}
	delDate := in.DeliveryDate
	if !delDate.Valid {
		delDate = pgtype.Date{Time: time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC), Valid: true}
	}
	dLines := make([]salesdb.ListDeliveryLinesRow, 0, len(so.Lines))
	for _, l := range so.Lines {
		dLines = append(dLines, salesdb.ListDeliveryLinesRow{
			ID:             uuid.New(),
			OrganizationID: so.Header.OrganizationID,
			DeliveryID:     delID,
			SoLineID:       l.ID,
			LineNo:         l.LineNo,
			ItemID:         l.ItemID,
			DeliveredQty:   l.OrderQty,
			RowVersion:     1,
		})
	}
	del := repository.DeliveryDetail{
		Header: salesdb.GetDeliveryRow{
			ID:             delID,
			OrganizationID: so.Header.OrganizationID,
			DeliveryNo:     delNo,
			SalesOrderID:   in.SalesOrderID,
			WarehouseID:    whID,
			DeliveryDate:   delDate,
			Status:         "Delivered",
			CorrelationID:  in.CorrelationID,
			RowVersion:     1,
		},
		Lines: dLines,
	}
	m.deliveries[delID] = del
	return del, nil
}

func (m *memSalesStore) GetDelivery(_ context.Context, _ []uuid.UUID, id uuid.UUID) (repository.DeliveryDetail, error) {
	d, ok := m.deliveries[id]
	if !ok {
		return repository.DeliveryDetail{}, httpx.NotFound("delivery tidak ditemukan")
	}
	return d, nil
}

func (m *memSalesStore) ListDeliveriesWithCount(_ context.Context, _ []uuid.UUID, _, _ int32) ([]salesdb.ListDeliveriesRow, int64, error) {
	var out []salesdb.ListDeliveriesRow
	for _, d := range m.deliveries {
		out = append(out, salesdb.ListDeliveriesRow{
			ID:             d.Header.ID,
			OrganizationID: d.Header.OrganizationID,
			DeliveryNo:     d.Header.DeliveryNo,
			SalesOrderID:   d.Header.SalesOrderID,
			WarehouseID:    d.Header.WarehouseID,
			DeliveryDate:   d.Header.DeliveryDate,
			Status:         d.Header.Status,
			RowVersion:     d.Header.RowVersion,
		})
	}
	return out, int64(len(out)), nil
}

func doSalesReq(t *testing.T, h http.Handler, method, path, body, perm, ifMatch string) *httptest.ResponseRecorder {
	t.Helper()
	var buf *bytes.Buffer
	if body != "" {
		buf = bytes.NewBufferString(body)
	} else {
		buf = &bytes.Buffer{}
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("X-Organization-Id", uuid.NewString())
	req.Header.Set("X-User-Id", uuid.NewString())
	if perm != "" {
		req.Header.Set("X-Permissions", perm)
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSalesOrderLifecycleValidationRBACAndDeliveries(t *testing.T) {
	store := newMemSalesStore()
	router := NewRouter(store, nil)

	custID := uuid.NewString()
	itemID := uuid.NewString()

	// 1. RBAC rejection
	rec := doSalesReq(t, router, http.MethodPost, "/sales-orders", `{"soNo":"SO-1"}`, "sales:read", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for sales:read on POST, got %d", rec.Code)
	}

	// 2. Line total mismatch -> 422 TOTAL_MISMATCH
	badBody := `{"soNo":"SO-BAD","customerId":"` + custID + `","orderDate":"2026-03-15","currencyCode":"IDR","totalAmount":"999.0000","lines":[{"lineNo":1,"itemId":"` + itemID + `","orderQty":"4","unitPrice":"300.0000"}]}`
	rec = doSalesReq(t, router, http.MethodPost, "/sales-orders", badBody, "sales:write", "")
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "TOTAL_MISMATCH") {
		t.Fatalf("expected 422 TOTAL_MISMATCH, got %d %s", rec.Code, rec.Body.String())
	}

	// 3. Create Draft SO -> 201
	goodBody := `{"soNo":"SO-100","customerId":"` + custID + `","orderDate":"2026-03-15","currencyCode":"IDR","totalAmount":"1200.0000","lines":[{"lineNo":1,"itemId":"` + itemID + `","orderQty":"4","unitPrice":"300.0000"}]}`
	rec = doSalesReq(t, router, http.MethodPost, "/sales-orders", goodBody, "sales:write", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d %s", rec.Code, rec.Body.String())
	}
	var created soDetailResp
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.Status != "Draft" || created.RowVersion != 1 || len(created.Lines) != 1 {
		t.Fatalf("unexpected created SO: %+v", created)
	}

	// 4. Update Draft SO -> 200
	updBody := `{"customerId":"` + custID + `","orderDate":"2026-03-15","currencyCode":"IDR","totalAmount":"1500.0000","lines":[{"lineNo":1,"itemId":"` + itemID + `","orderQty":"5","unitPrice":"300.0000"}]}`
	rec = doSalesReq(t, router, http.MethodPut, "/sales-orders/"+created.ID, updBody, "sales:write", `"1"`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 update draft, got %d %s", rec.Code, rec.Body.String())
	}

	// 5. Confirm SO with empty body -> 200
	rec = doSalesReq(t, router, http.MethodPost, "/sales-orders/"+created.ID+"/confirm", "", "sales:write", `"2"`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 confirm empty body, got %d %s", rec.Code, rec.Body.String())
	}
	var confirmed soDetailResp
	_ = json.Unmarshal(rec.Body.Bytes(), &confirmed)
	if confirmed.Status != "Confirmed" || confirmed.RowVersion != 3 {
		t.Fatalf("expected Confirmed v3, got %+v", confirmed)
	}

	// 6. Create Delivery -> 201 + GET /deliveries -> 200
	rec = doSalesReq(t, router, http.MethodPost, "/sales-orders/"+created.ID+"/deliveries", "", "sales:write", `"3"`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 delivery, got %d %s", rec.Code, rec.Body.String())
	}
	var del deliveryResp
	_ = json.Unmarshal(rec.Body.Bytes(), &del)
	if del.DeliveryNo != "DO-SO-100" || del.Status != "Delivered" || len(del.Lines) != 1 {
		t.Fatalf("unexpected delivery: %+v", del)
	}

	rec = doSalesReq(t, router, http.MethodGet, "/deliveries", "", "sales:read", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 list deliveries, got %d", rec.Code)
	}

	// 7. Create & Delete another Draft SO
	draft2Body := `{"soNo":"SO-101","customerId":"` + custID + `","orderDate":"2026-03-15","currencyCode":"IDR","totalAmount":"300.0000","lines":[{"lineNo":1,"itemId":"` + itemID + `","orderQty":"1","unitPrice":"300.0000"}]}`
	rec = doSalesReq(t, router, http.MethodPost, "/sales-orders", draft2Body, "sales:write", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 draft2, got %d", rec.Code)
	}
	var draft2 soDetailResp
	_ = json.Unmarshal(rec.Body.Bytes(), &draft2)
	rec = doSalesReq(t, router, http.MethodDelete, "/sales-orders/"+draft2.ID, "", "sales:write", `"1"`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 delete draft, got %d %s", rec.Code, rec.Body.String())
	}
}

