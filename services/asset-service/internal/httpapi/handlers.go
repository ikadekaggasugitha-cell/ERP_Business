package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/asset-service/internal/astdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type AssetStore interface {
	CreateAsset(ctx context.Context, scope []uuid.UUID, arg astdb.CreateAssetParams) (astdb.CreateAssetRow, error)
	GetAsset(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (astdb.AstAsset, error)
	ListAssets(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]astdb.AstAsset, int64, error)
	UpdateAsset(ctx context.Context, scope []uuid.UUID, arg astdb.UpdateAssetParams) (int32, error)
	DeleteAsset(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) error
	CreateLocationHistory(ctx context.Context, scope []uuid.UUID, arg astdb.CreateLocationHistoryParams) (astdb.CreateLocationHistoryRow, error)
	ListLocationHistory(ctx context.Context, scope []uuid.UUID, assetID uuid.UUID) ([]astdb.AstAssetLocationHistory, error)
	CreateMovement(ctx context.Context, scope []uuid.UUID, arg astdb.CreateAssetMovementParams) (astdb.CreateAssetMovementRow, error)
	ListMovements(ctx context.Context, scope []uuid.UUID, assetID *uuid.UUID, limit, offset int32) ([]astdb.AstAssetMovement, int64, error)
	ApproveMovement(ctx context.Context, scope []uuid.UUID, id, actor uuid.UUID, version int32) (int32, error)
	CompleteMovementAndEmit(ctx context.Context, scope []uuid.UUID, id, org, actor, corr uuid.UUID, version int32) (int32, error)
	RecordVerificationAndEmit(ctx context.Context, scope []uuid.UUID, arg astdb.CreateAssetVerificationParams, corr uuid.UUID) (astdb.CreateAssetVerificationRow, error)
	ListVerifications(ctx context.Context, scope []uuid.UUID, assetID *uuid.UUID, limit, offset int32) ([]astdb.AstAssetVerification, int64, error)
}

type Server struct {
	store AssetStore
	idem  *httpx.IdemStore
}

func NewServer(store AssetStore) *Server {
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

func rawJSONOrEmptyObj(b []byte) json.RawMessage {
	if len(bytes.TrimSpace(b)) == 0 {
		return json.RawMessage("{}")
	}
	return json.RawMessage(b)
}

type assetDTO struct {
	ID              uuid.UUID       `json:"id"`
	OrganizationID  uuid.UUID       `json:"organizationId"`
	AssetCode       string          `json:"assetCode"`
	Name            string          `json:"name"`
	Category        string          `json:"category"`
	Specification   json.RawMessage `json:"specification"`
	AcquisitionDate *string         `json:"acquisitionDate,omitempty"`
	OrganizationRef uuid.UUID       `json:"organizationRef"`
	FixedAssetRef   *uuid.UUID      `json:"fixedAssetRef,omitempty"`
	Barcode         *string         `json:"barcode,omitempty"`
	Status          string          `json:"status"`
	IsSystem        bool            `json:"isSystem"`
	CreatedAt       *string         `json:"createdAt,omitempty"`
	UpdatedAt       *string         `json:"updatedAt,omitempty"`
	RowVersion      int32           `json:"rowVersion"`
}

func toAssetDTO(r astdb.AstAsset) assetDTO {
	return assetDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, AssetCode: r.AssetCode, Name: r.Name,
		Category: r.Category, Specification: rawJSONOrEmptyObj(r.Specification),
		AcquisitionDate: fmtOptDate(r.AcquisitionDate), OrganizationRef: r.OrganizationRef,
		FixedAssetRef: r.FixedAssetRef, Barcode: r.Barcode, Status: r.Status,
		IsSystem: r.OrganizationID == platform.SystemOrgID,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type locationHistoryDTO struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  uuid.UUID  `json:"organizationId"`
	AssetID         uuid.UUID  `json:"assetId"`
	OrganizationRef uuid.UUID  `json:"organizationRef"`
	WarehouseRef    *uuid.UUID `json:"warehouseRef,omitempty"`
	LocationNote    *string    `json:"locationNote,omitempty"`
	EffectiveFrom   string     `json:"effectiveFrom"`
	CreatedAt       *string    `json:"createdAt,omitempty"`
	UpdatedAt       *string    `json:"updatedAt,omitempty"`
	RowVersion      int32      `json:"rowVersion"`
}

func toLocationHistoryDTO(r astdb.AstAssetLocationHistory) locationHistoryDTO {
	return locationHistoryDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, AssetID: r.AssetID,
		OrganizationRef: r.OrganizationRef, WarehouseRef: r.WarehouseRef, LocationNote: r.LocationNote,
		EffectiveFrom: platform.FormatDate(r.EffectiveFrom),
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type movementDTO struct {
	ID                  uuid.UUID  `json:"id"`
	OrganizationID      uuid.UUID  `json:"organizationId"`
	AssetID             uuid.UUID  `json:"assetId"`
	FromOrganizationRef *uuid.UUID `json:"fromOrganizationRef,omitempty"`
	ToOrganizationRef   uuid.UUID  `json:"toOrganizationRef"`
	Status              string     `json:"status"`
	ApprovedBy          *uuid.UUID `json:"approvedBy,omitempty"`
	MovedAt             *string    `json:"movedAt,omitempty"`
	CreatedAt           *string    `json:"createdAt,omitempty"`
	UpdatedAt           *string    `json:"updatedAt,omitempty"`
	RowVersion          int32      `json:"rowVersion"`
}

func toMovementDTO(r astdb.AstAssetMovement) movementDTO {
	return movementDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, AssetID: r.AssetID,
		FromOrganizationRef: r.FromOrganizationRef, ToOrganizationRef: r.ToOrganizationRef,
		Status: r.Status, ApprovedBy: r.ApprovedBy, MovedAt: fmtOptTS(r.MovedAt),
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type verificationDTO struct {
	ID               uuid.UUID `json:"id"`
	OrganizationID   uuid.UUID `json:"organizationId"`
	AssetID          uuid.UUID `json:"assetId"`
	VerificationDate string    `json:"verificationDate"`
	SystemStatus     *string   `json:"systemStatus,omitempty"`
	PhysicalStatus   *string   `json:"physicalStatus,omitempty"`
	DiscrepancyNote  *string   `json:"discrepancyNote,omitempty"`
	CreatedAt        *string   `json:"createdAt,omitempty"`
	UpdatedAt        *string   `json:"updatedAt,omitempty"`
	RowVersion       int32     `json:"rowVersion"`
}

func toVerificationDTO(r astdb.AstAssetVerification) verificationDTO {
	return verificationDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, AssetID: r.AssetID,
		VerificationDate: platform.FormatDate(r.VerificationDate),
		SystemStatus: r.SystemStatus, PhysicalStatus: r.PhysicalStatus, DiscrepancyNote: r.DiscrepancyNote,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /assets", s.listAssets)
	mux.HandleFunc("POST /assets", s.createAsset)
	mux.HandleFunc("GET /assets/{id}", s.getAsset)
	mux.HandleFunc("PUT /assets/{id}", s.updateAsset)
	mux.HandleFunc("DELETE /assets/{id}", s.deleteAsset)

	mux.HandleFunc("GET /assets/{assetId}/locations", s.listLocations)
	mux.HandleFunc("POST /assets/{assetId}/locations", s.createLocation)

	mux.HandleFunc("GET /asset-movements", s.listMovements)
	mux.HandleFunc("POST /asset-movements", s.createMovement)
	mux.HandleFunc("POST /asset-movements/{id}/approve", s.approveMovement)
	mux.HandleFunc("POST /asset-movements/{id}/complete", s.completeMovement)

	mux.HandleFunc("GET /asset-verifications", s.listVerifications)
	mux.HandleFunc("POST /asset-verifications", s.createVerification)
}

func requireRead(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	if err := httpx.RequirePermission(r, "asset:read"); err != nil {
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
	if err := httpx.RequirePermission(r, "asset:write"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
}

func validAssetStatus(st string) bool {
	switch st {
	case "Active", "UnderMaintenance", "Idle", "Damaged", "Disposed":
		return true
	}
	return false
}

func (s *Server) listAssets(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var st *string
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
		st = &v
	}
	rows, total, err := s.store.ListAssets(r.Context(), scope, st, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]assetDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toAssetDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createAsset(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		AssetCode       string          `json:"assetCode"`
		Name            string          `json:"name"`
		Category        string          `json:"category"`
		Specification   json.RawMessage `json:"specification"`
		AcquisitionDate *string         `json:"acquisitionDate"`
		OrganizationRef *string         `json:"organizationRef"`
		FixedAssetRef   *string         `json:"fixedAssetRef"`
		Barcode         *string         `json:"barcode"`
		Status          string          `json:"status"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	if strings.TrimSpace(req.AssetCode) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Category) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("assetCode, name, and category are required"))
		return
	}
	st := strings.TrimSpace(req.Status)
	if st == "" {
		st = "Active"
	}
	if !validAssetStatus(st) {
		httpx.WriteError(w, r, httpx.BadRequest("invalid asset status"))
		return
	}
	var acqDate pgtype.Date
	if req.AcquisitionDate != nil && strings.TrimSpace(*req.AcquisitionDate) != "" {
		d, err := platform.ParseDate(*req.AcquisitionDate)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("acquisitionDate must be YYYY-MM-DD"))
			return
		}
		acqDate = d
	}
	orgRef := scope[0]
	if req.OrganizationRef != nil && strings.TrimSpace(*req.OrganizationRef) != "" {
		p, err := uuid.Parse(*req.OrganizationRef)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid organizationRef"))
			return
		}
		orgRef = p
	}
	var faRef *uuid.UUID
	if req.FixedAssetRef != nil && strings.TrimSpace(*req.FixedAssetRef) != "" {
		p, err := uuid.Parse(*req.FixedAssetRef)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid fixedAssetRef"))
			return
		}
		faRef = &p
	}
	out, err := s.store.CreateAsset(r.Context(), scope, astdb.CreateAssetParams{
		OrganizationID: scope[0], AssetCode: strings.TrimSpace(req.AssetCode), Name: strings.TrimSpace(req.Name),
		Category: strings.TrimSpace(req.Category), Specification: req.Specification, AcquisitionDate: acqDate,
		OrganizationRef: orgRef, FixedAssetRef: faRef, Barcode: req.Barcode, Status: st, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": st, "rowVersion": out.RowVersion})
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
	row, err := s.store.GetAsset(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, toAssetDTO(row))
}

func (s *Server) updateAsset(w http.ResponseWriter, r *http.Request) {
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
		Name          string          `json:"name"`
		Category      string          `json:"category"`
		Specification json.RawMessage `json:"specification"`
		Barcode       *string         `json:"barcode"`
		Status        string          `json:"status"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Category) == "" || !validAssetStatus(req.Status) {
		httpx.WriteError(w, r, httpx.BadRequest("name, category, and valid status are required"))
		return
	}
	next, err := s.store.UpdateAsset(r.Context(), scope, astdb.UpdateAssetParams{
		ID: id, Name: strings.TrimSpace(req.Name), Category: strings.TrimSpace(req.Category),
		Specification: req.Specification, Barcode: req.Barcode, Status: req.Status, UpdatedBy: &user, RowVersion: ver,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "rowVersion": next})
}

func (s *Server) deleteAsset(w http.ResponseWriter, r *http.Request) {
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
	if err := s.store.DeleteAsset(r.Context(), scope, id, user, ver); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listLocations(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	aid, err := httpx.ParseUUIDParam("assetId", r.PathValue("assetId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListLocationHistory(r.Context(), scope, aid)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]locationHistoryDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toLocationHistoryDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) createLocation(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	aid, err := httpx.ParseUUIDParam("assetId", r.PathValue("assetId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		OrganizationRef *string `json:"organizationRef"`
		WarehouseRef    *string `json:"warehouseRef"`
		LocationNote    *string `json:"locationNote"`
		EffectiveFrom   string  `json:"effectiveFrom"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	eff, err := platform.ParseDate(req.EffectiveFrom)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("effectiveFrom must be YYYY-MM-DD"))
		return
	}
	orgRef := scope[0]
	if req.OrganizationRef != nil && strings.TrimSpace(*req.OrganizationRef) != "" {
		p, e := uuid.Parse(*req.OrganizationRef)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid organizationRef"))
			return
		}
		orgRef = p
	}
	var whRef *uuid.UUID
	if req.WarehouseRef != nil && strings.TrimSpace(*req.WarehouseRef) != "" {
		p, e := uuid.Parse(*req.WarehouseRef)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid warehouseRef"))
			return
		}
		whRef = &p
	}
	out, err := s.store.CreateLocationHistory(r.Context(), scope, astdb.CreateLocationHistoryParams{
		OrganizationID: scope[0], AssetID: aid, OrganizationRef: orgRef, WarehouseRef: whRef,
		LocationNote: req.LocationNote, EffectiveFrom: eff, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) listMovements(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var aid *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("assetId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid assetId"))
			return
		}
		aid = &parsed
	}
	rows, total, err := s.store.ListMovements(r.Context(), scope, aid, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]movementDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toMovementDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createMovement(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		AssetID             string  `json:"assetId"`
		FromOrganizationRef *string `json:"fromOrganizationRef"`
		ToOrganizationRef   string  `json:"toOrganizationRef"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	aid, err := uuid.Parse(req.AssetID)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid assetId is required"))
		return
	}
	toOrg, err := uuid.Parse(req.ToOrganizationRef)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid toOrganizationRef is required"))
		return
	}
	var fromOrg *uuid.UUID
	if req.FromOrganizationRef != nil && strings.TrimSpace(*req.FromOrganizationRef) != "" {
		p, e := uuid.Parse(*req.FromOrganizationRef)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid fromOrganizationRef"))
			return
		}
		fromOrg = &p
	}
	out, err := s.store.CreateMovement(r.Context(), scope, astdb.CreateAssetMovementParams{
		OrganizationID: scope[0], AssetID: aid,
		FromOrganizationRef: fromOrg, ToOrganizationRef: toOrg, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "Draft", "rowVersion": out.RowVersion})
}

func (s *Server) approveMovement(w http.ResponseWriter, r *http.Request) {
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
	var req struct{}
	idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	next, err := s.store.ApproveMovement(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Approved", "rowVersion": next})
}

func (s *Server) completeMovement(w http.ResponseWriter, r *http.Request) {
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
	var req struct{}
	idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	next, err := s.store.CompleteMovementAndEmit(r.Context(), scope, id, scope[0], user, httpx.CorrelationUUID(r), ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Completed", "rowVersion": next})
}

func (s *Server) listVerifications(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var aid *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("assetId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid assetId"))
			return
		}
		aid = &parsed
	}
	rows, total, err := s.store.ListVerifications(r.Context(), scope, aid, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]verificationDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toVerificationDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createVerification(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		AssetID          string  `json:"assetId"`
		VerificationDate string  `json:"verificationDate"`
		SystemStatus     *string `json:"systemStatus"`
		PhysicalStatus   *string `json:"physicalStatus"`
		DiscrepancyNote  *string `json:"discrepancyNote"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	aid, err := uuid.Parse(req.AssetID)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid assetId is required"))
		return
	}
	vd, err := platform.ParseDate(req.VerificationDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("verificationDate must be YYYY-MM-DD"))
		return
	}
	out, err := s.store.RecordVerificationAndEmit(r.Context(), scope, astdb.CreateAssetVerificationParams{
		OrganizationID: scope[0], AssetID: aid, VerificationDate: vd,
		SystemStatus: req.SystemStatus, PhysicalStatus: req.PhysicalStatus, DiscrepancyNote: req.DiscrepancyNote,
		CreatedBy: user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}
