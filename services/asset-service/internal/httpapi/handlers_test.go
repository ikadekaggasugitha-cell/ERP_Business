package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/asset-service/internal/astdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type fakeAssetStore struct {
	completed   bool
	createCalls int
	assetRow    *astdb.AstAsset
}

func (f *fakeAssetStore) CreateAsset(context.Context, []uuid.UUID, astdb.CreateAssetParams) (astdb.CreateAssetRow, error) {
	f.createCalls++
	return astdb.CreateAssetRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeAssetStore) GetAsset(context.Context, []uuid.UUID, uuid.UUID) (astdb.AstAsset, error) {
	if f.assetRow != nil {
		return *f.assetRow, nil
	}
	return astdb.AstAsset{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeAssetStore) ListAssets(context.Context, []uuid.UUID, *string, int32, int32) ([]astdb.AstAsset, int64, error) {
	return nil, 0, nil
}
func (f *fakeAssetStore) UpdateAsset(context.Context, []uuid.UUID, astdb.UpdateAssetParams) (int32, error) {
	return 2, nil
}
func (f *fakeAssetStore) DeleteAsset(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) error {
	return nil
}
func (f *fakeAssetStore) CreateLocationHistory(context.Context, []uuid.UUID, astdb.CreateLocationHistoryParams) (astdb.CreateLocationHistoryRow, error) {
	return astdb.CreateLocationHistoryRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeAssetStore) ListLocationHistory(context.Context, []uuid.UUID, uuid.UUID) ([]astdb.AstAssetLocationHistory, error) {
	return nil, nil
}
func (f *fakeAssetStore) CreateMovement(context.Context, []uuid.UUID, astdb.CreateAssetMovementParams) (astdb.CreateAssetMovementRow, error) {
	return astdb.CreateAssetMovementRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeAssetStore) ListMovements(context.Context, []uuid.UUID, *uuid.UUID, int32, int32) ([]astdb.AstAssetMovement, int64, error) {
	return nil, 0, nil
}
func (f *fakeAssetStore) ApproveMovement(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32) (int32, error) {
	return 2, nil
}
func (f *fakeAssetStore) CompleteMovementAndEmit(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, int32) (int32, error) {
	f.completed = true
	return 3, nil
}
func (f *fakeAssetStore) RecordVerificationAndEmit(context.Context, []uuid.UUID, astdb.CreateAssetVerificationParams, uuid.UUID) (astdb.CreateAssetVerificationRow, error) {
	return astdb.CreateAssetVerificationRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeAssetStore) ListVerifications(context.Context, []uuid.UUID, *uuid.UUID, int32, int32) ([]astdb.AstAssetVerification, int64, error) {
	return nil, 0, nil
}

func newTestMux(store AssetStore) http.Handler {
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

func TestAssetStatusValidationAndMovementCompletion(t *testing.T) {
	store := &fakeAssetStore{}
	h := newTestMux(store)

	req := httptest.NewRequest(http.MethodPost, "/assets", strings.NewReader(`{"assetCode":"AST-01","name":"Chiller","category":"HVAC","status":"Broken"}`))
	authHeaders(req, "asset:write")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got %d", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/asset-movements/"+uuid.NewString()+"/complete", nil)
	authHeaders(req2, "asset:write")
	req2.Header.Set("If-Match", `W/"2"`)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK || !store.completed {
		t.Fatalf("expected 200 and completed=true, got %d: %s", rec2.Code, rec2.Body.String())
	}
}

func TestAssetDTOWireFormatAndIdempotency(t *testing.T) {
	acq, _ := platform.ParseDate("2026-03-15")
	store := &fakeAssetStore{
		assetRow: &astdb.AstAsset{
			ID: uuid.New(), OrganizationID: platform.SystemOrgID, AssetCode: "AST-TPL",
			Name: "Template Chiller", Category: "HVAC", Specification: []byte(`{"btu":12000}`),
			AcquisitionDate: acq, Status: "Active", RowVersion: 1,
		},
	}
	h := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/assets/"+store.assetRow.ID.String(), nil)
	authHeaders(req, "asset:read")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["acquisitionDate"] != "2026-03-15" || got["isSystem"] != true {
		t.Fatalf("unexpected asset DTO: %v", got)
	}
	if _, ok := got["specification"].(map[string]any); !ok {
		t.Fatalf("expected specification JSON object, got %T", got["specification"])
	}

	org := uuid.NewString()
	user := uuid.NewString()
	body := `{"assetCode":"AST-02","name":"Generator","category":"Power","status":"Active"}`
	for i := 0; i < 2; i++ {
		r := httptest.NewRequest(http.MethodPost, "/assets", strings.NewReader(body))
		r.Header.Set("X-Organization-Id", org)
		r.Header.Set("X-User-Id", user)
		r.Header.Set("X-Permissions", "asset:write")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "idem-ast-1")
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
