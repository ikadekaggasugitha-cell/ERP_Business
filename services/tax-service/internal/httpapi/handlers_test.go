package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/tax-service/internal/taxdb"
)

type stubStore struct {
	code   taxdb.TaxTaxCode
	txn    taxdb.TaxTaxTransaction
	outbox taxdb.InsertOutboxParams
}

func (s *stubStore) CreateTaxCode(_ context.Context, _ []uuid.UUID, _ taxdb.CreateTaxCodeParams) (taxdb.CreateTaxCodeRow, error) {
	return taxdb.CreateTaxCodeRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (s *stubStore) GetTaxCode(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (taxdb.TaxTaxCode, error) {
	return s.code, nil
}
func (s *stubStore) ListTaxCodesWithCount(_ context.Context, _ []uuid.UUID, _, _ int32) ([]taxdb.ListTaxCodesRow, int64, error) {
	return nil, 0, nil
}
func (s *stubStore) CreateTransactionAndEmit(_ context.Context, _ []uuid.UUID, txn taxdb.CreateTaxTransactionParams, ev taxdb.InsertOutboxParams) (taxdb.CreateTaxTransactionRow, error) {
	id := uuid.New()
	s.txn = taxdb.TaxTaxTransaction{
		ID: id, OrganizationID: txn.OrganizationID, SourceDocumentRef: txn.SourceDocumentRef,
		TaxCodeID: txn.TaxCodeID, BaseAmount: txn.BaseAmount, TaxAmount: txn.TaxAmount,
		EfakturNo: txn.EfakturNo, EfakturStatus: txn.EfakturStatus, RowVersion: 1,
	}
	s.outbox = ev
	return taxdb.CreateTaxTransactionRow{ID: id, RowVersion: 1}, nil
}
func (s *stubStore) GetByID(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (taxdb.TaxTaxTransaction, error) {
	return s.txn, nil
}
func (s *stubStore) ListTxnWithCount(_ context.Context, _ []uuid.UUID, _ string, _, _ int32) ([]taxdb.ListTaxTransactionsRow, int64, error) {
	return nil, 0, nil
}
func (s *stubStore) ReportEfakturAndEmit(_ context.Context, _ []uuid.UUID, _ taxdb.ReportTaxTransactionParams, ev taxdb.InsertOutboxParams) (int32, error) {
	s.outbox = ev
	return 2, nil
}

func TestTaxAutoComputationAndEfakturReport(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	rate, _ := platform.ParseNumeric("11.0000")
	store := &stubStore{
		code: taxdb.TaxTaxCode{ID: uuid.New(), Code: "PPN11", Rate: rate, TaxType: "VAT"},
	}

	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	body := `{"sourceDocumentRef":"AR-100","taxCodeId":"` + store.code.ID.String() + `","baseAmount":"1000000.0000","efakturNo":"010.001-26.00000001"}`
	req := httptest.NewRequest(http.MethodPost, "/tax-invoices", strings.NewReader(body))
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "tax:write")
	req.Header.Set("Idempotency-Key", "tx-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create tax invoice failed: %d %s", rec.Code, rec.Body.String())
	}
	var created taxInvoiceDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.TaxAmount != "110000.0000" {
		t.Fatalf("expected computed 11%% VAT 110000.0000, got %s", created.TaxAmount)
	}

	req = httptest.NewRequest(http.MethodPost, "/tax-invoices/"+created.ID+"/report", strings.NewReader(`{"ntte":"NTTE-9988"}`))
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "tax:write")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || store.outbox.EventType != "tax.efaktur.reported" {
		t.Fatalf("report failed: %d outbox=%s body=%s", rec.Code, store.outbox.EventType, rec.Body.String())
	}
}

