package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type fakePeriodStore struct {
	periodID  uuid.UUID
	newStatus string
	calls     int
}

func (f *fakePeriodStore) ConsumePeriodStatus(_ context.Context, _, _, periodID, _ uuid.UUID, _, newStatus string) (bool, error) {
	f.periodID = periodID
	f.newStatus = newStatus
	f.calls++
	return true, nil
}

func TestCOAPeriodSyncConsumer(t *testing.T) {
	store := &fakePeriodStore{}
	c := NewHandler(store)

	// Poison: missing fiscalPeriodId
	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "financialclose.period.closed",
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{}`),
	})
	var poison eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison, got %v", err)
	}

	pid := uuid.New()
	for _, tc := range []struct {
		ev   string
		want string
	}{
		{"financialclose.period.closed", "Closed"},
		{"financialclose.period.locked", "Locked"},
		{"financialclose.period.reopened", "Open"},
	} {
		err = c.Handle(context.Background(), eventbus.Envelope{
			EventID:        uuid.NewString(),
			EventType:      tc.ev,
			OrganizationID: uuid.NewString(),
			Data:           json.RawMessage(`{"fiscalPeriodId":"` + pid.String() + `"}`),
		})
		if err != nil {
			t.Fatalf("%s unexpected error: %v", tc.ev, err)
		}
		if store.periodID != pid || store.newStatus != tc.want {
			t.Fatalf("%s: got (%v, %s), want (%v, %s)", tc.ev, store.periodID, store.newStatus, pid, tc.want)
		}
	}
}
