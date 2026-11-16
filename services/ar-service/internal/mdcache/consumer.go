// Package mdcache refreshes ar.masterdata_cache on masterdata.party.upserted.
package mdcache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/ar-service/internal/ardb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

var Topics = []string{eventbus.TopicFor("masterdata.party.upserted")}

type partyPayload struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	PartyType string `json:"partyType"`
}

// Store abstracts cache persistence for testing.
type Store interface {
	ConsumeAndCacheParty(ctx context.Context, scope []uuid.UUID, mark ardb.MarkEventProcessedParams, up ardb.UpsertMasterdataCacheParams) (bool, error)
}

type Consumer struct {
	repo Store
	c    *eventbus.Consumer
}

func New(repo Store, brokers []string) *Consumer {
	return &Consumer{repo: repo, c: eventbus.NewConsumer(brokers, "ar-service-mdcache", Topics)}
}

func NewHandler(repo Store) *Consumer {
	return &Consumer{repo: repo}
}

func (x *Consumer) Run(ctx context.Context) error { return x.c.Run(ctx, x.Handle) }

func (x *Consumer) Close() error {
	if x == nil || x.c == nil {
		return nil
	}
	return x.c.Close()
}

func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	if env.EventType != "masterdata.party.upserted" {
		return nil
	}
	org, err := uuid.Parse(strings.TrimSpace(env.OrganizationID))
	if err != nil || org == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId")}
	}
	eventID, err := uuid.Parse(strings.TrimSpace(env.EventID))
	if err != nil || eventID == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId")}
	}
	partyID, err := uuid.Parse(strings.TrimSpace(env.AggregateID))
	if err != nil || partyID == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid aggregateId (partyId)")}
	}
	var p partyPayload
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid payload: %w", err)}
		}
	}
	_, err = x.repo.ConsumeAndCacheParty(ctx, []uuid.UUID{org},
		ardb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType},
		ardb.UpsertMasterdataCacheParams{OrganizationID: org, PartyID: partyID, Code: p.Code, Name: p.Name, PartyType: p.PartyType})
	return err
}
