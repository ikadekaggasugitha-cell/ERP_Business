// Package httpapi implements ar-service REST handlers with strict decimal checks,
// line-sum validation, idempotency hash protection, RBAC, and real pagination.
package httpapi

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/ar-service/internal/ardb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Store abstracts AR repository calls for handler unit testing.
type Store interface {
	CreateInvoiceWithLinesAndEmit(ctx context.Context, scope []uuid.UUID, invoice ardb.CreateInvoiceParams, lines []ardb.CreateInvoiceLineParams, ev ardb.InsertOutboxParams) (ardb.CreateInvoiceRow, error)
	GetWithLines(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (ardb.ArArInvoice, []ardb.ListInvoiceLinesRow, error)
	GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (ardb.ArArInvoice, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, status string, limit, offset int32) ([]ardb.ListInvoicesRow, int64, error)
	UpdateDraftWithLines(ctx context.Context, scope []uuid.UUID, hdr ardb.UpdateDraftInvoiceParams, lines []ardb.CreateInvoiceLineParams) (int32, error)
	MarkPaid(ctx context.Context, scope []uuid.UUID, arg ardb.MarkInvoicePaidParams, ev ardb.InsertOutboxParams) (int64, error)
	DeleteDraft(ctx context.Context, scope []uuid.UUID, arg ardb.DeleteDraftInvoiceParams) (int64, error)
	ApplyCreditNoteAndEmit(ctx context.Context, scope []uuid.UUID, arg ardb.ApplyReceiptToInvoiceParams, ev ardb.InsertOutboxParams) (ardb.ApplyReceiptToInvoiceRow, error)
	WriteOffAndEmit(ctx context.Context, scope []uuid.UUID, arg ardb.WriteOffInvoiceParams, ev ardb.InsertOutboxParams) (int64, error)
}

// Server serves AR HTTP endpoints.
type Server struct {
	repo Store
	idem *httpx.IdemStore
}

// NewServer constructs an AR HTTP Server.
func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

// Register mounts AR routes onto mux.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /customer-invoices", s.create)
	mux.HandleFunc("GET /customer-invoices", s.list)
	mux.HandleFunc("GET /customer-invoices/aging", s.aging)
	mux.HandleFunc("GET /customer-invoices/{id}", s.get)
	mux.HandleFunc("PUT /customer-invoices/{id}", s.update)
	mux.HandleFunc("POST /customer-invoices/{id}/pay", s.markPaid)
	mux.HandleFunc("POST /customer-invoices/{id}/credit-note", s.creditNote)
	mux.HandleFunc("POST /customer-invoices/{id}/write-off", s.writeOff)
	mux.HandleFunc("DELETE /customer-invoices/{id}", s.deleteDraft)
}

type lineReq struct {
	LineNo      int32  `json:"lineNo"`
	AccountID   string `json:"accountId"`
	ItemID      string `json:"itemId,omitempty"`
	Quantity    string `json:"quantity,omitempty"`
	UnitPrice   string `json:"unitPrice,omitempty"`
	Amount      string `json:"amount"`
	Description string `json:"description"`
}

type createReq struct {
	InvoiceNo     string    `json:"invoiceNo"`
	CustomerID    string    `json:"customerId"`
	InvoiceDate   string    `json:"invoiceDate"`
	DueDate       string    `json:"dueDate"`
	CurrencyCode  string    `json:"currencyCode"`
	TotalAmount   string    `json:"totalAmount"`
	TaxAmount     string    `json:"taxAmount,omitempty"`
	SalesOrderRef string    `json:"salesOrderRef,omitempty"`
	Lines         []lineReq `json:"lines"`
}

type lineDTO struct {
	ID          string  `json:"id"`
	LineNo      int32   `json:"lineNo"`
	AccountID   string  `json:"accountId"`
	Amount      string  `json:"amount"`
	Description *string `json:"description,omitempty"`
}

type invoiceDTO struct {
	ID                string    `json:"id"`
	InvoiceNo         string    `json:"invoiceNo,omitempty"`
	CustomerID        string    `json:"customerId,omitempty"`
	InvoiceDate       string    `json:"invoiceDate"`
	DueDate           string    `json:"dueDate,omitempty"`
	CurrencyCode      string    `json:"currencyCode,omitempty"`
	TotalAmount       string    `json:"totalAmount,omitempty"`
	OutstandingAmount string    `json:"outstandingAmount,omitempty"`
	Status            string    `json:"status"`
	RowVersion        int32     `json:"rowVersion"`
	Lines             []lineDTO `json:"lines,omitempty"`
}

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

func requireARApprove(r *http.Request) error {
	if err := httpx.RequirePermission(r, "ar.approve"); err != nil {
		if httpx.RequirePermission(r, "ar:approve") == nil {
			return nil
		}
		return err
	}
	return nil
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

func buildAndValidateLines(orgID, user uuid.UUID, totalRaw string, reqLines []lineReq) (pgtype.Numeric, []ardb.CreateInvoiceLineParams, []map[string]any, error) {
	totalNum, err := platform.ParseNumeric(totalRaw)
	if err != nil {
		return pgtype.Numeric{}, nil, nil, httpx.BadRequest("totalAmount tidak valid", httpx.Detail{Field: "totalAmount", Issue: err.Error()})
	}
	totalRat := platform.NumericToRat(totalNum)
	if totalRat.Sign() <= 0 {
		return pgtype.Numeric{}, nil, nil, httpx.BadRequest("totalAmount harus > 0")
	}

	lineSum := new(big.Rat)
	lines := make([]ardb.CreateInvoiceLineParams, 0, len(reqLines))
	eventLines := make([]map[string]any, 0, len(reqLines))
	for _, l := range reqLines {
		acct, err := httpx.ParseUUIDParam("lines[].accountId", l.AccountID)
		if err != nil {
			return pgtype.Numeric{}, nil, nil, err
		}
		amtNum, err := platform.ParseNumeric(l.Amount)
		if err != nil {
			return pgtype.Numeric{}, nil, nil, httpx.BadRequest("lines[].amount tidak valid")
		}
		amtRat := platform.NumericToRat(amtNum)
		if amtRat.Sign() < 0 {
			return pgtype.Numeric{}, nil, nil, httpx.BadRequest("lines[].amount tidak boleh negatif")
		}
		lineSum.Add(lineSum, amtRat)

		var itemPtr *uuid.UUID
		if strings.TrimSpace(l.ItemID) != "" {
			u, err := httpx.ParseUUIDParam("lines[].itemId", l.ItemID)
			if err != nil {
				return pgtype.Numeric{}, nil, nil, err
			}
			itemPtr = &u
		}
		var qty, unitPrice pgtype.Numeric
		if strings.TrimSpace(l.Quantity) != "" {
			if qty, err = platform.ParseNumeric(l.Quantity); err != nil {
				return pgtype.Numeric{}, nil, nil, httpx.BadRequest("lines[].quantity tidak valid")
			}
		}
		if strings.TrimSpace(l.UnitPrice) != "" {
			if unitPrice, err = platform.ParseNumeric(l.UnitPrice); err != nil {
				return pgtype.Numeric{}, nil, nil, httpx.BadRequest("lines[].unitPrice tidak valid")
			}
		}
		var desc *string
		if d := strings.TrimSpace(l.Description); d != "" {
			desc = &d
		}
		lines = append(lines, ardb.CreateInvoiceLineParams{
			OrganizationID: orgID,
			LineNo:         l.LineNo,
			AccountID:      acct,
			ItemID:         itemPtr,
			Quantity:       qty,
			UnitPrice:      unitPrice,
			Amount:         amtNum,
			Description:    desc,
			CreatedBy:      user,
		})
		eventLines = append(eventLines, map[string]any{
			"lineNo":    l.LineNo,
			"accountId": acct.String(),
			"amount":    platform.FormatNumeric(amtNum, 4),
		})
	}
	if len(lines) > 0 && lineSum.Cmp(totalRat) != 0 {
		return pgtype.Numeric{}, nil, nil, httpx.Unprocessable("AMOUNT_MISMATCH", "jumlah lines[].amount harus sama dengan totalAmount")
	}
	return totalNum, lines, eventLines, nil
}

func deflt(v, d string) string {
	if s := strings.TrimSpace(v); s != "" {
		return s
	}
	return d
}

// aging computes AR aging buckets (current, 1-30, 31-60, 61-90, 90+, totalOutstanding) via *big.Rat.
func (s *Server) aging(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "ar:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	asOf := time.Now().UTC().Truncate(24 * time.Hour)
	if raw := strings.TrimSpace(r.URL.Query().Get("asOfDate")); raw != "" {
		d, perr := time.Parse("2006-01-02", raw)
		if perr != nil {
			httpx.WriteError(w, r, httpx.BadRequest("asOfDate harus YYYY-MM-DD"))
			return
		}
		asOf = d.UTC()
	}

	rows, _, err := s.repo.ListWithCount(r.Context(), scope, "", 10000, 0)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}

	current := new(big.Rat)
	d1To30 := new(big.Rat)
	d31To60 := new(big.Rat)
	d61To90 := new(big.Rat)
	d90Plus := new(big.Rat)
	total := new(big.Rat)
	openCount := 0

	for _, row := range rows {
		if row.Status == "Draft" || row.Status == "Paid" || row.Status == "Cancelled" {
			continue
		}
		outRat := platform.NumericToRat(row.OutstandingAmount)
		if outRat == nil || outRat.Sign() <= 0 {
			continue
		}
		openCount++
		total.Add(total, outRat)
		due := row.DueDate.Time.UTC().Truncate(24 * time.Hour)
		days := int(asOf.Sub(due).Hours() / 24)
		switch {
		case days <= 0:
			current.Add(current, outRat)
		case days <= 30:
			d1To30.Add(d1To30, outRat)
		case days <= 60:
			d31To60.Add(d31To60, outRat)
		case days <= 90:
			d61To90.Add(d61To90, outRat)
		default:
			d90Plus.Add(d90Plus, outRat)
		}
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"asOfDate":         asOf.Format("2006-01-02"),
		"invoiceCount":     openCount,
		"current":          current.FloatString(4),
		"days1To30":        d1To30.FloatString(4),
		"days31To60":       d31To60.FloatString(4),
		"days61To90":       d61To90.FloatString(4),
		"days90Plus":       d90Plus.FloatString(4),
		"totalOutstanding": total.FloatString(4),
	})
}

type updateReq struct {
	CustomerID    string    `json:"customerId"`
	DueDate       string    `json:"dueDate"`
	CurrencyCode  string    `json:"currencyCode"`
	TotalAmount   string    `json:"totalAmount"`
	SalesOrderRef string    `json:"salesOrderRef,omitempty"`
	Lines         []lineReq `json:"lines"`
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

type creditNoteReq struct {
	CreditNoteNo string `json:"creditNoteNo,omitempty"`
	Amount       string `json:"amount"`
	Reason       string `json:"reason"`
}

func (s *Server) creditNote(w http.ResponseWriter, r *http.Request) {
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

	var req creditNoteReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	amtNum, err := platform.ParseNumeric(req.Amount)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("amount tidak valid"))
		return
	}
	amtRat := platform.NumericToRat(amtNum)
	if amtRat == nil || amtRat.Sign() <= 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_AMOUNT", "credit note amount harus > 0"))
		return
	}

	inv, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if inv.Status != "Issued" && inv.Status != "PartiallyPaid" && inv.Status != "Overdue" {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_STATUS", "credit note hanya berlaku untuk invoice Issued, PartiallyPaid, atau Overdue"))
		return
	}
	outRat := platform.NumericToRat(inv.OutstandingAmount)
	if outRat == nil || amtRat.Cmp(outRat) > 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("CREDIT_NOTE_EXCEEDS_OUTSTANDING", "jumlah credit note melebihi sisa tagihan (outstanding)"))
		return
	}

	remaining := new(big.Rat).Sub(outRat, amtRat)
	cnNo := strings.TrimSpace(req.CreditNoteNo)
	if cnNo == "" {
		cnNo = "CN-" + inv.InvoiceNo
	}
	payload, _ := json.Marshal(map[string]any{
		"invoiceId":      inv.ID,
		"invoiceNo":      inv.InvoiceNo,
		"creditNoteNo":   cnNo,
		"customerId":     inv.CustomerID,
		"currencyCode":   inv.CurrencyCode,
		"creditAmount":   amtRat.FloatString(4),
		"remainingAfter": remaining.FloatString(4),
		"reason":         strings.TrimSpace(req.Reason),
		"actorId":        user,
	})

	updated, err := s.repo.ApplyCreditNoteAndEmit(r.Context(), scope, ardb.ApplyReceiptToInvoiceParams{
		ID:          inv.ID,
		InvoiceDate: inv.InvoiceDate,
		UpdatedBy:   &user,
		PaidAmount:  amtNum,
	}, ardb.InsertOutboxParams{
		AggregateType:  "CustomerInvoice",
		AggregateID:    inv.ID,
		EventType:      "ar.credit_note.issued",
		Payload:        payload,
		OrganizationID: inv.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, inv.RowVersion+1, map[string]any{
		"invoiceId":         inv.ID,
		"creditNoteNo":      cnNo,
		"creditAmount":      amtRat.FloatString(4),
		"outstandingAmount": platform.FormatNumeric(updated.OutstandingAmount, 4),
		"status":            updated.Status,
	})
}

type writeOffReq struct {
	Reason string `json:"reason"`
}

func (s *Server) writeOff(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := requireARApprove(r); err != nil {
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

	var req writeOffReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if strings.TrimSpace(req.Reason) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("reason wajib diisi untuk write-off piutang"))
		return
	}
	inv, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if inv.Status != "Issued" && inv.Status != "PartiallyPaid" && inv.Status != "Overdue" {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_STATUS", "write-off hanya berlaku untuk invoice Issued, PartiallyPaid, atau Overdue"))
		return
	}
	outRat := platform.NumericToRat(inv.OutstandingAmount)
	if outRat == nil || outRat.Sign() <= 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("NOTHING_TO_WRITEOFF", "invoice tidak memiliki sisa piutang untuk di-write-off"))
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"invoiceId":      inv.ID,
		"invoiceNo":      inv.InvoiceNo,
		"customerId":     inv.CustomerID,
		"currencyCode":   inv.CurrencyCode,
		"writeOffAmount": outRat.FloatString(4),
		"reason":         strings.TrimSpace(req.Reason),
		"actorId":        user,
	})
	n, err := s.repo.WriteOffAndEmit(r.Context(), scope, ardb.WriteOffInvoiceParams{
		ID:          inv.ID,
		InvoiceDate: inv.InvoiceDate,
		RowVersion:  rowVer,
		UpdatedBy:   &user,
	}, ardb.InsertOutboxParams{
		AggregateType:  "CustomerInvoice",
		AggregateID:    inv.ID,
		EventType:      "ar.invoice.written_off",
		Payload:        payload,
		OrganizationID: inv.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "rowVersion tidak cocok atau invoice sudah lunas/dibatalkan"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, rowVer+1, map[string]any{
		"invoiceId":      inv.ID,
		"writeOffAmount": outRat.FloatString(4),
		"status":         "Cancelled",
		"writtenOff":     true,
	})
}
