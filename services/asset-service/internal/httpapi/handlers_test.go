package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/asset-service/internal/astdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type fakeAssetStore struct {
	completed bool
}

func (f *fakeAssetStore) CreateAsset(context.Context, []uuid.UUID, astdb.CreateAssetParams) (astdb.CreateAssetRow, error) {
	return astdb.CreateAssetRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (f *fakeAssetStore) GetAsset(context.Context, []uuid.UUID, uuid.UUID) (astdb.AstAsset, error) {
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
