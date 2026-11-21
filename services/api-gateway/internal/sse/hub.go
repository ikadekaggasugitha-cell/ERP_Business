package sse

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

// SystemOrgID is the shared platform organization UUID visible to all tenant scopes.
const SystemOrgID = "00000000-0000-0000-0000-000000000000"

// DomainTopics lists all canonical domain event topics bridged to SSE clients.
var DomainTopics = []string{
	"erp.gl.journal.posted.v1",
	"erp.gl.journal.reversed.v1",
	"erp.coa.account.upserted.v1",
	"erp.coa.period.updated.v1",
	"erp.ap.invoice.recorded.v1",
	"erp.ar.invoice.issued.v1",
	"erp.ar.invoice.paid.v1",
	"erp.tax.invoice.issued.v1",
	"erp.tax.efaktur.reported.v1",
	"erp.treasury.payment.settled.v1",
	"erp.treasury.payment.reconciled.v1",
	"erp.cashbank.payment.settled.v1",
	"erp.hr.employee.updated.v1",
	"erp.payroll.run.calculated.v1",
	"erp.payroll.run.completed.v1",
	"erp.payroll.run.approved.v1",
	"erp.payroll.run.posted.v1",
	"erp.procurement.po.approved.v1",
	"erp.procurement.goods.received.v1",
	"erp.inventory.stock.reserved.v1",
	"erp.inventory.goods.issued.v1",
	"erp.inventory.reservation.failed.v1",
	"erp.sales.order.confirmed.v1",
	"erp.sales.delivery.completed.v1",
	"erp.manufacturing.workorder.released.v1",
	"erp.manufacturing.workorder.completed.v1",
	"erp.masterdata.party.upserted.v1",
	"erp.masterdata.item.upserted.v1",
	"erp.org.unit.upserted.v1",
	"erp.security.role.upserted.v1",
	"erp.user.account.upserted.v1",
	"erp.workflow.approval.requested.v1",
	"erp.workflow.approval.completed.v1",
	"erp.currency.created.v1",
	"erp.currency.rate.created.v1",
	"erp.currency.rate.updated.v1",
	"erp.dimension.cost_center.upserted.v1",
	"erp.dimension.profit_center.upserted.v1",
	"erp.dimension.allocation_rule.upserted.v1",
	"erp.dimension.allocation_rule.executed.v1",
	"erp.localization.language.updated.v1",
	"erp.localization.bundle.updated.v1",
	"erp.localization.locale_format.updated.v1",
	"erp.fixedasset.asset.acquired.v1",
	"erp.fixedasset.depreciation.posted.v1",
	"erp.fixedasset.asset.disposed.v1",
	"erp.budget.approved.v1",
	"erp.budget.commitment.created.v1",
	"erp.financialclose.period.closed.v1",
	"erp.financialclose.period.locked.v1",
	"erp.financialclose.period.reopened.v1",
	"erp.consolidation.run.finalized.v1",
	"erp.project.billing.approved.v1",
	"erp.asset.movement.completed.v1",
	"erp.asset.verification.recorded.v1",
	"erp.maintenance.workorder.parts_requested.v1",
	"erp.maintenance.workorder.completed.v1",
	"erp.crm.opportunity.won.v1",
}

// StreamEvent matches both frontend RealtimeEvent (`timestamp`, `payload`) and Go EventEnvelope (`occurredAt`, `data`).
type StreamEvent struct {
	EventID        string          `json:"eventId"`
	EventType      string          `json:"eventType"`
	AggregateType  string          `json:"aggregateType"`
	AggregateID    string          `json:"aggregateId"`
	OrganizationID string          `json:"organizationId"`
	CorrelationID  string          `json:"correlationId"`
	Timestamp      string          `json:"timestamp"`
	OccurredAt     string          `json:"occurredAt"`
	Payload        json.RawMessage `json:"payload"`
	Data           json.RawMessage `json:"data"`
}

type Subscriber struct {
	ID    uint64
	OrgID string
	Ch    chan StreamEvent
}

type Hub struct {
	mu     sync.RWMutex
	nextID uint64
	subs   map[uint64]*Subscriber
}

func NewHub() *Hub {
	return &Hub{
		subs: make(map[uint64]*Subscriber),
	}
}

func (h *Hub) Subscribe(orgID string) (*Subscriber, func()) {
	h.mu.Lock()
	h.nextID++
	sub := &Subscriber{
		ID:    h.nextID,
		OrgID: orgID,
		Ch:    make(chan StreamEvent, 64),
	}
	h.subs[sub.ID] = sub
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if existing, ok := h.subs[sub.ID]; ok {
			delete(h.subs, sub.ID)
			close(existing.Ch)
		}
		h.mu.Unlock()
	}
	return sub, cancel
}

func (h *Hub) Broadcast(ev StreamEvent) {
	evOrg := strings.TrimSpace(ev.OrganizationID)
	isSystemOrg := strings.EqualFold(evOrg, SystemOrgID)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, sub := range h.subs {
		if sub.OrgID != "" && !isSystemOrg && (evOrg == "" || !strings.EqualFold(sub.OrgID, evOrg)) {
			continue
		}
		select {
		case sub.Ch <- ev:
		default:
			// Drop slowest frame rather than blocking Kafka consumers.
		}
	}
}

// StartKafkaBridge tails domain topics from Kafka/Redpanda using a single multi-topic
// GroupTopics reader with a per-instance fan-out GroupID so multi-replica gateway pods
// each receive all domain events without 58-reader rebalance storms.
func (h *Hub) StartKafkaBridge(ctx context.Context, brokers []string, groupID string) {
	if len(brokers) == 0 {
		log.Println("sse bridge: no KAFKA_BROKERS configured, live Kafka push disabled")
		return
	}
	instanceGroupID := strings.TrimSpace(groupID)
	if instanceGroupID == "" {
		instanceGroupID = "api-gateway-sse"
	}
	host, _ := os.Hostname()
	if host = strings.TrimSpace(host); host != "" {
		instanceGroupID = instanceGroupID + "-" + host
	} else {
		instanceGroupID = instanceGroupID + "-" + uuid.NewString()[:8]
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     instanceGroupID,
		GroupTopics: DomainTopics,
		StartOffset: kafka.LastOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     500 * time.Millisecond,
	})
	defer r.Close()

	for {
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			time.Sleep(time.Second)
			continue
		}
		var env eventbus.Envelope
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			continue
		}
		ts := env.OccurredAt.UTC().Format(time.RFC3339)
		if env.OccurredAt.IsZero() {
			ts = time.Now().UTC().Format(time.RFC3339)
		}
		h.Broadcast(StreamEvent{
			EventID:        env.EventID,
			EventType:      env.EventType,
			AggregateType:  env.AggregateType,
			AggregateID:    env.AggregateID,
			OrganizationID: env.OrganizationID,
			CorrelationID:  env.CorrelationID,
			Timestamp:      ts,
			OccurredAt:     ts,
			Payload:        env.Data,
			Data:           env.Data,
		})
	}
}

