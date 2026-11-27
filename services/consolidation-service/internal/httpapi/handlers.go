package httpapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/consolidation-service/internal/consdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type Store interface {
	CreateGroup(ctx context.Context, scope []uuid.UUID, arg consdb.CreateConsolidationGroupParams) (consdb.CreateConsolidationGroupRow, error)
	GetGroup(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (consdb.ConsConsolidationGroup, error)
	ListGroups(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]consdb.ConsConsolidationGroup, int64, error)
	UpdateGroup(ctx context.Context, scope []uuid.UUID, arg consdb.UpdateConsolidationGroupParams) (int32, error)
	DeleteGroup(ctx context.Context, scope []uuid.UUID, arg consdb.DeleteConsolidationGroupParams) error
	AddMember(ctx context.Context, scope []uuid.UUID, arg consdb.CreateConsolidationMemberParams) (consdb.CreateConsolidationMemberRow, error)
	ListMembers(ctx context.Context, scope []uuid.UUID, groupID uuid.UUID) ([]consdb.ConsConsolidationMember, error)
	DeleteMember(ctx context.Context, scope []uuid.UUID, arg consdb.DeleteConsolidationMemberParams) error
	CreateRule(ctx context.Context, scope []uuid.UUID, arg consdb.CreateEliminationRuleParams) (consdb.CreateEliminationRuleRow, error)
	ListRules(ctx context.Context, scope []uuid.UUID, groupID *uuid.UUID, limit, offset int32) ([]consdb.ConsEliminationRule, int64, error)
	CreateRun(ctx context.Context, scope []uuid.UUID, arg consdb.CreateConsolidationRunParams) (consdb.CreateConsolidationRunRow, error)
	GetRunDetail(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (consdb.ConsConsolidationRun, []consdb.ConsEliminationEntry, []consdb.ConsConsolidationAdjustment, error)
	ListRuns(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]consdb.ConsConsolidationRun, int64, error)
	PostEliminations(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, rowVer int32, entries []consdb.CreateEliminationEntryParams, actor uuid.UUID) (int32, error)
	PostAdjustments(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, rowVer int32, adjs []consdb.CreateConsolidationAdjustmentParams, actor uuid.UUID) (int32, error)
	FinalizeRunAndEmit(ctx context.Context, scope []uuid.UUID, runID uuid.UUID, rowVer int32, actor, corr uuid.UUID) (int32, error)
}

type generatedConsSummary struct {
	ICElimTotal string
	NCIEquity   string
	NCINetInc   string
	CTAReserve  string
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
	mu   sync.RWMutex
	gen  map[uuid.UUID]generatedConsSummary
}

func NewServer(repo Store) *Server {
	return &Server{
		repo: repo,
		idem: httpx.NewIdemStore(),
		gen:  make(map[uuid.UUID]generatedConsSummary),
	}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /consolidation-groups", s.listGroups)
	mux.HandleFunc("POST /consolidation-groups", s.createGroup)
	mux.HandleFunc("GET /consolidation-groups/{id}", s.getGroup)
	mux.HandleFunc("PUT /consolidation-groups/{id}", s.updateGroup)
	mux.HandleFunc("DELETE /consolidation-groups/{id}", s.deleteGroup)
	mux.HandleFunc("GET /consolidation-groups/{id}/members", s.listMembers)
	mux.HandleFunc("POST /consolidation-groups/{id}/members", s.addMember)
	mux.HandleFunc("DELETE /consolidation-groups/{id}/members/{memberId}", s.deleteMember)
	mux.HandleFunc("GET /elimination-rules", s.listRules)
	mux.HandleFunc("POST /elimination-rules", s.createRule)
	mux.HandleFunc("GET /consolidation-runs", s.listRuns)
	mux.HandleFunc("POST /consolidation-runs", s.createRun)
	mux.HandleFunc("GET /consolidation-runs/{id}", s.getRun)
	mux.HandleFunc("GET /consolidation-runs/{id}/statements", s.consolidatedStatements)
	mux.HandleFunc("POST /consolidation-runs/{id}/generate-eliminations", s.generateEliminations)
	mux.HandleFunc("POST /consolidation-runs/{id}/eliminate", s.postEliminations)
	mux.HandleFunc("POST /consolidation-runs/{id}/adjustments", s.postAdjustments)
	mux.HandleFunc("POST /consolidation-runs/{id}/finalize", s.finalizeRun)
}

func beginIdemOptional(w http.ResponseWriter, r *http.Request, idem *httpx.IdemStore, scope []uuid.UUID, dst any) (string, string, func(), bool) {
	if dst == nil {
		var empty map[string]any
		dst = &empty
	}
	if r.Body == nil {
		r.Body = io.NopCloser(strings.NewReader("{}"))
	} else {
		raw, err := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
		if err == nil && len(bytes.TrimSpace(raw)) == 0 {
			r.Body = io.NopCloser(strings.NewReader("{}"))
		} else {
			r.Body = io.NopCloser(bytes.NewReader(raw))
		}
	}
	return httpx.BeginIdempotent(w, r, idem, scope, dst)
}

func requireRead(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	if err := httpx.RequirePermission(r, "consolidation:read"); err != nil {
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
	if err := httpx.RequirePermission(r, "consolidation:write"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	if err := httpx.RequireMaxAdminTier(r, httpx.AdminTierHolding); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
}
