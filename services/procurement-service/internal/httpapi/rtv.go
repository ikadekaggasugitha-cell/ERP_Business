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

func (s *Server) returnToVendor(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if _, err := httpx.RequireUser(r); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	grID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		RtvNo  string `json:"rtvNo,omitempty"`
		Reason string `json:"reason"`
		Lines  []struct {
			ItemID    string `json:"itemId"`
			ReturnQty string `json:"returnQty"`
		} `json:"lines"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if len(req.Lines) == 0 {
		httpx.WriteError(w, r, httpx.BadRequest("lines wajib diisi minimal 1 item retur"))
		return
	}
	gr, err := s.repo.GetGoodsReceipt(r.Context(), scope, grID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("goods receipt tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}

	poLineToItem := make(map[uuid.UUID]uuid.UUID)
	if po, poErr := s.repo.GetByID(r.Context(), scope, gr.Header.PurchaseOrderID); poErr == nil {
		for _, pl := range po.Lines {
			poLineToItem[pl.ID] = pl.ItemID
		}
	}

	receivedByKey := make(map[uuid.UUID]*big.Rat, len(gr.Lines)*2)
	for _, l := range gr.Lines {
		q := platform.NumericToRat(l.ReceivedQty)
		if q == nil {
			q = new(big.Rat)
		}
		if cur, ok := receivedByKey[l.PoLineID]; ok {
			cur.Add(cur, q)
		} else {
			receivedByKey[l.PoLineID] = new(big.Rat).Set(q)
		}
		if itemID, ok := poLineToItem[l.PoLineID]; ok {
			if cur, exists := receivedByKey[itemID]; exists {
				cur.Add(cur, q)
			} else {
				receivedByKey[itemID] = new(big.Rat).Set(q)
			}
		}
	}

	totRetQty := new(big.Rat)
	outLines := make([]map[string]any, 0, len(req.Lines))
	for i, rl := range req.Lines {
		itemID, err := httpx.ParseUUIDParam(fmt.Sprintf("lines[%d].itemId", i), rl.ItemID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		recQty, ok := receivedByKey[itemID]
		if !ok {
			httpx.WriteError(w, r, httpx.Unprocessable("ITEM_NOT_IN_GR", "item retur tidak ditemukan pada goods receipt"))
			return
		}
		qNum, err := platform.ParseNumeric(rl.ReturnQty)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("lines[%d].returnQty tidak valid", i)))
			return
		}
		qRat := platform.NumericToRat(qNum)
		if qRat == nil || qRat.Sign() <= 0 || qRat.Cmp(recQty) > 0 {
			httpx.WriteError(w, r, httpx.Unprocessable("RETURN_QTY_EXCEEDED", "returnQty harus > 0 dan <= receivedQty"))
			return
		}
		totRetQty.Add(totRetQty, qRat)
		recNum, _ := platform.RatToNumeric(recQty, 4)
		outLines = append(outLines, map[string]any{
			"itemId":      itemID.String(),
			"receivedQty": platform.FormatNumeric(recNum, 4),
			"returnQty":   platform.FormatNumeric(qNum, 4),
		})
	}

	rtvNo := strings.TrimSpace(req.RtvNo)
	if rtvNo == "" {
		rtvNo = "RTV-" + gr.Header.GrNo
	}
	totNum, _ := platform.RatToNumeric(totRetQty, 4)
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, gr.Header.RowVersion, map[string]any{
		"goodsReceiptId": gr.Header.ID.String(),
		"grNo":           gr.Header.GrNo,
		"rtvNo":          rtvNo,
		"reason":         strings.TrimSpace(req.Reason),
		"status":         "Returned",
		"totalReturnQty": platform.FormatNumeric(totNum, 4),
		"lines":          outLines,
	})
}
