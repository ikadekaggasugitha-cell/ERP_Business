package mdcache

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/inventorydb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type fakeMDCacheStore struct {
	calls int
}

func (f *fakeMDCacheStore) ConsumeAndCacheParty(_ context.Context, _ []uuid.UUID, _ inventorydb.MarkEventProcessedParams, _ inventorydb.UpsertMasterdataCacheParams) (bool, error) {
	f.calls++
	return true, nil
}

func TestInventoryMDCachePoisonAndValid(t *testing.T) {
	store := &fakeMDCacheStore{}
	c := &Consumer{repo: store}

	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "masterdata.party.upserted",
		AggregateID:    uuid.NewString(),
		OrganizationID: "bad-org",
	})
	var poison *eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison, got %v", err)
	}

	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "masterdata.party.updated",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{"code":"C-1","name":"PT Pelanggan","partyType":"Customer"}`),
	})
	if err != nil || store.calls != 1 {
		t.Fatalf("expected valid party cache update, err=%v calls=%d", err, store.calls)
	}
}

