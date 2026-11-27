package httpapi

import (
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type threeWayGRLineReq struct {
	ItemID      string `json:"itemId"`
	ReceivedQty string `json:"receivedQty"`
}

type threeWayInvoiceLineReq struct {
	ItemID      string `json:"itemId"`
	InvoicedQty string `json:"invoicedQty"`
	UnitPrice   string `json:"unitPrice"`
}

type threeWayMatchReq struct {
	InvoiceNo         string                   `json:"invoiceNo"`
	QtyTolerancePct   string                   `json:"qtyTolerancePct,omitempty"`
	PriceTolerancePct string                   `json:"priceTolerancePct,omitempty"`
	GRLines           []threeWayGRLineReq      `json:"grLines,omitempty"`
	InvoiceLines      []threeWayInvoiceLineReq `json:"invoiceLines"`
}

type threeWayLineResult struct {
	LineNo           int32  `json:"lineNo"`
	ItemID           string `json:"itemId"`
	POQty            string `json:"poQty"`
	ReceivedQty      string `json:"receivedQty"`
	InvoicedQty      string `json:"invoicedQty"`
	POUnitPrice      string `json:"poUnitPrice"`
	InvoiceUnitPrice string `json:"invoiceUnitPrice"`
	QtyVariance      string `json:"qtyVariance"`
	QtyVariancePct   string `json:"qtyVariancePct"`
	PriceVariance    string `json:"priceVariance"`
	PriceVariancePct string `json:"priceVariancePct"`
	AmountVariance   string `json:"amountVariance"`
	Matched          bool   `json:"matched"`
	Reason           string `json:"reason,omitempty"`
}

type threeWayMatchResp struct {
	PurchaseOrderID   string               `json:"purchaseOrderId"`
	PoNo              string               `json:"poNo"`
	InvoiceNo         string               `json:"invoiceNo,omitempty"`
	Status            string               `json:"status"` // Matched | WithinTolerance | VarianceExceeded
	Matched           bool                 `json:"matched"`
	QtyTolerancePct   string               `json:"qtyTolerancePct"`
	PriceTolerancePct string               `json:"priceTolerancePct"`
	POTotal           string               `json:"poTotal"`
	InvoiceTotal      string               `json:"invoiceTotal"`
	TotalVariance     string               `json:"totalVariance"`
	Lines             []threeWayLineResult `json:"lines"`
}

func (s *Server) threeWayMatchPO(w http.ResponseWriter, r *http.Request) {
	if !httpx.HasPermission(r.Context(), "procurement:read") && !httpx.HasPermission(r.Context(), "procurement:write") {
		httpx.WriteError(w, r, httpx.Forbidden("izin procurement:read dibutuhkan"))
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
	var req threeWayMatchReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if len(req.InvoiceLines) == 0 {
		httpx.WriteError(w, r, httpx.BadRequest("invoiceLines wajib diisi minimal 1 baris"))
		return
	}

	qtyTolRat := new(big.Rat)
	if strings.TrimSpace(req.QtyTolerancePct) != "" {
		n, err := platform.ParseNumeric(req.QtyTolerancePct)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("qtyTolerancePct tidak valid"))
			return
		}
		qtyTolRat = platform.NumericToRat(n)
		if qtyTolRat == nil || qtyTolRat.Sign() < 0 {
			httpx.WriteError(w, r, httpx.BadRequest("qtyTolerancePct harus >= 0"))
			return
		}
	}
	priceTolRat := new(big.Rat)
	if strings.TrimSpace(req.PriceTolerancePct) != "" {
		n, err := platform.ParseNumeric(req.PriceTolerancePct)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("priceTolerancePct tidak valid"))
			return
		}
		priceTolRat = platform.NumericToRat(n)
		if priceTolRat == nil || priceTolRat.Sign() < 0 {
			httpx.WriteError(w, r, httpx.BadRequest("priceTolerancePct harus >= 0"))
			return
		}
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

	grByItem := make(map[uuid.UUID]*big.Rat, len(req.GRLines))
	for i, gl := range req.GRLines {
		itemID, err := httpx.ParseUUIDParam(fmt.Sprintf("grLines[%d].itemId", i), gl.ItemID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		qNum, err := platform.ParseNumeric(gl.ReceivedQty)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("grLines[%d].receivedQty tidak valid", i)))
			return
		}
		qRat := platform.NumericToRat(qNum)
		if qRat == nil || qRat.Sign() < 0 {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("grLines[%d].receivedQty harus >= 0", i)))
			return
		}
		if cur, ok := grByItem[itemID]; ok {
			cur.Add(cur, qRat)
		} else {
			grByItem[itemID] = new(big.Rat).Set(qRat)
		}
	}

	type invLineAgg struct {
		qty   *big.Rat
		price *big.Rat
	}
	invByItem := make(map[uuid.UUID]invLineAgg, len(req.InvoiceLines))
	for i, il := range req.InvoiceLines {
		itemID, err := httpx.ParseUUIDParam(fmt.Sprintf("invoiceLines[%d].itemId", i), il.ItemID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		qNum, err := platform.ParseNumeric(il.InvoicedQty)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("invoiceLines[%d].invoicedQty tidak valid", i)))
			return
		}
		pNum, err := platform.ParseNumeric(il.UnitPrice)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("invoiceLines[%d].unitPrice tidak valid", i)))
			return
		}
		qRat := platform.NumericToRat(qNum)
		pRat := platform.NumericToRat(pNum)
		if qRat == nil || qRat.Sign() < 0 || pRat == nil || pRat.Sign() < 0 {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("invoiceLines[%d] qty/price harus >= 0", i)))
			return
		}
		invByItem[itemID] = invLineAgg{qty: qRat, price: pRat}
	}

	hundred := big.NewRat(100, 1)
	fmtRat := func(v *big.Rat, scale int) string {
		n, _ := platform.RatToNumeric(v, scale)
		return platform.FormatNumeric(n, scale)
	}

	poTotal := new(big.Rat)
	invTotal := new(big.Rat)
	results := make([]threeWayLineResult, 0, len(detail.Lines))
	allMatched := true
	hasToleranceDiff := false

	for _, pl := range detail.Lines {
		poQty := platform.NumericToRat(pl.OrderQty)
		poPrice := platform.NumericToRat(pl.UnitPrice)
		if poQty == nil {
			poQty = new(big.Rat)
		}
		if poPrice == nil {
			poPrice = new(big.Rat)
		}
		poLineAmt := new(big.Rat).Mul(poQty, poPrice)
		poTotal.Add(poTotal, poLineAmt)

		recQty := new(big.Rat)
		if len(req.GRLines) > 0 {
			if rVal, ok := grByItem[pl.ItemID]; ok {
				recQty = rVal
			}
		} else if detail.Header.Status == "Received" {
			recQty = new(big.Rat).Set(poQty)
		}

		invQty := new(big.Rat)
		invPrice := new(big.Rat)
		if il, ok := invByItem[pl.ItemID]; ok {
			invQty = il.qty
			invPrice = il.price
		}
		invLineAmt := new(big.Rat).Mul(invQty, invPrice)
		invTotal.Add(invTotal, invLineAmt)

		// Compare invoiced qty against received qty (or PO qty if GR > PO)
		qtyVar := new(big.Rat).Sub(invQty, recQty)
		qtyVarPct := new(big.Rat)
		if recQty.Sign() > 0 {
			qtyVarPct = new(big.Rat).Mul(new(big.Rat).Quo(new(big.Rat).Abs(qtyVar), recQty), hundred)
		} else if qtyVar.Sign() != 0 {
			qtyVarPct = new(big.Rat).Set(hundred)
		}

		priceVar := new(big.Rat).Sub(invPrice, poPrice)
		priceVarPct := new(big.Rat)
		if poPrice.Sign() > 0 {
			priceVarPct = new(big.Rat).Mul(new(big.Rat).Quo(new(big.Rat).Abs(priceVar), poPrice), hundred)
		} else if priceVar.Sign() != 0 {
			priceVarPct = new(big.Rat).Set(hundred)
		}

		amtVar := new(big.Rat).Sub(invLineAmt, poLineAmt)
		lineMatched := true
		var reasons []string
		if qtyVarPct.Cmp(qtyTolRat) > 0 {
			lineMatched = false
			reasons = append(reasons, fmt.Sprintf("qty variance %s%% exceeds tolerance %s%%", fmtRat(qtyVarPct, 2), fmtRat(qtyTolRat, 2)))
		}
		if priceVarPct.Cmp(priceTolRat) > 0 {
			lineMatched = false
			reasons = append(reasons, fmt.Sprintf("price variance %s%% exceeds tolerance %s%%", fmtRat(priceVarPct, 2), fmtRat(priceTolRat, 2)))
		}
		if !lineMatched {
			allMatched = false
		} else if qtyVar.Sign() != 0 || priceVar.Sign() != 0 {
			hasToleranceDiff = true
		}

		results = append(results, threeWayLineResult{
			LineNo:           pl.LineNo,
			ItemID:           pl.ItemID.String(),
			POQty:            fmtRat(poQty, 4),
			ReceivedQty:      fmtRat(recQty, 4),
			InvoicedQty:      fmtRat(invQty, 4),
			POUnitPrice:      fmtRat(poPrice, 4),
			InvoiceUnitPrice: fmtRat(invPrice, 4),
			QtyVariance:      fmtRat(qtyVar, 4),
			QtyVariancePct:   fmtRat(qtyVarPct, 2),
			PriceVariance:    fmtRat(priceVar, 4),
			PriceVariancePct: fmtRat(priceVarPct, 2),
			AmountVariance:   fmtRat(amtVar, 4),
			Matched:          lineMatched,
			Reason:           strings.Join(reasons, "; "),
		})
	}

	status := "Matched"
	if !allMatched {
		status = "VarianceExceeded"
	} else if hasToleranceDiff {
		status = "WithinTolerance"
	}
	totVar := new(big.Rat).Sub(invTotal, poTotal)

	httpx.WriteJSON(w, http.StatusOK, threeWayMatchResp{
		PurchaseOrderID:   detail.Header.ID.String(),
		PoNo:              detail.Header.PoNo,
		InvoiceNo:         strings.TrimSpace(req.InvoiceNo),
		Status:            status,
		Matched:           allMatched,
		QtyTolerancePct:   fmtRat(qtyTolRat, 2),
		PriceTolerancePct: fmtRat(priceTolRat, 2),
		POTotal:           fmtRat(poTotal, 4),
		InvoiceTotal:      fmtRat(invTotal, 4),
		TotalVariance:     fmtRat(totVar, 4),
		Lines:             results,
	})
}
