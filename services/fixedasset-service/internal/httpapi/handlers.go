package httpapi

import (
	"bytes"
	"context"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/fadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type Store interface {
	CreateCategory(ctx context.Context, scope []uuid.UUID, arg fadb.CreateAssetCategoryParams) (fadb.CreateAssetCategoryRow, error)
	GetCategory(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (fadb.FaAssetCategory, error)
	ListCategories(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]fadb.FaAssetCategory, int64, error)
	UpdateCategory(ctx context.Context, scope []uuid.UUID, arg fadb.UpdateAssetCategoryParams) (int32, error)
	DeleteCategory(ctx context.Context, scope []uuid.UUID, arg fadb.DeleteAssetCategoryParams) error
	CreateAsset(ctx context.Context, scope []uuid.UUID, arg fadb.CreateFixedAssetParams) (fadb.CreateFixedAssetRow, error)
	GetAsset(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (fadb.FaFixedAsset, error)
	ListAssets(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]fadb.FaFixedAsset, int64, error)
	UpdateAssetDraft(ctx context.Context, scope []uuid.UUID, arg fadb.UpdateFixedAssetDraftParams) (int32, error)
	DeleteAssetDraft(ctx context.Context, scope []uuid.UUID, arg fadb.DeleteFixedAssetDraftParams) error
	CapitalizeAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, txDate pgtype.Date, actor, corr uuid.UUID) (int32, error)
	DepreciateAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, fiscalPeriodRef uuid.UUID, runDate pgtype.Date, unitsProduced, totalEstimatedUnits *big.Rat, actor, corr uuid.UUID) (fadb.CreateDepreciationEntryRow, string, string, int32, error)
	DisposeAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, proceeds pgtype.Numeric, txDate pgtype.Date, notes *string, actor, corr uuid.UUID) (int32, string, error)
	ListDepreciationEntries(ctx context.Context, scope []uuid.UUID, assetID uuid.UUID) ([]fadb.FaDepreciationEntry, error)
	ListTransactions(ctx context.Context, scope []uuid.UUID, assetID uuid.UUID) ([]fadb.FaAssetTransaction, error)
}

type Server struct {
	repo         Store
	idem         *httpx.IdemStore
	mu           sync.RWMutex
	transfers    map[uuid.UUID][]map[string]any
	revaluations map[uuid.UUID][]map[string]any
}

func NewServer(repo Store) *Server {
	return &Server{
		repo:         repo,
		idem:         httpx.NewIdemStore(),
		transfers:    make(map[uuid.UUID][]map[string]any),
		revaluations: make(map[uuid.UUID][]map[string]any),
	}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /asset-categories", s.listCategories)
	mux.HandleFunc("POST /asset-categories", s.createCategory)
	mux.HandleFunc("GET /asset-categories/{id}", s.getCategory)
	mux.HandleFunc("PUT /asset-categories/{id}", s.updateCategory)
	mux.HandleFunc("DELETE /asset-categories/{id}", s.deleteCategory)
	mux.HandleFunc("GET /fixed-assets", s.listAssets)
	mux.HandleFunc("POST /fixed-assets", s.createAsset)
	mux.HandleFunc("POST /fixed-assets/depreciate-batch", s.depreciateBatch)
	mux.HandleFunc("GET /fixed-assets/{id}", s.getAsset)
	mux.HandleFunc("PUT /fixed-assets/{id}", s.updateAssetDraft)
	mux.HandleFunc("DELETE /fixed-assets/{id}", s.deleteAssetDraft)
	mux.HandleFunc("POST /fixed-assets/{id}/capitalize", s.capitalizeAsset)
	mux.HandleFunc("POST /fixed-assets/{id}/depreciate", s.depreciateAsset)
	mux.HandleFunc("POST /fixed-assets/{id}/dispose", s.disposeAsset)
	mux.HandleFunc("POST /fixed-assets/{id}/transfer", s.transferAsset)
	mux.HandleFunc("POST /fixed-assets/{id}/revalue", s.revalueAsset)
	mux.HandleFunc("GET /fixed-assets/{id}/depreciation-entries", s.listDepreciationEntries)
	mux.HandleFunc("GET /fixed-assets/{id}/transactions", s.listTransactions)
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
	if err := httpx.RequirePermission(r, "fixedasset:read"); err != nil {
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
	if err := httpx.RequirePermission(r, "fixedasset:write"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
}

func validMethod(m string) bool {
	return m == "StraightLine" || m == "DecliningBalance" || m == "UnitsOfProduction"
}
