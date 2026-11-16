// Package httpapi implements ap-service REST endpoints with strict decimals,
// optimistic locking, idempotency hash protection, RBAC, and clean error mapping.
package httpapi

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/ap-service/internal/apdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Store abstracts AP repository persistence for unit tests.
type Store interface {
	CreateInvoiceWithLines(ctx context.Context, scope []uuid.UUID, invoice apdb.CreateInvoiceParams, lines []apdb.CreateInvoiceLineParams) (apdb.CreateInvoiceRow, error)
	GetWithLines(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (apdb.ApApInvoice, []apdb.ListInvoiceLinesRow, error)
	GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (apdb.ApApInvoice, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, status string, limit, offset int32) ([]apdb.ListInvoicesRow, int64, error)
	UpdateDraftWithLines(ctx context.Context, scope []uuid.UUID, hdr apdb.UpdateDraftInvoiceParams, lines []apdb.CreateInvoiceLineParams) (int32, error)
	DeleteDraft(ctx context.Context, scope []uuid.UUID, arg apdb.DeleteDraftInvoiceParams) (int64, error)
	RecordAndEmit(ctx context.Context, scope []uuid.UUID, rec apdb.RecordInvoiceParams, ev apdb.InsertOutboxParams) (int64, error)
}

// Server serves AP HTTP endpoints.
type Server struct {
	repo Store
	idem *httpx.IdemStore
}

// NewServer constructs an AP HTTP Server.
func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

// Register mounts AP routes onto mux.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /vendor-invoices", s.create)
	mux.HandleFunc("GET /vendor-invoices", s.list)
	mux.HandleFunc("GET /vendor-invoices/{id}", s.get)
	mux.HandleFunc("PUT /vendor-invoices/{id}", s.update)
	mux.HandleFunc("DELETE /vendor-invoices/{id}", s.deleteDraft)
	mux.HandleFunc("POST /vendor-invoices/{id}/record", s.record)
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
	InvoiceNo       string    `json:"invoiceNo"`
	VendorID        string    `json:"vendorId"`
	InvoiceDate     string    `json:"invoiceDate"`
	DueDate         string    `json:"dueDate"`
	CurrencyCode    string    `json:"currencyCode"`
	TotalAmount     string    `json:"totalAmount"`
	GoodsReceiptRef string    `json:"goodsReceiptRef,omitempty"`
	Lines           []lineReq `json:"lines"`
}

type updateReq struct {
	VendorID        string    `json:"vendorId"`
	DueDate         string    `json:"dueDate"`
	CurrencyCode    string    `json:"currencyCode"`
	TotalAmount     string    `json:"totalAmount"`
	GoodsReceiptRef string    `json:"goodsReceiptRef,omitempty"`
	Lines           []lineReq `json:"lines"`
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
	VendorID          string    `json:"vendorId,omitempty"`
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
	if err := httpx.RequirePermission(r, "ap:write"); err != nil {
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
		httpx.WriteError(w, r, httpx.BadRequest("invoiceDate harus YYYY-MM-DD", httpx.Detail{Field: "invoiceDate", Issue: err.Error()}))
		return
	}
	dueDate, err := platform.ParseDate(req.DueDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("dueDate harus YYYY-MM-DD", httpx.Detail{Field: "dueDate", Issue: err.Error()}))
		return
	}
	vendorID, err := httpx.ParseUUIDParam("vendorId", req.VendorID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.InvoiceNo) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("invoiceNo wajib diisi", httpx.Detail{Field: "invoiceNo", Issue: "required"}))
		return
	}
	total, lines, err := validateTotalAndLines(scope[0], user, req.TotalAmount, req.Lines)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var grRef *string
	if strings.TrimSpace(req.GoodsReceiptRef) != "" {
		g := strings.TrimSpace(req.GoodsReceiptRef)
		grRef = &g
	}
	entry := apdb.CreateInvoiceParams{
		OrganizationID:    scope[0],
		InvoiceNo:         strings.TrimSpace(req.InvoiceNo),
		VendorID:          vendorID,
		InvoiceDate:       invDate,
		DueDate:           dueDate,
		CurrencyCode:      deflt(req.CurrencyCode, "IDR"),
		TotalAmount:       total,
		OutstandingAmount: total,
		GoodsReceiptRef:   grRef,
		CreatedBy:         user,
	}
	created, err := s.repo.CreateInvoiceWithLines(r.Context(), scope, entry, lines)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "nomor dokumen sudah dipakai"))
		return
	}
	dto := invoiceDTO{
		ID:          created.ID.String(),
		InvoiceNo:   entry.InvoiceNo,
		VendorID:    vendorID.String(),
		InvoiceDate: platform.FormatDate(created.InvoiceDate),
		DueDate:     platform.FormatDate(dueDate),
		TotalAmount: platform.FormatNumeric(total, 4),
		Status:      created.Status,
		RowVersion:  created.RowVersion,
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, dto)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "ap:read"); err != nil {
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
		VendorID:          inv.VendorID.String(),
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
	if err := httpx.RequirePermission(r, "ap:read"); err != nil {
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
			VendorID:          inv.VendorID.String(),
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

func (s *Server) update(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "ap:write"); err != nil {
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
	dueDate, err := platform.ParseDate(req.DueDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("dueDate harus YYYY-MM-DD"))
		return
	}
	vendorID, err := httpx.ParseUUIDParam("vendorId", req.VendorID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	inv, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	total, lines, err := validateTotalAndLines(inv.OrganizationID, user, req.TotalAmount, req.Lines)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var grRef *string
	if strings.TrimSpace(req.GoodsReceiptRef) != "" {
		g := strings.TrimSpace(req.GoodsReceiptRef)
		grRef = &g
	}
	newVer, err := s.repo.UpdateDraftWithLines(r.Context(), scope, apdb.UpdateDraftInvoiceParams{
		ID:              inv.ID,
		InvoiceDate:     inv.InvoiceDate,
		RowVersion:      rowVer,
		VendorID:        vendorID,
		DueDate:         dueDate,
		CurrencyCode:    deflt(req.CurrencyCode, inv.CurrencyCode),
		TotalAmount:     total,
		GoodsReceiptRef: grRef,
		UpdatedBy:       &user,
	}, lines)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, invoiceDTO{
		ID:           inv.ID.String(),
		InvoiceNo:    inv.InvoiceNo,
		VendorID:     vendorID.String(),
		InvoiceDate:  platform.FormatDate(inv.InvoiceDate),
		DueDate:      platform.FormatDate(dueDate),
		CurrencyCode: deflt(req.CurrencyCode, inv.CurrencyCode),
		TotalAmount:  platform.FormatNumeric(total, 4),
		Status:       inv.Status,
		RowVersion:   newVer,
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
	if err := httpx.RequirePermission(r, "ap:write"); err != nil {
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
	n, err := s.repo.DeleteDraft(r.Context(), scope, apdb.DeleteDraftInvoiceParams{
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

func (s *Server) record(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "ap:write"); err != nil {
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
	payload, _ := json.Marshal(map[string]any{
		"invoiceNo":    inv.InvoiceNo,
		"vendorId":     inv.VendorID.String(),
		"invoiceDate":  platform.FormatDate(inv.InvoiceDate),
		"dueDate":      platform.FormatDate(inv.DueDate),
		"currencyCode": inv.CurrencyCode,
		"totalAmount":  platform.FormatNumeric(inv.TotalAmount, 4),
		"status":       "Recorded",
	})
	ev := apdb.InsertOutboxParams{
		AggregateType:  "APInvoice",
		AggregateID:    inv.ID,
		EventType:      "ap.invoice.recorded",
		Payload:        payload,
		OrganizationID: inv.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	n, err := s.repo.RecordAndEmit(r.Context(), scope,
		apdb.RecordInvoiceParams{ID: id, InvoiceDate: inv.InvoiceDate, RowVersion: rowVer, UpdatedBy: &user}, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Draft"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, rowVer+1, invoiceDTO{
		ID:          id.String(),
		InvoiceNo:   inv.InvoiceNo,
		VendorID:    inv.VendorID.String(),
		InvoiceDate: platform.FormatDate(inv.InvoiceDate),
		DueDate:     platform.FormatDate(inv.DueDate),
		TotalAmount: platform.FormatNumeric(inv.TotalAmount, 4),
		Status:      "Recorded",
		RowVersion:  rowVer + 1,
	})
}

func validateTotalAndLines(orgID, user uuid.UUID, totalRaw string, reqLines []lineReq) (pgtype.Numeric, []apdb.CreateInvoiceLineParams, error) {
	totalNum, err := platform.ParseNumeric(totalRaw)
	if err != nil {
		return pgtype.Numeric{}, nil, httpx.BadRequest("totalAmount tidak valid", httpx.Detail{Field: "totalAmount", Issue: err.Error()})
	}
	totalRat := platform.NumericToRat(totalNum)
	if totalRat.Sign() <= 0 {
		return pgtype.Numeric{}, nil, httpx.BadRequest("totalAmount harus > 0", httpx.Detail{Field: "totalAmount", Issue: "must be positive"})
	}

	lineSum := new(big.Rat)
	lines := make([]apdb.CreateInvoiceLineParams, 0, len(reqLines))
	for _, l := range reqLines {
		acct, err := httpx.ParseUUIDParam("lines[].accountId", l.AccountID)
		if err != nil {
			return pgtype.Numeric{}, nil, err
		}
		amtNum, err := platform.ParseNumeric(l.Amount)
		if err != nil {
			return pgtype.Numeric{}, nil, httpx.BadRequest("lines[].amount tidak valid", httpx.Detail{Field: "lines[].amount", Issue: err.Error()})
		}
		amtRat := platform.NumericToRat(amtNum)
		if amtRat.Sign() < 0 {
			return pgtype.Numeric{}, nil, httpx.BadRequest("lines[].amount tidak boleh negatif")
		}
		lineSum.Add(lineSum, amtRat)

		var itemPtr *uuid.UUID
		if strings.TrimSpace(l.ItemID) != "" {
			u, err := httpx.ParseUUIDParam("lines[].itemId", l.ItemID)
			if err != nil {
				return pgtype.Numeric{}, nil, err
			}
			itemPtr = &u
		}
		var qty, unitPrice pgtype.Numeric
		if strings.TrimSpace(l.Quantity) != "" {
			qty, err = platform.ParseNumeric(l.Quantity)
			if err != nil {
				return pgtype.Numeric{}, nil, httpx.BadRequest("lines[].quantity tidak valid")
			}
		}
		if strings.TrimSpace(l.UnitPrice) != "" {
			unitPrice, err = platform.ParseNumeric(l.UnitPrice)
			if err != nil {
				return pgtype.Numeric{}, nil, httpx.BadRequest("lines[].unitPrice tidak valid")
			}
		}
		var desc *string
		if d := strings.TrimSpace(l.Description); d != "" {
			desc = &d
		}
		lines = append(lines, apdb.CreateInvoiceLineParams{
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
	}
	if len(lines) > 0 && lineSum.Cmp(totalRat) != 0 {
		return pgtype.Numeric{}, nil, httpx.Unprocessable("AMOUNT_MISMATCH", "jumlah lines[].amount harus sama dengan totalAmount")
	}
	return totalNum, lines, nil
}

func deflt(v, d string) string {
	if s := strings.TrimSpace(v); s != "" {
		return s
	}
	return d
}
