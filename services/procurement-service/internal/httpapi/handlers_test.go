package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/procurementdb"
	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type fakePOStore struct {
	pos     map[uuid.UUID]procurementdb.GetPurchaseOrderRow
	poLines map[uuid.UUID][]procurementdb.ListPurchaseOrderLinesRow
	grs     map[uuid.UUID]procurementdb.GetGoodsReceiptRow
	grLines map[uuid.UUID][]procurementdb.ListGoodsReceiptLinesRow
	prs     map[uuid.UUID]procurementdb.GetPurchaseRequisitionRow
	prLines map[uuid.UUID][]procurementdb.ListPurchaseRequisitionLinesRow
	outbox  []procurementdb.InsertOutboxParams
}

func newFakePOStore() *fakePOStore {
	return &fakePOStore{
		pos:     make(map[uuid.UUID]procurementdb.GetPurchaseOrderRow),
		poLines: make(map[uuid.UUID][]procurementdb.ListPurchaseOrderLinesRow),
		grs:     make(map[uuid.UUID]procurementdb.GetGoodsReceiptRow),
		grLines: make(map[uuid.UUID][]procurementdb.ListGoodsReceiptLinesRow),
		prs:     make(map[uuid.UUID]procurementdb.GetPurchaseRequisitionRow),
		prLines: make(map[uuid.UUID][]procurementdb.ListPurchaseRequisitionLinesRow),
	}
}

func (f *fakePOStore) CreateWithLines(_ context.Context, _ []uuid.UUID, po procurementdb.CreatePurchaseOrderParams, lines []procurementdb.CreatePurchaseOrderLineParams) (procurementdb.CreatePurchaseOrderRow, error) {
	id := uuid.New()
	f.pos[id] = procurementdb.GetPurchaseOrderRow{
		ID:                    id,
		OrganizationID:        po.OrganizationID,
		PoNo:                  po.PoNo,
		VendorID:              po.VendorID,
		PurchaseRequisitionID: po.PurchaseRequisitionID,
		OrderDate:             po.OrderDate,
		CurrencyCode:          po.CurrencyCode,
		TotalAmount:           po.TotalAmount,
		Status:                "Draft",
		CreatedBy:             po.CreatedBy,
		RowVersion:            1,
	}
	stored := make([]procurementdb.ListPurchaseOrderLinesRow, 0, len(lines))
	for _, l := range lines {
		stored = append(stored, procurementdb.ListPurchaseOrderLinesRow{
			ID:              uuid.New(),
			OrganizationID:  l.OrganizationID,
			PurchaseOrderID: id,
			LineNo:          l.LineNo,
			ItemID:          l.ItemID,
			OrderQty:        l.OrderQty,
			UnitPrice:       l.UnitPrice,
			RowVersion:      1,
		})
	}
	f.poLines[id] = stored
	return procurementdb.CreatePurchaseOrderRow{ID: id, Status: "Draft", RowVersion: 1}, nil
}

func (f *fakePOStore) GetByID(_ context.Context, _ []uuid.UUID, id uuid.UUID) (repository.PurchaseOrderDetail, error) {
	h, ok := f.pos[id]
	if !ok {
		return repository.PurchaseOrderDetail{}, pgx.ErrNoRows
	}
	return repository.PurchaseOrderDetail{Header: h, Lines: f.poLines[id]}, nil
}

func (f *fakePOStore) ListWithCount(_ context.Context, _ []uuid.UUID, arg procurementdb.ListPurchaseOrdersParams) ([]procurementdb.ListPurchaseOrdersRow, int64, error) {
	out := make([]procurementdb.ListPurchaseOrdersRow, 0, len(f.pos))
	for _, h := range f.pos {
		if arg.Status != nil && h.Status != *arg.Status {
			continue
		}
		out = append(out, procurementdb.ListPurchaseOrdersRow{
			ID:                    h.ID,
			OrganizationID:        h.OrganizationID,
			PoNo:                  h.PoNo,
			VendorID:              h.VendorID,
			PurchaseRequisitionID: h.PurchaseRequisitionID,
			OrderDate:             h.OrderDate,
			CurrencyCode:          h.CurrencyCode,
			TotalAmount:           h.TotalAmount,
			Status:                h.Status,
			RowVersion:            h.RowVersion,
		})
	}
	return out, int64(len(out)), nil
}

func (f *fakePOStore) UpdateDraftWithLines(_ context.Context, _ []uuid.UUID, arg procurementdb.UpdateDraftPurchaseOrderParams, lines []procurementdb.CreatePurchaseOrderLineParams) (int32, int64, error) {
	h, ok := f.pos[arg.ID]
	if !ok || h.Status != "Draft" || h.RowVersion != arg.RowVersion {
		return 0, 0, nil
	}
	h.VendorID = arg.VendorID
	h.PurchaseRequisitionID = arg.PurchaseRequisitionID
	h.OrderDate = arg.OrderDate
	h.CurrencyCode = arg.CurrencyCode
	h.TotalAmount = arg.TotalAmount
	h.UpdatedBy = arg.UpdatedBy
	h.RowVersion++
	f.pos[arg.ID] = h
	stored := make([]procurementdb.ListPurchaseOrderLinesRow, 0, len(lines))
	for _, l := range lines {
		stored = append(stored, procurementdb.ListPurchaseOrderLinesRow{
			ID:              uuid.New(),
			OrganizationID:  l.OrganizationID,
			PurchaseOrderID: arg.ID,
			LineNo:          l.LineNo,
			ItemID:          l.ItemID,
			OrderQty:        l.OrderQty,
			UnitPrice:       l.UnitPrice,
			RowVersion:      1,
		})
	}
	f.poLines[arg.ID] = stored
	return h.RowVersion, 1, nil
}

func (f *fakePOStore) DeleteDraft(_ context.Context, _ []uuid.UUID, arg procurementdb.SoftDeleteDraftPurchaseOrderParams) (int64, error) {
	h, ok := f.pos[arg.ID]
	if !ok || h.Status != "Draft" || h.RowVersion != arg.RowVersion {
		return 0, nil
	}
	delete(f.pos, arg.ID)
	delete(f.poLines, arg.ID)
	return 1, nil
}

func (f *fakePOStore) ApproveAndEmit(_ context.Context, _ []uuid.UUID, arg procurementdb.ApprovePurchaseOrderParams, ev procurementdb.InsertOutboxParams) (int32, int64, error) {
	h, ok := f.pos[arg.ID]
	if !ok || h.Status != "Draft" || h.RowVersion != arg.RowVersion {
		return 0, 0, nil
	}
	h.Status = "Approved"
	h.UpdatedBy = arg.UpdatedBy
	h.RowVersion++
	f.pos[arg.ID] = h
	f.outbox = append(f.outbox, ev)
	return h.RowVersion, 1, nil
}

func (f *fakePOStore) ReceiveAndEmit(_ context.Context, _ []uuid.UUID, arg procurementdb.ReceivePurchaseOrderParams, grArg procurementdb.CreateGoodsReceiptParams, buildOutbox repository.OutboxPayloadFunc) (int32, procurementdb.CreateGoodsReceiptRow, int64, error) {
	h, ok := f.pos[arg.ID]
	if !ok || h.Status != "Approved" || h.RowVersion != arg.RowVersion {
		return 0, procurementdb.CreateGoodsReceiptRow{}, 0, nil
	}
	h.Status = "Received"
	h.UpdatedBy = arg.UpdatedBy
	h.RowVersion++
	f.pos[arg.ID] = h

	grID := uuid.New()
	grHeader := procurementdb.GetGoodsReceiptRow{
		ID:              grID,
		OrganizationID:  grArg.OrganizationID,
		GrNo:            grArg.GrNo,
		PurchaseOrderID: grArg.PurchaseOrderID,
		WarehouseID:     grArg.WarehouseID,
		ReceiptDate:     grArg.ReceiptDate,
		Status:          grArg.Status,
		CorrelationID:   grArg.CorrelationID,
		RowVersion:      1,
	}
	f.grs[grID] = grHeader
	lines := f.poLines[arg.ID]
	grLines := make([]procurementdb.ListGoodsReceiptLinesRow, 0, len(lines))
	for i := range lines {
		lines[i].ReceivedQty = lines[i].OrderQty
		grLines = append(grLines, procurementdb.ListGoodsReceiptLinesRow{
			ID:             uuid.New(),
			OrganizationID: grArg.OrganizationID,
			GoodsReceiptID: grID,
			PoLineID:       lines[i].ID,
			LineNo:         lines[i].LineNo,
			ReceivedQty:    lines[i].OrderQty,
			RowVersion:     1,
		})
	}
	f.poLines[arg.ID] = lines
	f.grLines[grID] = grLines

	grCreated := procurementdb.CreateGoodsReceiptRow{
		ID:         grID,
		GrNo:       grArg.GrNo,
		Status:     grArg.Status,
		RowVersion: 1,
	}
	ev, err := buildOutbox(lines, grCreated)
	if err != nil {
		return 0, procurementdb.CreateGoodsReceiptRow{}, 0, err
	}
	f.outbox = append(f.outbox, ev)
	return h.RowVersion, grCreated, 1, nil
}

func (f *fakePOStore) GetGoodsReceipt(_ context.Context, _ []uuid.UUID, id uuid.UUID) (repository.GoodsReceiptDetail, error) {
	h, ok := f.grs[id]
	if !ok {
		return repository.GoodsReceiptDetail{}, pgx.ErrNoRows
	}
	return repository.GoodsReceiptDetail{Header: h, Lines: f.grLines[id]}, nil
}

func (f *fakePOStore) ListGoodsReceiptsWithCount(_ context.Context, _ []uuid.UUID, _ procurementdb.ListGoodsReceiptsParams) ([]procurementdb.ListGoodsReceiptsRow, int64, error) {
	out := make([]procurementdb.ListGoodsReceiptsRow, 0, len(f.grs))
	for _, g := range f.grs {
		out = append(out, procurementdb.ListGoodsReceiptsRow{
			ID:              g.ID,
			OrganizationID:  g.OrganizationID,
			GrNo:            g.GrNo,
			PurchaseOrderID: g.PurchaseOrderID,
			WarehouseID:     g.WarehouseID,
			ReceiptDate:     g.ReceiptDate,
			Status:          g.Status,
			CorrelationID:   g.CorrelationID,
			RowVersion:      g.RowVersion,
		})
	}
	return out, int64(len(out)), nil
}

func (f *fakePOStore) CreateRequisitionWithLines(_ context.Context, _ []uuid.UUID, pr procurementdb.CreatePurchaseRequisitionParams, lines []procurementdb.CreatePurchaseRequisitionLineParams) (procurementdb.CreatePurchaseRequisitionRow, error) {
	id := uuid.New()
	f.prs[id] = procurementdb.GetPurchaseRequisitionRow{
		ID:             id,
		OrganizationID: pr.OrganizationID,
		PrNo:           pr.PrNo,
		RequesterID:    pr.RequesterID,
		Status:         "Draft",
		WorkflowRef:    pr.WorkflowRef,
		NeededDate:     pr.NeededDate,
		CreatedBy:      pr.CreatedBy,
		RowVersion:     1,
	}
	stored := make([]procurementdb.ListPurchaseRequisitionLinesRow, 0, len(lines))
	for _, l := range lines {
		stored = append(stored, procurementdb.ListPurchaseRequisitionLinesRow{
			ID:                    uuid.New(),
			OrganizationID:        l.OrganizationID,
			PurchaseRequisitionID: id,
			LineNo:                l.LineNo,
			ItemID:                l.ItemID,
			RequestQty:            l.RequestQty,
			NeededDate:            l.NeededDate,
			RowVersion:            1,
		})
	}
	f.prLines[id] = stored
	return procurementdb.CreatePurchaseRequisitionRow{ID: id, Status: "Draft", RowVersion: 1}, nil
}

func (f *fakePOStore) GetRequisition(_ context.Context, _ []uuid.UUID, id uuid.UUID) (repository.PurchaseRequisitionDetail, error) {
	h, ok := f.prs[id]
	if !ok {
		return repository.PurchaseRequisitionDetail{}, pgx.ErrNoRows
	}
	return repository.PurchaseRequisitionDetail{Header: h, Lines: f.prLines[id]}, nil
}

func (f *fakePOStore) ListRequisitionsWithCount(_ context.Context, _ []uuid.UUID, _ procurementdb.ListPurchaseRequisitionsParams) ([]procurementdb.ListPurchaseRequisitionsRow, int64, error) {
	out := make([]procurementdb.ListPurchaseRequisitionsRow, 0, len(f.prs))
	for _, h := range f.prs {
		out = append(out, procurementdb.ListPurchaseRequisitionsRow{
			ID:             h.ID,
			OrganizationID: h.OrganizationID,
			PrNo:           h.PrNo,
			RequesterID:    h.RequesterID,
			Status:         h.Status,
			WorkflowRef:    h.WorkflowRef,
			NeededDate:     h.NeededDate,
			RowVersion:     h.RowVersion,
		})
	}
	return out, int64(len(out)), nil
}

func (f *fakePOStore) TransitionRequisition(_ context.Context, _ []uuid.UUID, arg procurementdb.TransitionPurchaseRequisitionParams) (procurementdb.TransitionPurchaseRequisitionRow, int64, error) {
	h, ok := f.prs[arg.ID]
	if !ok || h.Status != arg.Status_2 || h.RowVersion != arg.RowVersion {
		return procurementdb.TransitionPurchaseRequisitionRow{}, 0, nil
	}
	h.Status = arg.Status
	h.UpdatedBy = arg.UpdatedBy
	h.RowVersion++
	f.prs[arg.ID] = h
	return procurementdb.TransitionPurchaseRequisitionRow{Status: h.Status, RowVersion: h.RowVersion}, 1, nil
}

func doPOReq(t *testing.T, handler http.Handler, method, path, orgID, userID, perms string, headers map[string]string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
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

func TestProcurementPOLifecycleGRPRAndValidation(t *testing.T) {
	store := newFakePOStore()
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	handler := httpx.Middleware(mux)

	orgID := uuid.NewString()
	userID := uuid.NewString()
	vendorID := uuid.NewString()
	itemID := uuid.NewString()
	readPerm := "procurement:read"
	rwPerms := "procurement:read,procurement:write"

	// 1. RBAC rejection when missing procurement:write
	rec := doPOReq(t, handler, http.MethodPost, "/purchase-orders", orgID, userID, readPerm, nil, map[string]any{})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing procurement:write, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 2. Decimal mismatch rejected (3 * 300 != 900.0001)
	badBody := map[string]any{
		"poNo": "PO-BAD", "vendorId": vendorID, "orderDate": "2026-03-15", "currencyCode": "IDR", "totalAmount": "900.0001",
		"lines": []map[string]any{{"lineNo": 1, "itemId": itemID, "orderQty": "3", "unitPrice": "300.0000"}},
	}
	rec = doPOReq(t, handler, http.MethodPost, "/purchase-orders", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "k-bad"}, badBody)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 TOTAL_MISMATCH, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 3. Create PO with idempotency replay + GET detail + PUT Draft + Approve + Receive (empty body)
	poBody := map[string]any{
		"poNo": "PO-001", "vendorId": vendorID, "orderDate": "2026-03-15", "currencyCode": "IDR", "totalAmount": "900.0000",
		"lines": []map[string]any{{"lineNo": 1, "itemId": itemID, "orderQty": "3", "unitPrice": "300.0000"}},
	}
	rec = doPOReq(t, handler, http.MethodPost, "/purchase-orders", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "k-po1"}, poBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create PO expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var created poDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	// Replay with same Idempotency-Key returns same ID without duplicating
	recReplay := doPOReq(t, handler, http.MethodPost, "/purchase-orders", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "k-po1"}, poBody)
	if recReplay.Code != http.StatusCreated || len(store.pos) != 1 {
		t.Fatalf("idempotency replay failed: code=%d count=%d", recReplay.Code, len(store.pos))
	}

	// Update Draft PO with If-Match: 1 -> rowVersion becomes 2
	updBody := map[string]any{
		"poNo": "PO-001", "vendorId": vendorID, "orderDate": "2026-03-16", "currencyCode": "IDR", "totalAmount": "1200.0000",
		"lines": []map[string]any{{"lineNo": 1, "itemId": itemID, "orderQty": "4", "unitPrice": "300.0000"}},
	}
	rec = doPOReq(t, handler, http.MethodPut, "/purchase-orders/"+created.ID, orgID, userID, rwPerms, map[string]string{"If-Match": "1"}, updBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("update PO expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Stale Approve with If-Match: 1 fails 409; valid Approve with If-Match: 2 succeeds (rowVersion=3)
	recStale := doPOReq(t, handler, http.MethodPost, "/purchase-orders/"+created.ID+"/approve", orgID, userID, rwPerms, map[string]string{"If-Match": "1"}, nil)
	if recStale.Code != http.StatusConflict {
		t.Fatalf("expected 409 stale If-Match, got %d", recStale.Code)
	}
	rec = doPOReq(t, handler, http.MethodPost, "/purchase-orders/"+created.ID+"/approve", orgID, userID, rwPerms, map[string]string{"If-Match": "2"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve PO expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Receive PO with empty body (matches deploy/smoke-test.sh!) and If-Match: 3
	rec = doPOReq(t, handler, http.MethodPost, "/purchase-orders/"+created.ID+"/receive", orgID, userID, rwPerms, map[string]string{"If-Match": "3"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("receive PO expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var received poDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &received)
	if received.Status != "Received" || received.GrID == "" || len(store.grs) != 1 {
		t.Fatalf("expected Received status + GR record, got %+v grCount=%d", received, len(store.grs))
	}
	if len(store.outbox) != 2 || store.outbox[0].EventType != "procurement.po.approved" || store.outbox[1].EventType != "procurement.goods.received" {
		t.Fatalf("unexpected outbox events: %+v", store.outbox)
	}

	// GET GR & LIST PO with status filter
	rec = doPOReq(t, handler, http.MethodGet, "/goods-receipts/"+received.GrID, orgID, userID, readPerm, nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get GR expected 200, got %d", rec.Code)
	}
	rec = doPOReq(t, handler, http.MethodGet, "/purchase-orders?status=Received", orgID, userID, readPerm, nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list PO expected 200, got %d", rec.Code)
	}

	// 4. Draft Delete PO + PR lifecycle
	rec = doPOReq(t, handler, http.MethodPost, "/purchase-orders", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "k-po2"}, poBody)
	var po2 poDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &po2)
	rec = doPOReq(t, handler, http.MethodDelete, "/purchase-orders/"+po2.ID, orgID, userID, rwPerms, map[string]string{"If-Match": "1"}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete draft PO expected 204, got %d (%s)", rec.Code, rec.Body.String())
	}

	prBody := map[string]any{
		"prNo": "PR-001", "neededDate": time.Now().UTC().Format("2006-01-02"),
		"lines": []map[string]any{{"lineNo": 1, "itemId": itemID, "requestQty": "5"}},
	}
	rec = doPOReq(t, handler, http.MethodPost, "/purchase-requisitions", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "k-pr1"}, prBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create PR expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var pr prDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &pr)
	rec = doPOReq(t, handler, http.MethodPost, "/purchase-requisitions/"+pr.ID+"/submit", orgID, userID, rwPerms, map[string]string{"If-Match": "1"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit PR expected 200, got %d", rec.Code)
	}
	rec = doPOReq(t, handler, http.MethodPost, "/purchase-requisitions/"+pr.ID+"/approve", orgID, userID, rwPerms, map[string]string{"If-Match": "2"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve PR expected 200, got %d", rec.Code)
	}
}

