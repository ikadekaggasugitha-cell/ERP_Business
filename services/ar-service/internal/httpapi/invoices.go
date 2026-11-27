package httpapi

import (
	"encoding/json"
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/ar-service/internal/ardb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "ar:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req createReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	invDate, err := platform.ParseDate(req.InvoiceDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invoiceDate harus YYYY-MM-DD"))
		return
	}
	dueDate, err := platform.ParseDate(req.DueDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("dueDate harus YYYY-MM-DD"))
		return
	}
	if dueDate.Time.Before(invDate.Time) {
		httpx.WriteError(w, r, httpx.BadRequest("dueDate tidak boleh sebelum invoiceDate"))
		return
	}
	customerID, err := httpx.ParseUUIDParam("customerId", req.CustomerID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.InvoiceNo) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("invoiceNo wajib diisi"))
		return
	}
	totalNum, lines, eventLines, err := buildAndValidateLines(scope[0], user, req.TotalAmount, req.Lines)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	taxNum, err := platform.ParseOptionalNumeric(req.TaxAmount)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("taxAmount tidak valid"))
		return
	}
	if taxNum.Valid && platform.NumericToRat(taxNum).Sign() < 0 {
		httpx.WriteError(w, r, httpx.BadRequest("taxAmount tidak boleh negatif"))
		return
	}
	var soRef *string
	if ref := strings.TrimSpace(req.SalesOrderRef); ref != "" {
		soRef = &ref
	}

	currency := strings.ToUpper(deflt(req.CurrencyCode, "IDR"))
	if len(currency) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("currencyCode harus 3 huruf ISO"))
		return
	}
	entry := ardb.CreateInvoiceParams{
		OrganizationID:    scope[0],
		InvoiceNo:         strings.TrimSpace(req.InvoiceNo),
		CustomerID:        customerID,
		InvoiceDate:       invDate,
		DueDate:           dueDate,
		CurrencyCode:      currency,
		TotalAmount:       totalNum,
		OutstandingAmount: totalNum,
		SalesOrderRef:     soRef,
		CreatedBy:         user,
	}
	payloadMap := map[string]any{
		"invoiceNo":    entry.InvoiceNo,
		"customerId":   customerID.String(),
		"invoiceDate":  platform.FormatDate(invDate),
		"dueDate":      platform.FormatDate(dueDate),
		"currencyCode": currency,
		"totalAmount":  platform.FormatNumeric(totalNum, 4),
		"status":       "Issued",
		"lines":        eventLines,
	}
	if strings.TrimSpace(req.TaxAmount) != "" {
		payloadMap["taxAmount"] = platform.FormatNumeric(taxNum, 4)
	}
	payload, _ := json.Marshal(payloadMap)
	ev := ardb.InsertOutboxParams{
		AggregateType:  "ARInvoice",
		EventType:      "ar.invoice.issued",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	created, err := s.repo.CreateInvoiceWithLinesAndEmit(r.Context(), scope, entry, lines, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "nomor dokumen sudah dipakai"))
		return
	}
	dto := invoiceDTO{
		ID:           created.ID.String(),
		InvoiceNo:    entry.InvoiceNo,
		CustomerID:   customerID.String(),
		InvoiceDate:  platform.FormatDate(created.InvoiceDate),
		DueDate:      platform.FormatDate(dueDate),
		CurrencyCode: currency,
		TotalAmount:  platform.FormatNumeric(totalNum, 4),
		Status:       created.Status,
		RowVersion:   created.RowVersion,
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, dto)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "ar:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	inv, lines, err := s.repo.GetWithLines(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	lineDTOs := make([]lineDTO, 0, len(lines))
	for _, l := range lines {
		lineDTOs = append(lineDTOs, lineDTO{
			ID:          l.ID.String(),
			LineNo:      l.LineNo,
			AccountID:   l.AccountID.String(),
			Amount:      platform.FormatNumeric(l.Amount, 4),
			Description: l.Description,
		})
	}
	httpx.WriteIDResponse(w, http.StatusOK, inv.RowVersion, invoiceDTO{
		ID:                inv.ID.String(),
		InvoiceNo:         inv.InvoiceNo,
		CustomerID:        inv.CustomerID.String(),
		InvoiceDate:       platform.FormatDate(inv.InvoiceDate),
		DueDate:           platform.FormatDate(inv.DueDate),
		CurrencyCode:      inv.CurrencyCode,
		TotalAmount:       platform.FormatNumeric(inv.TotalAmount, 4),
		OutstandingAmount: platform.FormatNumeric(inv.OutstandingAmount, 4),
		Status:            inv.Status,
		RowVersion:        inv.RowVersion,
		Lines:             lineDTOs,
	})
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "ar:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	rows, total, err := s.repo.ListWithCount(r.Context(), scope, statusFilter, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]invoiceDTO, 0, len(rows))
	for _, inv := range rows {
		out = append(out, invoiceDTO{
			ID:                inv.ID.String(),
			InvoiceNo:         inv.InvoiceNo,
			CustomerID:        inv.CustomerID.String(),
			InvoiceDate:       platform.FormatDate(inv.InvoiceDate),
			DueDate:           platform.FormatDate(inv.DueDate),
			CurrencyCode:      inv.CurrencyCode,
			TotalAmount:       platform.FormatNumeric(inv.TotalAmount, 4),
			OutstandingAmount: platform.FormatNumeric(inv.OutstandingAmount, 4),
			Status:            inv.Status,
			RowVersion:        inv.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) markPaid(w http.ResponseWriter, r *http.Request) {
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
	if err := requireARApprove(r); err != nil {
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
	inv, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if inv.CreatedBy != uuid.Nil && inv.CreatedBy == user && !httpx.HasRole(r.Context(), "SYSTEM_ADMIN") {
		httpx.WriteError(w, r, httpx.Forbidden("maker-checker separation of duties: creator cannot approve own invoice"))
		return
	}
	if inv.Status != "Issued" {
		httpx.WriteError(w, r, httpx.Conflict("INVALID_STATE", "hanya invoice berstatus Issued yang dapat ditandai lunas"))
		return
	}
	payPayload, _ := json.Marshal(map[string]any{
		"invoiceNo":    inv.InvoiceNo,
		"customerId":   inv.CustomerID.String(),
		"currencyCode": inv.CurrencyCode,
		"paidAmount":   platform.FormatNumeric(inv.OutstandingAmount, 4),
		"totalAmount":  platform.FormatNumeric(inv.TotalAmount, 4),
		"status":       "Paid",
	})
	ev := ardb.InsertOutboxParams{
		AggregateType:  "ARInvoice",
		AggregateID:    inv.ID,
		EventType:      "ar.invoice.paid",
		Payload:        payPayload,
		OrganizationID: inv.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	n, err := s.repo.MarkPaid(r.Context(), scope, ardb.MarkInvoicePaidParams{
		ID:          inv.ID,
		InvoiceDate: inv.InvoiceDate,
		RowVersion:  rowVer,
		UpdatedBy:   &user,
	}, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Issued"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, rowVer+1, invoiceDTO{
		ID:                inv.ID.String(),
		InvoiceNo:         inv.InvoiceNo,
		CustomerID:        inv.CustomerID.String(),
		InvoiceDate:       platform.FormatDate(inv.InvoiceDate),
		DueDate:           platform.FormatDate(inv.DueDate),
		CurrencyCode:      inv.CurrencyCode,
		TotalAmount:       platform.FormatNumeric(inv.TotalAmount, 4),
		Status:            "Paid",
		OutstandingAmount: "0.0000",
		RowVersion:        rowVer + 1,
	})
}

func (s *Server) deleteDraft(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "ar:write"); err != nil {
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
	inv, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	paidRat := new(big.Rat).Sub(platform.NumericToRat(inv.TotalAmount), platform.NumericToRat(inv.OutstandingAmount))
	if inv.Status != "Draft" || paidRat.Sign() > 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("INVOICE_NOT_DRAFT", "hanya invoice Draft tanpa pembayaran yang dapat dihapus"))
		return
	}
	n, err := s.repo.DeleteDraft(r.Context(), scope, ardb.DeleteDraftInvoiceParams{
		ID:          inv.ID,
		InvoiceDate: inv.InvoiceDate,
		RowVersion:  rowVer,
		UpdatedBy:   &user,
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

func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "ar:write"); err != nil {
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
	var req updateReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	existing, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.Status != "Draft" && existing.Status != "Issued" {
		httpx.WriteError(w, r, httpx.Unprocessable("INVOICE_NOT_EDITABLE", "hanya invoice Draft/Issued tanpa pembayaran yang dapat diubah"))
		return
	}
	outRat := platform.NumericToRat(existing.OutstandingAmount)
	totRat := platform.NumericToRat(existing.TotalAmount)
	if outRat != nil && totRat != nil && outRat.Cmp(totRat) != 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("INVOICE_HAS_PAYMENTS", "invoice yang sudah memiliki pembayaran tidak dapat diubah"))
		return
	}

	customerID := existing.CustomerID
	if strings.TrimSpace(req.CustomerID) != "" {
		customerID, err = httpx.ParseUUIDParam("customerId", req.CustomerID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
	}
	dueDate := existing.DueDate
	if strings.TrimSpace(req.DueDate) != "" {
		dueDate, err = platform.ParseDate(req.DueDate)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("dueDate harus YYYY-MM-DD"))
			return
		}
	}
	totalStr := strings.TrimSpace(req.TotalAmount)
	if totalStr == "" {
		totalStr = platform.FormatNumeric(existing.TotalAmount, 4)
	}
	totalNum, lines, _, err := buildAndValidateLines(existing.OrganizationID, user, totalStr, req.Lines)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var soRef *string
	if v := strings.TrimSpace(req.SalesOrderRef); v != "" {
		soRef = &v
	} else {
		soRef = existing.SalesOrderRef
	}

	nextVer, err := s.repo.UpdateDraftWithLines(r.Context(), scope, ardb.UpdateDraftInvoiceParams{
		ID:            existing.ID,
		InvoiceDate:   existing.InvoiceDate,
		RowVersion:    rowVer,
		CustomerID:    customerID,
		DueDate:       dueDate,
		CurrencyCode:  strings.ToUpper(deflt(req.CurrencyCode, existing.CurrencyCode)),
		TotalAmount:   totalNum,
		SalesOrderRef: soRef,
		UpdatedBy:     &user,
	}, lines)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, nextVer, map[string]any{
		"id":         existing.ID,
		"rowVersion": nextVer,
	})
}
