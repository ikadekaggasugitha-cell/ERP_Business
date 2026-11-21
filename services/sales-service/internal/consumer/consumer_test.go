package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/sales-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type fakeCompStore struct {
	called    bool
	orgID     uuid.UUID
	evID      uuid.UUID
	soID      uuid.UUID
	evType    string
	draftIn   repository.CreateSOInput
	draftCall bool
	err       error
}

func (f *fakeCompStore) ConsumeAndCancel(_ context.Context, orgID, evID, soID uuid.UUID, evType string) error {
	f.called = true
	f.orgID, f.evID, f.soID, f.evType = orgID, evID, soID, evType
	return f.err
}

func (f *fakeCompStore) ConsumeAndCreateDraftSO(_ context.Context, orgID, evID uuid.UUID, evType string, in repository.CreateSOInput) error {
	f.draftCall = true
	f.orgID, f.evID, f.evType, f.draftIn = orgID, evID, evType, in
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

	custID := uuid.New()
	if err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "crm.opportunity.won",
		OrganizationID: orgID.String(),
		AggregateID:    uuid.NewString(),
		Data:           json.RawMessage(`{"code":"OPP-42","customerId":"` + custID.String() + `","estimatedValue":"1500000.0000","currencyCode":"USD"}`),
	}); err != nil {
		t.Fatalf("unexpected crm.opportunity.won error: %v", err)
	}
	if !store.draftCall || store.draftIn.SoNo != "SO-CRM-OPP-42" || store.draftIn.CustomerID != custID || store.draftIn.CurrencyCode != "USD" {
		t.Fatalf("unexpected draft SO input: %+v", store.draftIn)
	}
}
