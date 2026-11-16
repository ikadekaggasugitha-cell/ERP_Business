package mdcache

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/ar-service/internal/ardb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type stubStore struct {
	up ardb.UpsertMasterdataCacheParams
}

func (s *stubStore) ConsumeAndCacheParty(_ context.Context, _ []uuid.UUID, _ ardb.MarkEventProcessedParams, up ardb.UpsertMasterdataCacheParams) (bool, error) {
	s.up = up
	return true, nil
}

func TestMDCacheHandle(t *testing.T) {
	s := &stubStore{}
	c := NewHandler(s)

	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        "bad",
		EventType:      "masterdata.party.upserted",
		OrganizationID: uuid.NewString(),
		AggregateID:    uuid.NewString(),
	})
	var poison eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison, got %v", err)
	}

	partyID := uuid.New()
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "masterdata.party.upserted",
		OrganizationID: uuid.NewString(),
		AggregateID:    partyID.String(),
		Data:           []byte(`{"code":"C001","name":"PT Pembeli","partyType":"Customer"}`),
	})
	if err != nil || s.up.Code != "C001" {
		t.Fatalf("unexpected result err=%v up=%+v", err, s.up)
	}
}

