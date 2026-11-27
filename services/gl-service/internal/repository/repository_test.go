package repository

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/gl-service/internal/gldb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func TestFunctionalDebitCredit(t *testing.T) {
	mustNum := func(s string) pgtype.Numeric {
		t.Helper()
		n, err := platform.ParseNumeric(s)
		if err != nil {
			t.Fatalf("ParseNumeric(%q): %v", s, err)
		}
		return n
	}

	t.Run("uses explicit functional amount on debit line", func(t *testing.T) {
		line := gldb.ListJournalLinesRow{
			DebitAmount:      mustNum("100.0000"),
			CreditAmount:     mustNum("0"),
			FunctionalAmount: mustNum("1625000.0000"),
		}
		dr, cr := functionalDebitCredit(line, mustNum("16250.00000000"))
		if got := platform.FormatNumeric(dr, 4); got != "1625000.0000" {
			t.Fatalf("functional debit = %s, want 1625000.0000", got)
		}
		if got := platform.FormatNumeric(cr, 4); got != "0.0000" {
			t.Fatalf("functional credit = %s, want 0.0000", got)
		}
	})

	t.Run("derives functional credit from foreign credit * exchange rate", func(t *testing.T) {
		line := gldb.ListJournalLinesRow{
			DebitAmount:  mustNum("0"),
			CreditAmount: mustNum("250.5000"),
		}
		dr, cr := functionalDebitCredit(line, mustNum("16250.25000000"))
		if got := platform.FormatNumeric(dr, 4); got != "0.0000" {
			t.Fatalf("functional debit = %s, want 0.0000", got)
		}
		// 250.5 * 16250.25 = 4070687.6250
		if got := platform.FormatNumeric(cr, 4); got != "4070687.6250" {
			t.Fatalf("functional credit = %s, want 4070687.6250", got)
		}
	})

	t.Run("defaults invalid exchange rate to 1:1 without float drift", func(t *testing.T) {
		line := gldb.ListJournalLinesRow{
			DebitAmount:  mustNum("999999999999.1234"),
			CreditAmount: mustNum("0"),
		}
		dr, cr := functionalDebitCredit(line, pgtype.Numeric{})
		if got := platform.FormatNumeric(dr, 4); got != "999999999999.1234" {
			t.Fatalf("functional debit = %s, want 999999999999.1234", got)
		}
		if got := platform.FormatNumeric(cr, 4); got != "0.0000" {
			t.Fatalf("functional credit = %s, want 0.0000", got)
		}
	})

	t.Run("zero line returns zero debit and credit", func(t *testing.T) {
		dr, cr := functionalDebitCredit(gldb.ListJournalLinesRow{}, mustNum("1"))
		if got := platform.FormatNumeric(dr, 4); got != "0.0000" {
			t.Fatalf("dr = %s, want 0.0000", got)
		}
		if got := platform.FormatNumeric(cr, 4); got != "0.0000" {
			t.Fatalf("cr = %s, want 0.0000", got)
		}
	})
}

func TestWellKnownAccountCodesContract(t *testing.T) {
	if len(wellKnownAccountCodes) < 12 {
		t.Fatalf("expected at least 12 well-known control accounts, got %d", len(wellKnownAccountCodes))
	}
	for idStr, code := range wellKnownAccountCodes {
		if _, err := uuid.Parse(idStr); err != nil {
			t.Errorf("invalid account UUID %q: %v", idStr, err)
		}
		if len(code) != 4 {
			t.Errorf("account %s has unexpected code %q", idStr, code)
		}
	}
}

func TestCheckFiscalPeriodOpen_NilPeriodSkipsQuery(t *testing.T) {
	if err := checkFiscalPeriodOpen(context.Background(), nil, uuid.New(), uuid.Nil); err != nil {
		t.Fatalf("expected nil error for uuid.Nil period, got %v", err)
	}
}

func TestJournalRepository_EmptyScopeFailsClosed(t *testing.T) {
	repo := NewJournalRepository(&platform.DB{})
	ctx := context.Background()
	id := uuid.New()

	cases := []struct {
		name string
		call func() error
	}{
		{"CreateEntryWithLines", func() error { _, err := repo.CreateEntryWithLines(ctx, nil, gldb.CreateJournalEntryParams{}, nil); return err }},
		{"GetByID", func() error { _, _, err := repo.GetByID(ctx, nil, id); return err }},
		{"ListWithCount", func() error { _, _, err := repo.ListWithCount(ctx, nil, gldb.ListJournalEntriesParams{}); return err }},
		{"UpdateDraftEntry", func() error { _, err := repo.UpdateDraftEntry(ctx, nil, gldb.UpdateJournalEntryDraftParams{}, nil); return err }},
		{"SoftDeleteDraft", func() error { _, err := repo.SoftDeleteDraft(ctx, nil, gldb.SoftDeleteJournalEntryParams{}); return err }},
		{"Totals", func() error { _, err := repo.Totals(ctx, nil, gldb.JournalEntryTotalsParams{}); return err }},
		{"ConsumePeriodStatus", func() error { _, err := repo.ConsumePeriodStatus(ctx, nil, gldb.MarkEventProcessedParams{}, gldb.UpsertFiscalPeriodStatusParams{}); return err }},
		{"PostAndEmit", func() error { _, err := repo.PostAndEmit(ctx, nil, gldb.GlJournalEntry{}, 1, id); return err }},
		{"ReverseEntryAndEmit", func() error { _, _, err := repo.ReverseEntryAndEmit(ctx, nil, gldb.GlJournalEntry{}, 1, id, id); return err }},
		{"ListAccountBalances", func() error { _, _, err := repo.ListAccountBalances(ctx, nil, id, id, 10, 0); return err }},
		{"ConsumeAndPost", func() error { _, err := repo.ConsumeAndPost(ctx, nil, gldb.MarkEventProcessedParams{}, gldb.CreateJournalEntryParams{}, nil, "ev", "ref"); return err }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil || !strings.Contains(err.Error(), "org scope must not be empty") {
				t.Fatalf("%s: expected fail-closed empty scope error, got %v", tc.name, err)
			}
		})
	}
}

func TestGLDBErrorMappingContract(t *testing.T) {
	dup := httpx.MapDBError(&pgconn.PgError{Code: "23505"}, "duplicate journalNo")
	var apiErr *httpx.APIError
	if dup == nil {
		t.Fatal("expected mapped conflict error")
	}
	if ok := errorAsAPI(dup, &apiErr); !ok || apiErr.Status != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for SQLSTATE 23505, got %v", dup)
	}
}

func errorAsAPI(err error, target **httpx.APIError) bool {
	if ae, ok := err.(*httpx.APIError); ok {
		*target = ae
		return true
	}
	return false
}
