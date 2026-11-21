// Package consumer subscribes to hr.employee.updated and maintains the local
// pay.employee_cache read model so payroll runs block terminated/inactive employees.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

// Topics lists Kafka topics consumed by payroll-service.
var Topics = []string{
	eventbus.TopicFor("hr.employee.updated"),
}

// EmployeeCacheStore persists idempotent hr.employee.updated projections.
type EmployeeCacheStore interface {
	ConsumeEmployeeUpdated(ctx context.Context, orgID, eventID, employeeID uuid.UUID, eventType, employeeNo, status string) error
}

// Consumer handles inbound HR events for payroll-service.
type Consumer struct {
	store EmployeeCacheStore
	c     *eventbus.Consumer
}

// New constructs a payroll HR event Consumer.
func New(store EmployeeCacheStore, brokers []string) *Consumer {
	return &Consumer{
		store: store,
		c:     eventbus.NewConsumer(brokers, "payroll-service", Topics),
	}
}

// Run starts consuming events until ctx is cancelled.
func (x *Consumer) Run(ctx context.Context) error { return x.c.Run(ctx, x.Handle) }

// Close releases underlying Kafka reader resources.
func (x *Consumer) Close() error { return x.c.Close() }

// Handle processes one inbound event envelope.
func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	if env.EventType != "hr.employee.updated" {
		return nil
	}
	orgID, err := uuid.Parse(env.OrganizationID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId %q: %w", env.OrganizationID, err)}
	}
	evID, err := uuid.Parse(env.EventID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId %q: %w", env.EventID, err)}
	}
	var p struct {
		EmployeeID string `json:"employeeId"`
		EmployeeNo string `json:"employeeNo"`
		Status     string `json:"status"`
	}
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid hr.employee.updated payload: %w", err)}
		}
	}
	empIDStr := strings.TrimSpace(p.EmployeeID)
	if empIDStr == "" {
		empIDStr = strings.TrimSpace(env.AggregateID)
	}
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid employeeId %q: %w", empIDStr, err)}
	}
	status := strings.TrimSpace(p.Status)
	if status == "" {
		status = "Active"
	}
	return x.store.ConsumeEmployeeUpdated(ctx, orgID, evID, empID, env.EventType, strings.TrimSpace(p.EmployeeNo), status)
}
