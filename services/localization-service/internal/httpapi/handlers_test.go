package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/localization-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/localization-service/internal/localizationdb"
	"github.com/sinergiinvestasiproperti/erp/services/localization-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type mockStore struct {
	httpapi.Store
	bundle  map[string]string
	created *localizationdb.CreateLanguageParams
}

func (m *mockStore) CreateLanguage(_ context.Context, _ []uuid.UUID, a localizationdb.CreateLanguageParams, _ uuid.UUID) (localizationdb.CreateLanguageRow, error) {
	m.created = &a
	return localizationdb.CreateLanguageRow{ID: uuid.New(), RowVersion: 1}, nil
}

func (m *mockStore) CreateTranslation(_ context.Context, _ []uuid.UUID, _ localizationdb.CreateTranslationParams, _ uuid.UUID) (localizationdb.CreateTranslationRow, error) {
	return localizationdb.CreateTranslationRow{ID: uuid.New(), RowVersion: 1}, nil
}

func (m *mockStore) CreateLocaleFormat(_ context.Context, _ []uuid.UUID, _ localizationdb.CreateLocaleFormatParams, _ uuid.UUID) (localizationdb.CreateLocaleFormatRow, error) {
	return localizationdb.CreateLocaleFormatRow{ID: uuid.New(), RowVersion: 1}, nil
}

func (m *mockStore) GetBundle(_ context.Context, _ []uuid.UUID, _ string) (map[string]string, error) {
	return m.bundle, nil
}

func (m *mockStore) UpsertUserLanguagePreference(_ context.Context, _ []uuid.UUID, a localizationdb.UpsertUserLanguagePreferenceParams) (localizationdb.UpsertUserLanguagePreferenceRow, error) {
	return localizationdb.UpsertUserLanguagePreferenceRow{ID: uuid.New(), OrganizationID: a.OrganizationID, UserRef: a.UserRef, LanguageCode: a.LanguageCode, RowVersion: 1}, nil
}

func (m *mockStore) UpsertOrganizationLanguage(_ context.Context, _ []uuid.UUID, a localizationdb.UpsertOrganizationLanguageParams) (localizationdb.UpsertOrganizationLanguageRow, error) {
	return localizationdb.UpsertOrganizationLanguageRow{ID: uuid.New(), OrganizationID: a.OrganizationID, OrganizationRef: a.OrganizationRef, DefaultLanguageCode: a.DefaultLanguageCode, RowVersion: 1}, nil
}

func do(st httpapi.Store, method, path, body, perms string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	httpapi.NewServer(st).Register(mux)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-Organization-Id", "11111111-1111-1111-1111-111111111111")
	req.Header.Set("X-User-Id", "22222222-2222-2222-2222-222222222222")
	req.Header.Set("Content-Type", "application/json")
	if perms != "" {
		req.Header.Set("X-Permissions", perms)
	}
	rec := httptest.NewRecorder()
	httpx.Middleware(mux).ServeHTTP(rec, req)
	return rec
}

func TestCreateLanguage(t *testing.T) {
	st := &mockStore{}
	rec := do(st, "POST", "/languages", `{"code":"id","name":"Bahasa Indonesia","isFallback":true}`, "localization:write")
	if rec.Code != http.StatusCreated {
		t.Fatalf("want 201 got %d: %s", rec.Code, rec.Body)
	}
	if st.created == nil || !st.created.IsActive || !st.created.IsFallback {
		t.Fatalf("unexpected params %+v", st.created)
	}
}

func TestCreateLanguage_Validation(t *testing.T) {
	cases := map[string]struct {
		body string
		want int
	}{
		"bad code":          {`{"code":"INDONESIAN!","name":"x"}`, 400},
		"missing name":      {`{"code":"id"}`, 400},
		"inactive fallback": {`{"code":"id","name":"x","isActive":false,"isFallback":true}`, 422},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if rec := do(&mockStore{}, "POST", "/languages", c.body, "localization:write"); rec.Code != c.want {
				t.Fatalf("want %d got %d: %s", c.want, rec.Code, rec.Body)
			}
		})
	}
}

func TestCreateTranslationAndLocaleFormat_Validation(t *testing.T) {
	if rec := do(&mockStore{}, "POST", "/translations", `{"languageCode":"id","resourceKey":"","resourceValue":"x"}`, "localization:write"); rec.Code != 400 {
		t.Fatalf("empty key want 400 got %d", rec.Code)
	}
	if rec := do(&mockStore{}, "POST", "/translations", `{"languageCode":"id","resourceKey":"app.title","resourceValue":"Judul"}`, "localization:write"); rec.Code != 201 {
		t.Fatalf("valid translation want 201 got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(&mockStore{}, "POST", "/locale-formats", `{"localeCode":"indonesia","dateFormat":"a","numberFormat":"b","currencyFormat":"c"}`, "localization:write"); rec.Code != 400 {
		t.Fatalf("bad locale want 400 got %d", rec.Code)
	}
	if rec := do(&mockStore{}, "POST", "/locale-formats", `{"localeCode":"id-ID","dateFormat":"dd/MM/yyyy","numberFormat":"#.##0,00","currencyFormat":"Rp #.##0,00"}`, "localization:write"); rec.Code != 201 {
		t.Fatalf("valid locale want 201 got %d: %s", rec.Code, rec.Body)
	}
}

func TestBundle(t *testing.T) {
	st := &mockStore{bundle: map[string]string{"app.title": "Judul"}}
	rec := do(st, "GET", "/translation-bundles/id", "", "localization:read")
	if rec.Code != 200 {
		t.Fatalf("want 200 got %d", rec.Code)
	}
	var out struct {
		Entries map[string]string `json:"entries"`
		Count   int               `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.Entries["app.title"] != "Judul" || out.Count != 1 {
		t.Fatalf("unexpected body %s", rec.Body)
	}
	if rec := do(st, "GET", "/translation-bundles/NOT_A_CODE", "", "localization:read"); rec.Code != 400 {
		t.Fatalf("invalid code want 400 got %d", rec.Code)
	}
}

func TestPermissionDenied(t *testing.T) {
	if rec := do(&mockStore{}, "POST", "/languages", `{"code":"id","name":"x"}`, "localization:read"); rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d", rec.Code)
	}
}

func TestScopeWithSystemOrg(t *testing.T) {
	org := uuid.New()
	got := repository.ScopeWithSystemOrg([]uuid.UUID{org})
	if len(got) != 2 || got[1] != uuid.Nil {
		t.Fatalf("expected system org appended, got %v", got)
	}
	if again := repository.ScopeWithSystemOrg(got); len(again) != 2 {
		t.Fatalf("system org must not duplicate, got %v", again)
	}
	if repository.ScopeWithSystemOrg(nil) != nil {
		t.Fatal("empty scope must fail closed (nil)")
	}
}

func TestUserPreferencesAndOrgLanguages(t *testing.T) {
	st := &mockStore{}
	userRef := uuid.NewString()
	orgRef := uuid.NewString()
	if rec := do(st, "PUT", "/user-preferences/"+userRef, `{"languageCode":"id"}`, "localization:write"); rec.Code != http.StatusOK {
		t.Fatalf("upsert user preference want 200 got %d: %s", rec.Code, rec.Body.String())
	}
	if rec := do(st, "PUT", "/user-preferences/"+userRef, `{"languageCode":"INVALID!"}`, "localization:write"); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad user preference code want 400 got %d", rec.Code)
	}
	if rec := do(st, "PUT", "/organization-languages/"+orgRef, `{"defaultLanguageCode":"en-US"}`, "localization:write"); rec.Code != http.StatusOK {
		t.Fatalf("upsert org language want 200 got %d: %s", rec.Code, rec.Body.String())
	}
}

