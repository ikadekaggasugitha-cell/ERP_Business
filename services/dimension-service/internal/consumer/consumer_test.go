package consumer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/dimension-service/internal/dimensiondb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type fakeDimStore struct {
	created []dimensiondb.CreateCostCenterParams
}

func (f *fakeDimStore) CreateCostCenter(_ context.Context, _ []uuid.UUID, arg dimensiondb.CreateCostCenterParams, _ uuid.UUID) (dimensiondb.CreateCostCenterRow, error) {
	f.created = append(f.created, arg)
	return dimensiondb.CreateCostCenterRow{ID: uuid.New(), RowVersion: 1}, nil
}

func TestDimensionConsumerAutoProvisionsCostCenterForDivision(t *testing.T) {
	store := &fakeDimStore{}
	c := &Consumer{repo: store}

	// Auto-provision enabled for DIVISION -> creates CC-DIV-FIN.
	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "org.unit.upserted",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		CorrelationID:  uuid.NewString(),
		Data:           json.RawMessage(`{"code":"DIV-FIN","name":"Divisi Finance","typeCode":"DIVISION","autoProvisionCostCenter":true}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.created) != 1 || store.created[0].Code != "CC-DIV-FIN" {
		t.Fatalf("expected CC-DIV-FIN to be created, got %+v", store.created)
	}

	// Auto-provision disabled -> skipped.
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "org.unit.upserted",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{"code":"DIV-HR","name":"Divisi HR","typeCode":"DIVISION","autoProvisionCostCenter":false}`),
	})
	if err != nil || len(store.created) != 1 {
		t.Fatalf("expected skipped when autoProvisionCostCenter=false, err=%v count=%d", err, len(store.created))
	}
}
