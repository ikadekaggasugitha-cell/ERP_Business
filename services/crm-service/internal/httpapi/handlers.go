package httpapi

import (
	"bytes"
	"context"
	"io"
	"math/big"
	"net/http"
	"strings"

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
