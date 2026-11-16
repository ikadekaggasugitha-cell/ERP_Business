// Package consumer reacts to ar.invoice.issued by creating a Draft e-Faktur
// tax transaction using exact *big.Rat VAT arithmetic and emitting tax.invoice.issued.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/tax-service/internal/taxdb"
)

var Topics = []string{eventbus.TopicFor("ar.invoice.issued")}

var (
	sysActor = uuid.MustParse("00000000-0000-0000-0000-0000000000d2")
	vatRatio = big.NewRat(11, 100) // statutory 11% PPN
)

type arPayload struct {
	InvoiceNo   string `json:"invoiceNo"`
	TotalAmount string `json:"totalAmount"`
}

// Store abstracts tax repository operations for consumer unit testing.
type Store interface {
	ConsumeAndCreateTaxInvoice(ctx context.Context, scope []uuid.UUID, mark taxdb.MarkEventProcessedParams, sysActor uuid.UUID, txn taxdb.CreateTaxTransactionParams, ev taxdb.InsertOutboxParams) (bool, error)
}

type Consumer struct {
	repo Store
	c    *eventbus.Consumer
}

func New(repo Store, brokers []string) *Consumer {
	return &Consumer{repo: repo, c: eventbus.NewConsumer(brokers, "tax-service", Topics)}
}

func NewHandler(repo Store) *Consumer {
	return &Consumer{repo: repo}
}

func (x *Consumer) Run(ctx context.Context) error { return x.c.Run(ctx, x.Handle) }

func (x *Consumer) Close() error {
	if x == nil || x.c == nil {
		return nil
	}
	return x.c.Close()
}

func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	if env.EventType != "ar.invoice.issued" {
		return nil
	}
	org, err := uuid.Parse(strings.TrimSpace(env.OrganizationID))
	if err != nil || org == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId")}
	}
	eventID, err := uuid.Parse(strings.TrimSpace(env.EventID))
	if err != nil || eventID == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId")}
	}
	corr := eventID
	if c, err := uuid.Parse(strings.TrimSpace(env.CorrelationID)); err == nil && c != uuid.Nil {
		corr = c
	}

	var p arPayload
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid payload: %w", err)}
		}
	}
	ref := strings.TrimSpace(p.InvoiceNo)
	if ref == "" {
		ref = strings.TrimSpace(env.AggregateID)
	}
	if ref == "" {
		return eventbus.ErrPoison{Err: fmt.Errorf("missing invoiceNo/aggregateId")}
	}
	baseNum, err := platform.ParseOptionalNumeric(p.TotalAmount)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid totalAmount: %w", err)}
	}
	taxRat := new(big.Rat).Mul(platform.NumericToRat(baseNum), vatRatio)
	taxNum, err := platform.RatToNumeric(taxRat, 4)
	if err != nil {
		return err
	}

	efaktur := "EF-" + ref
	draft := "Draft"
	baseStr := platform.FormatNumeric(baseNum, 4)
	taxStr := platform.FormatNumeric(taxNum, 4)

	txn := taxdb.CreateTaxTransactionParams{
		OrganizationID:    org,
		SourceDocumentRef: ref,
		BaseAmount:        baseNum,
		TaxAmount:         taxNum,
		EfakturNo:         &efaktur,
		EfakturStatus:     &draft,
		CreatedBy:         sysActor,
	}
	payload, _ := json.Marshal(map[string]any{
		"sourceDocumentRef": ref,
		"efakturNo":         efaktur,
		"baseAmount":        baseStr,
		"taxAmount":         taxStr,
		"status":            "Draft",
	})
	ev := taxdb.InsertOutboxParams{
		AggregateType:  "TaxTransaction",
		EventType:      "tax.invoice.issued",
		Payload:        payload,
		OrganizationID: org,
		CorrelationID:  corr,
	}
	_, err = x.repo.ConsumeAndCreateTaxInvoice(ctx, []uuid.UUID{org},
		taxdb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType}, sysActor, txn, ev)
	return err
}
