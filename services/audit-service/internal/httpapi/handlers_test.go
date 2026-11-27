package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/audit-service/internal/auditdb"
	"github.com/sinergiinvestasiproperti/erp/services/audit-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

type stubStore struct {
	rows   []auditdb.ListAuditLogsFilteredRow
	total  int64
	detail auditdb.AuditAuditLog
	err    error
	filter repository.ListFilter
}

func (s *stubStore) ListFiltered(_ context.Context, _ []uuid.UUID, filter repository.ListFilter) ([]auditdb.ListAuditLogsFilteredRow, int64, error) {
	s.filter = filter
	return s.rows, s.total, s.err
}

func (s *stubStore) GetByID(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (auditdb.AuditAuditLog, error) {
	if s.err != nil {
		return auditdb.AuditAuditLog{}, s.err
	}
	return s.detail, nil
}

func TestAuditEndpointsPaginationRBACAndGetByID(t *testing.T) {
	orgID := uuid.New()
	id := uuid.New()
	corrID := uuid.New()
	now := time.Now().UTC()

	store := &stubStore{
		rows: []auditdb.ListAuditLogsFilteredRow{{
			ID:             id,
			OccurredAt:     pgtype.Timestamptz{Time: now, Valid: true},
			OrganizationID: orgID,
			Action:         "gl.journal.posted",
			EntityType:     "JournalEntry",
			CorrelationID:  corrID,
		}},
		total: 42,
		detail: auditdb.AuditAuditLog{
			ID:             id,
			OccurredAt:     pgtype.Timestamptz{Time: now, Valid: true},
			OrganizationID: orgID,
			Action:         "gl.journal.posted",
			EntityType:     "JournalEntry",
			AfterValue:     []byte(`{"journalNumber":"JV-01"}`),
			CorrelationID:  corrID,
		},
	}

	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	// 1. RBAC denies caller missing audit:read when gateway headers are present.
	req := httptest.NewRequest(http.MethodGet, "/audit-events", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Roles", "VIEWER")
	req.Header.Set("X-Permissions", "gl:read")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing audit:read, got %d", rec.Code)
	}

	// 2. Authorized list passes filters (including from/to/entityId/correlationId) and returns full pagination metadata.
	entID := uuid.New()
	req = httptest.NewRequest(http.MethodGet, "/audit-events?page=2&size=10&entityType=JournalEntry&action=gl.journal.posted&from=2026-11-01&to=2026-11-30&entityId="+entID.String()+"&correlationId="+corrID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Permissions", "audit:read")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.filter.EntityType != "JournalEntry" || store.filter.Action != "gl.journal.posted" {
		t.Fatalf("filters not forwarded: %+v", store.filter)
	}
	if !store.filter.From.Valid || !store.filter.To.Valid {
		t.Fatalf("expected from/to timestamps forwarded, got %+v", store.filter)
	}
	if store.filter.EntityID == nil || *store.filter.EntityID != entID {
		t.Fatalf("expected entityId=%s, got %v", entID, store.filter.EntityID)
	}
	if store.filter.CorrelationID == nil || *store.filter.CorrelationID != corrID {
		t.Fatalf("expected correlationId=%s, got %v", corrID, store.filter.CorrelationID)
	}

	// 2b. Malformed filter params return 400.
	for _, badQuery := range []string{"from=not-a-date", "to=bad", "entityId=not-uuid", "correlationId=not-uuid"} {
		badReq := httptest.NewRequest(http.MethodGet, "/audit-events?"+badQuery, nil)
		badReq.Header.Set("X-Organization-Id", orgID.String())
		badReq.Header.Set("X-Permissions", "audit:read")
		badRec := httptest.NewRecorder()
		h.ServeHTTP(badRec, badReq)
		if badRec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", badQuery, badRec.Code)
		}
	}
	var resp struct {
		Data []auditEventDTO `json:"data"`
		Page httpx.PageMeta  `json:"page"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Page.TotalElements != 42 || resp.Page.TotalPages != 5 || len(resp.Data) != 1 {
		t.Fatalf("unexpected page/data: %+v", resp)
	}

	// 3. Get by ID returns detail; ErrNoRows maps to 404.
	req = httptest.NewRequest(http.MethodGet, "/audit-events/"+id.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Permissions", "audit:read")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 detail, got %d", rec.Code)
	}

	store.err = pgx.ErrNoRows
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on ErrNoRows, got %d", rec.Code)
	}
}

func TestAuditExportAndVerifyChain(t *testing.T) {
	orgID := uuid.New()
	corrID := uuid.New()
	t1 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)

	store := &stubStore{
		rows: []auditdb.ListAuditLogsFilteredRow{
			{ID: uuid.New(), OccurredAt: pgtype.Timestamptz{Time: t2, Valid: true}, OrganizationID: orgID, Action: "=gl.journal.posted", EntityType: "JournalEntry", CorrelationID: corrID},
			{ID: uuid.New(), OccurredAt: pgtype.Timestamptz{Time: t1, Valid: true}, OrganizationID: orgID, Action: "ap.invoice.recorded", EntityType: "VendorInvoice", CorrelationID: corrID},
		},
		total: 2,
	}

	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	// 1. CSV export with formula neutralization
	req := httptest.NewRequest(http.MethodGet, "/audit-events/export?format=csv&entityType=JournalEntry", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Permissions", "audit:read")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("export csv got %d ct=%q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "'=gl.journal.posted") {
		t.Fatalf("expected formula-neutralized action in CSV: %s", rec.Body.String())
	}

	// 2. JSON export
	req = httptest.NewRequest(http.MethodGet, "/audit-events/export?format=json", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Permissions", "audit:read")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"count":2`) {
		t.Fatalf("export json got %d body=%s", rec.Code, rec.Body.String())
	}

	// 3. Verify chain is deterministic and returns valid=true, count=2, non-empty hashes
	req = httptest.NewRequest(http.MethodGet, "/audit-events/verify-chain", nil)
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Permissions", "audit:read")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify-chain got %d: %s", rec.Code, rec.Body.String())
	}
	var chain map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &chain)
	if chain["valid"] != true || chain["count"] != float64(2) || chain["headHash"] == "" || chain["headHash"] == chain["genesisHash"] {
		t.Fatalf("unexpected verify-chain result: %+v", chain)
	}
}

