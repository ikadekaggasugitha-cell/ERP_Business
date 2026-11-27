package httpapi

import (
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/gl-service/internal/gldb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type fxPositionReq struct {
	AccountID     string `json:"accountId"`
	CurrencyCode  string `json:"currencyCode"`
	ForeignAmount string `json:"foreignAmount"`
	BookedRate    string `json:"bookedRate"`
	ClosingRate   string `json:"closingRate"`
	Description   string `json:"description,omitempty"`
}

type fxRevaluationReq struct {
	JournalNo               string          `json:"journalNo"`
	FinancialOrganizationID string          `json:"financialOrganizationId"`
	LedgerID                string          `json:"ledgerId"`
	FiscalPeriodID          string          `json:"fiscalPeriodId"`
	PostingDate             string          `json:"postingDate"`
	FunctionalCurrency      string          `json:"functionalCurrency,omitempty"`
	FXGainAccountID         string          `json:"fxGainAccountId"`
	FXLossAccountID         string          `json:"fxLossAccountId"`
	Positions               []fxPositionReq `json:"positions"`
}

type fxPositionSummary struct {
	AccountID          string `json:"accountId"`
	CurrencyCode       string `json:"currencyCode"`
	ForeignAmount      string `json:"foreignAmount"`
	BookedRate         string `json:"bookedRate"`
	ClosingRate        string `json:"closingRate"`
	UnrealizedGainLoss string `json:"unrealizedGainLoss"`
}

type fxRevaluationResp struct {
	JournalEntry          *entryDTO           `json:"journalEntry,omitempty"`
	FunctionalCurrency    string              `json:"functionalCurrency"`
	TotalGain             string              `json:"totalGain"`
	TotalLoss             string              `json:"totalLoss"`
	NetUnrealizedGainLoss string              `json:"netUnrealizedGainLoss"`
	Positions             []fxPositionSummary `json:"positions"`
}

type recurringTemplateReq struct {
	Code                    string    `json:"code"`
	Name                    string    `json:"name"`
	Cadence                 string    `json:"cadence"` // Monthly | Quarterly
	FinancialOrganizationID string    `json:"financialOrganizationId"`
	LedgerID                string    `json:"ledgerId"`
	FiscalPeriodID          string    `json:"fiscalPeriodId"`
	PostingDate             string    `json:"postingDate,omitempty"`
	CurrencyCode            string    `json:"currencyCode,omitempty"`
	ExchangeRate            string    `json:"exchangeRate,omitempty"`
	GenerateDraft           bool      `json:"generateDraft,omitempty"`
	Lines                   []lineReq `json:"lines"`
}

type recurringTemplateDTO struct {
	ID                      string    `json:"id"`
	OrganizationID          string    `json:"organizationId"`
	Code                    string    `json:"code"`
	Name                    string    `json:"name"`
	Cadence                 string    `json:"cadence"`
	FinancialOrganizationID string    `json:"financialOrganizationId"`
	LedgerID                string    `json:"ledgerId"`
	FiscalPeriodID          string    `json:"fiscalPeriodId"`
	CurrencyCode            string    `json:"currencyCode"`
	ExchangeRate            string    `json:"exchangeRate"`
	TotalDebit              string    `json:"totalDebit"`
	TotalCredit             string    `json:"totalCredit"`
	Lines                   []lineReq `json:"lines"`
	CreatedAt               string    `json:"createdAt"`
	GeneratedEntry          *entryDTO `json:"generatedEntry,omitempty"`
}

func (s *Server) fxRevaluation(w http.ResponseWriter, r *http.Request) {
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
	if !httpx.HasPermission(r.Context(), "gl.create") && !httpx.HasPermission(r.Context(), "gl:write") {
		httpx.WriteError(w, r, httpx.Forbidden("izin gl.create dibutuhkan"))
		return
	}

	var req fxRevaluationReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if strings.TrimSpace(req.JournalNo) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("journalNo wajib diisi"))
		return
	}
	pd, err := parseDate(req.PostingDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("postingDate harus YYYY-MM-DD"))
		return
	}
	finOrg, e1 := uuid.Parse(req.FinancialOrganizationID)
	ledger, e2 := uuid.Parse(req.LedgerID)
	period, e3 := uuid.Parse(req.FiscalPeriodID)
	gainAcct, e4 := uuid.Parse(req.FXGainAccountID)
	lossAcct, e5 := uuid.Parse(req.FXLossAccountID)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil || e5 != nil {
		httpx.WriteError(w, r, httpx.BadRequest("financialOrganizationId/ledgerId/fiscalPeriodId/fxGainAccountId/fxLossAccountId harus UUID"))
		return
	}
	if len(req.Positions) == 0 {
		httpx.WriteError(w, r, httpx.BadRequest("minimal 1 posisi valas wajib disertakan"))
		return
	}

	funcCcy := strings.ToUpper(deflt(req.FunctionalCurrency, "IDR"))
	zeroNum, _ := platform.ParseNumeric("0.0000")
	oneRate, _ := platform.ParseNumeric("1.000000")

	totalGain := new(big.Rat)
	totalLoss := new(big.Rat)
	netDelta := new(big.Rat)
	summaries := make([]fxPositionSummary, 0, len(req.Positions))
	jlParams := make([]gldb.CreateJournalLineParams, 0, len(req.Positions)*2)
	lineNo := int32(1)

	for i, pos := range req.Positions {
		acctID, err := uuid.Parse(pos.AccountID)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("positions[%d].accountId harus UUID", i)))
			return
		}
		fAmtNum, err := platform.ParseNumeric(pos.ForeignAmount)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("positions[%d].foreignAmount tidak valid", i)))
			return
		}
		bRateNum, err := platform.ParseNumeric(pos.BookedRate)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("positions[%d].bookedRate tidak valid", i)))
			return
		}
		cRateNum, err := platform.ParseNumeric(pos.ClosingRate)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("positions[%d].closingRate tidak valid", i)))
			return
		}
		fRat := platform.NumericToRat(fAmtNum)
		bRat := platform.NumericToRat(bRateNum)
		cRat := platform.NumericToRat(cRateNum)
		if fRat == nil || bRat == nil || cRat == nil || bRat.Sign() <= 0 || cRat.Sign() <= 0 {
			httpx.WriteError(w, r, httpx.BadRequest(fmt.Sprintf("positions[%d] rate harus > 0", i)))
			return
		}

		// delta = (closingRate - bookedRate) * foreignAmount
		rateDiff := new(big.Rat).Sub(cRat, bRat)
		delta := new(big.Rat).Mul(rateDiff, fRat)
		netDelta.Add(netDelta, delta)

		deltaNum, _ := platform.RatToNumeric(delta, 4)
		ccy := strings.ToUpper(strings.TrimSpace(pos.CurrencyCode))
		if ccy == "" {
			ccy = "USD"
		}
		summaries = append(summaries, fxPositionSummary{
			AccountID:          acctID.String(),
			CurrencyCode:       ccy,
			ForeignAmount:      platform.FormatNumeric(fAmtNum, 4),
			BookedRate:         platform.FormatNumeric(bRateNum, 6),
			ClosingRate:        platform.FormatNumeric(cRateNum, 6),
			UnrealizedGainLoss: platform.FormatNumeric(deltaNum, 4),
		})

		if delta.Sign() == 0 {
			continue
		}
		desc := strings.TrimSpace(pos.Description)
		if desc == "" {
			desc = fmt.Sprintf("FX Revaluation %s (%s -> %s)", ccy, platform.FormatNumeric(bRateNum, 6), platform.FormatNumeric(cRateNum, 6))
		}
		if delta.Sign() > 0 {
			totalGain.Add(totalGain, delta)
			amtNum, _ := platform.RatToNumeric(delta, 4)
			jlParams = append(jlParams,
				gldb.CreateJournalLineParams{
					OrganizationID:   scope[0],
					LineNo:           lineNo,
					AccountID:        acctID,
					DebitAmount:      amtNum,
					CreditAmount:     zeroNum,
					FunctionalAmount: amtNum,
					Description:      &desc,
					CreatedBy:        user,
				},
				gldb.CreateJournalLineParams{
					OrganizationID:   scope[0],
					LineNo:           lineNo + 1,
					AccountID:        gainAcct,
					DebitAmount:      zeroNum,
					CreditAmount:     amtNum,
					FunctionalAmount: amtNum,
					Description:      &desc,
					CreatedBy:        user,
				},
			)
		} else {
			absLoss := new(big.Rat).Abs(delta)
			totalLoss.Add(totalLoss, absLoss)
			amtNum, _ := platform.RatToNumeric(absLoss, 4)
			jlParams = append(jlParams,
				gldb.CreateJournalLineParams{
					OrganizationID:   scope[0],
					LineNo:           lineNo,
					AccountID:        lossAcct,
					DebitAmount:      amtNum,
					CreditAmount:     zeroNum,
					FunctionalAmount: amtNum,
					Description:      &desc,
					CreatedBy:        user,
				},
				gldb.CreateJournalLineParams{
					OrganizationID:   scope[0],
					LineNo:           lineNo + 1,
					AccountID:        acctID,
					DebitAmount:      zeroNum,
					CreditAmount:     amtNum,
					FunctionalAmount: amtNum,
					Description:      &desc,
					CreatedBy:        user,
				},
			)
		}
		lineNo += 2
	}

	gainNum, _ := platform.RatToNumeric(totalGain, 4)
	lossNum, _ := platform.RatToNumeric(totalLoss, 4)
	netNum, _ := platform.RatToNumeric(netDelta, 4)
	resp := fxRevaluationResp{
		FunctionalCurrency:    funcCcy,
		TotalGain:             platform.FormatNumeric(gainNum, 4),
		TotalLoss:             platform.FormatNumeric(lossNum, 4),
		NetUnrealizedGainLoss: platform.FormatNumeric(netNum, 4),
		Positions:             summaries,
	}

	if len(jlParams) == 0 {
		httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, 0, resp)
		return
	}

	srcRef := "FX-REVAL-" + strings.TrimSpace(req.JournalNo)
	entry := gldb.CreateJournalEntryParams{
		OrganizationID:          scope[0],
		JournalNo:               strings.TrimSpace(req.JournalNo),
		FinancialOrganizationID: finOrg,
		LedgerID:                ledger,
		FiscalPeriodID:          period,
		PostingDate:             pd,
		JournalSource:           "FX_Revaluation",
		SourceDocumentRef:       &srcRef,
		CurrencyCode:            funcCcy,
		ExchangeRate:            oneRate,
		CorrelationID:           httpx.CorrelationUUID(r),
		CreatedBy:               user,
	}
	created, err := s.repo.CreateEntryWithLines(r.Context(), scope, entry, jlParams)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "journalNo sudah ada di tahun fiskal ini"))
		return
	}
	totDr, totCr := sumLineParams(jlParams)
	dto := entryDTO{
		ID:                      created.ID.String(),
		JournalNo:               entry.JournalNo,
		FinancialOrganizationID: entry.FinancialOrganizationID.String(),
		LedgerID:                entry.LedgerID.String(),
		FiscalPeriodID:          entry.FiscalPeriodID.String(),
		PostingDate:             dateStr(created.PostingDate),
		JournalSource:           entry.JournalSource,
		CurrencyCode:            entry.CurrencyCode,
		ExchangeRate:            formatExchangeRate(entry.ExchangeRate),
		TotalDebit:              totDr,
		TotalCredit:             totCr,
		SourceDocumentRef:       srcRef,
		Status:                  created.Status,
		RowVersion:              created.RowVersion,
	}
	resp.JournalEntry = &dto
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, resp)
}

func (s *Server) createRecurringTemplate(w http.ResponseWriter, r *http.Request) {
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
	if !httpx.HasPermission(r.Context(), "gl.create") && !httpx.HasPermission(r.Context(), "gl:write") {
		httpx.WriteError(w, r, httpx.Forbidden("izin gl.create dibutuhkan"))
		return
	}

	var req recurringTemplateReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan name wajib diisi"))
		return
	}
	cadence := strings.TrimSpace(req.Cadence)
	switch cadence {
	case "Monthly", "Quarterly":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("cadence harus Monthly atau Quarterly"))
		return
	}
	finOrg, e1 := uuid.Parse(req.FinancialOrganizationID)
	ledger, e2 := uuid.Parse(req.LedgerID)
	period, e3 := uuid.Parse(req.FiscalPeriodID)
	if e1 != nil || e2 != nil || e3 != nil {
		httpx.WriteError(w, r, httpx.BadRequest("financialOrganizationId/ledgerId/fiscalPeriodId harus UUID"))
		return
	}
	exRate, err := platform.ParseOptionalNumeric(deflt(req.ExchangeRate, "1"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("exchangeRate bukan angka desimal valid"))
		return
	}
	if rVal := platform.NumericToRat(exRate); rVal == nil || rVal.Sign() <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("exchangeRate harus > 0"))
		return
	}
	lines, err := buildLineParams(scope[0], user, exRate, req.Lines)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	totDr, totCr := sumLineParams(lines)
	ccy := strings.ToUpper(deflt(req.CurrencyCode, "IDR"))
	tpl := recurringTemplateDTO{
		ID:                      uuid.NewString(),
		OrganizationID:          scope[0].String(),
		Code:                    code,
		Name:                    name,
		Cadence:                 cadence,
		FinancialOrganizationID: finOrg.String(),
		LedgerID:                ledger.String(),
		FiscalPeriodID:          period.String(),
		CurrencyCode:            ccy,
		ExchangeRate:            formatExchangeRate(exRate),
		TotalDebit:              totDr,
		TotalCredit:             totCr,
		Lines:                   req.Lines,
		CreatedAt:               time.Now().UTC().Format(time.RFC3339),
	}

	if req.GenerateDraft {
		pd := pgtype.Date{Time: time.Now().UTC(), Valid: true}
		if strings.TrimSpace(req.PostingDate) != "" {
			if pd, err = parseDate(req.PostingDate); err != nil {
				httpx.WriteError(w, r, httpx.BadRequest("postingDate harus YYYY-MM-DD"))
				return
			}
		}
		jNo := fmt.Sprintf("REC-%s-%s", code, pd.Time.UTC().Format("200601"))
		srcRef := "RECURRING:" + code
		entry := gldb.CreateJournalEntryParams{
			OrganizationID:          scope[0],
			JournalNo:               jNo,
			FinancialOrganizationID: finOrg,
			LedgerID:                ledger,
			FiscalPeriodID:          period,
			PostingDate:             pd,
			JournalSource:           "Recurring",
			SourceDocumentRef:       &srcRef,
			CurrencyCode:            ccy,
			ExchangeRate:            exRate,
			CorrelationID:           httpx.CorrelationUUID(r),
			CreatedBy:               user,
		}
		created, err := s.repo.CreateEntryWithLines(r.Context(), scope, entry, lines)
		if err != nil {
			httpx.WriteError(w, r, httpx.MapDBError(err, "journalNo sudah ada di tahun fiskal ini"))
			return
		}
		tpl.GeneratedEntry = &entryDTO{
			ID:                      created.ID.String(),
			JournalNo:               entry.JournalNo,
			FinancialOrganizationID: entry.FinancialOrganizationID.String(),
			LedgerID:                entry.LedgerID.String(),
			FiscalPeriodID:          entry.FiscalPeriodID.String(),
			PostingDate:             dateStr(created.PostingDate),
			JournalSource:           entry.JournalSource,
			CurrencyCode:            entry.CurrencyCode,
			ExchangeRate:            formatExchangeRate(entry.ExchangeRate),
			TotalDebit:              totDr,
			TotalCredit:             totCr,
			SourceDocumentRef:       srcRef,
			Status:                  created.Status,
			RowVersion:              created.RowVersion,
		}
	}

	s.recMu.Lock()
	s.recurring[scope[0]] = append(s.recurring[scope[0]], tpl)
	s.recMu.Unlock()

	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, 1, tpl)
}

func (s *Server) listRecurringTemplates(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if !httpx.HasPermission(r.Context(), "gl.read") && !httpx.HasPermission(r.Context(), "gl:read") {
		httpx.WriteError(w, r, httpx.Forbidden("izin gl.read dibutuhkan"))
		return
	}
	page, size, _, _ := httpx.PaginateParams(r)

	s.recMu.RLock()
	out := make([]recurringTemplateDTO, 0)
	for _, orgID := range scope {
		if items, ok := s.recurring[orgID]; ok {
			out = append(out, items...)
		}
	}
	s.recMu.RUnlock()

	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), len(out)),
	})
}
