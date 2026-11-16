// Package httpapi implements audit-service read endpoints with RBAC, pagination,
// and structured error mapping.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/audit-service/internal/auditdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

// Store abstracts audit-service repository queries for handler unit testing.
type Store interface {
	ListFiltered(ctx context.Context, scope []uuid.UUID, entityType, action string, limit, offset int32) ([]auditdb.ListAuditLogsFilteredRow, int64, error)
	GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (auditdb.AuditAuditLog, error)
}

// Server serves audit-trail HTTP routes.
type Server struct{ repo Store }

// NewServer constructs an audit-service HTTP handler server.
func NewServer(repo Store) *Server { return &Server{repo: repo} }

// Register attaches audit-service endpoints onto mux.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /audit-events", s.listAuditEvents)
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
	entityType := strings.TrimSpace(r.URL.Query().Get("entityType"))
	action := strings.TrimSpace(r.URL.Query().Get("action"))

	rows, total, err := s.repo.ListFiltered(r.Context(), scope, entityType, action, limit, offset)
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
