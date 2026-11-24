package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/crm-service/internal/crmdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type CRMStore interface {
	CreateLead(ctx context.Context, scope []uuid.UUID, arg crmdb.CreateLeadParams) (crmdb.CreateLeadRow, error)
	GetLead(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (crmdb.CrmLead, error)
	ListLeads(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]crmdb.CrmLead, int64, error)
	UpdateLead(ctx context.Context, scope []uuid.UUID, arg crmdb.UpdateLeadParams) (int32, error)
	TransitionLead(ctx context.Context, scope []uuid.UUID, id, org, actor, corr uuid.UUID, version int32, fromStatus, toStatus string, customerRef *uuid.UUID) (int32, *uuid.UUID, error)
	LoseLead(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error)
	DeleteLead(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error
	CreateOpportunity(ctx context.Context, scope []uuid.UUID, arg crmdb.CreateOpportunityParams) (crmdb.CreateOpportunityRow, error)
	GetOpportunity(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (crmdb.CrmOpportunity, error)
	ListOpportunities(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]crmdb.CrmOpportunity, int64, error)
	UpdateOpportunity(ctx context.Context, scope []uuid.UUID, arg crmdb.UpdateOpportunityParams) (int32, error)
	WinOpportunityAndEmit(ctx context.Context, scope []uuid.UUID, id, org, actor, corr uuid.UUID, version int32, currencyCode string) (int32, error)
	LoseOpportunity(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error)
	CreateSegment(ctx context.Context, scope []uuid.UUID, arg crmdb.CreateCustomerSegmentParams) (crmdb.CreateCustomerSegmentRow, error)
	ListSegments(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]crmdb.CrmCustomerSegment, int64, error)
	UpdateSegment(ctx context.Context, scope []uuid.UUID, arg crmdb.UpdateCustomerSegmentParams) (int32, error)
	DeleteSegment(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error
	CreateInteraction(ctx context.Context, scope []uuid.UUID, arg crmdb.CreateInteractionParams) (crmdb.CreateInteractionRow, error)
	ListInteractions(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]crmdb.CrmInteraction, int64, error)
	CreateSupportCase(ctx context.Context, scope []uuid.UUID, arg crmdb.CreateSupportCaseParams) (crmdb.CreateSupportCaseRow, error)
	GetSupportCase(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (crmdb.CrmSupportCase, error)
	ListSupportCases(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]crmdb.CrmSupportCase, int64, error)
	EscalateSupportCase(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error)
	ResolveSupportCase(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32, resolution *string) (int32, error)
	CloseSupportCase(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error)
	CreateActivityTask(ctx context.Context, scope []uuid.UUID, arg crmdb.CreateActivityTaskParams) (crmdb.CreateActivityTaskRow, error)
	ListActivityTasks(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]crmdb.CrmActivityTask, int64, error)
	CompleteActivityTask(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error)
}

type Server struct {
	store CRMStore
	idem  *httpx.IdemStore
}

func NewServer(store CRMStore) *Server {
	return &Server{store: store, idem: httpx.NewIdemStore()}
}

func beginIdemOptional(w http.ResponseWriter, r *http.Request, idem *httpx.IdemStore, scope []uuid.UUID, dst any) (string, string, func(), bool) {
	if r.Body == nil {
		r.Body = io.NopCloser(strings.NewReader("{}"))
	} else {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("failed to read request body"))
			return "", "", func() {}, true
		}
		if len(bytes.TrimSpace(raw)) == 0 {
			raw = []byte("{}")
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))
	}
	return httpx.BeginIdempotent(w, r, idem, scope, dst)
}

func fmtOptTS(ts pgtype.Timestamptz) *string {
	if !ts.Valid {
		return nil
	}
	s := ts.Time.UTC().Format(time.RFC3339)
	return &s
}

func fmtOptDate(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := platform.FormatDate(d)
	return &s
}

func fmtOptNumeric(n pgtype.Numeric, scale int) *string {
	if !n.Valid {
		return nil
	}
	s := platform.FormatNumeric(n, scale)
	return &s
}

func rawJSONOrEmptyObj(b []byte) json.RawMessage {
	if len(bytes.TrimSpace(b)) == 0 {
		return json.RawMessage("{}")
	}
	return json.RawMessage(b)
}

type leadDTO struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	Source         *string    `json:"source,omitempty"`
	Status         string     `json:"status"`
	EstimatedValue *string    `json:"estimatedValue,omitempty"`
	CustomerRef    *uuid.UUID `json:"customerRef,omitempty"`
	CreatedAt      *string    `json:"createdAt,omitempty"`
	UpdatedAt      *string    `json:"updatedAt,omitempty"`
	RowVersion     int32      `json:"rowVersion"`
}

func toLeadDTO(r crmdb.CrmLead) leadDTO {
	return leadDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, Code: r.Code, Name: r.Name,
		Source: r.Source, Status: r.Status, EstimatedValue: fmtOptNumeric(r.EstimatedValue, 4),
		CustomerRef: r.CustomerRef, CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt),
		RowVersion: r.RowVersion,
	}
}

type opportunityDTO struct {
	ID                uuid.UUID  `json:"id"`
	OrganizationID    uuid.UUID  `json:"organizationId"`
	Code              string     `json:"code"`
	LeadID            *uuid.UUID `json:"leadId,omitempty"`
	CustomerRef       *uuid.UUID `json:"customerRef,omitempty"`
	Stage             string     `json:"stage"`
	ProbabilityPct    *string    `json:"probabilityPct,omitempty"`
	EstimatedValue    *string    `json:"estimatedValue,omitempty"`
	ExpectedCloseDate *string    `json:"expectedCloseDate,omitempty"`
	Status            string     `json:"status"`
	CreatedAt         *string    `json:"createdAt,omitempty"`
	UpdatedAt         *string    `json:"updatedAt,omitempty"`
	RowVersion        int32      `json:"rowVersion"`
}

func toOpportunityDTO(r crmdb.CrmOpportunity) opportunityDTO {
	return opportunityDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, Code: r.Code, LeadID: r.LeadID,
		CustomerRef: r.CustomerRef, Stage: r.Stage,
		ProbabilityPct: fmtOptNumeric(r.ProbabilityPct, 2),
		EstimatedValue: fmtOptNumeric(r.EstimatedValue, 4),
		ExpectedCloseDate: fmtOptDate(r.ExpectedCloseDate), Status: r.Status,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type segmentDTO struct {
	ID             uuid.UUID       `json:"id"`
	OrganizationID uuid.UUID       `json:"organizationId"`
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	Criteria       json.RawMessage `json:"criteria"`
	IsSystem       bool            `json:"isSystem"`
	CreatedAt      *string         `json:"createdAt,omitempty"`
	UpdatedAt      *string         `json:"updatedAt,omitempty"`
	RowVersion     int32           `json:"rowVersion"`
}

func toSegmentDTO(r crmdb.CrmCustomerSegment) segmentDTO {
	return segmentDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, Code: r.Code, Name: r.Name,
		Criteria: rawJSONOrEmptyObj(r.Criteria), IsSystem: r.OrganizationID == platform.SystemOrgID,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type interactionDTO struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  uuid.UUID  `json:"organizationId"`
	CustomerRef     *uuid.UUID `json:"customerRef,omitempty"`
	LeadID          *uuid.UUID `json:"leadId,omitempty"`
	InteractionType string     `json:"interactionType"`
	OccurredAt      *string    `json:"occurredAt,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	CreatedAt       *string    `json:"createdAt,omitempty"`
	UpdatedAt       *string    `json:"updatedAt,omitempty"`
	RowVersion      int32      `json:"rowVersion"`
}

func toInteractionDTO(r crmdb.CrmInteraction) interactionDTO {
	return interactionDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, CustomerRef: r.CustomerRef, LeadID: r.LeadID,
		InteractionType: r.InteractionType, OccurredAt: fmtOptTS(r.OccurredAt), Notes: r.Notes,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type supportCaseDTO struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	CaseNo         string     `json:"caseNo"`
	CustomerRef    *uuid.UUID `json:"customerRef,omitempty"`
	Subject        string     `json:"subject"`
	Status         string     `json:"status"`
	Resolution     *string    `json:"resolution,omitempty"`
	OpenedAt       *string    `json:"openedAt,omitempty"`
	ResolvedAt     *string    `json:"resolvedAt,omitempty"`
	CreatedAt      *string    `json:"createdAt,omitempty"`
	UpdatedAt      *string    `json:"updatedAt,omitempty"`
	RowVersion     int32      `json:"rowVersion"`
}

func toSupportCaseDTO(r crmdb.CrmSupportCase) supportCaseDTO {
	return supportCaseDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, CaseNo: r.CaseNo, CustomerRef: r.CustomerRef,
		Subject: r.Subject, Status: r.Status, Resolution: r.Resolution,
		OpenedAt: fmtOptTS(r.OpenedAt), ResolvedAt: fmtOptTS(r.ResolvedAt),
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type activityTaskDTO struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	AssigneeRef    uuid.UUID  `json:"assigneeRef"`
	RelatedType    *string    `json:"relatedType,omitempty"`
	RelatedRef     *uuid.UUID `json:"relatedRef,omitempty"`
	DueDate        *string    `json:"dueDate,omitempty"`
	Status         string     `json:"status"`
	CreatedAt      *string    `json:"createdAt,omitempty"`
	UpdatedAt      *string    `json:"updatedAt,omitempty"`
	RowVersion     int32      `json:"rowVersion"`
}

func toActivityTaskDTO(r crmdb.CrmActivityTask) activityTaskDTO {
	return activityTaskDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, AssigneeRef: r.AssigneeRef,
		RelatedType: r.RelatedType, RelatedRef: r.RelatedRef, DueDate: fmtOptDate(r.DueDate),
		Status: r.Status, CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt),
		RowVersion: r.RowVersion,
	}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /leads", s.listLeads)
	mux.HandleFunc("POST /leads", s.createLead)
	mux.HandleFunc("GET /leads/{id}", s.getLead)
	mux.HandleFunc("PUT /leads/{id}", s.updateLead)
	mux.HandleFunc("DELETE /leads/{id}", s.deleteLead)
	mux.HandleFunc("POST /leads/{id}/qualify", s.transitionLead("New", "Qualified"))
	mux.HandleFunc("POST /leads/{id}/convert", s.transitionLead("Qualified", "Converted"))
	mux.HandleFunc("POST /leads/{id}/lose", s.loseLead)

	mux.HandleFunc("GET /opportunities", s.listOpportunities)
	mux.HandleFunc("POST /opportunities", s.createOpportunity)
	mux.HandleFunc("GET /opportunities/{id}", s.getOpportunity)
	mux.HandleFunc("PUT /opportunities/{id}", s.updateOpportunity)
	mux.HandleFunc("POST /opportunities/{id}/win", s.winOpportunity)
	mux.HandleFunc("POST /opportunities/{id}/lose", s.loseOpportunity)

	mux.HandleFunc("GET /customer-segments", s.listSegments)
	mux.HandleFunc("POST /customer-segments", s.createSegment)
	mux.HandleFunc("PUT /customer-segments/{id}", s.updateSegment)
	mux.HandleFunc("DELETE /customer-segments/{id}", s.deleteSegment)

	mux.HandleFunc("GET /interactions", s.listInteractions)
	mux.HandleFunc("POST /interactions", s.createInteraction)

	mux.HandleFunc("GET /support-cases", s.listSupportCases)
	mux.HandleFunc("POST /support-cases", s.createSupportCase)
	mux.HandleFunc("GET /support-cases/{id}", s.getSupportCase)
	mux.HandleFunc("POST /support-cases/{id}/escalate", s.escalateSupportCase)
	mux.HandleFunc("POST /support-cases/{id}/resolve", s.resolveSupportCase)
	mux.HandleFunc("POST /support-cases/{id}/close", s.closeSupportCase)

	mux.HandleFunc("GET /activity-tasks", s.listActivityTasks)
	mux.HandleFunc("POST /activity-tasks", s.createActivityTask)
	mux.HandleFunc("POST /activity-tasks/{id}/complete", s.completeActivityTask)
}

func requireRead(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	if err := httpx.RequirePermission(r, "crm:read"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	return scope, true
}

func requireWrite(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	if err := httpx.RequirePermission(r, "crm:write"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
}

func parseOptUUID(raw *string, field string) (*uuid.UUID, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	u, err := uuid.Parse(strings.TrimSpace(*raw))
	if err != nil {
		return nil, httpx.BadRequest("invalid " + field)
	}
	return &u, nil
}

func parseOptNonNegDecimal(raw *string, field string) (pgtype.Numeric, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return pgtype.Numeric{}, nil
	}
	num, err := platform.ParseNumeric(strings.TrimSpace(*raw))
	if err != nil || platform.NumericToRat(num) == nil || platform.NumericToRat(num).Sign() < 0 {
		return pgtype.Numeric{}, httpx.BadRequest(field + " must be a non-negative decimal string")
	}
	return num, nil
}

func parseOptProb(raw *string) (pgtype.Numeric, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return pgtype.Numeric{}, nil
	}
	num, err := platform.ParseNumeric(strings.TrimSpace(*raw))
	if err != nil {
		return pgtype.Numeric{}, httpx.BadRequest("probabilityPct must be between 0 and 100")
	}
	rat := platform.NumericToRat(num)
	if rat == nil || rat.Sign() < 0 || rat.Cmp(big.NewRat(100, 1)) > 0 {
		return pgtype.Numeric{}, httpx.BadRequest("probabilityPct must be between 0 and 100")
	}
	return num, nil
}

func parseOptDate(raw *string) (pgtype.Date, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return pgtype.Date{}, nil
	}
	d, err := platform.ParseDate(*raw)
	if err != nil {
		return pgtype.Date{}, httpx.BadRequest("date must be YYYY-MM-DD")
	}
	return d, nil
}

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
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: rows, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
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
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
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
	httpx.WriteIDResponse(w, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "New", "rowVersion": out.RowVersion})
}

func (s *Server) getLead(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	row, err := s.store.GetLead(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, row)
}

func (s *Server) updateLead(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
		id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
		if err := httpx.DecodeOptionalJSON(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
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
		httpx.WriteIDResponse(w, http.StatusOK, next, resp)
	}
}

func (s *Server) loseLead(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.store.LoseLead(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": "Lost", "rowVersion": next})
}

func (s *Server) deleteLead(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
	if err := httpx.DecodeOptionalJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
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
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": "Won", "rowVersion": next})
}

func (s *Server) loseOpportunity(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.store.LoseOpportunity(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": "Lost", "rowVersion": next})
}

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
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.store.EscalateSupportCase(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": "Escalated", "rowVersion": next})
}

func (s *Server) resolveSupportCase(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
	if err := httpx.DecodeOptionalJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.store.ResolveSupportCase(r.Context(), scope, id, user, ver, req.Resolution)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": "Resolved", "rowVersion": next})
}

func (s *Server) closeSupportCase(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.store.CloseSupportCase(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": "Closed", "rowVersion": next})
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
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.store.CompleteActivityTask(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": "Done", "rowVersion": next})
}
