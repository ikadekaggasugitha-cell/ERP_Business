package httpapi

import (
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/sales-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type salesQuotationReq struct {
	QuotationNo   string      `json:"quotationNo"`
	CustomerID    string      `json:"customerId"`
	QuotationDate string      `json:"quotationDate,omitempty"`
	ValidUntil    string      `json:"validUntil,omitempty"`
	CurrencyCode  string      `json:"currencyCode,omitempty"`
	Lines         []soLineReq `json:"lines"`
}

type salesQuotationDTO struct {
	ID             string      `json:"id"`
	OrganizationID string      `json:"organizationId"`
	QuotationNo    string      `json:"quotationNo"`
	CustomerID     string      `json:"customerId"`
	QuotationDate  string      `json:"quotationDate"`
	ValidUntil     string      `json:"validUntil,omitempty"`
	CurrencyCode   string      `json:"currencyCode"`
	TotalAmount    string      `json:"totalAmount"`
	Status         string      `json:"status"`
	SalesOrderID   string      `json:"salesOrderId,omitempty"`
	Lines          []soLineReq `json:"lines"`
	RowVersion     int32       `json:"rowVersion"`
}

func (h *Handler) handleListQuotations(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "sales:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, _, _ := httpx.PaginateParams(r)
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))

	h.qMu.RLock()
	items := make([]salesQuotationDTO, 0)
	for _, q := range h.quotations {
		orgID, err := uuid.Parse(q.OrganizationID)
		if err != nil || !slices.Contains(scope, orgID) {
			continue
		}
		if statusFilter != "" && !strings.EqualFold(q.Status, statusFilter) {
			continue
		}
		items = append(items, q)
	}
	h.qMu.RUnlock()

	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: items,
		Page: httpx.NewPageMeta(int(page), int(size), len(items)),
	})
}

func (h *Handler) handleCreateQuotation(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if _, err := httpx.RequireUser(r); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "sales:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req salesQuotationReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, h.idem, scope, &req)
	if done {
		return
	}
	defer release()

	qNo := strings.TrimSpace(req.QuotationNo)
	if qNo == "" || len(req.Lines) == 0 {
		httpx.WriteError(w, r, httpx.BadRequest("quotationNo dan minimal 1 line wajib diisi"))
		return
	}
	custID, err := httpx.ParseUUIDParam("customerId", req.CustomerID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	qDate := time.Now().UTC().Format("2006-01-02")
	if strings.TrimSpace(req.QuotationDate) != "" {
		d, err := platform.ParseDate(req.QuotationDate)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("quotationDate harus YYYY-MM-DD"))
			return
		}
		qDate = platform.FormatDate(d)
	}
	validUntil := ""
	if strings.TrimSpace(req.ValidUntil) != "" {
		d, err := platform.ParseDate(req.ValidUntil)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("validUntil harus YYYY-MM-DD"))
			return
		}
		validUntil = platform.FormatDate(d)
	}
	ccy := strings.ToUpper(strings.TrimSpace(req.CurrencyCode))
	if ccy == "" {
		ccy = "IDR"
	}
	if len(ccy) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("currencyCode harus 3 karakter ISO-4217"))
		return
	}

	totRat := new(big.Rat)
	for i, l := range req.Lines {
		if _, err := httpx.ParseUUIDParam(fmt.Sprintf("lines[%d].itemId", i), l.ItemID); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		qNum, err := platform.ParseNumeric(l.OrderQty)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("lines[%d].orderQty tidak valid", i)))
			return
		}
		pNum, err := platform.ParseNumeric(l.UnitPrice)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("lines[%d].unitPrice tidak valid", i)))
			return
		}
		qRat := platform.NumericToRat(qNum)
		pRat := platform.NumericToRat(pNum)
		if qRat == nil || qRat.Sign() <= 0 || pRat == nil || pRat.Sign() < 0 {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("lines[%d] qty harus > 0 dan unitPrice >= 0", i)))
			return
		}
		totRat.Add(totRat, new(big.Rat).Mul(qRat, pRat))
	}
	totNum, _ := platform.RatToNumeric(totRat, 4)

	id := uuid.New()
	dto := salesQuotationDTO{
		ID:             id.String(),
		OrganizationID: scope[0].String(),
		QuotationNo:    qNo,
		CustomerID:     custID.String(),
		QuotationDate:  qDate,
		ValidUntil:     validUntil,
		CurrencyCode:   ccy,
		TotalAmount:    platform.FormatNumeric(totNum, 4),
		Status:         "Open",
		Lines:          req.Lines,
		RowVersion:     1,
	}
	h.qMu.Lock()
	h.quotations[id] = dto
	h.qMu.Unlock()

	httpx.SaveIDResponse(w, h.idem, scope, idemKey, reqHash, http.StatusCreated, dto.RowVersion, dto)
}

func (h *Handler) handleConvertQuotationToOrder(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "sales:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	qID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		SoNo      string `json:"soNo,omitempty"`
		OrderDate string `json:"orderDate,omitempty"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, h.idem, scope, &req)
	if done {
		return
	}
	defer release()

	h.qMu.RLock()
	q, ok := h.quotations[qID]
	h.qMu.RUnlock()
	if !ok {
		httpx.WriteError(w, r, httpx.NotFound("sales quotation tidak ditemukan"))
		return
	}
	qOrg, _ := uuid.Parse(q.OrganizationID)
	if !slices.Contains(scope, qOrg) {
		httpx.WriteError(w, r, httpx.NotFound("sales quotation tidak ditemukan"))
		return
	}
	if q.Status == "Converted" {
		httpx.WriteError(w, r, httpx.Conflict("ALREADY_CONVERTED", "sales quotation sudah dikonversi menjadi sales order"))
		return
	}

	custID, _ := uuid.Parse(q.CustomerID)
	soNo := strings.TrimSpace(req.SoNo)
	if soNo == "" {
		soNo = "SO-" + q.QuotationNo
	}
	orderDate := pgtype.Date{Time: time.Now().UTC(), Valid: true}
	if strings.TrimSpace(req.OrderDate) != "" {
		if orderDate, err = platform.ParseDate(req.OrderDate); err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("orderDate harus YYYY-MM-DD"))
			return
		}
	}
	totNum, _ := platform.ParseNumeric(q.TotalAmount)
	soLines := make([]repository.CreateSOLineInput, 0, len(q.Lines))
	for i, l := range q.Lines {
		itemID, _ := uuid.Parse(l.ItemID)
		qNum, _ := platform.ParseNumeric(l.OrderQty)
		pNum, _ := platform.ParseNumeric(l.UnitPrice)
		lineNo := l.LineNo
		if lineNo <= 0 {
			lineNo = int32(i + 1)
		}
		soLines = append(soLines, repository.CreateSOLineInput{
			LineNo:    lineNo,
			ItemID:    itemID,
			OrderQty:  qNum,
			UnitPrice: pNum,
		})
	}

	soDetail, err := h.store.CreateWithLines(r.Context(), scope, repository.CreateSOInput{
		OrganizationID: qOrg,
		SoNo:           soNo,
		CustomerID:     custID,
		OrderDate:      orderDate,
		CurrencyCode:   q.CurrencyCode,
		TotalAmount:    totNum,
		Lines:          soLines,
		CorrelationID:  httpx.CorrelationUUID(r),
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	h.qMu.Lock()
	q.Status = "Converted"
	q.SalesOrderID = soDetail.Header.ID.String()
	q.RowVersion++
	h.quotations[qID] = q
	h.qMu.Unlock()

	httpx.SaveIDResponse(w, h.idem, scope, idemKey, reqHash, http.StatusCreated, soDetail.Header.RowVersion, map[string]any{
		"quotation":  q,
		"salesOrder": toSalesOrderResp(soDetail),
	})
}

func (h *Handler) handleReturnSO(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if _, err := httpx.RequireUser(r); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "sales:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	soID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		ReturnNo string `json:"returnNo,omitempty"`
		Reason   string `json:"reason"`
		Lines    []struct {
			ItemID    string `json:"itemId"`
			ReturnQty string `json:"returnQty"`
		} `json:"lines"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, h.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if len(req.Lines) == 0 {
		httpx.WriteError(w, r, httpx.BadRequest("lines wajib diisi minimal 1 item retur"))
		return
	}
	detail, err := h.store.GetByID(r.Context(), scope, soID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if detail.Header.Status != "Confirmed" && detail.Header.Status != "Delivered" {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_SO_STATUS", "retur hanya diperbolehkan untuk sales order Confirmed atau Delivered"))
		return
	}

	type lineInfo struct {
		maxQty *big.Rat
		price  *big.Rat
	}
	byItem := make(map[uuid.UUID]lineInfo, len(detail.Lines))
	for _, l := range detail.Lines {
		maxQ := platform.NumericToRat(l.OrderQty)
		if dq := platform.NumericToRat(l.DeliveredQty); dq != nil && dq.Sign() > 0 {
			maxQ = dq
		}
		byItem[l.ItemID] = lineInfo{maxQty: maxQ, price: platform.NumericToRat(l.UnitPrice)}
	}

	creditTotal := new(big.Rat)
	retLines := make([]map[string]any, 0, len(req.Lines))
	for i, rl := range req.Lines {
		itemID, err := httpx.ParseUUIDParam(fmt.Sprintf("lines[%d].itemId", i), rl.ItemID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		info, ok := byItem[itemID]
		if !ok {
			httpx.WriteError(w, r, httpx.Unprocessable("ITEM_NOT_IN_ORDER", "item retur tidak ditemukan pada sales order"))
			return
		}
		qNum, err := platform.ParseNumeric(rl.ReturnQty)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("lines[%d].returnQty tidak valid", i)))
			return
		}
		qRat := platform.NumericToRat(qNum)
		if qRat == nil || qRat.Sign() <= 0 || qRat.Cmp(info.maxQty) > 0 {
			httpx.WriteError(w, r, httpx.Unprocessable("RETURN_QTY_EXCEEDED", "returnQty harus > 0 dan <= qty sales order"))
			return
		}
		lineAmt := new(big.Rat).Mul(qRat, info.price)
		creditTotal.Add(creditTotal, lineAmt)
		amtNum, _ := platform.RatToNumeric(lineAmt, 4)
		retLines = append(retLines, map[string]any{
			"itemId":     itemID.String(),
			"returnQty":  platform.FormatNumeric(qNum, 4),
			"lineCredit": platform.FormatNumeric(amtNum, 4),
		})
	}

	retNo := strings.TrimSpace(req.ReturnNo)
	if retNo == "" {
		retNo = "SR-" + detail.Header.SoNo
	}
	totCreditNum, _ := platform.RatToNumeric(creditTotal, 4)
	httpx.SaveIDResponse(w, h.idem, scope, idemKey, reqHash, http.StatusCreated, detail.Header.RowVersion, map[string]any{
		"salesOrderId":     detail.Header.ID.String(),
		"soNo":             detail.Header.SoNo,
		"returnNo":         retNo,
		"reason":           strings.TrimSpace(req.Reason),
		"currencyCode":     detail.Header.CurrencyCode,
		"creditMemoAmount": platform.FormatNumeric(totCreditNum, 4),
		"creditMemoStatus": "Pending",
		"lines":            retLines,
	})
}
