// Package consumer is ap-service's Kafka ingest (SDD BAB 6.5, Saga INT-002):
// reacts to procurement.goods.received by recording a vendor invoice and
// emitting ap.invoice.recorded atomically.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/ap-service/internal/apdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Topics consumed by ap-service (Saga INT-002).
var Topics = []string{eventbus.TopicFor("procurement.goods.received")}

var (
	sysActor  = uuid.MustParse("00000000-0000-0000-0000-0000000000c1")
	sysVendor = uuid.MustParse("10000000-0000-0000-0000-0000000000e2")
)

type goodsReceivedPayload struct {
	PoNo     string `json:"poNo"`
	VendorID string `json:"vendorId"`
	Amount   string `json:"amount"`
}

// Store abstracts AP repository persistence for consumer unit testing.
type Store interface {
	ConsumeAndRecordInvoice(ctx context.Context, scope []uuid.UUID, mark apdb.MarkEventProcessedParams, invoice apdb.CreateInvoiceParams, ev apdb.InsertOutboxParams) (bool, error)
}

// Consumer subscribes to Topics and records vendor invoices.
type Consumer struct {
	repo Store
	c    *eventbus.Consumer
}

// New constructs an ap-service Kafka consumer.
func New(repo Store, brokers []string) *Consumer {
	return &Consumer{repo: repo, c: eventbus.NewConsumer(brokers, "ap-service", Topics)}
}

// NewHandler creates a Consumer without Kafka handles for unit tests.
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

// Handle records a vendor invoice idempotently+atomically; returns ErrPoison on malformed envelopes.
func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	if env.EventType != "procurement.goods.received" {
		return nil
	}
	org, err := uuid.Parse(strings.TrimSpace(env.OrganizationID))
	if err != nil || org == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId: %q", env.OrganizationID)}
	}
	eventID, err := uuid.Parse(strings.TrimSpace(env.EventID))
	if err != nil || eventID == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId: %q", env.EventID)}
	}
	corr := eventID
	if c, err := uuid.Parse(strings.TrimSpace(env.CorrelationID)); err == nil && c != uuid.Nil {
		corr = c
	}

	var p goodsReceivedPayload
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid event payload: %w", err)}
		}
	}
	ref := strings.TrimSpace(p.PoNo)
	if ref == "" {
		ref = strings.TrimSpace(env.AggregateID)
	}
	if ref == "" {
		return eventbus.ErrPoison{Err: fmt.Errorf("missing poNo/aggregateId")}
	}
	total, err := platform.ParseOptionalNumeric(p.Amount)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid amount: %w", err)}
	}
	vendorID := sysVendor
	if strings.TrimSpace(p.VendorID) != "" {
		if v, err := uuid.Parse(strings.TrimSpace(p.VendorID)); err == nil && v != uuid.Nil {
			vendorID = v
		}
	}
	occurred := env.OccurredAt
	if occurred.IsZero() {
		occurred = time.Now().UTC()
	}
	dt := pgtype.Date{Time: occurred.UTC(), Valid: true}
	invNo := "PI-" + ref
	grRef := ref

	invoice := apdb.CreateInvoiceParams{
		OrganizationID:    org,
		InvoiceNo:         invNo,
		VendorID:          vendorID,
		InvoiceDate:       dt,
		DueDate:           dt,
		CurrencyCode:      "IDR",
		TotalAmount:       total,
		OutstandingAmount: total,
		GoodsReceiptRef:   &grRef,
		CreatedBy:         sysActor,
	}
	payload, _ := json.Marshal(map[string]any{
		"invoiceNo":         invNo,
		"vendorId":          vendorID.String(),
		"invoiceDate":       platform.FormatDate(dt),
		"dueDate":           platform.FormatDate(dt),
		"currencyCode":      "IDR",
		"totalAmount":       platform.FormatNumeric(total, 4),
		"status":            "Recorded",
		"sourceEventType":   env.EventType,
		"sourceDocumentRef": ref,
	})
	ev := apdb.InsertOutboxParams{
		AggregateType:  "APInvoice",
		EventType:      "ap.invoice.recorded",
		Payload:        payload,
		OrganizationID: org,
		CorrelationID:  corr,
	}
	_, err = x.repo.ConsumeAndRecordInvoice(ctx, []uuid.UUID{org},
		apdb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType}, invoice, ev)
	return err
}
