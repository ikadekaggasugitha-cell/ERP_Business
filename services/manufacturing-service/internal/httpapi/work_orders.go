package httpapi

import (
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/manufacturing-service/internal/manufacturingdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

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
	bom, err := s.repo.GetBOM(r.Context(), scope, bomID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.Unprocessable("BOM_NOT_FOUND", "bill of material tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if !bom.Header.IsActive {
		httpx.WriteError(w, r, httpx.Unprocessable("BOM_INACTIVE", "bill of material tidak aktif"))
		return
	}
	if bom.Header.ProductItemID != prodID {
		httpx.WriteError(w, r, httpx.Unprocessable("BOM_PRODUCT_MISMATCH", "bomId tidak sesuai dengan productItemId work order"))
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
	bom, err := s.repo.GetBOM(r.Context(), scope, bomID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.Unprocessable("BOM_NOT_FOUND", "bill of material tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if !bom.Header.IsActive {
		httpx.WriteError(w, r, httpx.Unprocessable("BOM_INACTIVE", "bill of material tidak aktif"))
		return
	}
	if bom.Header.ProductItemID != prodID {
		httpx.WriteError(w, r, httpx.Unprocessable("BOM_PRODUCT_MISMATCH", "bomId tidak sesuai dengan productItemId work order"))
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
	if err := httpx.DecodeOptionalJSON(w, r, &req); err != nil {
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
	prodRat := platform.NumericToRat(produced)
	if prodRat == nil {
		prodRat = big.NewRat(1, 1)
	}
	unitCostRat := new(big.Rat)
	if strings.TrimSpace(req.UnitCost) != "" {
		ucNum, err := platform.ParseNumeric(req.UnitCost)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("unitCost tidak valid"))
			return
		}
		ucRat := platform.NumericToRat(ucNum)
		if ucRat == nil || ucRat.Sign() < 0 {
			httpx.WriteError(w, r, httpx.BadRequest("unitCost tidak boleh negatif"))
			return
		}
		unitCostRat = ucRat
	}
	var bomLinesPayload []map[string]any
	bomQtySum := new(big.Rat)
	if bomDetail, err := s.repo.GetBOM(r.Context(), scope, wo.BomID); err == nil {
		bomLinesPayload = make([]map[string]any, 0, len(bomDetail.Lines))
		for _, bl := range bomDetail.Lines {
			lr := platform.NumericToRat(bl.Quantity)
			if lr == nil || lr.Sign() <= 0 {
				continue
			}
			bomQtySum.Add(bomQtySum, lr)
			scaled := new(big.Rat).Mul(lr, prodRat)
			scaledNum, _ := platform.RatToNumeric(scaled, 4)
			bomLinesPayload = append(bomLinesPayload, map[string]any{
				"componentItemId": bl.ComponentItemID.String(),
				"quantity":        platform.FormatNumeric(scaledNum, 4),
				"uom":             bl.Uom,
			})
		}
	}
	if unitCostRat.Sign() == 0 {
		if bomQtySum.Sign() > 0 {
			unitCostRat = bomQtySum
		} else {
			unitCostRat = big.NewRat(1, 1)
		}
	}
	totalCostRat := new(big.Rat).Mul(unitCostRat, prodRat)
	unitCostNum, _ := platform.RatToNumeric(unitCostRat, 4)
	totalCostNum, _ := platform.RatToNumeric(totalCostRat, 4)
	evMap := map[string]any{
		"woNo":          wo.WoNo,
		"productItemId": wo.ProductItemID.String(),
		"bomId":         wo.BomID.String(),
		"plannedQty":    platform.FormatNumeric(wo.PlannedQty, 4),
		"producedQty":   platform.FormatNumeric(produced, 4),
		"unitCost":      platform.FormatNumeric(unitCostNum, 4),
		"totalCost":     platform.FormatNumeric(totalCostNum, 4),
		"bomLines":      bomLinesPayload,
		"status":        "Completed",
	}
	if wh := strings.TrimSpace(req.WarehouseID); wh != "" {
		if _, err := httpx.ParseUUIDParam("warehouseId", wh); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		evMap["warehouseId"] = wh
	}
	payload, _ := json.Marshal(evMap)
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
