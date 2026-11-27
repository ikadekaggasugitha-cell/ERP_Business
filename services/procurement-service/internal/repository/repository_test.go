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

	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/procurementdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func TestPurchaseOrderRepository_EmptyScopeFailsClosed(t *testing.T) {
	repo := NewPurchaseOrderRepository(&platform.DB{})
	ctx := context.Background()
	id := uuid.New()

	cases := []struct {
		name string
		call func() error
	}{
		{"CreateWithLines", func() error { _, err := repo.CreateWithLines(ctx, nil, procurementdb.CreatePurchaseOrderParams{}, nil); return err }},
		{"GetByID", func() error { _, err := repo.GetByID(ctx, nil, id); return err }},
		{"ListWithCount", func() error { _, _, err := repo.ListWithCount(ctx, nil, procurementdb.ListPurchaseOrdersParams{}); return err }},
		{"UpdateDraftWithLines", func() error { _, _, err := repo.UpdateDraftWithLines(ctx, nil, procurementdb.UpdateDraftPurchaseOrderParams{}, nil); return err }},
		{"DeleteDraft", func() error { _, err := repo.DeleteDraft(ctx, nil, procurementdb.SoftDeleteDraftPurchaseOrderParams{}); return err }},
		{"ApproveAndEmit", func() error { _, _, err := repo.ApproveAndEmit(ctx, nil, procurementdb.ApprovePurchaseOrderParams{}, procurementdb.InsertOutboxParams{}); return err }},
		{"ReceiveAndEmit", func() error { _, _, _, err := repo.ReceiveAndEmit(ctx, nil, procurementdb.ReceivePurchaseOrderParams{}, procurementdb.CreateGoodsReceiptParams{}, nil); return err }},
		{"GetGoodsReceipt", func() error { _, err := repo.GetGoodsReceipt(ctx, nil, id); return err }},
		{"ListGoodsReceiptsWithCount", func() error { _, _, err := repo.ListGoodsReceiptsWithCount(ctx, nil, procurementdb.ListGoodsReceiptsParams{}); return err }},
		{"CreateRequisitionWithLines", func() error { _, err := repo.CreateRequisitionWithLines(ctx, nil, procurementdb.CreatePurchaseRequisitionParams{}, nil); return err }},
		{"GetRequisition", func() error { _, err := repo.GetRequisition(ctx, nil, id); return err }},
		{"ListRequisitionsWithCount", func() error { _, _, err := repo.ListRequisitionsWithCount(ctx, nil, procurementdb.ListPurchaseRequisitionsParams{}); return err }},
		{"TransitionRequisition", func() error { _, _, err := repo.TransitionRequisition(ctx, nil, procurementdb.TransitionPurchaseRequisitionParams{}); return err }},
		{"ConsumeAndCacheParty", func() error { _, err := repo.ConsumeAndCacheParty(ctx, nil, procurementdb.MarkEventProcessedParams{}, procurementdb.UpsertMasterdataCacheParams{}); return err }},
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

func TestProcurementNumericAndDetailContracts(t *testing.T) {
	qty, err := platform.ParseNumeric("12.5000")
	if err != nil {
		t.Fatalf("ParseNumeric qty: %v", err)
	}
	price, err := platform.ParseNumeric("350000.2500")
	if err != nil {
		t.Fatalf("ParseNumeric price: %v", err)
	}
	lineTotalRat := new(big.Rat).Mul(platform.NumericToRat(qty), platform.NumericToRat(price))
	lineTotalNum, err := platform.RatToNumeric(lineTotalRat, 4)
	if err != nil {
		t.Fatalf("RatToNumeric: %v", err)
	}
	// 12.5 * 350000.25 = 4375003.1250
	if got := platform.FormatNumeric(lineTotalNum, 4); got != "4375003.1250" {
		t.Fatalf("line total = %s, want 4375003.1250", got)
	}

	detail := PurchaseOrderDetail{
		Lines: []procurementdb.ListPurchaseOrderLinesRow{
			{LineNo: 1, OrderQty: qty, UnitPrice: price},
		},
	}
	if len(detail.Lines) != 1 || detail.Lines[0].LineNo != 1 {
		t.Fatalf("unexpected PurchaseOrderDetail: %+v", detail)
	}
}

func TestProcurementDBErrorMapping(t *testing.T) {
	var apiErr *httpx.APIError
	if !errors.As(httpx.MapDBError(&pgconn.PgError{Code: "23505"}, "duplicate poNo"), &apiErr) || apiErr.Status != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for 23505, got %v", apiErr)
	}
	if !errors.As(httpx.MapDBError(&pgconn.PgError{Code: "23514"}, "check constraint"), &apiErr) || apiErr.Status != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for 23514, got %v", apiErr)
	}
}
