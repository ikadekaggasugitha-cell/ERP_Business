package httpapi

import (
	"errors"
	"fmt"
	"math/big"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func (s *Server) mrpPlan(w http.ResponseWriter, r *http.Request) {
	if !httpx.HasPermission(r.Context(), "manufacturing:read") && !httpx.HasPermission(r.Context(), "manufacturing:write") {
		httpx.WriteError(w, r, httpx.Forbidden("izin manufacturing:read dibutuhkan"))
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		BomID          string            `json:"bomId"`
		TargetQty      string            `json:"targetQty"`
		AvailableStock map[string]string `json:"availableStock,omitempty"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	bomID, err := httpx.ParseUUIDParam("bomId", req.BomID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	targetNum, err := platform.ParseNumeric(req.TargetQty)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("targetQty tidak valid"))
		return
	}
	targetRat := platform.NumericToRat(targetNum)
	if targetRat == nil || targetRat.Sign() <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("targetQty harus > 0"))
		return
	}

	bom, err := s.repo.GetBOM(r.Context(), scope, bomID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("BOM tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}

	fmtRat := func(v *big.Rat, scale int) string {
		n, _ := platform.RatToNumeric(v, scale)
		return platform.FormatNumeric(n, scale)
	}

	reqs := make([]map[string]any, 0, len(bom.Lines))
	shortageCount := 0
	for _, bl := range bom.Lines {
		perUnit := platform.NumericToRat(bl.Quantity)
		if perUnit == nil {
			perUnit = new(big.Rat)
		}
		reqQty := new(big.Rat).Mul(perUnit, targetRat)
		availQty := new(big.Rat)
		if rawAvail, ok := req.AvailableStock[bl.ComponentItemID.String()]; ok && rawAvail != "" {
			an, err := platform.ParseNumeric(rawAvail)
			if err != nil {
				httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("availableStock[%s] tidak valid", bl.ComponentItemID)))
				return
			}
			if ar := platform.NumericToRat(an); ar != nil && ar.Sign() > 0 {
				availQty = ar
			}
		}
		shortage := new(big.Rat).Sub(reqQty, availQty)
		if shortage.Sign() < 0 {
			shortage = new(big.Rat)
		}
		needsPurchase := shortage.Sign() > 0
		if needsPurchase {
			shortageCount++
		}
		reqs = append(reqs, map[string]any{
			"lineNo":           bl.LineNo,
			"componentItemId":  bl.ComponentItemID.String(),
			"uom":              bl.Uom,
			"qtyPerUnit":       fmtRat(perUnit, 4),
			"requiredQty":      fmtRat(reqQty, 4),
			"availableQty":     fmtRat(availQty, 4),
			"netShortageQty":   fmtRat(shortage, 4),
			"purchaseRequired": needsPurchase,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"bomId":             bom.Header.ID.String(),
		"bomCode":           bom.Header.Code,
		"productItemId":     bom.Header.ProductItemID.String(),
		"targetQty":         fmtRat(targetRat, 4),
		"shortageItemCount": shortageCount,
		"canFulfillFromInv": shortageCount == 0,
		"requirements":      reqs,
	})
}

func (s *Server) workOrderVariance(w http.ResponseWriter, r *http.Request) {
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

	planned := platform.NumericToRat(wo.PlannedQty)
	if planned == nil {
		planned = new(big.Rat)
	}
	produced := platform.NumericToRat(wo.ProducedQty)
	if produced == nil {
		produced = new(big.Rat)
	}
	scrap := new(big.Rat).Sub(planned, produced)
	if scrap.Sign() < 0 {
		scrap = new(big.Rat)
	}
	hundred := big.NewRat(100, 1)
	yieldPct := new(big.Rat)
	scrapPct := new(big.Rat)
	if planned.Sign() > 0 {
		yieldPct = new(big.Rat).Mul(new(big.Rat).Quo(produced, planned), hundred)
		scrapPct = new(big.Rat).Mul(new(big.Rat).Quo(scrap, planned), hundred)
	}

	// Optional standardUnitCost query parameter for cost variance calculation
	stdUnitCost := big.NewRat(1, 1)
	if rawCost := r.URL.Query().Get("standardUnitCost"); rawCost != "" {
		if cn, err := platform.ParseNumeric(rawCost); err == nil {
			if cr := platform.NumericToRat(cn); cr != nil && cr.Sign() >= 0 {
				stdUnitCost = cr
			}
		}
	} else if bom, err := s.repo.GetBOM(r.Context(), scope, wo.BomID); err == nil {
		bomSum := new(big.Rat)
		for _, bl := range bom.Lines {
			if lr := platform.NumericToRat(bl.Quantity); lr != nil && lr.Sign() > 0 {
				bomSum.Add(bomSum, lr)
			}
		}
		if bomSum.Sign() > 0 {
			stdUnitCost = bomSum
		}
	}

	plannedCost := new(big.Rat).Mul(planned, stdUnitCost)
	producedCost := new(big.Rat).Mul(produced, stdUnitCost)
	scrapCostVariance := new(big.Rat).Sub(plannedCost, producedCost)

	fmtRat := func(v *big.Rat, scale int) string {
		n, _ := platform.RatToNumeric(v, scale)
		return platform.FormatNumeric(n, scale)
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"workOrderId":       wo.ID.String(),
		"woNo":              wo.WoNo,
		"status":            wo.Status,
		"productItemId":     wo.ProductItemID.String(),
		"plannedQty":        fmtRat(planned, 4),
		"producedQty":       fmtRat(produced, 4),
		"scrapQty":          fmtRat(scrap, 4),
		"yieldPct":          fmtRat(yieldPct, 2),
		"scrapPct":          fmtRat(scrapPct, 2),
		"standardUnitCost":  fmtRat(stdUnitCost, 4),
		"plannedCost":       fmtRat(plannedCost, 4),
		"producedCost":      fmtRat(producedCost, 4),
		"costVariance":      fmtRat(scrapCostVariance, 4),
	})
}

var _ = uuid.Nil
