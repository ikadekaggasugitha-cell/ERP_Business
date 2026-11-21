package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/fadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type Store interface {
	CreateCategory(ctx context.Context, scope []uuid.UUID, arg fadb.CreateAssetCategoryParams) (fadb.CreateAssetCategoryRow, error)
	ListCategories(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]fadb.FaAssetCategory, int64, error)
	CreateAsset(ctx context.Context, scope []uuid.UUID, arg fadb.CreateFixedAssetParams) (fadb.CreateFixedAssetRow, error)
	GetAsset(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (fadb.FaFixedAsset, error)
	ListAssets(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]fadb.FaFixedAsset, int64, error)
	CapitalizeAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, txDate pgtype.Date, actor, corr uuid.UUID) (int32, error)
	DepreciateAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, fiscalPeriodRef uuid.UUID, runDate pgtype.Date, actor, corr uuid.UUID) (fadb.CreateDepreciationEntryRow, string, string, int32, error)
	DisposeAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, rowVer int32, proceeds pgtype.Numeric, txDate pgtype.Date, notes *string, actor, corr uuid.UUID) (int32, string, error)
	ListDepreciationEntries(ctx context.Context, scope []uuid.UUID, assetID uuid.UUID) ([]fadb.FaDepreciationEntry, error)
	ListTransactions(ctx context.Context, scope []uuid.UUID, assetID uuid.UUID) ([]fadb.FaAssetTransaction, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server { return &Server{repo: repo, idem: httpx.NewIdemStore()} }

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /asset-categories", s.listCategories)
	mux.HandleFunc("POST /asset-categories", s.createCategory)
	mux.HandleFunc("GET /fixed-assets", s.listAssets)
	mux.HandleFunc("POST /fixed-assets", s.createAsset)
	mux.HandleFunc("GET /fixed-assets/{id}", s.getAsset)
	mux.HandleFunc("POST /fixed-assets/{id}/capitalize", s.capitalizeAsset)
	mux.HandleFunc("POST /fixed-assets/{id}/depreciate", s.depreciateAsset)
	mux.HandleFunc("POST /fixed-assets/{id}/dispose", s.disposeAsset)
	mux.HandleFunc("GET /fixed-assets/{id}/depreciation-entries", s.listDepreciationEntries)
	mux.HandleFunc("GET /fixed-assets/{id}/transactions", s.listTransactions)
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

func (s *Server) listCategories(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListCategories(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: rows, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createCategory(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		Code                      string `json:"code"`
		Name                      string `json:"name"`
		DefaultDepreciationMethod string `json:"defaultDepreciationMethod"`
		DefaultUsefulLifeMonths   *int32 `json:"defaultUsefulLifeMonths,omitempty"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code and name are required"))
		return
	}
	method := strings.TrimSpace(req.DefaultDepreciationMethod)
	if method == "" {
		method = "StraightLine"
	}
	if !validMethod(method) {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_METHOD", "invalid depreciation method"))
		return
	}
	created, err := s.repo.CreateCategory(r.Context(), scope, fadb.CreateAssetCategoryParams{
		OrganizationID: scope[0], Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name),
		DefaultDepreciationMethod: method, DefaultUsefulLifeMonths: req.DefaultUsefulLifeMonths, IsActive: true, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{
		"id": created.ID, "code": strings.TrimSpace(req.Code), "name": strings.TrimSpace(req.Name), "rowVersion": created.RowVersion,
	})
}

func assetDTO(a fadb.FaFixedAsset) map[string]any {
	return map[string]any{
		"id": a.ID, "assetNo": a.AssetNo, "name": a.Name, "assetCategoryId": a.AssetCategoryID,
		"financialOrganizationRef": a.FinancialOrganizationRef, "acquisitionDate": platform.FormatDate(a.AcquisitionDate),
		"acquisitionCost": platform.FormatNumeric(a.AcquisitionCost, 4), "salvageValue": platform.FormatNumeric(a.SalvageValue, 4),
		"usefulLifeMonths": a.UsefulLifeMonths, "depreciationMethod": a.DepreciationMethod,
		"netBookValue": platform.FormatNumeric(a.NetBookValue, 4), "status": a.Status, "rowVersion": a.RowVersion,
	}
}

func (s *Server) listAssets(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var status *string
	if st := strings.TrimSpace(r.URL.Query().Get("status")); st != "" {
		status = &st
	}
	rows, total, err := s.repo.ListAssets(r.Context(), scope, status, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, a := range rows {
		out = append(out, assetDTO(a))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createAsset(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		AssetNo                      string     `json:"assetNo"`
		Name                         string     `json:"name"`
		AssetCategoryID              uuid.UUID  `json:"assetCategoryId"`
		FinancialOrganizationRef     uuid.UUID  `json:"financialOrganizationRef"`
		AcquisitionDate              string     `json:"acquisitionDate"`
		AcquisitionCost              string     `json:"acquisitionCost"`
		SalvageValue                 string     `json:"salvageValue"`
		UsefulLifeMonths             int32      `json:"usefulLifeMonths"`
		DepreciationMethod           string     `json:"depreciationMethod"`
		AssetAccountID               uuid.UUID  `json:"assetAccountId"`
		AccumDepreciationAccountID   uuid.UUID  `json:"accumDepreciationAccountId"`
		DepreciationExpenseAccountID uuid.UUID  `json:"depreciationExpenseAccountId"`
		CostCenterRef                *uuid.UUID `json:"costCenterRef,omitempty"`
		ParentAssetID                *uuid.UUID `json:"parentAssetId,omitempty"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.AssetNo) == "" || strings.TrimSpace(req.Name) == "" || req.AssetCategoryID == uuid.Nil || req.FinancialOrganizationRef == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("assetNo, name, assetCategoryId, and financialOrganizationRef are required"))
		return
	}
	if req.UsefulLifeMonths <= 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_LIFE", "usefulLifeMonths must be > 0"))
		return
	}
	acqDate, err := platform.ParseDate(req.AcquisitionDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("acquisitionDate must be YYYY-MM-DD"))
		return
	}
	cost, err := platform.ParseNumeric(req.AcquisitionCost)
	if err != nil || platform.NumericToRat(cost).Sign() <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("acquisitionCost must be > 0"))
		return
	}
	salvage, err := platform.ParseOptionalNumeric(req.SalvageValue)
	if err != nil || platform.NumericToRat(salvage).Sign() < 0 {
		httpx.WriteError(w, r, httpx.BadRequest("salvageValue must be >= 0"))
		return
	}
	if platform.NumericToRat(salvage).Cmp(platform.NumericToRat(cost)) > 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_SALVAGE", "salvageValue cannot exceed acquisitionCost"))
		return
	}
	method := strings.TrimSpace(req.DepreciationMethod)
	if method == "" {
		method = "StraightLine"
	}
	if !validMethod(method) {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_METHOD", "invalid depreciationMethod"))
		return
	}
	created, err := s.repo.CreateAsset(r.Context(), scope, fadb.CreateFixedAssetParams{
		OrganizationID: scope[0], AssetNo: strings.TrimSpace(req.AssetNo), Name: strings.TrimSpace(req.Name),
		AssetCategoryID: req.AssetCategoryID, FinancialOrganizationRef: req.FinancialOrganizationRef,
		AcquisitionDate: acqDate, AcquisitionCost: cost, SalvageValue: salvage, UsefulLifeMonths: req.UsefulLifeMonths,
		DepreciationMethod: method, AssetAccountID: req.AssetAccountID, AccumDepreciationAccountID: req.AccumDepreciationAccountID,
		DepreciationExpenseAccountID: req.DepreciationExpenseAccountID, CostCenterRef: req.CostCenterRef,
		NetBookValue: cost, ParentAssetID: req.ParentAssetID, Status: "Draft", CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{
		"id": created.ID, "assetNo": strings.TrimSpace(req.AssetNo), "status": created.Status, "rowVersion": created.RowVersion,
	})
}

func (s *Server) getAsset(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	a, err := s.repo.GetAsset(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, a.RowVersion, assetDTO(a))
}

func (s *Server) capitalizeAsset(w http.ResponseWriter, r *http.Request) {
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
		TransactionDate string `json:"transactionDate"`
	}
	if err := httpx.DecodeOptionalJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	txDate := pgtype.Date{Time: time.Now().UTC(), Valid: true}
	if strings.TrimSpace(req.TransactionDate) != "" {
		if d, perr := platform.ParseDate(req.TransactionDate); perr == nil {
			txDate = d
		} else {
			httpx.WriteError(w, r, httpx.BadRequest("transactionDate must be YYYY-MM-DD"))
			return
		}
	}
	next, err := s.repo.CapitalizeAndEmit(r.Context(), scope, id, rowVer, txDate, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": "Active", "rowVersion": next})
}

func (s *Server) depreciateAsset(w http.ResponseWriter, r *http.Request) {
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
		FiscalPeriodRef uuid.UUID `json:"fiscalPeriodRef"`
		RunDate         string    `json:"runDate"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if req.FiscalPeriodRef == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("fiscalPeriodRef is required"))
		return
	}
	runDate := pgtype.Date{Time: time.Now().UTC(), Valid: true}
	if strings.TrimSpace(req.RunDate) != "" {
		if d, perr := platform.ParseDate(req.RunDate); perr == nil {
			runDate = d
		} else {
			httpx.WriteError(w, r, httpx.BadRequest("runDate must be YYYY-MM-DD"))
			return
		}
	}
	entry, depAmt, nbv, next, err := s.repo.DepreciateAndEmit(r.Context(), scope, id, rowVer, req.FiscalPeriodRef, runDate, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{
		"entryId": entry.ID, "depreciationAmount": depAmt, "netBookValue": nbv, "rowVersion": next,
	})
}

func (s *Server) disposeAsset(w http.ResponseWriter, r *http.Request) {
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
		ProceedsAmount  string  `json:"proceedsAmount"`
		TransactionDate string  `json:"transactionDate"`
		Notes           *string `json:"notes,omitempty"`
	}
	if err := httpx.DecodeOptionalJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	proceeds, err := platform.ParseOptionalNumeric(req.ProceedsAmount)
	if err != nil || platform.NumericToRat(proceeds).Sign() < 0 {
		httpx.WriteError(w, r, httpx.BadRequest("proceedsAmount must be >= 0"))
		return
	}
	txDate := pgtype.Date{Time: time.Now().UTC(), Valid: true}
	if strings.TrimSpace(req.TransactionDate) != "" {
		if d, perr := platform.ParseDate(req.TransactionDate); perr == nil {
			txDate = d
		} else {
			httpx.WriteError(w, r, httpx.BadRequest("transactionDate must be YYYY-MM-DD"))
			return
		}
	}
	next, gainLoss, err := s.repo.DisposeAndEmit(r.Context(), scope, id, rowVer, proceeds, txDate, req.Notes, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "status": "Disposed", "gainLossAmount": gainLoss, "rowVersion": next})
}

func (s *Server) listDepreciationEntries(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.repo.ListDepreciationEntries(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: rows, Page: httpx.NewPageMeta(1, len(rows), len(rows))})
}

func (s *Server) listTransactions(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.repo.ListTransactions(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: rows, Page: httpx.NewPageMeta(1, len(rows), len(rows))})
}
