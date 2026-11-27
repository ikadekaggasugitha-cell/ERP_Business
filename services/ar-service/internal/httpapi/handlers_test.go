package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/ar-service/internal/ardb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type stubStore struct {
	inv    ardb.ArArInvoice
	lines  []ardb.ListInvoiceLinesRow
	list   []ardb.ListInvoicesRow
	total  int64
	outbox ardb.InsertOutboxParams
}

func (s *stubStore) CreateInvoiceWithLinesAndEmit(_ context.Context, _ []uuid.UUID, inv ardb.CreateInvoiceParams, _ []ardb.CreateInvoiceLineParams, ev ardb.InsertOutboxParams) (ardb.CreateInvoiceRow, error) {
	id := uuid.New()
	s.inv = ardb.ArArInvoice{
		ID: id, OrganizationID: inv.OrganizationID, InvoiceNo: inv.InvoiceNo, CustomerID: inv.CustomerID,
		InvoiceDate: inv.InvoiceDate, DueDate: inv.DueDate, CurrencyCode: inv.CurrencyCode,
		TotalAmount: inv.TotalAmount, OutstandingAmount: inv.OutstandingAmount, Status: "Issued", RowVersion: 1,
	}
	s.outbox = ev
	return ardb.CreateInvoiceRow{ID: id, InvoiceDate: inv.InvoiceDate, Status: "Issued", RowVersion: 1}, nil
}
func (s *stubStore) GetWithLines(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (ardb.ArArInvoice, []ardb.ListInvoiceLinesRow, error) {
	return s.inv, s.lines, nil
}
func (s *stubStore) GetByID(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (ardb.ArArInvoice, error) {
	return s.inv, nil
}
func (s *stubStore) ListWithCount(_ context.Context, _ []uuid.UUID, _ string, _, _ int32) ([]ardb.ListInvoicesRow, int64, error) {
	return s.list, s.total, nil
}
func (s *stubStore) MarkPaid(_ context.Context, _ []uuid.UUID, _ ardb.MarkInvoicePaidParams, ev ardb.InsertOutboxParams) (int64, error) {
	s.outbox = ev
	return 1, nil
}
func (s *stubStore) DeleteDraft(_ context.Context, _ []uuid.UUID, _ ardb.DeleteDraftInvoiceParams) (int64, error) {
	return 1, nil
}
func (s *stubStore) UpdateDraftWithLines(_ context.Context, _ []uuid.UUID, _ ardb.UpdateDraftInvoiceParams, _ []ardb.CreateInvoiceLineParams) (int32, error) {
	return 2, nil
}
func (s *stubStore) ApplyCreditNoteAndEmit(_ context.Context, _ []uuid.UUID, arg ardb.ApplyReceiptToInvoiceParams, ev ardb.InsertOutboxParams) (ardb.ApplyReceiptToInvoiceRow, error) {
	s.outbox = ev
	rem, _ := platform.ParseNumeric("350.0000")
	return ardb.ApplyReceiptToInvoiceRow{
		ID: arg.ID, OutstandingAmount: rem, Status: "PartiallyPaid",
	}, nil
}
func (s *stubStore) WriteOffAndEmit(_ context.Context, _ []uuid.UUID, _ ardb.WriteOffInvoiceParams, ev ardb.InsertOutboxParams) (int64, error) {
	s.outbox = ev
	return 1, nil
}

func TestARInvoiceValidationAndEventEnrichment(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	custID := uuid.New()
	acctID := uuid.New()
	dt := pgtype.Date{Time: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	tot, _ := platform.ParseNumeric("200000.0000")

	store := &stubStore{
		list: []ardb.ListInvoicesRow{{
			ID: uuid.New(), InvoiceNo: "AR-01", CustomerID: custID, InvoiceDate: dt, DueDate: dt,
			Status: "Issued", CurrencyCode: "IDR", TotalAmount: tot, OutstandingAmount: tot, RowVersion: 1,
		}},
		total: 9,
	}
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	badSum := `{"invoiceNo":"AR-1","customerId":"` + custID.String() + `","invoiceDate":"2026-11-01","dueDate":"2026-11-15","totalAmount":"200000","lines":[{"lineNo":1,"accountId":"` + acctID.String() + `","amount":"150000"}]}`
	req := httptest.NewRequest(http.MethodPost, "/customer-invoices", strings.NewReader(badSum))
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "ar:write")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on line mismatch, got %d", rec.Code)
	}

	good := `{"invoiceNo":"AR-1","customerId":"` + custID.String() + `","invoiceDate":"2026-11-01","dueDate":"2026-11-15","totalAmount":"200000.0000","taxAmount":"22000.0000","lines":[{"lineNo":1,"accountId":"` + acctID.String() + `","amount":"200000.0000"}]}`
	req = httptest.NewRequest(http.MethodPost, "/customer-invoices", strings.NewReader(good))
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "ar:write")
	req.Header.Set("Idempotency-Key", "ar-1")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	_ = json.Unmarshal(store.outbox.Payload, &payload)
	if payload["customerId"] != custID.String() || payload["taxAmount"] != "22000.0000" {
		t.Fatalf("enriched ar.invoice.issued missing fields: %+v", payload)
	}

	// DELETE on Issued invoice must return 422 INVOICE_NOT_DRAFT.
	req = httptest.NewRequest(http.MethodDelete, "/customer-invoices/"+store.inv.ID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "ar:write")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "INVOICE_NOT_DRAFT") {
		t.Fatalf("expected 422 INVOICE_NOT_DRAFT on delete Issued, got %d: %s", rec.Code, rec.Body.String())
	}

	// POST /pay requires ar.approve and enforces SoD.
	store.inv.CreatedBy = userID
	req = httptest.NewRequest(http.MethodPost, "/customer-invoices/"+store.inv.ID.String()+"/pay", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "ar:write")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without ar.approve, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/customer-invoices/"+store.inv.ID.String()+"/pay", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", userID.String())
	req.Header.Set("X-Permissions", "ar:write,ar.approve")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for maker-checker SoD self-pay, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/customer-invoices/"+store.inv.ID.String()+"/pay", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Permissions", "ar:write,ar.approve")
	req.Header.Set("If-Match", `"1"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on approved pay by different user, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestARAgingUpdateCreditNoteAndWriteOff(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	custID := uuid.New()
	invID := uuid.New()
	asOf := time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)
	mkAmt := func(s string) pgtype.Numeric { n, _ := platform.ParseNumeric(s); return n }
	mkDate := func(daysAgo int) pgtype.Date {
		return pgtype.Date{Time: asOf.AddDate(0, 0, -daysAgo), Valid: true}
	}

	store := &stubStore{
		inv: ardb.ArArInvoice{
			ID:                invID,
			OrganizationID:    orgID,
			InvoiceNo:         "AR-200",
			CustomerID:        custID,
			InvoiceDate:       mkDate(20),
			DueDate:           mkDate(10),
			CurrencyCode:      "IDR",
			TotalAmount:       mkAmt("500.0000"),
			OutstandingAmount: mkAmt("500.0000"),
			Status:            "Issued",
			RowVersion:        1,
		},
		list: []ardb.ListInvoicesRow{
			{ID: uuid.New(), Status: "Issued", DueDate: mkDate(-2), OutstandingAmount: mkAmt("100.0000")},
			{ID: uuid.New(), Status: "Issued", DueDate: mkDate(20), OutstandingAmount: mkAmt("200.0000")},
			{ID: uuid.New(), Status: "Overdue", DueDate: mkDate(50), OutstandingAmount: mkAmt("300.0000")},
			{ID: uuid.New(), Status: "Overdue", DueDate: mkDate(80), OutstandingAmount: mkAmt("400.0000")},
			{ID: uuid.New(), Status: "Overdue", DueDate: mkDate(110), OutstandingAmount: mkAmt("500.0000")},
		},
	}

	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	// 1. Aging
	req := httptest.NewRequest(http.MethodGet, "/customer-invoices/aging?asOfDate=2026-11-24", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Permissions", "ar:read")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("aging got %d: %s", rec.Code, rec.Body.String())
	}
	var ag map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &ag)
	if ag["totalOutstanding"] != "1500.0000" || ag["days90Plus"] != "500.0000" {
		t.Fatalf("unexpected AR aging: %+v", ag)
	}

	// 2. PUT /customer-invoices/{id}
	putReq := httptest.NewRequest(http.MethodPut, "/customer-invoices/"+invID.String(), strings.NewReader(`{"dueDate":"2026-12-31","totalAmount":"500.0000"}`))
	putReq.Header.Set("X-Organization-Id", orgID.String())
	putReq.Header.Set("X-User-Id", userID.String())
	putReq.Header.Set("X-Permissions", "ar:write")
	putReq.Header.Set("If-Match", `"1"`)
	putRec := httptest.NewRecorder()
	h.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT customer-invoice got %d: %s", putRec.Code, putRec.Body.String())
	}

	// 3. POST /customer-invoices/{id}/credit-note
	cnReq := httptest.NewRequest(http.MethodPost, "/customer-invoices/"+invID.String()+"/credit-note", strings.NewReader(`{"amount":"150.0000","reason":"discount"}`))
	cnReq.Header.Set("X-Organization-Id", orgID.String())
	cnReq.Header.Set("X-User-Id", userID.String())
	cnReq.Header.Set("X-Permissions", "ar:write")
	cnReq.Header.Set("Idempotency-Key", "cn-1")
	cnRec := httptest.NewRecorder()
	h.ServeHTTP(cnRec, cnReq)
	if cnRec.Code != http.StatusCreated || store.outbox.EventType != "ar.credit_note.issued" {
		t.Fatalf("credit-note got %d ev=%q body=%s", cnRec.Code, store.outbox.EventType, cnRec.Body.String())
	}

	// 4. POST /customer-invoices/{id}/write-off
	woReq := httptest.NewRequest(http.MethodPost, "/customer-invoices/"+invID.String()+"/write-off", strings.NewReader(`{"reason":"insolvent customer"}`))
	woReq.Header.Set("X-Organization-Id", orgID.String())
	woReq.Header.Set("X-User-Id", userID.String())
	woReq.Header.Set("X-Permissions", "ar.approve")
	woReq.Header.Set("If-Match", `"1"`)
	woReq.Header.Set("Idempotency-Key", "wo-1")
	woRec := httptest.NewRecorder()
	h.ServeHTTP(woRec, woReq)
	if woRec.Code != http.StatusOK || store.outbox.EventType != "ar.invoice.written_off" {
		t.Fatalf("write-off got %d ev=%q body=%s", woRec.Code, store.outbox.EventType, woRec.Body.String())
	}
}

