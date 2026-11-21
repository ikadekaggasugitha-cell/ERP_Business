// Package consumer is ap-service's Kafka ingest (SDD BAB 6.5, Saga INT-002):
// reacts to procurement.goods.received by recording a vendor invoice and
// reacts to treasury.payment.settled by allocating payments and updating AP invoice status.
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
	"github.com/sinergiinvestasiproperti/erp/services/ap-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Topics consumed by ap-service (Saga INT-002 + Treasury Settlement).
var Topics = []string{
	eventbus.TopicFor("procurement.goods.received"),
	eventbus.TopicFor("treasury.payment.settled"),
}

var (
	sysActor  = uuid.MustParse("00000000-0000-0000-0000-0000000000c1")
	sysVendor = uuid.MustParse("10000000-0000-0000-0000-0000000000e2")
)

type goodsReceivedPayload struct {
	PoNo         string `json:"poNo"`
	VendorID     string `json:"vendorId"`
	CurrencyCode string `json:"currencyCode"`
	Amount       string `json:"amount"`
}

type paymentSettledPayload struct {
	PaymentID   string `json:"paymentId"`
	PaymentNo   string `json:"paymentNo"`
	PaymentType string `json:"paymentType"`
	Amount      string `json:"amount"`
	ExternalRef string `json:"externalRef"`
	Allocations []struct {
		InvoiceID         string `json:"invoiceId"`
		TargetDocumentRef string `json:"targetDocumentRef"`
		InvoiceType       string `json:"invoiceType"`
		AllocatedAmount   string `json:"allocatedAmount"`
	} `json:"allocations"`
}

// Store abstracts AP repository persistence for consumer unit testing.
type Store interface {
	ConsumeAndRecordInvoice(ctx context.Context, scope []uuid.UUID, mark apdb.MarkEventProcessedParams, invoice apdb.CreateInvoiceParams, ev apdb.InsertOutboxParams) (bool, error)
	ConsumePaymentSettled(ctx context.Context, scope []uuid.UUID, mark apdb.MarkEventProcessedParams, orgID, paymentID, actor uuid.UUID, allocs []repository.SettlementAllocation) (bool, error)
}

// Consumer subscribes to Topics and records/settles vendor invoices.
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

// Handle processes procurement.goods.received and treasury.payment.settled idempotently;
// returns ErrPoison on malformed envelopes.
func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	if env.EventType != "procurement.goods.received" && env.EventType != "treasury.payment.settled" {
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
	if env.EventType == "treasury.payment.settled" {
		return x.handlePaymentSettled(ctx, org, eventID, env)
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
	ccy := strings.ToUpper(strings.TrimSpace(p.CurrencyCode))
	if len(ccy) != 3 {
		ccy = "IDR"
	}
	occurred := env.OccurredAt
	if occurred.IsZero() {
		occurred = time.Now().UTC()
	}
	dt := pgtype.Date{Time: occurred.UTC(), Valid: true}
	invNo := "PI-" + ref
	if len(invNo) > 50 {
		invNo = invNo[:50]
	}
	grRef := ref

	invoice := apdb.CreateInvoiceParams{
		OrganizationID:    org,
		InvoiceNo:         invNo,
		VendorID:          vendorID,
		InvoiceDate:       dt,
		DueDate:           dt,
		CurrencyCode:      ccy,
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
		"currencyCode":      ccy,
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

func (x *Consumer) handlePaymentSettled(ctx context.Context, org, eventID uuid.UUID, env eventbus.Envelope) error {
	var p paymentSettledPayload
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid treasury.payment.settled payload: %w", err)}
		}
	}
	if strings.EqualFold(strings.TrimSpace(p.PaymentType), "Incoming") {
		return nil
	}
	paymentID, err := uuid.Parse(strings.TrimSpace(p.PaymentID))
	if err != nil || paymentID == uuid.Nil {
		paymentID, err = uuid.Parse(strings.TrimSpace(env.AggregateID))
		if err != nil || paymentID == uuid.Nil {
			paymentID = eventID
		}
	}
	var allocs []repository.SettlementAllocation
	for _, a := range p.Allocations {
		if a.InvoiceType != "" && !strings.EqualFold(strings.TrimSpace(a.InvoiceType), "AP") {
			continue
		}
		ref := strings.TrimSpace(a.InvoiceID)
		if ref == "" {
			ref = strings.TrimSpace(a.TargetDocumentRef)
		}
		if ref == "" {
			continue
		}
		amt, err := platform.ParseNumeric(a.AllocatedAmount)
		if err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid allocatedAmount %q: %w", a.AllocatedAmount, err)}
		}
		if rVal := platform.NumericToRat(amt); rVal == nil || rVal.Sign() <= 0 {
			continue
		}
		allocs = append(allocs, repository.SettlementAllocation{
			InvoiceRef:      ref,
			AllocatedAmount: amt,
		})
	}
	if len(allocs) == 0 && strings.TrimSpace(p.ExternalRef) != "" && strings.TrimSpace(p.Amount) != "" {
		amt, err := platform.ParseNumeric(p.Amount)
		if err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid payment amount %q: %w", p.Amount, err)}
		}
		if rVal := platform.NumericToRat(amt); rVal != nil && rVal.Sign() > 0 {
			allocs = append(allocs, repository.SettlementAllocation{
				InvoiceRef:      strings.TrimSpace(p.ExternalRef),
				AllocatedAmount: amt,
			})
		}
	}
	if len(allocs) == 0 {
		return nil
	}
	_, err = x.repo.ConsumePaymentSettled(ctx, []uuid.UUID{org},
		apdb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType},
		org, paymentID, sysActor, allocs)
	return err
}
