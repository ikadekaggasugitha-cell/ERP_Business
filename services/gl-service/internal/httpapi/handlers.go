// Package httpapi implements the gl-service REST endpoints (SDD BAB 4):
// strict fixed-point decimal parsing, optimistic locking, idempotency, RBAC,
// real total counts, and structured error envelopes.
package httpapi

import (
	"context"
	"net/http"
	"sync"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/gl-service/internal/gldb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type Store interface {
	CreateEntryWithLines(ctx context.Context, scope []uuid.UUID, entry gldb.CreateJournalEntryParams, lines []gldb.CreateJournalLineParams) (gldb.CreateJournalEntryRow, error)
	GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (gldb.GlJournalEntry, []gldb.ListJournalLinesRow, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, p gldb.ListJournalEntriesParams) ([]gldb.ListJournalEntriesRow, int64, error)
	UpdateDraftEntry(ctx context.Context, scope []uuid.UUID, hdr gldb.UpdateJournalEntryDraftParams, lines []gldb.CreateJournalLineParams) (int64, error)
	SoftDeleteDraft(ctx context.Context, scope []uuid.UUID, p gldb.SoftDeleteJournalEntryParams) (int64, error)
	Totals(ctx context.Context, scope []uuid.UUID, p gldb.JournalEntryTotalsParams) (gldb.JournalEntryTotalsRow, error)
	PostAndEmit(ctx context.Context, scope []uuid.UUID, entry gldb.GlJournalEntry, rowVer int32, actor uuid.UUID) (int64, error)
	ReverseEntryAndEmit(ctx context.Context, scope []uuid.UUID, orig gldb.GlJournalEntry, rowVer int32, actor, corrID uuid.UUID) (gldb.CreateJournalEntryRow, int64, error)
	ListAccountBalances(ctx context.Context, scope []uuid.UUID, ledgerID, periodID uuid.UUID, limit, offset int32) ([]gldb.ListAccountBalancesRow, int64, error)
}

type Server struct {
	repo      Store
	idem      *httpx.IdemStore
	recMu     sync.RWMutex
	recurring map[uuid.UUID][]recurringTemplateDTO
}

func NewServer(repo Store) *Server {
	return &Server{
		repo:      repo,
		idem:      httpx.NewIdemStore(),
		recurring: make(map[uuid.UUID][]recurringTemplateDTO),
	}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /journal-entries", s.create)
	mux.HandleFunc("GET /journal-entries", s.list)
	mux.HandleFunc("POST /journal-entries/fx-revaluation", s.fxRevaluation)
	mux.HandleFunc("POST /journal-entries/recurring", s.createRecurringTemplate)
	mux.HandleFunc("GET /journal-entries/recurring", s.listRecurringTemplates)
	mux.HandleFunc("GET /journal-entries/{id}", s.get)
	mux.HandleFunc("PUT /journal-entries/{id}", s.update)
	mux.HandleFunc("DELETE /journal-entries/{id}", s.deleteDraft)
	mux.HandleFunc("POST /journal-entries/{id}/post", s.post)
	mux.HandleFunc("POST /journal-entries/{id}/reverse", s.reverse)
	mux.HandleFunc("GET /trial-balance", s.trialBalance)
}

// ---- DTOs ------------------------------------------------------------------
