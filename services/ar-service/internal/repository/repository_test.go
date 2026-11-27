package repository

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/sinergiinvestasiproperti/erp/services/ar-service/internal/ardb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func TestARInvoiceRepository_EmptyScopeFailsClosed(t *testing.T) {
	repo := NewInvoiceRepository(&platform.DB{})
	ctx := context.Background()
	id := uuid.New()

	cases := []struct {
		name string
		call func() error
	}{
		{"ConsumeAndCacheParty", func() error { _, err := repo.ConsumeAndCacheParty(ctx, nil, ardb.MarkEventProcessedParams{}, ardb.UpsertMasterdataCacheParams{}); return err }},
		{"CreateInvoiceWithLinesAndEmit", func() error { _, err := repo.CreateInvoiceWithLinesAndEmit(ctx, nil, ardb.CreateInvoiceParams{}, nil, ardb.InsertOutboxParams{}); return err }},
		{"UpdateDraftWithLines", func() error { _, err := repo.UpdateDraftWithLines(ctx, nil, ardb.UpdateDraftInvoiceParams{}, nil); return err }},
		{"ApplyCreditNoteAndEmit", func() error { _, err := repo.ApplyCreditNoteAndEmit(ctx, nil, ardb.ApplyReceiptToInvoiceParams{}, ardb.InsertOutboxParams{}); return err }},
		{"WriteOffAndEmit", func() error { _, err := repo.WriteOffAndEmit(ctx, nil, ardb.WriteOffInvoiceParams{}, ardb.InsertOutboxParams{}); return err }},
		{"ConsumePaymentSettled", func() error { _, err := repo.ConsumePaymentSettled(ctx, nil, ardb.MarkEventProcessedParams{}, id, id, id, id, nil); return err }},
		{"ConsumeAndCreateInvoice", func() error { _, err := repo.ConsumeAndCreateInvoice(ctx, nil, ardb.MarkEventProcessedParams{}, ardb.CreateInvoiceParams{}, nil, ardb.InsertOutboxParams{}); return err }},
		{"GetWithLines", func() error { _, _, err := repo.GetWithLines(ctx, nil, id); return err }},
		{"GetByID", func() error { _, err := repo.GetByID(ctx, nil, id); return err }},
		{"ListWithCount", func() error { _, _, err := repo.ListWithCount(ctx, nil, "Issued", 20, 0); return err }},
		{"MarkPaid", func() error { _, err := repo.MarkPaid(ctx, nil, ardb.MarkInvoicePaidParams{}, ardb.InsertOutboxParams{}); return err }},
		{"DeleteDraft", func() error { _, err := repo.DeleteDraft(ctx, nil, ardb.DeleteDraftInvoiceParams{}); return err }},
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

func TestARSettlementAllocation_ExactFormatting(t *testing.T) {
	num, err := platform.ParseNumeric("165000000.50")
	if err != nil {
		t.Fatalf("ParseNumeric: %v", err)
	}
	alloc := SettlementAllocation{
		InvoiceRef:      "INV-2026-0001",
		AllocatedAmount: num,
	}
	if got := platform.FormatNumeric(alloc.AllocatedAmount, 4); got != "165000000.5000" {
		t.Fatalf("FormatNumeric = %s, want 165000000.5000", got)
	}
	outstanding := new(big.Rat).Sub(platform.NumericToRat(num), platform.NumericToRat(num))
	if outstanding.Sign() != 0 {
		t.Fatalf("expected zero outstanding after full allocation, got %s", outstanding.FloatString(4))
	}
}

func TestARScopeAndDBErrorContracts(t *testing.T) {
	if got := platform.ScopeWithSystemOrg(nil); got != nil {
		t.Fatalf("ScopeWithSystemOrg(nil) must fail closed, got %v", got)
	}
	org := uuid.New()
	withSys := platform.ScopeWithSystemOrg([]uuid.UUID{org})
	if len(withSys) != 2 || withSys[0] != org || withSys[1] != platform.SystemOrgID {
		t.Fatalf("unexpected ScopeWithSystemOrg result: %v", withSys)
	}

	var apiErr *httpx.APIError
	if !errors.As(httpx.MapDBError(&pgconn.PgError{Code: "22P02"}, "bad uuid"), &apiErr) || apiErr.Status != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for 22P02, got %v", apiErr)
	}
}
