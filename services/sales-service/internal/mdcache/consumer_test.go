package mdcache

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type fakePartyStore struct {
	called   bool
	orgID    uuid.UUID
	partyID  uuid.UUID
	code     string
	name     string
	partType string
}

func (f *fakePartyStore) ConsumeAndCacheParty(_ context.Context, orgID, _, partyID uuid.UUID, _, code, name, partyType string) error {
	f.called = true
	f.orgID, f.partyID, f.code, f.name, f.partType = orgID, partyID, code, name, partyType
	return nil
}

func TestMDCacheConsumerPoisonAndValid(t *testing.T) {
	store := &fakePartyStore{}
	c := New(store)

	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "masterdata.party.upserted",
		OrganizationID: uuid.NewString(),
		AggregateID:    uuid.NewString(),
		Data:           json.RawMessage(`{invalid`),
	})
	var poison eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison on malformed payload, got %v", err)
	}

	orgID, partyID := uuid.New(), uuid.New()
	if err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "masterdata.party.upserted",
		OrganizationID: orgID.String(),
		AggregateID:    partyID.String(),
		Data:           json.RawMessage(`{"code":"C-001","name":"PT Pembeli","partyType":"Customer"}`),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !store.called || store.code != "C-001" || store.partType != "Customer" {
		t.Fatalf("unexpected store state: %+v", store)
	}
}

