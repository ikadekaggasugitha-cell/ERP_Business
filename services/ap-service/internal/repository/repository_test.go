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

	"github.com/sinergiinvestasiproperti/erp/services/ap-service/internal/apdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func TestAPInvoiceRepository_EmptyScopeFailsClosed(t *testing.T) {
	repo := NewInvoiceRepository(&platform.DB{})
	ctx := context.Background()
	id := uuid.New()

	cases := []struct {
		name string
		call func() error
	}{
		{"ConsumeAndCacheParty", func() error { _, err := repo.ConsumeAndCacheParty(ctx, nil, apdb.MarkEventProcessedParams{}, apdb.UpsertMasterdataCacheParams{}); return err }},
		{"CreateInvoiceWithLines", func() error { _, err := repo.CreateInvoiceWithLines(ctx, nil, apdb.CreateInvoiceParams{}, nil); return err }},
		{"UpdateDraftWithLines", func() error { _, err := repo.UpdateDraftWithLines(ctx, nil, apdb.UpdateDraftInvoiceParams{}, nil); return err }},
		{"DeleteDraft", func() error { _, err := repo.DeleteDraft(ctx, nil, apdb.DeleteDraftInvoiceParams{}); return err }},
		{"GetByID", func() error { _, err := repo.GetByID(ctx, nil, id); return err }},
		{"GetWithLines", func() error { _, _, err := repo.GetWithLines(ctx, nil, id); return err }},
		{"ListWithCount", func() error { _, _, err := repo.ListWithCount(ctx, nil, "Draft", 20, 0); return err }},
		{"ListByStatus", func() error { _, err := repo.ListByStatus(ctx, nil, apdb.ListInvoicesByStatusParams{}); return err }},
		{"LineTotal", func() error { _, err := repo.LineTotal(ctx, nil, apdb.InvoiceLineTotalParams{}); return err }},
		{"ApplyDebitNoteAndEmit", func() error { _, err := repo.ApplyDebitNoteAndEmit(ctx, nil, apdb.ApplyPaymentToInvoiceParams{}, apdb.InsertOutboxParams{}); return err }},
		{"ConsumePaymentSettled", func() error { _, err := repo.ConsumePaymentSettled(ctx, nil, apdb.MarkEventProcessedParams{}, id, id, id, nil); return err }},
		{"ConsumeAndRecordInvoice", func() error { _, err := repo.ConsumeAndRecordInvoice(ctx, nil, apdb.MarkEventProcessedParams{}, apdb.CreateInvoiceParams{}, apdb.InsertOutboxParams{}); return err }},
		{"RecordAndEmit", func() error { _, err := repo.RecordAndEmit(ctx, nil, apdb.RecordInvoiceParams{}, apdb.InsertOutboxParams{}); return err }},
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

func TestAPSettlementAllocation_FixedPointPrecision(t *testing.T) {
	a1, err := platform.ParseNumeric("12500000.3333")
	if err != nil {
		t.Fatalf("ParseNumeric a1: %v", err)
	}
	a2, err := platform.ParseNumeric("2499999.6667")
	if err != nil {
		t.Fatalf("ParseNumeric a2: %v", err)
	}
	allocs := []SettlementAllocation{
		{InvoiceRef: "AP-2026-001", AllocatedAmount: a1},
		{InvoiceRef: "AP-2026-002", AllocatedAmount: a2},
	}

	sum := new(big.Rat)
	for _, a := range allocs {
		sum.Add(sum, platform.NumericToRat(a.AllocatedAmount))
	}
	expected, _ := platform.ParseNumeric("15000000.0000")
	if sum.Cmp(platform.NumericToRat(expected)) != 0 {
		t.Fatalf("sum = %s, want 15000000.0000", sum.FloatString(4))
	}
}

func TestAPScopeAndErrorMappingContracts(t *testing.T) {
	if got := platform.TenantOnlyScope(nil); got != nil {
		t.Fatalf("TenantOnlyScope(nil) = %v, want nil", got)
	}
	tenant := uuid.New()
	if got := platform.TenantOnlyScope([]uuid.UUID{platform.SystemOrgID, tenant}); len(got) != 1 || got[0] != tenant {
		t.Fatalf("TenantOnlyScope did not strip SystemOrgID: %v", got)
	}

	var apiErr *httpx.APIError
	if !errors.As(httpx.MapDBError(&pgconn.PgError{Code: "23505"}, "duplicate invoiceNo"), &apiErr) || apiErr.Status != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for 23505, got %v", apiErr)
	}
	if !errors.As(httpx.MapDBError(&pgconn.PgError{Code: "23503"}, "fk violation"), &apiErr) || apiErr.Status != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for 23503, got %v", apiErr)
	}
}
