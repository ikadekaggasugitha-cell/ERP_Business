package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/ar-service/internal/ardb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type stubStore struct {
	ev ardb.InsertOutboxParams
}

func (s *stubStore) ConsumeAndCreateInvoice(_ context.Context, _ []uuid.UUID, _ ardb.MarkEventProcessedParams, _ ardb.CreateInvoiceParams, _ []ardb.CreateInvoiceLineParams, ev ardb.InsertOutboxParams) (bool, error) {
	s.ev = ev
	return true, nil
}

func TestHandleGoodsIssuedARConsumer(t *testing.T) {
	s := &stubStore{}
	c := NewHandler(s)

	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        "bad",
		EventType:      "inventory.goods.issued",
		OrganizationID: uuid.NewString(),
	})
	var poison eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison, got %v", err)
	}

	custID := uuid.New()
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "inventory.goods.issued",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		OccurredAt:     time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC),
		Data:           []byte(`{"reference":"SO-55","customerId":"` + custID.String() + `","amount":"300000"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload map[string]any
	_ = json.Unmarshal(s.ev.Payload, &payload)
	if payload["invoiceNo"] != "SI-SO-55" || payload["customerId"] != custID.String() {
		t.Fatalf("unexpected AR outbox payload: %+v", payload)
	}

	// project.billing.approved creates AR invoice with PB- reference & currency
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "project.billing.approved",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		OccurredAt:     time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC),
		Data:           []byte(`{"arInvoiceRef":"PB-PRJ01-001","customerId":"` + custID.String() + `","amount":"750000.0000","currencyCode":"USD"}`),
	})
	if err != nil {
		t.Fatalf("unexpected project billing error: %v", err)
	}
	payload = map[string]any{}
	_ = json.Unmarshal(s.ev.Payload, &payload)
	if payload["invoiceNo"] != "PB-PRJ01-001" || payload["currencyCode"] != "USD" {
		t.Fatalf("unexpected project billing AR payload: %+v", payload)
	}
}

