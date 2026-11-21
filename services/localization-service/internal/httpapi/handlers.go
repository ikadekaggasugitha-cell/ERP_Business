// Package httpapi implements Language, Translation, Translation Bundle and
// Locale Format endpoints with RLS scope, optimistic locking and idempotency.
package httpapi

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/localization-service/internal/localizationdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Store abstracts repository operations for handler unit testing.
type Store interface {
	CreateLanguage(ctx context.Context, scope []uuid.UUID, arg localizationdb.CreateLanguageParams, corr uuid.UUID) (localizationdb.CreateLanguageRow, error)
	GetLanguage(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (localizationdb.LocLanguage, error)
	ListLanguages(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]localizationdb.LocLanguage, int64, error)
	UpdateLanguage(ctx context.Context, scope []uuid.UUID, arg localizationdb.UpdateLanguageParams, corr uuid.UUID) (int32, error)

	CreateTranslation(ctx context.Context, scope []uuid.UUID, arg localizationdb.CreateTranslationParams, corr uuid.UUID) (localizationdb.CreateTranslationRow, error)
	GetTranslation(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (localizationdb.LocTranslation, error)
	ListTranslations(ctx context.Context, scope []uuid.UUID, lang, module *string, limit, offset int32) ([]localizationdb.LocTranslation, int64, error)
	UpdateTranslation(ctx context.Context, scope []uuid.UUID, arg localizationdb.UpdateTranslationParams, corr uuid.UUID) (int32, error)
	DeleteTranslation(ctx context.Context, scope []uuid.UUID, arg localizationdb.DeleteTranslationParams, corr uuid.UUID) error
	GetBundle(ctx context.Context, scope []uuid.UUID, languageCode string) (map[string]string, error)

	CreateLocaleFormat(ctx context.Context, scope []uuid.UUID, arg localizationdb.CreateLocaleFormatParams, corr uuid.UUID) (localizationdb.CreateLocaleFormatRow, error)
	GetLocaleFormat(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (localizationdb.LocLocaleFormat, error)
	ListLocaleFormats(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]localizationdb.LocLocaleFormat, int64, error)
	UpdateLocaleFormat(ctx context.Context, scope []uuid.UUID, arg localizationdb.UpdateLocaleFormatParams, corr uuid.UUID) (int32, error)
	DeleteLanguage(ctx context.Context, scope []uuid.UUID, arg localizationdb.DeleteLanguageParams, corr uuid.UUID) error
	DeleteLocaleFormat(ctx context.Context, scope []uuid.UUID, arg localizationdb.DeleteLocaleFormatParams, corr uuid.UUID) error

	UpsertUserLanguagePreference(ctx context.Context, scope []uuid.UUID, arg localizationdb.UpsertUserLanguagePreferenceParams) (localizationdb.UpsertUserLanguagePreferenceRow, error)
	GetUserLanguagePreference(ctx context.Context, scope []uuid.UUID, orgID, userRef uuid.UUID) (localizationdb.GetUserLanguagePreferenceRow, error)
	UpsertOrganizationLanguage(ctx context.Context, scope []uuid.UUID, arg localizationdb.UpsertOrganizationLanguageParams) (localizationdb.UpsertOrganizationLanguageRow, error)
	GetOrganizationLanguage(ctx context.Context, scope []uuid.UUID, orgID, orgRef uuid.UUID) (localizationdb.GetOrganizationLanguageRow, error)
	ListOrganizationLanguages(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]localizationdb.ListOrganizationLanguagesRow, int64, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /languages", s.listLanguages)
	mux.HandleFunc("POST /languages", s.createLanguage)
	mux.HandleFunc("GET /languages/{id}", s.getLanguage)
	mux.HandleFunc("PUT /languages/{id}", s.updateLanguage)
	mux.HandleFunc("DELETE /languages/{id}", s.deleteLanguage)

	mux.HandleFunc("GET /translations", s.listTranslations)
	mux.HandleFunc("POST /translations", s.createTranslation)
	mux.HandleFunc("GET /translations/{id}", s.getTranslation)
	mux.HandleFunc("PUT /translations/{id}", s.updateTranslation)
	mux.HandleFunc("DELETE /translations/{id}", s.deleteTranslation)
	mux.HandleFunc("GET /translation-bundles/{languageCode}", s.getBundle)

	mux.HandleFunc("GET /locale-formats", s.listLocaleFormats)
	mux.HandleFunc("POST /locale-formats", s.createLocaleFormat)
	mux.HandleFunc("GET /locale-formats/{id}", s.getLocaleFormat)
	mux.HandleFunc("PUT /locale-formats/{id}", s.updateLocaleFormat)
	mux.HandleFunc("DELETE /locale-formats/{id}", s.deleteLocaleFormat)

	mux.HandleFunc("GET /user-preferences/{userRef}", s.getUserPreference)
	mux.HandleFunc("PUT /user-preferences/{userRef}", s.upsertUserPreference)

	mux.HandleFunc("GET /organization-languages", s.listOrganizationLanguages)
	mux.HandleFunc("GET /organization-languages/{orgRef}", s.getOrganizationLanguage)
	mux.HandleFunc("PUT /organization-languages/{orgRef}", s.upsertOrganizationLanguage)
}

func requireWrite(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	if scope[0] == uuid.Nil {
		httpx.WriteError(w, r, httpx.Forbidden("system organization cannot be used as active tenant"))
		return nil, uuid.Nil, false
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	if err := httpx.RequirePermission(r, "localization:write"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, uuid.Nil, false
	}
	return scope, user, true
}

func requireRead(w http.ResponseWriter, r *http.Request) ([]uuid.UUID, bool) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	if err := httpx.RequirePermission(r, "localization:read"); err != nil {
		httpx.WriteError(w, r, err)
		return nil, false
	}
	return scope, true
}

func pathID(w http.ResponseWriter, r *http.Request, what string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid "+what+" id"))
		return uuid.Nil, false
	}
	return id, true
}

func queryPtr(r *http.Request, key string) *string {
	if v := strings.TrimSpace(r.URL.Query().Get(key)); v != "" {
		return &v
	}
	return nil
}

var (
	langCodeRe   = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})?$`)
	localeCodeRe = regexp.MustCompile(`^[a-z]{2,3}-[A-Z]{2}$`)
)

// -----------------------------------------------------------------------------
// Languages
// -----------------------------------------------------------------------------

type languageReq struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	IsActive   *bool  `json:"isActive,omitempty"`
	IsFallback *bool  `json:"isFallback,omitempty"`
}

type languageResp struct {
	ID         uuid.UUID `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	IsActive   bool      `json:"isActive"`
	IsFallback bool      `json:"isFallback"`
	IsSystem   bool      `json:"isSystem"`
	RowVersion int32     `json:"rowVersion"`
}

func languageDTO(l localizationdb.LocLanguage) languageResp {
	return languageResp{ID: l.ID, Code: l.Code, Name: l.Name, IsActive: l.IsActive, IsFallback: l.IsFallback, IsSystem: l.OrganizationID == platform.SystemOrgID, RowVersion: l.RowVersion}
}

func boolOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func (s *Server) listLanguages(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListLanguages(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	data := make([]languageResp, 0, len(rows))
	for _, row := range rows {
		data = append(data, languageDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: data, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createLanguage(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req languageReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	req.Code = strings.TrimSpace(req.Code)
	if !langCodeRe.MatchString(req.Code) {
		httpx.WriteError(w, r, httpx.BadRequest("code must be a language tag such as en or id"))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name is required"))
		return
	}
	active, fallback := boolOr(req.IsActive, true), boolOr(req.IsFallback, false)
	if fallback && !active {
		httpx.WriteError(w, r, httpx.Unprocessable("INACTIVE_FALLBACK", "fallback language must be active"))
		return
	}
	created, err := s.repo.CreateLanguage(r.Context(), scope, localizationdb.CreateLanguageParams{
		OrganizationID: scope[0], Code: req.Code, Name: strings.TrimSpace(req.Name),
		IsActive: active, IsFallback: fallback, CreatedBy: user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, languageResp{
		ID: created.ID, Code: req.Code, Name: strings.TrimSpace(req.Name), IsActive: active, IsFallback: fallback, RowVersion: created.RowVersion,
	})
}

func (s *Server) getLanguage(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "language")
	if !ok {
		return
	}
	row, err := s.repo.GetLanguage(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, languageDTO(row))
}

func (s *Server) updateLanguage(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "language")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req languageReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name is required"))
		return
	}
	active, fallback := boolOr(req.IsActive, true), boolOr(req.IsFallback, false)
	if fallback && !active {
		httpx.WriteError(w, r, httpx.Unprocessable("INACTIVE_FALLBACK", "fallback language must be active"))
		return
	}
	next, err := s.repo.UpdateLanguage(r.Context(), scope, localizationdb.UpdateLanguageParams{
		ID: id, RowVersion: rowVer, Name: strings.TrimSpace(req.Name), IsActive: active, IsFallback: fallback, UpdatedBy: &user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"rowVersion": next})
}

// -----------------------------------------------------------------------------
// Translations & Bundles
// -----------------------------------------------------------------------------

type translationReq struct {
	LanguageCode  string  `json:"languageCode"`
	ResourceKey   string  `json:"resourceKey"`
	ResourceValue string  `json:"resourceValue"`
	Module        *string `json:"module,omitempty"`
}

type translationResp struct {
	ID            uuid.UUID `json:"id"`
	LanguageCode  string    `json:"languageCode"`
	ResourceKey   string    `json:"resourceKey"`
	ResourceValue string    `json:"resourceValue"`
	Module        *string   `json:"module,omitempty"`
	IsSystem      bool      `json:"isSystem"`
	RowVersion    int32     `json:"rowVersion"`
}

func translationDTO(t localizationdb.LocTranslation) translationResp {
	return translationResp{ID: t.ID, LanguageCode: t.LanguageCode, ResourceKey: t.ResourceKey,
		ResourceValue: t.ResourceValue, Module: t.Module, IsSystem: t.OrganizationID == platform.SystemOrgID, RowVersion: t.RowVersion}
}

func (s *Server) listTranslations(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListTranslations(r.Context(), scope, queryPtr(r, "languageCode"), queryPtr(r, "module"), limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	data := make([]translationResp, 0, len(rows))
	for _, row := range rows {
		data = append(data, translationDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: data, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createTranslation(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req translationReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	req.LanguageCode = strings.TrimSpace(req.LanguageCode)
	req.ResourceKey = strings.TrimSpace(req.ResourceKey)
	if !langCodeRe.MatchString(req.LanguageCode) || req.ResourceKey == "" || len(req.ResourceKey) > 200 || strings.TrimSpace(req.ResourceValue) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("languageCode, resourceKey (<=200 chars) and resourceValue are required"))
		return
	}
	created, err := s.repo.CreateTranslation(r.Context(), scope, localizationdb.CreateTranslationParams{
		OrganizationID: scope[0], LanguageCode: req.LanguageCode, ResourceKey: req.ResourceKey,
		ResourceValue: req.ResourceValue, Module: req.Module, CreatedBy: user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, translationResp{
		ID: created.ID, LanguageCode: req.LanguageCode, ResourceKey: req.ResourceKey,
		ResourceValue: req.ResourceValue, Module: req.Module, RowVersion: created.RowVersion,
	})
}

func (s *Server) getTranslation(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "translation")
	if !ok {
		return
	}
	row, err := s.repo.GetTranslation(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, translationDTO(row))
}

func (s *Server) updateTranslation(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "translation")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req translationReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.ResourceValue) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("resourceValue is required"))
		return
	}
	next, err := s.repo.UpdateTranslation(r.Context(), scope, localizationdb.UpdateTranslationParams{
		ID: id, RowVersion: rowVer, ResourceValue: req.ResourceValue, Module: req.Module, UpdatedBy: &user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"rowVersion": next})
}

func (s *Server) deleteTranslation(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "translation")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.repo.DeleteTranslation(r.Context(), scope, localizationdb.DeleteTranslationParams{ID: id, RowVersion: rowVer, UpdatedBy: &user}, httpx.CorrelationUUID(r)); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getBundle(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	code := r.PathValue("languageCode")
	if !langCodeRe.MatchString(code) {
		httpx.WriteError(w, r, httpx.BadRequest("invalid language code"))
		return
	}
	bundle, err := s.repo.GetBundle(r.Context(), scope, code)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"languageCode": code, "entries": bundle, "count": len(bundle)})
}

// -----------------------------------------------------------------------------
// Locale Formats
// -----------------------------------------------------------------------------

type localeFormatReq struct {
	LocaleCode     string `json:"localeCode"`
	DateFormat     string `json:"dateFormat"`
	NumberFormat   string `json:"numberFormat"`
	CurrencyFormat string `json:"currencyFormat"`
}

type localeFormatResp struct {
	ID             uuid.UUID `json:"id"`
	LocaleCode     string    `json:"localeCode"`
	DateFormat     string    `json:"dateFormat"`
	NumberFormat   string    `json:"numberFormat"`
	CurrencyFormat string    `json:"currencyFormat"`
	IsSystem       bool      `json:"isSystem"`
	RowVersion     int32     `json:"rowVersion"`
}

func localeFormatDTO(l localizationdb.LocLocaleFormat) localeFormatResp {
	return localeFormatResp{ID: l.ID, LocaleCode: l.LocaleCode, DateFormat: l.DateFormat,
		NumberFormat: l.NumberFormat, CurrencyFormat: l.CurrencyFormat, IsSystem: l.OrganizationID == platform.SystemOrgID, RowVersion: l.RowVersion}
}

func formatsValid(req localeFormatReq) bool {
	return strings.TrimSpace(req.DateFormat) != "" && strings.TrimSpace(req.NumberFormat) != "" && strings.TrimSpace(req.CurrencyFormat) != ""
}

func (s *Server) listLocaleFormats(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListLocaleFormats(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	data := make([]localeFormatResp, 0, len(rows))
	for _, row := range rows {
		data = append(data, localeFormatDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: data, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createLocaleFormat(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req localeFormatReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	req.LocaleCode = strings.TrimSpace(req.LocaleCode)
	if !localeCodeRe.MatchString(req.LocaleCode) {
		httpx.WriteError(w, r, httpx.BadRequest("localeCode must look like id-ID or en-US"))
		return
	}
	if !formatsValid(req) {
		httpx.WriteError(w, r, httpx.BadRequest("dateFormat, numberFormat and currencyFormat are required"))
		return
	}
	created, err := s.repo.CreateLocaleFormat(r.Context(), scope, localizationdb.CreateLocaleFormatParams{
		OrganizationID: scope[0], LocaleCode: req.LocaleCode, DateFormat: req.DateFormat,
		NumberFormat: req.NumberFormat, CurrencyFormat: req.CurrencyFormat, CreatedBy: user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, localeFormatResp{
		ID: created.ID, LocaleCode: req.LocaleCode, DateFormat: req.DateFormat, NumberFormat: req.NumberFormat,
		CurrencyFormat: req.CurrencyFormat, RowVersion: created.RowVersion,
	})
}

func (s *Server) getLocaleFormat(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "locale format")
	if !ok {
		return
	}
	row, err := s.repo.GetLocaleFormat(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, localeFormatDTO(row))
}

func (s *Server) updateLocaleFormat(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "locale format")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req localeFormatReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if !formatsValid(req) {
		httpx.WriteError(w, r, httpx.BadRequest("dateFormat, numberFormat and currencyFormat are required"))
		return
	}
	next, err := s.repo.UpdateLocaleFormat(r.Context(), scope, localizationdb.UpdateLocaleFormatParams{
		ID: id, RowVersion: rowVer, DateFormat: req.DateFormat, NumberFormat: req.NumberFormat,
		CurrencyFormat: req.CurrencyFormat, UpdatedBy: &user,
	}, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"rowVersion": next})
}

func (s *Server) deleteLanguage(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "language")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.repo.DeleteLanguage(r.Context(), scope, localizationdb.DeleteLanguageParams{
		ID: id, RowVersion: rowVer, UpdatedBy: &user,
	}, httpx.CorrelationUUID(r)); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteLocaleFormat(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "locale format")
	if !ok {
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.repo.DeleteLocaleFormat(r.Context(), scope, localizationdb.DeleteLocaleFormatParams{
		ID: id, RowVersion: rowVer, UpdatedBy: &user,
	}, httpx.CorrelationUUID(r)); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getUserPreference(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	userRef, err := uuid.Parse(r.PathValue("userRef"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid userRef"))
		return
	}
	row, err := s.repo.GetUserLanguagePreference(r.Context(), scope, scope[0], userRef)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, map[string]any{
		"id": row.ID, "userRef": row.UserRef, "languageCode": row.LanguageCode, "rowVersion": row.RowVersion,
	})
}

func (s *Server) upsertUserPreference(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	userRef, err := uuid.Parse(r.PathValue("userRef"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid userRef"))
		return
	}
	var req struct {
		LanguageCode string `json:"languageCode"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	req.LanguageCode = strings.TrimSpace(req.LanguageCode)
	if !langCodeRe.MatchString(req.LanguageCode) {
		httpx.WriteError(w, r, httpx.BadRequest("languageCode must be a valid BCP-47 code"))
		return
	}
	row, err := s.repo.UpsertUserLanguagePreference(r.Context(), scope, localizationdb.UpsertUserLanguagePreferenceParams{
		OrganizationID: scope[0], UserRef: userRef, LanguageCode: req.LanguageCode, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, map[string]any{
		"id": row.ID, "userRef": row.UserRef, "languageCode": row.LanguageCode, "rowVersion": row.RowVersion,
	})
}

func (s *Server) listOrganizationLanguages(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListOrganizationLanguages(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	data := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		data = append(data, map[string]any{
			"id": row.ID, "organizationRef": row.OrganizationRef, "defaultLanguageCode": row.DefaultLanguageCode, "rowVersion": row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: data, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) getOrganizationLanguage(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	orgRef, err := uuid.Parse(r.PathValue("orgRef"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid orgRef"))
		return
	}
	row, err := s.repo.GetOrganizationLanguage(r.Context(), scope, scope[0], orgRef)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, map[string]any{
		"id": row.ID, "organizationRef": row.OrganizationRef, "defaultLanguageCode": row.DefaultLanguageCode, "rowVersion": row.RowVersion,
	})
}

func (s *Server) upsertOrganizationLanguage(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	orgRef, err := uuid.Parse(r.PathValue("orgRef"))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid orgRef"))
		return
	}
	var req struct {
		DefaultLanguageCode string `json:"defaultLanguageCode"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	req.DefaultLanguageCode = strings.TrimSpace(req.DefaultLanguageCode)
	if !langCodeRe.MatchString(req.DefaultLanguageCode) {
		httpx.WriteError(w, r, httpx.BadRequest("defaultLanguageCode must be a valid BCP-47 code"))
		return
	}
	row, err := s.repo.UpsertOrganizationLanguage(r.Context(), scope, localizationdb.UpsertOrganizationLanguageParams{
		OrganizationID: scope[0], OrganizationRef: orgRef, DefaultLanguageCode: req.DefaultLanguageCode, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, map[string]any{
		"id": row.ID, "organizationRef": row.OrganizationRef, "defaultLanguageCode": row.DefaultLanguageCode, "rowVersion": row.RowVersion,
	})
}

