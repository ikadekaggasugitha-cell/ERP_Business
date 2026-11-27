// Package httpapi is procurement-service's REST layer (SDD BAB 4): Purchase Requisitions,
// Purchase Orders (Draft CRUD, Approve -> procurement.po.approved, Receive -> Goods Receipt + procurement.goods.received),
// and Goods Receipts read APIs.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/procurementdb"
	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
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

	mux.HandleFunc("POST /purchase-requisitions", s.createPR)
	mux.HandleFunc("GET /purchase-requisitions", s.listPRs)
	mux.HandleFunc("GET /purchase-requisitions/{id}", s.getPR)
	mux.HandleFunc("POST /purchase-requisitions/{id}/submit", s.submitPR)
	mux.HandleFunc("POST /purchase-requisitions/{id}/approve", s.approvePR)
}

type poLineReq struct {
	LineNo    int32  `json:"lineNo"`
	ItemID    string `json:"itemId"`
	OrderQty  string `json:"orderQty"`
	UnitPrice string `json:"unitPrice"`
}

type poUpsertReq struct {
	PoNo                  string      `json:"poNo"`
	VendorID              string      `json:"vendorId"`
	PurchaseRequisitionID string      `json:"purchaseRequisitionId,omitempty"`
	OrderDate             string      `json:"orderDate"`
	CurrencyCode          string      `json:"currencyCode"`
	TotalAmount           string      `json:"totalAmount"`
	Lines                 []poLineReq `json:"lines"`
}

type receiveReq struct {
	GrNo        string `json:"grNo,omitempty"`
	WarehouseID string `json:"warehouseId,omitempty"`
	ReceiptDate string `json:"receiptDate,omitempty"`
}

type poLineDTO struct {
	ID          string `json:"id,omitempty"`
	LineNo      int32  `json:"lineNo"`
	ItemID      string `json:"itemId"`
	OrderQty    string `json:"orderQty"`
	ReceivedQty string `json:"receivedQty"`
	UnitPrice   string `json:"unitPrice"`
}

type poDTO struct {
	ID                    string      `json:"id"`
	PoNo                  string      `json:"poNo,omitempty"`
	VendorID              string      `json:"vendorId,omitempty"`
	PurchaseRequisitionID string      `json:"purchaseRequisitionId,omitempty"`
	OrderDate             string      `json:"orderDate,omitempty"`
	CurrencyCode          string      `json:"currencyCode,omitempty"`
	TotalAmount           string      `json:"totalAmount,omitempty"`
	Status                string      `json:"status"`
	RowVersion            int32       `json:"rowVersion"`
	GrID                  string      `json:"grId,omitempty"`
	GrNo                  string      `json:"grNo,omitempty"`
	Lines                 []poLineDTO `json:"lines,omitempty"`
}

type grLineDTO struct {
	ID          string `json:"id"`
	PoLineID    string `json:"poLineId"`
	LineNo      int32  `json:"lineNo"`
	ReceivedQty string `json:"receivedQty"`
}

type grDTO struct {
	ID              string      `json:"id"`
	GrNo            string      `json:"grNo"`
	PurchaseOrderID string      `json:"purchaseOrderId"`
	WarehouseID     string      `json:"warehouseId"`
	ReceiptDate     string      `json:"receiptDate"`
	Status          string      `json:"status"`
	RowVersion      int32       `json:"rowVersion"`
	Lines           []grLineDTO `json:"lines,omitempty"`
}

type prLineReq struct {
	LineNo     int32  `json:"lineNo"`
	ItemID     string `json:"itemId"`
	RequestQty string `json:"requestQty"`
	NeededDate string `json:"neededDate,omitempty"`
}

type prCreateReq struct {
	PrNo        string      `json:"prNo"`
	RequesterID string      `json:"requesterId,omitempty"`
	WorkflowRef string      `json:"workflowRef,omitempty"`
	NeededDate  string      `json:"neededDate,omitempty"`
	Lines       []prLineReq `json:"lines"`
}

type prLineDTO struct {
	ID         string `json:"id"`
	LineNo     int32  `json:"lineNo"`
	ItemID     string `json:"itemId"`
	RequestQty string `json:"requestQty"`
	NeededDate string `json:"neededDate,omitempty"`
}

type prDTO struct {
	ID          string      `json:"id"`
	PrNo        string      `json:"prNo,omitempty"`
	RequesterID string      `json:"requesterId,omitempty"`
	WorkflowRef string      `json:"workflowRef,omitempty"`
	NeededDate  string      `json:"neededDate,omitempty"`
	Status      string      `json:"status"`
	RowVersion  int32       `json:"rowVersion"`
	Lines       []prLineDTO `json:"lines,omitempty"`
}

func (s *Server) createPO(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req poUpsertReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if strings.TrimSpace(req.PoNo) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("poNo wajib diisi"))
		return
	}
	vendor, prID, orderDate, currency, totalNum, lines, err := validatePOInput(scope[0], user, req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	po := procurementdb.CreatePurchaseOrderParams{
		OrganizationID:        scope[0],
		PoNo:                  strings.TrimSpace(req.PoNo),
		VendorID:              vendor,
		PurchaseRequisitionID: prID,
		OrderDate:             orderDate,
		CurrencyCode:          currency,
		TotalAmount:           totalNum,
		CreatedBy:             user,
	}
	created, err := s.repo.CreateWithLines(r.Context(), scope, po, lines)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "nomor PO sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, poDTO{
		ID:           created.ID.String(),
		PoNo:         po.PoNo,
		VendorID:     vendor.String(),
		OrderDate:    platform.FormatDate(orderDate),
		CurrencyCode: currency,
		TotalAmount:  platform.FormatNumeric(totalNum, 4),
		Status:       created.Status,
		RowVersion:   created.RowVersion,
	})
}

func (s *Server) listPOs(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var statusFilter *string
	if st := strings.TrimSpace(r.URL.Query().Get("status")); st != "" {
		statusFilter = &st
	}
	rows, total, err := s.repo.ListWithCount(r.Context(), scope, procurementdb.ListPurchaseOrdersParams{
		Limit:  limit,
		Offset: offset,
		Status: statusFilter,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]poDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, poDTO{
			ID:           row.ID.String(),
			PoNo:         row.PoNo,
			VendorID:     row.VendorID.String(),
			OrderDate:    platform.FormatDate(row.OrderDate),
			CurrencyCode: row.CurrencyCode,
			TotalAmount:  platform.FormatNumeric(row.TotalAmount, 4),
			Status:       row.Status,
			RowVersion:   row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getPO(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	detail, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("purchase order tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	lines := make([]poLineDTO, 0, len(detail.Lines))
	for _, l := range detail.Lines {
		lines = append(lines, poLineDTO{
			ID:          l.ID.String(),
			LineNo:      l.LineNo,
			ItemID:      l.ItemID.String(),
			OrderQty:    platform.FormatNumeric(l.OrderQty, 4),
			ReceivedQty: platform.FormatNumeric(l.ReceivedQty, 4),
			UnitPrice:   platform.FormatNumeric(l.UnitPrice, 4),
		})
	}
	dto := poDTO{
		ID:           detail.Header.ID.String(),
		PoNo:         detail.Header.PoNo,
		VendorID:     detail.Header.VendorID.String(),
		OrderDate:    platform.FormatDate(detail.Header.OrderDate),
		CurrencyCode: detail.Header.CurrencyCode,
		TotalAmount:  platform.FormatNumeric(detail.Header.TotalAmount, 4),
		Status:       detail.Header.Status,
		RowVersion:   detail.Header.RowVersion,
		Lines:        lines,
	}
	if detail.Header.PurchaseRequisitionID != nil {
		dto.PurchaseRequisitionID = detail.Header.PurchaseRequisitionID.String()
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.Header.RowVersion, dto)
}

func (s *Server) updatePO(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req poUpsertReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	vendor, prID, orderDate, currency, totalNum, lines, err := validatePOInput(scope[0], user, req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	newVer, n, err := s.repo.UpdateDraftWithLines(r.Context(), scope, procurementdb.UpdateDraftPurchaseOrderParams{
		ID:                    id,
		RowVersion:            rowVer,
		VendorID:              vendor,
		PurchaseRequisitionID: prID,
		OrderDate:             orderDate,
		CurrencyCode:          currency,
		TotalAmount:           totalNum,
		UpdatedBy:             &user,
	}, lines)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Draft"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, poDTO{
		ID:           id.String(),
		VendorID:     vendor.String(),
		OrderDate:    platform.FormatDate(orderDate),
		CurrencyCode: currency,
		TotalAmount:  platform.FormatNumeric(totalNum, 4),
		Status:       "Draft",
		RowVersion:   newVer,
	})
}

func (s *Server) deletePO(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	n, err := s.repo.DeleteDraft(r.Context(), scope, procurementdb.SoftDeleteDraftPurchaseOrderParams{
		ID:         id,
		RowVersion: rowVer,
		UpdatedBy:  &user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Draft"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) approvePO(w http.ResponseWriter, r *http.Request) {
	if !httpx.HasPermission(r.Context(), "procurement:approve") && !httpx.HasPermission(r.Context(), "procurement.approve") {
		httpx.WriteError(w, r, httpx.Forbidden("izin procurement:approve dibutuhkan"))
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var body struct {
		AccountID    string `json:"accountId,omitempty"`
		CostCenterID string `json:"costCenterId,omitempty"`
		FiscalYear   int32  `json:"fiscalYear,omitempty"`
	}
	if err := httpx.DecodeOptionalJSON(w, r, &body); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	detail, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("purchase order tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if detail.Header.CreatedBy != uuid.Nil && user == detail.Header.CreatedBy && !httpx.HasRole(r.Context(), "SYSTEM_ADMIN") {
		httpx.WriteError(w, r, &httpx.APIError{Status: http.StatusForbidden, Code: "SOD_VIOLATION", Message: "pembuat purchase order tidak boleh menyetujui PO sendiri"})
		return
	}
	fy := body.FiscalYear
	if fy <= 0 {
		if detail.Header.OrderDate.Valid {
			fy = int32(detail.Header.OrderDate.Time.Year())
		} else {
			fy = int32(time.Now().UTC().Year())
		}
	}
	evMap := map[string]any{
		"poNo":         detail.Header.PoNo,
		"vendorId":     detail.Header.VendorID.String(),
		"currencyCode": detail.Header.CurrencyCode,
		"totalAmount":  platform.FormatNumeric(detail.Header.TotalAmount, 4),
		"fiscalYear":   fy,
		"status":       "Approved",
	}
	if acc := strings.TrimSpace(body.AccountID); acc != "" {
		if _, err := httpx.ParseUUIDParam("accountId", acc); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		evMap["accountId"] = acc
	}
	if cc := strings.TrimSpace(body.CostCenterID); cc != "" {
		if _, err := httpx.ParseUUIDParam("costCenterId", cc); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		evMap["costCenterId"] = cc
	}
	payload, _ := json.Marshal(evMap)
	ev := procurementdb.InsertOutboxParams{
		AggregateType:  "PurchaseOrder",
		AggregateID:    detail.Header.ID,
		EventType:      "procurement.po.approved",
		Payload:        payload,
		OrganizationID: detail.Header.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	newVer, n, err := s.repo.ApproveAndEmit(r.Context(), scope, procurementdb.ApprovePurchaseOrderParams{
		ID:         id,
		RowVersion: rowVer,
		UpdatedBy:  &user,
	}, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Draft"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, poDTO{
		ID:         id.String(),
		PoNo:       detail.Header.PoNo,
		Status:     "Approved",
		RowVersion: newVer,
	})
}

func (s *Server) receivePO(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req receiveReq
	if err := httpx.DecodeOptionalJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	detail, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("purchase order tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}

	var whID uuid.UUID
	explicitWH := false
	if strings.TrimSpace(req.WarehouseID) != "" {
		if whID, err = httpx.ParseUUIDParam("warehouseId", req.WarehouseID); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if whID == detail.Header.OrganizationID || whID == platform.SystemOrgID {
			httpx.WriteError(w, r, httpx.BadRequest("warehouseId tidak valid"))
			return
		}
		explicitWH = true
	} else {
		// Deterministic non-org fallback for NOT NULL gr.warehouse_id; omitted from outbox so inventory-service resolves default warehouse.
		whID = uuid.NewSHA1(uuid.NameSpaceOID, []byte("default-warehouse:"+detail.Header.OrganizationID.String()))
	}
	recDate := pgtype.Date{Time: time.Now().UTC(), Valid: true}
	if strings.TrimSpace(req.ReceiptDate) != "" {
		if recDate, err = platform.ParseDate(req.ReceiptDate); err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("receiptDate harus YYYY-MM-DD"))
			return
		}
	}
	grNo := strings.TrimSpace(req.GrNo)
	if grNo == "" {
		grNo = "GR-" + detail.Header.PoNo
	}
	corrID := httpx.CorrelationUUID(r)
	grArg := procurementdb.CreateGoodsReceiptParams{
		OrganizationID:  detail.Header.OrganizationID,
		GrNo:            grNo,
		PurchaseOrderID: detail.Header.ID,
		WarehouseID:     whID,
		ReceiptDate:     recDate,
		Status:          "Received",
		CorrelationID:   corrID,
		CreatedBy:       user,
	}
	buildOutbox := func(lines []procurementdb.ListPurchaseOrderLinesRow, gr procurementdb.CreateGoodsReceiptRow) (procurementdb.InsertOutboxParams, error) {
		evLines := make([]map[string]any, 0, len(lines))
		for _, l := range lines {
			evLines = append(evLines, map[string]any{
				"lineNo":    l.LineNo,
				"itemId":    l.ItemID.String(),
				"qty":       platform.FormatNumeric(l.OrderQty, 4),
				"unitPrice": platform.FormatNumeric(l.UnitPrice, 4),
			})
		}
		evMap := map[string]any{
			"poNo":         detail.Header.PoNo,
			"vendorId":     detail.Header.VendorID.String(),
			"currencyCode": detail.Header.CurrencyCode,
			"grId":         gr.ID.String(),
			"grNo":         gr.GrNo,
			"amount":       platform.FormatNumeric(detail.Header.TotalAmount, 4),
			"status":       "Received",
			"lines":        evLines,
		}
		if explicitWH {
			evMap["warehouseId"] = whID.String()
		}
		payload, _ := json.Marshal(evMap)
		return procurementdb.InsertOutboxParams{
			AggregateType:  "PurchaseOrder",
			AggregateID:    detail.Header.ID,
			EventType:      "procurement.goods.received",
			Payload:        payload,
			OrganizationID: detail.Header.OrganizationID,
			CorrelationID:  corrID,
		}, nil
	}

	newVer, grRow, n, err := s.repo.ReceiveAndEmit(r.Context(), scope, procurementdb.ReceivePurchaseOrderParams{
		ID:         id,
		RowVersion: rowVer,
		UpdatedBy:  &user,
	}, grArg, buildOutbox)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "nomor goods receipt sudah dipakai"))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Approved"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, poDTO{
		ID:         id.String(),
		PoNo:       detail.Header.PoNo,
		Status:     "Received",
		RowVersion: newVer,
		GrID:       grRow.ID.String(),
		GrNo:       grRow.GrNo,
	})
}

func (s *Server) listGRs(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListGoodsReceiptsWithCount(r.Context(), scope, procurementdb.ListGoodsReceiptsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]grDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, grDTO{
			ID:              row.ID.String(),
			GrNo:            row.GrNo,
			PurchaseOrderID: row.PurchaseOrderID.String(),
			WarehouseID:     row.WarehouseID.String(),
			ReceiptDate:     platform.FormatDate(row.ReceiptDate),
			Status:          row.Status,
			RowVersion:      row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getGR(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	detail, err := s.repo.GetGoodsReceipt(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("goods receipt tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	lines := make([]grLineDTO, 0, len(detail.Lines))
	for _, l := range detail.Lines {
		lines = append(lines, grLineDTO{
			ID:          l.ID.String(),
			PoLineID:    l.PoLineID.String(),
			LineNo:      l.LineNo,
			ReceivedQty: platform.FormatNumeric(l.ReceivedQty, 4),
		})
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.Header.RowVersion, grDTO{
		ID:              detail.Header.ID.String(),
		GrNo:            detail.Header.GrNo,
		PurchaseOrderID: detail.Header.PurchaseOrderID.String(),
		WarehouseID:     detail.Header.WarehouseID.String(),
		ReceiptDate:     platform.FormatDate(detail.Header.ReceiptDate),
		Status:          detail.Header.Status,
		RowVersion:      detail.Header.RowVersion,
		Lines:           lines,
	})
}

func (s *Server) createPR(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req prCreateReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if strings.TrimSpace(req.PrNo) == "" || len(req.Lines) == 0 {
		httpx.WriteError(w, r, httpx.BadRequest("prNo dan minimal satu line wajib diisi"))
		return
	}
	reqID := user
	if strings.TrimSpace(req.RequesterID) != "" {
		if reqID, err = httpx.ParseUUIDParam("requesterId", req.RequesterID); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
	}
	var neededDate pgtype.Date
	if strings.TrimSpace(req.NeededDate) != "" {
		if neededDate, err = platform.ParseDate(req.NeededDate); err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("neededDate harus YYYY-MM-DD"))
			return
		}
	}
	var wfRef *string
	if w := strings.TrimSpace(req.WorkflowRef); w != "" {
		wfRef = &w
	}
	lines := make([]procurementdb.CreatePurchaseRequisitionLineParams, 0, len(req.Lines))
	for _, l := range req.Lines {
		if l.LineNo <= 0 {
			httpx.WriteError(w, r, httpx.BadRequest("lineNo harus > 0"))
			return
		}
		itemID, err := httpx.ParseUUIDParam("lines[].itemId", l.ItemID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		qtyNum, err := platform.ParseNumeric(l.RequestQty)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("lines[].requestQty tidak valid"))
			return
		}
		if qtyRat := platform.NumericToRat(qtyNum); qtyRat == nil || qtyRat.Sign() <= 0 {
			httpx.WriteError(w, r, httpx.BadRequest("lines[].requestQty harus > 0"))
			return
		}
		var lDate pgtype.Date
		if strings.TrimSpace(l.NeededDate) != "" {
			if lDate, err = platform.ParseDate(l.NeededDate); err != nil {
				httpx.WriteError(w, r, httpx.BadRequest("lines[].neededDate harus YYYY-MM-DD"))
				return
			}
		}
		lines = append(lines, procurementdb.CreatePurchaseRequisitionLineParams{
			OrganizationID: scope[0],
			LineNo:         l.LineNo,
			ItemID:         itemID,
			RequestQty:     qtyNum,
			NeededDate:     lDate,
			CreatedBy:      user,
		})
	}
	created, err := s.repo.CreateRequisitionWithLines(r.Context(), scope, procurementdb.CreatePurchaseRequisitionParams{
		OrganizationID: scope[0],
		PrNo:           strings.TrimSpace(req.PrNo),
		RequesterID:    reqID,
		WorkflowRef:    wfRef,
		NeededDate:     neededDate,
		CreatedBy:      user,
	}, lines)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "nomor PR sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, prDTO{
		ID:          created.ID.String(),
		PrNo:        strings.TrimSpace(req.PrNo),
		RequesterID: reqID.String(),
		NeededDate:  platform.FormatDate(neededDate),
		Status:      created.Status,
		RowVersion:  created.RowVersion,
	})
}

func (s *Server) listPRs(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var statusFilter *string
	if st := strings.TrimSpace(r.URL.Query().Get("status")); st != "" {
		statusFilter = &st
	}
	rows, total, err := s.repo.ListRequisitionsWithCount(r.Context(), scope, procurementdb.ListPurchaseRequisitionsParams{
		Limit:  limit,
		Offset: offset,
		Status: statusFilter,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]prDTO, 0, len(rows))
	for _, row := range rows {
		dto := prDTO{
			ID:          row.ID.String(),
			PrNo:        row.PrNo,
			RequesterID: row.RequesterID.String(),
			NeededDate:  platform.FormatDate(row.NeededDate),
			Status:      row.Status,
			RowVersion:  row.RowVersion,
		}
		if row.WorkflowRef != nil {
			dto.WorkflowRef = *row.WorkflowRef
		}
		out = append(out, dto)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getPR(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	detail, err := s.repo.GetRequisition(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("purchase requisition tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	lines := make([]prLineDTO, 0, len(detail.Lines))
	for _, l := range detail.Lines {
		lines = append(lines, prLineDTO{
			ID:         l.ID.String(),
			LineNo:     l.LineNo,
			ItemID:     l.ItemID.String(),
			RequestQty: platform.FormatNumeric(l.RequestQty, 4),
			NeededDate: platform.FormatDate(l.NeededDate),
		})
	}
	dto := prDTO{
		ID:          detail.Header.ID.String(),
		PrNo:        detail.Header.PrNo,
		RequesterID: detail.Header.RequesterID.String(),
		NeededDate:  platform.FormatDate(detail.Header.NeededDate),
		Status:      detail.Header.Status,
		RowVersion:  detail.Header.RowVersion,
		Lines:       lines,
	}
	if detail.Header.WorkflowRef != nil {
		dto.WorkflowRef = *detail.Header.WorkflowRef
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.Header.RowVersion, dto)
}

func (s *Server) submitPR(w http.ResponseWriter, r *http.Request) {
	s.transitionPR(w, r, "Draft", "Submitted")
}

func (s *Server) approvePR(w http.ResponseWriter, r *http.Request) {
	s.transitionPR(w, r, "Submitted", "Approved")
}

func (s *Server) transitionPR(w http.ResponseWriter, r *http.Request, fromStatus, toStatus string) {
	if toStatus == "Approved" {
		if !httpx.HasPermission(r.Context(), "procurement:approve") && !httpx.HasPermission(r.Context(), "procurement.approve") {
			httpx.WriteError(w, r, httpx.Forbidden("izin procurement:approve dibutuhkan"))
			return
		}
	} else if err := httpx.RequirePermission(r, "procurement:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if toStatus == "Approved" {
		prDetail, err := s.repo.GetRequisition(r.Context(), scope, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.WriteError(w, r, httpx.NotFound("purchase requisition tidak ditemukan"))
				return
			}
			httpx.WriteError(w, r, httpx.MapDBError(err, ""))
			return
		}
		if (prDetail.Header.CreatedBy != uuid.Nil && user == prDetail.Header.CreatedBy || user == prDetail.Header.RequesterID) && !httpx.HasRole(r.Context(), "SYSTEM_ADMIN") {
			httpx.WriteError(w, r, &httpx.APIError{Status: http.StatusForbidden, Code: "SOD_VIOLATION", Message: "pemohon purchase requisition tidak boleh menyetujui PR sendiri"})
			return
		}
	}
	row, n, err := s.repo.TransitionRequisition(r.Context(), scope, procurementdb.TransitionPurchaseRequisitionParams{
		ID:         id,
		RowVersion: rowVer,
		Status:     toStatus,
		UpdatedBy:  &user,
		Status_2:   fromStatus,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status PR tidak sesuai"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, prDTO{
		ID:         id.String(),
		Status:     row.Status,
		RowVersion: row.RowVersion,
	})
}

func validatePOInput(orgID, user uuid.UUID, req poUpsertReq) (uuid.UUID, *uuid.UUID, pgtype.Date, string, pgtype.Numeric, []procurementdb.CreatePurchaseOrderLineParams, error) {
	vendor, err := httpx.ParseUUIDParam("vendorId", req.VendorID)
	if err != nil {
		return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, err
	}
	var prID *uuid.UUID
	if strings.TrimSpace(req.PurchaseRequisitionID) != "" {
		parsed, err := httpx.ParseUUIDParam("purchaseRequisitionId", req.PurchaseRequisitionID)
		if err != nil {
			return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, err
		}
		prID = &parsed
	}
	orderDate, err := platform.ParseDate(req.OrderDate)
	if err != nil {
		return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("orderDate harus YYYY-MM-DD")
	}
	currency := strings.ToUpper(strings.TrimSpace(req.CurrencyCode))
	if currency == "" {
		currency = "IDR"
	}
	if len(currency) != 3 {
		return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("currencyCode harus 3 karakter ISO")
	}
	totalNum, err := platform.ParseNumeric(req.TotalAmount)
	if err != nil {
		return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("totalAmount tidak valid")
	}
	totalRat := platform.NumericToRat(totalNum)
	if totalRat == nil || totalRat.Sign() <= 0 {
		return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("totalAmount harus > 0")
	}
	if len(req.Lines) == 0 {
		return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("minimal satu PO line wajib diisi")
	}

	lines := make([]procurementdb.CreatePurchaseOrderLineParams, 0, len(req.Lines))
	seenLine := make(map[int32]bool, len(req.Lines))
	lineSum := new(big.Rat)
	for _, l := range req.Lines {
		if l.LineNo <= 0 || seenLine[l.LineNo] {
			return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("lines[].lineNo harus > 0 dan unik")
		}
		seenLine[l.LineNo] = true
		itemID, err := httpx.ParseUUIDParam("lines[].itemId", l.ItemID)
		if err != nil {
			return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, err
		}
		qtyNum, err := platform.ParseNumeric(l.OrderQty)
		if err != nil {
			return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("lines[].orderQty tidak valid")
		}
		qtyRat := platform.NumericToRat(qtyNum)
		if qtyRat == nil || qtyRat.Sign() <= 0 {
			return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("lines[].orderQty harus > 0")
		}
		priceNum, err := platform.ParseNumeric(l.UnitPrice)
		if err != nil {
			return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("lines[].unitPrice tidak valid")
		}
		priceRat := platform.NumericToRat(priceNum)
		if priceRat == nil || priceRat.Sign() < 0 {
			return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("lines[].unitPrice harus >= 0")
		}
		lineSum.Add(lineSum, new(big.Rat).Mul(qtyRat, priceRat))
		lines = append(lines, procurementdb.CreatePurchaseOrderLineParams{
			OrganizationID: orgID,
			LineNo:         l.LineNo,
			ItemID:         itemID,
			OrderQty:       qtyNum,
			UnitPrice:      priceNum,
			CreatedBy:      user,
		})
	}
	if lineSum.Cmp(totalRat) != 0 {
		return uuid.Nil, nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, &httpx.APIError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "TOTAL_MISMATCH",
			Message: "totalAmount harus sama dengan jumlah (orderQty * unitPrice)",
		}
	}
	return vendor, prID, orderDate, currency, totalNum, lines, nil
}
