// Package httpapi is manufacturing-service's REST layer (SDD BAB 4): Bills of Material,
// Work Orders, MRP Planning, and Scrap/Yield Variance.
package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/manufacturing-service/internal/manufacturingdb"
	"github.com/sinergiinvestasiproperti/erp/services/manufacturing-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

// Store abstracts manufacturing repository operations for handlers and tests.
type Store interface {
	CreateBOMWithLines(ctx context.Context, scope []uuid.UUID, bom manufacturingdb.CreateBillOfMaterialParams, lines []manufacturingdb.CreateBomLineParams) (manufacturingdb.CreateBillOfMaterialRow, error)
	GetBOM(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (repository.BOMDetail, error)
	ListBOMsWithCount(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.ListBillsOfMaterialParams) ([]manufacturingdb.ListBillsOfMaterialRow, int64, error)
	CreateWorkOrder(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.CreateWorkOrderParams) (manufacturingdb.CreateWorkOrderRow, error)
	GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (manufacturingdb.GetWorkOrderRow, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.ListWorkOrdersParams) ([]manufacturingdb.ListWorkOrdersRow, int64, error)
	UpdatePlanned(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.UpdatePlannedWorkOrderParams) (int32, int64, error)
	DeletePlanned(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.SoftDeletePlannedWorkOrderParams) (int64, error)
	ReleaseAndEmit(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.ReleaseWorkOrderParams, ev manufacturingdb.InsertOutboxParams) (int32, int64, error)
	CompleteAndEmit(ctx context.Context, scope []uuid.UUID, arg manufacturingdb.CompleteWorkOrderParams, ev manufacturingdb.InsertOutboxParams) (int32, int64, error)
}

// Server serves HTTP endpoints for manufacturing-service.
type Server struct {
	repo Store
	idem *httpx.IdemStore
}

// NewServer constructs an HTTP Server backed by repo.
func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

// Register attaches all manufacturing routes onto mux.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /bill-of-materials", s.createBOM)
	mux.HandleFunc("GET /bill-of-materials", s.listBOMs)
	mux.HandleFunc("GET /bill-of-materials/{id}", s.getBOM)
	mux.HandleFunc("POST /work-orders", s.createWO)
	mux.HandleFunc("GET /work-orders", s.listWOs)
	mux.HandleFunc("POST /work-orders/mrp-plan", s.mrpPlan)
	mux.HandleFunc("GET /work-orders/{id}", s.getWO)
	mux.HandleFunc("GET /work-orders/{id}/variance", s.workOrderVariance)
	mux.HandleFunc("PUT /work-orders/{id}", s.updateWO)
	mux.HandleFunc("DELETE /work-orders/{id}", s.deleteWO)
	mux.HandleFunc("POST /work-orders/{id}/release", s.releaseWO)
	mux.HandleFunc("POST /work-orders/{id}/complete", s.completeWO)
}

