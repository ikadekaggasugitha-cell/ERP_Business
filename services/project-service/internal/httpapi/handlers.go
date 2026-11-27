package httpapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/project-service/internal/prjdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type ProjectStore interface {
	CreateProject(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateProjectParams) (prjdb.CreateProjectRow, error)
	GetProject(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (prjdb.PrjProject, error)
	ListProjects(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]prjdb.PrjProject, int64, error)
	UpdateProject(ctx context.Context, scope []uuid.UUID, arg prjdb.UpdateProjectParams) (int32, error)
	TransitionProject(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32, fromStatus, toStatus string) (int32, error)
	DeleteProject(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error
	CreateWBSElement(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateWBSElementParams) (prjdb.CreateWBSElementRow, error)
	ListWBSElements(ctx context.Context, scope []uuid.UUID, projectID uuid.UUID) ([]prjdb.PrjWbsElement, error)
	UpdateWBSElement(ctx context.Context, scope []uuid.UUID, arg prjdb.UpdateWBSElementParams) (int32, error)
	DeleteWBSElement(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error
	CreateBudget(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateProjectBudgetParams) (prjdb.CreateProjectBudgetRow, error)
	ListBudgets(ctx context.Context, scope []uuid.UUID, projectID uuid.UUID) ([]prjdb.PrjProjectBudget, error)
	ApproveBudget(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error)
	CreateCost(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateProjectCostParams) (prjdb.CreateProjectCostRow, error)
	ListCosts(ctx context.Context, scope []uuid.UUID, projectID uuid.UUID) ([]prjdb.PrjProjectCost, error)
	UpsertProgress(ctx context.Context, scope []uuid.UUID, arg prjdb.UpsertProjectProgressParams) (prjdb.UpsertProjectProgressRow, error)
	ListProgress(ctx context.Context, scope []uuid.UUID, wbsID uuid.UUID) ([]prjdb.PrjProjectProgress, error)
	CreateBilling(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateProjectBillingParams) (prjdb.CreateProjectBillingRow, error)
	ListBillings(ctx context.Context, scope []uuid.UUID, projectID *uuid.UUID, limit, offset int32) ([]prjdb.PrjProjectBilling, int64, error)
	ApproveBillingAndEmit(ctx context.Context, scope []uuid.UUID, id, org, actor, corr uuid.UUID, version int32, customerID *uuid.UUID, arRef *string) (int32, string, error)
	CreateChangeOrder(ctx context.Context, scope []uuid.UUID, arg prjdb.CreateChangeOrderParams) (prjdb.CreateChangeOrderRow, error)
	ListChangeOrders(ctx context.Context, scope []uuid.UUID, projectID uuid.UUID) ([]prjdb.PrjChangeOrder, error)
	DecideChangeOrder(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32, status string) (int32, error)
}

type Server struct {
	store ProjectStore
	idem  *httpx.IdemStore
}

func NewServer(store ProjectStore) *Server {
	return &Server{store: store, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /projects", s.listProjects)
	mux.HandleFunc("POST /projects", s.createProject)
	mux.HandleFunc("GET /projects/{id}", s.getProject)
	mux.HandleFunc("PUT /projects/{id}", s.updateProject)
	mux.HandleFunc("DELETE /projects/{id}", s.deleteProject)
	mux.HandleFunc("POST /projects/{id}/activate", s.transitionProject("Draft", "Active"))
	mux.HandleFunc("POST /projects/{id}/hold", s.transitionProject("Active", "OnHold"))
	mux.HandleFunc("POST /projects/{id}/close", s.transitionProject("Active", "Closed"))

	mux.HandleFunc("GET /projects/{projectId}/wbs-elements", s.listWBS)
	mux.HandleFunc("POST /wbs-elements", s.createWBS)
	mux.HandleFunc("PUT /wbs-elements/{id}", s.updateWBS)
	mux.HandleFunc("DELETE /wbs-elements/{id}", s.deleteWBS)

	mux.HandleFunc("GET /projects/{projectId}/budgets", s.listBudgets)
	mux.HandleFunc("POST /projects/{projectId}/budgets", s.createBudget)
	mux.HandleFunc("POST /project-budgets/{id}/approve", s.approveBudget)

	mux.HandleFunc("GET /projects/{projectId}/costs", s.listCosts)
	mux.HandleFunc("POST /projects/{projectId}/costs", s.createCost)

	mux.HandleFunc("GET /wbs-elements/{wbsId}/progress", s.listProgress)
	mux.HandleFunc("POST /wbs-elements/{wbsId}/progress", s.upsertProgress)

	mux.HandleFunc("GET /project-billings", s.listBillings)
	mux.HandleFunc("POST /project-billings", s.createBilling)
	mux.HandleFunc("POST /project-billings/{id}/approve", s.approveBilling)

	mux.HandleFunc("GET /projects/{projectId}/change-orders", s.listChangeOrders)
	mux.HandleFunc("POST /projects/{projectId}/change-orders", s.createChangeOrder)
	mux.HandleFunc("POST /change-orders/{id}/approve", s.decideChangeOrder("Approved"))
	mux.HandleFunc("POST /change-orders/{id}/reject", s.decideChangeOrder("Rejected"))
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
	if err := httpx.RequirePermission(r, "project:read"); err != nil {
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
	if err := httpx.RequirePermission(r, "project:write"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
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

func parsePositiveNumeric(raw, field string) (pgtype.Numeric, error) {
	num, err := platform.ParseNumeric(strings.TrimSpace(raw))
	if err != nil || platform.NumericToRat(num) == nil || platform.NumericToRat(num).Sign() <= 0 {
		return pgtype.Numeric{}, httpx.BadRequest(field + " must be a positive decimal string")
	}
	return num, nil
}
