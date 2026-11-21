package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/ap-service/internal/apdb"
	"github.com/sinergiinvestasiproperti/erp/services/ap-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type stubStore struct {
	called  bool
	inv     apdb.CreateInvoiceParams
	ev      apdb.InsertOutboxParams
	settled []repository.SettlementAllocation
}

func (s *stubStore) ConsumeAndRecordInvoice(_ context.Context, _ []uuid.UUID, _ apdb.MarkEventProcessedParams, invoice apdb.CreateInvoiceParams, ev apdb.InsertOutboxParams) (bool, error) {
	s.called = true
	s.inv = invoice
	s.ev = ev
	return true, nil
}

func (s *stubStore) ConsumePaymentSettled(_ context.Context, _ []uuid.UUID, _ apdb.MarkEventProcessedParams, _, _, _ uuid.UUID, allocs []repository.SettlementAllocation) (bool, error) {
	s.settled = allocs
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

	// treasury.payment.settled applies allocations to AP invoices
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "treasury.payment.settled",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		Data:           []byte(`{"paymentNo":"PAY-1","paymentType":"Outgoing","allocations":[{"invoiceId":"PI-PO-99","invoiceType":"AP","allocatedAmount":"100000.0000"}]}`),
	})
	if err != nil {
		t.Fatalf("unexpected settlement err: %v", err)
	}
	if len(store.settled) != 1 || store.settled[0].InvoiceRef != "PI-PO-99" || platform.FormatNumeric(store.settled[0].AllocatedAmount, 4) != "100000.0000" {
		t.Fatalf("unexpected settlement allocations: %+v", store.settled)
	}
}

