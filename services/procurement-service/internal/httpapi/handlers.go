// Package httpapi is procurement-service's REST layer (SDD BAB 4): Purchase Requisitions,
// Purchase Orders, Goods Receipts, Three-Way Match, and Return-to-Vendor.
package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/procurementdb"
	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

// Store abstracts procurement repository operations for handlers and unit tests.
type Store interface {
	CreateWithLines(ctx context.Context, scope []uuid.UUID, po procurementdb.CreatePurchaseOrderParams, lines []procurementdb.CreatePurchaseOrderLineParams) (procurementdb.CreatePurchaseOrderRow, error)
	GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (repository.PurchaseOrderDetail, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, arg procurementdb.ListPurchaseOrdersParams) ([]procurementdb.ListPurchaseOrdersRow, int64, error)
	UpdateDraftWithLines(ctx context.Context, scope []uuid.UUID, arg procurementdb.UpdateDraftPurchaseOrderParams, lines []procurementdb.CreatePurchaseOrderLineParams) (int32, int64, error)
	DeleteDraft(ctx context.Context, scope []uuid.UUID, arg procurementdb.SoftDeleteDraftPurchaseOrderParams) (int64, error)
	ApproveAndEmit(ctx context.Context, scope []uuid.UUID, arg procurementdb.ApprovePurchaseOrderParams, ev procurementdb.InsertOutboxParams) (int32, int64, error)
	ReceiveAndEmit(ctx context.Context, scope []uuid.UUID, arg procurementdb.ReceivePurchaseOrderParams, grArg procurementdb.CreateGoodsReceiptParams, buildOutbox repository.OutboxPayloadFunc) (int32, procurementdb.CreateGoodsReceiptRow, int64, error)
	GetGoodsReceipt(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (repository.GoodsReceiptDetail, error)
	ListGoodsReceiptsWithCount(ctx context.Context, scope []uuid.UUID, arg procurementdb.ListGoodsReceiptsParams) ([]procurementdb.ListGoodsReceiptsRow, int64, error)
	CreateRequisitionWithLines(ctx context.Context, scope []uuid.UUID, pr procurementdb.CreatePurchaseRequisitionParams, lines []procurementdb.CreatePurchaseRequisitionLineParams) (procurementdb.CreatePurchaseRequisitionRow, error)
	GetRequisition(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (repository.PurchaseRequisitionDetail, error)
	ListRequisitionsWithCount(ctx context.Context, scope []uuid.UUID, arg procurementdb.ListPurchaseRequisitionsParams) ([]procurementdb.ListPurchaseRequisitionsRow, int64, error)
	TransitionRequisition(ctx context.Context, scope []uuid.UUID, arg procurementdb.TransitionPurchaseRequisitionParams) (procurementdb.TransitionPurchaseRequisitionRow, int64, error)
}

// Server serves HTTP endpoints for procurement-service.
type Server struct {
	repo Store
	idem *httpx.IdemStore
}

// NewServer constructs an HTTP Server backed by repo.
func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

// Register attaches all procurement routes onto mux.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /purchase-orders", s.createPO)
	mux.HandleFunc("GET /purchase-orders", s.listPOs)
	mux.HandleFunc("GET /purchase-orders/{id}", s.getPO)
	mux.HandleFunc("PUT /purchase-orders/{id}", s.updatePO)
	mux.HandleFunc("DELETE /purchase-orders/{id}", s.deletePO)
	mux.HandleFunc("POST /purchase-orders/{id}/approve", s.approvePO)
	mux.HandleFunc("POST /purchase-orders/{id}/receive", s.receivePO)
	mux.HandleFunc("POST /purchase-orders/{id}/three-way-match", s.threeWayMatchPO)

	mux.HandleFunc("GET /goods-receipts", s.listGRs)
	mux.HandleFunc("GET /goods-receipts/{id}", s.getGR)
	mux.HandleFunc("POST /goods-receipts/{id}/return-to-vendor", s.returnToVendor)

	mux.HandleFunc("POST /purchase-requisitions", s.createPR)
	mux.HandleFunc("GET /purchase-requisitions", s.listPRs)
	mux.HandleFunc("GET /purchase-requisitions/{id}", s.getPR)
	mux.HandleFunc("POST /purchase-requisitions/{id}/submit", s.submitPR)
	mux.HandleFunc("POST /purchase-requisitions/{id}/approve", s.approvePR)
}

