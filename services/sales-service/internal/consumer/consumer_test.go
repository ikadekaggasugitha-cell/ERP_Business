package consumer

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type fakeCompStore struct {
	called bool
	orgID  uuid.UUID
	evID   uuid.UUID
	soID   uuid.UUID
	evType string
	err    error
}

func (f *fakeCompStore) ConsumeAndCancel(_ context.Context, orgID, evID, soID uuid.UUID, evType string) error {
	f.called = true
	f.orgID, f.evID, f.soID, f.evType = orgID, evID, soID, evType
	return f.err
}

func TestConsumerHandleAndPoison(t *testing.T) {
	store := &fakeCompStore{}
	c := New(store)

	if err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "inventory.reservation.failed",
		OrganizationID: "not-a-uuid",
		AggregateID:    uuid.NewString(),
	}); err == nil {
		t.Fatal("expected error for invalid organizationId")
	} else {
		var poison eventbus.ErrPoison
		if !errors.As(err, &poison) {
			t.Fatalf("expected ErrPoison, got %T (%v)", err, err)
		}
	}

	orgID, evID, soID := uuid.New(), uuid.New(), uuid.New()
	if err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        evID.String(),
		EventType:      "inventory.reservation.failed",
		OrganizationID: orgID.String(),
		AggregateID:    soID.String(),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !store.called || store.soID != soID || store.orgID != orgID || store.evID != evID {
		t.Fatalf("store call mismatch: %+v", store)
	}
}

