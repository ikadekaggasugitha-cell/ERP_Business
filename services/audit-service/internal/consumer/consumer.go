// Package consumer subscribes to all ERP domain topics and appends an
// immutable audit trail record per event with at-least-once de-duplication.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/audit-service/internal/auditdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

// AllTopics enumerates every ERP domain topic persisted into audit.audit_log.
var AllTopics = []string{
	eventbus.TopicFor("gl.journal.posted"),
	eventbus.TopicFor("gl.journal.reversed"),
	eventbus.TopicFor("coa.account.upserted"),
	eventbus.TopicFor("coa.period.updated"),
	eventbus.TopicFor("ap.invoice.recorded"),
	eventbus.TopicFor("ar.invoice.issued"),
	eventbus.TopicFor("ar.invoice.paid"),
	eventbus.TopicFor("tax.invoice.issued"),
	eventbus.TopicFor("tax.efaktur.reported"),
	eventbus.TopicFor("treasury.payment.settled"),
	eventbus.TopicFor("treasury.payment.reconciled"),
	eventbus.TopicFor("cashbank.payment.settled"),
	eventbus.TopicFor("hr.employee.updated"),
	eventbus.TopicFor("payroll.run.calculated"),
	eventbus.TopicFor("payroll.run.completed"),
	eventbus.TopicFor("payroll.run.approved"),
	eventbus.TopicFor("payroll.run.posted"),
	eventbus.TopicFor("procurement.po.approved"),
	eventbus.TopicFor("procurement.goods.received"),
	eventbus.TopicFor("inventory.stock.reserved"),
	eventbus.TopicFor("inventory.goods.issued"),
	eventbus.TopicFor("inventory.reservation.failed"),
	eventbus.TopicFor("sales.order.confirmed"),
	eventbus.TopicFor("sales.delivery.completed"),
	eventbus.TopicFor("manufacturing.workorder.released"),
	eventbus.TopicFor("manufacturing.workorder.completed"),
	eventbus.TopicFor("masterdata.party.upserted"),
	eventbus.TopicFor("masterdata.item.upserted"),
	eventbus.TopicFor("org.unit.upserted"),
	eventbus.TopicFor("security.role.upserted"),
	eventbus.TopicFor("user.account.upserted"),
	eventbus.TopicFor("workflow.approval.requested"),
	eventbus.TopicFor("workflow.approval.completed"),
	eventbus.TopicFor("currency.created"),
	eventbus.TopicFor("currency.rate.created"),
	eventbus.TopicFor("currency.rate.updated"),
	eventbus.TopicFor("dimension.cost_center.upserted"),
	eventbus.TopicFor("dimension.profit_center.upserted"),
	eventbus.TopicFor("dimension.allocation_rule.upserted"),
	eventbus.TopicFor("dimension.allocation_rule.executed"),
	eventbus.TopicFor("localization.language.updated"),
	eventbus.TopicFor("localization.bundle.updated"),
	eventbus.TopicFor("localization.locale_format.updated"),
	eventbus.TopicFor("fixedasset.asset.acquired"),
	eventbus.TopicFor("fixedasset.depreciation.posted"),
	eventbus.TopicFor("fixedasset.asset.disposed"),
	eventbus.TopicFor("budget.approved"),
	eventbus.TopicFor("budget.commitment.created"),
	eventbus.TopicFor("financialclose.period.closed"),
	eventbus.TopicFor("financialclose.period.locked"),
	eventbus.TopicFor("financialclose.period.reopened"),
	eventbus.TopicFor("consolidation.run.finalized"),
	eventbus.TopicFor("project.billing.approved"),
	eventbus.TopicFor("asset.movement.completed"),
	eventbus.TopicFor("asset.verification.recorded"),
	eventbus.TopicFor("maintenance.workorder.parts_requested"),
	eventbus.TopicFor("maintenance.workorder.completed"),
	eventbus.TopicFor("crm.opportunity.won"),
}

// Recorder persists processed domain events into audit_db.
type Recorder interface {
	RecordFromEvent(ctx context.Context, scope []uuid.UUID, proc auditdb.MarkEventProcessedParams, entry auditdb.RecordAuditLogParams) (bool, error)
}

// Consumer wraps an eventbus.Consumer wired to an audit Recorder.
type Consumer struct {
	repo Recorder
	c    *eventbus.Consumer
}

// New constructs a Kafka consumer subscribed to AllTopics.
func New(repo Recorder, brokers []string) *Consumer {
	return &Consumer{
		repo: repo,
		c:    eventbus.NewConsumer(brokers, "audit-service-v1", AllTopics),
	}
}

// NewHandler creates a Consumer without Kafka connections for unit tests.
func NewHandler(repo Recorder) *Consumer {
	return &Consumer{repo: repo}
}

// Run blocks consuming domain events until ctx is cancelled.
func (h *Consumer) Run(ctx context.Context) error {
	return h.c.Run(ctx, h.Handle)
}

// Handle validates one Envelope and records it atomically into audit_db.
// Deterministic envelope errors wrap eventbus.ErrPoison for DLQ routing.
func (h *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	eventID, err := parseRequiredUUID("eventId", env.EventID)
	if err != nil {
		return eventbus.ErrPoison{Err: err}
	}
	eventType := strings.TrimSpace(env.EventType)
	if eventType == "" {
		return eventbus.ErrPoison{Err: fmt.Errorf("missing eventType")}
	}
	orgID, err := parseOrganizationID(eventType, env.OrganizationID)
	if err != nil {
		return eventbus.ErrPoison{Err: err}
	}
	corrID := uuid.New()
	if strings.TrimSpace(env.CorrelationID) != "" {
		parsedCorr, err := uuid.Parse(strings.TrimSpace(env.CorrelationID))
		if err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid correlationId: %w", err)}
		}
		corrID = parsedCorr
	}

	var entityPtr *uuid.UUID
	if strings.TrimSpace(env.AggregateID) != "" {
		if eid, err := uuid.Parse(strings.TrimSpace(env.AggregateID)); err == nil && eid != uuid.Nil {
			entityPtr = &eid
		}
	}
	entityType := strings.TrimSpace(env.AggregateType)
	if entityType == "" {
		entityType = eventType
	}
	occurredAt := env.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	afterValue := env.Data
	if len(afterValue) == 0 || !json.Valid(afterValue) {
		afterValue = []byte("{}")
	}

	_, err = h.repo.RecordFromEvent(
		ctx,
		[]uuid.UUID{orgID},
		auditdb.MarkEventProcessedParams{
			EventID:   eventID,
			EventType: eventType,
		},
		auditdb.RecordAuditLogParams{
			OccurredAt:     pgtype.Timestamptz{Time: occurredAt.UTC(), Valid: true},
			OrganizationID: orgID,
			Action:         eventType,
			EntityType:     entityType,
			EntityID:       entityPtr,
			AfterValue:     afterValue,
			CorrelationID:  corrID,
		},
	)
	return err
}

func parseRequiredUUID(field, raw string) (uuid.UUID, error) {
	u, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil || u == uuid.Nil {
		return uuid.Nil, fmt.Errorf("invalid or empty %s", field)
	}
	return u, nil
}

// parseOrganizationID allows the platform system organization UUID (uuid.Nil)
// for globally scoped currency domain events, while rejecting uuid.Nil on
// tenant-scoped events to prevent accidental cross-tenant visibility.
func parseOrganizationID(eventType, raw string) (uuid.UUID, error) {
	u, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid or empty organizationId")
	}
	if u == uuid.Nil && !strings.HasPrefix(eventType, "currency.") {
		return uuid.Nil, fmt.Errorf("invalid or empty organizationId")
	}
	return u, nil
}

// Close releases Kafka reader and DLQ writer handles.
func (h *Consumer) Close() error {
	if h == nil || h.c == nil {
		return nil
	}
	return h.c.Close()
}
