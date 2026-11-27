package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/crm-service/internal/crmdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

func (s *Server) listSegments(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.store.ListSegments(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]segmentDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toSegmentDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createSegment(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		Code     string          `json:"code"`
		Name     string          `json:"name"`
		Criteria json.RawMessage `json:"criteria"`
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
	out, err := s.store.CreateSegment(r.Context(), scope, crmdb.CreateCustomerSegmentParams{
		OrganizationID: scope[0], Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name),
		Criteria: req.Criteria, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) updateSegment(w http.ResponseWriter, r *http.Request) {
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
		Name     string          `json:"name"`
		Criteria json.RawMessage `json:"criteria"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name is required"))
		return
	}
	next, err := s.store.UpdateSegment(r.Context(), scope, crmdb.UpdateCustomerSegmentParams{
		ID: id, Name: strings.TrimSpace(req.Name), Criteria: req.Criteria, UpdatedBy: &user, RowVersion: ver,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "rowVersion": next})
}

func (s *Server) deleteSegment(w http.ResponseWriter, r *http.Request) {
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
	if err := s.store.DeleteSegment(r.Context(), scope, id, user, ver); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listInteractions(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.store.ListInteractions(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]interactionDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toInteractionDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createInteraction(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		CustomerRef     *string `json:"customerRef"`
		LeadID          *string `json:"leadId"`
		InteractionType string  `json:"interactionType"`
		OccurredAt      *string `json:"occurredAt"`
		Notes           *string `json:"notes"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	switch req.InteractionType {
	case "Call", "Meeting", "Email", "Note":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("interactionType must be Call, Meeting, Email, or Note"))
		return
	}
	custRef, err := parseOptUUID(req.CustomerRef, "customerRef")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	leadID, err := parseOptUUID(req.LeadID, "leadId")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	occ := time.Now().UTC()
	if req.OccurredAt != nil && strings.TrimSpace(*req.OccurredAt) != "" {
		t, e := time.Parse(time.RFC3339, strings.TrimSpace(*req.OccurredAt))
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("occurredAt must be RFC3339 timestamp"))
			return
		}
		occ = t
	}
	out, err := s.store.CreateInteraction(r.Context(), scope, crmdb.CreateInteractionParams{
		OrganizationID: scope[0], CustomerRef: custRef, LeadID: leadID,
		InteractionType: req.InteractionType, OccurredAt: pgtype.Timestamptz{Time: occ, Valid: true},
		Notes: req.Notes, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) listSupportCases(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var st *string
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
		st = &v
	}
	rows, total, err := s.store.ListSupportCases(r.Context(), scope, st, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]supportCaseDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toSupportCaseDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createSupportCase(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		CaseNo      string  `json:"caseNo"`
		CustomerRef *string `json:"customerRef"`
		Subject     string  `json:"subject"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	if strings.TrimSpace(req.CaseNo) == "" || strings.TrimSpace(req.Subject) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("caseNo and subject are required"))
		return
	}
	custRef, err := parseOptUUID(req.CustomerRef, "customerRef")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := s.store.CreateSupportCase(r.Context(), scope, crmdb.CreateSupportCaseParams{
		OrganizationID: scope[0], CaseNo: strings.TrimSpace(req.CaseNo),
		CustomerRef: custRef, Subject: strings.TrimSpace(req.Subject), CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "Open", "rowVersion": out.RowVersion})
}

func (s *Server) getSupportCase(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	row, err := s.store.GetSupportCase(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, toSupportCaseDTO(row))
}

func (s *Server) escalateSupportCase(w http.ResponseWriter, r *http.Request) {
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
	next, err := s.store.EscalateSupportCase(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Escalated", "rowVersion": next})
}

func (s *Server) resolveSupportCase(w http.ResponseWriter, r *http.Request) {
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
		Resolution *string `json:"resolution"`
	}
	idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	next, err := s.store.ResolveSupportCase(r.Context(), scope, id, user, ver, req.Resolution)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Resolved", "rowVersion": next})
}

func (s *Server) closeSupportCase(w http.ResponseWriter, r *http.Request) {
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
	next, err := s.store.CloseSupportCase(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Closed", "rowVersion": next})
}

func (s *Server) listActivityTasks(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var st *string
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
		st = &v
	}
	rows, total, err := s.store.ListActivityTasks(r.Context(), scope, st, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]activityTaskDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toActivityTaskDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createActivityTask(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		AssigneeRef string  `json:"assigneeRef"`
		RelatedType *string `json:"relatedType"`
		RelatedRef  *string `json:"relatedRef"`
		DueDate     *string `json:"dueDate"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	assignee, err := uuid.Parse(req.AssigneeRef)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid assigneeRef is required"))
		return
	}
	relRef, err := parseOptUUID(req.RelatedRef, "relatedRef")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	due, err := parseOptDate(req.DueDate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := s.store.CreateActivityTask(r.Context(), scope, crmdb.CreateActivityTaskParams{
		OrganizationID: scope[0], AssigneeRef: assignee, RelatedType: req.RelatedType,
		RelatedRef: relRef, DueDate: due, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "Pending", "rowVersion": out.RowVersion})
}

func (s *Server) completeActivityTask(w http.ResponseWriter, r *http.Request) {
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
	next, err := s.store.CompleteActivityTask(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Done", "rowVersion": next})
}
