// Package httpapi wires masterdata-service REST endpoints (/parties, /items).
// Creating or updating parties emits masterdata.party.upserted for downstream mdcache.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/masterdata-service/internal/masterdatadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type Store interface {
	CreatePartyAndEmit(ctx context.Context, scope []uuid.UUID, arg masterdatadb.CreatePartyParams, addrs []masterdatadb.CreatePartyAddressParams, ev masterdatadb.InsertOutboxParams) (masterdatadb.CreatePartyRow, error)
	GetPartyWithAddresses(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (masterdatadb.MdParty, []masterdatadb.ListPartyAddressesRow, error)
	ListPartiesWithCount(ctx context.Context, scope []uuid.UUID, arg masterdatadb.ListPartiesParams) ([]masterdatadb.ListPartiesRow, int64, error)
	UpdatePartyAndEmit(ctx context.Context, scope []uuid.UUID, arg masterdatadb.UpdatePartyParams, events ...masterdatadb.InsertOutboxParams) (int32, error)
	DeleteParty(ctx context.Context, scope []uuid.UUID, arg masterdatadb.DeletePartyParams) (int64, error)

	CreateItemAndEmit(ctx context.Context, scope []uuid.UUID, arg masterdatadb.CreateItemParams, ev masterdatadb.InsertOutboxParams) (masterdatadb.CreateItemRow, error)
	GetItemWithUOMs(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (masterdatadb.MdItem, []masterdatadb.ListItemUOMsRow, error)
	ListItemsWithCount(ctx context.Context, scope []uuid.UUID, arg masterdatadb.ListItemsParams) ([]masterdatadb.ListItemsRow, int64, error)
	UpdateItemAndEmit(ctx context.Context, scope []uuid.UUID, arg masterdatadb.UpdateItemParams, ev masterdatadb.InsertOutboxParams) (int32, error)
	DeleteItem(ctx context.Context, scope []uuid.UUID, arg masterdatadb.DeleteItemParams) (int64, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /parties", s.createParty)
	mux.HandleFunc("GET /parties", s.listParties)
	mux.HandleFunc("GET /parties/{id}", s.getParty)
	mux.HandleFunc("PUT /parties/{id}", s.updateParty)
	mux.HandleFunc("DELETE /parties/{id}", s.deleteParty)

	mux.HandleFunc("POST /items", s.createItem)
	mux.HandleFunc("GET /items", s.listItems)
	mux.HandleFunc("GET /items/{id}", s.getItem)
	mux.HandleFunc("PUT /items/{id}", s.updateItem)
	mux.HandleFunc("DELETE /items/{id}", s.deleteItem)
}

type addressReq struct {
	AddressType   string  `json:"addressType"`
	Line1         string  `json:"line1"`
	Line2         *string `json:"line2"`
	City          *string `json:"city"`
	StateProvince *string `json:"stateProvince"`
	PostalCode    *string `json:"postalCode"`
	CountryCode   *string `json:"countryCode"`
	IsPrimary     bool    `json:"isPrimary"`
}

type partyReq struct {
	Code      string       `json:"code"`
	Name      string       `json:"name"`
	PartyType string       `json:"partyType"`
	TaxID     string       `json:"taxId"`
	IsShared  bool         `json:"isShared"`
	Addresses []addressReq `json:"addresses"`
}

type itemReq struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	ItemType string `json:"itemType"`
	BaseUom  string `json:"baseUom"`
}

func (s *Server) createParty(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "masterdata:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req partyReq
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	pt := strings.ToLower(strings.TrimSpace(req.PartyType))
	if code == "" || name == "" || (pt != "customer" && pt != "vendor" && pt != "both") {
		httpx.WriteError(w, r, httpx.BadRequest("code, name, dan partyType (customer/vendor/both) wajib diisi"))
		return
	}
	addrs := make([]masterdatadb.CreatePartyAddressParams, 0, len(req.Addresses))
	for _, a := range req.Addresses {
		if strings.TrimSpace(a.Line1) == "" {
			httpx.WriteError(w, r, httpx.BadRequest("line1 alamat wajib diisi"))
			return
		}
		at := strings.TrimSpace(a.AddressType)
		if at == "" {
			at = "billing"
		}
		addrs = append(addrs, masterdatadb.CreatePartyAddressParams{
			AddressType:   at,
			Line1:         strings.TrimSpace(a.Line1),
			Line2:         a.Line2,
			City:          a.City,
			StateProvince: a.StateProvince,
			PostalCode:    a.PostalCode,
			CountryCode:   a.CountryCode,
			IsPrimary:     a.IsPrimary,
		})
	}
	var taxID *string
	if t := strings.TrimSpace(req.TaxID); t != "" {
		taxID = &t
	}
	payload, _ := json.Marshal(map[string]any{"code": code, "name": name, "partyType": pt, "taxId": taxID, "isShared": req.IsShared})
	ev := masterdatadb.InsertOutboxParams{
		AggregateType:  "Party",
		EventType:      "masterdata.party.upserted",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	created, err := s.repo.CreatePartyAndEmit(r.Context(), scope, masterdatadb.CreatePartyParams{
		OrganizationID: scope[0],
		Code:           code,
		Name:           name,
		PartyType:      pt,
		TaxID:          taxID,
		IsShared:       req.IsShared,
		CreatedBy:      actor,
	}, addrs, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode party sudah terdaftar"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, created.RowVersion, map[string]any{
		"id":         created.ID,
		"code":       code,
		"name":       name,
		"partyType":  pt,
		"rowVersion": created.RowVersion,
	})
}

func (s *Server) listParties(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "masterdata:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var ptPtr *string
	if pt := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("partyType"))); pt != "" {
		ptPtr = &pt
	}
	rows, total, err := s.repo.ListPartiesWithCount(r.Context(), scope, masterdatadb.ListPartiesParams{
		Limit:     limit,
		Offset:    offset,
		PartyType: ptPtr,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, p := range rows {
		out = append(out, map[string]any{
			"id":             p.ID,
			"organizationId": p.OrganizationID,
			"code":           p.Code,
			"name":           p.Name,
			"partyType":      p.PartyType,
			"taxId":          p.TaxID,
			"isShared":       p.IsShared,
			"rowVersion":     p.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getParty(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "masterdata:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, addrs, err := s.repo.GetPartyWithAddresses(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("party tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	addrOut := make([]map[string]any, 0, len(addrs))
	for _, a := range addrs {
		addrOut = append(addrOut, map[string]any{
			"id":          a.ID,
			"addressType": a.AddressType,
			"line1":       a.Line1,
			"line2":       a.Line2,
			"city":        a.City,
			"postalCode":  a.PostalCode,
			"countryCode": a.CountryCode,
			"isPrimary":   a.IsPrimary,
			"rowVersion":  a.RowVersion,
		})
	}
	httpx.WriteIDResponse(w, http.StatusOK, p.RowVersion, map[string]any{
		"id":             p.ID,
		"organizationId": p.OrganizationID,
		"code":           p.Code,
		"name":           p.Name,
		"partyType":      p.PartyType,
		"taxId":          p.TaxID,
		"isShared":       p.IsShared,
		"rowVersion":     p.RowVersion,
		"addresses":      addrOut,
	})
}

func (s *Server) updateParty(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "masterdata:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
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
	var req partyReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	pt := strings.ToLower(strings.TrimSpace(req.PartyType))
	if name == "" || (pt != "customer" && pt != "vendor" && pt != "both") {
		httpx.WriteError(w, r, httpx.BadRequest("name dan partyType (customer/vendor/both) wajib valid"))
		return
	}
	existing, _, err := s.repo.GetPartyWithAddresses(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("party tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.RowVersion != ver {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "row_version party tidak cocok"))
		return
	}
	var taxID *string
	if t := strings.TrimSpace(req.TaxID); t != "" {
		taxID = &t
	}
	payload, _ := json.Marshal(map[string]any{"code": existing.Code, "name": name, "partyType": pt, "taxId": taxID, "isShared": req.IsShared})
	corr := httpx.CorrelationUUID(r)
	evUpsert := masterdatadb.InsertOutboxParams{
		AggregateType:  "Party",
		AggregateID:    id,
		EventType:      "masterdata.party.upserted",
		Payload:        payload,
		OrganizationID: existing.OrganizationID,
		CorrelationID:  corr,
	}
	newVer, err := s.repo.UpdatePartyAndEmit(r.Context(), scope, masterdatadb.UpdatePartyParams{
		ID:         id,
		RowVersion: ver,
		Name:       name,
		PartyType:  pt,
		TaxID:      taxID,
		IsShared:   req.IsShared,
		UpdatedBy:  &actor,
	}, evUpsert)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "party berubah secara bersamaan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, map[string]any{
		"id":         id,
		"code":       existing.Code,
		"name":       name,
		"partyType":  pt,
		"rowVersion": newVer,
	})
}

func (s *Server) deleteParty(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "masterdata:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
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
	if _, _, err := s.repo.GetPartyWithAddresses(r.Context(), scope, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("party tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	n, err := s.repo.DeleteParty(r.Context(), scope, masterdatadb.DeletePartyParams{
		ID:         id,
		RowVersion: ver,
		UpdatedBy:  &actor,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "row_version party tidak cocok"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) createItem(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "masterdata:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req itemReq
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	it := strings.ToLower(strings.TrimSpace(req.ItemType))
	uom := strings.ToUpper(strings.TrimSpace(req.BaseUom))
	if code == "" || name == "" || uom == "" || (it != "stock" && it != "service" && it != "asset") {
		httpx.WriteError(w, r, httpx.BadRequest("code, name, baseUom, dan itemType (stock/service/asset) wajib diisi"))
		return
	}
	payload, _ := json.Marshal(map[string]any{"code": code, "name": name, "itemType": it, "baseUom": uom})
	ev := masterdatadb.InsertOutboxParams{
		AggregateType:  "Item",
		EventType:      "masterdata.item.upserted",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	created, err := s.repo.CreateItemAndEmit(r.Context(), scope, masterdatadb.CreateItemParams{
		OrganizationID: scope[0],
		Code:           code,
		Name:           name,
		ItemType:       it,
		BaseUom:        uom,
		CreatedBy:      actor,
	}, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode item sudah terdaftar"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, created.RowVersion, map[string]any{
		"id":         created.ID,
		"code":       code,
		"name":       name,
		"itemType":   it,
		"baseUom":    uom,
		"rowVersion": created.RowVersion,
	})
}

func (s *Server) listItems(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "masterdata:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var itPtr *string
	if it := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("itemType"))); it != "" {
		itPtr = &it
	}
	rows, total, err := s.repo.ListItemsWithCount(r.Context(), scope, masterdatadb.ListItemsParams{
		Limit:    limit,
		Offset:   offset,
		ItemType: itPtr,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, item := range rows {
		out = append(out, map[string]any{
			"id":             item.ID,
			"organizationId": item.OrganizationID,
			"code":           item.Code,
			"name":           item.Name,
			"itemType":       item.ItemType,
			"baseUom":        item.BaseUom,
			"rowVersion":     item.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getItem(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "masterdata:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	item, uoms, err := s.repo.GetItemWithUOMs(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("item tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	uomOut := make([]map[string]any, 0, len(uoms))
	for _, u := range uoms {
		uomOut = append(uomOut, map[string]any{
			"id":               u.ID,
			"uomCode":          u.UomCode,
			"conversionToBase": platform.FormatNumeric(u.ConversionToBase, 4),
			"isBase":           u.IsBase,
			"rowVersion":       u.RowVersion,
		})
	}
	httpx.WriteIDResponse(w, http.StatusOK, item.RowVersion, map[string]any{
		"id":             item.ID,
		"organizationId": item.OrganizationID,
		"code":           item.Code,
		"name":           item.Name,
		"itemType":       item.ItemType,
		"baseUom":        item.BaseUom,
		"rowVersion":     item.RowVersion,
		"uoms":           uomOut,
	})
}

func (s *Server) updateItem(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "masterdata:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
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
	var req itemReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	it := strings.ToLower(strings.TrimSpace(req.ItemType))
	uom := strings.ToUpper(strings.TrimSpace(req.BaseUom))
	if name == "" || uom == "" || (it != "stock" && it != "service" && it != "asset") {
		httpx.WriteError(w, r, httpx.BadRequest("name, baseUom, dan itemType (stock/service/asset) wajib valid"))
		return
	}
	existing, _, err := s.repo.GetItemWithUOMs(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("item tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.RowVersion != ver {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "row_version item tidak cocok"))
		return
	}
	payload, _ := json.Marshal(map[string]any{"code": existing.Code, "name": name, "itemType": it, "baseUom": uom})
	newVer, err := s.repo.UpdateItemAndEmit(r.Context(), scope, masterdatadb.UpdateItemParams{
		ID:         id,
		RowVersion: ver,
		Name:       name,
		ItemType:   it,
		BaseUom:    uom,
		UpdatedBy:  &actor,
	}, masterdatadb.InsertOutboxParams{
		AggregateType:  "Item",
		AggregateID:    id,
		EventType:      "masterdata.item.upserted",
		Payload:        payload,
		OrganizationID: existing.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "item berubah secara bersamaan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, map[string]any{
		"id":         id,
		"code":       existing.Code,
		"name":       name,
		"itemType":   it,
		"baseUom":    uom,
		"rowVersion": newVer,
	})
}

func (s *Server) deleteItem(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "masterdata:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
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
	if _, _, err := s.repo.GetItemWithUOMs(r.Context(), scope, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("item tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	n, err := s.repo.DeleteItem(r.Context(), scope, masterdatadb.DeleteItemParams{
		ID:         id,
		RowVersion: ver,
		UpdatedBy:  &actor,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "row_version item tidak cocok"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
