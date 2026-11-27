package httpapi

import (
	"net/http"
	"strings"

	"github.com/sinergiinvestasiproperti/erp/services/crm-service/internal/crmdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

func (s *Server) listLeads(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var st *string
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
		st = &v
	}
	rows, total, err := s.store.ListLeads(r.Context(), scope, st, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]leadDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toLeadDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createLead(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		Code           string  `json:"code"`
		Name           string  `json:"name"`
		Source         *string `json:"source"`
		EstimatedValue *string `json:"estimatedValue"`
		CustomerRef    *string `json:"customerRef"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code and name are required"))
		return
	}
	ev, err := parseOptNonNegDecimal(req.EstimatedValue, "estimatedValue")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	custRef, err := parseOptUUID(req.CustomerRef, "customerRef")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := s.store.CreateLead(r.Context(), scope, crmdb.CreateLeadParams{
		OrganizationID: scope[0], Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name),
		Source: req.Source, EstimatedValue: ev, CustomerRef: custRef, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "New", "rowVersion": out.RowVersion})
}

func (s *Server) getLead(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	row, err := s.store.GetLead(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, toLeadDTO(row))
}

func (s *Server) updateLead(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		Name           string  `json:"name"`
		Source         *string `json:"source"`
		EstimatedValue *string `json:"estimatedValue"`
		CustomerRef    *string `json:"customerRef"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name is required"))
		return
	}
	ev, err := parseOptNonNegDecimal(req.EstimatedValue, "estimatedValue")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	custRef, err := parseOptUUID(req.CustomerRef, "customerRef")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.store.UpdateLead(r.Context(), scope, crmdb.UpdateLeadParams{
		ID: id, Name: strings.TrimSpace(req.Name), Source: req.Source, EstimatedValue: ev,
		CustomerRef: custRef, UpdatedBy: &user, RowVersion: ver,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "rowVersion": next})
}

func (s *Server) transitionLead(from, to string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope, user, ok := requireWrite(w, r)
		if !ok {
			return
		}
		id, err := httpx.ParsePathUUID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		ver, err := httpx.IfMatch(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req struct {
			CustomerRef *string `json:"customerRef"`
		}
		idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
		if handled {
			return
		}
		defer release()
		custRef, err := parseOptUUID(req.CustomerRef, "customerRef")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		next, oppID, err := s.store.TransitionLead(r.Context(), scope, id, scope[0], user, httpx.CorrelationUUID(r), ver, from, to, custRef)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		resp := map[string]any{"id": id, "status": to, "rowVersion": next}
		if oppID != nil {
			resp["opportunityId"] = *oppID
		}
		httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, resp)
	}
}

func (s *Server) loseLead(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct{}
	idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	next, err := s.store.LoseLead(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Lost", "rowVersion": next})
}

func (s *Server) deleteLead(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.store.DeleteLead(r.Context(), scope, id, user, ver); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listOpportunities(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var st *string
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
		st = &v
	}
	rows, total, err := s.store.ListOpportunities(r.Context(), scope, st, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]opportunityDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toOpportunityDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createOpportunity(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		Code              string  `json:"code"`
		LeadID            *string `json:"leadId"`
		CustomerRef       *string `json:"customerRef"`
		Stage             string  `json:"stage"`
		ProbabilityPct    *string `json:"probabilityPct"`
		EstimatedValue    *string `json:"estimatedValue"`
		ExpectedCloseDate *string `json:"expectedCloseDate"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Stage) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code and stage are required"))
		return
	}
	leadID, err := parseOptUUID(req.LeadID, "leadId")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	custRef, err := parseOptUUID(req.CustomerRef, "customerRef")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	prob, err := parseOptProb(req.ProbabilityPct)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ev, err := parseOptNonNegDecimal(req.EstimatedValue, "estimatedValue")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ecd, err := parseOptDate(req.ExpectedCloseDate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := s.store.CreateOpportunity(r.Context(), scope, crmdb.CreateOpportunityParams{
		OrganizationID: scope[0], Code: strings.TrimSpace(req.Code), LeadID: leadID,
		CustomerRef: custRef, Stage: strings.TrimSpace(req.Stage), ProbabilityPct: prob,
		EstimatedValue: ev, ExpectedCloseDate: ecd, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "Open", "rowVersion": out.RowVersion})
}

func (s *Server) getOpportunity(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	row, err := s.store.GetOpportunity(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, toOpportunityDTO(row))
}

func (s *Server) updateOpportunity(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		Stage             string  `json:"stage"`
		ProbabilityPct    *string `json:"probabilityPct"`
		EstimatedValue    *string `json:"estimatedValue"`
		ExpectedCloseDate *string `json:"expectedCloseDate"`
		CustomerRef       *string `json:"customerRef"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Stage) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("stage is required"))
		return
	}
	prob, err := parseOptProb(req.ProbabilityPct)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ev, err := parseOptNonNegDecimal(req.EstimatedValue, "estimatedValue")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ecd, err := parseOptDate(req.ExpectedCloseDate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	custRef, err := parseOptUUID(req.CustomerRef, "customerRef")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.store.UpdateOpportunity(r.Context(), scope, crmdb.UpdateOpportunityParams{
		ID: id, Stage: strings.TrimSpace(req.Stage), ProbabilityPct: prob, EstimatedValue: ev,
		ExpectedCloseDate: ecd, CustomerRef: custRef, UpdatedBy: &user, RowVersion: ver,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "rowVersion": next})
}

func (s *Server) winOpportunity(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		CurrencyCode string `json:"currencyCode"`
	}
	idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	ccy := strings.ToUpper(strings.TrimSpace(req.CurrencyCode))
	if ccy != "" && len(ccy) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("currencyCode must be 3-letter ISO code"))
		return
	}
	next, err := s.store.WinOpportunityAndEmit(r.Context(), scope, id, scope[0], user, httpx.CorrelationUUID(r), ver, ccy)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Won", "rowVersion": next})
}

func (s *Server) loseOpportunity(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct{}
	idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	next, err := s.store.LoseOpportunity(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Lost", "rowVersion": next})
}
