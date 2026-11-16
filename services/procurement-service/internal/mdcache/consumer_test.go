package mdcache

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/procurementdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type fakeCacheStore struct {
	calls int
	last  procurementdb.UpsertMasterdataCacheParams
}

func (f *fakeCacheStore) ConsumeAndCacheParty(_ context.Context, _ []uuid.UUID, _ procurementdb.MarkEventProcessedParams, c procurementdb.UpsertMasterdataCacheParams) (int64, error) {
	f.calls++
	f.last = c
	return 1, nil
}

func TestMDCacheConsumerPoisonAndValid(t *testing.T) {
	store := &fakeCacheStore{}
	c := &Consumer{repo: store}

	// Poison: invalid organizationId
	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "masterdata.party.upserted",
		AggregateID:    uuid.NewString(),
		OrganizationID: "not-a-uuid",
		Data:           json.RawMessage(`{"code":"V1","name":"Vendor 1","partyType":"Vendor"}`),
	})
	var poison *eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison for invalid org UUID, got %v", err)
	}

	// Poison: invalid JSON payload
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "masterdata.party.updated",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{invalid`),
	})
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison for invalid JSON payload, got %v", err)
	}

	// Ignore unrelated event type cleanly
	if err := c.Handle(context.Background(), eventbus.Envelope{EventType: "other.event"}); err != nil {
		t.Fatalf("unexpected error ignoring unrelated event: %v", err)
	}

	// Valid upserted event updates cache
	org := uuid.New()
	party := uuid.New()
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "masterdata.party.upserted",
		AggregateID:    party.String(),
		OrganizationID: org.String(),
		Data:           json.RawMessage(`{"code":"V-001","name":"PT Pemasok Jaya","partyType":"Vendor"}`),
	})
	if err != nil {
		t.Fatalf("valid handle returned error: %v", err)
	}
	if store.calls != 1 || store.last.PartyID != party || store.last.Code != "V-001" {
		t.Fatalf("expected 1 cache upsert for party %s, got calls=%d last=%+v", party, store.calls, store.last)
	}
}

