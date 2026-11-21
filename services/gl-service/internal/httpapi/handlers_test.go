package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/gl-service/internal/gldb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type fakeStore struct {
	entry     gldb.GlJournalEntry
	lines     []gldb.ListJournalLinesRow
	totDebit  string
	totCredit string
	totLines  int64
}

func (f *fakeStore) CreateEntryWithLines(_ context.Context, _ []uuid.UUID, entry gldb.CreateJournalEntryParams, lines []gldb.CreateJournalLineParams) (gldb.CreateJournalEntryRow, error) {
	id := uuid.New()
	f.entry = gldb.GlJournalEntry{
		ID: id, OrganizationID: entry.OrganizationID, JournalNo: entry.JournalNo,
		FinancialOrganizationID: entry.FinancialOrganizationID, LedgerID: entry.LedgerID,
		FiscalPeriodID: entry.FiscalPeriodID, PostingDate: entry.PostingDate,
		JournalSource: entry.JournalSource, CurrencyCode: entry.CurrencyCode,
		Status: "Draft", RowVersion: 1,
	}
	return gldb.CreateJournalEntryRow{ID: id, PostingDate: entry.PostingDate, Status: "Draft", RowVersion: 1}, nil
}

func (f *fakeStore) GetByID(context.Context, []uuid.UUID, uuid.UUID) (gldb.GlJournalEntry, []gldb.ListJournalLinesRow, error) {
	return f.entry, f.lines, nil
}
func (f *fakeStore) ListWithCount(context.Context, []uuid.UUID, gldb.ListJournalEntriesParams) ([]gldb.ListJournalEntriesRow, int64, error) {
	dr, _ := platform.ParseNumeric(f.totDebit)
	cr, _ := platform.ParseNumeric(f.totCredit)
	return []gldb.ListJournalEntriesRow{{
		ID: f.entry.ID, JournalNo: f.entry.JournalNo, PostingDate: f.entry.PostingDate,
		CurrencyCode: "IDR", ExchangeRate: f.entry.ExchangeRate, Status: f.entry.Status, RowVersion: f.entry.RowVersion,
		TotalDebit: dr, TotalCredit: cr,
	}}, 1, nil
}
func (f *fakeStore) UpdateDraftEntry(context.Context, []uuid.UUID, gldb.UpdateJournalEntryDraftParams, []gldb.CreateJournalLineParams) (int64, error) {
	return 1, nil
}
func (f *fakeStore) SoftDeleteDraft(context.Context, []uuid.UUID, gldb.SoftDeleteJournalEntryParams) (int64, error) {
	return 1, nil
}
func (f *fakeStore) Totals(context.Context, []uuid.UUID, gldb.JournalEntryTotalsParams) (gldb.JournalEntryTotalsRow, error) {
	dr, _ := platform.ParseNumeric(f.totDebit)
	cr, _ := platform.ParseNumeric(f.totCredit)
	return gldb.JournalEntryTotalsRow{TotalDebit: dr, TotalCredit: cr, LineCount: f.totLines}, nil
}
func (f *fakeStore) PostAndEmit(context.Context, []uuid.UUID, gldb.GlJournalEntry, int32, uuid.UUID) (int64, error) {
	return 1, nil
}
func (f *fakeStore) ReverseEntryAndEmit(_ context.Context, _ []uuid.UUID, orig gldb.GlJournalEntry, _ int32, _, _ uuid.UUID) (gldb.CreateJournalEntryRow, int64, error) {
	return gldb.CreateJournalEntryRow{ID: uuid.New(), PostingDate: orig.PostingDate, Status: "Posted", RowVersion: 2}, 1, nil
}
func (f *fakeStore) ListAccountBalances(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, int32, int32) ([]gldb.ListAccountBalancesRow, int64, error) {
	return nil, 0, nil
}

func TestCreateRejectsInvalidNumericAndPostEnforcesBR556(t *testing.T) {
	st := &fakeStore{
		entry: gldb.GlJournalEntry{
			ID: uuid.New(), JournalNo: "JV-1", PostingDate: pgtype.Date{Time: time.Now(), Valid: true},
			Status: "Draft", RowVersion: 1, CurrencyCode: "IDR",
		},
		totDebit:  "1000.0000",
		totCredit: "900.0000",
		totLines:  2,
	}
	mux := http.NewServeMux()
	NewServer(st).Register(mux)
	h := httpx.Middleware(mux)

	// Invalid decimal is rejected instead of silently zeroed.
	badBody := `{"journalNo":"JV-BAD","financialOrganizationId":"10000000-0000-0000-0000-000000000001","ledgerId":"10000000-0000-0000-0000-000000000002","fiscalPeriodId":"10000000-0000-0000-0000-000000000003","postingDate":"2026-03-15","currencyCode":"IDR","lines":[{"lineNo":1,"accountId":"10000000-0000-0000-0000-00000000000a","debitAmount":"not-a-num","creditAmount":"0"}]}`
	req := httptest.NewRequest(http.MethodPost, "/journal-entries", bytes.NewBufferString(badBody))
	req.Header.Set("X-Organization-Id", uuid.NewString())
	req.Header.Set("X-User-Id", uuid.NewString())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid decimal status = %d, want 400", rec.Code)
	}

	// Unbalanced post returns 422 GL_JOURNAL_UNBALANCED.
	req = httptest.NewRequest(http.MethodPost, "/journal-entries/"+st.entry.ID.String()+"/post", nil)
	req.Header.Set("X-Organization-Id", uuid.NewString())
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("If-Match", "1")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unbalanced post status = %d, want 422, body=%s", rec.Code, rec.Body.String())
	}

	// Missing permission returns 403 when caller carries RBAC headers.
	req = httptest.NewRequest(http.MethodPost, "/journal-entries/"+st.entry.ID.String()+"/post", nil)
	req.Header.Set("X-Organization-Id", uuid.NewString())
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Roles", "VIEWER")
	req.Header.Set("X-Permissions", "gl.read")
	req.Header.Set("If-Match", "1")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("viewer post status = %d, want 403", rec.Code)
	}

	// Balanced journal passes and returns ETag + rowVersion.
	st.totCredit = "1000.0000"
	req = httptest.NewRequest(http.MethodPost, "/journal-entries/"+st.entry.ID.String()+"/post", nil)
	req.Header.Set("X-Organization-Id", uuid.NewString())
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Roles", "FINANCE_MANAGER")
	req.Header.Set("X-Permissions", "gl.approve")
	req.Header.Set("If-Match", "1")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("balanced post status = %d, want 200", rec.Code)
	}
	var out entryDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.RowVersion != 2 || rec.Header().Get("ETag") != `"2"` {
		t.Fatalf("unexpected rowVersion=%d etag=%q", out.RowVersion, rec.Header().Get("ETag"))
	}
	if out.ExchangeRate != "1.0000" || out.TotalDebit != "1000.0000" || out.TotalCredit != "1000.0000" {
		t.Fatalf("expected post response to include exchangeRate/totalDebit/totalCredit, got %+v", out)
	}

	// GET /journal-entries returns exchangeRate, totalDebit, and totalCredit.
	req = httptest.NewRequest(http.MethodGet, "/journal-entries", nil)
	req.Header.Set("X-Organization-Id", uuid.NewString())
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Permissions", "gl.read")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", rec.Code)
	}
	var listResp struct {
		Data []entryDTO `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if len(listResp.Data) != 1 {
		t.Fatalf("expected 1 journal entry in list, got %d", len(listResp.Data))
	}
	row := listResp.Data[0]
	if row.ExchangeRate != "1.0000" || row.TotalDebit != "1000.0000" || row.TotalCredit != "1000.0000" {
		t.Fatalf("expected list item to include exchangeRate=1.0000 totalDebit=1000.0000 totalCredit=1000.0000, got %+v", row)
	}
}

func TestBuildLineParamsValidationAndFXFunctionalAmount(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	acctID := uuid.NewString()
	rate, _ := platform.ParseNumeric("15850.0000")

	// 1. Reject negative debit/credit
	if _, err := buildLineParams(orgID, userID, rate, []lineReq{
		{LineNo: 1, AccountID: acctID, DebitAmount: "-10", CreditAmount: "0"},
	}); err == nil {
		t.Fatal("expected error for negative debitAmount")
	}

	// 2. Reject both debit and credit > 0 on the same line
	if _, err := buildLineParams(orgID, userID, rate, []lineReq{
		{LineNo: 1, AccountID: acctID, DebitAmount: "10", CreditAmount: "5"},
	}); err == nil {
		t.Fatal("expected error when both debitAmount and creditAmount > 0")
	}

	// 3. Reject duplicate lineNo
	if _, err := buildLineParams(orgID, userID, rate, []lineReq{
		{LineNo: 1, AccountID: acctID, DebitAmount: "10", CreditAmount: "0"},
		{LineNo: 1, AccountID: acctID, DebitAmount: "0", CreditAmount: "10"},
	}); err == nil {
		t.Fatal("expected error for duplicate lineNo")
	}

	// 4. Compute functionalAmount = amount * exchangeRate when omitted
	lines, err := buildLineParams(orgID, userID, rate, []lineReq{
		{LineNo: 1, AccountID: acctID, DebitAmount: "100.0000", CreditAmount: "0"},
		{LineNo: 2, AccountID: acctID, DebitAmount: "0", CreditAmount: "100.0000"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := platform.FormatNumeric(lines[0].FunctionalAmount, 4); got != "1585000.0000" {
		t.Fatalf("functionalAmount = %s, want 1585000.0000", got)
	}
}


