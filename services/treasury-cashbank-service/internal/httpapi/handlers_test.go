package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/treasury-cashbank-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/treasury-cashbank-service/internal/treasurydb"
)

type fakeTreasuryStore struct {
	settledEvents []treasurydb.InsertOutboxParams
}

func (f *fakeTreasuryStore) CreateBankAccount(_ context.Context, _ []uuid.UUID, _ treasurydb.CreateBankAccountParams) (treasurydb.CreateBankAccountRow, error) {
	return treasurydb.CreateBankAccountRow{ID: uuid.MustParse("10000000-0000-0000-0000-000000000101"), RowVersion: 1}, nil
}

func (f *fakeTreasuryStore) GetBankAccount(_ context.Context, _ []uuid.UUID, id uuid.UUID) (treasurydb.TrsBankAccount, error) {
	return treasurydb.TrsBankAccount{ID: id, Code: "BCA-01", BankName: "BCA", CurrencyCode: "IDR", RowVersion: 1}, nil
}

func (f *fakeTreasuryStore) ListBanksWithCount(_ context.Context, scope []uuid.UUID, _ treasurydb.ListBankAccountsParams) ([]treasurydb.ListBankAccountsRow, int64, error) {
	return []treasurydb.ListBankAccountsRow{{
		ID:             uuid.MustParse("10000000-0000-0000-0000-000000000101"),
		OrganizationID: scope[0],
		Code:           "BCA-01",
		BankName:       "BCA",
		AccountNo:      "123456",
		CurrencyCode:   "IDR",
		RowVersion:     1,
	}}, 1, nil
}

func (f *fakeTreasuryStore) CreatePayment(_ context.Context, _ []uuid.UUID, _ treasurydb.CreatePaymentParams, _ []treasurydb.CreatePaymentAllocationParams) (treasurydb.CreatePaymentRow, error) {
	return treasurydb.CreatePaymentRow{ID: uuid.MustParse("20000000-0000-0000-0000-000000000201"), Status: "Draft", RowVersion: 1}, nil
}

func (f *fakeTreasuryStore) GetWithAllocations(_ context.Context, scope []uuid.UUID, id uuid.UUID) (treasurydb.TrsPayment, []treasurydb.ListPaymentAllocationsRow, error) {
	amt, _ := platform.ParseNumeric("500.0000")
	dt, _ := platform.ParseDate("2026-03-15")
	return treasurydb.TrsPayment{
		ID:             id,
		OrganizationID: scope[0],
		PaymentNo:      "PAY-001",
		PaymentType:    "Outgoing",
		BankAccountID:  uuid.MustParse("10000000-0000-0000-0000-000000000101"),
		Amount:         amt,
		PaymentDate:    dt,
		Status:         "Draft",
		RowVersion:     1,
	}, nil, nil
}

func (f *fakeTreasuryStore) ListWithCount(_ context.Context, scope []uuid.UUID, _ treasurydb.ListPaymentsParams) ([]treasurydb.ListPaymentsRow, int64, error) {
	amt, _ := platform.ParseNumeric("500.0000")
	dt, _ := platform.ParseDate("2026-03-15")
	return []treasurydb.ListPaymentsRow{{
		ID:             uuid.MustParse("20000000-0000-0000-0000-000000000201"),
		OrganizationID: scope[0],
		PaymentNo:      "PAY-001",
		PaymentType:    "Outgoing",
		Amount:         amt,
		PaymentDate:    dt,
		Status:         "Settled",
		RowVersion:     2,
	}}, 1, nil
}

func (f *fakeTreasuryStore) UpdateDraft(_ context.Context, _ []uuid.UUID, _ treasurydb.UpdateDraftPaymentParams) (int32, error) {
	return 2, nil
}

func (f *fakeTreasuryStore) DeleteDraft(_ context.Context, _ []uuid.UUID, _ treasurydb.DeleteDraftPaymentParams) (int64, error) {
	return 1, nil
}

func (f *fakeTreasuryStore) SettleAndEmit(_ context.Context, _ []uuid.UUID, _ treasurydb.SettlePaymentParams, events ...treasurydb.InsertOutboxParams) (int32, error) {
	f.settledEvents = append(f.settledEvents, events...)
	return 2, nil
}

func (f *fakeTreasuryStore) CreateStatementLine(_ context.Context, _ []uuid.UUID, _ treasurydb.CreateBankStatementLineParams) (treasurydb.CreateBankStatementLineRow, error) {
	return treasurydb.CreateBankStatementLineRow{ID: uuid.MustParse("30000000-0000-0000-0000-000000000301"), IsReconciled: false, RowVersion: 1}, nil
}

func (f *fakeTreasuryStore) ListStatementLinesWithCount(_ context.Context, _ []uuid.UUID, _ treasurydb.ListBankStatementLinesParams) ([]treasurydb.ListBankStatementLinesRow, int64, error) {
	return nil, 0, nil
}

func (f *fakeTreasuryStore) ReconcileBankAccountAndEmit(_ context.Context, _ []uuid.UUID, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID) ([]repository.ReconcileMatch, error) {
	return []repository.ReconcileMatch{{
		StatementLineID: uuid.MustParse("30000000-0000-0000-0000-000000000301"),
		PaymentID:       uuid.MustParse("20000000-0000-0000-0000-000000000201"),
		PaymentNo:       "PAY-001",
		Amount:          "500.0000",
	}}, nil
}

func authHeaders(perms string) map[string]string {
	return map[string]string{
		"X-Org-Scope":   "00000000-0000-0000-0000-000000000001",
		"X-User-ID":     "00000000-0000-0000-0000-000000000010",
		"X-Permissions": perms,
	}
}

func doReq(t *testing.T, h http.Handler, method, path string, body any, hdrs map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestTreasuryEndpointsRBACPaginationSettleAndReconcile(t *testing.T) {
	store := &fakeTreasuryStore{}
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	rr := doReq(t, h, http.MethodGet, "/payments", nil, authHeaders("ap:read"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing treasury:read, got %d", rr.Code)
	}

	rr = doReq(t, h, http.MethodPost, "/payments", map[string]any{
		"paymentNo":     "PAY-BAD",
		"paymentType":   "Outgoing",
		"bankAccountId": "10000000-0000-0000-0000-000000000101",
		"amount":        "100.0000",
		"allocations":   []map[string]any{{"targetDocumentRef": "INV-1", "allocatedAmount": "150.0000"}},
	}, authHeaders("treasury:write"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when allocations exceed payment amount, got %d (%s)", rr.Code, rr.Body.String())
	}

	hdrs := authHeaders("treasury:read,treasury:write")
	hdrs["If-Match"] = `"1"`
	rr = doReq(t, h, http.MethodPost, "/payments/20000000-0000-0000-0000-000000000201/settle", nil, hdrs)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on settle, got %d (%s)", rr.Code, rr.Body.String())
	}
	if len(store.settledEvents) != 2 ||
		store.settledEvents[0].EventType != "treasury.payment.settled" ||
		store.settledEvents[1].EventType != "cashbank.payment.settled" {
		t.Fatalf("expected both treasury.payment.settled and cashbank.payment.settled, got %+v", store.settledEvents)
	}

	rr = doReq(t, h, http.MethodPost, "/bank-accounts/10000000-0000-0000-0000-000000000101/reconcile", nil, hdrs)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on reconcile, got %d (%s)", rr.Code, rr.Body.String())
	}
	var rec map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &rec)
	if rec["matchedCount"] != float64(1) {
		t.Fatalf("expected matchedCount=1, got %v", rec)
	}

	rr = doReq(t, h, http.MethodGet, "/payments", nil, hdrs)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on listPayments, got %d", rr.Code)
	}
	var env httpx.ListEnvelope
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Page.TotalElements != 1 {
		t.Fatalf("expected totalElements=1, got %+v", env.Page)
	}

	// Single-line bank statement creation
	rr = doReq(t, h, http.MethodPost, "/bank-statement-lines", map[string]any{
		"bankAccountId": "10000000-0000-0000-0000-000000000101",
		"statementDate": "2026-03-15",
		"description":   "Incoming customer transfer",
		"amount":        "1500.0000",
	}, hdrs)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 on single-line createStatementLine, got %d (%s)", rr.Code, rr.Body.String())
	}

	// Chrome Companion batch bank statement creation
	rr = doReq(t, h, http.MethodPost, "/bank-statement-lines", map[string]any{
		"lines": []map[string]any{
			{
				"date":        "2026-03-16",
				"description": "BI-FAST Incoming",
				"debit":       0,
				"credit":      2500000,
				"balance":     12500000,
			},
			{
				"date":        "17/03/2026",
				"description": "Vendor Payment",
				"debit":       500000,
				"credit":      0,
				"balance":     12000000,
			},
		},
		"bankName":          "BCA",
		"accountNumber":     "123456",
		"origin":            "chrome-extension",
		"extension_version": "1.0.0",
	}, hdrs)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 on Chrome Companion batch createStatementLine, got %d (%s)", rr.Code, rr.Body.String())
	}
	var batchResp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &batchResp); err != nil {
		t.Fatalf("unmarshal batch response: %v", err)
	}
	if batchResp["importedCount"] != float64(2) {
		t.Fatalf("expected importedCount=2, got %+v", batchResp)
	}
}

func TestMatchStatementLinesMatchingRules(t *testing.T) {
	amt1, _ := platform.ParseNumeric("500.0000")
	negAmt1, _ := platform.ParseNumeric("-500.0000")
	amt2, _ := platform.ParseNumeric("750.0000")
	ext := "PAY-002"
	l1 := uuid.New()
	l2 := uuid.New()
	p1 := uuid.New()
	p2 := uuid.New()

	matches := repository.MatchStatementLines(
		[]treasurydb.ListUnreconciledStatementLinesRow{
			{ID: l1, Amount: negAmt1, Description: "Outgoing bank debit"},
			{ID: l2, Amount: amt2, ExternalRef: &ext, Description: "Customer receipt"},
		},
		[]treasurydb.ListSettledPaymentsForBankRow{
			{ID: p1, PaymentNo: "PAY-001", PaymentType: "Outgoing", Amount: amt1, PaymentDate: pgtype.Date{Valid: true}},
			{ID: p2, PaymentNo: "PAY-002", PaymentType: "Incoming", Amount: amt2, PaymentDate: pgtype.Date{Valid: true}},
		},
	)
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
}

