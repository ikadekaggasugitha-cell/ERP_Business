package httpapi

import (
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/gl-service/internal/gldb"
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
	if err := httpx.RequirePermission(r, "gl.create"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	var req createReq
	reqHash, err := httpx.DecodeJSONIdempotent(w, r, &req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	sk := scopeKey(scope)
	if s.idem.ReplayWithHash(w, sk, idemKey, reqHash) {
		return
	}
	switch s.idem.ReserveWithHash(sk, idemKey, reqHash) {
	case httpx.ReserveAcquired:
	case httpx.ReserveCompleted:
		if s.idem.ReplayWithHash(w, sk, idemKey, reqHash) {
			return
		}
	case httpx.ReserveHashMismatch:
		httpx.WriteError(w, r, httpx.Conflict("IDEMPOTENCY_KEY_REUSED", "Idempotency-Key reused with different request payload"))
		return
	default:
		httpx.WriteError(w, r, httpx.Conflict("REQUEST_IN_PROGRESS", "Idempotency-Key sedang diproses"))
		return
	}
	defer s.idem.Unreserve(sk, idemKey)

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

	var srcRef *string
	if v := strings.TrimSpace(req.SourceDocumentRef); v != "" {
		srcRef = &v
	}

	entry := gldb.CreateJournalEntryParams{
		OrganizationID:          scope[0],
		JournalNo:               strings.TrimSpace(req.JournalNo),
		FinancialOrganizationID: finOrg,
		LedgerID:                ledger,
		FiscalPeriodID:          period,
		PostingDate:             pd,
		JournalSource:           deflt(req.JournalSource, "Manual"),
		SourceDocumentRef:       srcRef,
		CurrencyCode:            strings.ToUpper(deflt(req.CurrencyCode, "IDR")),
		ExchangeRate:            exRate,
		CorrelationID:           httpx.CorrelationUUID(r),
		CreatedBy:               user,
	}

	created, err := s.repo.CreateEntryWithLines(r.Context(), scope, entry, lines)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "journalNo sudah ada di tahun fiskal ini"))
		return
	}
	totDr, totCr := sumLineParams(lines)
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
		Status:                  created.Status,
		RowVersion:              created.RowVersion,
	}
	body, _ := marshalJSON(dto)
	s.idem.SaveWithHash(sk, idemKey, reqHash, http.StatusCreated, body)
	httpx.WriteIDResponse(w, http.StatusCreated, created.RowVersion, dto)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "gl.read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	e, lines, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	lineDTOs := make([]lineDTO, 0, len(lines))
	for _, l := range lines {
		desc := ""
		if l.Description != nil {
			desc = *l.Description
		}
		lineDTOs = append(lineDTOs, lineDTO{
			ID:               l.ID.String(),
			LineNo:           l.LineNo,
			AccountID:        l.AccountID.String(),
			DebitAmount:      platform.FormatNumeric(l.DebitAmount, 4),
			CreditAmount:     platform.FormatNumeric(l.CreditAmount, 4),
			FunctionalAmount: platform.FormatNumeric(l.FunctionalAmount, 4),
			Description:      desc,
		})
	}
	httpx.WriteIDResponse(w, http.StatusOK, e.RowVersion, toEntryDTO(e, lineDTOs))
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "gl.read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListWithCount(r.Context(), scope, gldb.ListJournalEntriesParams{Limit: limit, Offset: offset})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]entryDTO, 0, len(rows))
	for _, e := range rows {
		src := ""
		if e.SourceDocumentRef != nil {
			src = *e.SourceDocumentRef
		}
		rev := ""
		if e.ReversedEntryID != nil {
			rev = e.ReversedEntryID.String()
		}
		exRateStr := formatExchangeRate(e.ExchangeRate)
		totDrStr := platform.FormatNumeric(e.TotalDebit, 4)
		totCrStr := platform.FormatNumeric(e.TotalCredit, 4)
		out = append(out, entryDTO{
			ID:                      e.ID.String(),
			JournalNo:               e.JournalNo,
			FinancialOrganizationID: e.FinancialOrganizationID.String(),
			LedgerID:                e.LedgerID.String(),
			FiscalPeriodID:          e.FiscalPeriodID.String(),
			PostingDate:             dateStr(e.PostingDate),
			JournalSource:           e.JournalSource,
			CurrencyCode:            e.CurrencyCode,
			ExchangeRate:            exRateStr,
			TotalDebit:              totDrStr,
			TotalCredit:             totCrStr,
			SourceDocumentRef:       src,
			ReversedEntryID:         rev,
			Status:                  e.Status,
			RowVersion:              e.RowVersion,
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
	if err := httpx.RequirePermission(r, "gl.update"); err != nil {
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
	e, _, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	exRate := e.ExchangeRate
	rateChanged := false
	if strings.TrimSpace(req.ExchangeRate) != "" {
		exRate, err = platform.ParseOptionalNumeric(req.ExchangeRate)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("exchangeRate bukan angka desimal valid"))
			return
		}
		if rVal := platform.NumericToRat(exRate); rVal == nil || rVal.Sign() <= 0 {
			httpx.WriteError(w, r, httpx.BadRequest("exchangeRate harus > 0"))
			return
		}
		rateChanged = true
	}
	var lines []gldb.CreateJournalLineParams
	if len(req.Lines) > 0 {
		if rateChanged {
			for i := range req.Lines {
				req.Lines[i].FunctionalAmount = ""
			}
		}
		lines, err = buildLineParams(e.OrganizationID, user, exRate, req.Lines)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
	}
	srcRef := e.SourceDocumentRef
	if v := strings.TrimSpace(req.SourceDocumentRef); v != "" {
		srcRef = &v
	}
	curr := strings.ToUpper(deflt(req.CurrencyCode, e.CurrencyCode))
	n, err := s.repo.UpdateDraftEntry(r.Context(), scope, gldb.UpdateJournalEntryDraftParams{
		ID:                e.ID,
		PostingDate:       e.PostingDate,
		CurrencyCode:      curr,
		ExchangeRate:      exRate,
		SourceDocumentRef: srcRef,
		UpdatedBy:         &user,
		RowVersion:        rowVer,
	}, lines)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Draft"))
		return
	}
	updated, updatedLines, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	lineDTOs := make([]lineDTO, 0, len(updatedLines))
	for _, l := range updatedLines {
		lineDTOs = append(lineDTOs, lineDTO{
			ID:               l.ID.String(),
			LineNo:           l.LineNo,
			AccountID:        l.AccountID.String(),
			DebitAmount:      platform.FormatNumeric(l.DebitAmount, 4),
			CreditAmount:     platform.FormatNumeric(l.CreditAmount, 4),
			FunctionalAmount: platform.FormatNumeric(l.FunctionalAmount, 4),
		})
	}
	httpx.WriteIDResponse(w, http.StatusOK, updated.RowVersion, toEntryDTO(updated, lineDTOs))
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
	if err := httpx.RequirePermission(r, "gl.update"); err != nil {
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
	e, _, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	n, err := s.repo.SoftDeleteDraft(r.Context(), scope, gldb.SoftDeleteJournalEntryParams{
		ID:          e.ID,
		PostingDate: e.PostingDate,
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

func (s *Server) post(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "gl.approve"); err != nil {
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
	e, _, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	// BR-556: exact rational comparison + >=2 lines + positive total.
	tot, err := s.repo.Totals(r.Context(), scope, gldb.JournalEntryTotalsParams{JournalEntryID: e.ID, PostingDate: e.PostingDate})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	dr := platform.NumericToRat(tot.TotalDebit)
	cr := platform.NumericToRat(tot.TotalCredit)
	fDr := platform.NumericToRat(tot.TotalFunctionalDebit)
	fCr := platform.NumericToRat(tot.TotalFunctionalCredit)
	if tot.LineCount < 2 || dr.Sign() <= 0 || dr.Cmp(cr) != 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("GL_JOURNAL_UNBALANCED", "total debit tidak sama dengan total kredit",
			httpx.Detail{Field: "lines", Issue: "debit=" + dr.FloatString(4) + ", credit=" + cr.FloatString(4)}))
		return
	}
	if (fDr.Sign() > 0 || fCr.Sign() > 0) && fDr.Cmp(fCr) != 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("GL_JOURNAL_UNBALANCED", "total functional debit tidak sama dengan total functional kredit",
			httpx.Detail{Field: "lines", Issue: "functionalDebit=" + fDr.FloatString(4) + ", functionalCredit=" + fCr.FloatString(4)}))
		return
	}

	n, err := s.repo.PostAndEmit(r.Context(), scope, e, rowVer, user)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Draft"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, rowVer+1, entryDTO{
		ID:           id.String(),
		JournalNo:    e.JournalNo,
		PostingDate:  dateStr(e.PostingDate),
		CurrencyCode: e.CurrencyCode,
		ExchangeRate: formatExchangeRate(e.ExchangeRate),
		TotalDebit:   dr.FloatString(4),
		TotalCredit:  cr.FloatString(4),
		Status:       "Posted",
		RowVersion:   rowVer + 1,
	})
}

func (s *Server) reverse(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "gl.approve"); err != nil {
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
	orig, origLines, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	revRow, n, err := s.repo.ReverseEntryAndEmit(r.Context(), scope, orig, rowVer, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "reversal journal sudah pernah dibuat"))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status bukan Posted"))
		return
	}
	revDr := new(big.Rat)
	revCr := new(big.Rat)
	for _, l := range origLines {
		revDr.Add(revDr, platform.NumericToRat(l.CreditAmount))
		revCr.Add(revCr, platform.NumericToRat(l.DebitAmount))
	}
	revNo := "REV-" + orig.JournalNo
	if len(revNo) > 50 {
		revNo = revNo[:50]
	}
	httpx.WriteIDResponse(w, http.StatusCreated, revRow.RowVersion, entryDTO{
		ID:                      revRow.ID.String(),
		JournalNo:               revNo,
		FinancialOrganizationID: orig.FinancialOrganizationID.String(),
		LedgerID:                orig.LedgerID.String(),
		FiscalPeriodID:          orig.FiscalPeriodID.String(),
		PostingDate:             dateStr(revRow.PostingDate),
		JournalSource:           "Reversal",
		CurrencyCode:            orig.CurrencyCode,
		ExchangeRate:            formatExchangeRate(orig.ExchangeRate),
		TotalDebit:              revDr.FloatString(4),
		TotalCredit:             revCr.FloatString(4),
		SourceDocumentRef:       orig.JournalNo,
		ReversedEntryID:         orig.ID.String(),
		Status:                  revRow.Status,
		RowVersion:              revRow.RowVersion,
	})
}

func (s *Server) trialBalance(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "gl.read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var ledgerID, periodID uuid.UUID
	if v := strings.TrimSpace(r.URL.Query().Get("ledgerId")); v != "" {
		if ledgerID, err = httpx.ParseUUIDParam("ledgerId", v); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("fiscalPeriodId")); v != "" {
		if periodID, err = httpx.ParseUUIDParam("fiscalPeriodId", v); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListAccountBalances(r.Context(), scope, ledgerID, periodID, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]balanceDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, balanceDTO{
			ID:             row.ID.String(),
			LedgerID:       row.LedgerID.String(),
			AccountID:      row.AccountID.String(),
			FiscalPeriodID: row.FiscalPeriodID.String(),
			OpeningBalance: platform.FormatNumeric(row.OpeningBalance, 4),
			PeriodDebit:    platform.FormatNumeric(row.PeriodDebit, 4),
			PeriodCredit:   platform.FormatNumeric(row.PeriodCredit, 4),
			ClosingBalance: platform.FormatNumeric(row.ClosingBalance, 4),
			RowVersion:     row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

// ---- helpers ---------------------------------------------------------------
