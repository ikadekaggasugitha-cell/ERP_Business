// Package consumer is treasury-cashbank-service's Kafka ingest (SDD BAB 6.5):
// it handles ap.invoice.recorded (INT-002) and payroll.run.completed (INT-003)
// idempotently via trs.processed_event, scheduling Draft payments.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/treasury-cashbank-service/internal/treasurydb"
)

var Topics = []string{
	eventbus.TopicFor("ap.invoice.recorded"),   // INT-002
	eventbus.TopicFor("payroll.run.completed"), // INT-003
}

func envUUID(key string, fallback uuid.UUID) uuid.UUID {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		if parsed, err := uuid.Parse(raw); err == nil && parsed != uuid.Nil {
			return parsed
		}
	}
	return fallback
}

var (
	sysActor = uuid.MustParse("00000000-0000-0000-0000-0000000000d1")
	glClear  = envUUID("TREASURY_DEFAULT_GL_CLEAR_ID", uuid.MustParse("10000000-0000-0000-0000-0000000000e3"))
)

type ConsumeStore interface {
	ConsumeAndSchedulePayment(
		ctx context.Context, scope []uuid.UUID,
		evMark treasurydb.MarkEventProcessedParams,
		glClear, sysActor uuid.UUID,
		pay treasurydb.CreatePaymentParams,
	) (bool, uuid.UUID, error)
}

type Consumer struct {
	repo ConsumeStore
	bus  *eventbus.Consumer
}

func New(repo ConsumeStore, brokers []string) *Consumer {
	c := &Consumer{repo: repo}
	c.bus = eventbus.NewConsumer(brokers, "treasury-cashbank-service", Topics)
	return c
}

func (c *Consumer) Run(ctx context.Context) error { return c.bus.Run(ctx, c.Handle) }
func (c *Consumer) Close() error                 { return c.bus.Close() }

type inboundPayload struct {
	InvoiceNo  string `json:"invoiceNo"`
	VendorID   string `json:"vendorId"`
	PartyID    string `json:"partyId"`
	TotalAmt   string `json:"totalAmount"`
	DueDate    string `json:"dueDate"`
	InvDate    string `json:"invoiceDate"`
	RunNo      string `json:"runNo"`
	PeriodCode string `json:"periodCode"`
	TotalNet   string `json:"totalNet"`
	TotalGross string `json:"totalGross"`
}

func (c *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	eventID, err := uuid.Parse(env.EventID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId %q: %w", env.EventID, err)}
	}
	orgID, err := uuid.Parse(env.OrganizationID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId %q: %w", env.OrganizationID, err)}
	}
	corrID, err := uuid.Parse(env.CorrelationID)
	if err != nil {
		corrID = eventID
	}

	var p inboundPayload
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("decode %s payload: %w", env.EventType, err)}
		}
	}

	var ref, amtRaw string
	switch env.EventType {
	case "ap.invoice.recorded":
		ref = strings.TrimSpace(p.InvoiceNo)
		amtRaw = strings.TrimSpace(p.TotalAmt)
	case "payroll.run.completed":
		ref = strings.TrimSpace(p.RunNo)
		if ref == "" {
			ref = strings.TrimSpace(p.PeriodCode)
		}
		amtRaw = strings.TrimSpace(p.TotalNet)
		if amtRaw == "" {
			amtRaw = strings.TrimSpace(p.TotalGross)
		}
	default:
		return nil
	}

	if ref == "" {
		ref = env.EventID[:8]
	}
	amt, err := platform.ParseOptionalNumeric(amtRaw)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid amount %q: %w", amtRaw, err)}
	}
	if amtRat := platform.NumericToRat(amt); amtRat == nil || amtRat.Sign() <= 0 {
		return &eventbus.ErrPoison{Err: fmt.Errorf("payment amount must be > 0, got %q", amtRaw)}
	}

	var partyID *uuid.UUID
	rawParty := strings.TrimSpace(p.VendorID)
	if rawParty == "" {
		rawParty = strings.TrimSpace(p.PartyID)
	}
	if rawParty != "" {
		uid, err := uuid.Parse(rawParty)
		if err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid party/vendor id %q: %w", rawParty, err)}
		}
		partyID = &uid
	}

	payDate := pgtype.Date{Time: time.Now().UTC(), Valid: true}
	if !env.OccurredAt.IsZero() {
		payDate = pgtype.Date{Time: env.OccurredAt.UTC(), Valid: true}
	}
	dateStr := strings.TrimSpace(p.DueDate)
	if dateStr == "" {
		dateStr = strings.TrimSpace(p.InvDate)
	}
	if dateStr != "" {
		d, err := platform.ParseDate(dateStr)
		if err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid date %q: %w", dateStr, err)}
		}
		payDate = d
	}

	payNo := "PAY-" + ref
	ext := ref
	pay := treasurydb.CreatePaymentParams{
		OrganizationID: orgID,
		PaymentNo:      payNo,
		PaymentType:    "Outgoing",
		PartyID:        partyID,
		Amount:         amt,
		PaymentDate:    payDate,
		ExternalRef:    &ext,
		CorrelationID:  corrID,
		CreatedBy:      sysActor,
	}
	_, _, err = c.repo.ConsumeAndSchedulePayment(
		ctx,
		[]uuid.UUID{orgID},
		treasurydb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType},
		glClear, sysActor, pay,
	)
	return err
}
