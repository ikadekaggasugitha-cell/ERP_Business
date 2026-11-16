package mdcache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

// PartyCacheStore persists masterdata party replication events into local cache.
type PartyCacheStore interface {
	ConsumeAndCacheParty(ctx context.Context, orgID, eventID, partyID uuid.UUID, eventType, code, name, partyType string) error
}

// Consumer maintains the local masterdata_cache table from masterdata events.
type Consumer struct {
	store PartyCacheStore
}

func New(store PartyCacheStore) *Consumer {
	return &Consumer{store: store}
}

func (c *Consumer) Topics() []string {
	return []string{
		eventbus.TopicFor("masterdata.party.upserted"),
		eventbus.TopicFor("masterdata.party.updated"),
	}
}

type partyPayload struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	PartyType string `json:"partyType"`
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
	partyID, err := uuid.Parse(env.AggregateID)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid aggregateId %q: %w", env.AggregateID, err)}
	}
	var p partyPayload
	if err := json.Unmarshal(env.Data, &p); err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("malformed party payload: %w", err)}
	}
	return c.store.ConsumeAndCacheParty(ctx, orgID, evID, partyID, env.EventType, p.Code, p.Name, p.PartyType)
}
