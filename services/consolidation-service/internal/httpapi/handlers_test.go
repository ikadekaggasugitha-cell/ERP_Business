package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/consolidation-service/internal/consdb"
	"github.com/sinergiinvestasiproperti/erp/services/consolidation-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type mockConsStore struct {
	httpapi.Store
	elimCalls int
}

func (m *mockConsStore) CreateGroup(_ context.Context, _ []uuid.UUID, _ consdb.CreateConsolidationGroupParams) (consdb.CreateConsolidationGroupRow, error) {
	return consdb.CreateConsolidationGroupRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (m *mockConsStore) ListRules(_ context.Context, _ []uuid.UUID, _ *uuid.UUID, _, _ int32) ([]consdb.ConsEliminationRule, int64, error) {
	return []consdb.ConsEliminationRule{{
		ID: uuid.New(), OrganizationID: platform.SystemOrgID, ConsolidationGroupID: uuid.New(),
		RuleType: "ICReceivablePayable", Config: []byte(`{"matchOn":"counterparty"}`), IsActive: true, RowVersion: 1,
	}}, 1, nil
}
func (m *mockConsStore) PostEliminations(_ context.Context, _ []uuid.UUID, _ uuid.UUID, _ int32, _ []consdb.CreateEliminationEntryParams, _ uuid.UUID) (int32, error) {
	m.elimCalls++
	return 2, nil
}
func (m *mockConsStore) FinalizeRunAndEmit(_ context.Context, _ []uuid.UUID, _ uuid.UUID, _ int32, _, _ uuid.UUID) (int32, error) {
	return 3, nil
}

func TestConsolidationEliminationBalanceAndFinalize(t *testing.T) {
	st := &mockConsStore{}
	mux := http.NewServeMux()
	httpapi.NewServer(st).Register(mux)
	h := httpx.Middleware(mux)
	orgID, acc1, acc2 := uuid.NewString(), uuid.NewString(), uuid.NewString()

	// Verify camelCase DTO + JSON object (not base64) for elimination rule config
	ruleReq := httptest.NewRequest("GET", "/elimination-rules", nil)
	ruleReq.Header.Set("X-Organization-Id", orgID)
	ruleReq.Header.Set("X-Permissions", "consolidation:read")
	ruleRec := httptest.NewRecorder()
	h.ServeHTTP(ruleRec, ruleReq)
	var ruleEnv struct {
		Data []struct {
			RuleType string         `json:"ruleType"`
			Config   map[string]any `json:"config"`
			IsSystem bool           `json:"isSystem"`
		} `json:"data"`
	}
	if ruleRec.Code != http.StatusOK || json.Unmarshal(ruleRec.Body.Bytes(), &ruleEnv) != nil || len(ruleEnv.Data) != 1 || ruleEnv.Data[0].Config["matchOn"] != "counterparty" || !ruleEnv.Data[0].IsSystem {
		t.Fatalf("unexpected rule DTO wire payload: %d %s", ruleRec.Code, ruleRec.Body.String())
	}

	// Head Office Tier Guard: SUBSIDIARY_ADMIN blocked on consolidation mutations (403)
	subReq := httptest.NewRequest("POST", "/consolidation-groups", strings.NewReader(`{"code":"GRP-1","name":"Sub Group","presentationCurrencyCode":"IDR"}`))
	subReq.Header.Set("X-Organization-Id", orgID)
	subReq.Header.Set("X-User-Id", uuid.NewString())
	subReq.Header.Set("X-Roles", "SUBSIDIARY_ADMIN")
	subReq.Header.Set("X-Permissions", "consolidation:write")
	subRec := httptest.NewRecorder()
	h.ServeHTTP(subRec, subReq)
	if subRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for SUBSIDIARY_ADMIN on consolidation write, got %d %s", subRec.Code, subRec.Body.String())
	}

	// Unbalanced eliminations rejected with 422
	bad := `{"entries":[{"accountRef":"` + acc1 + `","debitAmount":"1000","creditAmount":"0"},{"accountRef":"` + acc2 + `","debitAmount":"0","creditAmount":"500"}]}`
	req := httptest.NewRequest("POST", "/consolidation-runs/"+uuid.NewString()+"/eliminate", strings.NewReader(bad))
	req.Header.Set("X-Organization-Id", orgID)
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Roles", "HOLDING_ADMIN")
	req.Header.Set("X-Permissions", "consolidation:write")
	req.Header.Set("If-Match", `"1"`)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 unbalanced eliminations, got %d %s", rec.Code, rec.Body.String())
	}

	// Negative amount rejected with 422
	neg := `{"entries":[{"accountRef":"` + acc1 + `","debitAmount":"-1000","creditAmount":"0"},{"accountRef":"` + acc2 + `","debitAmount":"2000","creditAmount":"0"},{"accountRef":"` + acc2 + `","debitAmount":"0","creditAmount":"1000"}]}`
	req = httptest.NewRequest("POST", "/consolidation-runs/"+uuid.NewString()+"/eliminate", strings.NewReader(neg))
	req.Header.Set("X-Organization-Id", orgID)
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Roles", "HOLDING_ADMIN")
	req.Header.Set("X-Permissions", "consolidation:write")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 negative elimination line, got %d %s", rec.Code, rec.Body.String())
	}

	// Balanced eliminations succeed with 200 and replay idempotently
	good := `{"entries":[{"accountRef":"` + acc1 + `","debitAmount":"1000.0000","creditAmount":"0"},{"accountRef":"` + acc2 + `","debitAmount":"0","creditAmount":"1000.0000"}]}`
	runID := uuid.NewString()
	for i := 0; i < 2; i++ {
		req = httptest.NewRequest("POST", "/consolidation-runs/"+runID+"/eliminate", strings.NewReader(good))
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", uuid.NewString())
		req.Header.Set("X-Roles", "HOLDING_ADMIN")
		req.Header.Set("X-Permissions", "consolidation:write")
		req.Header.Set("If-Match", `"1"`)
		req.Header.Set("Idempotency-Key", "elim-1")
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 balanced eliminations on call %d, got %d %s", i, rec.Code, rec.Body.String())
		}
	}
	if st.elimCalls != 1 {
		t.Fatalf("expected PostEliminations called once via idempotency, got %d", st.elimCalls)
	}
}
