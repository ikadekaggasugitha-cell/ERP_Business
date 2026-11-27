package httpapi

import (
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/gl-service/internal/gldb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type lineReq struct {
	LineNo           int32  `json:"lineNo"`
	AccountID        string `json:"accountId"`
	DebitAmount      string `json:"debitAmount"`
	CreditAmount     string `json:"creditAmount"`
	FunctionalAmount string `json:"functionalAmount"`
	Description      string `json:"description"`
}

type createReq struct {
	JournalNo               string    `json:"journalNo"`
	FinancialOrganizationID string    `json:"financialOrganizationId"`
	LedgerID                string    `json:"ledgerId"`
	FiscalPeriodID          string    `json:"fiscalPeriodId"`
	PostingDate             string    `json:"postingDate"`
	JournalSource           string    `json:"journalSource"`
	CurrencyCode            string    `json:"currencyCode"`
	ExchangeRate            string    `json:"exchangeRate"`
	SourceDocumentRef       string    `json:"sourceDocumentRef"`
	Lines                   []lineReq `json:"lines"`
}

type updateReq struct {
	CurrencyCode      string    `json:"currencyCode"`
	ExchangeRate      string    `json:"exchangeRate"`
	SourceDocumentRef string    `json:"sourceDocumentRef"`
	Lines             []lineReq `json:"lines"`
}

type lineDTO struct {
	ID               string `json:"id,omitempty"`
	LineNo           int32  `json:"lineNo"`
	AccountID        string `json:"accountId"`
	DebitAmount      string `json:"debitAmount"`
	CreditAmount     string `json:"creditAmount"`
	FunctionalAmount string `json:"functionalAmount"`
	Description      string `json:"description,omitempty"`
}

type entryDTO struct {
	ID                      string    `json:"id"`
	JournalNo               string    `json:"journalNo,omitempty"`
	FinancialOrganizationID string    `json:"financialOrganizationId,omitempty"`
	LedgerID                string    `json:"ledgerId,omitempty"`
	FiscalPeriodID          string    `json:"fiscalPeriodId,omitempty"`
	PostingDate             string    `json:"postingDate"`
	JournalSource           string    `json:"journalSource,omitempty"`
	CurrencyCode            string    `json:"currencyCode,omitempty"`
	ExchangeRate            string    `json:"exchangeRate"`
	TotalDebit              string    `json:"totalDebit"`
	TotalCredit             string    `json:"totalCredit"`
	SourceDocumentRef       string    `json:"sourceDocumentRef,omitempty"`
	ReversedEntryID         string    `json:"reversedEntryId,omitempty"`
	Status                  string    `json:"status"`
	RowVersion              int32     `json:"rowVersion"`
	Lines                   []lineDTO `json:"lines,omitempty"`
}

type balanceDTO struct {
	ID             string `json:"id"`
	LedgerID       string `json:"ledgerId"`
	AccountID      string `json:"accountId"`
	FiscalPeriodID string `json:"fiscalPeriodId"`
	OpeningBalance string `json:"openingBalance"`
	PeriodDebit    string `json:"periodDebit"`
	PeriodCredit   string `json:"periodCredit"`
	ClosingBalance string `json:"closingBalance"`
	RowVersion     int32  `json:"rowVersion"`
}

// ---- handlers --------------------------------------------------------------

func buildLineParams(orgID, user uuid.UUID, exRate pgtype.Numeric, reqLines []lineReq) ([]gldb.CreateJournalLineParams, error) {
	if len(reqLines) == 0 {
		return nil, httpx.BadRequest("lines minimal berisi 1 baris")
	}
	rateRat := platform.NumericToRat(exRate)
	if rateRat == nil || rateRat.Sign() <= 0 {
		rateRat = big.NewRat(1, 1)
	}
	seenLineNo := make(map[int32]bool, len(reqLines))
	lines := make([]gldb.CreateJournalLineParams, 0, len(reqLines))
	for _, l := range reqLines {
		if l.LineNo <= 0 {
			return nil, httpx.BadRequest("lines[].lineNo harus > 0")
		}
		if seenLineNo[l.LineNo] {
			return nil, httpx.BadRequest("lines[].lineNo tidak boleh duplikat")
		}
		seenLineNo[l.LineNo] = true

		acct, err := uuid.Parse(strings.TrimSpace(l.AccountID))
		if err != nil {
			return nil, httpx.BadRequest("lines[].accountId harus UUID valid")
		}
		dr, err := platform.ParseOptionalNumeric(l.DebitAmount)
		if err != nil {
			return nil, httpx.BadRequest("lines[].debitAmount tidak valid: " + err.Error())
		}
		cr, err := platform.ParseOptionalNumeric(l.CreditAmount)
		if err != nil {
			return nil, httpx.BadRequest("lines[].creditAmount tidak valid: " + err.Error())
		}
		drRat := platform.NumericToRat(dr)
		crRat := platform.NumericToRat(cr)
		if drRat.Sign() < 0 || crRat.Sign() < 0 {
			return nil, httpx.BadRequest("lines[].debitAmount dan creditAmount tidak boleh negatif")
		}
		if (drRat.Sign() > 0 && crRat.Sign() > 0) || (drRat.Sign() == 0 && crRat.Sign() == 0) {
			return nil, httpx.BadRequest("setiap baris jurnal harus mengisi tepat satu sisi debit atau kredit (> 0)")
		}

		var fnAmt pgtype.Numeric
		if strings.TrimSpace(l.FunctionalAmount) != "" {
			fnAmt, err = platform.ParseOptionalNumeric(l.FunctionalAmount)
			if err != nil {
				return nil, httpx.BadRequest("lines[].functionalAmount tidak valid: " + err.Error())
			}
			if fRat := platform.NumericToRat(fnAmt); fRat != nil && fRat.Sign() < 0 {
				return nil, httpx.BadRequest("lines[].functionalAmount tidak boleh negatif")
			}
		} else {
			base := drRat
			if crRat.Sign() > 0 {
				base = crRat
			}
			fnRat := new(big.Rat).Mul(base, rateRat)
			fnAmt, err = platform.RatToNumeric(fnRat, 4)
			if err != nil {
				return nil, httpx.BadRequest("gagal menghitung functionalAmount: " + err.Error())
			}
		}
		var desc *string
		if d := strings.TrimSpace(l.Description); d != "" {
			desc = &d
		}
		lines = append(lines, gldb.CreateJournalLineParams{
			OrganizationID:   orgID,
			LineNo:           l.LineNo,
			AccountID:        acct,
			DebitAmount:      dr,
			CreditAmount:     cr,
			FunctionalAmount: fnAmt,
			Description:      desc,
			CreatedBy:        user,
		})
	}
	return lines, nil
}

func toEntryDTO(e gldb.GlJournalEntry, lines []lineDTO) entryDTO {
	src := ""
	if e.SourceDocumentRef != nil {
		src = *e.SourceDocumentRef
	}
	rev := ""
	if e.ReversedEntryID != nil {
		rev = e.ReversedEntryID.String()
	}
	totDr, totCr := sumLineDTOs(lines)
	return entryDTO{
		ID:                      e.ID.String(),
		JournalNo:               e.JournalNo,
		FinancialOrganizationID: e.FinancialOrganizationID.String(),
		LedgerID:                e.LedgerID.String(),
		FiscalPeriodID:          e.FiscalPeriodID.String(),
		PostingDate:             dateStr(e.PostingDate),
		JournalSource:           e.JournalSource,
		CurrencyCode:            e.CurrencyCode,
		ExchangeRate:            formatExchangeRate(e.ExchangeRate),
		TotalDebit:              totDr,
		TotalCredit:             totCr,
		SourceDocumentRef:       src,
		ReversedEntryID:         rev,
		Status:                  e.Status,
		RowVersion:              e.RowVersion,
		Lines:                   lines,
	}
}

func formatExchangeRate(n pgtype.Numeric) string {
	if !n.Valid {
		return "1.00000000"
	}
	r := platform.NumericToRat(n)
	if r == nil || r.Sign() <= 0 {
		return "1.00000000"
	}
	return platform.FormatNumeric(n, 8)
}

func sumLineDTOs(lines []lineDTO) (string, string) {
	dr := new(big.Rat)
	cr := new(big.Rat)
	for _, l := range lines {
		if r, ok := new(big.Rat).SetString(strings.TrimSpace(l.DebitAmount)); ok {
			dr.Add(dr, r)
		}
		if r, ok := new(big.Rat).SetString(strings.TrimSpace(l.CreditAmount)); ok {
			cr.Add(cr, r)
		}
	}
	return dr.FloatString(4), cr.FloatString(4)
}

func sumLineParams(lines []gldb.CreateJournalLineParams) (string, string) {
	dr := new(big.Rat)
	cr := new(big.Rat)
	for _, l := range lines {
		dr.Add(dr, platform.NumericToRat(l.DebitAmount))
		cr.Add(cr, platform.NumericToRat(l.CreditAmount))
	}
	return dr.FloatString(4), cr.FloatString(4)
}

func scopeKey(s []uuid.UUID) string {
	parts := make([]string, len(s))
	for i, u := range s {
		parts[i] = u.String()
	}
	return strings.Join(parts, ",")
}

func parseDate(s string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func dateStr(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}

func deflt(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return strings.TrimSpace(v)
}

func marshalJSON(v any) ([]byte, error) {
	return jsonMarshal(v)
}
