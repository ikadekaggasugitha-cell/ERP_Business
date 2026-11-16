package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/inventorydb"
	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type fakeInvStore struct {
	warehouses map[uuid.UUID]inventorydb.ListWarehousesRow
	balances   map[string]inventorydb.ListStockBalancesRow
	movements  []inventorydb.ListStockMovementsRow
	res        map[uuid.UUID]inventorydb.GetReservationRow
	outbox     []inventorydb.InsertOutboxParams
}

func newFakeInvStore() *fakeInvStore {
	return &fakeInvStore{
		warehouses: make(map[uuid.UUID]inventorydb.ListWarehousesRow),
		balances:   make(map[string]inventorydb.ListStockBalancesRow),
		res:        make(map[uuid.UUID]inventorydb.GetReservationRow),
	}
}

func balKey(wh, item uuid.UUID) string { return wh.String() + ":" + item.String() }

func (f *fakeInvStore) CreateWarehouse(_ context.Context, _ []uuid.UUID, arg inventorydb.CreateWarehouseParams) (inventorydb.CreateWarehouseRow, error) {
	id := uuid.New()
	f.warehouses[id] = inventorydb.ListWarehousesRow{
		ID:                id,
		OrganizationID:    arg.OrganizationID,
		Code:              arg.Code,
		Name:              arg.Name,
		OrganizationRefID: arg.OrganizationRefID,
		RowVersion:        1,
	}
	return inventorydb.CreateWarehouseRow{
		ID:                id,
		Code:              arg.Code,
		Name:              arg.Name,
		OrganizationRefID: arg.OrganizationRefID,
		RowVersion:        1,
	}, nil
}

func (f *fakeInvStore) ListWarehousesWithCount(_ context.Context, _ []uuid.UUID, _ inventorydb.ListWarehousesParams) ([]inventorydb.ListWarehousesRow, int64, error) {
	out := make([]inventorydb.ListWarehousesRow, 0, len(f.warehouses))
	for _, w := range f.warehouses {
		out = append(out, w)
	}
	return out, int64(len(out)), nil
}

func (f *fakeInvStore) ListStockBalancesWithCount(_ context.Context, _ []uuid.UUID, arg inventorydb.ListStockBalancesParams) ([]inventorydb.ListStockBalancesRow, int64, error) {
	out := make([]inventorydb.ListStockBalancesRow, 0, len(f.balances))
	for _, b := range f.balances {
		if arg.WarehouseID != nil && b.WarehouseID != *arg.WarehouseID {
			continue
		}
		if arg.ItemID != nil && b.ItemID != *arg.ItemID {
			continue
		}
		out = append(out, b)
	}
	return out, int64(len(out)), nil
}

func (f *fakeInvStore) ListStockMovementsWithCount(_ context.Context, _ []uuid.UUID, _ inventorydb.ListStockMovementsParams) ([]inventorydb.ListStockMovementsRow, int64, error) {
	return f.movements, int64(len(f.movements)), nil
}

func (f *fakeInvStore) RecordStockMovement(_ context.Context, _ []uuid.UUID, arg inventorydb.CreateStockMovementParams, unitCost pgtype.Numeric) (inventorydb.CreateStockMovementRow, error) {
	k := balKey(arg.WarehouseID, arg.ItemID)
	qtyRat := platform.NumericToRat(arg.Quantity)
	bal, ok := f.balances[k]
	if !ok {
		zero, _ := platform.ParseNumeric("0")
		bal = inventorydb.ListStockBalancesRow{
			ID: uuid.New(), OrganizationID: arg.OrganizationID, WarehouseID: arg.WarehouseID, ItemID: arg.ItemID,
			OnHandQty: zero, ReservedQty: zero, AvailableQty: zero, AvgCost: zero, RowVersion: 1,
		}
	}
	onHand := platform.NumericToRat(bal.OnHandQty)
	resQty := platform.NumericToRat(bal.ReservedQty)
	if arg.MovementType == "Receipt" {
		onHand = new(big.Rat).Add(onHand, qtyRat)
		if c := platform.NumericToRat(unitCost); c != nil && c.Sign() > 0 {
			bal.AvgCost = unitCost
		}
	} else if arg.MovementType == "Issue" {
		avail := new(big.Rat).Sub(onHand, resQty)
		need := new(big.Rat).Abs(qtyRat)
		if avail.Cmp(need) < 0 {
			return inventorydb.CreateStockMovementRow{}, repository.ErrInsufficientStock
		}
		onHand = new(big.Rat).Sub(onHand, need)
		arg.Quantity, _ = platform.RatToNumeric(new(big.Rat).Neg(need), 4)
	}
	bal.OnHandQty, _ = platform.RatToNumeric(onHand, 4)
	bal.AvailableQty, _ = platform.RatToNumeric(new(big.Rat).Sub(onHand, resQty), 4)
	f.balances[k] = bal
	id := uuid.New()
	f.movements = append(f.movements, inventorydb.ListStockMovementsRow{
		ID: id, OrganizationID: arg.OrganizationID, MovementDate: arg.MovementDate,
		WarehouseID: arg.WarehouseID, ItemID: arg.ItemID, MovementType: arg.MovementType,
		Quantity: arg.Quantity, SourceDocumentRef: arg.SourceDocumentRef, CorrelationID: arg.CorrelationID, RowVersion: 1,
	})
	return inventorydb.CreateStockMovementRow{ID: id, MovementDate: arg.MovementDate, RowVersion: 1}, nil
}

func (f *fakeInvStore) CreateReservationAndEmit(_ context.Context, _ []uuid.UUID, arg inventorydb.CreateReservationParams, ev inventorydb.InsertOutboxParams) (inventorydb.CreateReservationRow, error) {
	k := balKey(arg.WarehouseID, arg.ItemID)
	bal, ok := f.balances[k]
	if !ok {
		return inventorydb.CreateReservationRow{}, repository.ErrInsufficientStock
	}
	need := platform.NumericToRat(arg.ReservedQty)
	avail := platform.NumericToRat(bal.AvailableQty)
	if avail.Cmp(need) < 0 {
		return inventorydb.CreateReservationRow{}, repository.ErrInsufficientStock
	}
	onHand := platform.NumericToRat(bal.OnHandQty)
	newRes := new(big.Rat).Add(platform.NumericToRat(bal.ReservedQty), need)
	bal.ReservedQty, _ = platform.RatToNumeric(newRes, 4)
	bal.AvailableQty, _ = platform.RatToNumeric(new(big.Rat).Sub(onHand, newRes), 4)
	f.balances[k] = bal

	id := uuid.New()
	f.res[id] = inventorydb.GetReservationRow{
		ID: id, OrganizationID: arg.OrganizationID, ItemID: arg.ItemID, WarehouseID: arg.WarehouseID,
		ReservedQty: arg.ReservedQty, SalesOrderRef: arg.SalesOrderRef, Status: "Reserved", RowVersion: 1,
	}
	ev.AggregateID = id
	f.outbox = append(f.outbox, ev)
	return inventorydb.CreateReservationRow{ID: id, Status: "Reserved", RowVersion: 1}, nil
}

func (f *fakeInvStore) IssueAndEmit(_ context.Context, _ []uuid.UUID, arg inventorydb.IssueReservationParams, corrID uuid.UUID, moveDate pgtype.Date, ev inventorydb.InsertOutboxParams) (int32, int64, error) {
	r, ok := f.res[arg.ID]
	if !ok || r.Status != "Reserved" || r.RowVersion != arg.RowVersion {
		return 0, 0, nil
	}
	r.Status = "Consumed"
	r.RowVersion++
	f.res[arg.ID] = r
	k := balKey(r.WarehouseID, r.ItemID)
	bal := f.balances[k]
	qty := platform.NumericToRat(r.ReservedQty)
	onHand := new(big.Rat).Sub(platform.NumericToRat(bal.OnHandQty), qty)
	resQty := new(big.Rat).Sub(platform.NumericToRat(bal.ReservedQty), qty)
	bal.OnHandQty, _ = platform.RatToNumeric(onHand, 4)
	bal.ReservedQty, _ = platform.RatToNumeric(resQty, 4)
	bal.AvailableQty, _ = platform.RatToNumeric(new(big.Rat).Sub(onHand, resQty), 4)
	f.balances[k] = bal
	negQty, _ := platform.RatToNumeric(new(big.Rat).Neg(qty), 4)
	f.movements = append(f.movements, inventorydb.ListStockMovementsRow{
		ID: uuid.New(), OrganizationID: r.OrganizationID, MovementDate: moveDate,
		WarehouseID: r.WarehouseID, ItemID: r.ItemID, MovementType: "Issue", Quantity: negQty, CorrelationID: corrID, RowVersion: 1,
	})
	f.outbox = append(f.outbox, ev)
	return r.RowVersion, 1, nil
}

func (f *fakeInvStore) ReleaseReservation(_ context.Context, _ []uuid.UUID, arg inventorydb.ReleaseReservationParams) (int32, int64, error) {
	r, ok := f.res[arg.ID]
	if !ok || r.Status != "Reserved" || r.RowVersion != arg.RowVersion {
		return 0, 0, nil
	}
	r.Status = "Released"
	r.RowVersion++
	f.res[arg.ID] = r
	return r.RowVersion, 1, nil
}

func (f *fakeInvStore) GetByID(_ context.Context, _ []uuid.UUID, id uuid.UUID) (inventorydb.GetReservationRow, error) {
	r, ok := f.res[id]
	if !ok {
		return inventorydb.GetReservationRow{}, pgx.ErrNoRows
	}
	return r, nil
}

func (f *fakeInvStore) ListWithCount(_ context.Context, _ []uuid.UUID, _ inventorydb.ListReservationsParams) ([]inventorydb.ListReservationsRow, int64, error) {
	out := make([]inventorydb.ListReservationsRow, 0, len(f.res))
	for _, r := range f.res {
		out = append(out, inventorydb.ListReservationsRow{
			ID: r.ID, OrganizationID: r.OrganizationID, ItemID: r.ItemID, WarehouseID: r.WarehouseID,
			ReservedQty: r.ReservedQty, SalesOrderRef: r.SalesOrderRef, Status: r.Status, RowVersion: r.RowVersion,
		})
	}
	return out, int64(len(out)), nil
}

func doInvReq(t *testing.T, handler http.Handler, method, path, orgID, userID, perms string, headers map[string]string, body any) *httptest.ResponseRecorder {
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

func TestInventoryBalanceMovementAndReservationFlow(t *testing.T) {
	store := newFakeInvStore()
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	handler := httpx.Middleware(mux)

	orgID := uuid.NewString()
	userID := uuid.NewString()
	itemID := uuid.NewString()
	readPerm := "inventory:read"
	rwPerms := "inventory:read,inventory:write"

	// 1. RBAC rejection
	rec := doInvReq(t, handler, http.MethodPost, "/warehouses", orgID, userID, readPerm, nil, map[string]any{"code": "WH1", "name": "Gudang"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without inventory:write, got %d", rec.Code)
	}

	// 2. Create warehouse
	rec = doInvReq(t, handler, http.MethodPost, "/warehouses", orgID, userID, rwPerms, nil, map[string]any{"code": "WH-01", "name": "Gudang Utama"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create warehouse expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var wh warehouseDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &wh)

	// 3. Reserving with 0 available stock fails with 422 INSUFFICIENT_STOCK
	resBody := map[string]any{"itemId": itemID, "warehouseId": wh.ID, "reservedQty": "5"}
	rec = doInvReq(t, handler, http.MethodPost, "/reservations", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "r-fail"}, resBody)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 INSUFFICIENT_STOCK, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 4. Receive 20 units via POST /stock-movements -> list balances shows available 20.0000
	movBody := map[string]any{"warehouseId": wh.ID, "itemId": itemID, "movementType": "Receipt", "quantity": "20", "unitCost": "150"}
	rec = doInvReq(t, handler, http.MethodPost, "/stock-movements", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "m-in"}, movBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("receipt movement expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	rec = doInvReq(t, handler, http.MethodGet, "/stock-balances?warehouseId="+wh.ID, orgID, userID, readPerm, nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list balances expected 200, got %d", rec.Code)
	}

	// 5. Reserve 5 -> Issue 5 -> balance drops to onHand=15, reserved=0, movements count = 2
	rec = doInvReq(t, handler, http.MethodPost, "/reservations", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "r-ok"}, resBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create reservation expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var res reservationDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &res)

	rec = doInvReq(t, handler, http.MethodPost, "/reservations/"+res.ID+"/issue", orgID, userID, rwPerms, map[string]string{"If-Match": "1"}, map[string]any{"amount": "750.0000"})
	if rec.Code != http.StatusOK {
		t.Fatalf("issue reservation expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if len(store.movements) != 2 || len(store.outbox) != 2 || store.outbox[1].EventType != "inventory.goods.issued" {
		t.Fatalf("expected 2 movements and 2 outbox events, got mov=%d outbox=%+v", len(store.movements), store.outbox)
	}

	// 6. Release workflow on a second reservation
	rec = doInvReq(t, handler, http.MethodPost, "/reservations", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "r-rel"}, resBody)
	var res2 reservationDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &res2)
	rec = doInvReq(t, handler, http.MethodPost, "/reservations/"+res2.ID+"/release", orgID, userID, rwPerms, map[string]string{"If-Match": "1"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("release expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
}

