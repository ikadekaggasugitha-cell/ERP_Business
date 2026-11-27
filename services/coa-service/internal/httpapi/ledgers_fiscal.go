package httpapi

import (
	"net/http"
	"strings"

	"github.com/sinergiinvestasiproperti/erp/services/coa-service/internal/coadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func (s *Server) createFinOrg(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req finOrgReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan name wajib diisi"))
		return
	}
	orgRef, err := httpx.ParseUUIDParam("organizationRefId", req.OrganizationRefID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	coaID, err := httpx.ParseUUIDParam("chartOfAccountsId", req.ChartOfAccountsID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	calID, err := httpx.ParseUUIDParam("fiscalCalendarId", req.FiscalCalendarID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ccy := strings.ToUpper(deflt(req.FunctionalCurrencyCode, "IDR"))
	if len(ccy) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("functionalCurrencyCode harus 3 huruf ISO"))
		return
	}
	created, err := s.repo.CreateFinancialOrg(r.Context(), scope, coadb.CreateFinancialOrganizationParams{
		OrganizationID:         scope[0],
		Code:                   strings.TrimSpace(req.Code),
		Name:                   strings.TrimSpace(req.Name),
		OrganizationRefID:      orgRef,
		FunctionalCurrencyCode: ccy,
		ChartOfAccountsID:      coaID,
		FiscalCalendarID:       calID,
		CreatedBy:              user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode financial organization sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID.String(), "rowVersion": created.RowVersion})
}

func (s *Server) listFinOrgs(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListFinancialOrgsWithCount(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":                     row.ID.String(),
			"code":                   row.Code,
			"name":                   row.Name,
			"organizationRefId":      row.OrganizationRefID.String(),
			"functionalCurrencyCode": row.FunctionalCurrencyCode,
			"chartOfAccountsId":      row.ChartOfAccountsID.String(),
			"fiscalCalendarId":       row.FiscalCalendarID.String(),
			"rowVersion":             row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createLedger(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req ledgerReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code wajib diisi"))
		return
	}
	finOrgID, err := httpx.ParseUUIDParam("financialOrganizationId", req.FinancialOrganizationID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	lt := deflt(req.LedgerType, "Primary")
	if lt != "Primary" && lt != "Statutory" && lt != "Management" && lt != "Consolidation" {
		httpx.WriteError(w, r, httpx.BadRequest("ledgerType harus Primary, Statutory, Management, atau Consolidation"))
		return
	}
	ccy := strings.ToUpper(deflt(req.CurrencyCode, "IDR"))
	if len(ccy) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("currencyCode harus 3 huruf ISO"))
		return
	}
	created, err := s.repo.CreateLedger(r.Context(), scope, coadb.CreateLedgerParams{
		OrganizationID:          scope[0],
		FinancialOrganizationID: finOrgID,
		Code:                    strings.TrimSpace(req.Code),
		LedgerType:              lt,
		CurrencyCode:            ccy,
		CreatedBy:               user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode ledger sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID.String(), "rowVersion": created.RowVersion})
}

func (s *Server) listLedgers(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListLedgersWithCount(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":                      row.ID.String(),
			"financialOrganizationId": row.FinancialOrganizationID.String(),
			"code":                    row.Code,
			"ledgerType":              row.LedgerType,
			"currencyCode":            row.CurrencyCode,
			"rowVersion":              row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createCalendar(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req calendarReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan name wajib diisi"))
		return
	}
	pt := deflt(req.PeriodType, "Monthly")
	if pt != "Monthly" && pt != "Quarterly" {
		httpx.WriteError(w, r, httpx.BadRequest("periodType harus Monthly atau Quarterly"))
		return
	}
	created, err := s.repo.CreateFiscalCalendar(r.Context(), scope, coadb.CreateFiscalCalendarParams{
		OrganizationID: scope[0],
		Code:           strings.TrimSpace(req.Code),
		Name:           strings.TrimSpace(req.Name),
		PeriodType:     pt,
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode fiscal calendar sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID.String(), "rowVersion": created.RowVersion})
}

func (s *Server) listCalendars(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListFiscalCalendarsWithCount(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{"id": row.ID.String(), "code": row.Code, "name": row.Name, "periodType": row.PeriodType, "rowVersion": row.RowVersion})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createYear(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req yearReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if req.YearNo <= 0 || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("yearNo (> 0) dan name wajib diisi"))
		return
	}
	calID, err := httpx.ParseUUIDParam("fiscalCalendarId", req.FiscalCalendarID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	start, err := platform.ParseDate(req.StartDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("startDate harus YYYY-MM-DD"))
		return
	}
	end, err := platform.ParseDate(req.EndDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("endDate harus YYYY-MM-DD"))
		return
	}
	if end.Time.Before(start.Time) {
		httpx.WriteError(w, r, httpx.BadRequest("endDate tidak boleh sebelum startDate"))
		return
	}
	created, err := s.repo.CreateFiscalYear(r.Context(), scope, coadb.CreateFiscalYearParams{
		OrganizationID:   scope[0],
		FiscalCalendarID: calID,
		YearNo:           req.YearNo,
		Name:             strings.TrimSpace(req.Name),
		StartDate:        start,
		EndDate:          end,
		CreatedBy:        user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "fiscal year sudah ada"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID.String(), "status": created.Status, "rowVersion": created.RowVersion})
}

func (s *Server) listYears(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListFiscalYearsWithCount(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":               row.ID.String(),
			"fiscalCalendarId": row.FiscalCalendarID.String(),
			"yearNo":           row.YearNo,
			"name":             row.Name,
			"startDate":        platform.FormatDate(row.StartDate),
			"endDate":          platform.FormatDate(row.EndDate),
			"status":           row.Status,
			"rowVersion":       row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createPeriod(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req periodReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if req.PeriodNo <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("periodNo harus > 0"))
		return
	}
	yearID, err := httpx.ParseUUIDParam("fiscalYearId", req.FiscalYearID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	start, err := platform.ParseDate(req.StartDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("startDate harus YYYY-MM-DD"))
		return
	}
	end, err := platform.ParseDate(req.EndDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("endDate harus YYYY-MM-DD"))
		return
	}
	if end.Time.Before(start.Time) {
		httpx.WriteError(w, r, httpx.BadRequest("endDate tidak boleh sebelum startDate"))
		return
	}
	created, err := s.repo.CreateFiscalPeriod(r.Context(), scope, coadb.CreateFiscalPeriodParams{
		OrganizationID: scope[0],
		FiscalYearID:   yearID,
		PeriodNo:       req.PeriodNo,
		StartDate:      start,
		EndDate:        end,
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "fiscal period sudah ada"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{"id": created.ID.String(), "status": created.Status, "rowVersion": created.RowVersion})
}

func (s *Server) listPeriods(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	fy := strings.TrimSpace(r.URL.Query().Get("fiscalYearId"))
	if fy != "" {
		u, err := httpx.ParseUUIDParam("fiscalYearId", fy)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		fy = u.String()
	}
	rows, total, err := s.repo.ListFiscalPeriodsWithCount(r.Context(), scope, fy, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":           row.ID.String(),
			"fiscalYearId": row.FiscalYearID.String(),
			"periodNo":     row.PeriodNo,
			"startDate":    platform.FormatDate(row.StartDate),
			"endDate":      platform.FormatDate(row.EndDate),
			"status":       row.Status,
			"rowVersion":   row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) transitionPeriod(expectedStatus, newStatus string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope, user, ok := requireWrite(w, r)
		if !ok {
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
		p, nextVer, err := s.repo.TransitionFiscalPeriodAndEmit(r.Context(), scope, id, rowVer, expectedStatus, newStatus, user, httpx.CorrelationUUID(r))
		if err != nil {
			httpx.WriteError(w, r, httpx.MapDBError(err, ""))
			return
		}
		httpx.WriteIDResponse(w, http.StatusOK, nextVer, map[string]any{
			"id":           p.ID.String(),
			"fiscalYearId": p.FiscalYearID.String(),
			"periodNo":     p.PeriodNo,
			"status":       p.Status,
			"rowVersion":   nextVer,
		})
	}
}
func (s *Server) updateLedger(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
	var req struct {
		Code         string `json:"code"`
		LedgerType   string `json:"ledgerType"`
		CurrencyCode string `json:"currencyCode"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code wajib diisi"))
		return
	}
	ledgerType := deflt(req.LedgerType, "Primary")
	currency := strings.ToUpper(deflt(req.CurrencyCode, "IDR"))
	if len(currency) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("currencyCode harus 3 karakter ISO-4217"))
		return
	}

	nextVer, err := s.repo.UpdateLedger(r.Context(), scope, coadb.UpdateLedgerParams{
		ID:           id,
		RowVersion:   rowVer,
		Code:         code,
		LedgerType:   ledgerType,
		CurrencyCode: currency,
		UpdatedBy:    &user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, nextVer, map[string]any{
		"id":           id.String(),
		"code":         code,
		"ledgerType":   ledgerType,
		"currencyCode": currency,
		"rowVersion":   nextVer,
	})
}

func (s *Server) updateFiscalYear(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
	var req struct {
		Name      string `json:"name"`
		StartDate string `json:"startDate"`
		EndDate   string `json:"endDate"`
		Status    string `json:"status"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name wajib diisi"))
		return
	}
	startDate, err := platform.ParseDate(req.StartDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("startDate harus YYYY-MM-DD"))
		return
	}
	endDate, err := platform.ParseDate(req.EndDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("endDate harus YYYY-MM-DD"))
		return
	}
	if endDate.Time.Before(startDate.Time) {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_DATE_RANGE", "endDate tidak boleh lebih awal dari startDate"))
		return
	}
	status := deflt(req.Status, "Open")
	if status != "Open" && status != "Closed" && status != "Locked" {
		httpx.WriteError(w, r, httpx.BadRequest("status harus Open, Closed, atau Locked"))
		return
	}

	nextVer, err := s.repo.UpdateFiscalYear(r.Context(), scope, coadb.UpdateFiscalYearParams{
		ID:         id,
		RowVersion: rowVer,
		Name:       name,
		StartDate:  startDate,
		EndDate:    endDate,
		Status:     status,
		UpdatedBy:  &user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, nextVer, map[string]any{
		"id":         id.String(),
		"name":       name,
		"startDate":  platform.FormatDate(startDate),
		"endDate":    platform.FormatDate(endDate),
		"status":     status,
		"rowVersion": nextVer,
	})
}
