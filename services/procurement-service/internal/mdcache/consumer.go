// Package mdcache keeps procurement-service's local party read model refreshed
// via masterdata.party.upserted / masterdata.party.updated events with DLQ poison handling.
package mdcache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/procurementdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

// Topics lists the Kafka topics consumed by the procurement masterdata cache.
var Topics = []string{
	eventbus.TopicFor("masterdata.party.upserted"),
	eventbus.TopicFor("masterdata.party.updated"),
}

// PartyCacheStore persists deduplicated party cache updates.
type PartyCacheStore interface {
	ConsumeAndCacheParty(ctx context.Context, scope []uuid.UUID, ev procurementdb.MarkEventProcessedParams, c procurementdb.UpsertMasterdataCacheParams) (int64, error)
}

type partyPayload struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	PartyType string `json:"partyType"`
}

// Consumer subscribes to masterdata party topics and refreshes pr.masterdata_cache.
type Consumer struct {
	repo PartyCacheStore
	c    *eventbus.Consumer
}

// New constructs a masterdata cache consumer bound to procurement-service's consumer group.
func New(repo PartyCacheStore, brokers []string) *Consumer {
	return &Consumer{repo: repo, c: eventbus.NewConsumer(brokers, "procurement-service-mdcache", Topics)}
}

// Run starts consuming events until ctx is cancelled.
func (x *Consumer) Run(ctx context.Context) error { return x.c.Run(ctx, x.Handle) }

// Close releases underlying Kafka reader resources.
func (x *Consumer) Close() error { return x.c.Close() }

// Handle processes a single masterdata envelope, flagging malformed payloads as poison for DLQ routing.
func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	if env.EventType != "masterdata.party.upserted" && env.EventType != "masterdata.party.updated" {
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
		procurementdb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType},
		procurementdb.UpsertMasterdataCacheParams{OrganizationID: org, PartyID: partyID, Code: p.Code, Name: p.Name, PartyType: p.PartyType})
	return err
}
