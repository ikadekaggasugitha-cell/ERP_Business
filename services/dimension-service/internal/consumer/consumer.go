// Package consumer subscribes dimension-service to org.unit.upserted events
// to idempotently auto-provision Cost Centers for newly created Divisions/Sites.
package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/dimension-service/internal/dimensiondb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

var Topics = []string{
	eventbus.TopicFor("org.unit.upserted"),
}

var sysActor = uuid.MustParse("00000000-0000-0000-0000-0000000000d2")

type Store interface {
	CreateCostCenter(ctx context.Context, scope []uuid.UUID, arg dimensiondb.CreateCostCenterParams, corr uuid.UUID) (dimensiondb.CreateCostCenterRow, error)
}

type Consumer struct {
	repo Store
	c    *eventbus.Consumer
}

func New(repo Store, brokers []string) *Consumer {
	return &Consumer{repo: repo, c: eventbus.NewConsumer(brokers, "dimension-service", Topics)}
}

func (x *Consumer) Run(ctx context.Context) error { return x.c.Run(ctx, x.Handle) }
func (x *Consumer) Close() error                  { return x.c.Close() }

func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	if env.EventType != "org.unit.upserted" {
		return nil
	}
	org, err := uuid.Parse(env.OrganizationID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId %q: %w", env.OrganizationID, err)}
	}
	var p struct {
		Code                    string `json:"code"`
		Name                    string `json:"name"`
		TypeCode                string `json:"typeCode"`
		AutoProvisionCostCenter bool   `json:"autoProvisionCostCenter"`
	}
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid org.unit.upserted payload: %w", err)}
		}
	}
	if !p.AutoProvisionCostCenter {
		return nil
	}
	switch strings.ToUpper(strings.TrimSpace(p.TypeCode)) {
	case "DIVISION", "DEPARTMENT", "BUSINESS_UNIT", "PROJECT_SITE":
	default:
		return nil
	}
	code := strings.TrimSpace(p.Code)
	name := strings.TrimSpace(p.Name)
	if code == "" || name == "" {
		return nil
	}
	ccCode := "CC-" + code
	if strings.HasPrefix(strings.ToUpper(code), "CC-") {
		ccCode = code
	}
	if len(ccCode) > 50 {
		ccCode = ccCode[:50]
	}
	corr, err := uuid.Parse(env.CorrelationID)
	if err != nil {
		corr = uuid.New()
	}
	_, err = x.repo.CreateCostCenter(ctx, []uuid.UUID{org}, dimensiondb.CreateCostCenterParams{
		OrganizationID: org,
		Code:           ccCode,
		Name:           name,
		Status:         "Active",
		CreatedBy:      sysActor,
	}, corr)
	if err != nil {
		var ae *httpx.APIError
		if errors.As(err, &ae) && ae.Code == "DUPLICATE_RESOURCE" {
			return nil
		}
		return err
	}
	return nil
}
