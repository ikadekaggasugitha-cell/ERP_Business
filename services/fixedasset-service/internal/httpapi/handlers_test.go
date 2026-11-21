package httpapi_test

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/fadb"
	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type mockFAStore struct{ httpapi.Store }

func (m *mockFAStore) CreateCategory(_ context.Context, _ []uuid.UUID, _ fadb.CreateAssetCategoryParams) (fadb.CreateAssetCategoryRow, error) {
	return fadb.CreateAssetCategoryRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (m *mockFAStore) CreateAsset(_ context.Context, _ []uuid.UUID, _ fadb.CreateFixedAssetParams) (fadb.CreateFixedAssetRow, error) {
	return fadb.CreateFixedAssetRow{ID: uuid.New(), Status: "Draft", RowVersion: 1}, nil
}
func (m *mockFAStore) CapitalizeAndEmit(_ context.Context, _ []uuid.UUID, _ uuid.UUID, _ int32, _ pgtype.Date, _, _ uuid.UUID) (int32, error) {
	return 2, nil
}
func (m *mockFAStore) DepreciateAndEmit(_ context.Context, _ []uuid.UUID, _ uuid.UUID, _ int32, _ uuid.UUID, _ pgtype.Date, _, _ uuid.UUID) (fadb.CreateDepreciationEntryRow, string, string, int32, error) {
	return fadb.CreateDepreciationEntryRow{ID: uuid.New(), RowVersion: 1}, "1000000.0000", "47000000.0000", 3, nil
}

func doFA(h http.Handler, method, path, body, perms, ifMatch string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-Organization-Id", uuid.NewString())
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Permissions", perms)
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestFixedAssetDepreciationAndLifecycle(t *testing.T) {
	// Exact StraightLine depreciation capped at salvageValue
	dep, nbv, err := repository.ComputeMonthlyDepreciation("StraightLine", big.NewRat(50000000, 1), big.NewRat(2000000, 1), big.NewRat(50000000, 1), 48)
	if err != nil || dep.FloatString(4) != "1000000.0000" || nbv.FloatString(4) != "49000000.0000" {
		t.Fatalf("unexpected SL depreciation: dep=%v nbv=%v err=%v", dep, nbv, err)
	}
	// Cap final month at remaining NBV - salvage
	dep, nbv, err = repository.ComputeMonthlyDepreciation("StraightLine", big.NewRat(50000000, 1), big.NewRat(2000000, 1), big.NewRat(2500000, 1), 48)
	if err != nil || dep.FloatString(4) != "500000.0000" || nbv.FloatString(4) != "2000000.0000" {
		t.Fatalf("expected capped depreciation 500000/2000000, got dep=%v nbv=%v err=%v", dep, nbv, err)
	}

	mux := http.NewServeMux()
	httpapi.NewServer(&mockFAStore{}).Register(mux)
	h := httpx.Middleware(mux)

	if rec := doFA(h, "POST", "/asset-categories", `{"code":"VEH","name":"Vehicles"}`, "fixedasset:read", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if rec := doFA(h, "POST", "/asset-categories", `{"code":"VEH","name":"Vehicles"}`, "fixedasset:write", ""); rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d %s", rec.Code, rec.Body.String())
	}
	catID, finOrg, acc := uuid.NewString(), uuid.NewString(), uuid.NewString()
	// Salvage > cost rejected
	bad := `{"assetNo":"FA-1","name":"Truck","assetCategoryId":"` + catID + `","financialOrganizationRef":"` + finOrg + `","acquisitionDate":"2026-01-01","acquisitionCost":"1000","salvageValue":"2000","usefulLifeMonths":12,"assetAccountId":"` + acc + `","accumDepreciationAccountId":"` + acc + `","depreciationExpenseAccountId":"` + acc + `"}`
	if rec := doFA(h, "POST", "/fixed-assets", bad, "fixedasset:write", ""); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 salvage>cost, got %d", rec.Code)
	}
	okBody := `{"assetNo":"FA-1","name":"Truck","assetCategoryId":"` + catID + `","financialOrganizationRef":"` + finOrg + `","acquisitionDate":"2026-01-01","acquisitionCost":"48000000.0000","salvageValue":"0","usefulLifeMonths":48,"assetAccountId":"` + acc + `","accumDepreciationAccountId":"` + acc + `","depreciationExpenseAccountId":"` + acc + `"}`
	if rec := doFA(h, "POST", "/fixed-assets", okBody, "fixedasset:write", ""); rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 asset, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := doFA(h, "POST", "/fixed-assets/"+uuid.NewString()+"/capitalize", "", "fixedasset:write", `"1"`); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 capitalize, got %d", rec.Code)
	}
	if rec := doFA(h, "POST", "/fixed-assets/"+uuid.NewString()+"/depreciate", `{"fiscalPeriodRef":"`+uuid.NewString()+`"}`, "fixedasset:write", `"2"`); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 depreciate, got %d %s", rec.Code, rec.Body.String())
	}
}
