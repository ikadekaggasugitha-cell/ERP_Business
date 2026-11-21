// Package consumer synchronizes fin.fiscal_period status from financialclose-service
// lifecycle events (closed, locked, reopened) idempotently via coa.processed_event.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

var Topics = []string{
	eventbus.TopicFor("financialclose.period.closed"),
	eventbus.TopicFor("financialclose.period.locked"),
	eventbus.TopicFor("financialclose.period.reopened"),
}

var sysActor = uuid.MustParse("00000000-0000-0000-0000-0000000000c2")

// PeriodSyncStore persists idempotent fiscal period status updates.
type PeriodSyncStore interface {
	ConsumePeriodStatus(ctx context.Context, orgID, eventID, periodID, actor uuid.UUID, eventType, newStatus string) (bool, error)
}

type Consumer struct {
	store PeriodSyncStore
	c     *eventbus.Consumer
}

func New(store PeriodSyncStore, brokers []string) *Consumer {
	return &Consumer{store: store, c: eventbus.NewConsumer(brokers, "coa-service", Topics)}
}

func NewHandler(store PeriodSyncStore) *Consumer {
	return &Consumer{store: store}
}

func (x *Consumer) Run(ctx context.Context) error { return x.c.Run(ctx, x.Handle) }

func (x *Consumer) Close() error {
	if x == nil || x.c == nil {
		return nil
	}
	return x.c.Close()
}

func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	var targetStatus string
	switch env.EventType {
	case "financialclose.period.closed":
		targetStatus = "Closed"
	case "financialclose.period.locked":
		targetStatus = "Locked"
	case "financialclose.period.reopened":
		targetStatus = "Open"
	default:
		return nil
	}

	orgID, err := uuid.Parse(strings.TrimSpace(env.OrganizationID))
	if err != nil || orgID == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId %q", env.OrganizationID)}
	}
	evID, err := uuid.Parse(strings.TrimSpace(env.EventID))
	if err != nil || evID == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId %q", env.EventID)}
	}
	var p struct {
		FiscalPeriodID string `json:"fiscalPeriodId"`
	}
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid financialclose payload: %w", err)}
		}
	}
	periodID, err := uuid.Parse(strings.TrimSpace(p.FiscalPeriodID))
	if err != nil || periodID == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid fiscalPeriodId %q", p.FiscalPeriodID)}
	}
	_, err = x.store.ConsumePeriodStatus(ctx, orgID, evID, periodID, sysActor, env.EventType, targetStatus)
	return err
}
