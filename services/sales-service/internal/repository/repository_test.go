package repository

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func TestNewSalesOrderRepository_NilSafe(t *testing.T) {
	rNil := NewSalesOrderRepository(nil)
	if rNil == nil || rNil.db != nil || rNil.q != nil {
		t.Fatalf("expected nil-safe repository for nil db, got %+v", rNil)
	}
	rEmpty := NewSalesOrderRepository(&platform.DB{})
	if rEmpty == nil || rEmpty.q != nil {
		t.Fatalf("expected nil queries when Pool is nil, got %+v", rEmpty)
	}
}

func TestSalesOrderRepository_EmptyScopeFailsClosed(t *testing.T) {
	repo := NewSalesOrderRepository(&platform.DB{})
	ctx := context.Background()
	id := uuid.New()

	cases := []struct {
		name string
		call func() error
	}{
		{"CreateWithLines", func() error { _, err := repo.CreateWithLines(ctx, nil, CreateSOInput{}); return err }},
		{"GetByID", func() error { _, err := repo.GetByID(ctx, nil, id); return err }},
		{"ListWithCount", func() error { _, _, err := repo.ListWithCount(ctx, nil, nil, 20, 0); return err }},
		{"UpdateDraftWithLines", func() error { _, err := repo.UpdateDraftWithLines(ctx, nil, UpdateDraftSOInput{}); return err }},
		{"DeleteDraft", func() error { return repo.DeleteDraft(ctx, nil, id, 1, id) }},
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

func TestSalesOrderInput_FixedPointLineSum(t *testing.T) {
	q1, _ := platform.ParseNumeric("3.0000")
	p1, _ := platform.ParseNumeric("1250000.5000")
	q2, _ := platform.ParseNumeric("2.5000")
	p2, _ := platform.ParseNumeric("400000.0000")

	in := CreateSOInput{
		OrganizationID: uuid.New(),
		SoNo:           "SO-2026-0001",
		CustomerID:     uuid.New(),
		OrderDate:      pgtype.Date{Time: time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC), Valid: true},
		CurrencyCode:   "IDR",
		Lines: []CreateSOLineInput{
			{LineNo: 1, ItemID: uuid.New(), OrderQty: q1, UnitPrice: p1},
			{LineNo: 2, ItemID: uuid.New(), OrderQty: q2, UnitPrice: p2},
		},
	}

	total := new(big.Rat)
	for _, l := range in.Lines {
		total.Add(total, new(big.Rat).Mul(platform.NumericToRat(l.OrderQty), platform.NumericToRat(l.UnitPrice)))
	}
	num, err := platform.RatToNumeric(total, 4)
	if err != nil {
		t.Fatalf("RatToNumeric: %v", err)
	}
	// 3 * 1250000.5 + 2.5 * 400000 = 3750001.5 + 1000000 = 4750001.5000
	if got := platform.FormatNumeric(num, 4); got != "4750001.5000" {
		t.Fatalf("total = %s, want 4750001.5000", got)
	}
}

func TestSalesOrderDBErrorMapping(t *testing.T) {
	var apiErr *httpx.APIError
	if !errors.As(httpx.MapDBError(&pgconn.PgError{Code: "23505"}, "soNo sudah digunakan"), &apiErr) || apiErr.Status != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for 23505, got %v", apiErr)
	}
	if !errors.As(httpx.Conflict("VERSION_CONFLICT", "stale"), &apiErr) || apiErr.Status != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for VERSION_CONFLICT, got %v", apiErr)
	}
}
