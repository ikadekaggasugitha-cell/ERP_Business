package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/consolidation-service/internal/consdb"
	"github.com/sinergiinvestasiproperti/erp/services/consolidation-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type mockConsStore struct{ httpapi.Store }

func (m *mockConsStore) CreateGroup(_ context.Context, _ []uuid.UUID, _ consdb.CreateConsolidationGroupParams) (consdb.CreateConsolidationGroupRow, error) {
	return consdb.CreateConsolidationGroupRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (m *mockConsStore) PostEliminations(_ context.Context, _ []uuid.UUID, _ uuid.UUID, _ int32, _ []consdb.CreateEliminationEntryParams, _ uuid.UUID) (int32, error) {
	return 2, nil
}
func (m *mockConsStore) FinalizeRunAndEmit(_ context.Context, _ []uuid.UUID, _ uuid.UUID, _ int32, _, _ uuid.UUID) (int32, error) {
	return 3, nil
}

func TestConsolidationEliminationBalanceAndFinalize(t *testing.T) {
	mux := http.NewServeMux()
	httpapi.NewServer(&mockConsStore{}).Register(mux)
	h := httpx.Middleware(mux)
	orgID, acc1, acc2 := uuid.NewString(), uuid.NewString(), uuid.NewString()

	// Unbalanced eliminations rejected with 422
	bad := `{"entries":[{"accountRef":"` + acc1 + `","debitAmount":"1000","creditAmount":"0"},{"accountRef":"` + acc2 + `","debitAmount":"0","creditAmount":"500"}]}`
	req := httptest.NewRequest("POST", "/consolidation-runs/"+uuid.NewString()+"/eliminate", strings.NewReader(bad))
	req.Header.Set("X-Organization-Id", orgID)
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Permissions", "consolidation:write")
	req.Header.Set("If-Match", `"1"`)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 unbalanced eliminations, got %d %s", rec.Code, rec.Body.String())
	}

	// Balanced eliminations succeed with 200
	good := `{"entries":[{"accountRef":"` + acc1 + `","debitAmount":"1000.0000","creditAmount":"0"},{"accountRef":"` + acc2 + `","debitAmount":"0","creditAmount":"1000.0000"}]}`
	req = httptest.NewRequest("POST", "/consolidation-runs/"+uuid.NewString()+"/eliminate", strings.NewReader(good))
	req.Header.Set("X-Organization-Id", orgID)
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Permissions", "consolidation:write")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 balanced eliminations, got %d %s", rec.Code, rec.Body.String())
	}
}
