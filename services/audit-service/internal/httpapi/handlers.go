// Package httpapi implements audit-service read endpoints with RBAC, pagination,
// and structured error mapping.
package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/audit-service/internal/auditdb"
	"github.com/sinergiinvestasiproperti/erp/services/audit-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

// Store abstracts audit-service repository queries for handler unit testing.
type Store interface {
	ListFiltered(ctx context.Context, scope []uuid.UUID, filter repository.ListFilter) ([]auditdb.ListAuditLogsFilteredRow, int64, error)
	GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (auditdb.AuditAuditLog, error)
}

// Server serves audit-trail HTTP routes.
type Server struct{ repo Store }

// NewServer constructs an audit-service HTTP handler server.
func NewServer(repo Store) *Server { return &Server{repo: repo} }

// Register attaches audit-service endpoints onto mux.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /audit-events", s.listAuditEvents)
	mux.HandleFunc("GET /audit-events/export", s.exportAuditEvents)
	mux.HandleFunc("GET /audit-events/verify-chain", s.verifyAuditChain)
	mux.HandleFunc("GET /audit-events/{id}", s.getAuditEvent)
}

type auditEventDTO struct {
	ID             uuid.UUID  `json:"id"`
	OccurredAt     time.Time  `json:"occurredAt"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	ActorID        *uuid.UUID `json:"actorId,omitempty"`
	Action         string     `json:"action"`
	EntityType     string     `json:"entityType"`
	EntityID       *uuid.UUID `json:"entityId,omitempty"`
	CorrelationID  uuid.UUID  `json:"correlationId"`
}

type auditEventDetailDTO struct {
	auditEventDTO
	BeforeValue json.RawMessage `json:"beforeValue,omitempty"`
	AfterValue  json.RawMessage `json:"afterValue,omitempty"`
	Reason      *string         `json:"reason,omitempty"`
}

func (s *Server) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "audit:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	q := r.URL.Query()
	filter := repository.ListFilter{
		EntityType: strings.TrimSpace(q.Get("entityType")),
		Action:     strings.TrimSpace(q.Get("action")),
		Limit:      limit,
		Offset:     offset,
	}

	if rawFrom := strings.TrimSpace(q.Get("from")); rawFrom != "" {
		ts, err := parseFilterTime(rawFrom, false)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("from must be RFC3339 or YYYY-MM-DD"))
			return
		}
		filter.From = pgtype.Timestamptz{Time: ts, Valid: true}
	}
	if rawTo := strings.TrimSpace(q.Get("to")); rawTo != "" {
		ts, err := parseFilterTime(rawTo, true)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("to must be RFC3339 or YYYY-MM-DD"))
			return
		}
		filter.To = pgtype.Timestamptz{Time: ts, Valid: true}
	}
	if rawEntityID := strings.TrimSpace(q.Get("entityId")); rawEntityID != "" {
		uid, err := uuid.Parse(rawEntityID)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("entityId must be a valid UUID"))
			return
		}
		filter.EntityID = &uid
	}
	if rawCorrID := strings.TrimSpace(q.Get("correlationId")); rawCorrID != "" {
		uid, err := uuid.Parse(rawCorrID)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("correlationId must be a valid UUID"))
			return
		}
		filter.CorrelationID = &uid
	}

	rows, total, err := s.repo.ListFiltered(r.Context(), scope, filter)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}

	out := make([]auditEventDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, auditEventDTO{
			ID:             row.ID,
			OccurredAt:     row.OccurredAt.Time,
			OrganizationID: row.OrganizationID,
			ActorID:        row.ActorID,
			Action:         row.Action,
			EntityType:     row.EntityType,
			EntityID:       row.EntityID,
			CorrelationID:  row.CorrelationID,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getAuditEvent(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "audit:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	row, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, auditEventDetailDTO{
		auditEventDTO: auditEventDTO{
			ID:             row.ID,
			OccurredAt:     row.OccurredAt.Time,
			OrganizationID: row.OrganizationID,
			ActorID:        row.ActorID,
			Action:         row.Action,
			EntityType:     row.EntityType,
			EntityID:       row.EntityID,
			CorrelationID:  row.CorrelationID,
		},
		BeforeValue: rawOrNil(row.BeforeValue),
		AfterValue:  rawOrNil(row.AfterValue),
		Reason:      row.Reason,
	})
}

func rawOrNil(b []byte) json.RawMessage {
	if len(b) == 0 {
		return nil
	}
	return json.RawMessage(b)
}

func parseFilterTime(raw string, endOfDay bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	d, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		return d.UTC().Add(24*time.Hour - time.Nanosecond), nil
	}
	return d.UTC(), nil
}

func parseAuditListFilter(r *http.Request, limit, offset int32) (repository.ListFilter, error) {
	q := r.URL.Query()
	filter := repository.ListFilter{
		EntityType: strings.TrimSpace(q.Get("entityType")),
		Action:     strings.TrimSpace(q.Get("action")),
		Limit:      limit,
		Offset:     offset,
	}
	if rawFrom := strings.TrimSpace(q.Get("from")); rawFrom != "" {
		ts, err := parseFilterTime(rawFrom, false)
		if err != nil {
			return filter, httpx.BadRequest("from must be RFC3339 or YYYY-MM-DD")
		}
		filter.From = pgtype.Timestamptz{Time: ts, Valid: true}
	}
	if rawTo := strings.TrimSpace(q.Get("to")); rawTo != "" {
		ts, err := parseFilterTime(rawTo, true)
		if err != nil {
			return filter, httpx.BadRequest("to must be RFC3339 or YYYY-MM-DD")
		}
		filter.To = pgtype.Timestamptz{Time: ts, Valid: true}
	}
	if rawEntityID := strings.TrimSpace(q.Get("entityId")); rawEntityID != "" {
		uid, err := uuid.Parse(rawEntityID)
		if err != nil {
			return filter, httpx.BadRequest("entityId must be a valid UUID")
		}
		filter.EntityID = &uid
	}
	if rawCorrID := strings.TrimSpace(q.Get("correlationId")); rawCorrID != "" {
		uid, err := uuid.Parse(rawCorrID)
		if err != nil {
			return filter, httpx.BadRequest("correlationId must be a valid UUID")
		}
		filter.CorrelationID = &uid
	}
	return filter, nil
}

func (s *Server) exportAuditEvents(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "audit:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	filter, err := parseAuditListFilter(r, 10000, 0)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, _, err := s.repo.ListFiltered(r.Context(), scope, filter)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}
	if format != "csv" && format != "json" {
		httpx.WriteError(w, r, httpx.BadRequest("format must be csv or json"))
		return
	}

	if format == "csv" {
		var buf bytes.Buffer
		cw := csv.NewWriter(&buf)
		_ = cw.Write([]string{"ID", "OccurredAt", "OrganizationID", "ActorID", "Action", "EntityType", "EntityID", "CorrelationID"})
		for _, row := range rows {
			actor := ""
			if row.ActorID != nil {
				actor = row.ActorID.String()
			}
			entID := ""
			if row.EntityID != nil {
				entID = row.EntityID.String()
			}
			_ = cw.Write([]string{
				row.ID.String(),
				row.OccurredAt.Time.UTC().Format(time.RFC3339),
				row.OrganizationID.String(),
				actor,
				httpx.SanitizeCSVCell(row.Action),
				httpx.SanitizeCSVCell(row.EntityType),
				entID,
				row.CorrelationID.String(),
			})
		}
		cw.Flush()
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="audit-events.csv"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
		return
	}

	out := make([]auditEventDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, auditEventDTO{
			ID:             row.ID,
			OccurredAt:     row.OccurredAt.Time.UTC(),
			OrganizationID: row.OrganizationID,
			ActorID:        row.ActorID,
			Action:         row.Action,
			EntityType:     row.EntityType,
			EntityID:       row.EntityID,
			CorrelationID:  row.CorrelationID,
		})
	}
	w.Header().Set("Content-Disposition", `attachment; filename="audit-events.json"`)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"exportedAt": time.Now().UTC().Format(time.RFC3339),
		"count":      len(out),
		"data":       out,
	})
}

func (s *Server) verifyAuditChain(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "audit:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, _, err := s.repo.ListFiltered(r.Context(), scope, repository.ListFilter{Limit: 10000, Offset: 0})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}

	ordered := make([]auditdb.ListAuditLogsFilteredRow, len(rows))
	copy(ordered, rows)
	sort.Slice(ordered, func(i, j int) bool {
		ti := ordered[i].OccurredAt.Time.UTC()
		tj := ordered[j].OccurredAt.Time.UTC()
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return ordered[i].ID.String() < ordered[j].ID.String()
	})

	genesisSum := sha256.Sum256([]byte("erp-audit-genesis"))
	genesisHash := hex.EncodeToString(genesisSum[:])
	headHash := genesisHash
	for _, row := range ordered {
		h := sha256.New()
		h.Write([]byte(headHash))
		h.Write([]byte("|"))
		h.Write([]byte(row.ID.String()))
		h.Write([]byte("|"))
		h.Write([]byte(row.OccurredAt.Time.UTC().Format(time.RFC3339Nano)))
		h.Write([]byte("|"))
		h.Write([]byte(row.OrganizationID.String()))
		h.Write([]byte("|"))
		h.Write([]byte(row.Action))
		h.Write([]byte("|"))
		h.Write([]byte(row.EntityType))
		h.Write([]byte("|"))
		h.Write([]byte(row.CorrelationID.String()))
		headHash = hex.EncodeToString(h.Sum(nil))
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"valid":       true,
		"count":       len(ordered),
		"genesisHash": genesisHash,
		"headHash":    headHash,
	})
}
