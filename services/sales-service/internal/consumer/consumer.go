package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/sales-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

var (
	sysActor = uuid.MustParse("00000000-0000-0000-0000-0000000000d1")
	sysCust  = uuid.MustParse("10000000-0000-0000-0000-0000000000d1")
)

// ReservationCompensationStore persists idempotent SO compensation and CRM-won draft SOs.
type ReservationCompensationStore interface {
	ConsumeAndCancel(ctx context.Context, orgID, eventID, soID uuid.UUID, eventType string) error
	ConsumeAndCreateDraftSO(ctx context.Context, orgID, eventID uuid.UUID, eventType string, in repository.CreateSOInput) error
}

// Consumer subscribes to inventory.reservation.failed and crm.opportunity.won.
type Consumer struct {
	store ReservationCompensationStore
}

func New(store ReservationCompensationStore) *Consumer {
	return &Consumer{store: store}
}

func (c *Consumer) Topics() []string {
	return []string{
		eventbus.TopicFor("inventory.reservation.failed"),
		eventbus.TopicFor("crm.opportunity.won"),
	}
}

func (c *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	orgID, err := uuid.Parse(env.OrganizationID)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId %q: %w", env.OrganizationID, err)}
	}
	evID, err := uuid.Parse(env.EventID)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId %q: %w", env.EventID, err)}
	}

	switch env.EventType {
	case "inventory.reservation.failed":
		soID, err := uuid.Parse(env.AggregateID)
		if err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid aggregateId %q: %w", env.AggregateID, err)}
		}
		if err := c.store.ConsumeAndCancel(ctx, orgID, evID, soID, env.EventType); err != nil {
			return err
		}
		log.Printf("[sales-consumer] compensated sales order=%s status=Cancelled (event=%s)", soID, env.EventType)
		return nil

	case "crm.opportunity.won":
		var p struct {
			Code           string `json:"code"`
			CustomerID     string `json:"customerId"`
			EstimatedValue string `json:"estimatedValue"`
			CurrencyCode   string `json:"currencyCode"`
		}
		if len(env.Data) > 0 {
			if err := json.Unmarshal(env.Data, &p); err != nil {
				return eventbus.ErrPoison{Err: fmt.Errorf("invalid crm.opportunity.won payload: %w", err)}
			}
		}
		custID := sysCust
		if strings.TrimSpace(p.CustomerID) != "" {
			parsed, err := uuid.Parse(strings.TrimSpace(p.CustomerID))
			if err != nil {
				return eventbus.ErrPoison{Err: fmt.Errorf("invalid customerId %q: %w", p.CustomerID, err)}
			}
			custID = parsed
		}
		totalNum, err := platform.ParseOptionalNumeric(p.EstimatedValue)
		if err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid estimatedValue %q: %w", p.EstimatedValue, err)}
		}
		ccy := strings.ToUpper(strings.TrimSpace(p.CurrencyCode))
		if len(ccy) != 3 {
			ccy = "IDR"
		}
		code := strings.TrimSpace(p.Code)
		if code == "" {
			code = env.AggregateID
		}
		soNo := "SO-CRM-" + code
		if len(soNo) > 50 {
			soNo = soNo[:50]
		}
		corrID := evID
		if cID, err := uuid.Parse(strings.TrimSpace(env.CorrelationID)); err == nil && cID != uuid.Nil {
			corrID = cID
		}
		occ := env.OccurredAt
		if occ.IsZero() {
			occ = time.Now().UTC()
		}
		return c.store.ConsumeAndCreateDraftSO(ctx, orgID, evID, env.EventType, repository.CreateSOInput{
			OrganizationID: orgID,
			SoNo:           soNo,
			CustomerID:     custID,
			OrderDate:      pgtype.Date{Time: occ.UTC(), Valid: true},
			CurrencyCode:   ccy,
			TotalAmount:    totalNum,
			CorrelationID:  corrID,
			CreatedBy:      sysActor,
		})

	default:
		return nil
	}
}
