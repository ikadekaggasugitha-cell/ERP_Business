// Package httpapi is inventory-service's REST layer (SDD BAB 4): warehouses,
// stock balances, stock movements (Receipt/Issue/Adjustment), and reservations.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/inventorydb"
	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Store abstracts inventory repository methods for handlers and unit tests.
type Store interface {
	CreateWarehouse(ctx context.Context, scope []uuid.UUID, arg inventorydb.CreateWarehouseParams) (inventorydb.CreateWarehouseRow, error)
	ListWarehousesWithCount(ctx context.Context, scope []uuid.UUID, arg inventorydb.ListWarehousesParams) ([]inventorydb.ListWarehousesRow, int64, error)
	ListStockBalancesWithCount(ctx context.Context, scope []uuid.UUID, arg inventorydb.ListStockBalancesParams) ([]inventorydb.ListStockBalancesRow, int64, error)
	ListStockMovementsWithCount(ctx context.Context, scope []uuid.UUID, arg inventorydb.ListStockMovementsParams) ([]inventorydb.ListStockMovementsRow, int64, error)
	RecordStockMovement(ctx context.Context, scope []uuid.UUID, arg inventorydb.CreateStockMovementParams, unitCost pgtype.Numeric) (inventorydb.CreateStockMovementRow, error)
	CreateReservationAndEmit(ctx context.Context, scope []uuid.UUID, arg inventorydb.CreateReservationParams, ev inventorydb.InsertOutboxParams) (inventorydb.CreateReservationRow, error)
	IssueAndEmit(ctx context.Context, scope []uuid.UUID, arg inventorydb.IssueReservationParams, corrID uuid.UUID, moveDate pgtype.Date, ev inventorydb.InsertOutboxParams) (int32, int64, error)
	ReleaseReservation(ctx context.Context, scope []uuid.UUID, arg inventorydb.ReleaseReservationParams) (int32, int64, error)
	GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (inventorydb.GetReservationRow, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, arg inventorydb.ListReservationsParams) ([]inventorydb.ListReservationsRow, int64, error)
}

// Server serves HTTP endpoints for inventory-service.
type Server struct {
	repo Store
	idem *httpx.IdemStore
}

// NewServer constructs an HTTP Server backed by repo.
func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

// Register attaches all inventory routes onto mux.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /warehouses", s.createWarehouse)
	mux.HandleFunc("GET /warehouses", s.listWarehouses)
	mux.HandleFunc("GET /stock-balances", s.listStockBalances)
	mux.HandleFunc("POST /stock-movements", s.createStockMovement)
	mux.HandleFunc("GET /stock-movements", s.listStockMovements)
	mux.HandleFunc("POST /reservations", s.createReservation)
	mux.HandleFunc("GET /reservations", s.listReservations)
	mux.HandleFunc("GET /reservations/{id}", s.getReservation)
	mux.HandleFunc("POST /reservations/{id}/issue", s.issueReservation)
	mux.HandleFunc("POST /reservations/{id}/release", s.releaseReservation)
}

type warehouseReq struct {
	Code              string `json:"code"`
	Name              string `json:"name"`
	OrganizationRefID string `json:"organizationRefId,omitempty"`
}

type warehouseDTO struct {
	ID                string `json:"id"`
	Code              string `json:"code"`
	Name              string `json:"name"`
	OrganizationRefID string `json:"organizationRefId"`
	RowVersion        int32  `json:"rowVersion"`
}

type stockBalanceDTO struct {
	ID           string `json:"id"`
	WarehouseID  string `json:"warehouseId"`
	ItemID       string `json:"itemId"`
	OnHandQty    string `json:"onHandQty"`
	ReservedQty  string `json:"reservedQty"`
	AvailableQty string `json:"availableQty"`
	AvgCost      string `json:"avgCost"`
	RowVersion   int32  `json:"rowVersion"`
}

type stockMovementReq struct {
	MovementDate      string `json:"movementDate,omitempty"`
	WarehouseID       string `json:"warehouseId"`
	ItemID            string `json:"itemId"`
	MovementType      string `json:"movementType"`
	Quantity          string `json:"quantity"`
	UnitCost          string `json:"unitCost,omitempty"`
	SourceDocumentRef string `json:"sourceDocumentRef,omitempty"`
}

type stockMovementDTO struct {
	ID                string `json:"id"`
	MovementDate      string `json:"movementDate"`
	WarehouseID       string `json:"warehouseId,omitempty"`
	ItemID            string `json:"itemId,omitempty"`
	MovementType      string `json:"movementType,omitempty"`
	Quantity          string `json:"quantity,omitempty"`
	SourceDocumentRef string `json:"sourceDocumentRef,omitempty"`
	RowVersion        int32  `json:"rowVersion"`
}

type reservationReq struct {
	ItemID        string `json:"itemId"`
	WarehouseID   string `json:"warehouseId"`
	ReservedQty   string `json:"reservedQty"`
	SalesOrderRef string `json:"salesOrderRef,omitempty"`
}

type reservationDTO struct {
	ID            string `json:"id"`
	ItemID        string `json:"itemId,omitempty"`
	WarehouseID   string `json:"warehouseId,omitempty"`
	ReservedQty   string `json:"reservedQty,omitempty"`
	SalesOrderRef string `json:"salesOrderRef,omitempty"`
	Status        string `json:"status,omitempty"`
	RowVersion    int32  `json:"rowVersion"`
}

func (s *Server) createWarehouse(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "inventory:write"); err != nil {
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
	var req warehouseReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan name wajib diisi"))
		return
	}
	orgRef := scope[0]
	if strings.TrimSpace(req.OrganizationRefID) != "" {
		if orgRef, err = httpx.ParseUUIDParam("organizationRefId", req.OrganizationRefID); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
	}
	created, err := s.repo.CreateWarehouse(r.Context(), scope, inventorydb.CreateWarehouseParams{
		OrganizationID:    scope[0],
		Code:              strings.TrimSpace(req.Code),
		Name:              strings.TrimSpace(req.Name),
		OrganizationRefID: orgRef,
		CreatedBy:         user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode warehouse sudah dipakai"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusCreated, created.RowVersion, warehouseDTO{
		ID:                created.ID.String(),
		Code:              created.Code,
		Name:              created.Name,
		OrganizationRefID: created.OrganizationRefID.String(),
		RowVersion:        created.RowVersion,
	})
}

func (s *Server) listWarehouses(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "inventory:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListWarehousesWithCount(r.Context(), scope, inventorydb.ListWarehousesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]warehouseDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, warehouseDTO{
			ID:                row.ID.String(),
			Code:              row.Code,
			Name:              row.Name,
			OrganizationRefID: row.OrganizationRefID.String(),
			RowVersion:        row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) listStockBalances(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "inventory:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	whID, itemID, err := parseStockFilters(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListStockBalancesWithCount(r.Context(), scope, inventorydb.ListStockBalancesParams{
		Limit:       limit,
		Offset:      offset,
		WarehouseID: whID,
		ItemID:      itemID,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]stockBalanceDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, stockBalanceDTO{
			ID:           row.ID.String(),
			WarehouseID:  row.WarehouseID.String(),
			ItemID:       row.ItemID.String(),
			OnHandQty:    platform.FormatNumeric(row.OnHandQty, 4),
			ReservedQty:  platform.FormatNumeric(row.ReservedQty, 4),
			AvailableQty: platform.FormatNumeric(row.AvailableQty, 4),
			AvgCost:      platform.FormatNumeric(row.AvgCost, 4),
			RowVersion:   row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) createStockMovement(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "inventory:write"); err != nil {
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
	var req stockMovementReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	whID, err := httpx.ParseUUIDParam("warehouseId", req.WarehouseID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	itemID, err := httpx.ParseUUIDParam("itemId", req.ItemID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	switch req.MovementType {
	case "Receipt", "Issue", "Adjustment":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("movementType harus Receipt, Issue, atau Adjustment"))
		return
	}
	qtyNum, err := platform.ParseNumeric(req.Quantity)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("quantity tidak valid"))
		return
	}
	qtyRat := platform.NumericToRat(qtyNum)
	if qtyRat == nil || qtyRat.Sign() == 0 {
		httpx.WriteError(w, r, httpx.BadRequest("quantity tidak boleh 0"))
		return
	}
	if req.MovementType == "Receipt" && qtyRat.Sign() < 0 {
		httpx.WriteError(w, r, httpx.BadRequest("receipt quantity harus > 0"))
		return
	}
	unitCost, err := platform.ParseOptionalNumeric(req.UnitCost)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("unitCost tidak valid"))
		return
	}
	moveDate := pgtype.Date{Time: time.Now().UTC(), Valid: true}
	if strings.TrimSpace(req.MovementDate) != "" {
		if moveDate, err = platform.ParseDate(req.MovementDate); err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("movementDate harus YYYY-MM-DD"))
			return
		}
	}
	var docRef *string
	if d := strings.TrimSpace(req.SourceDocumentRef); d != "" {
		docRef = &d
	}
	created, err := s.repo.RecordStockMovement(r.Context(), scope, inventorydb.CreateStockMovementParams{
		OrganizationID:    scope[0],
		MovementDate:      moveDate,
		WarehouseID:       whID,
		ItemID:            itemID,
		MovementType:      req.MovementType,
		Quantity:          qtyNum,
		SourceDocumentRef: docRef,
		CorrelationID:     httpx.CorrelationUUID(r),
		CreatedBy:         user,
	}, unitCost)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientStock) {
			httpx.WriteError(w, r, httpx.Unprocessable("INSUFFICIENT_STOCK", "stok tersedia tidak mencukupi"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, stockMovementDTO{
		ID:                created.ID.String(),
		MovementDate:      platform.FormatDate(created.MovementDate),
		WarehouseID:       whID.String(),
		ItemID:            itemID.String(),
		MovementType:      req.MovementType,
		Quantity:          platform.FormatNumeric(qtyNum, 4),
		SourceDocumentRef: req.SourceDocumentRef,
		RowVersion:        created.RowVersion,
	})
}

func (s *Server) listStockMovements(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "inventory:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	whID, itemID, err := parseStockFilters(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListStockMovementsWithCount(r.Context(), scope, inventorydb.ListStockMovementsParams{
		Limit:       limit,
		Offset:      offset,
		WarehouseID: whID,
		ItemID:      itemID,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]stockMovementDTO, 0, len(rows))
	for _, row := range rows {
		dto := stockMovementDTO{
			ID:           row.ID.String(),
			MovementDate: platform.FormatDate(row.MovementDate),
			WarehouseID:  row.WarehouseID.String(),
			ItemID:       row.ItemID.String(),
			MovementType: row.MovementType,
			Quantity:     platform.FormatNumeric(row.Quantity, 4),
			RowVersion:   row.RowVersion,
		}
		if row.SourceDocumentRef != nil {
			dto.SourceDocumentRef = *row.SourceDocumentRef
		}
		out = append(out, dto)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) createReservation(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "inventory:write"); err != nil {
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
	var req reservationReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	itemID, err := httpx.ParseUUIDParam("itemId", req.ItemID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	whID, err := httpx.ParseUUIDParam("warehouseId", req.WarehouseID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	qtyNum, err := platform.ParseNumeric(req.ReservedQty)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("reservedQty tidak valid"))
		return
	}
	qtyRat := platform.NumericToRat(qtyNum)
	if qtyRat == nil || qtyRat.Sign() <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("reservedQty harus > 0"))
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"itemId":        itemID.String(),
		"warehouseId":   whID.String(),
		"reservedQty":   platform.FormatNumeric(qtyNum, 4),
		"salesOrderRef": req.SalesOrderRef,
	})
	ev := inventorydb.InsertOutboxParams{
		AggregateType:  "StockReservation",
		EventType:      "inventory.stock.reserved",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	created, err := s.repo.CreateReservationAndEmit(r.Context(), scope, inventorydb.CreateReservationParams{
		OrganizationID: scope[0],
		ItemID:         itemID,
		WarehouseID:    whID,
		ReservedQty:    qtyNum,
		SalesOrderRef:  strings.TrimSpace(req.SalesOrderRef),
		CreatedBy:      user,
	}, ev)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientStock) {
			httpx.WriteError(w, r, httpx.Unprocessable("INSUFFICIENT_STOCK", "stok tersedia tidak mencukupi"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, reservationDTO{
		ID:            created.ID.String(),
		ItemID:        itemID.String(),
		WarehouseID:   whID.String(),
		ReservedQty:   platform.FormatNumeric(qtyNum, 4),
		SalesOrderRef: strings.TrimSpace(req.SalesOrderRef),
		Status:        created.Status,
		RowVersion:    created.RowVersion,
	})
}

func (s *Server) issueReservation(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "inventory:write"); err != nil {
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
	var body struct {
		Amount string `json:"amount"`
	}
	if err := httpx.DecodeOptionalJSON(w, r, &body); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	amountNum, err := platform.ParseOptionalNumeric(body.Amount)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("amount tidak valid"))
		return
	}
	res, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("reservation tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"itemId":      res.ItemID.String(),
		"warehouseId": res.WarehouseID.String(),
		"issuedQty":   platform.FormatNumeric(res.ReservedQty, 4),
		"amount":      platform.FormatNumeric(amountNum, 4),
		"reference":   res.SalesOrderRef,
	})
	corrID := httpx.CorrelationUUID(r)
	ev := inventorydb.InsertOutboxParams{
		AggregateType:  "StockReservation",
		AggregateID:    id,
		EventType:      "inventory.goods.issued",
		Payload:        payload,
		OrganizationID: res.OrganizationID,
		CorrelationID:  corrID,
	}
	moveDate := pgtype.Date{Time: time.Now().UTC(), Valid: true}
	newVer, n, err := s.repo.IssueAndEmit(r.Context(), scope, inventorydb.IssueReservationParams{
		ID:         id,
		RowVersion: rowVer,
		UpdatedBy:  &user,
	}, corrID, moveDate, ev)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientStock) {
			httpx.WriteError(w, r, httpx.Unprocessable("INSUFFICIENT_STOCK", "stok on-hand tidak mencukupi"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Reserved"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, reservationDTO{
		ID:         id.String(),
		Status:     "Consumed",
		RowVersion: newVer,
	})
}

func (s *Server) releaseReservation(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "inventory:write"); err != nil {
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
	newVer, n, err := s.repo.ReleaseReservation(r.Context(), scope, inventorydb.ReleaseReservationParams{
		ID:         id,
		RowVersion: rowVer,
		UpdatedBy:  &user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Reserved"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, reservationDTO{
		ID:         id.String(),
		Status:     "Released",
		RowVersion: newVer,
	})
}

func (s *Server) getReservation(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "inventory:read"); err != nil {
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
	res, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("reservation tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, res.RowVersion, reservationDTO{
		ID:            res.ID.String(),
		ItemID:        res.ItemID.String(),
		WarehouseID:   res.WarehouseID.String(),
		ReservedQty:   platform.FormatNumeric(res.ReservedQty, 4),
		SalesOrderRef: res.SalesOrderRef,
		Status:        res.Status,
		RowVersion:    res.RowVersion,
	})
}

func (s *Server) listReservations(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "inventory:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListWithCount(r.Context(), scope, inventorydb.ListReservationsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]reservationDTO, 0, len(rows))
	for _, res := range rows {
		out = append(out, reservationDTO{
			ID:            res.ID.String(),
			ItemID:        res.ItemID.String(),
			WarehouseID:   res.WarehouseID.String(),
			ReservedQty:   platform.FormatNumeric(res.ReservedQty, 4),
			SalesOrderRef: res.SalesOrderRef,
			Status:        res.Status,
			RowVersion:    res.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func parseStockFilters(r *http.Request) (*uuid.UUID, *uuid.UUID, error) {
	var whPtr, itemPtr *uuid.UUID
	if w := strings.TrimSpace(r.URL.Query().Get("warehouseId")); w != "" {
		id, err := httpx.ParseUUIDParam("warehouseId", w)
		if err != nil {
			return nil, nil, err
		}
		whPtr = &id
	}
	if it := strings.TrimSpace(r.URL.Query().Get("itemId")); it != "" {
		id, err := httpx.ParseUUIDParam("itemId", it)
		if err != nil {
			return nil, nil, err
		}
		itemPtr = &id
	}
	return whPtr, itemPtr, nil
}
