package httpapi_test

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/fadb"
	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type mockFAStore struct {
	httpapi.Store
	capCalls int
}

func (m *mockFAStore) CreateCategory(_ context.Context, _ []uuid.UUID, _ fadb.CreateAssetCategoryParams) (fadb.CreateAssetCategoryRow, error) {
	return fadb.CreateAssetCategoryRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (m *mockFAStore) ListCategories(_ context.Context, _ []uuid.UUID, _, _ int32) ([]fadb.FaAssetCategory, int64, error) {
	months := int32(48)
	return []fadb.FaAssetCategory{{
		ID: uuid.New(), OrganizationID: platform.SystemOrgID, Code: "VEH", Name: "Vehicles",
		DefaultDepreciationMethod: "StraightLine", DefaultUsefulLifeMonths: &months, IsActive: true, RowVersion: 1,
	}}, 1, nil
}
func (m *mockFAStore) CreateAsset(_ context.Context, _ []uuid.UUID, _ fadb.CreateFixedAssetParams) (fadb.CreateFixedAssetRow, error) {
	return fadb.CreateFixedAssetRow{ID: uuid.New(), Status: "Draft", RowVersion: 1}, nil
}
func (m *mockFAStore) CapitalizeAndEmit(_ context.Context, _ []uuid.UUID, _ uuid.UUID, _ int32, _ pgtype.Date, _, _ uuid.UUID) (int32, error) {
	m.capCalls++
	return 2, nil
}
func (m *mockFAStore) DepreciateAndEmit(_ context.Context, _ []uuid.UUID, _ uuid.UUID, _ int32, _ uuid.UUID, _ pgtype.Date, _, _ *big.Rat, _, _ uuid.UUID) (fadb.CreateDepreciationEntryRow, string, string, int32, error) {
	return fadb.CreateDepreciationEntryRow{ID: uuid.New(), RowVersion: 1}, "1000000.0000", "47000000.0000", 3, nil
}
func (m *mockFAStore) DisposeAndEmit(_ context.Context, _ []uuid.UUID, _ uuid.UUID, _ int32, _ pgtype.Numeric, _ pgtype.Date, _ *string, _, _ uuid.UUID) (int32, string, error) {
	return 4, "-47000000.0000", nil
}
func (m *mockFAStore) ListDepreciationEntries(_ context.Context, _ []uuid.UUID, assetID uuid.UUID) ([]fadb.FaDepreciationEntry, error) {
	dep, _ := platform.ParseNumeric("1000000.0000")
	nbv, _ := platform.ParseNumeric("47000000.0000")
	return []fadb.FaDepreciationEntry{{
		ID: uuid.New(), OrganizationID: uuid.New(), FixedAssetID: assetID, FiscalPeriodRef: uuid.New(),
		DepreciationAmount: dep, AccumulatedDepreciation: dep, NetBookValue: nbv, IsPosted: true,
		RunDate: pgtype.Date{Time: time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), Valid: true}, RowVersion: 1,
	}}, nil
}

func doFA(h http.Handler, method, path, body, perms, ifMatch string) *httptest.ResponseRecorder {
	return doFAWithIdem(h, method, path, body, perms, ifMatch, "", "")
}

func doFAWithIdem(h http.Handler, method, path, body, perms, ifMatch, orgID, idemKey string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if orgID == "" {
		orgID = uuid.NewString()
	}
	req.Header.Set("X-Organization-Id", orgID)
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Permissions", perms)
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
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
	// UnitsOfProduction with explicit unitsProduced / totalEstimatedUnits
	dep, nbv, err = repository.ComputeMonthlyDepreciationWithUnits("UnitsOfProduction", big.NewRat(50000000, 1), big.NewRat(2000000, 1), big.NewRat(50000000, 1), 48, big.NewRat(250, 1), big.NewRat(1000, 1))
	if err != nil || dep.FloatString(4) != "12000000.0000" || nbv.FloatString(4) != "38000000.0000" {
		t.Fatalf("expected UoP depreciation 12000000/38000000, got dep=%v nbv=%v err=%v", dep, nbv, err)
	}

	st := &mockFAStore{}
	mux := http.NewServeMux()
	httpapi.NewServer(st).Register(mux)
	h := httpx.Middleware(mux)

	if rec := doFA(h, "POST", "/asset-categories", `{"code":"VEH","name":"Vehicles"}`, "fixedasset:read", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if rec := doFA(h, "POST", "/asset-categories", `{"code":"VEH","name":"Vehicles"}`, "fixedasset:write", ""); rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d %s", rec.Code, rec.Body.String())
	}
	// Verify camelCase DTO on GET /asset-categories
	catRec := doFA(h, "GET", "/asset-categories", "", "fixedasset:read", "")
	if catRec.Code != http.StatusOK || !strings.Contains(catRec.Body.String(), `"defaultDepreciationMethod":"StraightLine"`) || !strings.Contains(catRec.Body.String(), `"isSystem":true`) {
		t.Fatalf("expected camelCase category DTO, got %d %s", catRec.Code, catRec.Body.String())
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

	// Capitalize idempotency & conflict check
	assetID := uuid.NewString()
	orgID := uuid.NewString()
	if rec := doFAWithIdem(h, "POST", "/fixed-assets/"+assetID+"/capitalize", "", "fixedasset:write", `"1"`, orgID, "cap-1"); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 capitalize, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec := doFAWithIdem(h, "POST", "/fixed-assets/"+assetID+"/capitalize", "", "fixedasset:write", `"1"`, orgID, "cap-1"); rec.Code != http.StatusOK || rec.Header().Get("Idempotency-Replayed") != "true" || st.capCalls != 1 {
		t.Fatalf("expected idempotent replay on capitalize, got %d calls=%d", rec.Code, st.capCalls)
	}
	if rec := doFAWithIdem(h, "POST", "/fixed-assets/"+assetID+"/capitalize", `{"transactionDate":"2026-02-01"}`, "fixedasset:write", `"1"`, orgID, "cap-1"); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 on mismatched capitalize payload, got %d", rec.Code)
	}

	if rec := doFA(h, "POST", "/fixed-assets/"+assetID+"/depreciate", `{"fiscalPeriodRef":"`+uuid.NewString()+`","runDate":"2026-01-31"}`, "fixedasset:write", ""); rec.Code != http.StatusPreconditionRequired {
		t.Fatalf("expected 428 missing If-Match, got %d", rec.Code)
	}
	if rec := doFA(h, "POST", "/fixed-assets/"+assetID+"/depreciate", `{"fiscalPeriodRef":"`+uuid.NewString()+`","runDate":"2026-01-31"}`, "fixedasset:write", `"2"`); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 depreciate, got %d %s", rec.Code, rec.Body.String())
	}

	// Verify camelCase decimal string DTO on GET /fixed-assets/{id}/depreciation-entries
	depRec := doFA(h, "GET", "/fixed-assets/"+assetID+"/depreciation-entries", "", "fixedasset:read", "")
	if depRec.Code != http.StatusOK {
		t.Fatalf("expected 200 listDepreciationEntries, got %d", depRec.Code)
	}
	var env struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(depRec.Body.Bytes(), &env); err != nil || len(env.Data) != 1 || env.Data[0]["depreciationAmount"] != "1000000.0000" || env.Data[0]["runDate"] != "2026-01-31" {
		t.Fatalf("unexpected depreciation DTO wire payload: %s", depRec.Body.String())
	}
}
