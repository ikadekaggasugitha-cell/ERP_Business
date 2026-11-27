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

	"github.com/sinergiinvestasiproperti/erp/services/ap-service/internal/apdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type stubStore struct {
	inv     apdb.ApApInvoice
	lines   []apdb.ListInvoiceLinesRow
	list    []apdb.ListInvoicesRow
	total   int64
	outbox  apdb.InsertOutboxParams
	deleted int64
	newVer  int32
}

func (s *stubStore) CreateInvoiceWithLines(_ context.Context, _ []uuid.UUID, invoice apdb.CreateInvoiceParams, _ []apdb.CreateInvoiceLineParams) (apdb.CreateInvoiceRow, error) {
	id := uuid.New()
	s.inv = apdb.ApApInvoice{
		ID:                id,
		OrganizationID:    invoice.OrganizationID,
		InvoiceNo:         invoice.InvoiceNo,
		VendorID:          invoice.VendorID,
		InvoiceDate:       invoice.InvoiceDate,
		DueDate:           invoice.DueDate,
		CurrencyCode:      invoice.CurrencyCode,
		TotalAmount:       invoice.TotalAmount,
		OutstandingAmount: invoice.OutstandingAmount,
		Status:            "Draft",
		RowVersion:        1,
	}
	return apdb.CreateInvoiceRow{ID: id, InvoiceDate: invoice.InvoiceDate, Status: "Draft", RowVersion: 1}, nil
}

func (s *stubStore) GetWithLines(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (apdb.ApApInvoice, []apdb.ListInvoiceLinesRow, error) {
	return s.inv, s.lines, nil
}

func (s *stubStore) GetByID(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (apdb.ApApInvoice, error) {
	return s.inv, nil
}

func (s *stubStore) ListWithCount(_ context.Context, _ []uuid.UUID, _ string, _, _ int32) ([]apdb.ListInvoicesRow, int64, error) {
	return s.list, s.total, nil
}

func (s *stubStore) UpdateDraftWithLines(_ context.Context, _ []uuid.UUID, _ apdb.UpdateDraftInvoiceParams, _ []apdb.CreateInvoiceLineParams) (int32, error) {
	if s.newVer == 0 {
		return 2, nil
	}
	return s.newVer, nil
}

func (s *stubStore) DeleteDraft(_ context.Context, _ []uuid.UUID, _ apdb.DeleteDraftInvoiceParams) (int64, error) {
	return s.deleted, nil
}

func (s *stubStore) RecordAndEmit(_ context.Context, _ []uuid.UUID, _ apdb.RecordInvoiceParams, ev apdb.InsertOutboxParams) (int64, error) {
	s.outbox = ev
	return 1, nil
}

func (s *stubStore) ApplyDebitNoteAndEmit(_ context.Context, _ []uuid.UUID, _ apdb.ApplyPaymentToInvoiceParams, ev apdb.InsertOutboxParams) (int64, error) {
	s.outbox = ev
	return 1, nil
}

func TestAPInvoiceLifecycleValidationAndRBAC(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	vendorID := uuid.New()
	acctID := uuid.New()
	dt := pgtype.Date{Time: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	tot, _ := platform.ParseNumeric("100000.0000")

	store := &stubStore{
		list: []apdb.ListInvoicesRow{{
			ID: uuid.New(), InvoiceNo: "AP-001", VendorID: vendorID, InvoiceDate: dt, DueDate: dt,
			Status: "Draft", CurrencyCode: "IDR", TotalAmount: tot, OutstandingAmount: tot, RowVersion: 1,
		}},
		total:   12,
		deleted: 1,
	}

	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	do := func(method, path, body, perms, idem, ifMatch string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("X-Organization-Id", orgID.String())
		req.Header.Set("X-User-Id", userID.String())
		if perms != "" {
			req.Header.Set("X-Permissions", perms)
		}
		if idem != "" {
			req.Header.Set("Idempotency-Key", idem)
		}
		if ifMatch != "" {
			req.Header.Set("If-Match", ifMatch)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// 1. Line sum mismatch returns 422.
	badSum := `{"invoiceNo":"AP-1","vendorId":"` + vendorID.String() + `","invoiceDate":"2026-11-01","dueDate":"2026-11-30","totalAmount":"100000.0000","lines":[{"lineNo":1,"accountId":"` + acctID.String() + `","amount":"90000.0000"}]}`
	if rec := do(http.MethodPost, "/vendor-invoices", badSum, "ap:write", "k-bad", ""); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on line sum mismatch, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Valid create + record enriches outbox with vendorId and dueDate.
	good := `{"invoiceNo":"AP-1","vendorId":"` + vendorID.String() + `","invoiceDate":"2026-11-01","dueDate":"2026-11-30","totalAmount":"100000.0000","lines":[{"lineNo":1,"accountId":"` + acctID.String() + `","amount":"100000.0000"}]}`
	rec := do(http.MethodPost, "/vendor-invoices", good, "ap:write", "k-ok", "")
	if rec.Code != http.StatusCreated || rec.Header().Get("ETag") != `"1"` {
		t.Fatalf("create failed: %d %s", rec.Code, rec.Body.String())
	}
	var created invoiceDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	// Reusing Idempotency-Key with mutated payload conflicts (409).
	diff := `{"invoiceNo":"AP-2","vendorId":"` + vendorID.String() + `","invoiceDate":"2026-11-01","dueDate":"2026-11-30","totalAmount":"100000.0000","lines":[{"lineNo":1,"accountId":"` + acctID.String() + `","amount":"100000.0000"}]}`
	if r409 := do(http.MethodPost, "/vendor-invoices", diff, "ap:write", "k-ok", ""); r409.Code != http.StatusConflict {
		t.Fatalf("expected 409 idempotency mismatch, got %d", r409.Code)
	}

	// Record requires ap.approve (ap:write alone is rejected) and enforces SoD when CreatedBy == caller.
	if rNoApprove := do(http.MethodPost, "/vendor-invoices/"+created.ID+"/record", "", "ap:write", "", `"1"`); rNoApprove.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without ap.approve, got %d", rNoApprove.Code)
	}
	store.inv.CreatedBy = userID
	if rSoD := do(http.MethodPost, "/vendor-invoices/"+created.ID+"/record", "", "ap:write,ap.approve", "", `"1"`); rSoD.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for maker-checker SoD self-approval, got %d", rSoD.Code)
	}
	store.inv.CreatedBy = uuid.New()

	rec = do(http.MethodPost, "/vendor-invoices/"+created.ID+"/record", "", "ap:write,ap.approve", "", `"1"`)
	if rec.Code != http.StatusOK {
		t.Fatalf("record failed: %d %s", rec.Code, rec.Body.String())
	}
	var evPayload map[string]any
	_ = json.Unmarshal(store.outbox.Payload, &evPayload)
	if evPayload["vendorId"] != vendorID.String() || evPayload["dueDate"] != "2026-11-30" {
		t.Fatalf("outbox missing vendorId/dueDate: %+v", evPayload)
	}

	// 3. List returns real pagination totalElements & rowVersion; RBAC blocks write on read-only token.
	if rList := do(http.MethodGet, "/vendor-invoices", "", "ap:read", "", ""); rList.Code != http.StatusOK || !strings.Contains(rList.Body.String(), `"totalElements":12`) {
		t.Fatalf("unexpected list response: %d %s", rList.Code, rList.Body.String())
	}
	if rForbid := do(http.MethodDelete, "/vendor-invoices/"+created.ID, "", "ap:read", "", `"1"`); rForbid.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without ap:write, got %d", rForbid.Code)
	}
	if rDel := do(http.MethodDelete, "/vendor-invoices/"+created.ID, "", "ap:write", "", `"1"`); rDel.Code != http.StatusNoContent {
		t.Fatalf("expected 204 delete, got %d", rDel.Code)
	}
}

func TestAPAgingAndDebitNote(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	vendorID := uuid.New()
	asOf := time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)
	mkAmt := func(s string) pgtype.Numeric { n, _ := platform.ParseNumeric(s); return n }
	mkDate := func(daysAgo int) pgtype.Date {
		return pgtype.Date{Time: asOf.AddDate(0, 0, -daysAgo), Valid: true}
	}

	invID := uuid.New()
	store := &stubStore{
		inv: apdb.ApApInvoice{
			ID:                invID,
			OrganizationID:    orgID,
			InvoiceNo:         "AP-100",
			VendorID:          vendorID,
			InvoiceDate:       mkDate(20),
			DueDate:           mkDate(10),
			CurrencyCode:      "IDR",
			TotalAmount:       mkAmt("500.0000"),
			OutstandingAmount: mkAmt("500.0000"),
			Status:            "Recorded",
			RowVersion:        1,
		},
		list: []apdb.ListInvoicesRow{
			{ID: uuid.New(), Status: "Recorded", DueDate: mkDate(-5), OutstandingAmount: mkAmt("100.0000")},  // current
			{ID: uuid.New(), Status: "Recorded", DueDate: mkDate(15), OutstandingAmount: mkAmt("200.0000")},  // 1-30
			{ID: uuid.New(), Status: "Recorded", DueDate: mkDate(45), OutstandingAmount: mkAmt("300.0000")},  // 31-60
			{ID: uuid.New(), Status: "Recorded", DueDate: mkDate(75), OutstandingAmount: mkAmt("400.0000")},  // 61-90
			{ID: uuid.New(), Status: "Recorded", DueDate: mkDate(120), OutstandingAmount: mkAmt("500.0000")}, // 90+
			{ID: uuid.New(), Status: "Paid", DueDate: mkDate(120), OutstandingAmount: mkAmt("999.0000")},     // excluded
		},
	}

	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	// 1. Aging buckets
	req := httptest.NewRequest(http.MethodGet, "/vendor-invoices/aging?asOfDate=2026-11-24", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Permissions", "ap:read")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("aging got %d: %s", rec.Code, rec.Body.String())
	}
	var ag map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &ag)
	if ag["current"] != "100.0000" || ag["days1To30"] != "200.0000" || ag["days31To60"] != "300.0000" ||
		ag["days61To90"] != "400.0000" || ag["days90Plus"] != "500.0000" || ag["totalOutstanding"] != "1500.0000" {
		t.Fatalf("unexpected aging buckets: %+v", ag)
	}

	// 2. Debit note exceeding outstanding -> 422
	dnOver := httptest.NewRequest(http.MethodPost, "/vendor-invoices/"+invID.String()+"/debit-note", strings.NewReader(`{"amount":"600.0000","reason":"return"}`))
	dnOver.Header.Set("X-Organization-Id", orgID.String())
	dnOver.Header.Set("X-User-Id", userID.String())
	dnOver.Header.Set("X-Permissions", "ap:write")
	dnOverRec := httptest.NewRecorder()
	h.ServeHTTP(dnOverRec, dnOver)
	if dnOverRec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for debit note > outstanding, got %d", dnOverRec.Code)
	}

	// 3. Valid debit note -> 201 + outbox event ap.debit_note.issued
	dnOK := httptest.NewRequest(http.MethodPost, "/vendor-invoices/"+invID.String()+"/debit-note", strings.NewReader(`{"amount":"150.0000","reason":"damaged goods"}`))
	dnOK.Header.Set("X-Organization-Id", orgID.String())
	dnOK.Header.Set("X-User-Id", userID.String())
	dnOK.Header.Set("X-Permissions", "ap:write")
	dnOK.Header.Set("Idempotency-Key", "dn-key-1")
	dnOKRec := httptest.NewRecorder()
	h.ServeHTTP(dnOKRec, dnOK)
	if dnOKRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for valid debit note, got %d: %s", dnOKRec.Code, dnOKRec.Body.String())
	}
	if store.outbox.EventType != "ap.debit_note.issued" {
		t.Fatalf("expected ap.debit_note.issued, got %q", store.outbox.EventType)
	}
}

