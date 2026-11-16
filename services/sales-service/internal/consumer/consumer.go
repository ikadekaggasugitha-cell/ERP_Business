package consumer

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

// ReservationCompensationStore persists idempotent SO cancellation from inventory failures.
type ReservationCompensationStore interface {
	ConsumeAndCancel(ctx context.Context, orgID, eventID, soID uuid.UUID, eventType string) error
}

// Consumer subscribes to inventory.reservation.failed to compensate confirmed sales orders.
type Consumer struct {
	store ReservationCompensationStore
}

func New(store ReservationCompensationStore) *Consumer {
	return &Consumer{store: store}
}

func (c *Consumer) Topics() []string {
	return []string{eventbus.TopicFor("inventory.reservation.failed")}
}

func (c *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	orgID, err := uuid.Parse(env.OrganizationID)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId %q: %w", env.OrganizationID, err)}
	}
	evID, err := uuid.Parse(env.EventID)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId %q: %w", env.EventID, err)}
	}
	soID, err := uuid.Parse(env.AggregateID)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid aggregateId %q: %w", env.AggregateID, err)}
	}
	if err := c.store.ConsumeAndCancel(ctx, orgID, evID, soID, env.EventType); err != nil {
		return err
	}
	log.Printf("[sales-consumer] compensated sales order=%s status=Cancelled (event=%s)", soID, env.EventType)
	return nil
}
