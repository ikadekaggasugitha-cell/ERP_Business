package httpapi

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/treasury-cashbank-service/internal/treasurydb"
)

func safeCSVCell(s string) string {
	if s == "" {
		return ""
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

func (s *Server) exportDisbursement(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "treasury:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	format := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "BIFAST"
	}
	switch format {
	case "BIFAST", "LLG", "SKN":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("format harus BIFAST, LLG, atau SKN"))
		return
	}

	payments, _, err := s.repo.ListWithCount(r.Context(), scope, treasurydb.ListPaymentsParams{
		Limit:  1000,
		Offset: 0,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}

	var buf bytes.Buffer
	cw := csv.NewWriter(&buf)
	_ = cw.Write([]string{"transfer_channel", "payment_no", "payment_date", "bank_account_id", "amount", "status", "external_ref"})
	for _, p := range payments {
		if !strings.EqualFold(p.PaymentType, "Outgoing") {
			continue
		}
		extRef := ""
		if p.ExternalRef != nil {
			extRef = *p.ExternalRef
		}
		_ = cw.Write([]string{
			format,
			safeCSVCell(p.PaymentNo),
			platform.FormatDate(p.PaymentDate),
			p.BankAccountID.String(),
			platform.FormatNumeric(p.Amount, 4),
			safeCSVCell(p.Status),
			safeCSVCell(extRef),
		})
	}
	cw.Flush()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="disbursement-%s.csv"`, strings.ToLower(format)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
