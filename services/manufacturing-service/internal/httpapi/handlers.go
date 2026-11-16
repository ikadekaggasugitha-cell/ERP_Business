// Package httpapi is manufacturing-service's REST layer (SDD BAB 4): Bills of Material
// and Work Orders (Planned CRUD, Release -> manufacturing.workorder.released, Complete -> manufacturing.workorder.completed).
package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/manufacturing-service/internal/manufacturingdb"
	"github.com/sinergiinvestasiproperti/erp/services/manufacturing-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
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
	mux.HandleFunc("GET /work-orders/{id}", s.getWO)
	mux.HandleFunc("PUT /work-orders/{id}", s.updateWO)
	mux.HandleFunc("DELETE /work-orders/{id}", s.deleteWO)
	mux.HandleFunc("POST /work-orders/{id}/release", s.releaseWO)
	mux.HandleFunc("POST /work-orders/{id}/complete", s.completeWO)
}

type bomLineReq struct {
	LineNo          int32  `json:"lineNo"`
	ComponentItemID string `json:"componentItemId"`
	Quantity        string `json:"quantity"`
	Uom             string `json:"uom"`
}

type bomReq struct {
	Code          string       `json:"code"`
	ProductItemID string       `json:"productItemId"`
	Version       int32        `json:"version,omitempty"`
	Lines         []bomLineReq `json:"lines,omitempty"`
}

type bomLineDTO struct {
	ID              string `json:"id"`
	LineNo          int32  `json:"lineNo"`
	ComponentItemID string `json:"componentItemId"`
	Quantity        string `json:"quantity"`
	Uom             string `json:"uom"`
}

type bomDTO struct {
	ID            string       `json:"id"`
	Code          string       `json:"code"`
	ProductItemID string       `json:"productItemId"`
	Version       int32        `json:"version"`
	IsActive      bool         `json:"isActive"`
	RowVersion    int32        `json:"rowVersion"`
	Lines         []bomLineDTO `json:"lines,omitempty"`
}

type woUpsertReq struct {
	WoNo          string `json:"woNo"`
	ProductItemID string `json:"productItemId"`
	BomID         string `json:"bomId"`
	PlannedQty    string `json:"plannedQty"`
	SalesOrderRef string `json:"salesOrderRef,omitempty"`
}

type completeReq struct {
	ProducedQty string `json:"producedQty,omitempty"`
}

type woDTO struct {
	ID            string `json:"id"`
	WoNo          string `json:"woNo,omitempty"`
	ProductItemID string `json:"productItemId,omitempty"`
	BomID         string `json:"bomId,omitempty"`
	PlannedQty    string `json:"plannedQty,omitempty"`
	ProducedQty   string `json:"producedQty,omitempty"`
	SalesOrderRef string `json:"salesOrderRef,omitempty"`
	Status        string `json:"status"`
	RowVersion    int32  `json:"rowVersion"`
}

func (s *Server) createBOM(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "manufacturing:write"); err != nil {
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
	var req bomReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code BOM wajib diisi"))
		return
	}
	prodID, err := httpx.ParseUUIDParam("productItemId", req.ProductItemID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver := req.Version
	if ver <= 0 {
		ver = 1
	}
	lines := make([]manufacturingdb.CreateBomLineParams, 0, len(req.Lines))
	for _, l := range req.Lines {
		if l.LineNo <= 0 || strings.TrimSpace(l.Uom) == "" {
			httpx.WriteError(w, r, httpx.BadRequest("lines[].lineNo harus > 0 dan uom wajib diisi"))
			return
		}
		compID, err := httpx.ParseUUIDParam("lines[].componentItemId", l.ComponentItemID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		qtyNum, err := platform.ParseNumeric(l.Quantity)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("lines[].quantity tidak valid"))
			return
		}
		if qr := platform.NumericToRat(qtyNum); qr == nil || qr.Sign() <= 0 {
			httpx.WriteError(w, r, httpx.BadRequest("lines[].quantity harus > 0"))
			return
		}
		lines = append(lines, manufacturingdb.CreateBomLineParams{
			OrganizationID:  scope[0],
			LineNo:          l.LineNo,
			ComponentItemID: compID,
			Quantity:        qtyNum,
			Uom:             strings.TrimSpace(l.Uom),
			CreatedBy:       user,
		})
	}
	created, err := s.repo.CreateBOMWithLines(r.Context(), scope, manufacturingdb.CreateBillOfMaterialParams{
		OrganizationID: scope[0],
		Code:           strings.TrimSpace(req.Code),
		ProductItemID:  prodID,
		Version:        ver,
		CreatedBy:      user,
	}, lines)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode dan versi BOM sudah dipakai"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusCreated, created.RowVersion, bomDTO{
		ID:            created.ID.String(),
		Code:          created.Code,
		ProductItemID: created.ProductItemID.String(),
		Version:       created.Version,
		IsActive:      created.IsActive,
		RowVersion:    created.RowVersion,
	})
}

func (s *Server) listBOMs(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "manufacturing:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListBOMsWithCount(r.Context(), scope, manufacturingdb.ListBillsOfMaterialParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]bomDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, bomDTO{
			ID:            row.ID.String(),
			Code:          row.Code,
			ProductItemID: row.ProductItemID.String(),
			Version:       row.Version,
			IsActive:      row.IsActive,
			RowVersion:    row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getBOM(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "manufacturing:read"); err != nil {
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
	detail, err := s.repo.GetBOM(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("BOM tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	lines := make([]bomLineDTO, 0, len(detail.Lines))
	for _, l := range detail.Lines {
		lines = append(lines, bomLineDTO{
			ID:              l.ID.String(),
			LineNo:          l.LineNo,
			ComponentItemID: l.ComponentItemID.String(),
			Quantity:        platform.FormatNumeric(l.Quantity, 4),
			Uom:             l.Uom,
		})
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.Header.RowVersion, bomDTO{
		ID:            detail.Header.ID.String(),
		Code:          detail.Header.Code,
		ProductItemID: detail.Header.ProductItemID.String(),
		Version:       detail.Header.Version,
		IsActive:      detail.Header.IsActive,
		RowVersion:    detail.Header.RowVersion,
		Lines:         lines,
	})
}

func (s *Server) createWO(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "manufacturing:write"); err != nil {
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
	var req woUpsertReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if strings.TrimSpace(req.WoNo) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("woNo wajib diisi"))
		return
	}
	prodID, err := httpx.ParseUUIDParam("productItemId", req.ProductItemID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	bomID, err := httpx.ParseUUIDParam("bomId", req.BomID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	qtyNum, err := platform.ParseNumeric(req.PlannedQty)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("plannedQty tidak valid"))
		return
	}
	if qr := platform.NumericToRat(qtyNum); qr == nil || qr.Sign() <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("plannedQty harus > 0"))
		return
	}
	var soRef *string
	if sr := strings.TrimSpace(req.SalesOrderRef); sr != "" {
		soRef = &sr
	}
	created, err := s.repo.CreateWorkOrder(r.Context(), scope, manufacturingdb.CreateWorkOrderParams{
		OrganizationID: scope[0],
		WoNo:           strings.TrimSpace(req.WoNo),
		ProductItemID:  prodID,
		BomID:          bomID,
		PlannedQty:     qtyNum,
		SalesOrderRef:  soRef,
		CorrelationID:  httpx.CorrelationUUID(r),
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "nomor work order sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, woDTO{
		ID:            created.ID.String(),
		WoNo:          strings.TrimSpace(req.WoNo),
		ProductItemID: prodID.String(),
		BomID:         bomID.String(),
		PlannedQty:    platform.FormatNumeric(qtyNum, 4),
		SalesOrderRef: req.SalesOrderRef,
		Status:        created.Status,
		RowVersion:    created.RowVersion,
	})
}

func (s *Server) listWOs(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "manufacturing:read"); err != nil {
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
	rows, total, err := s.repo.ListWithCount(r.Context(), scope, manufacturingdb.ListWorkOrdersParams{
		Limit:  limit,
		Offset: offset,
		Status: statusFilter,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]woDTO, 0, len(rows))
	for _, wo := range rows {
		dto := woDTO{
			ID:            wo.ID.String(),
			WoNo:          wo.WoNo,
			ProductItemID: wo.ProductItemID.String(),
			BomID:         wo.BomID.String(),
			PlannedQty:    platform.FormatNumeric(wo.PlannedQty, 4),
			ProducedQty:   platform.FormatNumeric(wo.ProducedQty, 4),
			Status:        wo.Status,
			RowVersion:    wo.RowVersion,
		}
		if wo.SalesOrderRef != nil {
			dto.SalesOrderRef = *wo.SalesOrderRef
		}
		out = append(out, dto)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getWO(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "manufacturing:read"); err != nil {
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
	wo, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("work order tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	dto := woDTO{
		ID:            wo.ID.String(),
		WoNo:          wo.WoNo,
		ProductItemID: wo.ProductItemID.String(),
		BomID:         wo.BomID.String(),
		PlannedQty:    platform.FormatNumeric(wo.PlannedQty, 4),
		ProducedQty:   platform.FormatNumeric(wo.ProducedQty, 4),
		Status:        wo.Status,
		RowVersion:    wo.RowVersion,
	}
	if wo.SalesOrderRef != nil {
		dto.SalesOrderRef = *wo.SalesOrderRef
	}
	httpx.WriteIDResponse(w, http.StatusOK, wo.RowVersion, dto)
}

func (s *Server) updateWO(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "manufacturing:write"); err != nil {
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
	var req woUpsertReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	prodID, err := httpx.ParseUUIDParam("productItemId", req.ProductItemID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	bomID, err := httpx.ParseUUIDParam("bomId", req.BomID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	qtyNum, err := platform.ParseNumeric(req.PlannedQty)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("plannedQty tidak valid"))
		return
	}
	if qr := platform.NumericToRat(qtyNum); qr == nil || qr.Sign() <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("plannedQty harus > 0"))
		return
	}
	var soRef *string
	if sr := strings.TrimSpace(req.SalesOrderRef); sr != "" {
		soRef = &sr
	}
	newVer, n, err := s.repo.UpdatePlanned(r.Context(), scope, manufacturingdb.UpdatePlannedWorkOrderParams{
		ID:            id,
		RowVersion:    rowVer,
		ProductItemID: prodID,
		BomID:         bomID,
		PlannedQty:    qtyNum,
		SalesOrderRef: soRef,
		UpdatedBy:     &user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Planned"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, woDTO{
		ID:            id.String(),
		ProductItemID: prodID.String(),
		BomID:         bomID.String(),
		PlannedQty:    platform.FormatNumeric(qtyNum, 4),
		SalesOrderRef: req.SalesOrderRef,
		Status:        "Planned",
		RowVersion:    newVer,
	})
}

func (s *Server) deleteWO(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "manufacturing:write"); err != nil {
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
	n, err := s.repo.DeletePlanned(r.Context(), scope, manufacturingdb.SoftDeletePlannedWorkOrderParams{
		ID:         id,
		RowVersion: rowVer,
		UpdatedBy:  &user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Planned"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) releaseWO(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "manufacturing:write"); err != nil {
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
	wo, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("work order tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"woNo":          wo.WoNo,
		"productItemId": wo.ProductItemID.String(),
		"bomId":         wo.BomID.String(),
		"plannedQty":    platform.FormatNumeric(wo.PlannedQty, 4),
		"status":        "Released",
	})
	ev := manufacturingdb.InsertOutboxParams{
		AggregateType:  "WorkOrder",
		AggregateID:    wo.ID,
		EventType:      "manufacturing.workorder.released",
		Payload:        payload,
		OrganizationID: wo.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	newVer, n, err := s.repo.ReleaseAndEmit(r.Context(), scope, manufacturingdb.ReleaseWorkOrderParams{
		ID:         id,
		RowVersion: rowVer,
		UpdatedBy:  &user,
	}, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Planned"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, woDTO{
		ID:         id.String(),
		WoNo:       wo.WoNo,
		Status:     "Released",
		RowVersion: newVer,
	})
}

func (s *Server) completeWO(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "manufacturing:write"); err != nil {
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
	var req completeReq
	if err := decodeOptionalJSON(r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	wo, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("work order tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	produced := wo.PlannedQty
	if strings.TrimSpace(req.ProducedQty) != "" {
		produced, err = platform.ParseNumeric(req.ProducedQty)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("producedQty tidak valid"))
			return
		}
		if pr := platform.NumericToRat(produced); pr == nil || pr.Sign() <= 0 {
			httpx.WriteError(w, r, httpx.BadRequest("producedQty harus > 0"))
			return
		}
	}
	payload, _ := json.Marshal(map[string]any{
		"woNo":          wo.WoNo,
		"productItemId": wo.ProductItemID.String(),
		"bomId":         wo.BomID.String(),
		"plannedQty":    platform.FormatNumeric(wo.PlannedQty, 4),
		"producedQty":   platform.FormatNumeric(produced, 4),
		"status":        "Completed",
	})
	ev := manufacturingdb.InsertOutboxParams{
		AggregateType:  "WorkOrder",
		AggregateID:    wo.ID,
		EventType:      "manufacturing.workorder.completed",
		Payload:        payload,
		OrganizationID: wo.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	newVer, n, err := s.repo.CompleteAndEmit(r.Context(), scope, manufacturingdb.CompleteWorkOrderParams{
		ID:          id,
		RowVersion:  rowVer,
		ProducedQty: produced,
		UpdatedBy:   &user,
	}, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status belum Released"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, woDTO{
		ID:          id.String(),
		WoNo:        wo.WoNo,
		ProducedQty: platform.FormatNumeric(produced, 4),
		Status:      "Completed",
		RowVersion:  newVer,
	})
}

func decodeOptionalJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return nil
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return httpx.BadRequest("gagal membaca body")
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return httpx.BadRequest("JSON tidak valid: " + err.Error())
	}
	return nil
}
