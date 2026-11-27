package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

func TestGLFXRevaluationAndRecurringJournals(t *testing.T) {
	store := &fakeStore{}
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	orgID := uuid.NewString()
	userID := uuid.NewString()
	ledgerID := uuid.NewString()
	periodID := uuid.NewString()
	usdBankAcct := uuid.NewString()
	eurApAcct := uuid.NewString()
	gainAcct := uuid.NewString()
	lossAcct := uuid.NewString()

	call := func(method, path string, body any, idem string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("X-Permissions", "gl.read,gl.create")
		req.Header.Set("Content-Type", "application/json")
		if idem != "" {
			req.Header.Set("Idempotency-Key", idem)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// 1. FX Revaluation:
	// - USD Bank: 1000 * (16000 - 15500) = +500,000.0000 gain
	// - EUR AP:   500 * (17000 - 17200) = -100,000.0000 loss
	// Net = +400,000.0000; balanced journal debit=600,000 credit=600,000
	fxBody := map[string]any{
		"journalNo":               "FX-2026-03",
		"financialOrganizationId": orgID,
		"ledgerId":                ledgerID,
		"fiscalPeriodId":          periodID,
		"postingDate":             "2026-03-31",
		"functionalCurrency":      "IDR",
		"fxGainAccountId":         gainAcct,
		"fxLossAccountId":         lossAcct,
		"positions": []map[string]any{
			{"accountId": usdBankAcct, "currencyCode": "USD", "foreignAmount": "1000.0000", "bookedRate": "15500.000000", "closingRate": "16000.000000"},
			{"accountId": eurApAcct, "currencyCode": "EUR", "foreignAmount": "500.0000", "bookedRate": "17200.000000", "closingRate": "17000.000000"},
		},
	}
	rec := call(http.MethodPost, "/journal-entries/fx-revaluation", fxBody, "fx-idem-1")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on fx-revaluation, got %d (%s)", rec.Code, rec.Body.String())
	}
	var fxResp fxRevaluationResp
	_ = json.Unmarshal(rec.Body.Bytes(), &fxResp)
	if fxResp.TotalGain != "500000.0000" || fxResp.TotalLoss != "100000.0000" || fxResp.NetUnrealizedGainLoss != "400000.0000" {
		t.Fatalf("unexpected FX revaluation totals: %+v", fxResp)
	}
	if fxResp.JournalEntry == nil || fxResp.JournalEntry.TotalDebit != "600000.0000" || fxResp.JournalEntry.TotalCredit != "600000.0000" {
		t.Fatalf("expected balanced 600000.0000 draft journal, got %+v", fxResp.JournalEntry)
	}

	// Idempotent replay returns identical 201 response without creating duplicate entry
	rec2 := call(http.MethodPost, "/journal-entries/fx-revaluation", fxBody, "fx-idem-1")
	if rec2.Code != http.StatusCreated || rec2.Body.String() != rec.Body.String() {
		t.Fatalf("expected idempotent replay on fx-revaluation, got %d", rec2.Code)
	}

	// 2. Recurring journal template + draft generation
	recBody := map[string]any{
		"code":                    "RENT-MTH",
		"name":                    "Monthly Office Rent Accrual",
		"cadence":                 "Monthly",
		"financialOrganizationId": orgID,
		"ledgerId":                ledgerID,
		"fiscalPeriodId":          periodID,
		"postingDate":             "2026-03-31",
		"currencyCode":            "IDR",
		"generateDraft":           true,
		"lines": []map[string]any{
			{"lineNo": 1, "accountId": lossAcct, "debitAmount": "25000000.0000", "creditAmount": "0.0000"},
			{"lineNo": 2, "accountId": gainAcct, "debitAmount": "0.0000", "creditAmount": "25000000.0000"},
		},
	}
	rec3 := call(http.MethodPost, "/journal-entries/recurring", recBody, "rec-idem-1")
	if rec3.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating recurring template, got %d (%s)", rec3.Code, rec3.Body.String())
	}
	var tpl recurringTemplateDTO
	_ = json.Unmarshal(rec3.Body.Bytes(), &tpl)
	if tpl.Code != "RENT-MTH" || tpl.GeneratedEntry == nil || tpl.GeneratedEntry.JournalNo != "REC-RENT-MTH-202603" {
		t.Fatalf("unexpected recurring template response: %+v", tpl)
	}

	// List recurring templates (verifies GET /journal-entries/recurring does not collide with /journal-entries/{id})
	recList := call(http.MethodGet, "/journal-entries/recurring", nil, "")
	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200 listing recurring templates, got %d (%s)", recList.Code, recList.Body.String())
	}
	var env httpx.ListEnvelope
	_ = json.Unmarshal(recList.Body.Bytes(), &env)
	if env.Page.TotalElements != 1 {
		t.Fatalf("expected 1 recurring template, got %+v", env.Page)
	}
}
