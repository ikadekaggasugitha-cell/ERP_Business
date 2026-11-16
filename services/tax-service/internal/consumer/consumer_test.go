package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/tax-service/internal/taxdb"
)

type stubStore struct {
	txn taxdb.CreateTaxTransactionParams
	ev  taxdb.InsertOutboxParams
}

func (s *stubStore) ConsumeAndCreateTaxInvoice(_ context.Context, _ []uuid.UUID, _ taxdb.MarkEventProcessedParams, _ uuid.UUID, txn taxdb.CreateTaxTransactionParams, ev taxdb.InsertOutboxParams) (bool, error) {
	s.txn = txn
	s.ev = ev
	return true, nil
}

func TestHandleExactRatVATAndPoisonRejection(t *testing.T) {
	s := &stubStore{}
	c := NewHandler(s)

	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "ar.invoice.issued",
		OrganizationID: uuid.NewString(),
		Data:           []byte(`{"invoiceNo":"SI-1","totalAmount":"not-a-number"}`),
	})
	var poison eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison for invalid decimal, got %v", err)
	}

	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "ar.invoice.issued",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		Data:           []byte(`{"invoiceNo":"SI-200","totalAmount":"1000000.0000"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if platform.FormatNumeric(s.txn.TaxAmount, 4) != "110000.0000" {
		t.Fatalf("expected exact 110000.0000 VAT, got %s", platform.FormatNumeric(s.txn.TaxAmount, 4))
	}
	var payload map[string]any
	_ = json.Unmarshal(s.ev.Payload, &payload)
	if payload["efakturNo"] != "EF-SI-200" {
		t.Fatalf("unexpected tax payload: %+v", payload)
	}
}

