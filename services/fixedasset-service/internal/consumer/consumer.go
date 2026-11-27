// Package consumer subscribes to AP & Procurement CapEx events and
// idempotently creates Draft fixed asset records for capitalization.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/fadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Topics consumed by fixedasset-service for automatic Draft CapEx asset registration.
var Topics = []string{
	eventbus.TopicFor("ap.invoice.posted"),
	eventbus.TopicFor("ap.invoice.recorded"),
	eventbus.TopicFor("procurement.goods.received"),
}

// Store abstracts transactional CapEx asset ingestion for unit testing.
type Store interface {
	ConsumeCapExAsset(ctx context.Context, scope []uuid.UUID, mark fadb.MarkEventProcessedParams, asset fadb.CreateFixedAssetParams) (bool, error)
}

// Consumer wraps an eventbus.Consumer wired to a fixed-asset Store.
type Consumer struct {
	repo Store
	c    *eventbus.Consumer
}

// New creates a Kafka consumer subscribed to CapEx-producing domain topics.
func New(repo Store, brokers []string) *Consumer {
	return &Consumer{
		repo: repo,
		c:    eventbus.NewConsumer(brokers, "fixedasset-service-v1", Topics),
	}
}

// NewHandler creates a Consumer without Kafka connections for unit tests.
func NewHandler(repo Store) *Consumer {
	return &Consumer{repo: repo}
}

// Run blocks consuming domain events until ctx is cancelled.
func (h *Consumer) Run(ctx context.Context) error {
	return h.c.Run(ctx, h.Handle)
}

// Close releases Kafka reader and DLQ writer handles.
func (h *Consumer) Close() error {
	if h == nil || h.c == nil {
		return nil
	}
	return h.c.Close()
}

type capexLine struct {
	Description     string `json:"description"`
	AccountCode     string `json:"accountCode"`
	AccountID       string `json:"accountId"`
	AssetCategoryID string `json:"assetCategoryId"`
	Amount          string `json:"amount"`
	IsCapex         bool   `json:"isCapex"`
}

type capexEventPayload struct {
	InvoiceNo       string      `json:"invoiceNo"`
	ReceiptNo       string      `json:"receiptNo"`
	ItemName        string      `json:"itemName"`
	Description     string      `json:"description"`
	IsCapex         bool        `json:"isCapex"`
	AssetCategoryID string      `json:"assetCategoryId"`
	CapexAmount     string      `json:"capexAmount"`
	TotalAmount     string      `json:"totalAmount"`
	AcquisitionDate string      `json:"acquisitionDate"`
	InvoiceDate     string      `json:"invoiceDate"`
	ReceivedDate    string      `json:"receivedDate"`
	ActorID         string      `json:"actorId"`
	Lines           []capexLine `json:"lines"`
}

// Handle inspects AP/Procurement events and creates a Draft fixed asset when CapEx is indicated.
func (h *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	eventID, err := uuid.Parse(strings.TrimSpace(env.EventID))
	if err != nil || eventID == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId: %q", env.EventID)}
	}
	orgID, err := uuid.Parse(strings.TrimSpace(env.OrganizationID))
	if err != nil || orgID == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId: %q", env.OrganizationID)}
	}

	var p capexEventPayload
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("decode payload: %w", err)}
		}
	}

	// Determine whether this event represents a CapEx acquisition.
	var (
		isCapex    = p.IsCapex || strings.TrimSpace(p.AssetCategoryID) != "" || strings.TrimSpace(p.CapexAmount) != ""
		amountStr  = firstNonEmpty(p.CapexAmount, p.TotalAmount)
		catIDStr   = strings.TrimSpace(p.AssetCategoryID)
		assetName  = firstNonEmpty(p.ItemName, p.Description)
		assetAcct  uuid.UUID
	)
	for _, ln := range p.Lines {
		lineIsCapex := ln.IsCapex || strings.TrimSpace(ln.AssetCategoryID) != "" ||
			strings.HasPrefix(strings.TrimSpace(ln.AccountCode), "15") ||
			strings.HasPrefix(strings.TrimSpace(ln.AccountCode), "16") ||
			strings.HasPrefix(strings.TrimSpace(ln.AccountCode), "17")
		if lineIsCapex {
			isCapex = true
			if strings.TrimSpace(ln.Amount) != "" {
				amountStr = strings.TrimSpace(ln.Amount)
			}
			if catIDStr == "" && strings.TrimSpace(ln.AssetCategoryID) != "" {
				catIDStr = strings.TrimSpace(ln.AssetCategoryID)
			}
			if assetName == "" && strings.TrimSpace(ln.Description) != "" {
				assetName = strings.TrimSpace(ln.Description)
			}
			if u, err := uuid.Parse(strings.TrimSpace(ln.AccountID)); err == nil && u != uuid.Nil {
				assetAcct = u
			}
			break
		}
	}
	if !isCapex {
		// Routine OpEx / inventory receipt: ignore cleanly.
		return nil
	}

	costNum, err := platform.ParseNumeric(amountStr)
	if err != nil {
		return nil
	}
	costRat := platform.NumericToRat(costNum)
	if costRat == nil || costRat.Sign() <= 0 {
		return nil
	}
	zeroNum, _ := platform.ParseNumeric("0.0000")

	var catID uuid.UUID
	if catIDStr != "" {
		if u, err := uuid.Parse(catIDStr); err == nil {
			catID = u
		}
	}

	acqDate := pgtype.Date{Time: time.Now().UTC(), Valid: true}
	for _, rawDate := range []string{p.AcquisitionDate, p.InvoiceDate, p.ReceivedDate} {
		if d, perr := platform.ParseDate(strings.TrimSpace(rawDate)); perr == nil && d.Valid {
			acqDate = d
			break
		}
	}

	docRef := firstNonEmpty(p.InvoiceNo, p.ReceiptNo, env.AggregateID, eventID.String()[:8])
	assetNo := "FA-" + docRef
	if len(assetNo) > 50 {
		assetNo = assetNo[:50]
	}
	if assetName == "" {
		assetName = "CapEx Asset " + docRef
	}

	actorID := orgID
	if u, err := uuid.Parse(strings.TrimSpace(p.ActorID)); err == nil && u != uuid.Nil {
		actorID = u
	}

	_, err = h.repo.ConsumeCapExAsset(
		ctx,
		[]uuid.UUID{orgID},
		fadb.MarkEventProcessedParams{
			EventID:   eventID,
			EventType: env.EventType,
		},
		fadb.CreateFixedAssetParams{
			OrganizationID:               orgID,
			AssetNo:                      assetNo,
			Name:                         assetName,
			AssetCategoryID:              catID,
			FinancialOrganizationRef:     orgID,
			AcquisitionDate:              acqDate,
			AcquisitionCost:              costNum,
			SalvageValue:                 zeroNum,
			UsefulLifeMonths:             60,
			DepreciationMethod:           "StraightLine",
			DepreciationStartDate:        acqDate,
			AssetAccountID:               assetAcct,
			AccumDepreciationAccountID:   assetAcct,
			DepreciationExpenseAccountID: assetAcct,
			NetBookValue:                 costNum,
			Status:                       "Draft",
			CreatedBy:                    actorID,
		},
	)
	return err
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
