package httpapi

import (
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/consolidation-service/internal/consdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListRuns(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, run := range rows {
		out = append(out, runDTO(run))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createRun(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		ConsolidationGroupID uuid.UUID `json:"consolidationGroupId"`
		FiscalPeriodRef      uuid.UUID `json:"fiscalPeriodRef"`
		Version              int32     `json:"version"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if req.ConsolidationGroupID == uuid.Nil || req.FiscalPeriodRef == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("consolidationGroupId and fiscalPeriodRef are required"))
		return
	}
	ver := req.Version
	if ver <= 0 {
		ver = 1
	}
	created, err := s.repo.CreateRun(r.Context(), scope, consdb.CreateConsolidationRunParams{
		OrganizationID: scope[0], ConsolidationGroupID: req.ConsolidationGroupID,
		FiscalPeriodRef: req.FiscalPeriodRef, Version: ver, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{
		"id": created.ID, "status": created.Status, "rowVersion": created.RowVersion,
	})
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	run, elims, adjs, err := s.repo.GetRunDetail(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	elimDTOs := make([]map[string]any, 0, len(elims))
	for _, e := range elims {
		elimDTOs = append(elimDTOs, eliminationEntryDTO(e))
	}
	adjDTOs := make([]map[string]any, 0, len(adjs))
	for _, a := range adjs {
		adjDTOs = append(adjDTOs, adjustmentDTO(a))
	}
	httpx.WriteIDResponse(w, http.StatusOK, run.RowVersion, map[string]any{
		"run": runDTO(run), "eliminations": elimDTOs, "adjustments": adjDTOs,
	})
}

func (s *Server) postEliminations(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	runID, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		Entries []struct {
			EliminationRuleID *uuid.UUID `json:"eliminationRuleId,omitempty"`
			AccountRef        uuid.UUID  `json:"accountRef"`
			DebitAmount       string     `json:"debitAmount"`
			CreditAmount      string     `json:"creditAmount"`
			Description       *string    `json:"description,omitempty"`
		} `json:"entries"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if len(req.Entries) < 2 {
		httpx.WriteError(w, r, httpx.Unprocessable("CONSOLIDATION_UNBALANCED", "at least 2 balanced elimination entries are required"))
		return
	}
	totDr, totCr := new(big.Rat), new(big.Rat)
	params := make([]consdb.CreateEliminationEntryParams, 0, len(req.Entries))
	for _, e := range req.Entries {
		dr, err := platform.ParseOptionalNumeric(e.DebitAmount)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid debitAmount"))
			return
		}
		cr, err := platform.ParseOptionalNumeric(e.CreditAmount)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid creditAmount"))
			return
		}
		drRat, crRat := platform.NumericToRat(dr), platform.NumericToRat(cr)
		if drRat.Sign() < 0 || crRat.Sign() < 0 || (drRat.Sign() > 0 && crRat.Sign() > 0) || (drRat.Sign() == 0 && crRat.Sign() == 0) {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_ENTRY_LINE", "each elimination entry must have strictly one positive debit or credit amount"))
			return
		}
		totDr.Add(totDr, drRat)
		totCr.Add(totCr, crRat)
		params = append(params, consdb.CreateEliminationEntryParams{
			OrganizationID: scope[0], ConsolidationRunID: runID, EliminationRuleID: e.EliminationRuleID,
			AccountRef: e.AccountRef, DebitAmount: dr, CreditAmount: cr, Description: e.Description, CreatedBy: user,
		})
	}
	if totDr.Sign() <= 0 || totDr.Cmp(totCr) != 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("CONSOLIDATION_UNBALANCED", "elimination entries must balance (debit == credit > 0)"))
		return
	}
	next, err := s.repo.PostEliminations(r.Context(), scope, runID, rowVer, params, user)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, next, map[string]any{"id": runID, "status": "Eliminated", "rowVersion": next})
}

func (s *Server) postAdjustments(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	runID, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		Adjustments []struct {
			AccountRef     uuid.UUID `json:"accountRef"`
			DebitAmount    string    `json:"debitAmount"`
			CreditAmount   string    `json:"creditAmount"`
			AdjustmentType string    `json:"adjustmentType"`
			Notes          *string   `json:"notes,omitempty"`
		} `json:"adjustments"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if len(req.Adjustments) < 2 {
		httpx.WriteError(w, r, httpx.Unprocessable("CONSOLIDATION_UNBALANCED", "at least 2 balanced adjustment entries are required"))
		return
	}
	totDr, totCr := new(big.Rat), new(big.Rat)
	params := make([]consdb.CreateConsolidationAdjustmentParams, 0, len(req.Adjustments))
	for _, a := range req.Adjustments {
		if a.AdjustmentType != "TopSide" && a.AdjustmentType != "Reclass" && a.AdjustmentType != "Policy" {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_ADJUSTMENT_TYPE", "adjustmentType must be TopSide, Reclass, or Policy"))
			return
		}
		dr, err := platform.ParseOptionalNumeric(a.DebitAmount)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid debitAmount"))
			return
		}
		cr, err := platform.ParseOptionalNumeric(a.CreditAmount)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid creditAmount"))
			return
		}
		drRat, crRat := platform.NumericToRat(dr), platform.NumericToRat(cr)
		if drRat.Sign() < 0 || crRat.Sign() < 0 || (drRat.Sign() > 0 && crRat.Sign() > 0) || (drRat.Sign() == 0 && crRat.Sign() == 0) {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_ENTRY_LINE", "each adjustment entry must have strictly one positive debit or credit amount"))
			return
		}
		totDr.Add(totDr, drRat)
		totCr.Add(totCr, crRat)
		params = append(params, consdb.CreateConsolidationAdjustmentParams{
			OrganizationID: scope[0], ConsolidationRunID: runID, AccountRef: a.AccountRef,
			DebitAmount: dr, CreditAmount: cr, AdjustmentType: a.AdjustmentType, Notes: a.Notes, CreatedBy: user,
		})
	}
	if totDr.Sign() <= 0 || totDr.Cmp(totCr) != 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("CONSOLIDATION_UNBALANCED", "adjustments must balance (debit == credit > 0)"))
		return
	}
	next, err := s.repo.PostAdjustments(r.Context(), scope, runID, rowVer, params, user)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, next, map[string]any{"id": runID, "status": "Adjusted", "rowVersion": next})
}

func (s *Server) finalizeRun(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	runID, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	idemKey, reqHash, release, done := beginIdemOptional(w, r, s.idem, scope, nil)
	if done {
		return
	}
	defer release()
	next, err := s.repo.FinalizeRunAndEmit(r.Context(), scope, runID, rowVer, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, next, map[string]any{"id": runID, "status": "Finalized", "rowVersion": next})
}

func (s *Server) generateEliminations(w http.ResponseWriter, r *http.Request) {
	scope, _, ok := requireWrite(w, r)
	if !ok {
		return
	}
	runID, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		IntercompanyBalances []struct {
			FromEntityID string `json:"fromEntityId"`
			ToEntityID   string `json:"toEntityId"`
			AccountCode  string `json:"accountCode"`
			Amount       string `json:"amount"`
		} `json:"intercompanyBalances"`
		SubsidiaryEquity []struct {
			EntityID         string `json:"entityId"`
			EquityAmount     string `json:"equityAmount"`
			NetIncome        string `json:"netIncome"`
			OwnershipPercent string `json:"ownershipPercent"`
		} `json:"subsidiaryEquity"`
		FXTranslations []struct {
			EntityID           string `json:"entityId"`
			FunctionalCurrency string `json:"functionalCurrency"`
			ReportingCurrency  string `json:"reportingCurrency"`
			ForeignNetAssets   string `json:"foreignNetAssets"`
			HistoricalRate     string `json:"historicalRate"`
			ClosingRate        string `json:"closingRate"`
		} `json:"fxTranslations"`
	}
	idemKey, reqHash, release, done := beginIdemOptional(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if _, _, _, err := s.repo.GetRunDetail(r.Context(), scope, runID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	hundred := big.NewRat(100, 1)
	icTotal := new(big.Rat)
	nciEquityTotal := new(big.Rat)
	nciNetIncTotal := new(big.Rat)
	ctaTotal := new(big.Rat)
	entries := make([]map[string]any, 0)

	for _, ic := range req.IntercompanyBalances {
		num, perr := platform.ParseNumeric(ic.Amount)
		if perr != nil {
			httpx.WriteError(w, r, httpx.BadRequest("intercompanyBalances[].amount is invalid"))
			return
		}
		amt := platform.NumericToRat(num)
		if amt == nil || amt.Sign() <= 0 {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_AMOUNT", "intercompany amount must be > 0"))
			return
		}
		icTotal.Add(icTotal, amt)
		entries = append(entries, map[string]any{
			"type":         "IC_ELIMINATION",
			"fromEntityId": ic.FromEntityID,
			"toEntityId":   ic.ToEntityID,
			"accountCode":  ic.AccountCode,
			"debit":        amt.FloatString(4),
			"credit":       amt.FloatString(4),
		})
	}

	for _, sub := range req.SubsidiaryEquity {
		ownNum, perr := platform.ParseNumeric(sub.OwnershipPercent)
		if perr != nil {
			httpx.WriteError(w, r, httpx.BadRequest("ownershipPercent is invalid"))
			return
		}
		ownPct := platform.NumericToRat(ownNum)
		if ownPct == nil || ownPct.Sign() < 0 || ownPct.Cmp(hundred) > 0 {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_OWNERSHIP", "ownershipPercent must be between 0 and 100"))
			return
		}
		nciPct := new(big.Rat).Sub(hundred, ownPct)
		nciRatio := new(big.Rat).Quo(nciPct, hundred)

		eqRat := new(big.Rat)
		if strings.TrimSpace(sub.EquityAmount) != "" {
			if n, e := platform.ParseNumeric(sub.EquityAmount); e == nil {
				eqRat = platform.NumericToRat(n)
			}
		}
		niRat := new(big.Rat)
		if strings.TrimSpace(sub.NetIncome) != "" {
			if n, e := platform.ParseNumeric(sub.NetIncome); e == nil {
				niRat = platform.NumericToRat(n)
			}
		}
		nciEq := new(big.Rat).Mul(eqRat, nciRatio)
		nciNI := new(big.Rat).Mul(niRat, nciRatio)
		nciEquityTotal.Add(nciEquityTotal, nciEq)
		nciNetIncTotal.Add(nciNetIncTotal, nciNI)
		entries = append(entries, map[string]any{
			"type":             "NCI_ALLOCATION",
			"entityId":         sub.EntityID,
			"nciPercent":       nciPct.FloatString(4),
			"nciEquity":        nciEq.FloatString(4),
			"nciNetIncome":     nciNI.FloatString(4),
			"ownershipPercent": ownPct.FloatString(4),
		})
	}

	for _, fx := range req.FXTranslations {
		fnaNum, err1 := platform.ParseNumeric(fx.ForeignNetAssets)
		hrNum, err2 := platform.ParseNumeric(fx.HistoricalRate)
		crNum, err3 := platform.ParseNumeric(fx.ClosingRate)
		if err1 != nil || err2 != nil || err3 != nil {
			httpx.WriteError(w, r, httpx.BadRequest("fxTranslations numeric fields are invalid"))
			return
		}
		diff := new(big.Rat).Sub(platform.NumericToRat(crNum), platform.NumericToRat(hrNum))
		cta := new(big.Rat).Mul(platform.NumericToRat(fnaNum), diff)
		ctaTotal.Add(ctaTotal, cta)
		entries = append(entries, map[string]any{
			"type":               "CTA_TRANSLATION",
			"entityId":           fx.EntityID,
			"functionalCurrency": fx.FunctionalCurrency,
			"reportingCurrency":  fx.ReportingCurrency,
			"ctaAmount":          cta.FloatString(4),
		})
	}

	nciCombined := new(big.Rat).Add(nciEquityTotal, nciNetIncTotal)
	s.mu.Lock()
	s.gen[runID] = generatedConsSummary{
		ICElimTotal: icTotal.FloatString(4),
		NCIEquity:   nciEquityTotal.FloatString(4),
		NCINetInc:   nciNetIncTotal.FloatString(4),
		CTAReserve:  ctaTotal.FloatString(4),
	}
	s.mu.Unlock()

	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusOK, 1, map[string]any{
		"runId":              runID,
		"icEliminationTotal": icTotal.FloatString(4),
		"nciEquityTotal":     nciEquityTotal.FloatString(4),
		"nciNetIncomeTotal":  nciNetIncTotal.FloatString(4),
		"nciTotal":           nciCombined.FloatString(4),
		"ctaTotal":           ctaTotal.FloatString(4),
		"balanced":           true,
		"entries":            entries,
	})
}

func (s *Server) consolidatedStatements(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	runID, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	run, elims, adjs, err := s.repo.GetRunDetail(r.Context(), scope, runID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	totDr := new(big.Rat)
	totCr := new(big.Rat)
	for _, e := range elims {
		if d := platform.NumericToRat(e.DebitAmount); d != nil {
			totDr.Add(totDr, d)
		}
		if c := platform.NumericToRat(e.CreditAmount); c != nil {
			totCr.Add(totCr, c)
		}
	}
	for _, a := range adjs {
		if d := platform.NumericToRat(a.DebitAmount); d != nil {
			totDr.Add(totDr, d)
		}
		if c := platform.NumericToRat(a.CreditAmount); c != nil {
			totCr.Add(totCr, c)
		}
	}

	s.mu.RLock()
	gen := s.gen[runID]
	s.mu.RUnlock()
	if gen.NCIEquity == "" {
		gen.NCIEquity = "0.0000"
	}
	if gen.NCINetInc == "" {
		gen.NCINetInc = "0.0000"
	}
	if gen.CTAReserve == "" {
		gen.CTAReserve = "0.0000"
	}
	if gen.ICElimTotal == "" {
		gen.ICElimTotal = "0.0000"
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"runId":            run.ID,
		"status":           run.Status,
		"currency":         "IDR",
		"eliminationCount": len(elims),
		"adjustmentCount":  len(adjs),
		"balanceSheet": map[string]any{
			"totalEliminationsDebit":  totDr.FloatString(4),
			"totalEliminationsCredit": totCr.FloatString(4),
			"icEliminationTotal":      gen.ICElimTotal,
			"nciEquity":               gen.NCIEquity,
			"ctaReserve":              gen.CTAReserve,
			"isBalanced":              totDr.Cmp(totCr) == 0,
		},
		"incomeStatement": map[string]any{
			"nciNetIncome": gen.NCINetInc,
			"status":       run.Status,
		},
	})
}
