package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/tax-service/internal/taxdb"
)

type stubStore struct {
	code   taxdb.TaxTaxCode
	txn    taxdb.TaxTaxTransaction
	codes  []taxdb.ListTaxCodesRow
	txns   []taxdb.ListTaxTransactionsRow
	outbox taxdb.InsertOutboxParams
}

func (s *stubStore) CreateTaxCode(_ context.Context, _ []uuid.UUID, _ taxdb.CreateTaxCodeParams) (taxdb.CreateTaxCodeRow, error) {
	return taxdb.CreateTaxCodeRow{ID: uuid.New(), RowVersion: 1}, nil
}
func (s *stubStore) GetTaxCode(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (taxdb.TaxTaxCode, error) {
	return s.code, nil
}
func (s *stubStore) ListTaxCodesWithCount(_ context.Context, _ []uuid.UUID, _, _ int32) ([]taxdb.ListTaxCodesRow, int64, error) {
	return s.codes, int64(len(s.codes)), nil
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
	return s.txns, int64(len(s.txns)), nil
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

	// Missing tax.approve returns 403
	req = httptest.NewRequest(http.MethodPost, "/tax-invoices/"+created.ID+"/report", strings.NewReader(`{"ntte":"NTTE-9988"}`))
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "tax:write")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without tax.approve, got %d", rec.Code)
	}

	// Maker-checker SoD blocks self-reporting
	store.txn.CreatedBy = userID
	req = httptest.NewRequest(http.MethodPost, "/tax-invoices/"+created.ID+"/report", strings.NewReader(`{"ntte":"NTTE-9988"}`))
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "tax:write,tax.approve")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for maker-checker SoD self-report, got %d", rec.Code)
	}

	// Empty efakturNo is rejected with 400
	store.txn.CreatedBy = uuid.New()
	savedEfaktur := store.txn.EfakturNo
	store.txn.EfakturNo = nil
	req = httptest.NewRequest(http.MethodPost, "/tax-invoices/"+created.ID+"/report", strings.NewReader(`{"ntte":"NTTE-9988"}`))
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "tax:write,tax.approve")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty efakturNo, got %d", rec.Code)
	}
	store.txn.EfakturNo = savedEfaktur

	// Valid report by different approver succeeds
	req = httptest.NewRequest(http.MethodPost, "/tax-invoices/"+created.ID+"/report", strings.NewReader(`{"ntte":"NTTE-9988"}`))
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "tax:write,tax.approve")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || store.outbox.EventType != "tax.efaktur.reported" {
		t.Fatalf("report failed: %d outbox=%s body=%s", rec.Code, store.outbox.EventType, rec.Body.String())
	}

	// Re-reporting an already Reported tax invoice returns 409
	reported := "Reported"
	store.txn.EfakturStatus = &reported
	req = httptest.NewRequest(http.MethodPost, "/tax-invoices/"+created.ID+"/report", strings.NewReader(`{"ntte":"NTTE-9988"}`))
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "tax:write,tax.approve")
	req.Header.Set("If-Match", `"2"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 on re-reporting Reported invoice, got %d", rec.Code)
	}
}

func TestTaxEfakturExportSptPPNAndWithholding(t *testing.T) {
	orgID := uuid.New()
	outCodeID := uuid.New()
	inCodeID := uuid.New()
	mkAmt := func(s string) pgtype.Numeric { n, _ := platform.ParseNumeric(s); return n }
	efNo := "=010.000-26.00000001"
	st := "Reported"

	store := &stubStore{
		codes: []taxdb.ListTaxCodesRow{
			{ID: outCodeID, Code: "PPN_OUT", TaxType: "VAT_OUT", Rate: mkAmt("11.0000")},
			{ID: inCodeID, Code: "PPN_IN", TaxType: "VAT_IN", Rate: mkAmt("11.0000")},
		},
		txns: []taxdb.ListTaxTransactionsRow{
			{ID: uuid.New(), TaxCodeID: outCodeID, SourceDocumentRef: "AR-001", BaseAmount: mkAmt("1000000.0000"), TaxAmount: mkAmt("110000.0000"), EfakturNo: &efNo, EfakturStatus: &st},
			{ID: uuid.New(), TaxCodeID: inCodeID, SourceDocumentRef: "AP-001", BaseAmount: mkAmt("400000.0000"), TaxAmount: mkAmt("44000.0000"), EfakturNo: &efNo, EfakturStatus: &st},
		},
	}

	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	// 1. e-Faktur CSV export with formula neutralization
	req := httptest.NewRequest(http.MethodGet, "/tax-invoices/export/efaktur", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Permissions", "tax:read")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("efaktur export got %d ct=%q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "'=010.000-26.00000001") {
		t.Fatalf("expected formula-neutralized efakturNo in CSV: %s", rec.Body.String())
	}

	// 2. SPT Masa PPN summary
	req = httptest.NewRequest(http.MethodGet, "/tax-invoices/spt-ppn?period=2026-11", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Permissions", "tax:read")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("spt-ppn got %d: %s", rec.Code, rec.Body.String())
	}
	var spt map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &spt)
	if spt["pajakKeluaran"] != "110000.0000" || spt["pajakMasukan"] != "44000.0000" || spt["kurangLebihBayar"] != "66000.0000" {
		t.Fatalf("unexpected SPT PPN: %+v", spt)
	}

	// 3. PPh 23 non-NPWP surcharge (2% * 2 = 4% -> 40000 on 1000000)
	req = httptest.NewRequest(http.MethodPost, "/withholding-tax/calculate", strings.NewReader(`{"taxType":"PPH23","category":"SERVICES","grossAmount":"1000000.0000","hasNpwp":false}`))
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Permissions", "tax:read")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("withholding calc got %d: %s", rec.Code, rec.Body.String())
	}
	var wh map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &wh)
	if wh["effectiveRatePercent"] != "4.0000" || wh["withholdingAmount"] != "40000.0000" || wh["netAmount"] != "960000.0000" {
		t.Fatalf("unexpected PPh 23 calc: %+v", wh)
	}
}

