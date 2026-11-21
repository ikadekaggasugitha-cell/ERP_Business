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
	taxCode     string
	explicitTax bool
	txn         taxdb.CreateTaxTransactionParams
	ev          taxdb.InsertOutboxParams
}

func (s *stubStore) ConsumeAndCreateTaxInvoice(_ context.Context, _ []uuid.UUID, _ taxdb.MarkEventProcessedParams, _ uuid.UUID, taxCode string, explicitTax bool, txn taxdb.CreateTaxTransactionParams, ev taxdb.InsertOutboxParams) (bool, error) {
	s.taxCode = taxCode
	s.explicitTax = explicitTax
	s.txn = txn
	s.ev = ev
	return true, nil
}

func TestHandleExactRatVATAndExplicitTaxHandling(t *testing.T) {
	s := &stubStore{}
	c := NewHandler(s)

	// 1. Invalid decimal -> ErrPoison
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

	// 2. No explicit taxAmount -> deferred to tenant VAT code rate (explicitTax=false)
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
	if s.explicitTax {
		t.Fatal("expected explicitTax=false when taxAmount omitted")
	}
	if platform.FormatNumeric(s.txn.TaxAmount, 4) != "110000.0000" {
		t.Fatalf("expected default 110000.0000 VAT, got %s", platform.FormatNumeric(s.txn.TaxAmount, 4))
	}
	var payload map[string]any
	_ = json.Unmarshal(s.ev.Payload, &payload)
	if payload["efakturNo"] != "EF-SI-200" {
		t.Fatalf("unexpected tax payload: %+v", payload)
	}

	// 3. Explicit taxAmount (e.g. 12% PPN inclusive total) -> preserves exact taxAmount and derives tax-exclusive baseAmount
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "ar.invoice.issued",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		Data:           []byte(`{"invoiceNo":"SI-201","totalAmount":"1120000.0000","taxAmount":"120000.0000","taxCode":"PPN12"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error with explicit tax: %v", err)
	}
	if !s.explicitTax || s.taxCode != "PPN12" {
		t.Fatalf("expected explicitTax=true taxCode=PPN12, got explicit=%v code=%q", s.explicitTax, s.taxCode)
	}
	if platform.FormatNumeric(s.txn.BaseAmount, 4) != "1000000.0000" || platform.FormatNumeric(s.txn.TaxAmount, 4) != "120000.0000" {
		t.Fatalf("expected base=1000000.0000 tax=120000.0000, got base=%s tax=%s",
			platform.FormatNumeric(s.txn.BaseAmount, 4), platform.FormatNumeric(s.txn.TaxAmount, 4))
	}

	// 4. taxAmount exceeding totalAmount -> ErrPoison
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "ar.invoice.issued",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		Data:           []byte(`{"invoiceNo":"SI-202","totalAmount":"100.0000","taxAmount":"200.0000"}`),
	})
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison when taxAmount > totalAmount, got %v", err)
	}

	// 5. Explicit zero taxAmount ("0.0000") must NOT be overwritten with 11% VAT, and long ref capped to 50 chars
	longRef := "INV-2026-VERY-LONG-REFERENCE-NUMBER-EXCEEDING-FIFTY-CHARACTERS-TOTAL"
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "ar.invoice.issued",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		Data:           []byte(`{"invoiceNo":"` + longRef + `","totalAmount":"500000.0000","taxAmount":"0.0000"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error on explicit zero tax: %v", err)
	}
	if !s.explicitTax || platform.FormatNumeric(s.txn.TaxAmount, 4) != "0.0000" {
		t.Fatalf("expected explicitTax=true taxAmount=0.0000, got explicit=%v tax=%s", s.explicitTax, platform.FormatNumeric(s.txn.TaxAmount, 4))
	}
	if s.txn.EfakturNo == nil || len(*s.txn.EfakturNo) > 50 {
		t.Fatalf("expected efakturNo capped <= 50 chars, got %v", s.txn.EfakturNo)
	}
}
