// Package httpapi implements ar-service REST handlers with strict decimal checks,
// line-sum validation, idempotency hash protection, RBAC, and real pagination.
package httpapi

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"

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
	MarkPaid(ctx context.Context, scope []uuid.UUID, arg ardb.MarkInvoicePaidParams) (int64, error)
	DeleteDraft(ctx context.Context, scope []uuid.UUID, arg ardb.DeleteDraftInvoiceParams) (int64, error)
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
	mux.HandleFunc("GET /customer-invoices/{id}", s.get)
	mux.HandleFunc("POST /customer-invoices/{id}/pay", s.markPaid)
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
	var soRef *string
	if ref := strings.TrimSpace(req.SalesOrderRef); ref != "" {
		soRef = &ref
	}

	currency := deflt(req.CurrencyCode, "IDR")
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
	payload, _ := json.Marshal(map[string]any{
		"invoiceNo":    entry.InvoiceNo,
		"customerId":   customerID.String(),
		"invoiceDate":  platform.FormatDate(invDate),
		"dueDate":      platform.FormatDate(dueDate),
		"currencyCode": currency,
		"totalAmount":  platform.FormatNumeric(totalNum, 4),
		"taxAmount":    platform.FormatNumeric(taxNum, 4),
		"status":       "Issued",
		"lines":        eventLines,
	})
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
	n, err := s.repo.MarkPaid(r.Context(), scope, ardb.MarkInvoicePaidParams{
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
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Issued"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, rowVer+1, invoiceDTO{
		ID:                inv.ID.String(),
		InvoiceNo:         inv.InvoiceNo,
		InvoiceDate:       platform.FormatDate(inv.InvoiceDate),
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
