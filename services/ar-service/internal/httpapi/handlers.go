// Package httpapi implements ar-service REST handlers with strict decimal checks,
// line-sum validation, idempotency hash protection, RBAC, and real pagination.
package httpapi

import (
	"context"
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

func requireARApprove(r *http.Request) error {
	if err := httpx.RequirePermission(r, "ar.approve"); err != nil {
		if httpx.RequirePermission(r, "ar:approve") == nil {
			return nil
		}
		return err
	}
	return nil
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
type updateReq struct {
	CustomerID    string    `json:"customerId"`
	DueDate       string    `json:"dueDate"`
	CurrencyCode  string    `json:"currencyCode"`
	TotalAmount   string    `json:"totalAmount"`
	SalesOrderRef string    `json:"salesOrderRef,omitempty"`
	Lines         []lineReq `json:"lines"`
}
