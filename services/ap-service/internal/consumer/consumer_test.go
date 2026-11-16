package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/ap-service/internal/apdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type stubStore struct {
	called bool
	inv    apdb.CreateInvoiceParams
	ev     apdb.InsertOutboxParams
}

func (s *stubStore) ConsumeAndRecordInvoice(_ context.Context, _ []uuid.UUID, _ apdb.MarkEventProcessedParams, invoice apdb.CreateInvoiceParams, ev apdb.InsertOutboxParams) (bool, error) {
	s.called = true
	s.inv = invoice
	s.ev = ev
	return true, nil
}

func TestHandleGoodsReceivedEnrichesEventAndRejectsPoison(t *testing.T) {
	store := &stubStore{}
	c := NewHandler(store)

	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "procurement.goods.received",
		OrganizationID: "bad-org",
		Data:           []byte(`{"poNo":"PO-1","amount":"150000"}`),
	})
	var poison eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison, got %v", err)
	}

	vendorID := uuid.New()
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "procurement.goods.received",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		OccurredAt:     time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
		Data:           []byte(`{"poNo":"PO-99","vendorId":"` + vendorID.String() + `","amount":"250000.0000"}`),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !store.called || store.inv.InvoiceNo != "PI-PO-99" || store.inv.VendorID != vendorID {
		t.Fatalf("unexpected invoice recorded: %+v", store.inv)
	}
	var p map[string]any
	_ = json.Unmarshal(store.ev.Payload, &p)
	if p["vendorId"] != vendorID.String() || p["dueDate"] != "2026-11-02" {
		t.Fatalf("outbox missing vendorId/dueDate: %+v", p)
	}
}

