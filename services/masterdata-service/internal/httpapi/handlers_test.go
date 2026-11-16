package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/masterdata-service/internal/masterdatadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type fakeMDStore struct {
	updatedEvents []masterdatadb.InsertOutboxParams
}

func (f *fakeMDStore) CreatePartyAndEmit(_ context.Context, _ []uuid.UUID, _ masterdatadb.CreatePartyParams, _ []masterdatadb.CreatePartyAddressParams, _ masterdatadb.InsertOutboxParams) (masterdatadb.CreatePartyRow, error) {
	return masterdatadb.CreatePartyRow{ID: uuid.MustParse("10000000-0000-0000-0000-000000000031"), RowVersion: 1}, nil
}

func (f *fakeMDStore) GetPartyWithAddresses(_ context.Context, scope []uuid.UUID, id uuid.UUID) (masterdatadb.MdParty, []masterdatadb.ListPartyAddressesRow, error) {
	return masterdatadb.MdParty{ID: id, OrganizationID: scope[0], Code: "CUST-1", Name: "PT Acme", PartyType: "customer", RowVersion: 1}, nil, nil
}

func (f *fakeMDStore) ListPartiesWithCount(_ context.Context, scope []uuid.UUID, _ masterdatadb.ListPartiesParams) ([]masterdatadb.ListPartiesRow, int64, error) {
	return []masterdatadb.ListPartiesRow{{ID: uuid.New(), OrganizationID: scope[0], Code: "CUST-1", Name: "PT Acme", PartyType: "customer", RowVersion: 1}}, 1, nil
}

func (f *fakeMDStore) UpdatePartyAndEmit(_ context.Context, _ []uuid.UUID, _ masterdatadb.UpdatePartyParams, events ...masterdatadb.InsertOutboxParams) (int32, error) {
	f.updatedEvents = append(f.updatedEvents, events...)
	return 2, nil
}

func (f *fakeMDStore) DeleteParty(_ context.Context, _ []uuid.UUID, _ masterdatadb.DeletePartyParams) (int64, error) {
	return 1, nil
}

func (f *fakeMDStore) CreateItemAndEmit(_ context.Context, _ []uuid.UUID, _ masterdatadb.CreateItemParams, _ masterdatadb.InsertOutboxParams) (masterdatadb.CreateItemRow, error) {
	return masterdatadb.CreateItemRow{ID: uuid.MustParse("10000000-0000-0000-0000-000000000032"), RowVersion: 1}, nil
}

func (f *fakeMDStore) GetItemWithUOMs(_ context.Context, scope []uuid.UUID, id uuid.UUID) (masterdatadb.MdItem, []masterdatadb.ListItemUOMsRow, error) {
	one, _ := platform.ParseNumeric("1")
	return masterdatadb.MdItem{ID: id, OrganizationID: scope[0], Code: "SKU-1", Name: "Unit", ItemType: "stock", BaseUom: "PCS", RowVersion: 1},
		[]masterdatadb.ListItemUOMsRow{{ID: uuid.New(), ItemID: id, UomCode: "PCS", ConversionToBase: one, IsBase: true, RowVersion: 1}}, nil
}

func (f *fakeMDStore) ListItemsWithCount(_ context.Context, scope []uuid.UUID, _ masterdatadb.ListItemsParams) ([]masterdatadb.ListItemsRow, int64, error) {
	return []masterdatadb.ListItemsRow{{ID: uuid.New(), OrganizationID: scope[0], Code: "SKU-1", Name: "Unit", ItemType: "stock", BaseUom: "PCS", RowVersion: 1}}, 1, nil
}

func (f *fakeMDStore) UpdateItemAndEmit(_ context.Context, _ []uuid.UUID, _ masterdatadb.UpdateItemParams, _ masterdatadb.InsertOutboxParams) (int32, error) {
	return 2, nil
}

func (f *fakeMDStore) DeleteItem(_ context.Context, _ []uuid.UUID, _ masterdatadb.DeleteItemParams) (int64, error) {
	return 1, nil
}

func mdHdrs(perms string) map[string]string {
	return map[string]string{
		"X-Org-Scope":   "00000000-0000-0000-0000-000000000001",
		"X-User-ID":     "00000000-0000-0000-0000-000000000010",
		"X-Permissions": perms,
	}
}

func doMDReq(t *testing.T, h http.Handler, method, path string, body any, hdrs map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestMasterdataPartiesAndItemsCRUD(t *testing.T) {
	store := &fakeMDStore{}
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	if rr := doMDReq(t, h, http.MethodGet, "/items", nil, mdHdrs("user:read")); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without masterdata:read, got %d", rr.Code)
	}

	hdrs := mdHdrs("masterdata:read,masterdata:write")
	if rr := doMDReq(t, h, http.MethodPost, "/parties", map[string]any{
		"code":      "BAD",
		"name":      "Bad",
		"partyType": "alien",
	}, hdrs); rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid partyType, got %d", rr.Code)
	}

	hdrs["If-Match"] = `"1"`
	rr := doMDReq(t, h, http.MethodPut, "/parties/10000000-0000-0000-0000-000000000031", map[string]any{
		"name":      "PT Acme Updated",
		"partyType": "both",
	}, hdrs)
	if rr.Code != http.StatusOK || len(store.updatedEvents) != 2 {
		t.Fatalf("expected 200 and 2 outbox events on party update, got %d (%d events)", rr.Code, len(store.updatedEvents))
	}

	if rr := doMDReq(t, h, http.MethodPost, "/items", map[string]any{
		"code":     "SKU-1",
		"name":     "Item 1",
		"itemType": "stock",
		"baseUom":  "pcs",
	}, hdrs); rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 on create item, got %d", rr.Code)
	}

	if rr := doMDReq(t, h, http.MethodGet, "/items", nil, hdrs); rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on list items, got %d", rr.Code)
	}
}

