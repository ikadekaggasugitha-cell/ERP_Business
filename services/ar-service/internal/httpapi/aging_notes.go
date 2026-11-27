package httpapi

import (
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/sinergiinvestasiproperti/erp/services/ar-service/internal/ardb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

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
