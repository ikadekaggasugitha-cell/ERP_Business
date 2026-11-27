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
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/inventorydb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func TestStockRepository_EmptyScopeFailsClosed(t *testing.T) {
	repo := NewStockRepository(&platform.DB{})
	ctx := context.Background()

	cases := []struct {
		name string
		call func() error
	}{
		{"ConsumeAndCacheParty", func() error { _, err := repo.ConsumeAndCacheParty(ctx, nil, inventorydb.MarkEventProcessedParams{}, inventorydb.UpsertMasterdataCacheParams{}); return err }},
		{"CreateWarehouse", func() error { _, err := repo.CreateWarehouse(ctx, nil, inventorydb.CreateWarehouseParams{}); return err }},
		{"ListWarehousesWithCount", func() error { _, _, err := repo.ListWarehousesWithCount(ctx, nil, inventorydb.ListWarehousesParams{}); return err }},
		{"ListStockBalancesWithCount", func() error { _, _, err := repo.ListStockBalancesWithCount(ctx, nil, inventorydb.ListStockBalancesParams{}); return err }},
		{"ListStockMovementsWithCount", func() error { _, _, err := repo.ListStockMovementsWithCount(ctx, nil, inventorydb.ListStockMovementsParams{}); return err }},
		{"RecordStockMovement", func() error { _, err := repo.RecordStockMovement(ctx, nil, inventorydb.CreateStockMovementParams{}, pgtype.Numeric{}); return err }},
		{"CreateReservationAndEmit", func() error { _, err := repo.CreateReservationAndEmit(ctx, nil, inventorydb.CreateReservationParams{}, inventorydb.InsertOutboxParams{}); return err }},
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

func TestInventorySentinelErrorsAndStockLineNumeric(t *testing.T) {
	if ErrInsufficientStock == nil || !strings.Contains(ErrInsufficientStock.Error(), "insufficient") {
		t.Fatalf("unexpected ErrInsufficientStock: %v", ErrInsufficientStock)
	}
	if ErrNoWarehouse == nil || !strings.Contains(ErrNoWarehouse.Error(), "warehouse") {
		t.Fatalf("unexpected ErrNoWarehouse: %v", ErrNoWarehouse)
	}

	qty, err := platform.ParseNumeric("45.2500")
	if err != nil {
		t.Fatalf("ParseNumeric qty: %v", err)
	}
	price, err := platform.ParseNumeric("120000.0000")
	if err != nil {
		t.Fatalf("ParseNumeric price: %v", err)
	}
	req := StockLineRequest{ItemID: uuid.New(), Qty: qty, UnitPrice: price}

	// Issue movements negate positive quantities via *big.Rat without float64.
	negNum, err := platform.RatToNumeric(new(big.Rat).Neg(platform.NumericToRat(req.Qty)), 4)
	if err != nil {
		t.Fatalf("RatToNumeric neg: %v", err)
	}
	if got := platform.FormatNumeric(negNum, 4); got != "-45.2500" {
		t.Fatalf("negated qty = %s, want -45.2500", got)
	}
	absNum, err := platform.RatToNumeric(new(big.Rat).Abs(platform.NumericToRat(negNum)), 4)
	if err != nil {
		t.Fatalf("RatToNumeric abs: %v", err)
	}
	if got := platform.FormatNumeric(absNum, 4); got != "45.2500" {
		t.Fatalf("abs qty = %s, want 45.2500", got)
	}
}

func TestInventoryDBErrorMapping(t *testing.T) {
	var apiErr *httpx.APIError
	if !errors.As(httpx.MapDBError(&pgconn.PgError{Code: "23505"}, "duplicate warehouse code"), &apiErr) || apiErr.Status != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for 23505, got %v", apiErr)
	}
}
