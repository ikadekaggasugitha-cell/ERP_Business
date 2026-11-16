// Package mdcache keeps inventory-service's local party read model refreshed
// via masterdata.party.upserted with DLQ poison handling.
package mdcache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/inventorydb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

// Topics lists the masterdata Kafka topics subscribed by inventory-service.
var Topics = []string{
	eventbus.TopicFor("masterdata.party.upserted"),
}

// PartyCacheStore persists deduplicated party cache updates.
type PartyCacheStore interface {
	ConsumeAndCacheParty(ctx context.Context, scope []uuid.UUID, mark inventorydb.MarkEventProcessedParams, up inventorydb.UpsertMasterdataCacheParams) (bool, error)
}

type partyPayload struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	PartyType string `json:"partyType"`
}

// Consumer subscribes to masterdata party topics and refreshes inv.masterdata_cache.
type Consumer struct {
	repo PartyCacheStore
	c    *eventbus.Consumer
}

// New constructs a Consumer for inventory-service's masterdata cache group.
func New(repo PartyCacheStore, brokers []string) *Consumer {
	return &Consumer{repo: repo, c: eventbus.NewConsumer(brokers, "inventory-service-mdcache", Topics)}
}

// Run starts consuming events until ctx is cancelled.
func (x *Consumer) Run(ctx context.Context) error { return x.c.Run(ctx, x.Handle) }

// Close releases underlying Kafka reader resources.
func (x *Consumer) Close() error { return x.c.Close() }

// Handle processes a single masterdata envelope and wraps malformed payloads as ErrPoison.
func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	if env.EventType != "masterdata.party.upserted" {
		return nil
	}
	org, err := uuid.Parse(env.OrganizationID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId %q: %w", env.OrganizationID, err)}
	}
	eventID, err := uuid.Parse(env.EventID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId %q: %w", env.EventID, err)}
	}
	partyID, err := uuid.Parse(env.AggregateID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid aggregateId %q: %w", env.AggregateID, err)}
	}
	var p partyPayload
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid party payload: %w", err)}
		}
	}
	_, err = x.repo.ConsumeAndCacheParty(ctx, []uuid.UUID{org},
		inventorydb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType},
		inventorydb.UpsertMasterdataCacheParams{OrganizationID: org, PartyID: partyID, Code: p.Code, Name: p.Name, PartyType: p.PartyType})
	return err
}
