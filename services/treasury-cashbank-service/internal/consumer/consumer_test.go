package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/treasury-cashbank-service/internal/treasurydb"
)

type fakeConsumeStore struct {
	calls []treasurydb.CreatePaymentParams
}

func (f *fakeConsumeStore) ConsumeAndSchedulePayment(
	_ context.Context, _ []uuid.UUID,
	_ treasurydb.MarkEventProcessedParams,
	_, _ uuid.UUID,
	pay treasurydb.CreatePaymentParams,
) (bool, uuid.UUID, error) {
	f.calls = append(f.calls, pay)
	return true, uuid.New(), nil
}

func TestHandleSchedulesPaymentsAndRoutesPoison(t *testing.T) {
	store := &fakeConsumeStore{}
	c := &Consumer{repo: store}

	apData, _ := json.Marshal(map[string]any{
		"invoiceNo":   "INV-100",
		"vendorId":    "10000000-0000-0000-0000-000000000050",
		"totalAmount": "1250000.0000",
		"dueDate":     "2026-04-01",
	})
	if err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "ap.invoice.recorded",
		OrganizationID: "00000000-0000-0000-0000-000000000001",
		CorrelationID:  uuid.NewString(),
		OccurredAt:     time.Now().UTC(),
		Data:           apData,
	}); err != nil {
		t.Fatalf("unexpected error on ap.invoice.recorded: %v", err)
	}
	if len(store.calls) != 1 || store.calls[0].PaymentNo != "PAY-INV-100" || store.calls[0].PartyID == nil {
		t.Fatalf("unexpected scheduled ap payment: %+v", store.calls)
	}

	payData, _ := json.Marshal(map[string]any{
		"runNo":    "RUN-2026-03",
		"totalNet": "8500000.0000",
	})
	if err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "payroll.run.completed",
		OrganizationID: "00000000-0000-0000-0000-000000000001",
		Data:           payData,
	}); err != nil {
		t.Fatalf("unexpected error on payroll.run.completed: %v", err)
	}
	if len(store.calls) != 2 || store.calls[1].PaymentNo != "PAY-RUN-2026-03" {
		t.Fatalf("unexpected scheduled payroll payment: %+v", store.calls)
	}

	badAmt, _ := json.Marshal(map[string]any{"invoiceNo": "BAD", "totalAmount": "not-a-number"})
	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "ap.invoice.recorded",
		OrganizationID: "00000000-0000-0000-0000-000000000001",
		Data:           badAmt,
	})
	var poison *eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison for invalid amount, got %v", err)
	}

	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        "not-a-uuid",
		EventType:      "ap.invoice.recorded",
		OrganizationID: "00000000-0000-0000-0000-000000000001",
	})
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison for bad eventId, got %v", err)
	}
}

